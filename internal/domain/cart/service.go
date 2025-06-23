// internal/domain/cart/service.go
package cart

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/your-org/ecommerce-backend/internal/config"
	"github.com/your-org/ecommerce-backend/internal/domain/product"
	"gorm.io/gorm"
)

// Service handles cart business logic
type Service struct {
	db          *gorm.DB
	redisClient *redis.Client
	config      *config.Config
}

// NewService creates a new cart service
func NewService(db *gorm.DB, redisClient *redis.Client, cfg *config.Config) *Service {
	return &Service{
		db:          db,
		redisClient: redisClient,
		config:      cfg,
	}
}

// CartItemResponse represents a cart item with product details
type CartItemResponse struct {
	ProductID        uint                    `json:"product_id"`
	ProductVariantID *uint                   `json:"product_variant_id,omitempty"`
	Quantity         int                     `json:"quantity"`
	Price            int64                   `json:"price"`
	Product          *product.Product        `json:"product,omitempty"`
	ProductVariant   *product.ProductVariant `json:"product_variant,omitempty"`
	AddedAt          time.Time               `json:"added_at"`
}

// CartResponse represents a shopping cart with items and summary
type CartResponse struct {
	SessionID string             `json:"session_id,omitempty"`
	UserID    *uint              `json:"user_id,omitempty"`
	Items     []CartItemResponse `json:"items"`
	Totals    CartTotals         `json:"totals"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
}

// AddToCartRequest represents add to cart request
type AddToCartRequest struct {
	ProductID        uint  `json:"product_id" binding:"required"`
	ProductVariantID *uint `json:"product_variant_id"`
	Quantity         int   `json:"quantity" binding:"required,min=1"`
}

// UpdateCartItemRequest represents update cart item request
type UpdateCartItemRequest struct {
	Quantity int `json:"quantity" binding:"required,min=0"`
}

// GetCart retrieves cart for user or session
func (s *Service) GetCart(userID *uint, sessionID string) (*CartResponse, error) {
	var cartItems []CartItemResponse
	var createdAt, updatedAt time.Time

	if userID != nil {
		// Get user cart from database
		var dbItems []CartItem
		err := s.db.Where("user_id = ?", *userID).Find(&dbItems).Error
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve user cart: %w", err)
		}

		// Convert to response format
		cartItems = make([]CartItemResponse, len(dbItems))
		for i, item := range dbItems {
			cartItems[i] = CartItemResponse{
				ProductID:        item.ProductID,
				ProductVariantID: item.ProductVariantID,
				Quantity:         item.Quantity,
				Price:            item.Price,
				AddedAt:          item.CreatedAt,
			}
		}

		if len(dbItems) > 0 {
			createdAt = dbItems[0].CreatedAt
			updatedAt = dbItems[0].UpdatedAt
		} else {
			createdAt = time.Now().UTC()
			updatedAt = time.Now().UTC()
		}
	} else {
		// Get guest cart from Redis
		sessionCart, err := s.getGuestCart(sessionID)
		if err != nil {
			return nil, err
		}

		// Convert to response format
		cartItems = make([]CartItemResponse, len(sessionCart.Items))
		for i, item := range sessionCart.Items {
			cartItems[i] = CartItemResponse{
				ProductID:        item.ProductID,
				ProductVariantID: item.ProductVariantID,
				Quantity:         item.Quantity,
				Price:            item.Price,
				AddedAt:          item.AddedAt,
			}
		}

		createdAt = sessionCart.CreatedAt
		updatedAt = sessionCart.UpdatedAt
	}

	// Load product details for each item
	if err := s.loadProductDetails(cartItems); err != nil {
		return nil, err
	}

	// Calculate totals
	totals := s.calculateTotals(cartItems)

	return &CartResponse{
		SessionID: sessionID,
		UserID:    userID,
		Items:     cartItems,
		Totals:    totals,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

// AddToCart adds an item to the cart
func (s *Service) AddToCart(userID *uint, sessionID string, req *AddToCartRequest) (*CartResponse, error) {
	// Validate product exists and is active
	var prod product.Product
	result := s.db.Where("id = ? AND is_active = ?", req.ProductID, true).First(&prod)
	if result.Error != nil {
		return nil, fmt.Errorf("product not found or inactive")
	}

	// Validate variant if specified
	var variant *product.ProductVariant
	if req.ProductVariantID != nil {
		var v product.ProductVariant
		result := s.db.Where("id = ? AND product_id = ? AND is_active = ?",
			*req.ProductVariantID, req.ProductID, true).First(&v)
		if result.Error != nil {
			return nil, fmt.Errorf("product variant not found or inactive")
		}
		variant = &v
	}

	// Check inventory availability
	availableQuantity := prod.Quantity
	if variant != nil {
		availableQuantity = variant.Quantity
	}

	if prod.TrackQuantity && availableQuantity < req.Quantity {
		return nil, fmt.Errorf("insufficient inventory. Available: %d", availableQuantity)
	}

	// Determine price to use
	itemPrice := prod.Price
	if variant != nil && variant.Price > 0 {
		itemPrice = variant.Price
	}

	if userID != nil {
		// Handle user cart
		err := s.addToUserCart(*userID, req.ProductID, req.ProductVariantID, req.Quantity, itemPrice, availableQuantity, prod.TrackQuantity)
		if err != nil {
			return nil, err
		}
	} else {
		// Handle guest cart
		err := s.addToGuestCart(sessionID, req.ProductID, req.ProductVariantID, req.Quantity, itemPrice, availableQuantity, prod.TrackQuantity)
		if err != nil {
			return nil, err
		}
	}

	// Return updated cart
	return s.GetCart(userID, sessionID)
}

// UpdateCartItem updates quantity of a cart item
func (s *Service) UpdateCartItem(userID *uint, sessionID string, productID uint, variantID *uint, req *UpdateCartItemRequest) (*CartResponse, error) {
	if req.Quantity < 0 {
		return nil, fmt.Errorf("quantity cannot be negative")
	}

	if req.Quantity > 0 {
		// Validate inventory if updating to non-zero quantity
		var prod product.Product
		s.db.Where("id = ?", productID).First(&prod)

		availableQuantity := prod.Quantity
		if variantID != nil {
			var variant product.ProductVariant
			s.db.Where("id = ?", *variantID).First(&variant)
			availableQuantity = variant.Quantity
		}

		if prod.TrackQuantity && availableQuantity < req.Quantity {
			return nil, fmt.Errorf("insufficient inventory. Available: %d", availableQuantity)
		}
	}

	if userID != nil {
		// Update user cart
		err := s.updateUserCartItem(*userID, productID, variantID, req.Quantity)
		if err != nil {
			return nil, err
		}
	} else {
		// Update guest cart
		err := s.updateGuestCartItem(sessionID, productID, variantID, req.Quantity)
		if err != nil {
			return nil, err
		}
	}

	// Return updated cart
	return s.GetCart(userID, sessionID)
}

// RemoveFromCart removes an item from the cart
func (s *Service) RemoveFromCart(userID *uint, sessionID string, productID uint, variantID *uint) (*CartResponse, error) {
	return s.UpdateCartItem(userID, sessionID, productID, variantID, &UpdateCartItemRequest{Quantity: 0})
}

// ClearCart removes all items from the cart
func (s *Service) ClearCart(userID *uint, sessionID string) error {
	if userID != nil {
		// Clear user cart from database
		return s.db.Where("user_id = ?", *userID).Delete(&CartItem{}).Error
	} else {
		// Clear guest cart from Redis
		ctx := context.Background()
		cartKey := fmt.Sprintf("cart:session:%s", sessionID)
		return s.redisClient.Del(ctx, cartKey).Err()
	}
}

// GetCartItemCount returns the number of items in cart
func (s *Service) GetCartItemCount(userID *uint, sessionID string) (int, error) {
	cartResponse, err := s.GetCart(userID, sessionID)
	if err != nil {
		return 0, nil // Return 0 if cart doesn't exist
	}

	totalItems := 0
	for _, item := range cartResponse.Items {
		totalItems += item.Quantity
	}

	return totalItems, nil
}

// MergeGuestCartToUser merges guest cart to user cart when user logs in
func (s *Service) MergeGuestCartToUser(userID uint, sessionID string) error {
	// Get guest cart
	guestCart, err := s.getGuestCart(sessionID)
	if err != nil || len(guestCart.Items) == 0 {
		return nil // No guest cart to merge
	}

	// Merge each guest cart item
	for _, guestItem := range guestCart.Items {
		// Check if item already exists in user cart
		var existingItem CartItem
		result := s.db.Where("user_id = ? AND product_id = ? AND product_variant_id = ?",
			userID, guestItem.ProductID, guestItem.ProductVariantID).First(&existingItem)

		if result.Error == gorm.ErrRecordNotFound {
			// Item doesn't exist, create new
			newItem := CartItem{
				UserID:           &userID,
				ProductID:        guestItem.ProductID,
				ProductVariantID: guestItem.ProductVariantID,
				Quantity:         guestItem.Quantity,
				Price:            guestItem.Price,
			}
			s.db.Create(&newItem)
		} else {
			// Item exists, update quantity
			existingItem.Quantity += guestItem.Quantity
			s.db.Save(&existingItem)
		}
	}

	// Clear guest cart
	return s.ClearCart(nil, sessionID)
}

// Private helper methods

func (s *Service) addToUserCart(userID, productID uint, variantID *uint, quantity int, price int64, availableQuantity int, trackQuantity bool) error {
	// Check if item already exists
	var existingItem CartItem
	result := s.db.Where("user_id = ? AND product_id = ? AND product_variant_id = ?",
		userID, productID, variantID).First(&existingItem)

	if result.Error == gorm.ErrRecordNotFound {
		// Item doesn't exist, create new
		newItem := CartItem{
			UserID:           &userID,
			ProductID:        productID,
			ProductVariantID: variantID,
			Quantity:         quantity,
			Price:            price,
		}
		return s.db.Create(&newItem).Error
	} else {
		// Item exists, update quantity
		newQuantity := existingItem.Quantity + quantity

		// Check inventory for new total quantity
		if trackQuantity && availableQuantity < newQuantity {
			return fmt.Errorf("insufficient inventory for total quantity. Available: %d", availableQuantity)
		}

		existingItem.Quantity = newQuantity
		existingItem.Price = price // Update price in case it changed
		return s.db.Save(&existingItem).Error
	}
}

func (s *Service) addToGuestCart(sessionID string, productID uint, variantID *uint, quantity int, price int64, availableQuantity int, trackQuantity bool) error {
	sessionCart, err := s.getGuestCart(sessionID)
	if err != nil {
		return err
	}

	// Check if item already exists
	itemExists := false
	for i := range sessionCart.Items {
		if sessionCart.Items[i].ProductID == productID &&
			((sessionCart.Items[i].ProductVariantID == nil && variantID == nil) ||
				(sessionCart.Items[i].ProductVariantID != nil && variantID != nil &&
					*sessionCart.Items[i].ProductVariantID == *variantID)) {

			// Update existing item quantity
			newQuantity := sessionCart.Items[i].Quantity + quantity

			// Check inventory for new total quantity
			if trackQuantity && availableQuantity < newQuantity {
				return fmt.Errorf("insufficient inventory for total quantity. Available: %d", availableQuantity)
			}

			sessionCart.Items[i].Quantity = newQuantity
			sessionCart.Items[i].Price = price // Update price in case it changed
			itemExists = true
			break
		}
	}

	// Add new item if it doesn't exist
	if !itemExists {
		newItem := SessionCartItem{
			ProductID:        productID,
			ProductVariantID: variantID,
			Quantity:         quantity,
			Price:            price,
			AddedAt:          time.Now().UTC(),
		}
		sessionCart.Items = append(sessionCart.Items, newItem)
	}

	sessionCart.UpdatedAt = time.Now().UTC()
	return s.saveGuestCart(sessionID, sessionCart)
}

func (s *Service) updateUserCartItem(userID, productID uint, variantID *uint, quantity int) error {
	if quantity == 0 {
		// Remove item
		return s.db.Where("user_id = ? AND product_id = ? AND product_variant_id = ?",
			userID, productID, variantID).Delete(&CartItem{}).Error
	} else {
		// Update quantity
		return s.db.Model(&CartItem{}).
			Where("user_id = ? AND product_id = ? AND product_variant_id = ?", userID, productID, variantID).
			Update("quantity", quantity).Error
	}
}

func (s *Service) updateGuestCartItem(sessionID string, productID uint, variantID *uint, quantity int) error {
	sessionCart, err := s.getGuestCart(sessionID)
	if err != nil {
		return err
	}

	// Find and update the item
	itemFound := false
	for i := range sessionCart.Items {
		if sessionCart.Items[i].ProductID == productID &&
			((sessionCart.Items[i].ProductVariantID == nil && variantID == nil) ||
				(sessionCart.Items[i].ProductVariantID != nil && variantID != nil &&
					*sessionCart.Items[i].ProductVariantID == *variantID)) {

			if quantity == 0 {
				// Remove item from cart
				sessionCart.Items = append(sessionCart.Items[:i], sessionCart.Items[i+1:]...)
			} else {
				sessionCart.Items[i].Quantity = quantity
			}

			itemFound = true
			break
		}
	}

	if !itemFound {
		return fmt.Errorf("item not found in cart")
	}

	sessionCart.UpdatedAt = time.Now().UTC()
	return s.saveGuestCart(sessionID, sessionCart)
}

func (s *Service) getGuestCart(sessionID string) (*SessionCart, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID required for guest cart")
	}

	ctx := context.Background()
	cartKey := fmt.Sprintf("cart:session:%s", sessionID)

	cartData, err := s.redisClient.Get(ctx, cartKey).Result()
	if err == redis.Nil {
		// Cart doesn't exist, return empty cart
		return &SessionCart{
			SessionID: sessionID,
			Items:     []SessionCartItem{},
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
			ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		}, nil
	} else if err != nil {
		return nil, err
	}

	var sessionCart SessionCart
	if err := json.Unmarshal([]byte(cartData), &sessionCart); err != nil {
		return nil, err
	}

	return &sessionCart, nil
}

func (s *Service) saveGuestCart(sessionID string, cart *SessionCart) error {
	ctx := context.Background()
	cartKey := fmt.Sprintf("cart:session:%s", sessionID)

	cartData, err := json.Marshal(cart)
	if err != nil {
		return err
	}

	// Set cart with 24 hour expiration
	return s.redisClient.Set(ctx, cartKey, cartData, 24*time.Hour).Err()
}

func (s *Service) loadProductDetails(cartItems []CartItemResponse) error {
	for i := range cartItems {
		// Load product details
		var prod product.Product
		err := s.db.Preload("Category").Preload("Brand").
			Where("id = ?", cartItems[i].ProductID).First(&prod).Error
		if err != nil {
			continue // Skip if product not found
		}
		cartItems[i].Product = &prod

		// Load variant details if applicable
		if cartItems[i].ProductVariantID != nil {
			var variant product.ProductVariant
			err := s.db.Where("id = ?", *cartItems[i].ProductVariantID).First(&variant).Error
			if err == nil {
				cartItems[i].ProductVariant = &variant
			}
		}
	}

	return nil
}

func (s *Service) calculateTotals(cartItems []CartItemResponse) CartTotals {
	var totals CartTotals

	totals.ItemCount = len(cartItems)

	for _, item := range cartItems {
		totals.TotalQuantity += item.Quantity
		totals.SubTotal += item.Price * int64(item.Quantity)
	}

	// TODO: Calculate tax and shipping based on business rules
	totals.TaxAmount = 0
	totals.ShippingCost = 0
	totals.DiscountAmount = 0
	totals.TotalAmount = totals.SubTotal + totals.TaxAmount + totals.ShippingCost - totals.DiscountAmount

	return totals
}
