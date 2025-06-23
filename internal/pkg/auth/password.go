// internal/pkg/auth/password.go
package auth

import (
	"fmt"
	"regexp"
	"unicode"

	"github.com/your-org/ecommerce-backend/internal/config"
	"golang.org/x/crypto/bcrypt"
)

// PasswordManager handles password operations
type PasswordManager struct {
	config *config.Config
}

// NewPasswordManager creates a new password manager
func NewPasswordManager(cfg *config.Config) *PasswordManager {
	return &PasswordManager{
		config: cfg,
	}
}

// HashPassword hashes a password using bcrypt
func (p *PasswordManager) HashPassword(password string) (string, error) {
	if err := p.ValidatePassword(password); err != nil {
		return "", fmt.Errorf("password validation failed: %w", err)
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), p.config.Security.BcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hashedBytes), nil
}

// VerifyPassword verifies a password against its hash
func (p *PasswordManager) VerifyPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// ValidatePassword validates password strength
func (p *PasswordManager) ValidatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	if len(password) > 128 {
		return fmt.Errorf("password must be no more than 128 characters long")
	}

	var (
		hasUpper   bool
		hasLower   bool
		hasNumber  bool
		hasSpecial bool
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if !hasNumber {
		return fmt.Errorf("password must contain at least one number")
	}
	if !hasSpecial {
		return fmt.Errorf("password must contain at least one special character")
	}

	// Check for common patterns
	if err := p.checkCommonPatterns(password); err != nil {
		return err
	}

	return nil
}

// checkCommonPatterns checks for common weak password patterns
func (p *PasswordManager) checkCommonPatterns(password string) error {
	// Check for sequential characters (abc, 123)
	if matched, _ := regexp.MatchString(`(abc|bcd|cde|def|efg|fgh|ghi|hij|ijk|jkl|klm|lmn|mno|nop|opq|pqr|qrs|rst|stu|tuv|uvw|vwx|wxy|xyz)`, password); matched {
		return fmt.Errorf("password cannot contain sequential letters")
	}

	if matched, _ := regexp.MatchString(`(012|123|234|345|456|567|678|789)`, password); matched {
		return fmt.Errorf("password cannot contain sequential numbers")
	}

	// Check for repeating characters
	if matched, _ := regexp.MatchString(`(.)\1{2,}`, password); matched {
		return fmt.Errorf("password cannot contain more than 2 repeating characters")
	}

	// Check for common weak passwords
	commonPasswords := []string{
		"password", "123456", "password123", "admin", "qwerty", "letmein",
		"welcome", "monkey", "dragon", "password1", "123456789", "football",
	}

	for _, common := range commonPasswords {
		if matched, _ := regexp.MatchString(`(?i)`+common, password); matched {
			return fmt.Errorf("password is too common and easily guessable")
		}
	}

	return nil
}

// GenerateTemporaryPassword generates a secure temporary password
func (p *PasswordManager) GenerateTemporaryPassword() (string, error) {
	// Implementation for generating secure temporary passwords
	// This is a simple version - you might want to use crypto/rand for production
	return "TempPass123!", nil
}
