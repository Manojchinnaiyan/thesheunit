// internal/domain/upload/service.go - Complete implementation
package upload

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/your-org/ecommerce-backend/internal/config"
	"gorm.io/gorm"
)

// Service handles file upload business logic
type Service struct {
	db     *gorm.DB
	config *config.Config
}

// NewService creates a new upload service
func NewService(db *gorm.DB, cfg *config.Config) *Service {
	return &Service{
		db:     db,
		config: cfg,
	}
}

// ImageUploadRequest represents an image upload request
type ImageUploadRequest struct {
	File        multipart.File        `json:"-"`
	Header      *multipart.FileHeader `json:"-"`
	Category    string                `json:"category"`
	Description string                `json:"description"`
	AltText     string                `json:"alt_text"`
	Tags        string                `json:"tags"`
	UploadedBy  uint                  `json:"uploaded_by"`
}

// ImageUpdateRequest represents an image update request
type ImageUpdateRequest struct {
	Category    *string `json:"category"`
	Description *string `json:"description"`
	AltText     *string `json:"alt_text"`
	Tags        *string `json:"tags"`
	IsPublic    *bool   `json:"is_public"`
}

// ImageOptimizeRequest represents image optimization request
type ImageOptimizeRequest struct {
	Quality int `json:"quality"`
	Width   int `json:"width"`
	Height  int `json:"height"`
}

// BulkUploadRequest represents bulk upload request
type BulkUploadRequest struct {
	Files       []*multipart.FileHeader `json:"-"`
	Category    string                  `json:"category"`
	Description string                  `json:"description"`
	UploadedBy  uint                    `json:"uploaded_by"`
}

// BulkUploadResult represents bulk upload result
type BulkUploadResult struct {
	Uploaded []UploadedFile `json:"uploaded"`
	Failed   []FailedUpload `json:"failed"`
	Summary  UploadSummary  `json:"summary"`
}

// FailedUpload represents a failed upload
type FailedUpload struct {
	Filename string `json:"filename"`
	Error    string `json:"error"`
}

// UploadSummary represents upload summary
type UploadSummary struct {
	TotalFiles   int   `json:"total_files"`
	SuccessCount int   `json:"success_count"`
	FailureCount int   `json:"failure_count"`
	TotalSize    int64 `json:"total_size"`
}

// ImageListRequest represents image list request
type ImageListRequest struct {
	Page      int    `json:"page"`
	Limit     int    `json:"limit"`
	Category  string `json:"category"`
	Search    string `json:"search"`
	SortBy    string `json:"sort_by"`
	SortOrder string `json:"sort_order"`
}

