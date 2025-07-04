// internal/pkg/email/service.go - FIXED TO MATCH EXISTING PATTERNS
package email

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/your-org/ecommerce-backend/internal/config"
)

// EmailService handles all email operations
type EmailService struct {
	config    *config.Config
	templates map[string]*template.Template
	client    *http.Client
}

// NewEmailService creates a new email service
func NewEmailService(cfg *config.Config) *EmailService {
	service := &EmailService{
		config:    cfg,
		templates: make(map[string]*template.Template),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Load email templates
	if err := service.loadTemplates(); err != nil {
		log.Printf("Warning: Failed to load email templates: %v", err)
	}

	return service
}

// SendEmail sends an email using the configured provider
func (s *EmailService) SendEmail(ctx context.Context, email *Email) error {
	switch s.config.External.Email.Provider {
	case "smtp":
		return s.sendSMTPEmail(email)
	case "resend":
		return s.sendResendEmail(email)
	case "sendgrid":
		return s.sendSendGridEmail(email)
	case "mailersend":
		return s.sendMailerSendEmail(email)
	default:
		// Development fallback - just log the email
		log.Printf("📧 EMAIL FALLBACK - Provider: %s not configured", s.config.External.Email.Provider)
		log.Printf("📧 To: %v", email.To)
		log.Printf("📧 Subject: %s", email.Subject)
		log.Printf("📧 Type: %s", email.Type)
		return nil
	}
}

// ENHANCED: SendTemplateEmail - Generic method for sending templated emails (MATCHING EXISTING PATTERN)
func (s *EmailService) SendTemplateEmail(to, subject, templateName string, data interface{}) error {
	htmlContent, err := s.renderTemplate(templateName, data)
	if err != nil {
		// If template fails, send a basic email
		log.Printf("Template render failed for %s: %v", templateName, err)
		htmlContent = s.createBasicEmailHTML(subject, fmt.Sprintf("Email content for %s", templateName))
	}

	email := &Email{
		To:          []string{to},
		Subject:     subject,
		HTMLContent: htmlContent,
		Type:        EmailType(templateName),
		Data:        map[string]interface{}{"template": templateName},
	}

	return s.SendEmail(context.Background(), email)
}

// SendWelcomeEmail sends a welcome email to new users
func (s *EmailService) SendWelcomeEmail(ctx context.Context, userEmail, userName, verificationToken string) error {
	data := WelcomeEmailData{
		EmailTemplateData: GetBaseTemplateData(
			s.config.External.Email.FromName,
			s.config.External.Email.BaseURL,
			userName,
			userEmail,
		),
		VerificationURL: fmt.Sprintf("%s/verify-email?token=%s", s.config.External.Email.BaseURL, verificationToken),
	}

	htmlContent, err := s.renderTemplate("welcome", data)
	if err != nil {
		return fmt.Errorf("failed to render welcome email template: %w", err)
	}

	email := &Email{
		To:          []string{userEmail},
		Subject:     fmt.Sprintf("Welcome to %s!", s.config.External.Email.FromName),
		HTMLContent: htmlContent,
		Type:        EmailTypeWelcome,
		Data:        map[string]interface{}{"user_name": userName},
	}

	return s.SendEmail(ctx, email)
}

// SendOrderConfirmationEmail sends order confirmation email
func (s *EmailService) SendOrderConfirmationEmail(ctx context.Context, data OrderConfirmationData) error {
	data.EmailTemplateData = GetBaseTemplateData(
		s.config.External.Email.FromName,
		s.config.External.Email.BaseURL,
		data.UserName,
		data.UserEmail,
	)

	htmlContent, err := s.renderTemplate("order_confirmation", data)
	if err != nil {
		return fmt.Errorf("failed to render order confirmation template: %w", err)
	}

	email := &Email{
		To:          []string{data.UserEmail},
		Subject:     fmt.Sprintf("Order Confirmation - %s", data.OrderNumber),
		HTMLContent: htmlContent,
		Type:        EmailTypeOrderConfirmation,
		Data: map[string]interface{}{
			"order_number": data.OrderNumber,
			"order_total":  data.OrderTotal,
		},
	}

	return s.SendEmail(ctx, email)
}

// SendPaymentSuccessEmail sends payment success notification
func (s *EmailService) SendPaymentSuccessEmail(ctx context.Context, data PaymentNotificationData) error {
	data.EmailTemplateData = GetBaseTemplateData(
		s.config.External.Email.FromName,
		s.config.External.Email.BaseURL,
		data.UserName,
		data.UserEmail,
	)

	htmlContent, err := s.renderTemplate("payment_success", data)
	if err != nil {
		return fmt.Errorf("failed to render payment success template: %w", err)
	}

	email := &Email{
		To:          []string{data.UserEmail},
		Subject:     fmt.Sprintf("Payment Successful - %s", data.OrderNumber),
		HTMLContent: htmlContent,
		Type:        EmailTypePaymentSuccess,
		Data: map[string]interface{}{
			"order_number":   data.OrderNumber,
			"amount":         data.Amount,
			"transaction_id": data.TransactionID,
		},
	}

	return s.SendEmail(ctx, email)
}

// SendPaymentFailedEmail sends payment failure notification
func (s *EmailService) SendPaymentFailedEmail(ctx context.Context, data PaymentNotificationData) error {
	data.EmailTemplateData = GetBaseTemplateData(
		s.config.External.Email.FromName,
		s.config.External.Email.BaseURL,
		data.UserName,
		data.UserEmail,
	)

	htmlContent, err := s.renderTemplate("payment_failed", data)
	if err != nil {
		return fmt.Errorf("failed to render payment failed template: %w", err)
	}

	email := &Email{
		To:          []string{data.UserEmail},
		Subject:     fmt.Sprintf("Payment Failed - %s", data.OrderNumber),
		HTMLContent: htmlContent,
		Type:        EmailTypePaymentFailed,
		Data: map[string]interface{}{
			"order_number": data.OrderNumber,
			"amount":       data.Amount,
			"reason":       data.Reason,
		},
	}

	return s.SendEmail(ctx, email)
}

// SendPasswordResetEmail sends password reset email
func (s *EmailService) SendPasswordResetEmail(ctx context.Context, userEmail, userName, resetToken string) error {
	data := PasswordResetData{
		EmailTemplateData: GetBaseTemplateData(
			s.config.External.Email.FromName,
			s.config.External.Email.BaseURL,
			userName,
			userEmail,
		),
		ResetURL:   fmt.Sprintf("%s/reset-password?token=%s", s.config.External.Email.BaseURL, resetToken),
		ExpiryTime: "24 hours",
	}

	htmlContent, err := s.renderTemplate("password_reset", data)
	if err != nil {
		return fmt.Errorf("failed to render password reset template: %w", err)
	}

	email := &Email{
		To:          []string{userEmail},
		Subject:     "Reset Your Password",
		HTMLContent: htmlContent,
		Type:        EmailTypePasswordReset,
		Data:        map[string]interface{}{"user_name": userName},
	}

	return s.SendEmail(ctx, email)
}

// SendOrderStatusUpdateEmail sends order status update notification
func (s *EmailService) SendOrderStatusUpdateEmail(ctx context.Context, data OrderStatusUpdateData) error {
	data.EmailTemplateData = GetBaseTemplateData(
		s.config.External.Email.FromName,
		s.config.External.Email.BaseURL,
		data.UserName,
		data.UserEmail,
	)

	htmlContent, err := s.renderTemplate("order_status_update", data)
	if err != nil {
		return fmt.Errorf("failed to render order status update template: %w", err)
	}

	email := &Email{
		To:          []string{data.UserEmail},
		Subject:     fmt.Sprintf("Order Update - %s", data.OrderNumber),
		HTMLContent: htmlContent,
		Type:        EmailTypeOrderStatusUpdate,
		Data: map[string]interface{}{
			"order_number": data.OrderNumber,
			"status":       data.Status,
		},
	}

	return s.SendEmail(ctx, email)
}

// SendEmailVerificationEmail sends email verification email
func (s *EmailService) SendEmailVerificationEmail(ctx context.Context, data EmailVerificationData) error {
	data.EmailTemplateData = GetBaseTemplateData(
		s.config.External.Email.FromName,
		s.config.External.Email.BaseURL,
		data.UserName,
		data.UserEmail,
	)

	htmlContent, err := s.renderTemplate("email_verification", data)
	if err != nil {
		return fmt.Errorf("failed to render email verification template: %w", err)
	}

	emailInstance := &Email{
		To:          []string{data.UserEmail},
		Subject:     "Verify Your Email Address",
		HTMLContent: htmlContent,
		Type:        EmailTypeEmailVerification,
		Data: map[string]interface{}{
			"user_name": data.UserName,
		},
	}

	return s.SendEmail(ctx, emailInstance)
}

// loadTemplates loads all email templates
func (s *EmailService) loadTemplates() error {
	templateDir := s.config.External.Email.TemplateDir
	if templateDir == "" {
		templateDir = "./templates/emails"
	}

	templates := []string{
		"welcome",
		"order_confirmation",
		"payment_success",
		"payment_failed",
		"password_reset",
		"email_verification",
		"order_status_update",
	}

	for _, name := range templates {
		templatePath := filepath.Join(templateDir, name+".html")
		tmpl, err := template.ParseFiles(templatePath)
		if err != nil {
			log.Printf("Warning: Could not load template %s: %v", name, err)
			// Create a basic fallback template
			s.templates[name] = s.createFallbackTemplate(name)
		} else {
			s.templates[name] = tmpl
		}
	}

	return nil
}

// renderTemplate renders an email template with data
func (s *EmailService) renderTemplate(templateName string, data interface{}) (string, error) {
	tmpl, exists := s.templates[templateName]
	if !exists {
		return "", fmt.Errorf("template %s not found", templateName)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", templateName, err)
	}

	return buf.String(), nil
}

// createFallbackTemplate creates a basic HTML template as fallback
func (s *EmailService) createFallbackTemplate(name string) *template.Template {
	basicTemplate := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>{{.SiteName}}</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; background-color: #f4f4f4; }
        .container { max-width: 600px; margin: 0 auto; background-color: white; padding: 20px; border-radius: 8px; }
        .header { background-color: #007bff; color: white; padding: 20px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { padding: 20px; }
        .footer { background-color: #f8f9fa; padding: 15px; text-align: center; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.SiteName}}</h1>
        </div>
        <div class="content">
            <p>Hello {{.UserName}},</p>
            <p>This is a notification from {{.SiteName}}.</p>
            {{if .VerificationURL}}
                <p><a href="{{.VerificationURL}}" style="background-color: #007bff; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">Verify Email</a></p>
            {{end}}
            {{if .ResetURL}}
                <p><a href="{{.ResetURL}}" style="background-color: #007bff; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">Reset Password</a></p>
            {{end}}
            {{if .OrderURL}}
                <p><a href="{{.OrderURL}}" style="background-color: #28a745; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">View Order</a></p>
            {{end}}
            <p>If you have any questions, please contact our support team.</p>
            <p>Best regards,<br>{{.SiteName}} Team</p>
        </div>
        <div class="footer">
            <p>© {{.Year}} {{.SiteName}}. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`

	tmpl, _ := template.New(name).Parse(basicTemplate)
	return tmpl
}

// createBasicEmailHTML creates a simple HTML email when templates fail
func (s *EmailService) createBasicEmailHTML(subject, message string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>%s</title>
</head>
<body style="font-family: Arial, sans-serif; margin: 0; padding: 20px; background-color: #f4f4f4;">
    <div style="max-width: 600px; margin: 0 auto; background-color: white; padding: 20px; border-radius: 8px;">
        <h2>%s</h2>
        <p>%s</p>
        <hr>
        <p style="font-size: 12px; color: #666;">
            © %d %s. All rights reserved.
        </p>
    </div>
</body>
</html>`, subject, subject, message, time.Now().Year(), s.config.External.Email.FromName)
}

func (s *EmailService) SendPasswordResetEmailByToken(ctx context.Context, userEmail, userName, resetToken string) error {
	data := PasswordResetData{
		EmailTemplateData: GetBaseTemplateData(
			s.config.External.Email.FromName,
			s.config.External.Email.BaseURL,
			userName,
			userEmail,
		),
		ResetURL:   fmt.Sprintf("%s/reset-password?token=%s", s.config.External.Email.BaseURL, resetToken),
		ExpiryTime: "24 hours",
	}

	htmlContent, err := s.renderTemplate("password_reset", data)
	if err != nil {
		return fmt.Errorf("failed to render password reset template: %w", err)
	}

	emailInstance := &Email{
		To:          []string{userEmail},
		Subject:     "Reset Your Password",
		HTMLContent: htmlContent,
		Type:        EmailTypePasswordReset,
		Data:        map[string]interface{}{"user_name": userName},
	}

	return s.SendEmail(ctx, emailInstance)
}
