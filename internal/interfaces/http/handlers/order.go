// internal/interfaces/http/handlers/order.go
package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/your-org/ecommerce-backend/internal/config"
	"github.com/your-org/ecommerce-backend/internal/domain/cart"
	"github.com/your-org/ecommerce-backend/internal/domain/order"
	"github.com/your-org/ecommerce-backend/internal/interfaces/http/middleware"
	"gorm.io/gorm"
)

// OrderHandler handles order endpoints
type OrderHandler struct {
	orderService *order.Service
	config       *config.Config
}

// NewOrderHandler creates a new order handler
func NewOrderHandler(db *gorm.DB, redisClient *redis.Client, cfg *config.Config) *OrderHandler {
	cartService := cart.NewService(db, redisClient, cfg)
	orderService := order.NewService(db, cfg, cartService)

	return &OrderHandler{
		orderService: orderService,
		config:       cfg,
	}
}

// CreateOrder handles POST /orders
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req order.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	// Get session ID for cart access
	sessionID, _ := c.Cookie("session_id")

	createdOrder, err := h.orderService.CreateOrder(userID, sessionID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Order created successfully",
		"data":    createdOrder,
	})
}

// GetOrders handles GET /orders (user's own orders)
func (h *OrderHandler) GetOrders(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// Parse query parameters
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

	response, err := h.orderService.GetUserOrders(userID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve orders",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Orders retrieved successfully",
		"data":    response,
	})
}

// GetOrder handles GET /orders/:id
func (h *OrderHandler) GetOrder(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	idParam := c.Param("id")
	orderID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid order ID",
		})
		return
	}

	order, err := h.orderService.GetOrder(uint(orderID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Ensure user can only access their own orders
	if order.UserID == nil || *order.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Order retrieved successfully",
		"data":    order,
	})
}

// GetOrderByNumber handles GET /orders/number/:orderNumber
func (h *OrderHandler) GetOrderByNumber(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	orderNumber := c.Param("orderNumber")
	if orderNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Order number is required",
		})
		return
	}

	order, err := h.orderService.GetOrderByNumber(orderNumber)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Ensure user can only access their own orders
	if order.UserID == nil || *order.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Order retrieved successfully",
		"data":    order,
	})
}

// CancelOrder handles PUT /orders/:id/cancel
func (h *OrderHandler) CancelOrder(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	idParam := c.Param("id")
	orderID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid order ID",
		})
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	// Verify order belongs to user
	order, err := h.orderService.GetOrder(uint(orderID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Order not found",
		})
		return
	}

	if order.UserID == nil || *order.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
		})
		return
	}

	err = h.orderService.CancelOrder(uint(orderID), req.Reason, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Order cancelled successfully",
	})
}

// TrackOrder handles GET /orders/:id/track
func (h *OrderHandler) TrackOrder(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	idParam := c.Param("id")
	orderID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid order ID",
		})
		return
	}

	order, err := h.orderService.GetOrder(uint(orderID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Ensure user can only track their own orders
	if order.UserID == nil || *order.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
		})
		return
	}

	// Return tracking information
	trackingInfo := gin.H{
		"order_number":       order.OrderNumber,
		"status":             order.Status,
		"payment_status":     order.PaymentStatus,
		"tracking_number":    order.TrackingNumber,
		"shipping_carrier":   order.ShippingCarrier,
		"shipped_at":         order.ShippedAt,
		"delivered_at":       order.DeliveredAt,
		"status_history":     order.StatusHistory,
		"estimated_delivery": h.calculateEstimatedDelivery(order),
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Order tracking information retrieved successfully",
		"data":    trackingInfo,
	})
}

// --- ADMIN ENDPOINTS ---

// AdminGetOrders handles GET /admin/orders
func (h *OrderHandler) AdminGetOrders(c *gin.Context) {
	var req order.OrderListRequest

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid query parameters",
			"details": err.Error(),
		})
		return
	}

	// Set default values
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}

	response, err := h.orderService.GetOrders(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve orders",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Orders retrieved successfully",
		"data":    response,
	})
}

// AdminGetOrder handles GET /admin/orders/:id
func (h *OrderHandler) AdminGetOrder(c *gin.Context) {
	idParam := c.Param("id")
	orderID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid order ID",
		})
		return
	}

	order, err := h.orderService.GetOrder(uint(orderID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Order retrieved successfully",
		"data":    order,
	})
}

