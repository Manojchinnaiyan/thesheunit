// internal/interfaces/http/handlers/payment.go
package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/your-org/ecommerce-backend/internal/config"
	"github.com/your-org/ecommerce-backend/internal/domain/order"
	"github.com/your-org/ecommerce-backend/internal/domain/payment"
	"github.com/your-org/ecommerce-backend/internal/interfaces/http/middleware"
	"gorm.io/gorm"
)

type PaymentHandler struct {
	razorpayService *payment.RazorpayService
	config          *config.Config
	db              *gorm.DB
}

func NewPaymentHandler(db *gorm.DB, redisClient *redis.Client, cfg *config.Config) *PaymentHandler {
	return &PaymentHandler{
		razorpayService: payment.NewRazorpayService(db, cfg),
		config:          cfg,
		db:              db,
	}
}

// InitiatePayment handles POST /payment/initiate
func (h *PaymentHandler) InitiatePayment(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req struct {
		OrderID uint `json:"order_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	// Verify order belongs to user
	var orderRecord order.Order
	result := h.db.Where("id = ? AND user_id = ?", req.OrderID, userID).First(&orderRecord)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Order not found or access denied",
		})
		return
	}

	// Check if order is in correct status for payment
	if orderRecord.Status != order.OrderStatusPending {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Order is not eligible for payment. Current status: " + string(orderRecord.Status),
		})
		return
	}

	// Create payment order in Razorpay
	paymentResponse, err := h.razorpayService.CreatePaymentOrder(req.OrderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Payment initiated successfully",
		"data":    paymentResponse,
	})
}

// VerifyPayment handles POST /payment/verify
func (h *PaymentHandler) VerifyPayment(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req payment.PaymentVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	// Verify order belongs to user
	var orderRecord order.Order
	result := h.db.Where("id = ? AND user_id = ?", req.OrderID, userID).First(&orderRecord)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Order not found or access denied",
		})
		return
	}

	// Verify payment with Razorpay
	err := h.razorpayService.VerifyPayment(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Payment verified successfully",
		"data": gin.H{
			"order_id":    req.OrderID,
			"payment_id":  req.RazorpayPaymentID,
			"status":      "paid",
		},
	})
}

// HandlePaymentFailure handles POST /payment/failure
func (h *PaymentHandler) HandlePaymentFailure(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req struct {
		OrderID uint   `json:"order_id" binding:"required"`
		Reason  string `json:"reason"`
		Code    string `json:"code"`
		Source  string `json:"source"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	// Verify order belongs to user
	var orderRecord order.Order
	result := h.db.Where("id = ? AND user_id = ?", req.OrderID, userID).First(&orderRecord)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Order not found or access denied",
		})
		return
	}

	// Update payment status to failed
	h.db.Model(&order.Payment{}).
		Where("order_id = ?", req.OrderID).
		Updates(map[string]interface{}{
			"status":        order.PaymentStatusFailed,
			"failure_reason": req.Reason,
			"failure_code":   req.Code,
			"updated_at":     time.Now().UTC(),
		})

	// Add status history
	statusHistory := order.OrderStatusHistory{
		OrderID: req.OrderID,
		Status:  order.OrderStatusPaymentFailed,
		Comment: fmt.Sprintf("Payment failed: %s (%s)", req.Reason, req.Code),
		CreatedAt: time.Now().UTC(),
	}
	h.db.Create(&statusHistory)

	c.JSON(http.StatusOK, gin.H{
		"message": "Payment failure recorded",
		"data": gin.H{
			"order_id": req.OrderID,
			"status":   "failed",
		},
	})
}

// GetPaymentStatus handles GET /payment/status/:orderId
func (h *PaymentHandler) GetPaymentStatus(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	orderID, err := strconv.ParseUint(c.Param("orderId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid order ID",
		})
		return
	}

	// Verify order belongs to user
	var orderRecord order.Order
	result := h.db.Where("id = ? AND user_id = ?", uint(orderID), userID).First(&orderRecord)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Order not found or access denied",
		})
		return
	}

	payment, err := h.razorpayService.GetPaymentStatus(uint(orderID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Payment status not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Payment status retrieved successfully",
		"data":    payment,
	})
}

