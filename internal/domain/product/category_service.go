// internal/domain/product/category_service.go
package product

import (
	"fmt"
	"strings"
	"time"

	"github.com/your-org/ecommerce-backend/internal/config"
	"gorm.io/gorm"
)

// CategoryService handles category business logic
type CategoryService struct {
	db     *gorm.DB
	config *config.Config
}

// NewCategoryService creates a new category service
func NewCategoryService(db *gorm.DB, cfg *config.Config) *CategoryService {
	return &CategoryService{
		db:     db,
		config: cfg,
	}
}

// CategoryCreateRequest represents category creation data
type CategoryCreateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Image       string `json:"image"`
	ParentID    *uint  `json:"parent_id"`
	SortOrder   int    `json:"sort_order"`
	IsActive    bool   `json:"is_active"`
}

// CategoryUpdateRequest represents category update data
type CategoryUpdateRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Image       *string `json:"image"`
	ParentID    *uint   `json:"parent_id"`
	SortOrder   *int    `json:"sort_order"`
	IsActive    *bool   `json:"is_active"`
}

// CategoryWithProductCount represents category with product count
type CategoryWithProductCount struct {
	Category
	ProductCount int64 `json:"product_count"`
}

// CategoryTree represents hierarchical category structure
type CategoryTree struct {
	Category
	Children []CategoryTree `json:"children,omitempty"`
}

// GetCategories retrieves all categories with optional filtering
func (s *CategoryService) GetCategories(includeInactive bool) ([]Category, error) {
	var categories []Category

	query := s.db.Model(&Category{}).
		Preload("Parent").
		Order("sort_order ASC, name ASC")

	if !includeInactive {
		query = query.Where("is_active = ?", true)
	}

	if err := query.Find(&categories).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve categories: %w", err)
	}

	return categories, nil
}

// GetCategoryTree retrieves categories in hierarchical tree structure
func (s *CategoryService) GetCategoryTree(includeInactive bool) ([]CategoryTree, error) {
	categories, err := s.GetCategories(includeInactive)
	if err != nil {
		return nil, err
	}

	// Build tree structure
	categoryMap := make(map[uint]*CategoryTree)
	var rootCategories []CategoryTree

	// Create tree nodes
	for _, cat := range categories {
		categoryMap[cat.ID] = &CategoryTree{
			Category: cat,
			Children: []CategoryTree{},
		}
	}

	// Build hierarchy
	for _, cat := range categories {
		if cat.ParentID == nil {
			// Root category
			rootCategories = append(rootCategories, *categoryMap[cat.ID])
		} else {
			// Child category
			if parent, exists := categoryMap[*cat.ParentID]; exists {
				parent.Children = append(parent.Children, *categoryMap[cat.ID])
			}
		}
	}

	return rootCategories, nil
}

// GetCategoriesWithProductCount retrieves categories with product counts
func (s *CategoryService) GetCategoriesWithProductCount(includeInactive bool) ([]CategoryWithProductCount, error) {
	var categories []Category

	query := s.db.Model(&Category{}).Order("sort_order ASC, name ASC")
	if !includeInactive {
		query = query.Where("is_active = ?", true)
	}

	if err := query.Find(&categories).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve categories: %w", err)
	}

	var result []CategoryWithProductCount

	for _, cat := range categories {
		var productCount int64

		// Count products in this category (only active products for public view)
		countQuery := s.db.Model(&Product{}).Where("category_id = ?", cat.ID)
		if !includeInactive {
			countQuery = countQuery.Where("is_active = ?", true)
		}

		countQuery.Count(&productCount)

		result = append(result, CategoryWithProductCount{
			Category:     cat,
			ProductCount: productCount,
		})
	}

	return result, nil
}

// GetCategory retrieves a single category by ID
func (s *CategoryService) GetCategory(id uint) (*Category, error) {
	var category Category
	result := s.db.
		Preload("Parent").
		Preload("Children", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_active = ?", true).Order("sort_order ASC, name ASC")
		}).
		Where("id = ?", id).
		First(&category)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("category not found")
		}
		return nil, fmt.Errorf("failed to retrieve category: %w", result.Error)
	}

	return &category, nil
}

// GetCategoryBySlug retrieves a single category by slug
func (s *CategoryService) GetCategoryBySlug(slug string) (*Category, error) {
	var category Category
	result := s.db.
		Preload("Parent").
		Preload("Children", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_active = ?", true).Order("sort_order ASC, name ASC")
		}).
		Where("slug = ? AND is_active = ?", slug, true).
		First(&category)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("category not found")
		}
		return nil, fmt.Errorf("failed to retrieve category: %w", result.Error)
	}

	return &category, nil
}

