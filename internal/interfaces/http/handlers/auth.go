// internal/interfaces/http/handlers/auth.go - Complete implementation with email integration
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/your-org/ecommerce-backend/internal/config"
	"github.com/your-org/ecommerce-backend/internal/domain/user"
	"github.com/your-org/ecommerce-backend/internal/interfaces/http/middleware"
	"github.com/your-org/ecommerce-backend/internal/pkg/email"
	"gorm.io/gorm"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	userService  *user.Service
	config       *config.Config
	db           *gorm.DB
	redisClient  *redis.Client
	emailService *email.EmailService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(db *gorm.DB, redisClient *redis.Client, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		userService:  user.NewService(db, cfg),
		config:       cfg,
		db:           db,
		redisClient:  redisClient,
		emailService: email.NewEmailService(cfg),
	}
}

// TokenService for managing verification and reset tokens
type TokenService struct {
	redisClient *redis.Client
}

type TokenData struct {
	UserID    uint      `json:"user_id"`
	Email     string    `json:"email"`
	TokenType string    `json:"token_type"` // "email_verification", "password_reset"
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Register handles user registration
func (h *AuthHandler) Register(c *gin.Context) {
	var req user.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	response, err := h.userService.Register(&req)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Generate email verification token
	verificationToken, err := h.generateVerificationToken(response.User.ID, response.User.Email)
	if err != nil {
		log.Printf("Failed to generate verification token for user %d: %v", response.User.ID, err)
		// Don't fail registration, just log the error
	} else {
		// Send verification email asynchronously
		go func() {
			ctx := context.Background()
			userName := response.User.GetDisplayName()

			if err := h.emailService.SendWelcomeEmail(ctx, response.User.Email, userName, verificationToken); err != nil {
				log.Printf("Failed to send welcome email to %s: %v", response.User.Email, err)
			}
		}()
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully. Please check your email to verify your account.",
		"data":    response,
	})
}

// Login handles user login
func (h *AuthHandler) Login(c *gin.Context) {
	var req user.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	response, err := h.userService.Login(&req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"data":    response,
	})
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	response, err := h.userService.RefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Token refreshed successfully",
		"data":    response,
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// In a stateless JWT system, logout is handled client-side
	// Here you could add token blacklisting logic if needed

	userID, exists := middleware.GetUserIDFromContext(c)
	if exists {
		log.Printf("User %d logged out", userID)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}

// GetProfile gets current user profile
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	profile, err := h.userService.GetProfile(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile retrieved successfully",
		"data":    profile,
	})
}

// UpdateProfile updates current user profile
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	profile, err := h.userService.UpdateProfile(userID, updates)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully",
		"data":    profile,
	})
}

// ForgotPassword handles password reset request
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	// Find user by email
	var userRecord user.User
	result := h.db.Where("email = ? AND is_active = ?", req.Email, true).First(&userRecord)
	if result.Error != nil {
		// Don't reveal if email exists or not for security
		c.JSON(http.StatusOK, gin.H{
			"message": "If an account with this email exists, a password reset link has been sent.",
		})
		return
	}

	// Generate reset token
	resetToken, err := h.generatePasswordResetToken(userRecord.ID, userRecord.Email)
	if err != nil {
		log.Printf("Failed to generate reset token for user %d: %v", userRecord.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process password reset request",
		})
		return
	}

	// Send reset email asynchronously
	go func() {
		ctx := context.Background()
		userName := userRecord.GetDisplayName()

		if err := h.emailService.SendPasswordResetEmailByToken(ctx, userRecord.Email, userName, resetToken); err != nil {
			log.Printf("Failed to send password reset email to %s: %v", userRecord.Email, err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "If an account with this email exists, a password reset link has been sent.",
	})
}

