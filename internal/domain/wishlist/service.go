package wishlist

import (
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/your-org/ecommerce-backend/internal/config"
	"github.com/your-org/ecommerce-backend/internal/domain/cart"
	"github.com/your-org/ecommerce-backend/internal/domain/product"
	"gorm.io/gorm"
)

// Service handles wishlist business logic
type Service struct {
	db          *gorm.DB
	redisClient *redis.Client
	config      *config.Config
	cartService *cart.Service
}

// NewService creates a new wishlist service
func NewService(db *gorm.DB, redisClient *redis.Client, cfg *config.Config) *Service {
	return &Service{
		db:          db,
		redisClient: redisClient,
		config:      cfg,
		cartService: cart.NewService(db, redisClient, cfg),
	}
}

// WishlistItemResponse represents a wishlist item with product details
type WishlistItemResponse struct {
	ID               uint                    `json:"id"`
	ProductID        uint                    `json:"product_id"`
	ProductVariantID *uint                   `json:"product_variant_id,omitempty"`
	Product          *product.Product        `json:"product,omitempty"`
	ProductVariant   *product.ProductVariant `json:"product_variant,omitempty"`
	AddedAt          time.Time               `json:"added_at"`
	IsAvailable      bool                    `json:"is_available"`
	CurrentPrice     int64                   `json:"current_price"`
	PriceChanged     bool                    `json:"price_changed"`
	OriginalPrice    int64                   `json:"original_price,omitempty"`
}

// WishlistResponse represents a wishlist with items and pagination
type WishlistResponse struct {
	Items      []WishlistItemResponse `json:"items"`
	Count      int                    `json:"count"`
	Pagination Pagination             `json:"pagination"`
	Summary    WishlistSummary        `json:"summary"`
}

// WishlistSummary provides summary information
type WishlistSummary struct {
	TotalItems       int     `json:"total_items"`
	AvailableItems   int     `json:"available_items"`
	UnavailableItems int     `json:"unavailable_items"`
	TotalValue       int64   `json:"total_value"`
	AveragePrice     float64 `json:"average_price"`
	RecentlyAdded    int     `json:"recently_added"` // Items added in last 7 days
}

