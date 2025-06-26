// internal/domain/inventory/entity.go
package inventory

import (
	"time"

	"gorm.io/gorm"
)

// InventoryStatus represents the status of inventory
type InventoryStatus string

const (
	InventoryStatusActive       InventoryStatus = "active"
	InventoryStatusInactive     InventoryStatus = "inactive"
	InventoryStatusDiscontinued InventoryStatus = "discontinued"
)

// MovementType represents the type of inventory movement
type MovementType string

const (
	MovementTypeInbound     MovementType = "inbound"     // Purchase, return, adjustment increase
	MovementTypeOutbound    MovementType = "outbound"    // Sale, damage, adjustment decrease
	MovementTypeReservation MovementType = "reservation" // Order placement reservation
	MovementTypeRelease     MovementType = "release"     // Cancel reservation
)

// MovementReason represents the reason for inventory movement
type MovementReason string

const (
	ReasonSale              MovementReason = "sale"
	ReasonPurchase          MovementReason = "purchase"
	ReasonReturn            MovementReason = "return"
	ReasonDamage            MovementReason = "damage"
	ReasonAdjustment        MovementReason = "adjustment"
	ReasonReservation       MovementReason = "reservation"
	ReasonCancelReservation MovementReason = "cancel_reservation"
)

// Warehouse represents a storage location
type Warehouse struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	Name       string         `gorm:"not null;size:100" json:"name"`
	Code       string         `gorm:"uniqueIndex;not null;size:20" json:"code"`
	Address    string         `gorm:"type:text" json:"address"`
	City       string         `gorm:"size:50" json:"city"`
	State      string         `gorm:"size:50" json:"state"`
	Country    string         `gorm:"size:50" json:"country"`
	PostalCode string         `gorm:"size:20" json:"postal_code"`
	Phone      string         `gorm:"size:20" json:"phone"`
	Email      string         `gorm:"size:100" json:"email"`
	IsActive   bool           `gorm:"default:true" json:"is_active"`
	IsDefault  bool           `gorm:"default:false" json:"is_default"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	InventoryItems []InventoryItem `gorm:"foreignKey:WarehouseID" json:"inventory_items,omitempty"`
}

// InventoryItem represents stock levels for a product in a warehouse
type InventoryItem struct {
	ID                uint            `gorm:"primaryKey" json:"id"`
	ProductID         uint            `gorm:"not null;index" json:"product_id"`
	WarehouseID       uint            `gorm:"not null;index" json:"warehouse_id"`
	SKU               string          `gorm:"not null;size:100;index" json:"sku"`
	Quantity          int             `gorm:"default:0" json:"quantity"`
	ReservedQuantity  int             `gorm:"default:0" json:"reserved_quantity"`
	AvailableQuantity int             `gorm:"default:0;index" json:"available_quantity"`
	ReorderLevel      int             `gorm:"default:10" json:"reorder_level"`
	MaxStockLevel     int             `gorm:"default:1000" json:"max_stock_level"`
	CostPrice         int64           `gorm:"default:0" json:"cost_price"` // In cents like your order system
	Status            InventoryStatus `gorm:"default:'active'" json:"status"`
	LastRestockDate   *time.Time      `json:"last_restock_date,omitempty"`
	BatchNumber       string          `gorm:"size:50" json:"batch_number"`
	Location          string          `gorm:"size:100" json:"location"` // Aisle/Shelf location
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	DeletedAt         gorm.DeletedAt  `gorm:"index" json:"-"`

	// Relationships
	Warehouse Warehouse           `gorm:"foreignKey:WarehouseID" json:"warehouse,omitempty"`
	Movements []InventoryMovement `gorm:"foreignKey:InventoryItemID" json:"movements,omitempty"`
}

// InventoryMovement represents a record of stock movement
type InventoryMovement struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	InventoryItemID  uint           `gorm:"not null;index" json:"inventory_item_id"`
	MovementType     MovementType   `gorm:"not null" json:"movement_type"`
	Reason           MovementReason `gorm:"not null" json:"reason"`
	Quantity         int            `gorm:"not null" json:"quantity"`
	PreviousQuantity int            `gorm:"not null" json:"previous_quantity"`
	NewQuantity      int            `gorm:"not null" json:"new_quantity"`
	ReferenceType    string         `gorm:"size:50" json:"reference_type"` // "order", "purchase", etc.
	ReferenceID      uint           `json:"reference_id"`
	Notes            string         `gorm:"type:text" json:"notes"`
	CreatedBy        uint           `gorm:"index" json:"created_by"`
	CreatedAt        time.Time      `json:"created_at"`

	// Relationships
	InventoryItem InventoryItem `gorm:"foreignKey:InventoryItemID" json:"inventory_item,omitempty"`
}

// StockAlert represents low stock alerts
type StockAlert struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	InventoryItemID uint       `gorm:"not null;index" json:"inventory_item_id"`
	AlertType       string     `gorm:"not null" json:"alert_type"` // "low_stock", "out_of_stock"
	Message         string     `gorm:"type:text" json:"message"`
	IsResolved      bool       `gorm:"default:false" json:"is_resolved"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`

	// Relationships
	InventoryItem InventoryItem `gorm:"foreignKey:InventoryItemID" json:"inventory_item,omitempty"`
}

// StockReservation represents reserved stock for orders
type StockReservation struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	InventoryItemID uint      `gorm:"not null;index" json:"inventory_item_id"`
	OrderID         uint      `gorm:"not null;index" json:"order_id"`
	OrderItemID     uint      `gorm:"not null;index" json:"order_item_id"`
	Quantity        int       `gorm:"not null" json:"quantity"`
	Status          string    `gorm:"default:'active'" json:"status"` // active, fulfilled, cancelled
	ExpiresAt       time.Time `json:"expires_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Relationships
	InventoryItem InventoryItem `gorm:"foreignKey:InventoryItemID" json:"inventory_item,omitempty"`
}

// Entity methods

// BeforeCreate hook to calculate available quantity
func (ii *InventoryItem) BeforeCreate(tx *gorm.DB) error {
	ii.AvailableQuantity = ii.Quantity - ii.ReservedQuantity
	return nil
}

// BeforeUpdate hook to calculate available quantity
func (ii *InventoryItem) BeforeUpdate(tx *gorm.DB) error {
	ii.AvailableQuantity = ii.Quantity - ii.ReservedQuantity
	return nil
}

// IsLowStock checks if inventory is below reorder level
func (ii *InventoryItem) IsLowStock() bool {
	return ii.AvailableQuantity <= ii.ReorderLevel
}

// IsOutOfStock checks if inventory is out of stock
func (ii *InventoryItem) IsOutOfStock() bool {
	return ii.AvailableQuantity <= 0
}

// CanFulfillOrder checks if there's enough stock for an order
func (ii *InventoryItem) CanFulfillOrder(quantity int) bool {
	return ii.AvailableQuantity >= quantity
}