// ResetPassword handles password reset
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req struct {
		Token           string `json:"token" binding:"required"`
		Password        string `json:"password" binding:"required"`
		ConfirmPassword string `json:"confirm_password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	if req.Password != req.ConfirmPassword {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Passwords do not match",
		})
		return
	}

	// Validate reset token
	tokenData, err := h.validatePasswordResetToken(req.Token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid or expired reset token",
		})
		return
	}

	// Update user password
	err = h.userService.ResetPassword(tokenData.UserID, req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Invalidate the reset token
	h.invalidateToken(req.Token)

	c.JSON(http.StatusOK, gin.H{
		"message": "Password reset successfully",
	})
}

// ChangePassword handles password change for authenticated users
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required"`
		ConfirmPassword string `json:"confirm_password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	if req.NewPassword != req.ConfirmPassword {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "New passwords do not match",
		})
		return
	}

	err := h.userService.ChangePassword(userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password changed successfully",
	})
}

// VerifyEmail handles email verification
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Verification token is required",
		})
		return
	}

	// Validate verification token
	tokenData, err := h.validateVerificationToken(token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid or expired verification token",
		})
		return
	}

	// Mark email as verified
	now := time.Now().UTC()
	err = h.db.Model(&user.User{}).
		Where("id = ?", tokenData.UserID).
		Updates(map[string]interface{}{
			"email_verified":    true,
			"email_verified_at": now,
		}).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to verify email",
		})
		return
	}

	// Invalidate the verification token
	h.invalidateToken(token)

	c.JSON(http.StatusOK, gin.H{
		"message": "Email verified successfully",
	})
}

// ResendVerification handles resending email verification
func (h *AuthHandler) ResendVerification(c *gin.Context) {
	userID, hasAuth := middleware.GetUserIDFromContext(c)

	var userRecord user.User
	var err error

	if hasAuth {
		// User is authenticated - use their ID
		err = h.db.Where("id = ?", userID).First(&userRecord).Error
	} else {
		// User not authenticated - require email in request
		var req struct {
			Email string `json:"email" binding:"required,email"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Email is required when not authenticated",
			})
			return
		}

		err = h.db.Where("email = ? AND is_active = ?", req.Email, true).First(&userRecord).Error
	}

	if err != nil {
		// Don't reveal if email exists for security
		c.JSON(http.StatusOK, gin.H{
			"message": "If the email exists, a verification email has been sent",
		})
		return
	}

	// Check if email is already verified
	if userRecord.EmailVerified {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Email is already verified",
		})
		return
	}

	// Generate new verification token
	verificationToken, err := h.generateVerificationToken(userRecord.ID, userRecord.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate verification token",
		})
		return
	}

	// Send verification email asynchronously
	go func() {
		ctx := context.Background()
		userName := userRecord.GetDisplayName()

		emailData := email.EmailVerificationData{
			EmailTemplateData: email.GetBaseTemplateData(
				h.config.External.Email.FromName,
				h.config.External.Email.BaseURL,
				userName,
				userRecord.Email,
			),
			VerificationURL: fmt.Sprintf("%s/verify-email?token=%s", h.config.External.Email.BaseURL, verificationToken),
			ExpiryTime:      "24 hours",
		}

		if err := h.emailService.SendEmailVerificationEmail(ctx, emailData); err != nil {
			log.Printf("Failed to send verification email to %s: %v", userRecord.Email, err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "Verification email sent",
	})
}

// GetCurrentUser returns the current authenticated user's information
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	user, err := h.userService.GetProfile(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": user,
	})
}

// ValidateToken validates if the provided token is still valid
func (h *AuthHandler) ValidateToken(c *gin.Context) {
	// If we reach here, the token is valid (middleware already validated it)
	userID, _ := middleware.GetUserIDFromContext(c)
	email, _ := middleware.GetUserEmailFromContext(c)
	isAdmin := middleware.IsAdminFromContext(c)

	c.JSON(http.StatusOK, gin.H{
		"valid": true,
		"user": gin.H{
			"id":       userID,
			"email":    email,
			"is_admin": isAdmin,
		},
	})
}

// Token management helper methods

func (h *AuthHandler) generateVerificationToken(userID uint, email string) (string, error) {
	token := uuid.New().String()

	tokenData := TokenData{
		UserID:    userID,
		Email:     email,
		TokenType: "email_verification",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour), // 24 hours expiry
	}

	return h.storeToken(token, tokenData)
}

func (h *AuthHandler) generatePasswordResetToken(userID uint, email string) (string, error) {
	token := uuid.New().String()

	tokenData := TokenData{
		UserID:    userID,
		Email:     email,
		TokenType: "password_reset",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour), // 24 hours expiry
	}

	return h.storeToken(token, tokenData)
}

// Better storage format (JSON)
func (h *AuthHandler) storeToken(token string, data TokenData) (string, error) {
	ctx := context.Background()
	key := fmt.Sprintf("auth_token:%s", token)

	// Store as JSON instead of colon-separated
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal token data: %w", err)
	}

	err = h.redisClient.Set(ctx, key, jsonData, time.Until(data.ExpiresAt)).Err()
	if err != nil {
		return "", fmt.Errorf("failed to store token: %w", err)
	}

	return token, nil
}

func (h *AuthHandler) validateToken(token, expectedType string) (*TokenData, error) {
	ctx := context.Background()
	key := fmt.Sprintf("auth_token:%s", token)

	tokenStr, err := h.redisClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("token not found or expired")
		}
		return nil, fmt.Errorf("failed to validate token: %w", err)
	}

	var tokenData TokenData
	err = json.Unmarshal([]byte(tokenStr), &tokenData)
	if err != nil {
		return nil, fmt.Errorf("invalid token format: %w", err)
	}

	// Check token type
	if tokenData.TokenType != expectedType {
		return nil, fmt.Errorf("invalid token type")
	}

	// Check expiry
	if time.Now().After(tokenData.ExpiresAt) {
		h.invalidateToken(token)
		return nil, fmt.Errorf("token has expired")
	}

	return &tokenData, nil
}

func (h *AuthHandler) validateVerificationToken(token string) (*TokenData, error) {
	return h.validateToken(token, "email_verification")
}

func (h *AuthHandler) validatePasswordResetToken(token string) (*TokenData, error) {
	return h.validateToken(token, "password_reset")
}

func (h *AuthHandler) invalidateToken(token string) {
	ctx := context.Background()
	key := fmt.Sprintf("auth_token:%s", token)
	h.redisClient.Del(ctx, key)
}
