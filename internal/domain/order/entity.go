// internal/domain/order/entity.go
package order

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// OrderStatus represents the order status
type OrderStatus string

const (
	OrderStatusPending           OrderStatus = "pending"
	OrderStatusPaymentProcessing OrderStatus = "payment_processing"
	OrderStatusConfirmed         OrderStatus = "confirmed"
	OrderStatusProcessing        OrderStatus = "processing"
	OrderStatusShipped           OrderStatus = "shipped"
	OrderStatusOutForDelivery    OrderStatus = "out_for_delivery"
	OrderStatusDelivered         OrderStatus = "delivered"
	OrderStatusCompleted         OrderStatus = "completed"
	OrderStatusCancelled         OrderStatus = "cancelled"
	OrderStatusRefunded          OrderStatus = "refunded"
)

// PaymentStatus represents payment status
type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "pending"
	PaymentStatusProcessing PaymentStatus = "processing"
	PaymentStatusPaid       PaymentStatus = "paid"
	PaymentStatusFailed     PaymentStatus = "failed"
	PaymentStatusCancelled  PaymentStatus = "cancelled"
	PaymentStatusRefunded   PaymentStatus = "refunded"
)

// Order represents the order entity
type Order struct {
	ID            uint          `gorm:"primaryKey" json:"id"`
	OrderNumber   string        `gorm:"uniqueIndex;not null;size:50" json:"order_number"`
	UserID        *uint         `gorm:"index" json:"user_id"` // Nullable for guest orders
	Email         string        `gorm:"not null;size:255" json:"email"`
	Status        OrderStatus   `gorm:"not null;default:'pending'" json:"status"`
	PaymentStatus PaymentStatus `gorm:"not null;default:'pending'" json:"payment_status"`

	// Financial Information
	SubtotalAmount int64 `gorm:"not null" json:"subtotal_amount"` // In cents
	TaxAmount      int64 `gorm:"default:0" json:"tax_amount"`
	ShippingAmount int64 `gorm:"default:0" json:"shipping_amount"`
	DiscountAmount int64 `gorm:"default:0" json:"discount_amount"`
	TotalAmount    int64 `gorm:"not null" json:"total_amount"`

	// Addresses
	ShippingAddress Address `gorm:"embedded;embeddedPrefix:shipping_" json:"shipping_address"`
	BillingAddress  Address `gorm:"embedded;embeddedPrefix:billing_" json:"billing_address"`

	// Additional Information
	Currency      string `gorm:"size:3;default:'USD'" json:"currency"`
	Notes         string `gorm:"type:text" json:"notes"`
	InternalNotes string `gorm:"type:text" json:"internal_notes"`

	// Coupon/Discount
	CouponCode string `gorm:"size:50" json:"coupon_code"`

	// Shipping Information
	ShippingMethod  string `gorm:"size:100" json:"shipping_method"`
	TrackingNumber  string `gorm:"size:100" json:"tracking_number"`
	ShippingCarrier string `gorm:"size:50" json:"shipping_carrier"`

	// Timestamps
	ProcessedAt *time.Time     `json:"processed_at"`
	ShippedAt   *time.Time     `json:"shipped_at"`
	DeliveredAt *time.Time     `json:"delivered_at"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Items         []OrderItem          `gorm:"foreignKey:OrderID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"items"`
	Payments      []Payment            `gorm:"foreignKey:OrderID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"payments,omitempty"`
	StatusHistory []OrderStatusHistory `gorm:"foreignKey:OrderID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"status_history,omitempty"`
}

