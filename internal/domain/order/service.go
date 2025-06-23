// internal/domain/order/service.go
package order

import (
	"fmt"
	"time"

	"github.com/your-org/ecommerce-backend/internal/config"
	"github.com/your-org/ecommerce-backend/internal/domain/cart"
	"github.com/your-org/ecommerce-backend/internal/domain/product"
	"gorm.io/gorm"
)

// Service handles order business logic
type Service struct {
	db          *gorm.DB
	config      *config.Config
	cartService *cart.Service
}

// NewService creates a new order service
func NewService(db *gorm.DB, cfg *config.Config, cartService *cart.Service) *Service {
	return &Service{
		db:          db,
		config:      cfg,
		cartService: cartService,
	}
}

// CreateOrderRequest represents order creation data
type CreateOrderRequest struct {
	ShippingAddress      Address  `json:"shipping_address" binding:"required"`
	BillingAddress       *Address `json:"billing_address,omitempty"` // Optional, defaults to shipping
	ShippingMethod       string   `json:"shipping_method" binding:"required"`
	PaymentMethod        string   `json:"payment_method" binding:"required"`
	Notes                string   `json:"notes,omitempty"`
	CouponCode           string   `json:"coupon_code,omitempty"`
	UseShippingAsBilling bool     `json:"use_shipping_as_billing"`
}

// OrderListRequest represents order list query parameters
type OrderListRequest struct {
	Page      int         `form:"page,default=1"`
	Limit     int         `form:"limit,default=20"`
	Status    OrderStatus `form:"status"`
	UserID    uint        `form:"user_id"`
	SortBy    string      `form:"sort_by,default=created_at"`
	SortOrder string      `form:"sort_order,default=desc"`
	DateFrom  string      `form:"date_from"`
	DateTo    string      `form:"date_to"`
}

// OrderResponse represents order response with pagination
type OrderResponse struct {
	Orders     []Order    `json:"orders"`
	Pagination Pagination `json:"pagination"`
}

// Pagination represents pagination information
type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// CreateOrder creates a new order from user's cart
func (s *Service) CreateOrder(userID uint, sessionID string, req *CreateOrderRequest) (*Order, error) {
	// Start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get user's cart
	userIDPtr := &userID
	cartResponse, err := s.cartService.GetCart(userIDPtr, sessionID)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to retrieve cart: %w", err)
	}

	// Validate cart is not empty
	if len(cartResponse.Items) == 0 {
		tx.Rollback()
		return nil, fmt.Errorf("cart is empty")
	}

	// Validate cart items (inventory, pricing, etc.)
	if err := s.validateCartItems(cartResponse.Items); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("cart validation failed: %w", err)
	}

	// Calculate totals
	subtotal := s.calculateSubtotal(cartResponse.Items)
	taxAmount := s.calculateTax(subtotal, req.ShippingAddress)
	shippingCost := s.calculateShipping(req.ShippingMethod, cartResponse.Items)
	discountAmount := s.calculateDiscount(req.CouponCode, subtotal)
	totalAmount := subtotal + taxAmount + shippingCost - discountAmount

	// Set billing address
	billingAddress := req.ShippingAddress
	if !req.UseShippingAsBilling && req.BillingAddress != nil {
		billingAddress = *req.BillingAddress
	}

	// Create order
	order := Order{
		UserID:          &userID,
		Email:           "", // We'll get this from user
		Status:          OrderStatusPending,
		PaymentStatus:   PaymentStatusPending,
		SubtotalAmount:  subtotal,
		TaxAmount:       taxAmount,
		ShippingAmount:  shippingCost,
		DiscountAmount:  discountAmount,
		TotalAmount:     totalAmount,
		ShippingAddress: req.ShippingAddress,
		BillingAddress:  billingAddress,
		Currency:        "USD", // TODO: Make configurable
		Notes:           req.Notes,
		CouponCode:      req.CouponCode,
		ShippingMethod:  req.ShippingMethod,
	}

	// Get user email
	var user struct {
		Email string
	}
	if err := tx.Select("email").Where("id = ?", userID).First(&user).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to get user email: %w", err)
	}
	order.Email = user.Email

	// Save order
	if err := tx.Create(&order).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Generate order number
	order.OrderNumber = s.generateOrderNumber(order.ID)
	if err := tx.Model(&order).Update("order_number", order.OrderNumber).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to update order number: %w", err)
	}

	// Create order items
	for _, cartItem := range cartResponse.Items {
		orderItem := OrderItem{
			OrderID:          order.ID,
			ProductID:        cartItem.ProductID,
			ProductVariantID: cartItem.ProductVariantID,
			SKU:              cartItem.Product.SKU,
			Name:             cartItem.Product.Name,
			Quantity:         cartItem.Quantity,
			Price:            cartItem.Price,
			TotalPrice:       cartItem.Price * int64(cartItem.Quantity),
		}

		// Add variant title if applicable
		if cartItem.ProductVariant != nil {
			orderItem.VariantTitle = cartItem.ProductVariant.Name
		}

		if err := tx.Create(&orderItem).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to create order item: %w", err)
		}
	}

	// Reserve inventory
	if err := s.reserveInventory(tx, cartResponse.Items); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to reserve inventory: %w", err)
	}

	// Add initial status history
	order.AddStatusHistory(OrderStatusPending, "Order created", userID)
	for _, history := range order.StatusHistory {
		if err := tx.Create(&history).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to create status history: %w", err)
		}
	}

	// Clear user's cart
	if err := s.cartService.ClearCart(userIDPtr, sessionID); err != nil {
		// Log error but don't fail the order
		// In production, you might want to handle this differently
		fmt.Printf("Warning: failed to clear cart after order creation: %v\n", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit order transaction: %w", err)
	}

	// Load complete order with relationships
	if err := s.db.Preload("Items").Preload("StatusHistory").First(&order, order.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to load complete order: %w", err)
	}

	return &order, nil
}

