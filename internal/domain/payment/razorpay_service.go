// internal/domain/payment/razorpay_service.go - Complete implementation
package payment

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"time"

	"github.com/your-org/ecommerce-backend/internal/config"
	"github.com/your-org/ecommerce-backend/internal/domain/order"
	"github.com/your-org/ecommerce-backend/internal/domain/user"
	"github.com/your-org/ecommerce-backend/internal/pkg/email"
	"gorm.io/gorm"
)

// RazorpayService handles Razorpay payment processing
type RazorpayService struct {
	db           *gorm.DB
	config       *config.Config
	keyID        string
	keySecret    string
	baseURL      string
	httpClient   *http.Client
	emailService *email.EmailService
}

// NewRazorpayService creates a new Razorpay service
// NewRazorpayService creates a new Razorpay service
func NewRazorpayService(db *gorm.DB, cfg *config.Config) *RazorpayService {
	// Debug - check what we're getting from config
	fmt.Printf("🔍 Config Debug - KeyID from config: '%s'\n", cfg.External.Razorpay.KeyID)
	fmt.Printf("🔍 Config Debug - KeySecret from config: '%s'\n", cfg.External.Razorpay.KeySecret)

	// Check raw environment variables
	fmt.Printf("🔍 Raw Env - RAZORPAY_KEY_ID: '%s'\n", os.Getenv("RAZORPAY_KEY_ID"))
	fmt.Printf("🔍 Raw Env - RAZORPAY_KEY_SECRET: '%s'\n", os.Getenv("RAZORPAY_KEY_SECRET"))

	// If config values are empty, use env directly (temporary workaround)
	keyID := "rzp_test_JKEqkGjDCkWMed"
	keySecret := "lrcsYTuX1Y1iQmEzzfmYdxQZ"

	if keyID == "" {
		keyID = os.Getenv("RAZORPAY_KEY_ID")
		fmt.Printf("🔧 Using direct env for KeyID: '%s'\n", keyID)
	}

	if keySecret == "" {
		keySecret = os.Getenv("RAZORPAY_KEY_SECRET")
		fmt.Printf("🔧 Using direct env for KeySecret: '%s'\n", keySecret)
	}

	return &RazorpayService{
		db:        db,
		config:    cfg,
		keyID:     keyID,     // Use the resolved value
		keySecret: keySecret, // Use the resolved value
		baseURL:   "https://api.razorpay.com/v1",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		emailService: email.NewEmailService(cfg),
	}
}

// RazorpayOrder represents Razorpay order structure
type RazorpayOrder struct {
	ID        string                 `json:"id"`
	Entity    string                 `json:"entity"`
	Amount    int64                  `json:"amount"`
	Currency  string                 `json:"currency"`
	Receipt   string                 `json:"receipt"`
	Status    string                 `json:"status"`
	Notes     map[string]interface{} `json:"notes"`
	CreatedAt int64                  `json:"created_at"`
}

// CreateOrderRequest represents request to create Razorpay order
type CreateOrderRequest struct {
	Amount   int64                  `json:"amount"`   // Amount in paise
	Currency string                 `json:"currency"` // INR, USD, etc.
	Receipt  string                 `json:"receipt"`  // Order receipt
	Notes    map[string]interface{} `json:"notes,omitempty"`
}

// PaymentVerificationRequest represents payment verification data
type PaymentVerificationRequest struct {
	RazorpayOrderID   string `json:"razorpay_order_id" binding:"required"`
	RazorpayPaymentID string `json:"razorpay_payment_id" binding:"required"`
	RazorpaySignature string `json:"razorpay_signature" binding:"required"`
	OrderID           uint   `json:"order_id" binding:"required"`
}

// RazorpayPayment represents Razorpay payment structure
type RazorpayPayment struct {
	ID          string                 `json:"id"`
	Entity      string                 `json:"entity"`
	Amount      int64                  `json:"amount"`
	Currency    string                 `json:"currency"`
	Status      string                 `json:"status"`
	OrderID     string                 `json:"order_id"`
	Method      string                 `json:"method"`
	Description string                 `json:"description"`
	Email       string                 `json:"email"`
	Contact     string                 `json:"contact"`
	Fee         int64                  `json:"fee"`
	Tax         int64                  `json:"tax"`
	Notes       map[string]interface{} `json:"notes"`
	CreatedAt   int64                  `json:"created_at"`
}

