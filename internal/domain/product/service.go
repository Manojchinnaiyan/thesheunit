// internal/domain/product/service.go
package product

import (
	"fmt"
	"strings"
	"time"

	"github.com/your-org/ecommerce-backend/internal/config"
	"gorm.io/gorm"
)

// Service handles product business logic
type Service struct {
	db     *gorm.DB
	config *config.Config
}

// NewService creates a new product service
func NewService(db *gorm.DB, cfg *config.Config) *Service {
	return &Service{
		db:     db,
		config: cfg,
	}
}

// ProductListRequest represents product list query parameters
type ProductListRequest struct {
	Page       int    `form:"page,default=1"`
	Limit      int    `form:"limit,default=20"`
	CategoryID uint   `form:"category_id"`
	BrandID    uint   `form:"brand_id"`
	Search     string `form:"search"`
	SortBy     string `form:"sort_by,default=created_at"`
	SortOrder  string `form:"sort_order,default=desc"`
	MinPrice   int64  `form:"min_price"`
	MaxPrice   int64  `form:"max_price"`
	IsActive   *bool  `form:"is_active"`
	IsFeatured *bool  `form:"is_featured"`
}

// ProductCreateRequest represents product creation data
type ProductCreateRequest struct {
	SKU               string  `json:"sku" binding:"required"`
	Name              string  `json:"name" binding:"required"`
	Description       string  `json:"description"`
	ShortDesc         string  `json:"short_description"`
	Price             int64   `json:"price" binding:"required"`
	ComparePrice      int64   `json:"compare_price"`
	CostPrice         int64   `json:"cost_price"`
	CategoryID        uint    `json:"category_id" binding:"required"`
	BrandID           *uint   `json:"brand_id"`
	Weight            float64 `json:"weight"`
	Dimensions        string  `json:"dimensions"`
	IsActive          bool    `json:"is_active"`
	IsFeatured        bool    `json:"is_featured"`
	IsDigital         bool    `json:"is_digital"`
	RequiresShipping  bool    `json:"requires_shipping"`
	TrackQuantity     bool    `json:"track_quantity"`
	Quantity          int     `json:"quantity"`
	LowStockThreshold int     `json:"low_stock_threshold"`
	SeoTitle          string  `json:"seo_title"`
	SeoDescription    string  `json:"seo_description"`
	Tags              string  `json:"tags"`
}

// ProductUpdateRequest represents product update data
type ProductUpdateRequest struct {
	Name              *string  `json:"name"`
	Description       *string  `json:"description"`
	ShortDesc         *string  `json:"short_description"`
	Price             *int64   `json:"price"`
	ComparePrice      *int64   `json:"compare_price"`
	CostPrice         *int64   `json:"cost_price"`
	CategoryID        *uint    `json:"category_id"`
	BrandID           *uint    `json:"brand_id"`
	Weight            *float64 `json:"weight"`
	Dimensions        *string  `json:"dimensions"`
	IsActive          *bool    `json:"is_active"`
	IsFeatured        *bool    `json:"is_featured"`
	IsDigital         *bool    `json:"is_digital"`
	RequiresShipping  *bool    `json:"requires_shipping"`
	TrackQuantity     *bool    `json:"track_quantity"`
	Quantity          *int     `json:"quantity"`
	LowStockThreshold *int     `json:"low_stock_threshold"`
	SeoTitle          *string  `json:"seo_title"`
	SeoDescription    *string  `json:"seo_description"`
	Tags              *string  `json:"tags"`
}

// ProductResponse represents product response with pagination
type ProductResponse struct {
	Products   []Product  `json:"products"`
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

// GetProducts retrieves products with filtering and pagination
func (s *Service) GetProducts(req *ProductListRequest) (*ProductResponse, error) {
	var products []Product
	var total int64

	// Build query
	query := s.db.Model(&Product{}).
		Preload("Category").
		Preload("Brand").
		Preload("Images", func(db *gorm.DB) *gorm.DB {
			return db.Order("is_primary DESC, sort_order ASC, id ASC")
		})

	// Apply filters
	if req.CategoryID > 0 {
		query = query.Where("category_id = ?", req.CategoryID)
	}

	if req.BrandID > 0 {
		query = query.Where("brand_id = ?", req.BrandID)
	}

	if req.Search != "" {
		search := "%" + strings.ToLower(req.Search) + "%"
		query = query.Where("LOWER(name) LIKE ? OR LOWER(description) LIKE ? OR LOWER(tags) LIKE ?", search, search, search)
	}

	if req.MinPrice > 0 {
		query = query.Where("price >= ?", req.MinPrice)
	}

	if req.MaxPrice > 0 {
		query = query.Where("price <= ?", req.MaxPrice)
	}

	if req.IsActive != nil {
		query = query.Where("is_active = ?", *req.IsActive)
	}

	if req.IsFeatured != nil {
		query = query.Where("is_featured = ?", *req.IsFeatured)
	}

	// Count total records
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count products: %w", err)
	}

	// Apply sorting
	orderClause := s.buildOrderClause(req.SortBy, req.SortOrder)
	query = query.Order(orderClause)

	// Apply pagination
	offset := (req.Page - 1) * req.Limit
	if err := query.Offset(offset).Limit(req.Limit).Find(&products).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve products: %w", err)
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

	return &ProductResponse{
		Products:   products,
		Pagination: pagination,
	}, nil
}

// GetProduct retrieves a single product by ID
func (s *Service) GetProduct(id uint) (*Product, error) {
	var product Product
	result := s.db.
		Preload("Category").
		Preload("Brand").
		Preload("Images", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC, id ASC")
		}).
		Preload("Variants", "is_active = ?", true).
		Where("id = ?", id).
		First(&product)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("product not found")
		}
		return nil, fmt.Errorf("failed to retrieve product: %w", result.Error)
	}

	return &product, nil
}

