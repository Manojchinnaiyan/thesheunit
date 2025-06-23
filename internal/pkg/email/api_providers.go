// internal/pkg/email/api_providers.go
package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Resend API structures
type ResendEmailRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
	ReplyTo string   `json:"reply_to,omitempty"`
}

type ResendResponse struct {
	ID string `json:"id"`
}

// SendGrid API structures
type SendGridEmailRequest struct {
	Personalizations []SendGridPersonalization `json:"personalizations"`
	From             SendGridEmail             `json:"from"`
	Subject          string                    `json:"subject"`
	Content          []SendGridContent         `json:"content"`
	ReplyTo          *SendGridEmail            `json:"reply_to,omitempty"`
}

type SendGridPersonalization struct {
	To []SendGridEmail `json:"to"`
}

type SendGridEmail struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type SendGridContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// MailerSend API structures
type MailerSendRequest struct {
	From      MailerSendEmail   `json:"from"`
	To        []MailerSendEmail `json:"to"`
	Subject   string            `json:"subject"`
	HTML      string            `json:"html"`
	ReplyTo   *MailerSendEmail  `json:"reply_to,omitempty"`
	Tags      []string          `json:"tags,omitempty"`
	Variables []interface{}     `json:"variables,omitempty"`
}

type MailerSendEmail struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

// sendResendEmail sends email using Resend API (Free: 3,000 emails/month)
func (s *EmailService) sendResendEmail(email *Email) error {
	apiKey := s.config.External.Email.APIKey
	if apiKey == "" {
		return fmt.Errorf("Resend API key not configured")
	}

	// Prepare from address
	fromEmail := s.config.External.Email.FromEmail
	fromName := s.config.External.Email.FromName
	var from string
	if fromName != "" {
		from = fmt.Sprintf("%s <%s>", fromName, fromEmail)
	} else {
		from = fromEmail
	}

	// Prepare request
	reqData := ResendEmailRequest{
		From:    from,
		To:      email.To,
		Subject: email.Subject,
		HTML:    email.HTMLContent,
		ReplyTo: s.config.External.Email.ReplyTo,
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return fmt.Errorf("failed to marshal Resend request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create Resend request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Resend request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Resend API returned status %d", resp.StatusCode)
	}

	return nil
}

// sendSendGridEmail sends email using SendGrid API (Free: 100 emails/day)
func (s *EmailService) sendSendGridEmail(email *Email) error {
	apiKey := s.config.External.Email.APIKey
	if apiKey == "" {
		return fmt.Errorf("SendGrid API key not configured")
	}

	// Prepare recipients
	var to []SendGridEmail
	for _, recipient := range email.To {
		to = append(to, SendGridEmail{Email: recipient})
	}

	// Prepare from
	from := SendGridEmail{
		Email: s.config.External.Email.FromEmail,
		Name:  s.config.External.Email.FromName,
	}

	// Prepare reply-to
	var replyTo *SendGridEmail
	if s.config.External.Email.ReplyTo != "" {
		replyTo = &SendGridEmail{Email: s.config.External.Email.ReplyTo}
	}

	// Prepare request
	reqData := SendGridEmailRequest{
		Personalizations: []SendGridPersonalization{
			{To: to},
		},
		From:    from,
		Subject: email.Subject,
		Content: []SendGridContent{
			{
				Type:  "text/html",
				Value: email.HTMLContent,
			},
		},
		ReplyTo: replyTo,
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return fmt.Errorf("failed to marshal SendGrid request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.sendgrid.com/v3/mail/send", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create SendGrid request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send SendGrid request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("SendGrid API returned status %d", resp.StatusCode)
	}

	return nil
}

// sendMailerSendEmail sends email using MailerSend API (Free: 12,000 emails/month)
func (s *EmailService) sendMailerSendEmail(email *Email) error {
	apiKey := s.config.External.Email.APIKey
	if apiKey == "" {
		return fmt.Errorf("MailerSend API key not configured")
	}

	// Prepare recipients
	var to []MailerSendEmail
	for _, recipient := range email.To {
		to = append(to, MailerSendEmail{Email: recipient})
	}

	// Prepare from
	from := MailerSendEmail{
		Email: s.config.External.Email.FromEmail,
		Name:  s.config.External.Email.FromName,
	}

	// Prepare reply-to
	var replyTo *MailerSendEmail
	if s.config.External.Email.ReplyTo != "" {
		replyTo = &MailerSendEmail{Email: s.config.External.Email.ReplyTo}
	}

	// Prepare request
	reqData := MailerSendRequest{
		From:    from,
		To:      to,
		Subject: email.Subject,
		HTML:    email.HTMLContent,
		ReplyTo: replyTo,
		Tags:    []string{string(email.Type)},
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return fmt.Errorf("failed to marshal MailerSend request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.mailersend.com/v1/email", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create MailerSend request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send MailerSend request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("MailerSend API returned status %d", resp.StatusCode)
	}

	return nil
}
