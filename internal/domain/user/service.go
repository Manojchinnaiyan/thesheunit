// internal/domain/user/service.go - Updated with ResetPassword method
package user

import (
	"fmt"
	"log"
	"time"

	"github.com/your-org/ecommerce-backend/internal/config"
	"github.com/your-org/ecommerce-backend/internal/pkg/auth"
	"github.com/your-org/ecommerce-backend/internal/pkg/email"
	"gorm.io/gorm"
)

// Service handles user business logic
type Service struct {
	db              *gorm.DB
	config          *config.Config
	passwordManager *auth.PasswordManager
	jwtManager      *auth.JWTManager
	emailService    *email.EmailService
}

// NewService creates a new user service
func NewService(db *gorm.DB, cfg *config.Config) *Service {
	return &Service{
		db:              db,
		config:          cfg,
		passwordManager: auth.NewPasswordManager(cfg),
		jwtManager:      auth.NewJWTManager(cfg),
		emailService:    email.NewEmailService(cfg),
	}
}

// RegisterRequest represents user registration data
type RegisterRequest struct {
	Email           string `json:"email" binding:"required,email"`
	Password        string `json:"password" binding:"required"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
	FirstName       string `json:"first_name" binding:"required"`
	LastName        string `json:"last_name" binding:"required"`
	Phone           string `json:"phone"`
}

// LoginRequest represents user login data
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse represents authentication response
type AuthResponse struct {
	User         *User  `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// Register creates a new user account
func (s *Service) Register(req *RegisterRequest) (*AuthResponse, error) {
	// Validate password confirmation
	if req.Password != req.ConfirmPassword {
		return nil, fmt.Errorf("passwords do not match")
	}

	// Check if user already exists
	var existingUser User
	result := s.db.Where("email = ?", req.Email).First(&existingUser)
	if result.Error == nil {
		return nil, fmt.Errorf("user with this email already exists")
	}

	// Hash password
	hashedPassword, err := s.passwordManager.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create new user
	user := User{
		Email:     req.Email,
		Password:  hashedPassword,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Phone:     req.Phone,
		IsActive:  true,
		IsAdmin:   false,
	}

	if err := s.db.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate tokens
	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email, user.IsAdmin)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(user.ID, user.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Update last login
	user.LastLoginAt = &time.Time{}
	*user.LastLoginAt = time.Now().UTC()
	s.db.Save(&user)

	// Clear password from response
	user.Password = ""

	return &AuthResponse{
		User:         &user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.JWT.AccessTokenExpiry.Seconds()),
	}, nil
}

// Login authenticates a user
func (s *Service) Login(req *LoginRequest) (*AuthResponse, error) {
	// Find user by email
	var user User
	result := s.db.Where("email = ? AND is_active = ?", req.Email, true).First(&user)
	if result.Error != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Verify password
	if err := s.passwordManager.VerifyPassword(req.Password, user.Password); err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Generate tokens
	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email, user.IsAdmin)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(user.ID, user.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Update last login
	now := time.Now().UTC()
	user.LastLoginAt = &now
	s.db.Save(&user)

	// Clear password from response
	user.Password = ""

	return &AuthResponse{
		User:         &user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.JWT.AccessTokenExpiry.Seconds()),
	}, nil
}

// RefreshToken generates new tokens using refresh token
func (s *Service) RefreshToken(refreshToken string) (*AuthResponse, error) {
	// Validate refresh token
	claims, err := s.jwtManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Find user
	var user User
	result := s.db.Where("id = ? AND is_active = ?", claims.UserID, true).First(&user)
	if result.Error != nil {
		return nil, fmt.Errorf("user not found or inactive")
	}

	// Generate new tokens
	newAccessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email, user.IsAdmin)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	var newRefreshToken string
	if s.config.JWT.RefreshTokenRotation {
		// Generate new refresh token (rotation)
		newRefreshToken, err = s.jwtManager.GenerateRefreshToken(user.ID, user.Email)
		if err != nil {
			return nil, fmt.Errorf("failed to generate refresh token: %w", err)
		}
	} else {
		// Reuse existing refresh token
		newRefreshToken = refreshToken
	}

	// Clear password from response
	user.Password = ""

	return &AuthResponse{
		User:         &user,
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(s.config.JWT.AccessTokenExpiry.Seconds()),
	}, nil
}

