// internal/domain/product/review_service.go
package product

import (
	"fmt"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ReviewService handles review business logic
type ReviewService struct {
	db *gorm.DB
}

// NewReviewService creates a new review service
func NewReviewService(db *gorm.DB) *ReviewService {
	return &ReviewService{
		db: db,
	}
}

// CreateReview creates a new product review
func (s *ReviewService) CreateReview(userID uint, req *CreateReviewRequest) (*ReviewResponse, error) {
	// Check if user has already reviewed this product
	var existingReview ProductReview
	result := s.db.Where("user_id = ? AND product_id = ?", userID, req.ProductID).First(&existingReview)
	if result.Error == nil {
		return nil, fmt.Errorf("you have already reviewed this product")
	}

	// Verify product exists and is active
	var product Product
	if err := s.db.Where("id = ? AND is_active = ?", req.ProductID, true).First(&product).Error; err != nil {
		return nil, fmt.Errorf("product not found or inactive")
	}

	// Verify order if provided (for verified purchase)
	isVerified := false
	if req.OrderID != nil {
		var orderExists bool
		s.db.Raw(`
			SELECT EXISTS(
				SELECT 1 FROM orders o 
				JOIN order_items oi ON o.id = oi.order_id 
				WHERE o.id = ? AND o.user_id = ? AND oi.product_id = ? 
				AND o.status IN ('delivered', 'completed')
			)
		`, req.OrderID, userID, req.ProductID).Scan(&orderExists)

		if !orderExists {
			return nil, fmt.Errorf("cannot review: order not found or product not purchased")
		}
		isVerified = true
	}

	// Create review
	review := ProductReview{
		ProductID:    req.ProductID,
		UserID:       userID,
		OrderID:      req.OrderID,
		Rating:       req.Rating,
		Title:        strings.TrimSpace(req.Title),
		Content:      strings.TrimSpace(req.Content),
		Pros:         strings.TrimSpace(req.Pros),
		Cons:         strings.TrimSpace(req.Cons),
		IsVerified:   isVerified,
		IsApproved:   false, // Requires admin approval by default
		HelpfulCount: 0,
		IsReported:   false,
	}

	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Create(&review).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create review: %w", err)
	}

	// Add review images if provided
	if len(req.Images) > 0 {
		for i, imageURL := range req.Images {
			reviewImage := ProductReviewImage{
				ReviewID:  review.ID,
				ImageURL:  imageURL,
				Caption:   "",
				SortOrder: i + 1,
			}
			if err := tx.Create(&reviewImage).Error; err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("failed to save review image: %w", err)
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit review creation: %w", err)
	}

	// Return created review
	return s.GetReview(review.ID, &userID)
}

// GetReview retrieves a single review by ID
func (s *ReviewService) GetReview(reviewID uint, currentUserID *uint) (*ReviewResponse, error) {
	var review ProductReview
	err := s.db.Preload("Images").First(&review, reviewID).Error
	if err != nil {
		return nil, fmt.Errorf("review not found: %w", err)
	}

	return s.buildReviewResponse(&review, currentUserID), nil
}

// GetReviews retrieves reviews with filtering and pagination
func (s *ReviewService) GetReviews(req *ReviewListRequest, currentUserID *uint) (*ReviewListResponse, error) {
	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}
	if req.SortBy == "" {
		req.SortBy = "created_at"
	}
	if req.SortOrder == "" {
		req.SortOrder = "desc"
	}

	// Build query
	query := s.db.Model(&ProductReview{}).Preload("Images")

	// Apply filters
	if req.ProductID != nil {
		query = query.Where("product_id = ?", *req.ProductID)
	}
	if req.UserID != nil {
		query = query.Where("user_id = ?", *req.UserID)
	}
	if req.Rating != nil {
		query = query.Where("rating = ?", *req.Rating)
	}
	if req.IsVerified != nil {
		query = query.Where("is_verified = ?", *req.IsVerified)
	}
	if req.IsApproved != nil {
		query = query.Where("is_approved = ?", *req.IsApproved)
	} else {
		// By default, only show approved reviews to regular users
		if currentUserID == nil || !s.isUserAdmin(*currentUserID) {
			query = query.Where("is_approved = ?", true)
		}
	}

	// Search functionality
	if req.SearchQuery != "" {
		searchTerm := "%" + strings.ToLower(req.SearchQuery) + "%"
		query = query.Where("LOWER(title) LIKE ? OR LOWER(content) LIKE ?", searchTerm, searchTerm)
	}

	// Count total
	var total int64
	query.Count(&total)

	// Apply sorting
	orderClause := s.buildOrderClause(req.SortBy, req.SortOrder)
	query = query.Order(orderClause)

	// Apply pagination
	offset := (req.Page - 1) * req.Limit
	query = query.Offset(offset).Limit(req.Limit)

	// Execute query
	var reviews []ProductReview
	if err := query.Find(&reviews).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve reviews: %w", err)
	}

	// Build responses
	reviewResponses := make([]ReviewResponse, len(reviews))
	for i, review := range reviews {
		reviewResponses[i] = *s.buildReviewResponse(&review, currentUserID)
	}

	// Calculate pagination
	totalPages := int(math.Ceil(float64(total) / float64(req.Limit)))

	// Get summary (for product reviews)
	var summary ReviewSummary
	if req.ProductID != nil {
		summary = s.getReviewSummary(*req.ProductID)
	}

	return &ReviewListResponse{
		Reviews: reviewResponses,
		Pagination: PaginationInfo{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      total,
			TotalPages: totalPages,
			HasNext:    req.Page < totalPages,
			HasPrev:    req.Page > 1,
		},
		Summary: summary,
	}, nil
}

