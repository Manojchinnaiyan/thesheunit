// internal/pkg/email/types.go
package email

import (
	"time"
)

// EmailType represents the type of email being sent
type EmailType string

const (
	EmailTypeWelcome           EmailType = "welcome"
	EmailTypeEmailVerification EmailType = "email_verification"
	EmailTypePasswordReset     EmailType = "password_reset"
	EmailTypeOrderConfirmation EmailType = "order_confirmation"
	EmailTypeOrderStatusUpdate EmailType = "order_status_update"
	EmailTypePaymentSuccess    EmailType = "payment_success"
	EmailTypePaymentFailed     EmailType = "payment_failed"
	EmailTypeShippingUpdate    EmailType = "shipping_update"
	EmailTypeAccountUpdate     EmailType = "account_update"
)

// Email represents an email message
type Email struct {
	To          []string               `json:"to"`
	CC          []string               `json:"cc,omitempty"`
	BCC         []string               `json:"bcc,omitempty"`
	Subject     string                 `json:"subject"`
	HTMLContent string                 `json:"html_content"`
	TextContent string                 `json:"text_content,omitempty"`
	Type        EmailType              `json:"type"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

// EmailTemplateData contains common data for all email templates
type EmailTemplateData struct {
	SiteName   string `json:"site_name"`
	SiteURL    string `json:"site_url"`
	SupportURL string `json:"support_url"`
	UserName   string `json:"user_name"`
	UserEmail  string `json:"user_email"`
	Year       int    `json:"year"`
}

// WelcomeEmailData contains data for welcome email
type WelcomeEmailData struct {
	EmailTemplateData
	VerificationURL string `json:"verification_url"`
}

// OrderConfirmationData contains data for order confirmation email
type OrderConfirmationData struct {
	EmailTemplateData
	OrderNumber     string      `json:"order_number"`
	OrderDate       string      `json:"order_date"`
	OrderTotal      float64     `json:"order_total"`
	OrderURL        string      `json:"order_url"`
	TrackingURL     string      `json:"tracking_url"`
	Items           []OrderItem `json:"items"`
	ShippingMethod  string      `json:"shipping_method"`
	PaymentMethod   string      `json:"payment_method"`
	BillingAddress  Address     `json:"billing_address"`
	ShippingAddress Address     `json:"shipping_address"`
}

// OrderItem represents an item in the order
type OrderItem struct {
	Name       string  `json:"name"`
	SKU        string  `json:"sku"`
	Quantity   int     `json:"quantity"`
	Price      float64 `json:"price"`
	Total      float64 `json:"total"`
	ImageURL   string  `json:"image_url"`
	ProductURL string  `json:"product_url"`
}

// Address represents shipping/billing address
type Address struct {
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Company      string `json:"company"`
	AddressLine1 string `json:"address_line1"`
	AddressLine2 string `json:"address_line2"`
	City         string `json:"city"`
	State        string `json:"state"`
	PostalCode   string `json:"postal_code"`
	Country      string `json:"country"`
	Phone        string `json:"phone"`
}

// PaymentNotificationData contains data for payment notifications
type PaymentNotificationData struct {
	EmailTemplateData
	OrderNumber   string  `json:"order_number"`
	Amount        float64 `json:"amount"`
	PaymentMethod string  `json:"payment_method"`
	TransactionID string  `json:"transaction_id"`
	OrderURL      string  `json:"order_url"`
	Date          string  `json:"date"`
	Status        string  `json:"status"`
	Reason        string  `json:"reason,omitempty"` // For failed payments
}

// PasswordResetData contains data for password reset email
type PasswordResetData struct {
	EmailTemplateData
	ResetURL   string `json:"reset_url"`
	ExpiryTime string `json:"expiry_time"`
}

// EmailVerificationData contains data for email verification
type EmailVerificationData struct {
	EmailTemplateData
	VerificationURL string `json:"verification_url"`
	ExpiryTime      string `json:"expiry_time"`
}

// OrderStatusUpdateData contains data for order status updates
type OrderStatusUpdateData struct {
	EmailTemplateData
	OrderNumber       string `json:"order_number"`
	Status            string `json:"status"`
	StatusMessage     string `json:"status_message"`
	TrackingNumber    string `json:"tracking_number,omitempty"`
	TrackingURL       string `json:"tracking_url,omitempty"`
	OrderURL          string `json:"order_url"`
	EstimatedDelivery string `json:"estimated_delivery,omitempty"`
}

// GetBaseTemplateData returns common template data
func GetBaseTemplateData(siteName, siteURL, userName, userEmail string) EmailTemplateData {
	return EmailTemplateData{
		SiteName:   siteName,
		SiteURL:    siteURL,
		SupportURL: siteURL + "/support",
		UserName:   userName,
		UserEmail:  userEmail,
		Year:       time.Now().Year(),
	}
}