// OrderItem represents items in an order
type OrderItem struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	OrderID          uint      `gorm:"not null;index" json:"order_id"`
	ProductID        uint      `gorm:"not null;index" json:"product_id"`
	ProductVariantID *uint     `gorm:"index" json:"product_variant_id"`
	SKU              string    `gorm:"not null;size:100" json:"sku"`
	Name             string    `gorm:"not null;size:255" json:"name"`
	VariantTitle     string    `gorm:"size:255" json:"variant_title"`
	Quantity         int       `gorm:"not null" json:"quantity"`
	Price            int64     `gorm:"not null" json:"price"`       // Price per unit in cents
	TotalPrice       int64     `gorm:"not null" json:"total_price"` // Quantity * Price
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Payment represents payment transactions
type Payment struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	OrderID           uint           `gorm:"not null;index" json:"order_id"`
	PaymentMethod     string         `gorm:"not null;size:50" json:"payment_method"` // stripe, paypal, etc.
	PaymentProviderID string         `gorm:"size:255" json:"payment_provider_id"`    // External payment ID
	Amount            int64          `gorm:"not null" json:"amount"`                 // In cents
	Currency          string         `gorm:"size:3;default:'USD'" json:"currency"`
	Status            PaymentStatus  `gorm:"not null" json:"status"`
	Gateway           string         `gorm:"size:50" json:"gateway"`
	GatewayResponse   string         `gorm:"type:text" json:"gateway_response"` // JSON response from gateway
	ProcessedAt       *time.Time     `json:"processed_at"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

// OrderStatusHistory tracks order status changes
type OrderStatusHistory struct {
	ID        uint        `gorm:"primaryKey" json:"id"`
	OrderID   uint        `gorm:"not null;index" json:"order_id"`
	Status    OrderStatus `gorm:"not null" json:"status"`
	Comment   string      `gorm:"type:text" json:"comment"`
	CreatedBy uint        `gorm:"index" json:"created_by"` // User ID who made the change
	CreatedAt time.Time   `json:"created_at"`
}

// Address represents shipping/billing address (embedded in Order)
type Address struct {
	FirstName    string `gorm:"size:100" json:"first_name"`
	LastName     string `gorm:"size:100" json:"last_name"`
	Company      string `gorm:"size:100" json:"company"`
	AddressLine1 string `gorm:"size:255" json:"address_line1"`
	AddressLine2 string `gorm:"size:255" json:"address_line2"`
	City         string `gorm:"size:100" json:"city"`
	State        string `gorm:"size:100" json:"state"`
	PostalCode   string `gorm:"size:20" json:"postal_code"`
	Country      string `gorm:"size:2" json:"country"`
	Phone        string `gorm:"size:20" json:"phone"`
}

// TableName overrides
func (Order) TableName() string              { return "orders" }
func (OrderItem) TableName() string          { return "order_items" }
func (Payment) TableName() string            { return "payments" }
func (OrderStatusHistory) TableName() string { return "order_status_history" }

// Business methods for Order

// GenerateOrderNumber generates a unique order number
func (o *Order) GenerateOrderNumber() string {
	// Format: ORD-YYYYMMDD-XXXXX
	return fmt.Sprintf("ORD-%s-%05d", time.Now().Format("20060102"), o.ID)
}

// GetFormattedTotal returns total amount as float
func (o *Order) GetFormattedTotal() float64 {
	return float64(o.TotalAmount) / 100
}

// CanBeCancelled checks if order can be cancelled
func (o *Order) CanBeCancelled() bool {
	return o.Status == OrderStatusPending ||
		o.Status == OrderStatusPaymentProcessing ||
		o.Status == OrderStatusConfirmed
}

// CanBeRefunded checks if order can be refunded
func (o *Order) CanBeRefunded() bool {
	return o.PaymentStatus == PaymentStatusPaid &&
		(o.Status == OrderStatusDelivered || o.Status == OrderStatusCompleted)
}

// IsCompleted checks if order is completed
func (o *Order) IsCompleted() bool {
	return o.Status == OrderStatusCompleted || o.Status == OrderStatusDelivered
}

// AddStatusHistory adds a new status change to history
func (o *Order) AddStatusHistory(status OrderStatus, comment string, createdBy uint) {
	history := OrderStatusHistory{
		OrderID:   o.ID,
		Status:    status,
		Comment:   comment,
		CreatedBy: createdBy,
		CreatedAt: time.Now().UTC(),
	}
	o.StatusHistory = append(o.StatusHistory, history)
}
