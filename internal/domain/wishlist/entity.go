package wishlist

import (
	"time"

	"gorm.io/gorm"
)

// WishlistItem represents a wishlist item
type WishlistItem struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	UserID           uint           `gorm:"not null;index" json:"user_id"`
	ProductID        uint           `gorm:"not null;index" json:"product_id"`
	ProductVariantID *uint          `gorm:"index" json:"product_variant_id"`
	AddedAt          time.Time      `json:"added_at"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName overrides the table name
func (WishlistItem) TableName() string {
	return "wishlist_items"
}
