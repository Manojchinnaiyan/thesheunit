// internal/domain/user/address_service.go
package user

import (
	"fmt"
	"strings"

	"github.com/your-org/ecommerce-backend/internal/config"
	"gorm.io/gorm"
)

// AddressService handles address business logic
type AddressService struct {
	db     *gorm.DB
	config *config.Config
}

// NewAddressService creates a new address service
func NewAddressService(db *gorm.DB, cfg *config.Config) *AddressService {
	return &AddressService{
		db:     db,
		config: cfg,
	}
}

// CreateAddressRequest represents address creation data
type CreateAddressRequest struct {
	Type         string `json:"type" binding:"required,oneof=shipping billing"` // shipping or billing
	FirstName    string `json:"first_name" binding:"required"`
	LastName     string `json:"last_name" binding:"required"`
	Company      string `json:"company"`
	AddressLine1 string `json:"address_line1" binding:"required"`
	AddressLine2 string `json:"address_line2"`
	City         string `json:"city" binding:"required"`
	State        string `json:"state" binding:"required"`
	PostalCode   string `json:"postal_code" binding:"required"`
	Country      string `json:"country" binding:"required,len=2"` // ISO 2-letter code
	Phone        string `json:"phone"`
	IsDefault    bool   `json:"is_default"`
}

// UpdateAddressRequest represents address update data
type UpdateAddressRequest struct {
	Type         *string `json:"type" binding:"omitempty,oneof=shipping billing"`
	FirstName    *string `json:"first_name"`
	LastName     *string `json:"last_name"`
	Company      *string `json:"company"`
	AddressLine1 *string `json:"address_line1"`
	AddressLine2 *string `json:"address_line2"`
	City         *string `json:"city"`
	State        *string `json:"state"`
	PostalCode   *string `json:"postal_code"`
	Country      *string `json:"country" binding:"omitempty,len=2"`
	Phone        *string `json:"phone"`
	IsDefault    *bool   `json:"is_default"`
}

// GetUserAddresses retrieves all addresses for a user
func (s *AddressService) GetUserAddresses(userID uint, addressType string) ([]Address, error) {
	var addresses []Address

	query := s.db.Where("user_id = ?", userID)

	// Filter by type if specified
	if addressType != "" {
		query = query.Where("type = ?", addressType)
	}

	// Order by default first, then by creation date
	if err := query.Order("is_default DESC, created_at DESC").Find(&addresses).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve addresses: %w", err)
	}

	return addresses, nil
}

// GetAddress retrieves a specific address for a user
func (s *AddressService) GetAddress(userID, addressID uint) (*Address, error) {
	var address Address
	result := s.db.Where("id = ? AND user_id = ?", addressID, userID).First(&address)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("address not found")
		}
		return nil, fmt.Errorf("failed to retrieve address: %w", result.Error)
	}

	return &address, nil
}