// AdminUpdateOrderStatus handles PUT /admin/orders/:id/status
func (h *OrderHandler) AdminUpdateOrderStatus(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c) // Admin user ID

	idParam := c.Param("id")
	orderID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid order ID",
		})
		return
	}

	var req struct {
		Status          order.OrderStatus `json:"status" binding:"required"`
		Comment         string            `json:"comment"`
		TrackingNumber  string            `json:"tracking_number"`
		ShippingCarrier string            `json:"shipping_carrier"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	// Update order status
	err = h.orderService.UpdateOrderStatus(uint(orderID), req.Status, req.Comment, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Update tracking information if provided
	if req.TrackingNumber != "" || req.ShippingCarrier != "" {
		err = h.updateTrackingInfo(uint(orderID), req.TrackingNumber, req.ShippingCarrier)
		if err != nil {
			// Log error but don't fail the status update
			c.JSON(http.StatusPartialContent, gin.H{
				"message": "Order status updated, but failed to update tracking info",
				"warning": err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Order status updated successfully",
	})
}

// AdminCancelOrder handles PUT /admin/orders/:id/cancel
func (h *OrderHandler) AdminCancelOrder(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c) // Admin user ID

	idParam := c.Param("id")
	orderID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid order ID",
		})
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	err = h.orderService.CancelOrder(uint(orderID), req.Reason, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Order cancelled successfully",
	})
}

// AdminRefundOrder handles POST /admin/orders/:id/refund
func (h *OrderHandler) AdminRefundOrder(c *gin.Context) {
	idParam := c.Param("id")
	orderID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid order ID",
		})
		return
	}

	var req struct {
		Amount int64  `json:"amount"` // Amount in cents, 0 for full refund
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	// TODO: Implement refund processing
	// This would integrate with payment gateway (Stripe, PayPal, etc.)
	c.JSON(http.StatusOK, gin.H{
		"message": "Refund processing - Coming soon",
		"data": gin.H{
			"order_id": orderID,
			"amount":   req.Amount,
			"reason":   req.Reason,
		},
	})
}

// AdminExportOrders handles GET /admin/orders/export
func (h *OrderHandler) AdminExportOrders(c *gin.Context) {
	// TODO: Implement order export functionality
	c.JSON(http.StatusOK, gin.H{
		"message": "Order export - Coming soon",
	})
}

// AdminGetOrderStats handles GET /admin/orders/stats
func (h *OrderHandler) AdminGetOrderStats(c *gin.Context) {
	// TODO: Implement order statistics
	c.JSON(http.StatusOK, gin.H{
		"message": "Order statistics - Coming soon",
		"data": gin.H{
			"total_orders":     100,
			"pending_orders":   15,
			"completed_orders": 75,
			"cancelled_orders": 10,
			"total_revenue":    125000, // In cents
		},
	})
}

// Helper methods

// calculateEstimatedDelivery calculates estimated delivery date
func (h *OrderHandler) calculateEstimatedDelivery(order *order.Order) *string {
	if order.ShippedAt == nil {
		return nil
	}

	// Simple calculation based on shipping method
	var daysToAdd int
	switch order.ShippingMethod {
	case "standard":
		daysToAdd = 5
	case "express":
		daysToAdd = 2
	case "overnight":
		daysToAdd = 1
	default:
		daysToAdd = 5
	}

	estimatedDate := order.ShippedAt.Add(time.Duration(daysToAdd) * 24 * time.Hour)
	dateStr := estimatedDate.Format("2006-01-02")
	return &dateStr
}

// updateTrackingInfo updates order tracking information
func (h *OrderHandler) updateTrackingInfo(orderID uint, trackingNumber, carrier string) error {
	updates := make(map[string]interface{})

	if trackingNumber != "" {
		updates["tracking_number"] = trackingNumber
	}

	if carrier != "" {
		updates["shipping_carrier"] = carrier
	}

	if len(updates) == 0 {
		return nil
	}

	// Use the order service's database connection
	db := h.orderService.GetDB() // We need to add this method to the service
	return db.Model(&order.Order{}).Where("id = ?", orderID).Updates(updates).Error
}
