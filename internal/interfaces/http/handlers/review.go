// internal/interfaces/http/handlers/review.go
package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/your-org/ecommerce-backend/internal/domain/product"
	"github.com/your-org/ecommerce-backend/internal/interfaces/http/middleware"
)

// ReviewHandler handles review-related HTTP requests
type ReviewHandler struct {
	reviewService *product.ReviewService
}

// NewReviewHandler creates a new review handler
func NewReviewHandler(reviewService *product.ReviewService) *ReviewHandler {
	return &ReviewHandler{
		reviewService: reviewService,
	}
}

// CreateReview handles POST /reviews
func (h *ReviewHandler) CreateReview(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req product.CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	review, err := h.reviewService.CreateReview(userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Review created successfully",
		"data":    review,
	})
}

// GetReviews handles GET /reviews
func (h *ReviewHandler) GetReviews(c *gin.Context) {
	var req product.ReviewListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid query parameters",
			"details": err.Error(),
		})
		return
	}

	// Get current user ID if authenticated
	var currentUserID *uint
	if userID, exists := middleware.GetUserIDFromContext(c); exists {
		currentUserID = &userID
	}

	response, err := h.reviewService.GetReviews(&req, currentUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve reviews",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Reviews retrieved successfully",
		"data":    response,
	})
}

// GetReview handles GET /reviews/:id
func (h *ReviewHandler) GetReview(c *gin.Context) {
	idParam := c.Param("id")
	reviewID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid review ID",
		})
		return
	}

	// Get current user ID if authenticated
	var currentUserID *uint
	if userID, exists := middleware.GetUserIDFromContext(c); exists {
		currentUserID = &userID
	}

	review, err := h.reviewService.GetReview(uint(reviewID), currentUserID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Review retrieved successfully",
		"data":    review,
	})
}

// UpdateReview handles PUT /reviews/:id
func (h *ReviewHandler) UpdateReview(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	idParam := c.Param("id")
	reviewID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid review ID",
		})
		return
	}

	var req product.UpdateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	review, err := h.reviewService.UpdateReview(uint(reviewID), userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Review updated successfully",
		"data":    review,
	})
}

// DeleteReview handles DELETE /reviews/:id
func (h *ReviewHandler) DeleteReview(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	idParam := c.Param("id")
	reviewID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid review ID",
		})
		return
	}

	err = h.reviewService.DeleteReview(uint(reviewID), userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Review deleted successfully",
	})
}

// GetProductReviews handles GET /products/:id/reviews
func (h *ReviewHandler) GetProductReviews(c *gin.Context) {
	idParam := c.Param("id")
	productIDUint64, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid product ID",
		})
		return
	}

	productID := uint(productIDUint64)

	var req product.ReviewListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid query parameters",
			"details": err.Error(),
		})
		return
	}

	// Set product ID from URL parameter
	req.ProductID = &productID

	// Get current user ID if authenticated
	var currentUserID *uint
	if userID, exists := middleware.GetUserIDFromContext(c); exists {
		currentUserID = &userID
	}

	response, err := h.reviewService.GetReviews(&req, currentUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve product reviews",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Product reviews retrieved successfully",
		"data":    response,
	})
}

// GetProductReviewSummary handles GET /products/:id/reviews/summary
func (h *ReviewHandler) GetProductReviewSummary(c *gin.Context) {
	idParam := c.Param("id")
	productIDUint64, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid product ID",
		})
		return
	}

	productID := uint(productIDUint64)

	summary, err := h.reviewService.GetProductReviewSummary(productID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve review summary",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Review summary retrieved successfully",
		"data":    summary,
	})
}

// VoteHelpful handles POST /reviews/:id/helpful
func (h *ReviewHandler) VoteHelpful(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	idParam := c.Param("id")
	reviewID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid review ID",
		})
		return
	}

	var req product.ReviewHelpfulRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	err = h.reviewService.VoteHelpful(uint(reviewID), userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Vote recorded successfully",
	})
}

// ReportReview handles POST /reviews/:id/report
func (h *ReviewHandler) ReportReview(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	idParam := c.Param("id")
	reviewID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid review ID",
		})
		return
	}

	var req product.ReviewReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	err = h.reviewService.ReportReview(uint(reviewID), userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Review reported successfully",
	})
}

// --- ADMIN ENDPOINTS ---

// AdminGetReviews handles GET /admin/reviews
func (h *ReviewHandler) AdminGetReviews(c *gin.Context) {
	var req product.ReviewListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid query parameters",
			"details": err.Error(),
		})
		return
	}

	// Admin can see all reviews including unapproved
	response, err := h.reviewService.GetReviews(&req, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve reviews",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Reviews retrieved successfully",
		"data":    response,
	})
}

// AdminApproveReview handles PUT /admin/reviews/:id/approve
func (h *ReviewHandler) AdminApproveReview(c *gin.Context) {
	idParam := c.Param("id")
	reviewID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid review ID",
		})
		return
	}

	var req product.AdminReviewActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	err = h.reviewService.AdminApproveReview(uint(reviewID), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	message := "Review approved successfully"
	if req.Action == "reject" {
		message = "Review rejected successfully"
	}

	c.JSON(http.StatusOK, gin.H{
		"message": message,
	})
}

// AdminGetReportedReviews handles GET /admin/reviews/reported
func (h *ReviewHandler) AdminGetReportedReviews(c *gin.Context) {
	var req product.ReviewListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid query parameters",
			"details": err.Error(),
		})
		return
	}

	// Filter for reported reviews only
	req.IsApproved = nil // Show all approval statuses

	// Custom query to get only reported reviews
	// You may want to add a specific method in the service for this

	response, err := h.reviewService.GetReviews(&req, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve reported reviews",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Reported reviews retrieved successfully",
		"data":    response,
	})
}