// GetProfile gets user profile by ID
func (s *Service) GetProfile(userID uint) (*User, error) {
	var user User
	result := s.db.Where("id = ? AND is_active = ?", userID, true).First(&user)
	if result.Error != nil {
		return nil, fmt.Errorf("user not found")
	}

	// Clear password
	user.Password = ""

	return &user, nil
}

// UpdateProfile updates user profile
func (s *Service) UpdateProfile(userID uint, updates map[string]interface{}) (*User, error) {
	var user User
	result := s.db.Where("id = ? AND is_active = ?", userID, true).First(&user)
	if result.Error != nil {
		return nil, fmt.Errorf("user not found")
	}

	// Remove sensitive fields from updates
	delete(updates, "password")
	delete(updates, "is_admin")
	delete(updates, "is_active")
	delete(updates, "email_verified")

	if err := s.db.Model(&user).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}

	// Clear password
	user.Password = ""

	return &user, nil
}

// ChangePassword changes user password after verifying current password
func (s *Service) ChangePassword(userID uint, currentPassword, newPassword string) error {
	// Find user
	var user User
	result := s.db.Where("id = ? AND is_active = ?", userID, true).First(&user)
	if result.Error != nil {
		return fmt.Errorf("user not found")
	}

	// Verify current password
	if err := s.passwordManager.VerifyPassword(currentPassword, user.Password); err != nil {
		return fmt.Errorf("current password is incorrect")
	}

	// Hash new password
	hashedPassword, err := s.passwordManager.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update password
	if err := s.db.Model(&user).Update("password", hashedPassword).Error; err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// ResetPassword resets user password using token (without current password)
func (s *Service) ResetPassword(userID uint, newPassword string) error {
	// Find user
	var user User
	result := s.db.Where("id = ? AND is_active = ?", userID, true).First(&user)
	if result.Error != nil {
		return fmt.Errorf("user not found")
	}

	// Hash new password
	hashedPassword, err := s.passwordManager.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update password
	if err := s.db.Model(&user).Update("password", hashedPassword).Error; err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Log password reset
	log.Printf("Password reset successfully for user ID: %d", userID)

	return nil
}

// SendPasswordResetEmail sends password reset email (now handled by auth handler)
func (s *Service) SendPasswordResetEmail(email string) error {
	// This method is deprecated - password reset is now handled by auth handlers
	// with proper token management and email service integration
	return fmt.Errorf("use auth handler for password reset functionality")
}

// VerifyEmail marks user email as verified
func (s *Service) VerifyEmail(userID uint) error {
	now := time.Now().UTC()

	err := s.db.Model(&User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"email_verified":    true,
			"email_verified_at": now,
		}).Error

	if err != nil {
		return fmt.Errorf("failed to verify email: %w", err)
	}

	log.Printf("Email verified for user ID: %d", userID)
	return nil
}

// DeactivateUser deactivates a user account
func (s *Service) DeactivateUser(userID uint) error {
	err := s.db.Model(&User{}).
		Where("id = ?", userID).
		Update("is_active", false).Error

	if err != nil {
		return fmt.Errorf("failed to deactivate user: %w", err)
	}

	log.Printf("User deactivated: %d", userID)
	return nil
}

// ActivateUser reactivates a user account
func (s *Service) ActivateUser(userID uint) error {
	err := s.db.Model(&User{}).
		Where("id = ?", userID).
		Update("is_active", true).Error

	if err != nil {
		return fmt.Errorf("failed to activate user: %w", err)
	}

	log.Printf("User activated: %d", userID)
	return nil
}

// GetUserByEmail retrieves user by email
func (s *Service) GetUserByEmail(email string) (*User, error) {
	var user User
	result := s.db.Where("email = ?", email).First(&user)
	if result.Error != nil {
		return nil, fmt.Errorf("user not found")
	}

	// Clear password
	user.Password = ""
	return &user, nil
}

// UpdateLastLogin updates user's last login timestamp
func (s *Service) UpdateLastLogin(userID uint) error {
	now := time.Now().UTC()
	return s.db.Model(&User{}).
		Where("id = ?", userID).
		Update("last_login_at", now).Error
}

// IsEmailVerified checks if user's email is verified
func (s *Service) IsEmailVerified(userID uint) (bool, error) {
	var user User
	result := s.db.Select("email_verified").Where("id = ?", userID).First(&user)
	if result.Error != nil {
		return false, fmt.Errorf("user not found")
	}
	return user.EmailVerified, nil
}