// GetOrders retrieves orders with filtering and pagination
func (s *Service) GetOrders(req *OrderListRequest) (*OrderResponse, error) {
	var orders []Order
	var total int64

	// Build query
	query := s.db.Model(&Order{}).
		Preload("Items").
		Preload("StatusHistory", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC")
		})

	// Apply filters
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	if req.UserID > 0 {
		query = query.Where("user_id = ?", req.UserID)
	}

	if req.DateFrom != "" {
		query = query.Where("created_at >= ?", req.DateFrom)
	}

	if req.DateTo != "" {
		query = query.Where("created_at <= ?", req.DateTo)
	}

	// Count total records
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count orders: %w", err)
	}

	// Apply sorting
	orderClause := s.buildOrderClause(req.SortBy, req.SortOrder)
	query = query.Order(orderClause)

	// Apply pagination
	offset := (req.Page - 1) * req.Limit
	if err := query.Offset(offset).Limit(req.Limit).Find(&orders).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve orders: %w", err)
	}

	// Calculate pagination info
	totalPages := int((total + int64(req.Limit) - 1) / int64(req.Limit))
	pagination := Pagination{
		Page:       req.Page,
		Limit:      req.Limit,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    req.Page < totalPages,
		HasPrev:    req.Page > 1,
	}

	return &OrderResponse{
		Orders:     orders,
		Pagination: pagination,
	}, nil
}

// GetOrder retrieves a single order by ID
func (s *Service) GetOrder(id uint) (*Order, error) {
	var order Order
	result := s.db.
		Preload("Items").
		Preload("StatusHistory", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC")
		}).
		Where("id = ?", id).
		First(&order)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("order not found")
		}
		return nil, fmt.Errorf("failed to retrieve order: %w", result.Error)
	}

	return &order, nil
}

// GetOrderByNumber retrieves a single order by order number
func (s *Service) GetOrderByNumber(orderNumber string) (*Order, error) {
	var order Order
	result := s.db.
		Preload("Items").
		Preload("StatusHistory", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC")
		}).
		Where("order_number = ?", orderNumber).
		First(&order)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("order not found")
		}
		return nil, fmt.Errorf("failed to retrieve order: %w", result.Error)
	}

	return &order, nil
}

