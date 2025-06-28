// internal/domain/product/entity.go
package product

import (
	"time"

	"gorm.io/gorm"
)

// Product represents the product entity
type Product struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	SKU               string         `gorm:"uniqueIndex;not null;size:100" json:"sku"`
	Name              string         `gorm:"not null;size:255" json:"name"`
	Slug              string         `gorm:"uniqueIndex;not null;size:255" json:"slug"`
	Description       string         `gorm:"type:text" json:"description"`
	ShortDesc         string         `gorm:"size:500" json:"short_description"`
	Price             int64          `gorm:"not null" json:"price"` // Price in cents
	ComparePrice      int64          `json:"compare_price"`         // Original price for discounts
	CostPrice         int64          `json:"cost_price"`            // Cost price for profit calculation
	CategoryID        uint           `gorm:"not null;index" json:"category_id"`
	BrandID           *uint          `gorm:"index" json:"brand_id"`
	Weight            float64        `json:"weight"`                     // Weight in grams
	Dimensions        string         `gorm:"size:100" json:"dimensions"` // LxWxH format
	IsActive          bool           `gorm:"default:true" json:"is_active"`
	IsFeatured        bool           `gorm:"default:false" json:"is_featured"`
	IsDigital         bool           `gorm:"default:false" json:"is_digital"`
	RequiresShipping  bool           `gorm:"default:true" json:"requires_shipping"`
	TrackQuantity     bool           `gorm:"default:true" json:"track_quantity"`
	Quantity          int            `gorm:"default:0" json:"quantity"`
	LowStockThreshold int            `gorm:"default:5" json:"low_stock_threshold"`
	SeoTitle          string         `gorm:"size:255" json:"seo_title"`
	SeoDescription    string         `gorm:"size:500" json:"seo_description"`
	Tags              string         `gorm:"size:500" json:"tags"` // Comma-separated tags
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Category Category         `gorm:"foreignKey:CategoryID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"category"`
	Brand    *Brand           `gorm:"foreignKey:BrandID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"brand,omitempty"`
	Images   []ProductImage   `gorm:"foreignKey:ProductID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"images,omitempty"`
	Variants []ProductVariant `gorm:"foreignKey:ProductID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"variants,omitempty"`
	Reviews  []ProductReview  `gorm:"foreignKey:ProductID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"reviews,omitempty"`
}

// Category represents product categories
type Category struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"not null;size:255" json:"name"`
	Slug        string         `gorm:"uniqueIndex;not null;size:255" json:"slug"`
	Description string         `gorm:"size:500" json:"description"`
	Image       string         `gorm:"size:500" json:"image"`
	ParentID    *uint          `gorm:"index" json:"parent_id"`
	SortOrder   int            `gorm:"default:0" json:"sort_order"`
	IsActive    bool           `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Parent   *Category  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []Category `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Products []Product  `gorm:"foreignKey:CategoryID" json:"products,omitempty"`
}

// Brand represents product brands
type Brand struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"not null;size:255" json:"name"`
	Slug        string         `gorm:"uniqueIndex;not null;size:255" json:"slug"`
	Description string         `gorm:"size:500" json:"description"`
	Logo        string         `gorm:"size:500" json:"logo"`
	Website     string         `gorm:"size:255" json:"website"`
	IsActive    bool           `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Products []Product `gorm:"foreignKey:BrandID" json:"products,omitempty"`
}

// ProductImage represents product images
type ProductImage struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ProductID uint      `gorm:"not null;index" json:"product_id"`
	URL       string    `gorm:"not null;size:500" json:"url"`
	AltText   string    `gorm:"size:255" json:"alt_text"`
	SortOrder int       `gorm:"default:0" json:"sort_order"`
	IsPrimary bool      `gorm:"default:false" json:"is_primary"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProductVariant represents product variants (size, color, etc.)