// PaymentInitiationResponse represents response for payment initiation
type PaymentInitiationResponse struct {
	RazorpayOrderID string                 `json:"razorpay_order_id"`
	Amount          int64                  `json:"amount"`
	Currency        string                 `json:"currency"`
	Receipt         string                 `json:"receipt"`
	KeyID           string                 `json:"key_id"`
	Notes           map[string]interface{} `json:"notes"`
	OrderDetails    *order.Order           `json:"order_details"`
}

// CreatePaymentOrder creates a Razorpay order for payment
func (r *RazorpayService) CreatePaymentOrder(orderID uint) (*PaymentInitiationResponse, error) {
	// Check if Razorpay is configured
	// if r.keyID == "" || r.keySecret == "" {
	// 	return nil, fmt.Errorf("Razorpay configuration missing. Please set RAZORPAY_KEY_ID and RAZORPAY_KEY_SECRET")
	// }

	// Get order details
	var orderDetails order.Order
	err := r.db.Preload("Items").Where("id = ?", orderID).First(&orderDetails).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	// Check if order is in correct status for payment
	if orderDetails.Status != order.OrderStatusPending && orderDetails.Status != order.OrderStatusPaymentProcessing {
		return nil, fmt.Errorf("order is not in correct status for payment. Current status: %s", orderDetails.Status)
	}

	// Check if payment already exists for this order
	var existingPayment order.Payment
	result := r.db.Where("order_id = ? AND status IN ?", orderID, []order.PaymentStatus{
		order.PaymentStatusProcessing,
		order.PaymentStatusPaid,
	}).First(&existingPayment)

	if result.Error == nil {
		return nil, fmt.Errorf("payment already exists for this order. Status: %s", existingPayment.Status)
	}

	// Convert amount to paise (Razorpay uses smallest currency unit)
	amountInPaise := orderDetails.TotalAmount // Already in paise/cents

	// Create Razorpay order
	createReq := CreateOrderRequest{
		Amount:   amountInPaise,
		Currency: orderDetails.Currency,
		Receipt:  orderDetails.OrderNumber,
		Notes: map[string]interface{}{
			"order_id":     orderID,
			"user_id":      orderDetails.UserID,
			"order_number": orderDetails.OrderNumber,
		},
	}

	razorpayOrder, err := r.createRazorpayOrder(createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create Razorpay order: %w", err)
	}

	// Update order status to payment processing
	err = r.db.Model(&orderDetails).Updates(map[string]interface{}{
		"status":         order.OrderStatusPaymentProcessing,
		"payment_status": order.PaymentStatusProcessing,
	}).Error
	if err != nil {
		return nil, fmt.Errorf("failed to update order status: %w", err)
	}

	// Create payment record
	payment := order.Payment{
		OrderID:           orderID,
		PaymentMethod:     "razorpay",
		PaymentProviderID: razorpayOrder.ID,
		Amount:            amountInPaise,
		Currency:          orderDetails.Currency,
		Status:            order.PaymentStatusProcessing,
		Gateway:           "razorpay",
		GatewayResponse:   r.structToJSON(razorpayOrder),
	}

	err = r.db.Create(&payment).Error
	if err != nil {
		return nil, fmt.Errorf("failed to create payment record: %w", err)
	}

	// Prepare response for frontend
	response := &PaymentInitiationResponse{
		RazorpayOrderID: razorpayOrder.ID,
		Amount:          amountInPaise,
		Currency:        razorpayOrder.Currency,
		Receipt:         razorpayOrder.Receipt,
		KeyID:           "rzp_test_JKEqkGjDCkWMed",
		Notes:           razorpayOrder.Notes,
		OrderDetails:    &orderDetails,
	}

	return response, nil
}

