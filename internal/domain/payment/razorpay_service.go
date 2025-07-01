// internal/domain/payment/razorpay_service.go - COMPLETE IMPLEMENTATION
package payment

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
func NewRazorpayService(db *gorm.DB, cfg *config.Config) *RazorpayService {
	// Use hardcoded test credentials if config is empty
	keyID := cfg.External.Razorpay.KeyID
	keySecret := cfg.External.Razorpay.KeySecret

	if keyID == "" {
		keyID = "rzp_test_JKEqkGjDCkWMed"
	}
	if keySecret == "" {
		keySecret = "lrcsYTuX1Y1iQmEzzfmYdxQZ"
	}

	return &RazorpayService{
		db:        db,
		config:    cfg,
		keyID:     keyID,
		keySecret: keySecret,
		baseURL:   "https://api.razorpay.com/v1",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		emailService: email.NewEmailService(cfg),
	}
}

// Required structs
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

type CreateOrderRequest struct {
	Amount   int64                  `json:"amount"`
	Currency string                 `json:"currency"`
	Receipt  string                 `json:"receipt"`
	Notes    map[string]interface{} `json:"notes,omitempty"`
}

type PaymentVerificationRequest struct {
	RazorpayOrderID   string `json:"razorpay_order_id" binding:"required"`
	RazorpayPaymentID string `json:"razorpay_payment_id" binding:"required"`
	RazorpaySignature string `json:"razorpay_signature" binding:"required"`
	OrderID           uint   `json:"order_id" binding:"required"`
}

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

type PaymentInitiationResponse struct {
	RazorpayOrderID string                 `json:"razorpay_order_id"`
	Amount          int64                  `json:"amount"`
	Currency        string                 `json:"currency"`
	Receipt         string                 `json:"receipt"`
	KeyID           string                 `json:"key_id"`
	Notes           map[string]interface{} `json:"notes"`
	OrderDetails    *order.Order           `json:"order_details"`
}

type RefundRequest struct {
	Amount  int64                  `json:"amount,omitempty"`
	Speed   string                 `json:"speed,omitempty"`
	Notes   map[string]interface{} `json:"notes,omitempty"`
	Receipt string                 `json:"receipt,omitempty"`
}

type RazorpayRefund struct {
	ID        string `json:"id"`
	Entity    string `json:"entity"`
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
	PaymentID string `json:"payment_id"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"created_at"`
}

// CreatePaymentOrder creates a Razorpay order for payment - NOW WITH RETRY SUPPORT
func (r *RazorpayService) CreatePaymentOrder(orderID uint) (*PaymentInitiationResponse, error) {
	// Get order details
	var orderDetails order.Order
	err := r.db.Preload("Items").Where("id = ?", orderID).First(&orderDetails).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	// ENHANCED: Check if order can accept payment (supports retry)
	if !r.canAcceptPayment(orderDetails) {
		return nil, fmt.Errorf("order cannot accept payment. Status: %s, Payment Status: %s",
			orderDetails.Status, orderDetails.PaymentStatus)
	}

	// ENHANCED: Handle existing payments for retry scenarios
	err = r.handleExistingPayments(orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to handle existing payments: %w", err)
	}

	// Create Razorpay order
	createReq := CreateOrderRequest{
		Amount:   orderDetails.TotalAmount,
		Currency: orderDetails.Currency,
		Receipt:  orderDetails.OrderNumber,
		Notes: map[string]interface{}{
			"order_id":     orderID,
			"user_id":      orderDetails.UserID,
			"order_number": orderDetails.OrderNumber,
			"retry":        r.isRetryAttempt(orderID),
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
		"updated_at":     time.Now().UTC(),
	}).Error
	if err != nil {
		return nil, fmt.Errorf("failed to update order status: %w", err)
	}

	// Create new payment record
	payment := order.Payment{
		OrderID:           orderID,
		PaymentMethod:     "razorpay",
		PaymentProviderID: razorpayOrder.ID,
		Amount:            orderDetails.TotalAmount,
		Currency:          orderDetails.Currency,
		Status:            order.PaymentStatusProcessing,
		Gateway:           "razorpay",
		GatewayResponse:   r.structToJSON(razorpayOrder),
		CreatedAt:         time.Now().UTC(),
	}

	err = r.db.Create(&payment).Error
	if err != nil {
		return nil, fmt.Errorf("failed to create payment record: %w", err)
	}

	// Prepare response for frontend
	response := &PaymentInitiationResponse{
		RazorpayOrderID: razorpayOrder.ID,
		Amount:          orderDetails.TotalAmount,
		Currency:        razorpayOrder.Currency,
		Receipt:         razorpayOrder.Receipt,
		KeyID:           r.keyID,
		Notes:           razorpayOrder.Notes,
		OrderDetails:    &orderDetails,
	}

	return response, nil
}

