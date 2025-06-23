// internal/pkg/email/smtp.go
package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

// sendSMTPEmail sends email using SMTP (Gmail, Outlook, or self-hosted)
func (s *EmailService) sendSMTPEmail(email *Email) error {
	// SMTP configuration
	host := s.config.External.Email.SMTPHost
	port := s.config.External.Email.SMTPPort
	username := s.config.External.Email.SMTPUsername
	password := s.config.External.Email.SMTPPassword
	useTLS := s.config.External.Email.SMTPUseTLS

	// Validate SMTP configuration
	if host == "" || username == "" || password == "" {
		return fmt.Errorf("SMTP configuration incomplete")
	}

	// Setup authentication
	auth := smtp.PlainAuth("", username, password, host)

	// Prepare email headers and body
	from := s.config.External.Email.FromEmail
	fromName := s.config.External.Email.FromName
	replyTo := s.config.External.Email.ReplyTo

	// Build email message
	message := s.buildEmailMessage(email, from, fromName, replyTo)

	// Setup TLS config
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         host,
	}

	// Connect to server
	serverAddr := fmt.Sprintf("%s:%d", host, port)

	if useTLS {
		// Use STARTTLS
		return s.sendWithSTARTTLS(serverAddr, auth, from, email.To, message, tlsConfig)
	} else {
		// Use plain SMTP (not recommended for production)
		return smtp.SendMail(serverAddr, auth, from, email.To, []byte(message))
	}
}

// sendWithSTARTTLS sends email using STARTTLS for security
func (s *EmailService) sendWithSTARTTLS(serverAddr string, auth smtp.Auth, from string, to []string, message string, tlsConfig *tls.Config) error {
	// Connect to the server
	client, err := smtp.Dial(serverAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer client.Close()

	// Start TLS
	if err = client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("failed to start TLS: %w", err)
	}

	// Authenticate
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}

	// Set sender
	if err = client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Add recipients
	for _, recipient := range to {
		if err = client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to add recipient %s: %w", recipient, err)
		}
	}

	// Send message
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	_, err = writer.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	return client.Quit()
}

// buildEmailMessage builds the complete email message with headers
func (s *EmailService) buildEmailMessage(email *Email, from, fromName, replyTo string) string {
	var message strings.Builder

	// From header
	if fromName != "" {
		message.WriteString(fmt.Sprintf("From: %s <%s>\r\n", fromName, from))
	} else {
		message.WriteString(fmt.Sprintf("From: %s\r\n", from))
	}

	// To header
	message.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(email.To, ", ")))

	// CC header
	if len(email.CC) > 0 {
		message.WriteString(fmt.Sprintf("CC: %s\r\n", strings.Join(email.CC, ", ")))
	}

	// BCC header (usually not included in message headers)
	// BCC recipients are added separately during SMTP transmission

	// Reply-To header
	if replyTo != "" {
		message.WriteString(fmt.Sprintf("Reply-To: %s\r\n", replyTo))
	}

	// Subject header
	message.WriteString(fmt.Sprintf("Subject: %s\r\n", email.Subject))

	// MIME headers for HTML email
	message.WriteString("MIME-Version: 1.0\r\n")
	message.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	message.WriteString("Content-Transfer-Encoding: 8bit\r\n")

	// Additional headers
	message.WriteString("X-Mailer: Go E-commerce Backend\r\n")
	message.WriteString(fmt.Sprintf("X-Email-Type: %s\r\n", email.Type))

	// Empty line to separate headers from body
	message.WriteString("\r\n")

	// Email body
	message.WriteString(email.HTMLContent)

	return message.String()
}

// ValidateSMTPConfig validates SMTP configuration
func (s *EmailService) ValidateSMTPConfig() error {
	cfg := s.config.External.Email

	if cfg.SMTPHost == "" {
		return fmt.Errorf("SMTP host is required")
	}

	if cfg.SMTPPort == 0 {
		return fmt.Errorf("SMTP port is required")
	}

	if cfg.SMTPUsername == "" {
		return fmt.Errorf("SMTP username is required")
	}

	if cfg.SMTPPassword == "" {
		return fmt.Errorf("SMTP password is required")
	}

	if cfg.FromEmail == "" {
		return fmt.Errorf("from email is required")
	}

	return nil
}

// TestSMTPConnection tests SMTP connection
func (s *EmailService) TestSMTPConnection() error {
	if err := s.ValidateSMTPConfig(); err != nil {
		return err
	}

	cfg := s.config.External.Email
	serverAddr := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)

	// Try to connect
	client, err := smtp.Dial(serverAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer client.Close()

	// Test STARTTLS if enabled
	if cfg.SMTPUseTLS {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         cfg.SMTPHost,
		}

		if err = client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	// Test authentication
	auth := smtp.PlainAuth("", cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPHost)
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}

	return client.Quit()
}
