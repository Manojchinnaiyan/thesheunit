// internal/pkg/email/smtp.go - MISSING SMTP IMPLEMENTATION
package email

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

// sendSMTPEmail sends email using SMTP (Gmail, Outlook, or self-hosted)
func (s *EmailService) sendSMTPEmail(email *Email) error {
	// Validate SMTP configuration
	if s.config.External.Email.SMTPHost == "" || s.config.External.Email.SMTPUsername == "" {
		return fmt.Errorf("SMTP configuration incomplete: missing host or username")
	}

	// Set up authentication
	auth := smtp.PlainAuth("",
		s.config.External.Email.SMTPUsername,
		s.config.External.Email.SMTPPassword,
		s.config.External.Email.SMTPHost)

	// Prepare from address
	fromEmail := s.config.External.Email.FromEmail
	fromName := s.config.External.Email.FromName
	var from string
	if fromName != "" {
		from = fmt.Sprintf("%s <%s>", fromName, fromEmail)
	} else {
		from = fromEmail
	}

	// Prepare email headers and body
	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = strings.Join(email.To, ", ")
	headers["Subject"] = email.Subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=\"utf-8\""

	if s.config.External.Email.ReplyTo != "" {
		headers["Reply-To"] = s.config.External.Email.ReplyTo
	}

	// Build the email message
	var msg bytes.Buffer
	for key, value := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}
	msg.WriteString("\r\n")
	msg.WriteString(email.HTMLContent)

	// Determine server address
	serverAddr := fmt.Sprintf("%s:%d", s.config.External.Email.SMTPHost, s.config.External.Email.SMTPPort)

	// Send email based on TLS configuration
	if s.config.External.Email.SMTPUseTLS {
		return s.sendSMTPWithTLS(serverAddr, auth, fromEmail, email.To, msg.Bytes())
	} else {
		return smtp.SendMail(serverAddr, auth, fromEmail, email.To, msg.Bytes())
	}
}

// sendSMTPWithTLS sends email using explicit TLS connection
func (s *EmailService) sendSMTPWithTLS(serverAddr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	// Create TLS connection
	tlsConfig := &tls.Config{
		ServerName: s.config.External.Email.SMTPHost,
	}

	conn, err := tls.Dial("tcp", serverAddr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to create TLS connection: %w", err)
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, s.config.External.Email.SMTPHost)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Quit()

	// Authenticate
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	// Set sender
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, addr := range to {
		if err := client.Rcpt(addr); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", addr, err)
		}
	}

	// Send email content
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to send DATA command: %w", err)
	}
	defer writer.Close()

	_, err = writer.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to write email content: %w", err)
	}

	return nil
}