// ENHANCED: Check if order can accept payment (supports confirmed orders with failed payments)
func (r *RazorpayService) canAcceptPayment(orderDetails order.Order) bool {
	// Allow pending and payment processing orders
	if orderDetails.Status == order.OrderStatusPending ||
		orderDetails.Status == order.OrderStatusPaymentProcessing {
		return true
	}

	// Allow confirmed orders ONLY if payment failed
	if orderDetails.Status == order.OrderStatusConfirmed &&
		orderDetails.PaymentStatus == order.PaymentStatusFailed {
		return true
	}

	// Don't allow if order is cancelled, shipped, delivered, etc.
	return false
}

// ENHANCED: Handle existing payments for retry scenarios
func (r *RazorpayService) handleExistingPayments(orderID uint) error {
	var existingPayments []order.Payment
	err := r.db.Where("order_id = ?", orderID).Order("created_at DESC").Find(&existingPayments).Error
	if err != nil {
		return err
	}

	for _, payment := range existingPayments {
		switch payment.Status {
		case order.PaymentStatusPaid:
			return fmt.Errorf("payment already completed for this order")
		case order.PaymentStatusProcessing:
			// Check if payment is stuck (>15 minutes old)
			if time.Since(payment.CreatedAt) > 15*time.Minute {
				// Mark as expired/failed
				r.db.Model(&payment).Updates(map[string]interface{}{
					"status":         order.PaymentStatusFailed,
					"failure_reason": "Payment timeout - expired",
					"updated_at":     time.Now().UTC(),
				})
			} else {
				return fmt.Errorf("payment is currently being processed")
			}
		}
	}

	return nil
}

// Check if this is a retry attempt
func (r *RazorpayService) isRetryAttempt(orderID uint) bool {
	var count int64
	r.db.Model(&order.Payment{}).Where("order_id = ?", orderID).Count(&count)
	return count > 0
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
		return fmt.Errorf("payment amount mismatch. Expected: %d, Got: %d",
			orderDetails.TotalAmount, payment.Amount)
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
			"updated_at":     time.Now().UTC(),
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

	// Commit transaction
	err = tx.Commit().Error
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Send success email asynchronously
	go r.sendPaymentSuccessEmail(req.OrderID)

	return nil
}

// HandlePaymentFailure handles payment failure scenarios
func (r *RazorpayService) HandlePaymentFailure(orderID uint, reason, code string) error {
	// Start transaction
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update payment status to failed
	err := tx.Model(&order.Payment{}).
		Where("order_id = ?", orderID).
		Updates(map[string]interface{}{
			"status":         order.PaymentStatusFailed,
			"failure_reason": reason,
			"failure_code":   code,
			"updated_at":     time.Now().UTC(),
		}).Error

	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update payment: %w", err)
	}

	// Reset order status to confirmed but mark payment as failed to allow retry
	err = tx.Model(&order.Order{}).
		Where("id = ?", orderID).
		Updates(map[string]interface{}{
			"status":         order.OrderStatusConfirmed, // Keep confirmed to allow retry
			"payment_status": order.PaymentStatusFailed,
			"updated_at":     time.Now().UTC(),
		}).Error

	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update order: %w", err)
	}

	// Add status history
	statusHistory := order.OrderStatusHistory{
		OrderID:   orderID,
		Status:    order.OrderStatusConfirmed, // Keep as confirmed for retry
		Comment:   fmt.Sprintf("Payment failed: %s (%s)", reason, code),
		CreatedBy: 0,
		CreatedAt: time.Now().UTC(),
	}

	err = tx.Create(&statusHistory).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create status history: %w", err)
	}

	// Commit transaction
	err = tx.Commit().Error
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Send failure email asynchronously
	go r.sendPaymentFailureEmail(orderID, reason)

	return nil
}

// GetPaymentStatus retrieves payment status for an order
func (r *RazorpayService) GetPaymentStatus(orderID uint) (*order.Payment, error) {
	var payment order.Payment
	err := r.db.Where("order_id = ?", orderID).
		Order("created_at DESC").
		First(&payment).Error

	if err != nil {
		return nil, fmt.Errorf("payment not found: %w", err)
	}

	return &payment, nil
}