// VerifyPayment verifies Razorpay payment signature and updates order status
func (r *RazorpayService) VerifyPayment(req *PaymentVerificationRequest) error {
	// Verify signature
	if !r.verifySignature(req.RazorpayOrderID, req.RazorpayPaymentID, req.RazorpaySignature) {
		return fmt.Errorf("invalid payment signature")
	}

	// Get payment details from Razorpay
	payment, err := r.getPaymentDetails(req.RazorpayPaymentID)
	if err != nil {
		return fmt.Errorf("failed to get payment details: %w", err)
	}

	// Verify payment amount and order details
	var orderDetails order.Order
	err = r.db.Where("id = ?", req.OrderID).First(&orderDetails).Error
	if err != nil {
		return fmt.Errorf("order not found: %w", err)
	}

	// Verify payment amount matches order amount
	if payment.Amount != orderDetails.TotalAmount {
		return fmt.Errorf("payment amount mismatch. Expected: %d, Got: %d", orderDetails.TotalAmount, payment.Amount)
	}

	// Start transaction
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update payment record
	err = tx.Model(&order.Payment{}).
		Where("order_id = ? AND payment_provider_id = ?", req.OrderID, req.RazorpayOrderID).
		Updates(map[string]interface{}{
			"status":           order.PaymentStatusPaid,
			"gateway_response": r.structToJSON(payment),
			"processed_at":     time.Now().UTC(),
		}).Error

	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update payment record: %w", err)
	}

	// Update order status
	err = tx.Model(&order.Order{}).
		Where("id = ?", req.OrderID).
		Updates(map[string]interface{}{
			"status":         order.OrderStatusConfirmed,
			"payment_status": order.PaymentStatusPaid,
		}).Error

	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// Add status history
	statusHistory := order.OrderStatusHistory{
		OrderID:   req.OrderID,
		Status:    order.OrderStatusConfirmed,
		Comment:   fmt.Sprintf("Payment confirmed via Razorpay. Payment ID: %s", req.RazorpayPaymentID),
		CreatedBy: 0, // System generated
		CreatedAt: time.Now().UTC(),
	}

	err = tx.Create(&statusHistory).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create status history: %w", err)
	}

	go func() {
		ctx := context.Background()

		// Get order and user details
		var orderRecord order.Order
		if err := r.db.Where("id = ?", req.OrderID).First(&orderRecord).Error; err != nil {
			log.Printf("Failed to get order for payment success email: %v", err)
			return
		}

		var userRecord user.User
		if err := r.db.Select("email, first_name, last_name").Where("id = ?", *orderRecord.UserID).First(&userRecord).Error; err != nil {
			log.Printf("Failed to get user for payment success email: %v", err)
			return
		}

		// Prepare email data
		emailData := email.PaymentNotificationData{
			EmailTemplateData: email.GetBaseTemplateData(
				r.config.External.Email.FromName,
				r.config.External.Email.BaseURL,
				userRecord.GetFullName(), // Use GetFullName()
				userRecord.Email,
			),
			OrderNumber:   orderRecord.OrderNumber,
			Amount:        float64(orderRecord.TotalAmount) / 100,
			PaymentMethod: "Razorpay",
			TransactionID: req.RazorpayPaymentID,
			OrderURL:      fmt.Sprintf("%s/orders/%s", r.config.External.Email.BaseURL, orderRecord.OrderNumber),
			Date:          time.Now().Format("January 2, 2006 at 3:04 PM"),
			Status:        "Success",
		}

		// Send payment success email
		if err := r.emailService.SendPaymentSuccessEmail(ctx, emailData); err != nil {
			log.Printf("Failed to send payment success email for order %s: %v", orderRecord.OrderNumber, err)
		}
	}()

	return tx.Commit().Error
}

// HandlePaymentFailure handles failed payments
func (r *RazorpayService) HandlePaymentFailure(orderID uint, reason string) error {
	// Start transaction
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update order status
	err := tx.Model(&order.Order{}).
		Where("id = ?", orderID).
		Updates(map[string]interface{}{
			"status":         order.OrderStatusPending,
			"payment_status": order.PaymentStatusFailed,
		}).Error

	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// Update payment record
	err = tx.Model(&order.Payment{}).
		Where("order_id = ?", orderID).
		Updates(map[string]interface{}{
			"status": order.PaymentStatusFailed,
		}).Error

	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update payment record: %w", err)
	}

	// Add status history
	statusHistory := order.OrderStatusHistory{
		OrderID:   orderID,
		Status:    order.OrderStatusPending,
		Comment:   fmt.Sprintf("Payment failed: %s", reason),
		CreatedBy: 0, // System generated
		CreatedAt: time.Now().UTC(),
	}

	err = tx.Create(&statusHistory).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create status history: %w", err)
	}

	go func() {
		ctx := context.Background()

		// Get order and user details
		var orderRecord order.Order
		if err := r.db.Where("id = ?", orderID).First(&orderRecord).Error; err != nil {
			log.Printf("Failed to get order for payment failure email: %v", err)
			return
		}

		var userRecord user.User
		if err := r.db.Select("email, first_name, last_name").Where("id = ?", *orderRecord.UserID).First(&userRecord).Error; err != nil {
			log.Printf("Failed to get user for payment failure email: %v", err)
			return
		}

		// Prepare email data
		emailData := email.PaymentNotificationData{
			EmailTemplateData: email.GetBaseTemplateData(
				r.config.External.Email.FromName,
				r.config.External.Email.BaseURL,
				userRecord.GetFullName(), // Use GetFullName()
				userRecord.Email,
			),
			OrderNumber:   orderRecord.OrderNumber,
			Amount:        float64(orderRecord.TotalAmount) / 100,
			PaymentMethod: "Razorpay",
			OrderURL:      fmt.Sprintf("%s/orders/%s", r.config.External.Email.BaseURL, orderRecord.OrderNumber),
			Date:          time.Now().Format("January 2, 2006 at 3:04 PM"),
			Status:        "Failed",
			Reason:        reason,
		}

		// Send payment failure email
		if err := r.emailService.SendPaymentFailedEmail(ctx, emailData); err != nil {
			log.Printf("Failed to send payment failure email for order %s: %v", orderRecord.OrderNumber, err)
		}
	}()

	return tx.Commit().Error
}