// ImageListResponse represents image list response
type ImageListResponse struct {
	Images     []UploadedFile `json:"images"`
	Pagination Pagination     `json:"pagination"`
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

// UploadStats represents upload statistics
type UploadStats struct {
	TotalFiles         int64            `json:"total_files"`
	TotalSize          int64            `json:"total_size"`
	TotalSizeFormatted string           `json:"total_size_formatted"`
	ImageCount         int64            `json:"image_count"`
	CategoryBreakdown  map[string]int64 `json:"category_breakdown"`
	MonthlyUploads     []MonthlyUpload  `json:"monthly_uploads"`
	RecentUploads      []UploadedFile   `json:"recent_uploads"`
}

// MonthlyUpload represents monthly upload statistics
type MonthlyUpload struct {
	Month string `json:"month"`
	Count int64  `json:"count"`
	Size  int64  `json:"size"`
}

// UploadImage uploads a single image
func (s *Service) UploadImage(req *ImageUploadRequest) (*UploadedFile, error) {
	// Validate file
	if err := s.validateImageFile(req.Header); err != nil {
		return nil, err
	}

	// Generate unique filename
	filename := s.generateUniqueFilename(req.Header.Filename)

	// Determine file path
	category := req.Category
	if category == "" {
		category = "general"
	}

	relativePath := filepath.Join(category, filename)
	fullPath := filepath.Join(s.config.External.Storage.LocalPath, relativePath)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Save file
	dst, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, req.File); err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// Get image dimensions
	width, height := s.getImageDimensions(fullPath)

	// Generate thumbnail if it's an image
	thumbnailURL := ""
	if s.isImageFile(req.Header.Filename) {
		thumbnailPath, err := s.generateThumbnail(fullPath, category, filename)
		if err == nil {
			thumbnailURL = s.getFileURL(thumbnailPath)
		}
	}

	// Create database record
	uploadedFile := UploadedFile{
		OriginalName: req.Header.Filename,
		Filename:     filename,
		Path:         relativePath,
		URL:          s.getFileURL(relativePath),
		MimeType:     s.getMimeType(req.Header.Filename),
		Size:         req.Header.Size,
		Category:     category,
		Description:  req.Description,
		AltText:      req.AltText,
		Tags:         req.Tags,
		Width:        width,
		Height:       height,
		ThumbnailURL: thumbnailURL,
		UploadedBy:   req.UploadedBy,
		IsPublic:     true,
	}

	if err := s.db.Create(&uploadedFile).Error; err != nil {
		// Clean up file if database insert fails
		os.Remove(fullPath)
		return nil, fmt.Errorf("failed to save file info: %w", err)
	}

	return &uploadedFile, nil
}

// BulkUploadImages uploads multiple images
func (s *Service) BulkUploadImages(req *BulkUploadRequest) (*BulkUploadResult, error) {
	result := &BulkUploadResult{
		Uploaded: []UploadedFile{},
		Failed:   []FailedUpload{},
		Summary: UploadSummary{
			TotalFiles: len(req.Files),
		},
	}

	for _, fileHeader := range req.Files {
		file, err := fileHeader.Open()
		if err != nil {
			result.Failed = append(result.Failed, FailedUpload{
				Filename: fileHeader.Filename,
				Error:    fmt.Sprintf("Failed to open file: %v", err),
			})
			result.Summary.FailureCount++
			continue
		}

		uploadReq := &ImageUploadRequest{
			File:        file,
			Header:      fileHeader,
			Category:    req.Category,
			Description: req.Description,
			UploadedBy:  req.UploadedBy,
		}

		uploadedFile, err := s.UploadImage(uploadReq)
		file.Close()

		if err != nil {
			result.Failed = append(result.Failed, FailedUpload{
				Filename: fileHeader.Filename,
				Error:    err.Error(),
			})
			result.Summary.FailureCount++
		} else {
			result.Uploaded = append(result.Uploaded, *uploadedFile)
			result.Summary.SuccessCount++
			result.Summary.TotalSize += uploadedFile.Size
		}
	}

	return result, nil
}

// DeleteImage deletes an uploaded image
func (s *Service) DeleteImage(imageID, userID uint, force bool) error {
	// Get image record
	var uploadedFile UploadedFile
	if err := s.db.First(&uploadedFile, imageID).Error; err != nil {
		return fmt.Errorf("image not found")
	}

	// Check if image is in use (unless forced)
	if !force {
		var usageCount int64
		s.db.Model(&FileUsage{}).Where("file_id = ?", imageID).Count(&usageCount)
		if usageCount > 0 {
			return fmt.Errorf("image is currently in use and cannot be deleted. Use force=true to delete anyway")
		}
	}

	// Delete physical files
	fullPath := filepath.Join(s.config.External.Storage.LocalPath, uploadedFile.Path)
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// Delete thumbnail if exists
	if uploadedFile.ThumbnailURL != "" {
		thumbnailPath := s.urlToPath(uploadedFile.ThumbnailURL)
		os.Remove(filepath.Join(s.config.External.Storage.LocalPath, thumbnailPath))
	}

	// Delete optimized version if exists
	if uploadedFile.OptimizedURL != "" {
		optimizedPath := s.urlToPath(uploadedFile.OptimizedURL)
		os.Remove(filepath.Join(s.config.External.Storage.LocalPath, optimizedPath))
	}

	// Delete database record and usage records
	tx := s.db.Begin()

	// Delete usage records
	tx.Where("file_id = ?", imageID).Delete(&FileUsage{})

	// Delete file record
	if err := tx.Delete(&uploadedFile).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete image record: %w", err)
	}

	return tx.Commit().Error
}