// CreateAddress creates a new address for a user
func (s *AddressService) CreateAddress(userID uint, req *CreateAddressRequest) (*Address, error) {
	// Validate country code
	if err := s.validateCountryCode(req.Country); err != nil {
		return nil, err
	}

	// Start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// If this is set as default, unset other defaults of the same type
	if req.IsDefault {
		if err := s.unsetDefaultAddresses(tx, userID, req.Type); err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// Create address
	address := Address{
		UserID:       userID,
		Type:         req.Type,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Company:      req.Company,
		AddressLine1: req.AddressLine1,
		AddressLine2: req.AddressLine2,
		City:         req.City,
		State:        req.State,
		PostalCode:   req.PostalCode,
		Country:      strings.ToUpper(req.Country),
		Phone:        req.Phone,
		IsDefault:    req.IsDefault,
	}

	if err := tx.Create(&address).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create address: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &address, nil
}

// UpdateAddress updates an existing address
func (s *AddressService) UpdateAddress(userID, addressID uint, req *UpdateAddressRequest) (*Address, error) {
	// Get existing address
	address, err := s.GetAddress(userID, addressID)
	if err != nil {
		return nil, err
	}

	// Validate country code if provided
	if req.Country != nil {
		if err := s.validateCountryCode(*req.Country); err != nil {
			return nil, err
		}
	}

	// Start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// If setting as default, unset other defaults of the same type
	if req.IsDefault != nil && *req.IsDefault {
		addressType := address.Type
		if req.Type != nil {
			addressType = *req.Type
		}
		if err := s.unsetDefaultAddresses(tx, userID, addressType); err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// Build updates map
	updates := make(map[string]interface{})

	if req.Type != nil {
		updates["type"] = *req.Type
	}
	if req.FirstName != nil {
		updates["first_name"] = *req.FirstName
	}
	if req.LastName != nil {
		updates["last_name"] = *req.LastName
	}
	if req.Company != nil {
		updates["company"] = *req.Company
	}
	if req.AddressLine1 != nil {
		updates["address_line1"] = *req.AddressLine1
	}
	if req.AddressLine2 != nil {
		updates["address_line2"] = *req.AddressLine2
	}
	if req.City != nil {
		updates["city"] = *req.City
	}
	if req.State != nil {
		updates["state"] = *req.State
	}
	if req.PostalCode != nil {
		updates["postal_code"] = *req.PostalCode
	}
	if req.Country != nil {
		updates["country"] = strings.ToUpper(*req.Country)
	}
	if req.Phone != nil {
		updates["phone"] = *req.Phone
	}
	if req.IsDefault != nil {
		updates["is_default"] = *req.IsDefault
	}

	// Update address
	if err := tx.Model(address).Updates(updates).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to update address: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Reload address with updates
	updatedAddress, err := s.GetAddress(userID, addressID)
	if err != nil {
		return nil, err
	}

	return updatedAddress, nil
}

// DeleteAddress deletes an address
func (s *AddressService) DeleteAddress(userID, addressID uint) error {
	// Check if address exists and belongs to user
	_, err := s.GetAddress(userID, addressID)
	if err != nil {
		return err
	}

	// Delete address
	result := s.db.Where("id = ? AND user_id = ?", addressID, userID).Delete(&Address{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete address: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("address not found")
	}

	return nil
}

// SetDefaultAddress sets an address as default for a specific type
func (s *AddressService) SetDefaultAddress(userID, addressID uint, addressType string) error {
	// Validate address type
	if addressType != "shipping" && addressType != "billing" {
		return fmt.Errorf("invalid address type. Must be 'shipping' or 'billing'")
	}

	// Check if address exists and belongs to user
	address, err := s.GetAddress(userID, addressID)
	if err != nil {
		return err
	}

	// Start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Unset all default addresses of this type for the user
	if err := s.unsetDefaultAddresses(tx, userID, addressType); err != nil {
		tx.Rollback()
		return err
	}

	// Set this address as default and update its type if necessary
	updates := map[string]interface{}{
		"is_default": true,
		"type":       addressType,
	}

	if err := tx.Model(address).Updates(updates).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to set default address: %w", err)
	}

	return tx.Commit().Error
}

// GetDefaultAddress gets the default address for a user and type
func (s *AddressService) GetDefaultAddress(userID uint, addressType string) (*Address, error) {
	var address Address
	result := s.db.Where("user_id = ? AND type = ? AND is_default = ?", userID, addressType, true).First(&address)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no default %s address found", addressType)
		}
		return nil, fmt.Errorf("failed to retrieve default address: %w", result.Error)
	}

	return &address, nil
}

// Private helper methods

// unsetDefaultAddresses removes default flag from all addresses of a specific type
func (s *AddressService) unsetDefaultAddresses(tx *gorm.DB, userID uint, addressType string) error {
	return tx.Model(&Address{}).
		Where("user_id = ? AND type = ? AND is_default = ?", userID, addressType, true).
		Update("is_default", false).Error
}

// validateCountryCode validates ISO 2-letter country code
func (s *AddressService) validateCountryCode(countryCode string) error {
	// List of valid ISO 3166-1 alpha-2 country codes (sample - you can expand this)
	validCountries := map[string]bool{
		"IN": true, // India
		"US": true, // United States
		"GB": true, // United Kingdom
		"CA": true, // Canada
		"AU": true, // Australia
		"DE": true, // Germany
		"FR": true, // France
		"JP": true, // Japan
		"SG": true, // Singapore
		"AE": true, // United Arab Emirates
		// Add more countries as needed
	}

	upperCode := strings.ToUpper(countryCode)
	if !validCountries[upperCode] {
		return fmt.Errorf("invalid country code: %s", countryCode)
	}

	return nil
}

// ValidateAddress validates address completeness for orders
func (s *AddressService) ValidateAddress(address *Address) error {
	if address.FirstName == "" {
		return fmt.Errorf("first name is required")
	}
	if address.LastName == "" {
		return fmt.Errorf("last name is required")
	}
	if address.AddressLine1 == "" {
		return fmt.Errorf("address line 1 is required")
	}
	if address.City == "" {
		return fmt.Errorf("city is required")
	}
	if address.State == "" {
		return fmt.Errorf("state is required")
	}
	if address.PostalCode == "" {
		return fmt.Errorf("postal code is required")
	}
	if address.Country == "" {
		return fmt.Errorf("country is required")
	}

	return s.validateCountryCode(address.Country)
}

// GetAddressCount returns the number of addresses for a user
func (s *AddressService) GetAddressCount(userID uint) (int64, error) {
	var count int64
	err := s.db.Model(&Address{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

// GetAddressesByType returns addresses filtered by type
func (s *AddressService) GetAddressesByType(userID uint, addressType string) ([]Address, error) {
	var addresses []Address
	err := s.db.Where("user_id = ? AND type = ?", userID, addressType).
		Order("is_default DESC, created_at DESC").
		Find(&addresses).Error

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve %s addresses: %w", addressType, err)
	}

	return addresses, nil
}
