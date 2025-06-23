// internal/domain/user/entity.go
package user

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// User represents the user entity
type User struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	Email           string         `gorm:"uniqueIndex;not null;size:255" json:"email"`
	Password        string         `gorm:"not null;size:255" json:"-"` // Don't return in JSON
	FirstName       string         `gorm:"size:100" json:"first_name"`
	LastName        string         `gorm:"size:100" json:"last_name"`
	Phone           string         `gorm:"size:20" json:"phone"`
	DateOfBirth     *time.Time     `json:"date_of_birth"`
	Avatar          string         `gorm:"size:500" json:"avatar"`
	IsActive        bool           `gorm:"default:true" json:"is_active"`
	IsAdmin         bool           `gorm:"default:false" json:"is_admin"`
	EmailVerified   bool           `gorm:"default:false" json:"email_verified"`
	EmailVerifiedAt *time.Time     `json:"email_verified_at"`
	LastLoginAt     *time.Time     `json:"last_login_at"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Addresses []Address `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"addresses,omitempty"`
}

// Address represents user addresses
type Address struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"not null;index" json:"user_id"`
	Type         string    `gorm:"size:20;default:'shipping'" json:"type"` // shipping, billing
	FirstName    string    `gorm:"size:100" json:"first_name"`
	LastName     string    `gorm:"size:100" json:"last_name"`
	Company      string    `gorm:"size:100" json:"company"`
	AddressLine1 string    `gorm:"size:255;not null" json:"address_line1"`
	AddressLine2 string    `gorm:"size:255" json:"address_line2"`
	City         string    `gorm:"size:100;not null" json:"city"`
	State        string    `gorm:"size:100" json:"state"`
	PostalCode   string    `gorm:"size:20" json:"postal_code"`
	Country      string    `gorm:"size:2;not null;default:'US'" json:"country"` // ISO 2-letter code
	IsDefault    bool      `gorm:"default:false" json:"is_default"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TableName overrides the table name for User
func (User) TableName() string {
	return "users"
}

// TableName overrides the table name for Address
func (Address) TableName() string {
	return "addresses"
}

// BeforeCreate hook to handle business logic before user creation
func (u *User) BeforeCreate(tx *gorm.DB) error {
	// Email should be lowercase
	u.Email = strings.ToLower(u.Email)
	return nil
}

// GetFullName returns the user's full name
func (u *User) GetFullName() string {
	return strings.TrimSpace(u.FirstName + " " + u.LastName)
}

// GetDisplayName returns display name (full name or email)
func (u *User) GetDisplayName() string {
	fullName := u.GetFullName()
	if fullName != "" {
		return fullName
	}
	return u.Email
}
