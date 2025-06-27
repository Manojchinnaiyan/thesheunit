// internal/domain/user/admin_service.go
package user

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/your-org/ecommerce-backend/internal/config"
	"gorm.io/gorm"
)

// AdminService handles admin user management operations
type AdminService struct {
	db     *gorm.DB
	config *config.Config
}

// NewAdminService creates a new admin user service
func NewAdminService(db *gorm.DB, cfg *config.Config) *AdminService {
	return &AdminService{
		db:     db,
		config: cfg,
	}
}

// UserListRequest represents user list query parameters
type UserListRequest struct {
	Page          int    `form:"page,default=1"`
	Limit         int    `form:"limit,default=20"`
	Search        string `form:"search"`
	Status        string `form:"status"` // active, inactive, all
	Role          string `form:"role"`   // admin, user, all
	SortBy        string `form:"sort_by,default=created_at"`
	SortOrder     string `form:"sort_order,default=desc"`
	DateFrom      string `form:"date_from"`
	DateTo        string `form:"date_to"`
	EmailVerified *bool  `form:"email_verified"`
}

// UserListResponse represents user list with pagination
type UserListResponse struct {
	Users      []UserWithStats `json:"users"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	Limit      int             `json:"limit"`
	TotalPages int             `json:"total_pages"`
}

// UserWithStats represents user with additional statistics
type UserWithStats struct {
	User
	OrderCount   int64      `json:"order_count"`
	TotalSpent   int64      `json:"total_spent"` // In cents
	LastOrderAt  *time.Time `json:"last_order_at"`
	AddressCount int        `json:"address_count"`
}

// UserStatusUpdateRequest represents user status update data
type UserStatusUpdateRequest struct {
	IsActive bool   `json:"is_active" binding:"required"`
	Reason   string `json:"reason,omitempty"`
}

// UserAdminToggleRequest represents admin status toggle data
type UserAdminToggleRequest struct {
	IsAdmin bool   `json:"is_admin" binding:"required"`
	Reason  string `json:"reason,omitempty"`
}

// UserExportRequest represents user export parameters
type UserExportRequest struct {
	Format        string `form:"format,default=csv"` // csv, json
	Status        string `form:"status"`
	Role          string `form:"role"`
	DateFrom      string `form:"date_from"`
	DateTo        string `form:"date_to"`
	EmailVerified *bool  `form:"email_verified"`
	IncludeStats  bool   `form:"include_stats,default=false"`
}

// GetUsers retrieves users with filtering and pagination
func (s *AdminService) GetUsers(req *UserListRequest) (*UserListResponse, error) {
	var users []User
	var total int64

	// Build base query
	query := s.db.Model(&User{})

	// Apply filters
	if req.Search != "" {
		searchTerm := "%" + strings.ToLower(req.Search) + "%"
		query = query.Where(
			"LOWER(email) LIKE ? OR LOWER(first_name) LIKE ? OR LOWER(last_name) LIKE ? OR phone LIKE ?",
			searchTerm, searchTerm, searchTerm, "%"+req.Search+"%",
		)
	}

	if req.Status != "" && req.Status != "all" {
		if req.Status == "active" {
			query = query.Where("is_active = ?", true)
		} else if req.Status == "inactive" {
			query = query.Where("is_active = ?", false)
		}
	}

	if req.Role != "" && req.Role != "all" {
		if req.Role == "admin" {
			query = query.Where("is_admin = ?", true)
		} else if req.Role == "user" {
			query = query.Where("is_admin = ?", false)
		}
	}

	if req.EmailVerified != nil {
		query = query.Where("email_verified = ?", *req.EmailVerified)
	}

	// Date range filter
	if req.DateFrom != "" {
		if dateFrom, err := time.Parse("2006-01-02", req.DateFrom); err == nil {
			query = query.Where("created_at >= ?", dateFrom)
		}
	}

	if req.DateTo != "" {
		if dateTo, err := time.Parse("2006-01-02", req.DateTo); err == nil {
			query = query.Where("created_at <= ?", dateTo.Add(24*time.Hour-time.Second))
		}
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	// Apply sorting
	orderClause := req.SortBy
	if req.SortOrder == "desc" {
		orderClause += " DESC"
	} else {
		orderClause += " ASC"
	}
	query = query.Order(orderClause)

	// Apply pagination
	offset := (req.Page - 1) * req.Limit
	if err := query.Offset(offset).Limit(req.Limit).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve users: %w", err)
	}

	// Get additional stats for each user
	var usersWithStats []UserWithStats
	for _, user := range users {
		userStats, err := s.getUserStats(user.ID)
		if err != nil {
			// Use defaults if stats retrieval fails
			userStats = &UserWithStats{User: user}
		} else {
			userStats.User = user
		}

		// Clear password from response
		userStats.User.Password = ""
		usersWithStats = append(usersWithStats, *userStats)
	}

	totalPages := int((total + int64(req.Limit) - 1) / int64(req.Limit))

	return &UserListResponse{
		Users:      usersWithStats,
		Total:      total,
		Page:       req.Page,
		Limit:      req.Limit,
		TotalPages: totalPages,
	}, nil
}

// GetUser retrieves a single user by ID with stats
func (s *AdminService) GetUser(userID uint) (*UserWithStats, error) {
	var user User
	if err := s.db.Preload("Addresses").First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found")
	}

	userStats, err := s.getUserStats(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}

	userStats.User = user
	userStats.User.Password = "" // Clear password

	return userStats, nil
}

// UpdateUserStatus updates user active status
func (s *AdminService) UpdateUserStatus(userID uint, req *UserStatusUpdateRequest, adminID uint) error {
	// Check if user exists
	var user User
	if err := s.db.First(&user, userID).Error; err != nil {
		return fmt.Errorf("user not found")
	}

	// Prevent admin from deactivating themselves
	if userID == adminID && !req.IsActive {
		return fmt.Errorf("cannot deactivate your own account")
	}

	// Update user status
	updates := map[string]interface{}{
		"is_active":  req.IsActive,
		"updated_at": time.Now(),
	}

	if err := s.db.Model(&user).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update user status: %w", err)
	}

	return nil
}

// ToggleUserAdmin toggles user admin status
func (s *AdminService) ToggleUserAdmin(userID uint, req *UserAdminToggleRequest, adminID uint) error {
	// Check if user exists
	var user User
	if err := s.db.First(&user, userID).Error; err != nil {
		return fmt.Errorf("user not found")
	}

	// Prevent admin from removing their own admin privileges
	if userID == adminID && !req.IsAdmin {
		return fmt.Errorf("cannot remove your own admin privileges")
	}

	// Check if there will be at least one admin left
	if !req.IsAdmin {
		var adminCount int64
		s.db.Model(&User{}).Where("is_admin = ? AND id != ?", true, userID).Count(&adminCount)
		if adminCount == 0 {
			return fmt.Errorf("cannot remove admin privileges: at least one admin must remain")
		}
	}

	// Update admin status
	updates := map[string]interface{}{
		"is_admin":   req.IsAdmin,
		"updated_at": time.Now(),
	}

	if err := s.db.Model(&user).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update admin status: %w", err)
	}

	return nil
}

// ExportUsers exports users data
func (s *AdminService) ExportUsers(req *UserExportRequest) ([]byte, string, error) {
	// Build query with filters
	query := s.db.Model(&User{})

	if req.Status != "" && req.Status != "all" {
		if req.Status == "active" {
			query = query.Where("is_active = ?", true)
		} else if req.Status == "inactive" {
			query = query.Where("is_active = ?", false)
		}
	}

	if req.Role != "" && req.Role != "all" {
		if req.Role == "admin" {
			query = query.Where("is_admin = ?", true)
		} else if req.Role == "user" {
			query = query.Where("is_admin = ?", false)
		}
	}

	if req.EmailVerified != nil {
		query = query.Where("email_verified = ?", *req.EmailVerified)
	}

	// Date range filter
	if req.DateFrom != "" {
		if dateFrom, err := time.Parse("2006-01-02", req.DateFrom); err == nil {
			query = query.Where("created_at >= ?", dateFrom)
		}
	}

	if req.DateTo != "" {
		if dateTo, err := time.Parse("2006-01-02", req.DateTo); err == nil {
			query = query.Where("created_at <= ?", dateTo.Add(24*time.Hour-time.Second))
		}
	}

	// Get users
	var users []User
	if err := query.Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, "", fmt.Errorf("failed to retrieve users for export: %w", err)
	}

	// Generate export based on format
	switch req.Format {
	case "csv":
		return s.generateCSVExport(users, req.IncludeStats)
	case "json":
		return s.generateJSONExport(users, req.IncludeStats)
	default:
		return nil, "", fmt.Errorf("unsupported export format: %s", req.Format)
	}
}

// getUserStats gets additional statistics for a user
func (s *AdminService) getUserStats(userID uint) (*UserWithStats, error) {
	stats := &UserWithStats{}

	// Get order count and total spent
	type OrderStats struct {
		OrderCount  int64
		TotalSpent  int64
		LastOrderAt *time.Time
	}

	var orderStats OrderStats
	err := s.db.Raw(`
		SELECT 
			COUNT(*) as order_count,
			COALESCE(SUM(total_amount), 0) as total_spent,
			MAX(created_at) as last_order_at
		FROM orders 
		WHERE user_id = ? AND status != 'cancelled'
	`, userID).Scan(&orderStats).Error

	if err != nil {
		// If orders table doesn't exist or query fails, use defaults
		orderStats = OrderStats{OrderCount: 0, TotalSpent: 0, LastOrderAt: nil}
	}

	stats.OrderCount = orderStats.OrderCount
	stats.TotalSpent = orderStats.TotalSpent
	stats.LastOrderAt = orderStats.LastOrderAt

	// Get address count
	var addressCount int64
	s.db.Model(&Address{}).Where("user_id = ?", userID).Count(&addressCount)
	stats.AddressCount = int(addressCount)

	return stats, nil
}

// generateCSVExport generates CSV export
func (s *AdminService) generateCSVExport(users []User, includeStats bool) ([]byte, string, error) {
	var records [][]string

	// CSV headers
	headers := []string{
		"ID", "Email", "First Name", "Last Name", "Phone",
		"Is Active", "Is Admin", "Email Verified", "Created At", "Last Login",
	}

	if includeStats {
		headers = append(headers, "Order Count", "Total Spent", "Address Count", "Last Order")
	}

	records = append(records, headers)

	// Add user data
	for _, user := range users {
		record := []string{
			strconv.Itoa(int(user.ID)),
			user.Email,
			user.FirstName,
			user.LastName,
			user.Phone,
			strconv.FormatBool(user.IsActive),
			strconv.FormatBool(user.IsAdmin),
			strconv.FormatBool(user.EmailVerified),
			user.CreatedAt.Format("2006-01-02 15:04:05"),
		}

		if user.LastLoginAt != nil {
			record = append(record, user.LastLoginAt.Format("2006-01-02 15:04:05"))
		} else {
			record = append(record, "Never")
		}

		if includeStats {
			stats, _ := s.getUserStats(user.ID)
			record = append(record,
				strconv.FormatInt(stats.OrderCount, 10),
				fmt.Sprintf("%.2f", float64(stats.TotalSpent)/100), // Convert cents to currency
				strconv.Itoa(stats.AddressCount),
			)

			if stats.LastOrderAt != nil {
				record = append(record, stats.LastOrderAt.Format("2006-01-02 15:04:05"))
			} else {
				record = append(record, "Never")
			}
		}

		records = append(records, record)
	}

	// Convert to CSV
	var csvData strings.Builder
	writer := csv.NewWriter(&csvData)

	for _, record := range records {
		if err := writer.Write(record); err != nil {
			return nil, "", fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, "", fmt.Errorf("failed to write CSV: %w", err)
	}

	filename := fmt.Sprintf("users_export_%s.csv", time.Now().Format("2006-01-02_15-04-05"))
	return []byte(csvData.String()), filename, nil
}

// generateJSONExport generates JSON export
func (s *AdminService) generateJSONExport(users []User, includeStats bool) ([]byte, string, error) {
	var exportData []interface{}

	for _, user := range users {
		// Clear password
		user.Password = ""

		userData := map[string]interface{}{
			"id":             user.ID,
			"email":          user.Email,
			"first_name":     user.FirstName,
			"last_name":      user.LastName,
			"phone":          user.Phone,
			"is_active":      user.IsActive,
			"is_admin":       user.IsAdmin,
			"email_verified": user.EmailVerified,
			"created_at":     user.CreatedAt,
			"last_login_at":  user.LastLoginAt,
		}

		if includeStats {
			stats, _ := s.getUserStats(user.ID)
			userData["order_count"] = stats.OrderCount
			userData["total_spent"] = float64(stats.TotalSpent) / 100 // Convert cents to currency
			userData["address_count"] = stats.AddressCount
			userData["last_order_at"] = stats.LastOrderAt
		}

		exportData = append(exportData, userData)
	}

	jsonData, err := json.MarshalIndent(map[string]interface{}{
		"exported_at": time.Now(),
		"total_users": len(users),
		"users":       exportData,
	}, "", "  ")

	if err != nil {
		return nil, "", fmt.Errorf("failed to generate JSON: %w", err)
	}

	filename := fmt.Sprintf("users_export_%s.json", time.Now().Format("2006-01-02_15-04-05"))
	return jsonData, filename, nil
}