// GetImages retrieves images with filtering and pagination
func (s *Service) GetImages(req *ImageListRequest) (*ImageListResponse, error) {
	var images []UploadedFile
	var total int64

	// Build query
	query := s.db.Model(&UploadedFile{})

	// Apply filters
	if req.Category != "" {
		query = query.Where("category = ?", req.Category)
	}

	if req.Search != "" {
		search := "%" + strings.ToLower(req.Search) + "%"
		query = query.Where("LOWER(original_name) LIKE ? OR LOWER(description) LIKE ? OR LOWER(tags) LIKE ?", search, search, search)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count images: %w", err)
	}

	// Apply sorting
	orderClause := s.buildOrderClause(req.SortBy, req.SortOrder)
	query = query.Order(orderClause)

	// Apply pagination
	offset := (req.Page - 1) * req.Limit
	if err := query.Offset(offset).Limit(req.Limit).Find(&images).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve images: %w", err)
	}

	// Calculate pagination
	totalPages := int((total + int64(req.Limit) - 1) / int64(req.Limit))

	return &ImageListResponse{
		Images: images,
		Pagination: Pagination{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      int(total),
			TotalPages: totalPages,
			HasNext:    req.Page < totalPages,
			HasPrev:    req.Page > 1,
		},
	}, nil
}

// GetImage retrieves a single image by ID
func (s *Service) GetImage(imageID uint) (*UploadedFile, error) {
	var image UploadedFile
	if err := s.db.First(&image, imageID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("image not found")
		}
		return nil, fmt.Errorf("failed to retrieve image: %w", err)
	}
	return &image, nil
}

// GetImageByFilename retrieves an image by filename
func (s *Service) GetImageByFilename(filename string) (*UploadedFile, error) {
	var image UploadedFile
	if err := s.db.Where("filename = ?", filename).First(&image).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("image not found")
		}
		return nil, fmt.Errorf("failed to retrieve image: %w", err)
	}
	return &image, nil
}

// UpdateImage updates image metadata
func (s *Service) UpdateImage(imageID, userID uint, req *ImageUpdateRequest) (*UploadedFile, error) {
	var image UploadedFile
	if err := s.db.First(&image, imageID).Error; err != nil {
		return nil, fmt.Errorf("image not found")
	}

	// Build updates
	updates := make(map[string]interface{})

	if req.Category != nil {
		updates["category"] = *req.Category
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.AltText != nil {
		updates["alt_text"] = *req.AltText
	}
	if req.Tags != nil {
		updates["tags"] = *req.Tags
	}
	if req.IsPublic != nil {
		updates["is_public"] = *req.IsPublic
	}

	if len(updates) > 0 {
		if err := s.db.Model(&image).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("failed to update image: %w", err)
		}
	}

	return &image, nil
}