// UpdateReview updates a review (user can only edit their own)
func (s *ReviewService) UpdateReview(reviewID, userID uint, req *UpdateReviewRequest) (*ReviewResponse, error) {
	var review ProductReview
	err := s.db.First(&review, reviewID).Error
	if err != nil {
		return nil, fmt.Errorf("review not found: %w", err)
	}

	// Check permissions
	if !review.CanBeEditedBy(userID) {
		return nil, fmt.Errorf("you cannot edit this review")
	}

	// Build updates
	updates := make(map[string]interface{})
	if req.Rating != nil {
		updates["rating"] = *req.Rating
	}
	if req.Title != nil {
		updates["title"] = strings.TrimSpace(*req.Title)
	}
	if req.Content != nil {
		updates["content"] = strings.TrimSpace(*req.Content)
	}
	if req.Pros != nil {
		updates["pros"] = strings.TrimSpace(*req.Pros)
	}
	if req.Cons != nil {
		updates["cons"] = strings.TrimSpace(*req.Cons)
	}

	// If content is updated, reset approval status
	if len(updates) > 0 {
		updates["is_approved"] = false
		updates["updated_at"] = time.Now()
	}

	if err := s.db.Model(&review).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update review: %w", err)
	}

	return s.GetReview(reviewID, &userID)
}

// DeleteReview deletes a review (user can only delete their own)
func (s *ReviewService) DeleteReview(reviewID, userID uint) error {
	var review ProductReview
	err := s.db.First(&review, reviewID).Error
	if err != nil {
		return fmt.Errorf("review not found: %w", err)
	}

	// Check permissions
	if !review.CanBeDeletedBy(userID) && !s.isUserAdmin(userID) {
		return fmt.Errorf("you cannot delete this review")
	}

	// Soft delete
	if err := s.db.Delete(&review).Error; err != nil {
		return fmt.Errorf("failed to delete review: %w", err)
	}

	return nil
}

