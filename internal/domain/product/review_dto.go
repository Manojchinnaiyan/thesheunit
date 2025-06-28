// internal/domain/product/review_dto.go
package product

import "time"

// Review DTOs and Request/Response structures

// CreateReviewRequest represents the request to create a review
type CreateReviewRequest struct {
	ProductID uint     `json:"product_id" binding:"required"`
	OrderID   *uint    `json:"order_id,omitempty"`
	Rating    int      `json:"rating" binding:"required,min=1,max=5"`
	Title     string   `json:"title" binding:"required,max=255"`
	Content   string   `json:"content" binding:"required,max=2000"`
	Pros      string   `json:"pros,omitempty" binding:"max=1000"`
	Cons      string   `json:"cons,omitempty" binding:"max=1000"`
	Images    []string `json:"images,omitempty" binding:"max=5"` // Image URLs
}

// UpdateReviewRequest represents the request to update a review
type UpdateReviewRequest struct {
	Rating  *int    `json:"rating,omitempty" binding:"omitempty,min=1,max=5"`
	Title   *string `json:"title,omitempty" binding:"omitempty,max=255"`
	Content *string `json:"content,omitempty" binding:"omitempty,max=2000"`
	Pros    *string `json:"pros,omitempty" binding:"omitempty,max=1000"`
	Cons    *string `json:"cons,omitempty" binding:"omitempty,max=1000"`
}

// ReviewListRequest represents query parameters for listing reviews
type ReviewListRequest struct {
	ProductID   *uint  `form:"product_id"`
	UserID      *uint  `form:"user_id"`
	Rating      *int   `form:"rating" binding:"omitempty,min=1,max=5"`
	IsVerified  *bool  `form:"is_verified"`
	IsApproved  *bool  `form:"is_approved"`
	SortBy      string `form:"sort_by"`    // created_at, rating, helpful_count
	SortOrder   string `form:"sort_order"` // asc, desc
	Page        int    `form:"page"`
	Limit       int    `form:"limit"`
	SearchQuery string `form:"search"` // Search in title and content
}

// ReviewResponse represents a single review response
type ReviewResponse struct {
	ID           uint                   `json:"id"`
	ProductID    uint                   `json:"product_id"`
	UserID       uint                   `json:"user_id"`
	OrderID      *uint                  `json:"order_id,omitempty"`
	Rating       int                    `json:"rating"`
	Title        string                 `json:"title"`
	Content      string                 `json:"content"`
	Pros         string                 `json:"pros,omitempty"`
	Cons         string                 `json:"cons,omitempty"`
	IsVerified   bool                   `json:"is_verified"`
	IsApproved   bool                   `json:"is_approved"`
	HelpfulCount int                    `json:"helpful_count"`
	IsReported   bool                   `json:"is_reported"`
	Images       []ProductReviewImage   `json:"images,omitempty"`
	User         *ReviewUserResponse    `json:"user,omitempty"`
	Product      *ReviewProductResponse `json:"product,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`

	// Additional fields for user context
	UserVoted   *bool `json:"user_voted,omitempty"`   // Did current user vote on this review
	UserHelpful *bool `json:"user_helpful,omitempty"` // Did current user find it helpful
	CanEdit     bool  `json:"can_edit"`               // Can current user edit this review
	CanDelete   bool  `json:"can_delete"`             // Can current user delete this review
}

// ReviewListResponse represents paginated review list
type ReviewListResponse struct {
	Reviews    []ReviewResponse `json:"reviews"`
	Pagination PaginationInfo   `json:"pagination"`
	Summary    ReviewSummary    `json:"summary"`
}

// ReviewSummary provides review statistics
type ReviewSummary struct {
	TotalReviews    int            `json:"total_reviews"`
	AverageRating   float64        `json:"average_rating"`
	RatingBreakdown map[string]int `json:"rating_breakdown"` // "5": 10, "4": 5, etc.
	VerifiedCount   int            `json:"verified_count"`
	RecentReviews   int            `json:"recent_reviews"` // Reviews in last 30 days
}

// PaginationInfo represents pagination info
type PaginationInfo struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// ReviewUserResponse represents user info in review responses
type ReviewUserResponse struct {
	ID          uint   `json:"id"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Avatar      string `json:"avatar,omitempty"`
	IsVerified  bool   `json:"is_verified"`
	ReviewCount int    `json:"review_count"`
}

// ReviewProductResponse represents basic product info in review responses
type ReviewProductResponse struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	ImageURL string `json:"image_url,omitempty"`
}

// ReviewHelpfulRequest represents helpful vote request
type ReviewHelpfulRequest struct {
	IsHelpful bool `json:"is_helpful" binding:"required"`
}

// ReviewReportRequest represents review report request
type ReviewReportRequest struct {
	Reason  string `json:"reason" binding:"required,oneof=spam inappropriate fake other"`
	Comment string `json:"comment,omitempty" binding:"max=500"`
}

// AdminReviewActionRequest represents admin actions on reviews
type AdminReviewActionRequest struct {
	Action  string `json:"action" binding:"required,oneof=approve reject"`
	Comment string `json:"comment,omitempty" binding:"max=500"`
}
