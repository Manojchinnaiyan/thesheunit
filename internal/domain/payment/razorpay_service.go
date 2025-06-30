// internal/domain/payment/razorpay_service.go - DEBUG VERSION
package payment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/your-org/ecommerce-backend/internal/config"
	"github.com/your-org/ecommerce-backend/internal/domain/order"
	"github.com/your-org/ecommerce-backend/internal/pkg/email"
	"gorm.io/gorm"
)

// Debug function to track where errors come from
func debugPrint(message string) {
	_, file, line, _ := runtime.Caller(1)
	fmt.Printf("🐛 DEBUG [%s:%d] %s\n", file, line, message)
}

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
	debugPrint("Starting NewRazorpayService")

	// FORCE hardcoded values
	keyID := "rzp_test_JKEqkGjDCkWMed"
	keySecret := "lrcsYTuX1Y1iQmEzzfmYdxQZ"

	debugPrint(fmt.Sprintf("Forced KeyID: '%s'", keyID))
	debugPrint(fmt.Sprintf("Forced KeySecret: '%s'", keySecret))

	service := &RazorpayService{
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

	debugPrint(fmt.Sprintf("Service created with KeyID: '%s', KeySecret: '%s'", service.keyID, service.keySecret))
	return service
}

// CreatePaymentOrder creates a Razorpay order for payment
func (r *RazorpayService) CreatePaymentOrder(orderID uint) (*PaymentInitiationResponse, error) {
	debugPrint(fmt.Sprintf("Starting CreatePaymentOrder for order ID: %d", orderID))
	debugPrint(fmt.Sprintf("Service KeyID: '%s'", r.keyID))
	debugPrint(fmt.Sprintf("Service KeySecret: '%s'", r.keySecret))

	// Check 1: Validate input
	if orderID == 0 {
		debugPrint("ERROR: Invalid order ID")
		return nil, fmt.Errorf("invalid order ID")
	}

	// Check 2: Get order details
	var orderDetails order.Order
	err := r.db.Preload("Items").Where("id = ?", orderID).First(&orderDetails).Error
	if err != nil {
		debugPrint(fmt.Sprintf("ERROR: Failed to get order: %v", err))
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	debugPrint(fmt.Sprintf("Order found: %s, Amount: %d", orderDetails.OrderNumber, orderDetails.TotalAmount))

	// Check 3: Validate order status
	if orderDetails.Status != order.OrderStatusPending && orderDetails.Status != order.OrderStatusPaymentProcessing {
		debugPrint(fmt.Sprintf("ERROR: Invalid order status: %s", orderDetails.Status))
		return nil, fmt.Errorf("order is not in correct status for payment. Current status: %s", orderDetails.Status)
	}

	// Check 4: Create Razorpay order request
	createReq := CreateOrderRequest{
		Amount:   orderDetails.TotalAmount,
		Currency: "INR",
		Receipt:  orderDetails.OrderNumber,
		Notes: map[string]interface{}{
			"order_id": fmt.Sprintf("%d", orderID),
		},
	}

	debugPrint(fmt.Sprintf("Creating Razorpay order with amount: %d", createReq.Amount))

	// Check 5: Call Razorpay API
	razorpayOrder, err := r.createRazorpayOrder(createReq)
	if err != nil {
		debugPrint(fmt.Sprintf("ERROR: Failed to create Razorpay order: %v", err))
		return nil, fmt.Errorf("failed to create Razorpay order: %w", err)
	}

	debugPrint(fmt.Sprintf("Razorpay order created successfully: %s", razorpayOrder.ID))

	// Return response
	response := &PaymentInitiationResponse{
		RazorpayOrderID: razorpayOrder.ID,
		Amount:          orderDetails.TotalAmount,
		Currency:        "INR",
		Receipt:         orderDetails.OrderNumber,
		KeyID:           r.keyID,
		Notes:           razorpayOrder.Notes,
		OrderDetails:    &orderDetails,
	}

	debugPrint("Payment order creation completed successfully")
	return response, nil
}

// createRazorpayOrder creates order in Razorpay
func (r *RazorpayService) createRazorpayOrder(req CreateOrderRequest) (*RazorpayOrder, error) {
	debugPrint("Starting createRazorpayOrder")
	debugPrint(fmt.Sprintf("About to call makeAPICall with KeyID: '%s'", r.keyID))

	response, err := r.makeAPICall("POST", "/orders", req)
	if err != nil {
		debugPrint(fmt.Sprintf("ERROR in makeAPICall: %v", err))
		return nil, err
	}

	var razorpayOrder RazorpayOrder
	err = json.Unmarshal(response, &razorpayOrder)
	if err != nil {
		debugPrint(fmt.Sprintf("ERROR: Failed to parse response: %v", err))
		return nil, fmt.Errorf("failed to parse Razorpay order response: %w", err)
	}

	debugPrint("Successfully created Razorpay order")
	return &razorpayOrder, nil
}

// makeAPICall makes HTTP calls to Razorpay API
func (r *RazorpayService) makeAPICall(method, endpoint string, data interface{}) ([]byte, error) {
	debugPrint(fmt.Sprintf("Starting makeAPICall: %s %s", method, endpoint))
	debugPrint(fmt.Sprintf("KeyID in makeAPICall: '%s'", r.keyID))
	debugPrint(fmt.Sprintf("KeySecret in makeAPICall: '%s'", r.keySecret))

	// This is where the error might be coming from
	// if r.keyID == "" || r.keySecret == "" {
	// 	debugPrint(fmt.Sprintf("ERROR: Empty credentials - KeyID: '%s', KeySecret: '%s'", r.keyID, r.keySecret))
	// 	return nil, fmt.Errorf("Razorpay API credentials not configured")
	// }

	var reqBody []byte
	var err error

	if data != nil {
		reqBody, err = json.Marshal(data)
		if err != nil {
			debugPrint(fmt.Sprintf("ERROR: Failed to marshal data: %v", err))
			return nil, fmt.Errorf("failed to marshal request data: %w", err)
		}
		debugPrint(fmt.Sprintf("Request body: %s", string(reqBody)))
	}

	req, err := http.NewRequest(method, r.baseURL+endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		debugPrint(fmt.Sprintf("ERROR: Failed to create request: %v", err))
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(r.keyID, r.keySecret)

	debugPrint(fmt.Sprintf("Making HTTP request to: %s", r.baseURL+endpoint))

	// Make request
	resp, err := r.httpClient.Do(req)
	if err != nil {
		debugPrint(fmt.Sprintf("ERROR: HTTP request failed: %v", err))
		return nil, fmt.Errorf("failed to make API call: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	var respBody bytes.Buffer
	_, err = respBody.ReadFrom(resp.Body)
	if err != nil {
		debugPrint(fmt.Sprintf("ERROR: Failed to read response: %v", err))
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	debugPrint(fmt.Sprintf("Response status: %d", resp.StatusCode))
	debugPrint(fmt.Sprintf("Response body: %s", respBody.String()))

	// Check status code
	if resp.StatusCode >= 400 {
		debugPrint(fmt.Sprintf("ERROR: API returned error status: %d", resp.StatusCode))
		return nil, fmt.Errorf("API call failed with status %d: %s", resp.StatusCode, respBody.String())
	}

	debugPrint("API call completed successfully")
	return respBody.Bytes(), nil
}

// Stub implementations for other required methods
func (r *RazorpayService) VerifyPayment(req *PaymentVerificationRequest) error {
	return nil
}

func (r *RazorpayService) HandlePaymentFailure(orderID uint, reason, code string) error {
	return nil
}

func (r *RazorpayService) GetPaymentStatus(orderID uint) (*order.Payment, error) {
	return nil, nil
}

func (r *RazorpayService) CreateRefund(paymentID string, amount int64, reason string) error {
	return nil
}

func (r *RazorpayService) verifySignature(orderID, paymentID, signature string) bool {
	return true
}

func (r *RazorpayService) getPaymentDetails(paymentID string) (*RazorpayPayment, error) {
	return nil, nil
}

func (r *RazorpayService) structToJSON(data interface{}) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(jsonData)
}

// Required structs (add these at the top of the file)
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
