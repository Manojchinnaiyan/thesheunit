// internal/interfaces/http/handlers/payment.go
package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/your-org/ecommerce-backend/internal/config"
	"github.com/your-org/ecommerce-backend/internal/domain/payment"
	"github.com/your-org/ecommerce-backend/internal/interfaces/http/middleware"
	"gorm.io/gorm"
)

// PaymentHandler handles payment endpoints
type PaymentHandler struct {
	razorpayService *payment.RazorpayService
	config          *config.Config
}

// NewPaymentHandler creates a new payment handler
func NewPaymentHandler(db *gorm.DB, redisClient *redis.Client, cfg *config.Config) *PaymentHandler {
	return &PaymentHandler{
		razorpayService: payment.NewRazorpayService(db, cfg),
		config:          cfg,
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

	// TODO: Verify order belongs to user (add this validation)

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

	// TODO: Verify order belongs to user

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

	// TODO: Verify order belongs to user

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

	// TODO: Verify order belongs to user

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
	methods := []gin.H{
		{
			"id":          "razorpay",
			"name":        "Razorpay",
			"description": "Pay using Credit Card, Debit Card, NetBanking, UPI, or Wallets",
			"logo":        "/images/razorpay-logo.png",
			"enabled":     true,
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
	// TODO: Implement admin payment listing
	c.JSON(http.StatusOK, gin.H{
		"message": "Admin payment listing - Coming soon",
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

	err := h.razorpayService.CreateRefund(paymentID, req.Amount, req.Reason)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to create refund: " + err.Error(),
		})
		return
	}

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
	// TODO: Implement payment statistics
	c.JSON(http.StatusOK, gin.H{
		"message": "Payment statistics - Coming soon",
		"data": gin.H{
			"total_payments":      150,
			"successful_payments": 142,
			"failed_payments":     8,
			"total_amount":        2500000, // In paise
			"refunds_issued":      5,
			"refund_amount":       50000,
		},
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
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to read request body",
		})
		return
	}

	// TODO: Verify webhook signature
	// TODO: Process webhook events (payment.captured, payment.failed, etc.)

	c.JSON(http.StatusOK, gin.H{
		"status": "received",
	})
}
