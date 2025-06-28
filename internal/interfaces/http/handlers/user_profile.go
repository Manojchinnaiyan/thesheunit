// internal/interfaces/http/handlers/user_profile.go
package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/your-org/ecommerce-backend/internal/config"
	"github.com/your-org/ecommerce-backend/internal/domain/user"
	"github.com/your-org/ecommerce-backend/internal/interfaces/http/middleware"
	"gorm.io/gorm"
)

// UserProfileHandler handles user profile endpoints
type UserProfileHandler struct {
	userService *user.Service
	config      *config.Config
	db          *gorm.DB
}

// NewUserProfileHandler creates a new user profile handler
func NewUserProfileHandler(db *gorm.DB, cfg *config.Config) *UserProfileHandler {
	return &UserProfileHandler{
		userService: user.NewService(db, cfg),
		config:      cfg,
		db:          db,
	}
}

// GetProfile handles GET /users/profile
func (h *UserProfileHandler) GetProfile(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	profile, err := h.userService.GetProfile(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile retrieved successfully",
		"data":    profile,
	})
}

// UpdateProfile handles PUT /users/profile
func (h *UserProfileHandler) UpdateProfile(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req struct {
		FirstName   *string `json:"first_name,omitempty"`
		LastName    *string `json:"last_name,omitempty"`
		Phone       *string `json:"phone,omitempty"`
		DateOfBirth *string `json:"date_of_birth,omitempty"` // Format: "2006-01-02"
		Avatar      *string `json:"avatar,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	// Convert request to updates map
	updates := make(map[string]interface{})

	if req.FirstName != nil {
		updates["first_name"] = *req.FirstName
	}
	if req.LastName != nil {
		updates["last_name"] = *req.LastName
	}
	if req.Phone != nil {
		updates["phone"] = *req.Phone
	}
	if req.Avatar != nil {
		updates["avatar"] = *req.Avatar
	}
	if req.DateOfBirth != nil && *req.DateOfBirth != "" {
		// Parse date if provided
		if parsedDate, err := parseDate(*req.DateOfBirth); err == nil {
			updates["date_of_birth"] = parsedDate
		}
	}

	profile, err := h.userService.UpdateProfile(userID, updates)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully",
		"data":    profile,
	})
}

// GetDashboard handles GET /users/dashboard
func (h *UserProfileHandler) GetDashboard(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// Get user profile
	profile, err := h.userService.GetProfile(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get user profile",
		})
		return
	}

	// Get user statistics
	stats, err := h.getUserDashboardStats(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get user statistics",
		})
		return
	}

	dashboardData := map[string]interface{}{
		"user":  profile,
		"stats": stats,
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Dashboard data retrieved successfully",
		"data":    dashboardData,
	})
}

// ChangePassword handles PUT /users/change-password
func (h *UserProfileHandler) ChangePassword(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=8"`
		ConfirmPassword string `json:"confirm_password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	// Validate passwords match
	if req.NewPassword != req.ConfirmPassword {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "New password and confirmation do not match",
		})
		return
	}

	err := h.userService.ChangePassword(userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password changed successfully",
	})
}

// GetAccount handles GET /users/account
func (h *UserProfileHandler) GetAccount(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// Get user with addresses
	var userWithAddresses user.User
	if err := h.db.Preload("Addresses").Where("id = ?", userID).First(&userWithAddresses).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	// Clear password
	userWithAddresses.Password = ""

	c.JSON(http.StatusOK, gin.H{
		"message": "Account information retrieved successfully",
		"data":    userWithAddresses,
	})
}

// Helper functions

// getUserDashboardStats gets dashboard statistics for a user
func (h *UserProfileHandler) getUserDashboardStats(userID uint) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get order count
	var orderCount int64
	h.db.Raw("SELECT COUNT(*) FROM orders WHERE user_id = ?", userID).Scan(&orderCount)
	stats["total_orders"] = orderCount

	// Get total spent
	var totalSpent int64
	h.db.Raw("SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE user_id = ? AND status NOT IN ('cancelled', 'failed')", userID).Scan(&totalSpent)
	stats["total_spent"] = totalSpent
	stats["total_spent_formatted"] = h.formatUserCurrency(totalSpent)

	// Get pending orders
	var pendingOrders int64
	h.db.Raw("SELECT COUNT(*) FROM orders WHERE user_id = ? AND status IN ('pending', 'processing')", userID).Scan(&pendingOrders)
	stats["pending_orders"] = pendingOrders

	// Get address count
	var addressCount int64
	h.db.Raw("SELECT COUNT(*) FROM addresses WHERE user_id = ?", userID).Scan(&addressCount)
	stats["address_count"] = addressCount

	// Get wishlist count (if wishlist exists)
	var wishlistCount int64
	h.db.Raw("SELECT COUNT(*) FROM wishlist_items WHERE user_id = ?", userID).Scan(&wishlistCount)
	stats["wishlist_count"] = wishlistCount

	// Get recent orders
	type RecentOrder struct {
		ID          uint   `json:"id"`
		OrderNumber string `json:"order_number"`
		Status      string `json:"status"`
		TotalAmount int64  `json:"total_amount"`
		CreatedAt   string `json:"created_at"`
	}

	var recentOrders []RecentOrder
	h.db.Raw(`
		SELECT id, order_number, status, total_amount, created_at
		FROM orders 
		WHERE user_id = ? 
		ORDER BY created_at DESC 
		LIMIT 5
	`, userID).Scan(&recentOrders)

	// Format currency for recent orders
	for i := range recentOrders {
		recentOrders[i].TotalAmount = recentOrders[i].TotalAmount // Keep raw value
	}

	stats["recent_orders"] = recentOrders

	return stats, nil
}

// parseDate parses date string to time.Time
func parseDate(dateStr string) (*time.Time, error) {
	if dateStr == "" {
		return nil, nil
	}

	parsedTime, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, err
	}

	return &parsedTime, nil
}

// formatUserCurrency formats cents to currency string (renamed to avoid conflict)
func (h *UserProfileHandler) formatUserCurrency(cents int64) string {
	return strconv.FormatFloat(float64(cents)/100, 'f', 2, 64)
}
