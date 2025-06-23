// internal/domain/upload/entity.go
package upload

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// UploadedFile represents an uploaded file in the database
type UploadedFile struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	OriginalName string `gorm:"not null;size:255" json:"original_name"`
	Filename     string `gorm:"not null;size:255;uniqueIndex" json:"filename"`
	Path         string `gorm:"not null;size:500" json:"path"`
	URL          string `gorm:"not null;size:500" json:"url"`
	MimeType     string `gorm:"not null;size:100" json:"mime_type"`
	Size         int64  `gorm:"not null" json:"size"`
	Category     string `gorm:"size:50;index" json:"category"`
	Description  string `gorm:"size:500" json:"description"`
	AltText      string `gorm:"size:255" json:"alt_text"`
	Tags         string `gorm:"size:500" json:"tags"`

	// Image specific fields
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
	ThumbnailURL string `gorm:"size:500" json:"thumbnail_url,omitempty"`
	OptimizedURL string `gorm:"size:500" json:"optimized_url,omitempty"`

	// Metadata
	UploadedBy uint       `gorm:"not null;index" json:"uploaded_by"`
	IsPublic   bool       `gorm:"default:true" json:"is_public"`
	UsageCount int        `gorm:"default:0" json:"usage_count"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`

	// Timestamps
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// FileUsage represents where a file is being used
type FileUsage struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	FileID     uint      `gorm:"not null;index" json:"file_id"`
	EntityType string    `gorm:"not null;size:50" json:"entity_type"` // product, category, user, etc.
	EntityID   uint      `gorm:"not null" json:"entity_id"`
	UsageType  string    `gorm:"size:50" json:"usage_type"` // primary, gallery, thumbnail, etc.
	CreatedAt  time.Time `json:"created_at"`

	// Relationships
	File UploadedFile `gorm:"foreignKey:FileID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"file,omitempty"`
}

// TableName overrides
func (UploadedFile) TableName() string { return "uploaded_files" }
func (FileUsage) TableName() string    { return "file_usage" }

// Business methods

// IsImage checks if the file is an image
func (f *UploadedFile) IsImage() bool {
	imageTypes := []string{
		"image/jpeg", "image/png", "image/gif",
		"image/webp", "image/svg+xml", "image/bmp",
	}

	for _, imageType := range imageTypes {
		if f.MimeType == imageType {
			return true
		}
	}
	return false
}

// GetFormattedSize returns human-readable file size
func (f *UploadedFile) GetFormattedSize() string {
	const unit = 1024
	if f.Size < unit {
		return fmt.Sprintf("%d B", f.Size)
	}

	div, exp := int64(unit), 0
	for n := f.Size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(f.Size)/float64(div), "KMGTPE"[exp])
}

// GetDimensions returns image dimensions as string
func (f *UploadedFile) GetDimensions() string {
	if f.Width > 0 && f.Height > 0 {
		return fmt.Sprintf("%dx%d", f.Width, f.Height)
	}
	return ""
}

// IncrementUsage increments the usage count
func (f *UploadedFile) IncrementUsage() {
	f.UsageCount++
	now := time.Now().UTC()
	f.LastUsedAt = &now
}
