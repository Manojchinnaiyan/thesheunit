// internal/interfaces/http/handlers/payment.go - Fixed Implementation
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
	// Get user ID from context
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// Parse request body
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

	// Validate order ID
	if req.OrderID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Order ID must be greater than 0",
		})
		return
	}

	// Verify order exists and belongs to user
	var orderRecord order.Order
	result := h.db.Where("id = ? AND user_id = ?", req.OrderID, userID).First(&orderRecord)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Order not found or access denied",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve order",
			})
		}
		return
	}

	// Check if order is in correct status for payment
	validStatuses := []order.OrderStatus{
		order.OrderStatusPending,
		order.OrderStatusCancelled, // Allow retry for cancelled/failed payments
	}

	isValidStatus := false
	for _, status := range validStatuses {
		if orderRecord.Status == status {
			isValidStatus = true
			break
		}
	}

	if !isValidStatus {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":          "Order is not eligible for payment",
			"current_status": string(orderRecord.Status),
			"valid_statuses": []string{
				string(order.OrderStatusPending),
				string(order.OrderStatusCancelled),
			},
		})
		return
	}

	// Check if order has valid amount
	if orderRecord.TotalAmount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Order amount must be greater than 0",
		})
		return
	}

	// Create payment order through Razorpay service
	paymentResponse, err := h.razorpayService.CreatePaymentOrder(req.OrderID)
	if err != nil {
		// Log the error for debugging
		fmt.Printf("Payment initiation error for order %d: %v\n", req.OrderID, err)

		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Return success response
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

	// Verify payment through Razorpay service
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
			"order_id":            req.OrderID,
			"razorpay_order_id":   req.RazorpayOrderID,
			"razorpay_payment_id": req.RazorpayPaymentID,
			"status":              "verified",
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

	// Handle payment failure through service
	err := h.razorpayService.HandlePaymentFailure(req.OrderID, req.Reason, req.Code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Payment failure recorded successfully",
		"data": gin.H{
			"order_id": req.OrderID,
			"status":   "failed",
			"reason":   req.Reason,
		},
	})
}

// GetPaymentStatus handles GET /payment/status/:order_id
func (h *PaymentHandler) GetPaymentStatus(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	orderIDStr := c.Param("order_id")
	orderID, err := strconv.ParseUint(orderIDStr, 10, 32)
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
			"key_id": func() string {
				if razorpayEnabled {
					return h.config.External.Razorpay.KeyID
				}
				return ""
			}(),
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

// WebhookHandler handles POST /webhooks/razorpay
func (h *PaymentHandler) WebhookHandler(c *gin.Context) {
	// Read the request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to read request body",
		})
		return
	}

	// Get signature from header
	signature := c.GetHeader("X-Razorpay-Signature")
	if signature == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing signature header",
		})
		return
	}

	// Verify webhook signature
	if !h.verifyWebhookSignature(string(body), signature) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid signature",
		})
		return
	}

	// Parse webhook data
	var webhookData map[string]interface{}
	if err := json.Unmarshal(body, &webhookData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON payload",
		})
		return
	}

	// Get event type
	eventType, ok := webhookData["event"].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing event type",
		})
		return
	}

	// Handle different webhook events
	switch eventType {
	case "payment.captured":
		h.handlePaymentCaptured(webhookData)
	case "payment.failed":
		h.handlePaymentFailed(webhookData)
	case "order.paid":
		h.handleOrderPaid(webhookData)
	default:
		// Log unknown event type but don't fail
		fmt.Printf("Unknown webhook event: %s\n", eventType)
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "received",
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

	// Get total count
	query.Count(&total)

	// Apply pagination
	offset := (page - 1) * limit
	query = query.Offset(offset).Limit(limit).Order("created_at DESC")

	if err := query.Find(&payments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve payments",
		})
		return
	}

	// Calculate pagination info
	totalPages := (int(total) + limit - 1) / limit
	hasNext := page < totalPages
	hasPrev := page > 1

	c.JSON(http.StatusOK, gin.H{
		"data": payments,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
			"has_next":    hasNext,
			"has_prev":    hasPrev,
		},
	})
}

// AdminGetPaymentDetails handles GET /admin/payments/:id
func (h *PaymentHandler) AdminGetPaymentDetails(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid payment ID",
		})
		return
	}

	var payment order.Payment
	result := h.db.Where("id = ?", id).First(&payment)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Payment not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": payment,
	})
}

// AdminRefundPayment handles POST /admin/payments/:paymentId/refund
func (h *PaymentHandler) AdminRefundPayment(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Payment refund endpoint - Coming soon",
	})
}

// AdminGetPaymentStats handles GET /admin/payments/stats
func (h *PaymentHandler) AdminGetPaymentStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Payment stats endpoint - Coming soon",
	})
}

// RazorpayWebhook handles POST /webhooks/razorpay (alias for WebhookHandler)
func (h *PaymentHandler) RazorpayWebhook(c *gin.Context) {
	h.WebhookHandler(c)
}

// --- WEBHOOK EVENT HANDLERS ---

func (h *PaymentHandler) handlePaymentCaptured(data map[string]interface{}) {
	payload := data["payload"].(map[string]interface{})
	paymentEntity := payload["payment"].(map[string]interface{})["entity"].(map[string]interface{})

	// Extract payment ID and order ID
	_ = paymentEntity["id"].(string) // paymentID (not used currently)
	orderID := paymentEntity["order_id"].(string)

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
