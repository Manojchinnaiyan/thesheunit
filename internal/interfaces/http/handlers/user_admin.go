// internal/interfaces/http/handlers/user_admin.go
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

// UserAdminHandler handles admin user management endpoints
type UserAdminHandler struct {
	adminService *user.AdminService
	config       *config.Config
}

// NewUserAdminHandler creates a new user admin handler
func NewUserAdminHandler(db *gorm.DB, cfg *config.Config) *UserAdminHandler {
	return &UserAdminHandler{
		adminService: user.NewAdminService(db, cfg),
		config:       cfg,
	}
}

// GetUsers handles GET /admin/users
func (h *UserAdminHandler) GetUsers(c *gin.Context) {
	var req user.UserListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid query parameters",
			"details": err.Error(),
		})
		return
	}

	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}

	response, err := h.adminService.GetUsers(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve users",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Users retrieved successfully",
		"data":    response,
	})
}

// GetUser handles GET /admin/users/:id
func (h *UserAdminHandler) GetUser(c *gin.Context) {
	idParam := c.Param("id")
	userID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	userWithStats, err := h.adminService.GetUser(uint(userID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User retrieved successfully",
		"data":    userWithStats,
	})
}

// UpdateUserStatus handles PUT /admin/users/:id/status
func (h *UserAdminHandler) UpdateUserStatus(c *gin.Context) {
	adminID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Admin not authenticated",
		})
		return
	}

	idParam := c.Param("id")
	userID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	var req user.UserStatusUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	err = h.adminService.UpdateUserStatus(uint(userID), &req, adminID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	action := "activated"
	if !req.IsActive {
		action = "deactivated"
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User " + action + " successfully",
	})
}

// ToggleUserAdmin handles PUT /admin/users/:id/admin
func (h *UserAdminHandler) ToggleUserAdmin(c *gin.Context) {
	adminID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Admin not authenticated",
		})
		return
	}

	idParam := c.Param("id")
	userID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	var req user.UserAdminToggleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	err = h.adminService.ToggleUserAdmin(uint(userID), &req, adminID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	action := "granted"
	if !req.IsAdmin {
		action = "revoked"
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Admin privileges " + action + " successfully",
	})
}

// ExportUsers handles GET /admin/users/export
func (h *UserAdminHandler) ExportUsers(c *gin.Context) {
	var req user.UserExportRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid query parameters",
			"details": err.Error(),
		})
		return
	}

	// Default to CSV if no format specified
	if req.Format == "" {
		req.Format = "csv"
	}

	data, filename, err := h.adminService.ExportUsers(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to export users: " + err.Error(),
		})
		return
	}

	// Set appropriate headers for file download
	var contentType string
	switch req.Format {
	case "csv":
		contentType = "text/csv"
	case "json":
		contentType = "application/json"
	default:
		contentType = "application/octet-stream"
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Length", strconv.Itoa(len(data)))

	c.Data(http.StatusOK, contentType, data)
}
