// internal/interfaces/http/handlers/cart.go
package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/your-org/ecommerce-backend/internal/config"
	"github.com/your-org/ecommerce-backend/internal/domain/cart"
	"github.com/your-org/ecommerce-backend/internal/interfaces/http/middleware"
	"gorm.io/gorm"
)

// CartHandler handles cart endpoints
type CartHandler struct {
	cartService *cart.Service
	config      *config.Config
}

// NewCartHandler creates a new cart handler
func NewCartHandler(db *gorm.DB, redisClient *redis.Client, cfg *config.Config) *CartHandler {
	return &CartHandler{
		cartService: cart.NewService(db, redisClient, cfg),
		config:      cfg,
	}
}

// GetCart handles GET /cart
func (h *CartHandler) GetCart(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)
	sessionID := h.getOrCreateSessionID(c)

	cartResponse, err := h.cartService.GetCart(userID, sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve cart",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Cart retrieved successfully",
		"data":    cartResponse,
	})
}

// AddToCart handles POST /cart/items
func (h *CartHandler) AddToCart(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)
	sessionID := h.getOrCreateSessionID(c)

	var req cart.AddToCartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	cartResponse, err := h.cartService.AddToCart(userID, sessionID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Item added to cart successfully",
		"data":    cartResponse,
	})
}

// UpdateCartItem handles PUT /cart/items/:id
func (h *CartHandler) UpdateCartItem(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)
	sessionID := h.getOrCreateSessionID(c)

	// Parse product ID
	productIDParam := c.Param("id")
	productID, err := strconv.ParseUint(productIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid product ID",
		})
		return
	}

	// Parse variant ID from query parameter (optional)
	var variantID *uint
	if variantIDParam := c.Query("variant_id"); variantIDParam != "" {
		if vID, err := strconv.ParseUint(variantIDParam, 10, 32); err == nil {
			variantIDUint := uint(vID)
			variantID = &variantIDUint
		}
	}

	var req cart.UpdateCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	cartResponse, err := h.cartService.UpdateCartItem(userID, sessionID, uint(productID), variantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Cart item updated successfully",
		"data":    cartResponse,
	})
}

// RemoveFromCart handles DELETE /cart/items/:id
func (h *CartHandler) RemoveFromCart(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)
	sessionID := h.getOrCreateSessionID(c)

	// Parse product ID
	productIDParam := c.Param("id")
	productID, err := strconv.ParseUint(productIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid product ID",
		})
		return
	}

	// Parse variant ID from query parameter (optional)
	var variantID *uint
	if variantIDParam := c.Query("variant_id"); variantIDParam != "" {
		if vID, err := strconv.ParseUint(variantIDParam, 10, 32); err == nil {
			variantIDUint := uint(vID)
			variantID = &variantIDUint
		}
	}

	cartResponse, err := h.cartService.RemoveFromCart(userID, sessionID, uint(productID), variantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Item removed from cart successfully",
		"data":    cartResponse,
	})
}

// ClearCart handles DELETE /cart
func (h *CartHandler) ClearCart(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)
	sessionID := h.getOrCreateSessionID(c)

	err := h.cartService.ClearCart(userID, sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to clear cart",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Cart cleared successfully",
	})
}

// GetCartCount handles GET /cart/count
func (h *CartHandler) GetCartCount(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)
	sessionID := h.getOrCreateSessionID(c)

	count, err := h.cartService.GetCartItemCount(userID, sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get cart count",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Cart count retrieved successfully",
		"data": gin.H{
			"count": count,
		},
	})
}

// MergeGuestCart handles POST /cart/merge - called when user logs in
func (h *CartHandler) MergeGuestCart(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	sessionID := h.getOrCreateSessionID(c)

	err := h.cartService.MergeGuestCartToUser(*userID, sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to merge cart",
		})
		return
	}

	// Return updated cart
	cartResponse, err := h.cartService.GetCart(userID, sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve merged cart",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Guest cart merged successfully",
		"data":    cartResponse,
	})
}

// ValidateCart handles POST /cart/validate - validates cart items before checkout
func (h *CartHandler) ValidateCart(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)
	sessionID := h.getOrCreateSessionID(c)

	cartResponse, err := h.cartService.GetCart(userID, sessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Cart not found",
		})
		return
	}

	// Validate each item in cart
	validationErrors := []string{}

	for _, item := range cartResponse.Items {
		if item.Product == nil {
			validationErrors = append(validationErrors, fmt.Sprintf("Product %d not found", item.ProductID))
			continue
		}

		if !item.Product.IsActive {
			validationErrors = append(validationErrors, fmt.Sprintf("Product '%s' is no longer available", item.Product.Name))
			continue
		}

		// Check inventory
		availableQuantity := item.Product.Quantity
		if item.ProductVariant != nil {
			availableQuantity = item.ProductVariant.Quantity
		}

		if item.Product.TrackQuantity && availableQuantity < item.Quantity {
			validationErrors = append(validationErrors,
				fmt.Sprintf("Product '%s' has insufficient stock. Available: %d, Requested: %d",
					item.Product.Name, availableQuantity, item.Quantity))
		}

		// Check if price has changed
		currentPrice := item.Product.Price
		if item.ProductVariant != nil && item.ProductVariant.Price > 0 {
			currentPrice = item.ProductVariant.Price
		}

		if item.Price != currentPrice {
			validationErrors = append(validationErrors,
				fmt.Sprintf("Price for product '%s' has changed. Current: $%.2f, Cart: $%.2f",
					item.Product.Name, float64(currentPrice)/100, float64(item.Price)/100))
		}
	}

	// Return validation results
	if len(validationErrors) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "Cart validation failed",
			"validation_errors": validationErrors,
			"data":              cartResponse,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Cart validation successful",
		"data":    cartResponse,
	})
}

// getOrCreateSessionID gets session ID from cookie or creates a new one
func (h *CartHandler) getOrCreateSessionID(c *gin.Context) string {
	// Try to get session ID from cookie
	sessionID, err := c.Cookie("session_id")
	if err != nil || sessionID == "" {
		// Generate new session ID
		sessionID = uuid.New().String()

		// Set session cookie (24 hours)
		c.SetCookie("session_id", sessionID, 86400, "/", "", false, true)
	}

	return sessionID
}