// GetPaymentStatus gets payment status for an order
func (r *RazorpayService) GetPaymentStatus(orderID uint) (*order.Payment, error) {
	var payment order.Payment
	err := r.db.Where("order_id = ?", orderID).Order("created_at DESC").First(&payment).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get payment status: %w", err)
	}
	return &payment, nil
}

// CreateRefund creates a refund for a payment
func (r *RazorpayService) CreateRefund(paymentID string, amount int64, reason string) error {
	// if r.keyID == "" || r.keySecret == "" {
	// 	return fmt.Errorf("Razorpay configuration missing")
	// }

	refundData := map[string]interface{}{
		"amount": amount,
		"notes": map[string]interface{}{
			"reason": reason,
		},
	}

	endpoint := fmt.Sprintf("/payments/%s/refund", paymentID)
	_, err := r.makeAPICall("POST", endpoint, refundData)
	if err != nil {
		return fmt.Errorf("failed to create refund: %w", err)
	}

	return nil
}

// Private helper methods

// createRazorpayOrder creates order in Razorpay
func (r *RazorpayService) createRazorpayOrder(req CreateOrderRequest) (*RazorpayOrder, error) {
	response, err := r.makeAPICall("POST", "/orders", req)
	if err != nil {
		return nil, err
	}

	var razorpayOrder RazorpayOrder
	err = json.Unmarshal(response, &razorpayOrder)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Razorpay order response: %w", err)
	}

	return &razorpayOrder, nil
}

// getPaymentDetails gets payment details from Razorpay
func (r *RazorpayService) getPaymentDetails(paymentID string) (*RazorpayPayment, error) {
	endpoint := fmt.Sprintf("/payments/%s", paymentID)
	response, err := r.makeAPICall("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var payment RazorpayPayment
	err = json.Unmarshal(response, &payment)
	if err != nil {
		return nil, fmt.Errorf("failed to parse payment response: %w", err)
	}

	return &payment, nil
}

// verifySignature verifies Razorpay webhook signature
func (r *RazorpayService) verifySignature(orderID, paymentID, signature string) bool {
	message := orderID + "|" + paymentID
	mac := hmac.New(sha256.New, []byte(r.keySecret))
	mac.Write([]byte(message))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// makeAPICall makes HTTP calls to Razorpay API
func (r *RazorpayService) makeAPICall(method, endpoint string, data interface{}) ([]byte, error) {
	if r.keyID == "" || r.keySecret == "" {
		return nil, fmt.Errorf("Razorpay API credentials not configured")
	}

	var reqBody []byte
	var err error

	if data != nil {
		reqBody, err = json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request data: %w", err)
		}
	}

	req, err := http.NewRequest(method, r.baseURL+endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(r.keyID, r.keySecret)

	// Make request
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make API call: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	var respBody bytes.Buffer
	_, err = respBody.ReadFrom(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API call failed with status %d: %s", resp.StatusCode, respBody.String())
	}

	return respBody.Bytes(), nil
}

// structToJSON converts struct to JSON string
func (r *RazorpayService) structToJSON(data interface{}) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(jsonData)
}