// UpdateOrderStatus updates order status
func (s *Service) UpdateOrderStatus(orderID uint, status OrderStatus, comment string, updatedBy uint) error {
	// Get current order
	var order Order
	if err := s.db.First(&order, orderID).Error; err != nil {
		return fmt.Errorf("order not found: %w", err)
	}

	// Validate status transition
	if !s.isValidStatusTransition(order.Status, status) {
		return fmt.Errorf("invalid status transition from %s to %s", order.Status, status)
	}

	// Update order status
	updates := map[string]interface{}{
		"status": status,
	}

	// Set timestamps based on status
	now := time.Now().UTC()
	switch status {
	case OrderStatusProcessing:
		updates["processed_at"] = now
	case OrderStatusShipped:
		updates["shipped_at"] = now
	case OrderStatusDelivered:
		updates["delivered_at"] = now
	}

	if err := s.db.Model(&order).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// Add status history
	statusHistory := OrderStatusHistory{
		OrderID:   orderID,
		Status:    status,
		Comment:   comment,
		CreatedBy: updatedBy,
		CreatedAt: now,
	}

	if err := s.db.Create(&statusHistory).Error; err != nil {
		return fmt.Errorf("failed to create status history: %w", err)
	}

	return nil
}

// CancelOrder cancels an order
func (s *Service) CancelOrder(orderID uint, reason string, cancelledBy uint) error {
	var order Order
	if err := s.db.First(&order, orderID).Error; err != nil {
		return fmt.Errorf("order not found: %w", err)
	}

	if !order.CanBeCancelled() {
		return fmt.Errorf("order cannot be cancelled in current status: %s", order.Status)
	}

	// Start transaction for inventory restoration
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Restore inventory
	if err := s.restoreInventory(tx, orderID); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to restore inventory: %w", err)
	}

	// Update order status
	if err := tx.Model(&order).Updates(map[string]interface{}{
		"status": OrderStatusCancelled,
	}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// Add status history
	statusHistory := OrderStatusHistory{
		OrderID:   orderID,
		Status:    OrderStatusCancelled,
		Comment:   fmt.Sprintf("Order cancelled: %s", reason),
		CreatedBy: cancelledBy,
		CreatedAt: time.Now().UTC(),
	}

	if err := tx.Create(&statusHistory).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create status history: %w", err)
	}

	return tx.Commit().Error
}

// GetUserOrders retrieves orders for a specific user
func (s *Service) GetUserOrders(userID uint, page, limit int) (*OrderResponse, error) {
	req := &OrderListRequest{
		Page:   page,
		Limit:  limit,
		UserID: userID,
	}
	return s.GetOrders(req)
}

// Private helper methods

func (s *Service) validateCartItems(items []cart.CartItemResponse) error {
	for _, item := range items {
		if item.Product == nil {
			return fmt.Errorf("product %d not found", item.ProductID)
		}

		if !item.Product.IsActive {
			return fmt.Errorf("product '%s' is no longer available", item.Product.Name)
		}

		// Check inventory
		availableQuantity := item.Product.Quantity
		if item.ProductVariant != nil {
			availableQuantity = item.ProductVariant.Quantity
		}

		if item.Product.TrackQuantity && availableQuantity < item.Quantity {
			return fmt.Errorf("insufficient inventory for product '%s'. Available: %d, Requested: %d",
				item.Product.Name, availableQuantity, item.Quantity)
		}
	}
	return nil
}

func (s *Service) calculateSubtotal(items []cart.CartItemResponse) int64 {
	var subtotal int64
	for _, item := range items {
		subtotal += item.Price * int64(item.Quantity)
	}
	return subtotal
}

func (s *Service) calculateTax(subtotal int64, address Address) int64 {
	// TODO: Implement tax calculation based on address
	// For now, return 0
	return 0
}

