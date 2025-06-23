// internal/domain/cart/entity.go
package cart

import (
	"time"

	"gorm.io/gorm"
)

// CartItem represents a cart item stored in database for authenticated users
type CartItem struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	UserID           *uint          `gorm:"index" json:"user_id"`
	ProductID        uint           `gorm:"not null;index" json:"product_id"`
	ProductVariantID *uint          `gorm:"index" json:"product_variant_id"`
	Quantity         int            `gorm:"not null;default:1" json:"quantity"`
	Price            int64          `gorm:"not null" json:"price"` // Price at time of adding
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName overrides the table name
func (CartItem) TableName() string {
	return "cart_items"
}

// SessionCart represents a cart session for guest users (stored in Redis)
type SessionCart struct {
	SessionID string            `json:"session_id"`
	Items     []SessionCartItem `json:"items"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	ExpiresAt time.Time         `json:"expires_at"`
}

// SessionCartItem represents a cart item for guest users
type SessionCartItem struct {
	ProductID        uint      `json:"product_id"`
	ProductVariantID *uint     `json:"product_variant_id,omitempty"`
	Quantity         int       `json:"quantity"`
	Price            int64     `json:"price"`
	AddedAt          time.Time `json:"added_at"`
}

// CartTotals represents calculated cart totals
type CartTotals struct {
	ItemCount      int   `json:"item_count"`     // Number of unique items
	TotalQuantity  int   `json:"total_quantity"` // Sum of all quantities
	SubTotal       int64 `json:"sub_total"`      // Total before tax/shipping
	TaxAmount      int64 `json:"tax_amount"`
	ShippingCost   int64 `json:"shipping_cost"`
	DiscountAmount int64 `json:"discount_amount"`
	TotalAmount    int64 `json:"total_amount"` // Final total
}