type ProductVariant struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	ProductID    uint           `gorm:"not null;index" json:"product_id"`
	SKU          string         `gorm:"uniqueIndex;not null;size:100" json:"sku"`
	Name         string         `gorm:"not null;size:255" json:"name"`
	Price        int64          `json:"price"` // Override product price if set
	ComparePrice int64          `json:"compare_price"`
	CostPrice    int64          `json:"cost_price"`
	Quantity     int            `gorm:"default:0" json:"quantity"`
	Weight       float64        `json:"weight"`
	Options      string         `gorm:"type:text" json:"options"` // JSON string for variant options
	IsActive     bool           `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName overrides
func (Product) TableName() string        { return "products" }
func (Category) TableName() string       { return "categories" }
func (Brand) TableName() string          { return "brands" }
func (ProductImage) TableName() string   { return "product_images" }
func (ProductVariant) TableName() string { return "product_variants" }

// Business methods for Product
func (p *Product) IsInStock() bool {
	return p.Quantity > 0 || !p.TrackQuantity
}

func (p *Product) IsLowStock() bool {
	return p.TrackQuantity && p.Quantity <= p.LowStockThreshold
}

func (p *Product) GetFormattedPrice() float64 {
	return float64(p.Price) / 100
}

func (p *Product) GetDiscountPercentage() int {
	if p.ComparePrice > 0 && p.Price < p.ComparePrice {
		return int(((p.ComparePrice - p.Price) * 100) / p.ComparePrice)
	}
	return 0
}

// Add these to your existing internal/domain/product/entity.go file

// ProductReview represents customer reviews with enhanced features
type ProductReview struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	ProductID    uint           `gorm:"not null;index" json:"product_id"`
	UserID       uint           `gorm:"not null;index" json:"user_id"`
	OrderID      *uint          `gorm:"index" json:"order_id,omitempty"` // Link to verified purchase
	Rating       int            `gorm:"not null;check:rating >= 1 AND rating <= 5" json:"rating"`
	Title        string         `gorm:"size:255" json:"title"`
	Content      string         `gorm:"type:text" json:"content"`
	Pros         string         `gorm:"type:text" json:"pros,omitempty"`  // What users liked
	Cons         string         `gorm:"type:text" json:"cons,omitempty"`  // What users didn't like
	IsVerified   bool           `gorm:"default:false" json:"is_verified"` // Verified purchase
	IsApproved   bool           `gorm:"default:false" json:"is_approved"` // Admin approved
	HelpfulCount int            `gorm:"default:0" json:"helpful_count"`   // Helpful votes
	IsReported   bool           `gorm:"default:false" json:"is_reported"` // Flagged by users
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Images []ProductReviewImage `gorm:"foreignKey:ReviewID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"images,omitempty"`
}

// ProductReviewImage represents images attached to reviews
type ProductReviewImage struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ReviewID  uint      `gorm:"not null;index" json:"review_id"`
	ImageURL  string    `gorm:"not null;size:500" json:"image_url"`
	Caption   string    `gorm:"size:255" json:"caption,omitempty"`
	SortOrder int       `gorm:"default:0" json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
}

// ProductReviewHelpful tracks helpful votes
type ProductReviewHelpful struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ReviewID  uint      `gorm:"not null;index" json:"review_id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	IsHelpful bool      `gorm:"not null" json:"is_helpful"` // true for helpful, false for not helpful
	CreatedAt time.Time `json:"created_at"`
}

// ProductReviewReport represents user reports on reviews
type ProductReviewReport struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ReviewID  uint      `gorm:"not null;index" json:"review_id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	Reason    string    `gorm:"not null;size:100" json:"reason"` // spam, inappropriate, fake, other
	Comment   string    `gorm:"type:text" json:"comment,omitempty"`
	Status    string    `gorm:"default:'pending'" json:"status"` // pending, reviewed, resolved
	CreatedAt time.Time `json:"created_at"`
}

// Table names
func (ProductReviewImage) TableName() string   { return "product_review_images" }
func (ProductReviewHelpful) TableName() string { return "product_review_helpful" }
func (ProductReviewReport) TableName() string  { return "product_review_reports" }

// Business methods for ProductReview
func (r *ProductReview) CanBeEditedBy(userID uint) bool {
	// Users can edit their own reviews within 30 days
	if r.UserID != userID {
		return false
	}

	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	return r.CreatedAt.After(thirtyDaysAgo)
}

func (r *ProductReview) CanBeDeletedBy(userID uint) bool {
	// Users can delete their own reviews
	return r.UserID == userID
}

func (r *ProductReview) IsFromVerifiedPurchase() bool {
	return r.OrderID != nil && r.IsVerified
}