// CreateCategory creates a new category
func (s *CategoryService) CreateCategory(req *CategoryCreateRequest) (*Category, error) {
	// Validate parent category if specified
	if req.ParentID != nil {
		var parent Category
		if result := s.db.Where("id = ?", *req.ParentID).First(&parent); result.Error != nil {
			return nil, fmt.Errorf("parent category not found")
		}
	}

	// Generate slug from name
	slug := s.generateSlug(req.Name)

	// Check if slug already exists
	var existing Category
	if result := s.db.Where("slug = ?", slug).First(&existing); result.Error == nil {
		return nil, fmt.Errorf("category with similar name already exists")
	}

	// Create category
	category := Category{
		Name:        req.Name,
		Slug:        slug,
		Description: req.Description,
		Image:       req.Image,
		ParentID:    req.ParentID,
		SortOrder:   req.SortOrder,
		IsActive:    req.IsActive,
	}

	if err := s.db.Create(&category).Error; err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	// Load relationships
	s.db.Preload("Parent").First(&category, category.ID)

	return &category, nil
}

// UpdateCategory updates an existing category
func (s *CategoryService) UpdateCategory(id uint, req *CategoryUpdateRequest) (*Category, error) {
	var category Category
	result := s.db.Where("id = ?", id).First(&category)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("category not found")
		}
		return nil, fmt.Errorf("failed to find category: %w", result.Error)
	}

	// Validate parent category if being updated
	if req.ParentID != nil {
		// Prevent circular references
		if *req.ParentID == id {
			return nil, fmt.Errorf("category cannot be its own parent")
		}

		// Check if parent exists
		var parent Category
		if result := s.db.Where("id = ?", *req.ParentID).First(&parent); result.Error != nil {
			return nil, fmt.Errorf("parent category not found")
		}

		// Check for circular reference in ancestry
		if s.isCircularReference(id, *req.ParentID) {
			return nil, fmt.Errorf("circular reference detected")
		}
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
	if req.Image != nil {
		updates["image"] = *req.Image
	}
	if req.ParentID != nil {
		updates["parent_id"] = *req.ParentID
	}
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if err := s.db.Model(&category).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update category: %w", err)
	}

	// Load updated category with relationships
	s.db.Preload("Parent").First(&category, category.ID)

	return &category, nil
}

// DeleteCategory soft deletes a category
func (s *CategoryService) DeleteCategory(id uint) error {
	// Check if category has products
	var productCount int64
	s.db.Model(&Product{}).Where("category_id = ?", id).Count(&productCount)
	if productCount > 0 {
		return fmt.Errorf("cannot delete category with existing products")
	}

	// Check if category has children
	var childCount int64
	s.db.Model(&Category{}).Where("parent_id = ?", id).Count(&childCount)
	if childCount > 0 {
		return fmt.Errorf("cannot delete category with subcategories")
	}

	result := s.db.Where("id = ?", id).Delete(&Category{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete category: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("category not found")
	}
	return nil
}

// GetRootCategories retrieves only root categories (no parent)
func (s *CategoryService) GetRootCategories(includeInactive bool) ([]Category, error) {
	var categories []Category

	query := s.db.Model(&Category{}).
		Where("parent_id IS NULL").
		Order("sort_order ASC, name ASC")

	if !includeInactive {
		query = query.Where("is_active = ?", true)
	}

	if err := query.Find(&categories).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve root categories: %w", err)
	}

	return categories, nil
}

// GetSubcategories retrieves subcategories of a parent category
func (s *CategoryService) GetSubcategories(parentID uint, includeInactive bool) ([]Category, error) {
	var categories []Category

	query := s.db.Model(&Category{}).
		Where("parent_id = ?", parentID).
		Order("sort_order ASC, name ASC")

	if !includeInactive {
		query = query.Where("is_active = ?", true)
	}

	if err := query.Find(&categories).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve subcategories: %w", err)
	}

	return categories, nil
}

// isCircularReference checks if making parentID the parent of categoryID would create a circular reference
func (s *CategoryService) isCircularReference(categoryID, parentID uint) bool {
	// Get all ancestors of the parentID
	ancestors := s.getAncestors(parentID)

	// Check if categoryID is in the ancestors
	for _, ancestor := range ancestors {
		if ancestor == categoryID {
			return true
		}
	}

	return false
}

// getAncestors returns all ancestor IDs of a category
func (s *CategoryService) getAncestors(categoryID uint) []uint {
	var ancestors []uint
	currentID := categoryID

	for {
		var category Category
		result := s.db.Select("parent_id").Where("id = ?", currentID).First(&category)
		if result.Error != nil || category.ParentID == nil {
			break
		}

		ancestors = append(ancestors, *category.ParentID)
		currentID = *category.ParentID
	}

	return ancestors
}

// generateSlug generates URL-friendly slug from name
func (s *CategoryService) generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")

	// Remove special characters (basic implementation)
	// In production, you might want to use a more robust slug generation library

	return slug + "-" + fmt.Sprintf("%d", time.Now().Unix())
}
