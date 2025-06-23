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

// ProductReview represents customer reviews
type ProductReview struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	ProductID  uint           `gorm:"not null;index" json:"product_id"`
	UserID     uint           `gorm:"not null;index" json:"user_id"`
	Rating     int            `gorm:"not null;check:rating >= 1 AND rating <= 5" json:"rating"`
	Title      string         `gorm:"size:255" json:"title"`
	Content    string         `gorm:"type:text" json:"content"`
	IsVerified bool           `gorm:"default:false" json:"is_verified"`
	IsApproved bool           `gorm:"default:false" json:"is_approved"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName overrides
func (Product) TableName() string        { return "products" }
func (Category) TableName() string       { return "categories" }
func (Brand) TableName() string          { return "brands" }
func (ProductImage) TableName() string   { return "product_images" }
func (ProductVariant) TableName() string { return "product_variants" }
func (ProductReview) TableName() string  { return "product_reviews" }

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