// VoteHelpful allows users to vote if a review is helpful
func (s *ReviewService) VoteHelpful(reviewID, userID uint, req *ReviewHelpfulRequest) error {
	// Check if review exists
	var review ProductReview
	if err := s.db.First(&review, reviewID).Error; err != nil {
		return fmt.Errorf("review not found: %w", err)
	}

	// Check if user already voted
	var existingVote ProductReviewHelpful
	result := s.db.Where("review_id = ? AND user_id = ?", reviewID, userID).First(&existingVote)

	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if result.Error == gorm.ErrRecordNotFound {
		// Create new vote
		vote := ProductReviewHelpful{
			ReviewID:  reviewID,
			UserID:    userID,
			IsHelpful: req.IsHelpful,
		}
		if err := tx.Create(&vote).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to create vote: %w", err)
		}
	} else {
		// Update existing vote
		if err := tx.Model(&existingVote).Update("is_helpful", req.IsHelpful).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to update vote: %w", err)
		}
	}

	// Update helpful count in review
	var helpfulCount int64
	tx.Model(&ProductReviewHelpful{}).Where("review_id = ? AND is_helpful = ?", reviewID, true).Count(&helpfulCount)

	if err := tx.Model(&ProductReview{}).Where("id = ?", reviewID).Update("helpful_count", helpfulCount).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update helpful count: %w", err)
	}

	return tx.Commit().Error
}

// ReportReview allows users to report inappropriate reviews
func (s *ReviewService) ReportReview(reviewID, userID uint, req *ReviewReportRequest) error {
	// Check if review exists
	var review ProductReview
	if err := s.db.First(&review, reviewID).Error; err != nil {
		return fmt.Errorf("review not found: %w", err)
	}

	// Check if user already reported this review
	var existingReport ProductReviewReport
	result := s.db.Where("review_id = ? AND user_id = ?", reviewID, userID).First(&existingReport)
	if result.Error == nil {
		return fmt.Errorf("you have already reported this review")
	}

	// Create report
	report := ProductReviewReport{
		ReviewID: reviewID,
		UserID:   userID,
		Reason:   req.Reason,
		Comment:  req.Comment,
		Status:   "pending",
	}

	if err := s.db.Create(&report).Error; err != nil {
		return fmt.Errorf("failed to create report: %w", err)
	}

	// Mark review as reported
	s.db.Model(&review).Update("is_reported", true)

	return nil
}

// GetProductReviewSummary gets review statistics for a product
func (s *ReviewService) GetProductReviewSummary(productID uint) (*ReviewSummary, error) {
	summary := s.getReviewSummary(productID)
	return &summary, nil
}

// Admin methods

// AdminApproveReview approves/rejects a review
func (s *ReviewService) AdminApproveReview(reviewID uint, action *AdminReviewActionRequest) error {
	var review ProductReview
	if err := s.db.First(&review, reviewID).Error; err != nil {
		return fmt.Errorf("review not found: %w", err)
	}

	updates := map[string]interface{}{
		"is_approved": action.Action == "approve",
	}

	if err := s.db.Model(&review).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update review status: %w", err)
	}

	return nil
}

// Private helper methods

func (s *ReviewService) buildReviewResponse(review *ProductReview, currentUserID *uint) *ReviewResponse {
	response := &ReviewResponse{
		ID:           review.ID,
		ProductID:    review.ProductID,
		UserID:       review.UserID,
		OrderID:      review.OrderID,
		Rating:       review.Rating,
		Title:        review.Title,
		Content:      review.Content,
		Pros:         review.Pros,
		Cons:         review.Cons,
		IsVerified:   review.IsVerified,
		IsApproved:   review.IsApproved,
		HelpfulCount: review.HelpfulCount,
		IsReported:   review.IsReported,
		Images:       review.Images,
		CreatedAt:    review.CreatedAt,
		UpdatedAt:    review.UpdatedAt,
	}

	// Load user info
	response.User = s.getReviewUser(review.UserID)

	// Load product info
	response.Product = s.getReviewProduct(review.ProductID)

	// Set permissions for current user
	if currentUserID != nil {
		response.CanEdit = review.CanBeEditedBy(*currentUserID)
		response.CanDelete = review.CanBeDeletedBy(*currentUserID) || s.isUserAdmin(*currentUserID)

		// Check if user voted on this review
		var vote ProductReviewHelpful
		result := s.db.Where("review_id = ? AND user_id = ?", review.ID, *currentUserID).First(&vote)
		if result.Error == nil {
			response.UserVoted = &vote.IsHelpful
			response.UserHelpful = &vote.IsHelpful
		}
	}

	return response
}

