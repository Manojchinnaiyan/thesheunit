// internal/interfaces/http/handlers/wishlist.go
package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/your-org/ecommerce-backend/internal/config"
	"github.com/your-org/ecommerce-backend/internal/domain/wishlist"
	"github.com/your-org/ecommerce-backend/internal/interfaces/http/middleware"
	"gorm.io/gorm"
)

// WishlistHandler handles wishlist endpoints
type WishlistHandler struct {
	wishlistService *wishlist.Service
	config          *config.Config
}

// NewWishlistHandler creates a new wishlist handler
func NewWishlistHandler(db *gorm.DB, redisClient *redis.Client, cfg *config.Config) *WishlistHandler {
	return &WishlistHandler{
		wishlistService: wishlist.NewService(db, redisClient, cfg),
		config:          cfg,
	}
}

// GetWishlist handles GET /wishlist
func (h *WishlistHandler) GetWishlist(c *gin.Context) {
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

	sortBy := c.Query("sort_by")
	if sortBy == "" {
		sortBy = "created_at"
	}

	sortOrder := c.Query("sort_order")
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	wishlistResponse, err := h.wishlistService.GetWishlist(userID, page, limit, sortBy, sortOrder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve wishlist",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Wishlist retrieved successfully",
		"data":    wishlistResponse,
	})
}

// AddToWishlist handles POST /wishlist/items
func (h *WishlistHandler) AddToWishlist(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req wishlist.AddToWishlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	item, err := h.wishlistService.AddToWishlist(userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Item added to wishlist successfully",
		"data":    item,
	})
}

// RemoveFromWishlist handles DELETE /wishlist/items/:id
func (h *WishlistHandler) RemoveFromWishlist(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

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

	err = h.wishlistService.RemoveFromWishlist(userID, uint(productID), variantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Item removed from wishlist successfully",
	})
}

// ClearWishlist handles DELETE /wishlist
func (h *WishlistHandler) ClearWishlist(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	err := h.wishlistService.ClearWishlist(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to clear wishlist",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Wishlist cleared successfully",
	})
}

// GetWishlistCount handles GET /wishlist/count
func (h *WishlistHandler) GetWishlistCount(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	count, err := h.wishlistService.GetWishlistCount(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get wishlist count",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Wishlist count retrieved successfully",
		"data": gin.H{
			"count": count,
		},
	})
}

// MoveToCart handles POST /wishlist/items/:id/move-to-cart
func (h *WishlistHandler) MoveToCart(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	productIDParam := c.Param("id")
	productID, err := strconv.ParseUint(productIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid product ID",
		})
		return
	}

	var req struct {
		Quantity         int   `json:"quantity" binding:"required,min=1"`
		ProductVariantID *uint `json:"product_variant_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	err = h.wishlistService.MoveToCart(userID, uint(productID), req.ProductVariantID, req.Quantity)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Item moved to cart successfully",
	})
}

// CheckItemInWishlist handles GET /wishlist/check/:id
func (h *WishlistHandler) CheckItemInWishlist(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

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

	isInWishlist, err := h.wishlistService.IsInWishlist(userID, uint(productID), variantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to check wishlist status",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Wishlist status checked successfully",
		"data": gin.H{
			"in_wishlist": isInWishlist,
			"product_id":  productID,
			"variant_id":  variantID,
		},
	})
}

// BulkAddToWishlist handles POST /wishlist/bulk-add
func (h *WishlistHandler) BulkAddToWishlist(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req struct {
		ProductIDs []uint `json:"product_ids" binding:"required,min=1,max=50"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	result, err := h.wishlistService.BulkAddToWishlist(userID, req.ProductIDs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Items processed for wishlist",
		"data":    result,
	})
}

// GetWishlistSummary handles GET /wishlist/summary
func (h *WishlistHandler) GetWishlistSummary(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	summary, err := h.wishlistService.GetWishlistSummary(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get wishlist summary",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Wishlist summary retrieved successfully",
		"data":    summary,
	})
}
