// internal/domain/upload/service.go
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
	"time"

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
	File        multipart.File   `json:"-"`
	Header      *multipart.FileHeader `json:"-"`
	Category    string           `json:"category"`
	Description string           `json:"description"`
	AltText     string           `json:"alt_text"`
	Tags        string           `json:"tags"`
	UploadedBy  uint            `json:"uploaded_by"`
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
	UploadedBy  uint                   `json:"uploaded_by"`
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
	TotalFiles    int   `json:"total_files"`
	SuccessCount  int   `json:"success_count"`
	FailureCount  int   `json:"failure_count"`
	TotalSize     int64 `json:"total_size"`
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
	TotalFiles       int64   `json:"total_files"`
	TotalSize        int64   `json:"total_size"`
	TotalSizeFormatted string `json:"total_size_formatted"`
	ImageCount       int64   `json:"image_count"`
	CategoryBreakdown map[string]int64 `json:"category_breakdown"`
	MonthlyUploads   []MonthlyUpload `json:"monthly_uploads"`
	RecentUploads    []UploadedFile  `json:"recent_uploads"`
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
	if err := tx.Delete(&uploadedFile).Error; err