// GetPaymentMethods handles GET /payment/methods
func (h *PaymentHandler) GetPaymentMethods(c *gin.Context) {
	razorpayEnabled := h.config.External.Razorpay.KeyID != "" && h.config.External.Razorpay.KeySecret != ""

	methods := []gin.H{
		{
			"id":          "razorpay",
			"name":        "Razorpay",
			"description": "Pay using Credit Card, Debit Card, NetBanking, UPI, or Wallets",
			"logo":        "/images/razorpay-logo.png",
			"enabled":     razorpayEnabled,
			"key_id":      h.config.External.Razorpay.KeyID,
			"types": []string{
				"card", "netbanking", "upi", "wallet", "emi",
			},
		},
		{
			"id":          "cod",
			"name":        "Cash on Delivery",
			"description": "Pay cash when your order is delivered",
			"logo":        "/images/cod-logo.png",
			"enabled":     true,
			"types": []string{
				"cash",
			},
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Payment methods retrieved successfully",
		"data":    methods,
	})
}

// RazorpayWebhook handles POST /webhooks/razorpay
func (h *PaymentHandler) RazorpayWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Verify webhook signature
	signature := c.GetHeader("X-Razorpay-Signature")
	if !h.verifyWebhookSignature(string(body), signature) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
		return
	}

	// Parse webhook data
	var webhookData map[string]interface{}
	if err := json.Unmarshal(body, &webhookData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	// Handle different webhook events
	event := webhookData["event"].(string)
	switch event {
	case "payment.captured":
		h.handlePaymentCaptured(webhookData)
	case "payment.failed":
		h.handlePaymentFailed(webhookData)
	case "order.paid":
		h.handleOrderPaid(webhookData)
	default:
		// Log unknown event
		fmt.Printf("Unknown webhook event: %s\n", event)
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *PaymentHandler) handlePaymentCaptured(data map[string]interface{}) {
	payload := data["payload"].(map[string]interface{})
	paymentEntity := payload["payment"].(map[string]interface{})["entity"].(map[string]interface{})

	paymentID := paymentEntity["id"].(string)
	orderID := paymentEntity["order_id"].(string)

	// Find payment in database and update status
	var payment order.Payment
	result := h.db.Where("payment_provider_id = ?", orderID).First(&payment)
	if result.Error != nil {
		return // Payment not found
	}

	// Update payment status
	h.db.Model(&payment).Updates(map[string]interface{}{
		"status":           order.PaymentStatusPaid,
		"gateway_response": h.structToJSON(paymentEntity),
		"processed_at":     time.Now().UTC(),
	})

	// Update order status
	h.db.Model(&order.Order{}).Where("id = ?", payment.OrderID).Updates(map[string]interface{}{
		"status":         order.OrderStatusConfirmed,
		"payment_status": order.PaymentStatusPaid,
	})
}

func (h *PaymentHandler) handlePaymentFailed(data map[string]interface{}) {
	payload := data["payload"].(map[string]interface{})
	paymentEntity := payload["payment"].(map[string]interface{})["entity"].(map[string]interface{})

	orderID := paymentEntity["order_id"].(string)

	var payment order.Payment
	result := h.db.Where("payment_provider_id = ?", orderID).First(&payment)
	if result.Error != nil {
		return
	}

	h.db.Model(&payment).Updates(map[string]interface{}{
		"status": order.PaymentStatusFailed,
	})

	h.db.Model(&order.Order{}).Where("id = ?", payment.OrderID).Updates(map[string]interface{}{
		"status":         order.OrderStatusPaymentFailed,
		"payment_status": order.PaymentStatusFailed,
	})
}

func (h *PaymentHandler) handleOrderPaid(data map[string]interface{}) {
	// Handle when entire order is paid
	// Implementation depends on business requirements
}

func (h *PaymentHandler) verifyWebhookSignature(body, signature string) bool {
	if h.config.External.Razorpay.WebhookSecret == "" {
		return h.config.IsDevelopment()
	}

	mac := hmac.New(sha256.New, []byte(h.config.External.Razorpay.WebhookSecret))
	mac.Write([]byte(body))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

func (h *PaymentHandler) structToJSON(data interface{}) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(jsonData)
}