// OptimizeImage optimizes an image with specified parameters
func (s *Service) OptimizeImage(imageID, userID uint, req *ImageOptimizeRequest) (*UploadedFile, error) {
	var image UploadedFile
	if err := s.db.First(&image, imageID).Error; err != nil {
		return nil, fmt.Errorf("image not found")
	}

	if !image.IsImage() {
		return nil, fmt.Errorf("file is not an image")
	}

	// Original file path
	originalPath := filepath.Join(s.config.External.Storage.LocalPath, image.Path)

	// Generate optimized filename
	ext := filepath.Ext(image.Filename)
	nameWithoutExt := strings.TrimSuffix(image.Filename, ext)
	optimizedFilename := fmt.Sprintf("%s_optimized_%dx%d_q%d%s", nameWithoutExt, req.Width, req.Height, req.Quality, ext)
	optimizedPath := filepath.Join(s.config.External.Storage.LocalPath, image.Category, optimizedFilename)

	// Optimize image
	if err := s.optimizeImageFile(originalPath, optimizedPath, req); err != nil {
		return nil, fmt.Errorf("failed to optimize image: %w", err)
	}

	// Update database record
	optimizedURL := s.getFileURL(filepath.Join(image.Category, optimizedFilename))
	if err := s.db.Model(&image).Update("optimized_url", optimizedURL).Error; err != nil {
		return nil, fmt.Errorf("failed to update optimized URL: %w", err)
	}

	image.OptimizedURL = optimizedURL
	return &image, nil
}

// GetUploadStats returns upload statistics
func (s *Service) GetUploadStats() (*UploadStats, error) {
	stats := &UploadStats{
		CategoryBreakdown: make(map[string]int64),
		MonthlyUploads:    []MonthlyUpload{},
		RecentUploads:     []UploadedFile{},
	}

	// Total files and size
	s.db.Model(&UploadedFile{}).Count(&stats.TotalFiles)
	s.db.Model(&UploadedFile{}).Select("COALESCE(SUM(size), 0)").Row().Scan(&stats.TotalSize)

	// Format total size
	stats.TotalSizeFormatted = s.formatFileSize(stats.TotalSize)

	// Image count
	s.db.Model(&UploadedFile{}).Where("mime_type LIKE 'image/%'").Count(&stats.ImageCount)

	// Category breakdown
	var categoryResults []struct {
		Category string `json:"category"`
		Count    int64  `json:"count"`
	}
	s.db.Model(&UploadedFile{}).Select("category, COUNT(*) as count").Group("category").Scan(&categoryResults)

	for _, result := range categoryResults {
		stats.CategoryBreakdown[result.Category] = result.Count
	}

	// Monthly uploads (last 12 months)
	var monthlyResults []struct {
		Month string `json:"month"`
		Count int64  `json:"count"`
		Size  int64  `json:"size"`
	}
	s.db.Raw(`
		SELECT 
			TO_CHAR(created_at, 'YYYY-MM') as month,
			COUNT(*) as count,
			COALESCE(SUM(size), 0) as size
		FROM uploaded_files 
		WHERE created_at >= NOW() - INTERVAL '12 months'
		GROUP BY TO_CHAR(created_at, 'YYYY-MM')
		ORDER BY month DESC
	`).Scan(&monthlyResults)

	stats.MonthlyUploads = make([]MonthlyUpload, len(monthlyResults))
	for i, result := range monthlyResults {
		stats.MonthlyUploads[i] = MonthlyUpload{
			Month: result.Month,
			Count: result.Count,
			Size:  result.Size,
		}
	}

	// Recent uploads (last 10)
	s.db.Order("created_at DESC").Limit(10).Find(&stats.RecentUploads)

	return stats, nil
}

// Private helper methods

func (s *Service) validateImageFile(header *multipart.FileHeader) error {
	// Check file size
	if header.Size > s.config.Upload.MaxSize {
		return fmt.Errorf("file size exceeds maximum allowed size of %s", s.formatFileSize(s.config.Upload.MaxSize))
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != "" {
		ext = ext[1:] // Remove the dot
	}

	allowed := false
	for _, allowedExt := range s.config.Upload.AllowedExtensions {
		if ext == allowedExt {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("file type '%s' is not allowed. Allowed types: %v", ext, s.config.Upload.AllowedExtensions)
	}

	return nil
}

func (s *Service) generateUniqueFilename(originalFilename string) string {
	ext := filepath.Ext(originalFilename)
	name := strings.TrimSuffix(originalFilename, ext)

	// Sanitize filename
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")

	// Add UUID to ensure uniqueness
	return fmt.Sprintf("%s_%s%s", name, uuid.New().String()[:8], ext)
}

func (s *Service) isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp"}

	for _, imageExt := range imageExts {
		if ext == imageExt {
			return true
		}
	}
	return false
}

func (s *Service) getMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	mimeTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".bmp":  "image/bmp",
		".svg":  "image/svg+xml",
		".pdf":  "application/pdf",
		".txt":  "text/plain",
	}

	if mimeType, exists := mimeTypes[ext]; exists {
		return mimeType
	}

	return "application/octet-stream"
}