func (s *Service) calculateShipping(method string, items []cart.CartItemResponse) int64 {
	// TODO: Implement shipping calculation based on method and items
	// For now, return flat rate
	switch method {
	case "standard":
		return 999 // $9.99
	case "express":
		return 1999 // $19.99
	case "overnight":
		return 2999 // $29.99
	default:
		return 999
	}
}

func (s *Service) calculateDiscount(couponCode string, subtotal int64) int64 {
	// TODO: Implement coupon/discount calculation
	// For now, return 0
	return 0
}

func (s *Service) generateOrderNumber(orderID uint) string {
	// Format: ORD-YYYYMMDD-XXXXX
	return fmt.Sprintf("ORD-%s-%05d", time.Now().Format("20060102"), orderID)
}

func (s *Service) reserveInventory(tx *gorm.DB, items []cart.CartItemResponse) error {
	for _, item := range items {
		if !item.Product.TrackQuantity {
			continue
		}

		if item.ProductVariant != nil {
			// Update variant inventory
			result := tx.Model(&product.ProductVariant{}).
				Where("id = ?", *item.ProductVariantID).
				UpdateColumn("quantity", gorm.Expr("quantity - ?", item.Quantity))

			if result.Error != nil {
				return fmt.Errorf("failed to update variant inventory: %w", result.Error)
			}
		} else {
			// Update product inventory
			result := tx.Model(&product.Product{}).
				Where("id = ?", item.ProductID).
				UpdateColumn("quantity", gorm.Expr("quantity - ?", item.Quantity))

			if result.Error != nil {
				return fmt.Errorf("failed to update product inventory: %w", result.Error)
			}
		}
	}
	return nil
}

func (s *Service) restoreInventory(tx *gorm.DB, orderID uint) error {
	var orderItems []OrderItem
	if err := tx.Where("order_id = ?", orderID).Find(&orderItems).Error; err != nil {
		return fmt.Errorf("failed to get order items: %w", err)
	}

	for _, item := range orderItems {
		if item.ProductVariantID != nil {
			// Restore variant inventory
			tx.Model(&product.ProductVariant{}).
				Where("id = ?", *item.ProductVariantID).
				UpdateColumn("quantity", gorm.Expr("quantity + ?", item.Quantity))
		} else {
			// Restore product inventory
			tx.Model(&product.Product{}).
				Where("id = ?", item.ProductID).
				UpdateColumn("quantity", gorm.Expr("quantity + ?", item.Quantity))
		}
	}
	return nil
}

func (s *Service) isValidStatusTransition(from, to OrderStatus) bool {
	validTransitions := map[OrderStatus][]OrderStatus{
		OrderStatusPending: {
			OrderStatusPaymentProcessing,
			OrderStatusConfirmed,
			OrderStatusCancelled,
		},
		OrderStatusPaymentProcessing: {
			OrderStatusConfirmed,
			OrderStatusCancelled,
		},
		OrderStatusConfirmed: {
			OrderStatusProcessing,
			OrderStatusCancelled,
		},
		OrderStatusProcessing: {
			OrderStatusShipped,
			OrderStatusCancelled,
		},
		OrderStatusShipped: {
			OrderStatusOutForDelivery,
			OrderStatusDelivered,
		},
		OrderStatusOutForDelivery: {
			OrderStatusDelivered,
		},
		OrderStatusDelivered: {
			OrderStatusCompleted,
			OrderStatusRefunded,
		},
	}

	allowedStatuses, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, status := range allowedStatuses {
		if status == to {
			return true
		}
	}
	return false
}

func (s *Service) buildOrderClause(sortBy, sortOrder string) string {
	validSortFields := map[string]bool{
		"created_at":   true,
		"updated_at":   true,
		"total_amount": true,
		"status":       true,
		"order_number": true,
	}

	if !validSortFields[sortBy] {
		sortBy = "created_at"
	}

	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	return fmt.Sprintf("%s %s", sortBy, sortOrder)
}
