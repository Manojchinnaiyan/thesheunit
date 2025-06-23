// internal/interfaces/http/handlers/product.go
package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/your-org/ecommerce-backend/internal/config"
	"github.com/your-org/ecommerce-backend/internal/domain/product"
	"gorm.io/gorm"
)

// ProductHandler handles product endpoints
type ProductHandler struct {
	productService *product.Service
	config         *config.Config
}

// NewProductHandler creates a new product handler
func NewProductHandler(db *gorm.DB, cfg *config.Config) *ProductHandler {
	return &ProductHandler{
		productService: product.NewService(db, cfg),
		config:         cfg,
	}
}

// GetProducts handles GET /products
func (h *ProductHandler) GetProducts(c *gin.Context) {
	var req product.ProductListRequest

	// Bind query parameters
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid query parameters",
			"details": err.Error(),
		})
		return
	}

	// Set default values
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}

	// For public endpoint, only show active products
	isActive := true
	req.IsActive = &isActive

	response, err := h.productService.GetProducts(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve products",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Products retrieved successfully",
		"data":    response,
	})
}

// GetProduct handles GET /products/:id
func (h *ProductHandler) GetProduct(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid product ID",
		})
		return
	}

	product, err := h.productService.GetProduct(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	// For public endpoint, only show active products
	if !product.IsActive {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Product not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Product retrieved successfully",
		"data":    product,
	})
}

// GetProductBySlug handles GET /products/slug/:slug
func (h *ProductHandler) GetProductBySlug(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Product slug is required",
		})
		return
	}

	product, err := h.productService.GetProductBySlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Product retrieved successfully",
		"data":    product,
	})
}

// SearchProducts handles GET /products/search
func (h *ProductHandler) SearchProducts(c *gin.Context) {
	var req product.ProductListRequest

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid query parameters",
			"details": err.Error(),
		})
		return
	}

	// Set default values
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}

	// For search, only show active products
	isActive := true
	req.IsActive = &isActive

	// Require search term
	if req.Search == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Search term is required",
		})
		return
	}

	response, err := h.productService.GetProducts(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to search products",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Products found",
		"data":    response,
	})
}

// --- ADMIN ENDPOINTS ---

// AdminGetProducts handles GET /admin/products
func (h *ProductHandler) AdminGetProducts(c *gin.Context) {
	var req product.ProductListRequest

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid query parameters",
			"details": err.Error(),
		})
		return
	}

	// Set default values
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}

	// Admin can see all products (don't filter by is_active)

	response, err := h.productService.GetProducts(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve products",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Products retrieved successfully",
		"data":    response,
	})
}

// AdminGetProduct handles GET /admin/products/:id
func (h *ProductHandler) AdminGetProduct(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid product ID",
		})
		return
	}

	product, err := h.productService.GetProduct(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Product retrieved successfully",
		"data":    product,
	})
}

// AdminCreateProduct handles POST /admin/products
func (h *ProductHandler) AdminCreateProduct(c *gin.Context) {
	var req product.ProductCreateRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	product, err := h.productService.CreateProduct(&req)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Product created successfully",
		"data":    product,
	})
}

// AdminUpdateProduct handles PUT /admin/products/:id
func (h *ProductHandler) AdminUpdateProduct(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid product ID",
		})
		return
	}

	var req product.ProductUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	product, err := h.productService.UpdateProduct(uint(id), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Product updated successfully",
		"data":    product,
	})
}

// AdminDeleteProduct handles DELETE /admin/products/:id
func (h *ProductHandler) AdminDeleteProduct(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid product ID",
		})
		return
	}

	err = h.productService.DeleteProduct(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Product deleted successfully",
	})
}

// AdminUpdateInventory handles PUT /admin/products/:id/inventory
func (h *ProductHandler) AdminUpdateInventory(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid product ID",
		})
		return
	}

	var req struct {
		Quantity int `json:"quantity" binding:"required,min=0"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	err = h.productService.UpdateInventory(uint(id), req.Quantity)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Inventory updated successfully",
	})
}