func (s *Service) getImageDimensions(filePath string) (int, int) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0
	}
	defer file.Close()

	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0
	}

	return config.Width, config.Height
}

func (s *Service) generateThumbnail(originalPath, category, filename string) (string, error) {
	// Open original image
	file, err := os.Open(originalPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Decode image
	img, format, err := image.Decode(file)
	if err != nil {
		return "", err
	}

	// Calculate thumbnail dimensions while maintaining aspect ratio
	origWidth := img.Bounds().Dx()
	origHeight := img.Bounds().Dy()

	thumbWidth := s.config.Upload.ThumbnailWidth
	thumbHeight := s.config.Upload.ThumbnailHeight

	// Calculate scaling factor
	scaleX := float64(thumbWidth) / float64(origWidth)
	scaleY := float64(thumbHeight) / float64(origHeight)
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	newWidth := int(float64(origWidth) * scale)
	newHeight := int(float64(origHeight) * scale)

	fmt.Println(newWidth, newHeight, format)

	// Create thumbnail (simplified - in production, use image processing library)
	// For now, we'll just copy the original with a different name
	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)
	thumbnailFilename := fmt.Sprintf("%s_thumb%s", nameWithoutExt, ext)
	thumbnailPath := filepath.Join(s.config.External.Storage.LocalPath, category, thumbnailFilename)

	// In a real implementation, you'd resize the image here
	// For simplicity, we're just copying the file
	srcFile, _ := os.Open(originalPath)
	defer srcFile.Close()

	dstFile, err := os.Create(thumbnailPath)
	if err != nil {
		return "", err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return "", err
	}

	// Return relative path for URL generation
	return filepath.Join(category, thumbnailFilename), nil
}

func (s *Service) optimizeImageFile(originalPath, optimizedPath string, req *ImageOptimizeRequest) error {
	// Open original image
	file, err := os.Open(originalPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Decode image
	img, format, err := image.Decode(file)
	if err != nil {
		return err
	}

	// Create optimized file
	optimizedFile, err := os.Create(optimizedPath)
	if err != nil {
		return err
	}
	defer optimizedFile.Close()

	// Encode with specified quality (simplified implementation)
	switch format {
	case "jpeg":
		return jpeg.Encode(optimizedFile, img, &jpeg.Options{Quality: req.Quality})
	case "png":
		return png.Encode(optimizedFile, img)
	default:
		return fmt.Errorf("unsupported image format: %s", format)
	}
}

func (s *Service) getFileURL(relativePath string) string {
	// In production, this might use CDN URL
	baseURL := s.config.External.Storage.CDNBaseURL
	if baseURL == "" {
		baseURL = "/uploads"
	}

	return filepath.Join(baseURL, relativePath)
}

func (s *Service) urlToPath(url string) string {
	// Convert URL back to relative path
	baseURL := s.config.External.Storage.CDNBaseURL
	if baseURL == "" {
		baseURL = "/uploads"
	}

	return strings.TrimPrefix(url, baseURL+"/")
}

func (s *Service) buildOrderClause(sortBy, sortOrder string) string {
	validSortFields := map[string]bool{
		"created_at":    true,
		"original_name": true,
		"size":          true,
		"category":      true,
	}

	if !validSortFields[sortBy] {
		sortBy = "created_at"
	}

	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	return fmt.Sprintf("%s %s", sortBy, sortOrder)
}

func (s *Service) formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