// CreateRefund creates a refund for a payment
func (r *RazorpayService) CreateRefund(paymentID string, amount int64, reason string) error {
	refundReq := RefundRequest{
		Amount: amount,
		Speed:  "normal",
		Notes: map[string]interface{}{
			"reason": reason,
		},
		Receipt: fmt.Sprintf("refund_%d", time.Now().Unix()),
	}

	endpoint := fmt.Sprintf("/payments/%s/refund", paymentID)
	response, err := r.makeAPICall("POST", endpoint, refundReq)
	if err != nil {
		return fmt.Errorf("failed to create refund: %w", err)
	}

	var refund RazorpayRefund
	err = json.Unmarshal(response, &refund)
	if err != nil {
		return fmt.Errorf("failed to parse refund response: %w", err)
	}

	// Update payment record with refund info
	var payment order.Payment
	err = r.db.Where("payment_provider_id = ?", paymentID).First(&payment).Error
	if err != nil {
		return fmt.Errorf("payment not found: %w", err)
	}

	err = r.db.Model(&payment).Updates(map[string]interface{}{
		"status":     order.PaymentStatusRefunded,
		"refund_id":  refund.ID,
		"updated_at": time.Now().UTC(),
	}).Error

	if err != nil {
		return fmt.Errorf("failed to update payment record: %w", err)
	}

	return nil
}

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

// Email notification helpers
func (r *RazorpayService) sendPaymentSuccessEmail(orderID uint) {
	var orderRecord order.Order
	if err := r.db.Where("id = ?", orderID).First(&orderRecord).Error; err != nil {
		log.Printf("Failed to get order for payment success email: %v", err)
		return
	}

	var userRecord user.User
	if err := r.db.Select("email, first_name, last_name").Where("id = ?", orderRecord.UserID).First(&userRecord).Error; err != nil {
		log.Printf("Failed to get user for payment success email: %v", err)
		return
	}

	// Send email using email service
	emailData := map[string]interface{}{
		"UserName":    fmt.Sprintf("%s %s", userRecord.FirstName, userRecord.LastName),
		"OrderNumber": orderRecord.OrderNumber,
		"Amount":      float64(orderRecord.TotalAmount) / 100,
		"OrderURL":    fmt.Sprintf("%s/orders/%d", r.config.App.FrontendURL, orderID),
	}

	err := r.emailService.SendTemplateEmail(
		userRecord.Email,
		"Payment Successful - "+orderRecord.OrderNumber,
		"payment_success",
		emailData,
	)

	if err != nil {
		log.Printf("Failed to send payment success email: %v", err)
	}
}

func (r *RazorpayService) sendPaymentFailureEmail(orderID uint, reason string) {
	var orderRecord order.Order
	if err := r.db.Where("id = ?", orderID).First(&orderRecord).Error; err != nil {
		log.Printf("Failed to get order for payment failure email: %v", err)
		return
	}

	var userRecord user.User
	if err := r.db.Select("email, first_name, last_name").Where("id = ?", orderRecord.UserID).First(&userRecord).Error; err != nil {
		log.Printf("Failed to get user for payment failure email: %v", err)
		return
	}

	// Send email using email service
	emailData := map[string]interface{}{
		"UserName":      fmt.Sprintf("%s %s", userRecord.FirstName, userRecord.LastName),
		"OrderNumber":   orderRecord.OrderNumber,
		"Amount":        float64(orderRecord.TotalAmount) / 100,
		"PaymentMethod": "Razorpay",
		"Reason":        reason,
		"OrderURL":      fmt.Sprintf("%s/orders/%d", r.config.App.FrontendURL, orderID),
		"SupportURL":    fmt.Sprintf("%s/support", r.config.App.FrontendURL),
		"Year":          time.Now().Year(),
		"SiteName":      r.config.App.Name,
	}

	err := r.emailService.SendTemplateEmail(
		userRecord.Email,
		"Payment Failed - "+orderRecord.OrderNumber,
		"payment_failed",
		emailData,
	)

	if err != nil {
		log.Printf("Failed to send payment failure email: %v", err)
	}
}

// structToJSON converts struct to JSON string
func (r *RazorpayService) structToJSON(data interface{}) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(jsonData)
}
