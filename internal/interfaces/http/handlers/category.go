// internal/interfaces/http/handlers/category.go
package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/your-org/ecommerce-backend/internal/config"
	"github.com/your-org/ecommerce-backend/internal/domain/product"
	"gorm.io/gorm"
)

// CategoryHandler handles category endpoints
type CategoryHandler struct {
	categoryService *product.CategoryService
	config          *config.Config
}

// NewCategoryHandler creates a new category handler
func NewCategoryHandler(db *gorm.DB, cfg *config.Config) *CategoryHandler {
	return &CategoryHandler{
		categoryService: product.NewCategoryService(db, cfg),
		config:          cfg,
	}
}

// GetCategories handles GET /products/categories
func (h *CategoryHandler) GetCategories(c *gin.Context) {
	// Query parameter to include product counts
	includeCounts := c.Query("include_counts") == "true"

	if includeCounts {
		categories, err := h.categoryService.GetCategoriesWithProductCount(false)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve categories",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Categories retrieved successfully",
			"data":    categories,
		})
		return
	}

	categories, err := h.categoryService.GetCategories(false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve categories",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Categories retrieved successfully",
		"data":    categories,
	})
}

// GetCategoryTree handles GET /products/categories/tree
func (h *CategoryHandler) GetCategoryTree(c *gin.Context) {
	tree, err := h.categoryService.GetCategoryTree(false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve category tree",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Category tree retrieved successfully",
		"data":    tree,
	})
}

// GetRootCategories handles GET /products/categories/root
func (h *CategoryHandler) GetRootCategories(c *gin.Context) {
	categories, err := h.categoryService.GetRootCategories(false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve root categories",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Root categories retrieved successfully",
		"data":    categories,
	})
}

// GetCategory handles GET /products/categories/:id
func (h *CategoryHandler) GetCategory(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid category ID",
		})
		return
	}

	category, err := h.categoryService.GetCategory(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	// For public endpoint, only show active categories
	if !category.IsActive {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Category not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Category retrieved successfully",
		"data":    category,
	})
}

// GetCategoryBySlug handles GET /products/categories/slug/:slug
func (h *CategoryHandler) GetCategoryBySlug(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Category slug is required",
		})
		return
	}

	category, err := h.categoryService.GetCategoryBySlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Category retrieved successfully",
		"data":    category,
	})
}

// GetSubcategories handles GET /products/categories/:id/subcategories
func (h *CategoryHandler) GetSubcategories(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid category ID",
		})
		return
	}

	subcategories, err := h.categoryService.GetSubcategories(uint(id), false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve subcategories",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Subcategories retrieved successfully",
		"data":    subcategories,
	})
}

// --- ADMIN ENDPOINTS ---

// AdminGetCategories handles GET /admin/categories
func (h *CategoryHandler) AdminGetCategories(c *gin.Context) {
	// Query parameter to include product counts
	includeCounts := c.Query("include_counts") == "true"

	if includeCounts {
		categories, err := h.categoryService.GetCategoriesWithProductCount(true)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve categories",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Categories retrieved successfully",
			"data":    categories,
		})
		return
	}

	categories, err := h.categoryService.GetCategories(true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve categories",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Categories retrieved successfully",
		"data":    categories,
	})
}

// AdminGetCategoryTree handles GET /admin/categories/tree
func (h *CategoryHandler) AdminGetCategoryTree(c *gin.Context) {
	tree, err := h.categoryService.GetCategoryTree(true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve category tree",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Category tree retrieved successfully",
		"data":    tree,
	})
}

// AdminGetCategory handles GET /admin/categories/:id
func (h *CategoryHandler) AdminGetCategory(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid category ID",
		})
		return
	}

	category, err := h.categoryService.GetCategory(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Category retrieved successfully",
		"data":    category,
	})
}

// AdminCreateCategory handles POST /admin/categories
func (h *CategoryHandler) AdminCreateCategory(c *gin.Context) {
	var req product.CategoryCreateRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	category, err := h.categoryService.CreateCategory(&req)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Category created successfully",
		"data":    category,
	})
}

// AdminUpdateCategory handles PUT /admin/categories/:id
func (h *CategoryHandler) AdminUpdateCategory(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid category ID",
		})
		return
	}

	var req product.CategoryUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	category, err := h.categoryService.UpdateCategory(uint(id), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Category updated successfully",
		"data":    category,
	})
}

// AdminDeleteCategory handles DELETE /admin/categories/:id
func (h *CategoryHandler) AdminDeleteCategory(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid category ID",
		})
		return
	}

	err = h.categoryService.DeleteCategory(uint(id))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Category deleted successfully",
	})
}