func (s *ReviewService) getReviewUser(userID uint) *ReviewUserResponse {
	var user struct {
		ID        uint   `json:"id"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Avatar    string `json:"avatar"`
	}

	s.db.Table("users").Select("id, first_name, last_name, avatar").Where("id = ?", userID).Scan(&user)

	// Get review count for this user
	var reviewCount int64
	s.db.Model(&ProductReview{}).Where("user_id = ? AND is_approved = ?", userID, true).Count(&reviewCount)

	return &ReviewUserResponse{
		ID:          user.ID,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Avatar:      user.Avatar,
		IsVerified:  true, // You can implement user verification logic
		ReviewCount: int(reviewCount),
	}
}

func (s *ReviewService) getReviewProduct(productID uint) *ReviewProductResponse {
	var product struct {
		ID       uint   `json:"id"`
		Name     string `json:"name"`
		Slug     string `json:"slug"`
		ImageURL string `json:"image_url"`
	}

	s.db.Table("products").Select("id, name, slug").Where("id = ?", productID).Scan(&product)

	// Get first product image
	s.db.Table("product_images").Select("image_url").Where("product_id = ?", productID).Order("sort_order ASC").Limit(1).Scan(&product.ImageURL)

	return &ReviewProductResponse{
		ID:       product.ID,
		Name:     product.Name,
		Slug:     product.Slug,
		ImageURL: product.ImageURL,
	}
}

func (s *ReviewService) getReviewSummary(productID uint) ReviewSummary {
	var summary ReviewSummary

	// Get basic stats
	var totalReviews int64
	var verifiedCount int64
	s.db.Model(&ProductReview{}).Where("product_id = ? AND is_approved = ?", productID, true).Count(&totalReviews)
	s.db.Model(&ProductReview{}).Where("product_id = ? AND is_approved = ? AND is_verified = ?", productID, true, true).Count(&verifiedCount)

	summary.TotalReviews = int(totalReviews)
	summary.VerifiedCount = int(verifiedCount)

	// Get average rating
	var avgRating float64
	s.db.Model(&ProductReview{}).Where("product_id = ? AND is_approved = ?", productID, true).Select("AVG(rating)").Scan(&avgRating)
	summary.AverageRating = math.Round(avgRating*100) / 100

	// Get rating breakdown
	summary.RatingBreakdown = make(map[string]int)
	for i := 1; i <= 5; i++ {
		var count int64
		s.db.Model(&ProductReview{}).Where("product_id = ? AND is_approved = ? AND rating = ?", productID, true, i).Count(&count)
		summary.RatingBreakdown[fmt.Sprintf("%d", i)] = int(count)
	}

	// Get recent reviews count (last 30 days)
	var recentReviews int64
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	s.db.Model(&ProductReview{}).Where("product_id = ? AND is_approved = ? AND created_at >= ?", productID, true, thirtyDaysAgo).Count(&recentReviews)
	summary.RecentReviews = int(recentReviews)

	return summary
}

func (s *ReviewService) buildOrderClause(sortBy, sortOrder string) string {
	validSortFields := map[string]bool{
		"created_at":    true,
		"updated_at":    true,
		"rating":        true,
		"helpful_count": true,
	}

	if !validSortFields[sortBy] {
		sortBy = "created_at"
	}

	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	return fmt.Sprintf("%s %s", sortBy, sortOrder)
}

func (s *ReviewService) isUserAdmin(userID uint) bool {
	// Check if user has admin role
	var count int64
	s.db.Table("users").Where("id = ? AND role = ?", userID, "admin").Count(&count)
	return count > 0
}
