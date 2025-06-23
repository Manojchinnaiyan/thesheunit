// internal/interfaces/http/handlers/payment.go - Complete implementation
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

// PaymentHandler handles payment endpoints
type PaymentHandler struct {
	razorpayService *payment.RazorpayService
	config          *config.Config
	db              *gorm.DB
}

// NewPaymentHandler creates a new payment handler
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
		// Handle payment failure
		h.razorpayService.HandlePaymentFailure(req.OrderID, err.Error())

		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Payment verification failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Payment verified successfully",
		"data": gin.H{
			"order_id":            req.OrderID,
			"razorpay_payment_id": req.RazorpayPaymentID,
			"status":              "success",
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

	reason := req.Reason
	if reason == "" {
		reason = "Payment failed"
	}
	if req.Code != "" {
		reason += " (Code: " + req.Code + ")"
	}

	err := h.razorpayService.HandlePaymentFailure(req.OrderID, reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to handle payment failure: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Payment failure handled successfully",
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

	orderIDParam := c.Param("orderId")
	orderID, err := strconv.ParseUint(orderIDParam, 10, 32)
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
			"key_id":      h.config.External.Razorpay.KeyID, // Frontend needs this
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

// --- ADMIN ENDPOINTS ---

// AdminGetPayments handles GET /admin/payments
func (h *PaymentHandler) AdminGetPayments(c *gin.Context) {
	// Query parameters for filtering
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	status := c.Query("status")
	orderID := c.Query("order_id")

	var payments []order.Payment
	var total int64

	query := h.db.Model(&order.Payment{})

	// Apply filters
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if orderID != "" {
		query = query.Where("order_id = ?", orderID)
	}

	// Count total
	query.Count(&total)

	// Apply pagination
	offset := (page - 1) * limit
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&payments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve payments",
		})
		return
	}

	// Calculate pagination
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	c.JSON(http.StatusOK, gin.H{
		"message": "Payments retrieved successfully",
		"data": gin.H{
			"payments": payments,
			"pagination": gin.H{
				"page":        page,
				"limit":       limit,
				"total":       total,
				"total_pages": totalPages,
				"has_next":    page < totalPages,
				"has_prev":    page > 1,
			},
		},
	})
}

// AdminRefundPayment handles POST /admin/payments/:paymentId/refund
func (h *PaymentHandler) AdminRefundPayment(c *gin.Context) {
	paymentID := c.Param("paymentId")
	if paymentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Payment ID is required",
		})
		return
	}

	var req struct {
		Amount int64  `json:"amount"` // Amount in paise, 0 for full refund
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	// Find payment record
	var paymentRecord order.Payment
	result := h.db.Where("payment_provider_id = ?", paymentID).First(&paymentRecord)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Payment not found",
		})
		return
	}

	err := h.razorpayService.CreateRefund(paymentID, req.Amount, req.Reason)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to create refund: " + err.Error(),
		})
		return
	}

	// Update payment status to refunded
	h.db.Model(&paymentRecord).Updates(map[string]interface{}{
		"status": order.PaymentStatusRefunded,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Refund created successfully",
		"data": gin.H{
			"payment_id": paymentID,
			"amount":     req.Amount,
			"reason":     req.Reason,
		},
	})
}

// AdminGetPaymentStats handles GET /admin/payments/stats
func (h *PaymentHandler) AdminGetPaymentStats(c *gin.Context) {
	var stats struct {
		TotalPayments      int64 `json:"total_payments"`
		SuccessfulPayments int64 `json:"successful_payments"`
		FailedPayments     int64 `json:"failed_payments"`
		PendingPayments    int64 `json:"pending_payments"`
		TotalAmount        int64 `json:"total_amount"`
		RefundsIssued      int64 `json:"refunds_issued"`
		RefundAmount       int64 `json:"refund_amount"`
	}

	// Get payment statistics
	h.db.Model(&order.Payment{}).Count(&stats.TotalPayments)
	h.db.Model(&order.Payment{}).Where("status = ?", order.PaymentStatusPaid).Count(&stats.SuccessfulPayments)
	h.db.Model(&order.Payment{}).Where("status = ?", order.PaymentStatusFailed).Count(&stats.FailedPayments)
	h.db.Model(&order.Payment{}).Where("status = ?", order.PaymentStatusPending).Count(&stats.PendingPayments)

	// Get total amount from successful payments
	h.db.Model(&order.Payment{}).Where("status = ?", order.PaymentStatusPaid).Select("COALESCE(SUM(amount), 0)").Scan(&stats.TotalAmount)

	// Get refund statistics
	h.db.Model(&order.Payment{}).Where("status = ?", order.PaymentStatusRefunded).Count(&stats.RefundsIssued)
	h.db.Model(&order.Payment{}).Where("status = ?", order.PaymentStatusRefunded).Select("COALESCE(SUM(amount), 0)").Scan(&stats.RefundAmount)

	c.JSON(http.StatusOK, gin.H{
		"message": "Payment statistics retrieved successfully",
		"data":    stats,
	})
}

// RazorpayWebhook handles Razorpay webhooks
func (h *PaymentHandler) RazorpayWebhook(c *gin.Context) {
	// Get webhook signature
	webhookSignature := c.GetHeader("X-Razorpay-Signature")
	if webhookSignature == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing webhook signature",
		})
		return
	}

	// Read raw body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to read request body",
		})
		return
	}

	// Verify webhook signature
	if !h.verifyWebhookSignature(string(body), webhookSignature) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid webhook signature",
		})
		return
	}

	// Parse webhook payload
	var webhookData map[string]interface{}
	if err := json.Unmarshal(body, &webhookData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid webhook payload",
		})
		return
	}

	// Process webhook events
	event := webhookData["event"].(string)

	switch event {
	case "payment.captured":
		h.handlePaymentCaptured(webhookData)
	case "payment.failed":
		h.handlePaymentFailed(webhookData)
	case "order.paid":
		h.handleOrderPaid(webhookData)
	default:
		// Log unknown event but don't fail
		// In production, you might want to log this for monitoring
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "received",
	})
}

// Webhook event handlers

func (h *PaymentHandler) handlePaymentCaptured(data map[string]interface{}) {
	// Extract payment details from webhook payload
	payload := data["payload"].(map[string]interface{})
	paymentEntity := payload["payment"].(map[string]interface{})["entity"].(map[string]interface{})

	paymentID := paymentEntity["id"].(string)
	orderID := paymentEntity["order_id"].(string)

	fmt.Println(paymentID)

	// Find payment in database and update status
	var payment order.Payment
	result := h.db.Where("payment_provider_id = ?", orderID).First(&payment)
	if result.Error != nil {
		return // Payment not found, might be from different system
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
	// Similar to handlePaymentCaptured but for failed payments
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
		"status":         order.OrderStatusPending,
		"payment_status": order.PaymentStatusFailed,
	})
}

func (h *PaymentHandler) handleOrderPaid(data map[string]interface{}) {
	// Handle when entire order is paid (for subscription/recurring payments)
	// Implementation depends on your business requirements
}

// verifyWebhookSignature verifies Razorpay webhook signature
func (h *PaymentHandler) verifyWebhookSignature(body, signature string) bool {
	if h.config.External.Razorpay.WebhookSecret == "" {
		// If webhook secret not configured, skip verification in development
		return h.config.IsDevelopment()
	}

	mac := hmac.New(sha256.New, []byte(h.config.External.Razorpay.WebhookSecret))
	mac.Write([]byte(body))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// structToJSON converts struct to JSON string
func (h *PaymentHandler) structToJSON(data interface{}) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(jsonData)
}
