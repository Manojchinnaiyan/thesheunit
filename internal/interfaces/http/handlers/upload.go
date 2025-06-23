// internal/interfaces/http/handlers/upload.go
package handlers

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/your-org/ecommerce-backend/internal/config"
	"github.com/your-org/ecommerce-backend/internal/domain/upload"
	"github.com/your-org/ecommerce-backend/internal/interfaces/http/middleware"
	"gorm.io/gorm"
)

// UploadHandler handles file upload endpoints
type UploadHandler struct {
	uploadService *upload.Service
	config        *config.Config
}

// NewUploadHandler creates a new upload handler
func NewUploadHandler(db *gorm.DB, cfg *config.Config) *UploadHandler {
	return &UploadHandler{
		uploadService: upload.NewService(db, cfg),
		config:        cfg,
	}
}

// UploadImage handles POST /admin/uploads/image
func (h *UploadHandler) UploadImage(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// Parse multipart form
	err := c.Request.ParseMultipartForm(h.config.Upload.MaxSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to parse upload form",
		})
		return
	}

	// Get file from form
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No image file provided",
		})
		return
	}
	defer file.Close()

	// Get additional metadata
	category := c.PostForm("category")       // product, category, brand, user, etc.
	description := c.PostForm("description") // Optional description
	altText := c.PostForm("alt_text")        // For accessibility
	tags := c.PostForm("tags")               // Comma-separated tags

	// Create upload request
	req := &upload.ImageUploadRequest{
		File:        file,
		Header:      header,
		Category:    category,
		Description: description,
		AltText:     altText,
		Tags:        tags,
		UploadedBy:  userID,
	}

	// Upload the image
	uploadedImage, err := h.uploadService.UploadImage(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Image uploaded successfully",
		"data":    uploadedImage,
	})
}

// UploadMultipleImages handles POST /admin/uploads/bulk-upload
func (h *UploadHandler) UploadMultipleImages(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// Parse multipart form
	err := c.Request.ParseMultipartForm(h.config.Upload.MaxSize * 10) // Allow larger total size for bulk
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to parse upload form",
		})
		return
	}

	// Get files from form
	form := c.Request.MultipartForm
	files := form.File["images"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No image files provided",
		})
		return
	}

	// Limit number of files
	maxFiles := 20
	if len(files) > maxFiles {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Too many files. Maximum %d files allowed", maxFiles),
		})
		return
	}

	// Get common metadata
	category := c.PostForm("category")
	description := c.PostForm("description")

	// Create bulk upload request
	req := &upload.BulkUploadRequest{
		Files:       files,
		Category:    category,
		Description: description,
		UploadedBy:  userID,
	}

	// Upload the images
	result, err := h.uploadService.BulkUploadImages(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Bulk upload completed",
		"data":    result,
	})
}

// DeleteImage handles DELETE /admin/uploads/image/:id
func (h *UploadHandler) DeleteImage(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	imageIDParam := c.Param("id")
	imageID, err := strconv.ParseUint(imageIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid image ID",
		})
		return
	}

	// Check if image is in use (optional force delete)
	force := c.Query("force") == "true"

	err = h.uploadService.DeleteImage(uint(imageID), userID, force)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Image deleted successfully",
	})
}

// GetImages handles GET /admin/uploads/images
func (h *UploadHandler) GetImages(c *gin.Context) {
	// Parse query parameters
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	category := c.Query("category")
	search := c.Query("search")
	sortBy := c.Query("sort_by")
	if sortBy == "" {
		sortBy = "created_at"
	}

	sortOrder := c.Query("sort_order")
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	req := &upload.ImageListRequest{
		Page:      page,
		Limit:     limit,
		Category:  category,
		Search:    search,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}

	response, err := h.uploadService.GetImages(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve images",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Images retrieved successfully",
		"data":    response,
	})
}

// GetImage handles GET /admin/uploads/image/:id
func (h *UploadHandler) GetImage(c *gin.Context) {
	imageIDParam := c.Param("id")
	imageID, err := strconv.ParseUint(imageIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid image ID",
		})
		return
	}

	image, err := h.uploadService.GetImage(uint(imageID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Image retrieved successfully",
		"data":    image,
	})
}

// UpdateImage handles PUT /admin/uploads/image/:id
func (h *UploadHandler) UpdateImage(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	imageIDParam := c.Param("id")
	imageID, err := strconv.ParseUint(imageIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid image ID",
		})
		return
	}

	var req upload.ImageUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	image, err := h.uploadService.UpdateImage(uint(imageID), userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Image updated successfully",
		"data":    image,
	})
}

// GetUploadStats handles GET /admin/uploads/stats
func (h *UploadHandler) GetUploadStats(c *gin.Context) {
	stats, err := h.uploadService.GetUploadStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve upload statistics",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Upload statistics retrieved successfully",
		"data":    stats,
	})
}

// OptimizeImage handles POST /admin/uploads/image/:id/optimize
func (h *UploadHandler) OptimizeImage(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	imageIDParam := c.Param("id")
	imageID, err := strconv.ParseUint(imageIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid image ID",
		})
		return
	}

	var req struct {
		Quality int `json:"quality" binding:"min=1,max=100"`
		Width   int `json:"width,omitempty"`
		Height  int `json:"height,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	optimizeReq := &upload.ImageOptimizeRequest{
		Quality: req.Quality,
		Width:   req.Width,
		Height:  req.Height,
	}

	result, err := h.uploadService.OptimizeImage(uint(imageID), userID, optimizeReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Image optimized successfully",
		"data":    result,
	})
}

// ServeFile handles GET /uploads/:filename for serving uploaded files
func (h *UploadHandler) ServeFile(c *gin.Context) {
	filename := c.Param("filename")
	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Filename is required",
		})
		return
	}

	// Validate filename to prevent directory traversal
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid filename",
		})
		return
	}

	// Get file info from database
	image, err := h.uploadService.GetImageByFilename(filename)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "File not found",
		})
		return
	}

	// Serve the file
	filePath := filepath.Join(h.config.External.Storage.LocalPath, image.Path)

	// Set appropriate headers
	c.Header("Content-Type", image.MimeType)
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", image.OriginalName))
	c.Header("Cache-Control", "public, max-age=31536000") // 1 year cache

	c.File(filePath)
}

// GetUploadConfig handles GET /admin/uploads/config
func (h *UploadHandler) GetUploadConfig(c *gin.Context) {
	config := gin.H{
		"max_file_size":        h.config.Upload.MaxSize,
		"allowed_extensions":   h.config.Upload.AllowedExtensions,
		"max_image_width":      h.config.Upload.ImageMaxWidth,
		"max_image_height":     h.config.Upload.ImageMaxHeight,
		"thumbnail_width":      h.config.Upload.ThumbnailWidth,
		"thumbnail_height":     h.config.Upload.ThumbnailHeight,
		"storage_provider":     h.config.External.Storage.Provider,
		"supported_categories": []string{"product", "category", "brand", "user", "general"},
		"max_bulk_files":       20,
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Upload configuration retrieved successfully",
		"data":    config,
	})
}