// Pagination represents pagination information
type Pagination struct {
	Page       int  `json:"page"`
	Limit      int  `json:"limit"`
	Total      int  `json:"total"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

// AddToWishlistRequest represents add to wishlist request
type AddToWishlistRequest struct {
	ProductID        uint  `json:"product_id" binding:"required"`
	ProductVariantID *uint `json:"product_variant_id"`
}

// BulkAddResult represents bulk add operation result
type BulkAddResult struct {
	Added    []uint   `json:"added"`
	Skipped  []uint   `json:"skipped"`
	Failed   []uint   `json:"failed"`
	Messages []string `json:"messages,omitempty"`
}

// GetWishlist retrieves wishlist for a user with pagination
func (s *Service) GetWishlist(userID uint, page, limit int, sortBy, sortOrder string) (*WishlistResponse, error) {
	var items []WishlistItem
	var total int64

	// Build query
	query := s.db.Where("user_id = ?", userID)

	// Count total items
	if err := query.Model(&WishlistItem{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count wishlist items: %w", err)
	}

	// Apply sorting
	orderClause := s.buildOrderClause(sortBy, sortOrder)
	query = query.Order(orderClause)

	// Apply pagination
	offset := (page - 1) * limit
	if err := query.Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve wishlist items: %w", err)
	}

	// Convert to response format and load product details
	wishlistItems := make([]WishlistItemResponse, len(items))
	for i, item := range items {
		wishlistItems[i] = WishlistItemResponse{
			ID:               item.ID,
			ProductID:        item.ProductID,
			ProductVariantID: item.ProductVariantID,
			AddedAt:          item.AddedAt,
		}
	}

	// Load product details
	if err := s.loadProductDetails(wishlistItems); err != nil {
		return nil, err
	}

	// Calculate pagination
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	pagination := Pagination{
		Page:       page,
		Limit:      limit,
		Total:      int(total),
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}

	// Generate summary
	summary := s.generateWishlistSummary(userID, wishlistItems)

	return &WishlistResponse{
		Items:      wishlistItems,
		Count:      len(wishlistItems),
		Pagination: pagination,
		Summary:    summary,
	}, nil
}

// AddToWishlist adds an item to the wishlist
func (s *Service) AddToWishlist(userID uint, req *AddToWishlistRequest) (*WishlistItemResponse, error) {
	// Validate product exists and is active
	var prod product.Product
	result := s.db.Where("id = ? AND is_active = ?", req.ProductID, true).First(&prod)
	if result.Error != nil {
		return nil, fmt.Errorf("product not found or inactive")
	}

	// Validate variant if specified
	if req.ProductVariantID != nil {
		var variant product.ProductVariant
		result := s.db.Where("id = ? AND product_id = ? AND is_active = ?",
			*req.ProductVariantID, req.ProductID, true).First(&variant)
		if result.Error != nil {
			return nil, fmt.Errorf("product variant not found or inactive")
		}
	}

	// Check if item already exists in wishlist
	var existingItem WishlistItem
	query := s.db.Where("user_id = ? AND product_id = ?", userID, req.ProductID)
	if req.ProductVariantID == nil {
		query = query.Where("product_variant_id IS NULL")
	} else {
		query = query.Where("product_variant_id = ?", *req.ProductVariantID)
	}

	if query.First(&existingItem).Error == nil {
		return nil, fmt.Errorf("item already exists in wishlist")
	}

	// Create wishlist item
	now := time.Now().UTC()
	wishlistItem := WishlistItem{
		UserID:           userID,
		ProductID:        req.ProductID,
		ProductVariantID: req.ProductVariantID,
		AddedAt:          now,
	}

	if err := s.db.Create(&wishlistItem).Error; err != nil {
		return nil, fmt.Errorf("failed to add item to wishlist: %w", err)
	}

	// Return item with product details
	response := WishlistItemResponse{
		ID:               wishlistItem.ID,
		ProductID:        wishlistItem.ProductID,
		ProductVariantID: wishlistItem.ProductVariantID,
		AddedAt:          wishlistItem.AddedAt,
	}

	responseItems := []WishlistItemResponse{response}
	if err := s.loadProductDetails(responseItems); err != nil {
		return nil, err
	}

	return &responseItems[0], nil
}

// RemoveFromWishlist removes an item from the wishlist
func (s *Service) RemoveFromWishlist(userID, productID uint, variantID *uint) error {
	query := s.db.Where("user_id = ? AND product_id = ?", userID, productID)

	if variantID == nil {
		query = query.Where("product_variant_id IS NULL")
	} else {
		query = query.Where("product_variant_id = ?", *variantID)
	}

	result := query.Delete(&WishlistItem{})
	if result.Error != nil {
		return fmt.Errorf("failed to remove item from wishlist: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("item not found in wishlist")
	}

	return nil
}

// ClearWishlist removes all items from the wishlist
func (s *Service) ClearWishlist(userID uint) error {
	return s.db.Where("user_id = ?", userID).Delete(&WishlistItem{}).Error
}

// GetWishlistCount returns the number of items in wishlist
func (s *Service) GetWishlistCount(userID uint) (int64, error) {
	var count int64
	err := s.db.Model(&WishlistItem{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

// IsInWishlist checks if a product is in the user's wishlist
func (s *Service) IsInWishlist(userID, productID uint, variantID *uint) (bool, error) {
	var count int64
	query := s.db.Model(&WishlistItem{}).Where("user_id = ? AND product_id = ?", userID, productID)

	if variantID == nil {
		query = query.Where("product_variant_id IS NULL")
	} else {
		query = query.Where("product_variant_id = ?", *variantID)
	}

	err := query.Count(&count).Error
	return count > 0, err
}

// MoveToCart moves an item from wishlist to cart
func (s *Service) MoveToCart(userID, productID uint, variantID *uint, quantity int) error {
	// Check if item exists in wishlist
	inWishlist, err := s.IsInWishlist(userID, productID, variantID)
	if err != nil {
		return err
	}
	if !inWishlist {
		return fmt.Errorf("item not found in wishlist")
	}

	// Add to cart
	userIDPtr := &userID
	addToCartReq := &cart.AddToCartRequest{
		ProductID:        productID,
		ProductVariantID: variantID,
		Quantity:         quantity,
	}

	_, err = s.cartService.AddToCart(userIDPtr, "", addToCartReq)
	if err != nil {
		return fmt.Errorf("failed to add item to cart: %w", err)
	}

	// Remove from wishlist
	return s.RemoveFromWishlist(userID, productID, variantID)
}

// BulkAddToWishlist adds multiple products to wishlist
func (s *Service) BulkAddToWishlist(userID uint, productIDs []uint) (*BulkAddResult, error) {
	result := &BulkAddResult{
		Added:    []uint{},
		Skipped:  []uint{},
		Failed:   []uint{},
		Messages: []string{},
	}

	for _, productID := range productIDs {
		req := &AddToWishlistRequest{ProductID: productID}

		_, err := s.AddToWishlist(userID, req)
		if err != nil {
			if fmt.Sprintf("%v", err) == "item already exists in wishlist" {
				result.Skipped = append(result.Skipped, productID)
				result.Messages = append(result.Messages, fmt.Sprintf("Product %d already in wishlist", productID))
			} else {
				result.Failed = append(result.Failed, productID)
				result.Messages = append(result.Messages, fmt.Sprintf("Product %d failed: %v", productID, err))
			}
		} else {
			result.Added = append(result.Added, productID)
		}
	}

	return result, nil
}

// GetWishlistSummary returns wishlist summary
func (s *Service) GetWishlistSummary(userID uint) (*WishlistSummary, error) {
	// Get basic count
	totalItems, err := s.GetWishlistCount(userID)
	if err != nil {
		return nil, err
	}

	if totalItems == 0 {
		return &WishlistSummary{}, nil
	}

	// Get all wishlist items with product details
	wishlistResp, err := s.GetWishlist(userID, 1, int(totalItems), "created_at", "desc")
	if err != nil {
		return nil, err
	}

	return &wishlistResp.Summary, nil
}

// Private helper methods

func (s *Service) loadProductDetails(items []WishlistItemResponse) error {
	for i := range items {
		// Load product details
		var prod product.Product
		err := s.db.Preload("Category").Preload("Brand").
			Where("id = ?", items[i].ProductID).First(&prod).Error
		if err != nil {
			items[i].IsAvailable = false
			continue
		}
		items[i].Product = &prod
		items[i].IsAvailable = prod.IsActive

		// Load variant details if applicable
		if items[i].ProductVariantID != nil {
			var variant product.ProductVariant
			err := s.db.Where("id = ?", *items[i].ProductVariantID).First(&variant).Error
			if err == nil {
				items[i].ProductVariant = &variant
				items[i].IsAvailable = items[i].IsAvailable && variant.IsActive
			}
		}

		// Set current price and check for price changes
		items[i].CurrentPrice = prod.Price
		if items[i].ProductVariant != nil && items[i].ProductVariant.Price > 0 {
			items[i].CurrentPrice = items[i].ProductVariant.Price
		}

		// For price change detection, you could store original price when added
		// For now, we'll assume no price change
		items[i].PriceChanged = false
		items[i].OriginalPrice = items[i].CurrentPrice
	}

	return nil
}

func (s *Service) generateWishlistSummary(userID uint, items []WishlistItemResponse) WishlistSummary {
	summary := WishlistSummary{
		TotalItems: len(items),
	}

	if len(items) == 0 {
		return summary
	}

	var totalValue int64
	recentThreshold := time.Now().AddDate(0, 0, -7) // 7 days ago

	for _, item := range items {
		if item.IsAvailable {
			summary.AvailableItems++
			totalValue += item.CurrentPrice
		} else {
			summary.UnavailableItems++
		}

		if item.AddedAt.After(recentThreshold) {
			summary.RecentlyAdded++
		}
	}

	summary.TotalValue = totalValue
	if summary.AvailableItems > 0 {
		summary.AveragePrice = float64(totalValue) / float64(summary.AvailableItems) / 100
	}

	return summary
}

func (s *Service) buildOrderClause(sortBy, sortOrder string) string {
	validSortFields := map[string]bool{
		"created_at": true,
		"added_at":   true,
		"product_id": true,
	}

	if !validSortFields[sortBy] {
		sortBy = "added_at"
	}

	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	return fmt.Sprintf("%s %s", sortBy, sortOrder)
}