// GetProductBySlug retrieves a single product by slug
func (s *Service) GetProductBySlug(slug string) (*Product, error) {
	var product Product
	result := s.db.
		Preload("Category").
		Preload("Brand").
		Preload("Images", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC, id ASC")
		}).
		Preload("Variants", "is_active = ?", true).
		Where("slug = ? AND is_active = ?", slug, true).
		First(&product)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("product not found")
		}
		return nil, fmt.Errorf("failed to retrieve product: %w", result.Error)
	}

	return &product, nil
}

// CreateProduct creates a new product
func (s *Service) CreateProduct(req *ProductCreateRequest) (*Product, error) {
	// Check if SKU already exists
	var existing Product
	if result := s.db.Where("sku = ?", req.SKU).First(&existing); result.Error == nil {
		return nil, fmt.Errorf("product with SKU %s already exists", req.SKU)
	}

	// Generate slug from name
	slug := s.generateSlug(req.Name)

	// Create product
	product := Product{
		SKU:               req.SKU,
		Name:              req.Name,
		Slug:              slug,
		Description:       req.Description,
		ShortDesc:         req.ShortDesc,
		Price:             req.Price,
		ComparePrice:      req.ComparePrice,
		CostPrice:         req.CostPrice,
		CategoryID:        req.CategoryID,
		BrandID:           req.BrandID,
		Weight:            req.Weight,
		Dimensions:        req.Dimensions,
		IsActive:          req.IsActive,
		IsFeatured:        req.IsFeatured,
		IsDigital:         req.IsDigital,
		RequiresShipping:  req.RequiresShipping,
		TrackQuantity:     req.TrackQuantity,
		Quantity:          req.Quantity,
		LowStockThreshold: req.LowStockThreshold,
		SeoTitle:          req.SeoTitle,
		SeoDescription:    req.SeoDescription,
		Tags:              req.Tags,
	}

	if err := s.db.Create(&product).Error; err != nil {
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	// Load relationships
	s.db.Preload("Category").Preload("Brand").First(&product, product.ID)

	return &product, nil
}

// UpdateProduct updates an existing product
func (s *Service) UpdateProduct(id uint, req *ProductUpdateRequest) (*Product, error) {
	var product Product
	result := s.db.Where("id = ?", id).First(&product)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("product not found")
		}
		return nil, fmt.Errorf("failed to find product: %w", result.Error)
	}

	// Update fields
	updates := make(map[string]interface{})

	if req.Name != nil {
		updates["name"] = *req.Name
		updates["slug"] = s.generateSlug(*req.Name)
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.ShortDesc != nil {
		updates["short_desc"] = *req.ShortDesc
	}
	if req.Price != nil {
		updates["price"] = *req.Price
	}
	if req.ComparePrice != nil {
		updates["compare_price"] = *req.ComparePrice
	}
	if req.CostPrice != nil {
		updates["cost_price"] = *req.CostPrice
	}
	if req.CategoryID != nil {
		updates["category_id"] = *req.CategoryID
	}
	if req.BrandID != nil {
		updates["brand_id"] = *req.BrandID
	}
	if req.Weight != nil {
		updates["weight"] = *req.Weight
	}
	if req.Dimensions != nil {
		updates["dimensions"] = *req.Dimensions
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.IsFeatured != nil {
		updates["is_featured"] = *req.IsFeatured
	}
	if req.IsDigital != nil {
		updates["is_digital"] = *req.IsDigital
	}
	if req.RequiresShipping != nil {
		updates["requires_shipping"] = *req.RequiresShipping
	}
	if req.TrackQuantity != nil {
		updates["track_quantity"] = *req.TrackQuantity
	}
	if req.Quantity != nil {
		updates["quantity"] = *req.Quantity
	}
	if req.LowStockThreshold != nil {
		updates["low_stock_threshold"] = *req.LowStockThreshold
	}
	if req.SeoTitle != nil {
		updates["seo_title"] = *req.SeoTitle
	}
	if req.SeoDescription != nil {
		updates["seo_description"] = *req.SeoDescription
	}
	if req.Tags != nil {
		updates["tags"] = *req.Tags
	}

	if err := s.db.Model(&product).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update product: %w", err)
	}

	// Load updated product with relationships
	s.db.Preload("Category").Preload("Brand").First(&product, product.ID)

	return &product, nil
}

// DeleteProduct soft deletes a product
func (s *Service) DeleteProduct(id uint) error {
	result := s.db.Where("id = ?", id).Delete(&Product{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete product: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("product not found")
	}
	return nil
}

// UpdateInventory updates product inventory
func (s *Service) UpdateInventory(productID uint, quantity int) error {
	result := s.db.Model(&Product{}).
		Where("id = ? AND track_quantity = ?", productID, true).
		Update("quantity", quantity)

	if result.Error != nil {
		return fmt.Errorf("failed to update inventory: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("product not found or inventory tracking disabled")
	}
	return nil
}

// buildOrderClause builds ORDER BY clause for sorting
func (s *Service) buildOrderClause(sortBy, sortOrder string) string {
	validSortFields := map[string]bool{
		"name":       true,
		"price":      true,
		"created_at": true,
		"updated_at": true,
		"quantity":   true,
	}

	if !validSortFields[sortBy] {
		sortBy = "created_at"
	}

	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	return fmt.Sprintf("%s %s", sortBy, sortOrder)
}

// generateSlug generates URL-friendly slug from name
func (s *Service) generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")

	// Remove special characters (basic implementation)
	// In production, you might want to use a more robust slug generation library

	return slug + "-" + fmt.Sprintf("%d", time.Now().Unix())
}
