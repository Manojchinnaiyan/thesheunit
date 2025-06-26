// internal/domain/inventory/service.go
package inventory

import (
	"fmt"
	"time"

	"github.com/your-org/ecommerce-backend/internal/config"
	"gorm.io/gorm"
)

// Service handles inventory business logic
type Service struct {
	db     *gorm.DB
	config *config.Config
}

// NewService creates a new inventory service
func NewService(db *gorm.DB, cfg *config.Config) *Service {
	return &Service{
		db:     db,
		config: cfg,
	}
}

// CreateWarehouseRequest represents warehouse creation data
type CreateWarehouseRequest struct {
	Name       string `json:"name" binding:"required"`
	Code       string `json:"code" binding:"required"`
	Address    string `json:"address"`
	City       string `json:"city"`
	State      string `json:"state"`
	Country    string `json:"country"`
	PostalCode string `json:"postal_code"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
	IsDefault  bool   `json:"is_default"`
}

// StockMovementRequest represents stock movement data
type StockMovementRequest struct {
	ProductID     uint           `json:"product_id" binding:"required"`
	WarehouseID   uint           `json:"warehouse_id" binding:"required"`
	MovementType  MovementType   `json:"movement_type" binding:"required"`
	Reason        MovementReason `json:"reason" binding:"required"`
	Quantity      int            `json:"quantity" binding:"required"`
	ReferenceType string         `json:"reference_type,omitempty"`
	ReferenceID   uint           `json:"reference_id,omitempty"`
	Notes         string         `json:"notes,omitempty"`
	CostPrice     int64          `json:"cost_price,omitempty"` // In cents
}

// ReservationRequest represents stock reservation data
type ReservationRequest struct {
	ProductID   uint `json:"product_id" binding:"required"`
	WarehouseID uint `json:"warehouse_id" binding:"required"`
	OrderID     uint `json:"order_id" binding:"required"`
	OrderItemID uint `json:"order_item_id" binding:"required"`
	Quantity    int  `json:"quantity" binding:"required"`
}

// WAREHOUSE MANAGEMENT

// CreateWarehouse creates a new warehouse
func (s *Service) CreateWarehouse(req *CreateWarehouseRequest) (*Warehouse, error) {
	// Check if code already exists
	var existing Warehouse
	if err := s.db.Where("code = ?", req.Code).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("warehouse with code '%s' already exists", req.Code)
	}

	// If this is set as default, unset others
	if req.IsDefault {
		s.db.Model(&Warehouse{}).Where("is_default = ?", true).Update("is_default", false)
	}

	warehouse := &Warehouse{
		Name:       req.Name,
		Code:       req.Code,
		Address:    req.Address,
		City:       req.City,
		State:      req.State,
		Country:    req.Country,
		PostalCode: req.PostalCode,
		Phone:      req.Phone,
		Email:      req.Email,
		IsDefault:  req.IsDefault,
		IsActive:   true,
	}

	if err := s.db.Create(warehouse).Error; err != nil {
		return nil, fmt.Errorf("failed to create warehouse: %w", err)
	}

	return warehouse, nil
}

// GetWarehouses retrieves all active warehouses
func (s *Service) GetWarehouses() ([]Warehouse, error) {
	var warehouses []Warehouse
	if err := s.db.Where("is_active = ?", true).Find(&warehouses).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve warehouses: %w", err)
	}
	return warehouses, nil
}

// GetDefaultWarehouse gets the default warehouse
func (s *Service) GetDefaultWarehouse() (*Warehouse, error) {
	var warehouse Warehouse
	if err := s.db.Where("is_default = ? AND is_active = ?", true, true).First(&warehouse).Error; err != nil {
		return nil, fmt.Errorf("default warehouse not found")
	}
	return &warehouse, nil
}

// INVENTORY MANAGEMENT

// GetInventoryItem gets a specific inventory item
func (s *Service) GetInventoryItem(productID, warehouseID uint) (*InventoryItem, error) {
	var item InventoryItem
	if err := s.db.Preload("Warehouse").Where("product_id = ? AND warehouse_id = ?", productID, warehouseID).First(&item).Error; err != nil {
		return nil, fmt.Errorf("inventory item not found")
	}
	return &item, nil
}

// CreateOrUpdateInventoryItem creates or updates inventory for a product
func (s *Service) CreateOrUpdateInventoryItem(productID, warehouseID uint, sku string, initialQuantity int) (*InventoryItem, error) {
	var item InventoryItem

	// Check if item already exists
	err := s.db.Where("product_id = ? AND warehouse_id = ?", productID, warehouseID).First(&item).Error
	if err == gorm.ErrRecordNotFound {
		// Create new inventory item
		item = InventoryItem{
			ProductID:     productID,
			WarehouseID:   warehouseID,
			SKU:           sku,
			Quantity:      initialQuantity,
			Status:        InventoryStatusActive,
			ReorderLevel:  10,
			MaxStockLevel: 1000,
		}
		if err := s.db.Create(&item).Error; err != nil {
			return nil, fmt.Errorf("failed to create inventory item: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to check inventory item: %w", err)
	}

	return &item, nil
}

// STOCK MOVEMENTS

// RecordStockMovement records a stock movement and updates inventory
func (s *Service) RecordStockMovement(req *StockMovementRequest, userID uint) (*InventoryMovement, error) {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get inventory item
	var item InventoryItem
	if err := tx.Where("product_id = ? AND warehouse_id = ?", req.ProductID, req.WarehouseID).First(&item).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("inventory item not found")
	}

	previousQuantity := item.Quantity
	var newQuantity int

	// Calculate new quantity based on movement type
	switch req.MovementType {
	case MovementTypeInbound:
		newQuantity = previousQuantity + req.Quantity
	case MovementTypeOutbound:
		if previousQuantity < req.Quantity {
			tx.Rollback()
			return nil, fmt.Errorf("insufficient stock: available %d, requested %d", previousQuantity, req.Quantity)
		}
		newQuantity = previousQuantity - req.Quantity
	case MovementTypeReservation:
		if item.AvailableQuantity < req.Quantity {
			tx.Rollback()
			return nil, fmt.Errorf("insufficient available stock for reservation")
		}
		newQuantity = previousQuantity
		item.ReservedQuantity += req.Quantity
	case MovementTypeRelease:
		newQuantity = previousQuantity
		item.ReservedQuantity -= req.Quantity
		if item.ReservedQuantity < 0 {
			item.ReservedQuantity = 0
		}
	default:
		tx.Rollback()
		return nil, fmt.Errorf("invalid movement type: %s", req.MovementType)
	}

	// Update inventory item
	item.Quantity = newQuantity
	if req.CostPrice > 0 {
		item.CostPrice = req.CostPrice
	}
	if req.MovementType == MovementTypeInbound {
		now := time.Now()
		item.LastRestockDate = &now
	}

	if err := tx.Save(&item).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to update inventory: %w", err)
	}

	// Create movement record
	movement := &InventoryMovement{
		InventoryItemID:  item.ID,
		MovementType:     req.MovementType,
		Reason:           req.Reason,
		Quantity:         req.Quantity,
		PreviousQuantity: previousQuantity,
		NewQuantity:      newQuantity,
		ReferenceType:    req.ReferenceType,
		ReferenceID:      req.ReferenceID,
		Notes:            req.Notes,
		CreatedBy:        userID,
	}

	if err := tx.Create(movement).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to record movement: %w", err)
	}

	// Check for alerts
	go s.checkAndCreateAlerts(item.ID)

	tx.Commit()
	return movement, nil
}

// STOCK RESERVATIONS

// ReserveStock reserves stock for an order
func (s *Service) ReserveStock(req *ReservationRequest) (*StockReservation, error) {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get inventory item
	var item InventoryItem
	if err := tx.Where("product_id = ? AND warehouse_id = ?", req.ProductID, req.WarehouseID).First(&item).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("inventory item not found")
	}

	// Check available stock
	if !item.CanFulfillOrder(req.Quantity) {
		tx.Rollback()
		return nil, fmt.Errorf("insufficient stock: available %d, requested %d", item.AvailableQuantity, req.Quantity)
	}

	// Update reserved quantity
	item.ReservedQuantity += req.Quantity
	if err := tx.Save(&item).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to update reserved quantity: %w", err)
	}

	// Create reservation record
	reservation := &StockReservation{
		InventoryItemID: item.ID,
		OrderID:         req.OrderID,
		OrderItemID:     req.OrderItemID,
		Quantity:        req.Quantity,
		Status:          "active",
		ExpiresAt:       time.Now().Add(24 * time.Hour), // 24 hour expiry
	}

	if err := tx.Create(reservation).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create reservation: %w", err)
	}

	tx.Commit()
	return reservation, nil
}

// ReleaseReservation releases reserved stock
func (s *Service) ReleaseReservation(orderID, orderItemID uint) error {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Find active reservation
	var reservation StockReservation
	if err := tx.Where("order_id = ? AND order_item_id = ? AND status = ?", orderID, orderItemID, "active").First(&reservation).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("reservation not found")
	}

	// Update inventory item
	var item InventoryItem
	if err := tx.Where("id = ?", reservation.InventoryItemID).First(&item).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("inventory item not found")
	}

	item.ReservedQuantity -= reservation.Quantity
	if item.ReservedQuantity < 0 {
		item.ReservedQuantity = 0
	}

	if err := tx.Save(&item).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update inventory: %w", err)
	}

	// Update reservation status
	reservation.Status = "cancelled"
	if err := tx.Save(&reservation).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update reservation: %w", err)
	}

	tx.Commit()
	return nil
}

// FulfillReservation fulfills a reservation (converts to actual sale)
func (s *Service) FulfillReservation(orderID, orderItemID uint) error {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Find active reservation
	var reservation StockReservation
	if err := tx.Where("order_id = ? AND order_item_id = ? AND status = ?", orderID, orderItemID, "active").First(&reservation).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("reservation not found")
	}

	// Update inventory item
	var item InventoryItem
	if err := tx.Where("id = ?", reservation.InventoryItemID).First(&item).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("inventory item not found")
	}

	// Move from reserved to sold
	item.Quantity -= reservation.Quantity
	item.ReservedQuantity -= reservation.Quantity

	if err := tx.Save(&item).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update inventory: %w", err)
	}

	// Update reservation status
	reservation.Status = "fulfilled"
	if err := tx.Save(&reservation).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update reservation: %w", err)
	}

	tx.Commit()
	return nil
}

// GetStockLevel gets current stock level for a product
func (s *Service) GetStockLevel(productID uint, warehouseID *uint) (int, error) {
	query := s.db.Model(&InventoryItem{}).Where("product_id = ? AND status = ?", productID, InventoryStatusActive)

	if warehouseID != nil {
		query = query.Where("warehouse_id = ?", *warehouseID)
	}

	var totalStock int64
	if err := query.Select("COALESCE(SUM(available_quantity), 0)").Scan(&totalStock).Error; err != nil {
		return 0, fmt.Errorf("failed to get stock level: %w", err)
	}

	return int(totalStock), nil
}

// checkAndCreateAlerts checks for low stock and creates alerts
func (s *Service) checkAndCreateAlerts(inventoryItemID uint) {
	var item InventoryItem
	if err := s.db.Where("id = ?", inventoryItemID).First(&item).Error; err != nil {
		return
	}

	// Check for existing unresolved alerts
	var existingAlert StockAlert
	hasExisting := s.db.Where("inventory_item_id = ? AND is_resolved = ?", inventoryItemID, false).First(&existingAlert).Error == nil

	if item.IsOutOfStock() && !hasExisting {
		alert := StockAlert{
			InventoryItemID: inventoryItemID,
			AlertType:       "out_of_stock",
			Message:         fmt.Sprintf("Product %s is out of stock", item.SKU),
		}
		s.db.Create(&alert)
	} else if item.IsLowStock() && !hasExisting {
		alert := StockAlert{
			InventoryItemID: inventoryItemID,
			AlertType:       "low_stock",
			Message:         fmt.Sprintf("Product %s is running low (Available: %d, Reorder Level: %d)", item.SKU, item.AvailableQuantity, item.ReorderLevel),
		}
		s.db.Create(&alert)
	}
}
