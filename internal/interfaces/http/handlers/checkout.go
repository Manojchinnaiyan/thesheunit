// internal/interfaces/http/handlers/checkout.go
package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/your-org/ecommerce-backend/internal/config"
	"github.com/your-org/ecommerce-backend/internal/domain/checkout"
	"github.com/your-org/ecommerce-backend/internal/interfaces/http/middleware"
	"gorm.io/gorm"
)

// CheckoutHandler handles checkout endpoints
type CheckoutHandler struct {
	checkoutService *checkout.Service
	config          *config.Config
}

// NewCheckoutHandler creates a new checkout handler
func NewCheckoutHandler(db *gorm.DB, redisClient *redis.Client, cfg *config.Config) *CheckoutHandler {
	return &CheckoutHandler{
		checkoutService: checkout.NewService(db, redisClient, cfg),
		config:          cfg,
	}
}

// GetShippingMethods handles GET /checkout/shipping-methods
func (h *CheckoutHandler) GetShippingMethods(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// Get shipping address from query params or user's default
	var req struct {
		AddressID *uint   `form:"address_id"`
		Country   *string `form:"country"`
		State     *string `form:"state"`
		City      *string `form:"city"`
	}

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid query parameters",
			"details": err.Error(),
		})
		return
	}

	methods, err := h.checkoutService.GetShippingMethods(userID, req.AddressID, req.Country, req.State, req.City)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Shipping methods retrieved successfully",
		"data":    methods,
	})
}

// CalculateShipping handles POST /checkout/calculate-shipping
func (h *CheckoutHandler) CalculateShipping(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req checkout.ShippingCalculationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	calculation, err := h.checkoutService.CalculateShipping(userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Shipping calculated successfully",
		"data":    calculation,
	})
}

// ApplyCoupon handles POST /checkout/apply-coupon
func (h *CheckoutHandler) ApplyCoupon(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req struct {
		CouponCode string `json:"coupon_code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	result, err := h.checkoutService.ApplyCoupon(userID, req.CouponCode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Coupon applied successfully",
		"data":    result,
	})
}

// RemoveCoupon handles POST /checkout/remove-coupon
func (h *CheckoutHandler) RemoveCoupon(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	err := h.checkoutService.RemoveCoupon(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Coupon removed successfully",
	})
}

// GetTaxCalculation handles POST /checkout/calculate-tax
func (h *CheckoutHandler) GetTaxCalculation(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req checkout.TaxCalculationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	taxInfo, err := h.checkoutService.CalculateTax(userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Tax calculated successfully",
		"data":    taxInfo,
	})
}

// GetCheckoutSummary handles GET /checkout/summary
func (h *CheckoutHandler) GetCheckoutSummary(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// Get optional parameters
	shippingMethodParam := c.Query("shipping_method")
	addressIDParam := c.Query("address_id")
	couponCode := c.Query("coupon_code")

	var addressID *uint
	if addressIDParam != "" {
		if id, err := strconv.ParseUint(addressIDParam, 10, 32); err == nil {
			addr := uint(id)
			addressID = &addr
		}
	}

	summary, err := h.checkoutService.GetCheckoutSummary(userID, addressID, shippingMethodParam, couponCode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Checkout summary retrieved successfully",
		"data":    summary,
	})
}

// ValidateCheckout handles POST /checkout/validate
func (h *CheckoutHandler) ValidateCheckout(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req checkout.CheckoutValidationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	validation, err := h.checkoutService.ValidateCheckout(userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if !validation.IsValid {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "Checkout validation failed",
			"validation_errors": validation.Errors,
			"data":              validation,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Checkout validation successful",
		"data":    validation,
	})
}
