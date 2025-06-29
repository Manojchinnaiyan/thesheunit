// Step 2: Fix the UserAddressHandler
// File: internal/interfaces/http/handlers/user_address.go

package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/your-org/ecommerce-backend/internal/config"
	"github.com/your-org/ecommerce-backend/internal/domain/user"
	"github.com/your-org/ecommerce-backend/internal/interfaces/http/middleware"
	"gorm.io/gorm"
)

// UserAddressHandler handles user address endpoints
type UserAddressHandler struct {
	addressService *user.AddressService
	config         *config.Config
}

// NewUserAddressHandler creates a new user address handler
func NewUserAddressHandler(db *gorm.DB, cfg *config.Config) *UserAddressHandler {
	return &UserAddressHandler{
		addressService: user.NewAddressService(db, cfg),
		config:         cfg,
	}
}

// GetAddresses handles GET /users/addresses
func (h *UserAddressHandler) GetAddresses(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// Get address type filter (optional)
	addressType := c.Query("type") // shipping, billing, or empty for all

	addresses, err := h.addressService.GetUserAddresses(userID, addressType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve addresses",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Addresses retrieved successfully",
		"data":    addresses,
	})
}

// GetAddress handles GET /users/addresses/:id
func (h *UserAddressHandler) GetAddress(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	addressIDParam := c.Param("id")
	addressID, err := strconv.ParseUint(addressIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid address ID",
		})
		return
	}

	address, err := h.addressService.GetAddress(userID, uint(addressID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Address retrieved successfully",
		"data":    address,
	})
}

// CreateAddress handles POST /users/addresses
func (h *UserAddressHandler) CreateAddress(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req user.CreateAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	address, err := h.addressService.CreateAddress(userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Address created successfully",
		"data":    address,
	})
}

// UpdateAddress handles PUT /users/addresses/:id
func (h *UserAddressHandler) UpdateAddress(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	addressIDParam := c.Param("id")
	addressID, err := strconv.ParseUint(addressIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid address ID",
		})
		return
	}

	var req user.UpdateAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	address, err := h.addressService.UpdateAddress(userID, uint(addressID), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Address updated successfully",
		"data":    address,
	})
}

// DeleteAddress handles DELETE /users/addresses/:id
func (h *UserAddressHandler) DeleteAddress(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	addressIDParam := c.Param("id")
	addressID, err := strconv.ParseUint(addressIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid address ID",
		})
		return
	}

	err = h.addressService.DeleteAddress(userID, uint(addressID))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Address deleted successfully",
	})
}

func (h *UserAddressHandler) SetDefaultAddress(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	addressIDParam := c.Param("id")
	addressID, err := strconv.ParseUint(addressIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid address ID",
		})
		return
	}

	var req struct {
		Type string `json:"type"` // Make it optional, no binding:"required"
	}

	// Try to bind JSON, but don't fail if empty body
	if err := c.ShouldBindJSON(&req); err != nil {
		// If no JSON body provided, default to "shipping"
		req.Type = "shipping"
	}

	// Validate type if provided
	if req.Type != "" && req.Type != "shipping" && req.Type != "billing" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid address type. Must be 'shipping' or 'billing'",
		})
		return
	}

	// Default to shipping if empty
	if req.Type == "" {
		req.Type = "shipping"
	}

	// Call service to set default
	err = h.addressService.SetDefaultAddress(userID, uint(addressID), req.Type)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Get the updated address to return
	address, err := h.addressService.GetAddress(userID, uint(addressID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve updated address",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Default address updated successfully",
		"data":    address,
	})
}
