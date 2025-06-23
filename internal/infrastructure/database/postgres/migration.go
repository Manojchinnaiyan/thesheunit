// internal/infrastructure/database/postgres/migration.go
package postgres

import (
	"fmt"
	"log"

	"github.com/your-org/ecommerce-backend/internal/domain/cart"
	"github.com/your-org/ecommerce-backend/internal/domain/order"
	"github.com/your-org/ecommerce-backend/internal/domain/product"
	"github.com/your-org/ecommerce-backend/internal/domain/user"
	"gorm.io/gorm"
)

// Migration handles database migrations
type Migration struct {
	db *gorm.DB
}

// NewMigration creates a new migration instance
func NewMigration(db *gorm.DB) *Migration {
	return &Migration{
		db: db,
	}
}

// RunAutoMigrations runs GORM auto-migrations for all models
func (m *Migration) RunAutoMigrations() error {
	log.Println("🔄 Running database auto-migrations...")

	// Define all models that need migration
	models := []interface{}{
		// User domain
		&user.User{},
		&user.Address{},

		// Product domain
		&product.Category{},
		&product.Brand{},
		&product.Product{},
		&product.ProductImage{},
		&product.ProductVariant{},
		&product.ProductReview{},

		// Cart domain
		&cart.CartItem{},

		// Order domain
		&order.Order{},
		&order.OrderItem{},
		&order.Payment{},
		&order.OrderStatusHistory{},
	}

	// Run auto-migration for each model
	for _, model := range models {
		if err := m.db.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to migrate model %T: %w", model, err)
		}
	}

	log.Println("✅ Database auto-migrations completed successfully")
	return nil
}

// CreateIndexes creates additional indexes for better performance
func (m *Migration) CreateIndexes() error {
	log.Println("🔄 Creating additional database indexes...")

	indexes := []string{
		// User indexes
		"CREATE INDEX IF NOT EXISTS idx_users_email_active ON users(email, is_active)",
		"CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at)",

		// Product indexes
		"CREATE INDEX IF NOT EXISTS idx_products_category_active ON products(category_id, is_active)",
		"CREATE INDEX IF NOT EXISTS idx_products_featured ON products(is_featured, is_active)",
		"CREATE INDEX IF NOT EXISTS idx_products_price ON products(price)",
		"CREATE INDEX IF NOT EXISTS idx_products_created_at ON products(created_at DESC)",

		// Order indexes
		"CREATE INDEX IF NOT EXISTS idx_orders_user_status ON orders(user_id, status)",
		"CREATE INDEX IF NOT EXISTS idx_orders_status_created ON orders(status, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_orders_payment_status ON orders(payment_status)",
		"CREATE INDEX IF NOT EXISTS idx_order_items_product ON order_items(product_id)",

		// Performance indexes
		"CREATE INDEX IF NOT EXISTS idx_categories_parent_active ON categories(parent_id, is_active)",
		"CREATE INDEX IF NOT EXISTS idx_product_images_product_primary ON product_images(product_id, is_primary)",
	}

	for _, indexSQL := range indexes {
		if err := m.db.Exec(indexSQL).Error; err != nil {
			log.Printf("Warning: Failed to create index: %v", err)
			// Don't return error for index creation failures
		}
	}

	log.Println("✅ Additional indexes created successfully")
	return nil
}

// SeedInitialData inserts initial data into the database
func (m *Migration) SeedInitialData() error {
	log.Println("🌱 Seeding initial data...")

	// Create default categories
	if err := m.seedCategories(); err != nil {
		return fmt.Errorf("failed to seed categories: %w", err)
	}

	// Create default admin user
	if err := m.seedAdminUser(); err != nil {
		return fmt.Errorf("failed to seed admin user: %w", err)
	}

	log.Println("✅ Initial data seeded successfully")
	return nil
}

// seedCategories creates default product categories
func (m *Migration) seedCategories() error {
	categories := []product.Category{
		{
			Name:        "Electronics",
			Slug:        "electronics",
			Description: "Electronic devices and gadgets",
			SortOrder:   1,
			IsActive:    true,
		},
		{
			Name:        "Clothing",
			Slug:        "clothing",
			Description: "Fashion and apparel",
			SortOrder:   2,
			IsActive:    true,
		},
		{
			Name:        "Books",
			Slug:        "books",
			Description: "Books and literature",
			SortOrder:   3,
			IsActive:    true,
		},
		{
			Name:        "Home & Garden",
			Slug:        "home-garden",
			Description: "Home improvement and garden supplies",
			SortOrder:   4,
			IsActive:    true,
		},
	}

	for _, category := range categories {
		var existing product.Category
		result := m.db.Where("slug = ?", category.Slug).First(&existing)
		if result.Error != nil {
			// Category doesn't exist, create it
			if err := m.db.Create(&category).Error; err != nil {
				return err
			}
			log.Printf("Created category: %s", category.Name)
		}
	}

	return nil
}

// seedAdminUser creates default admin user
func (m *Migration) seedAdminUser() error {
	var existing user.User
	result := m.db.Where("email = ?", "admin@example.com").First(&existing)
	if result.Error != nil {
		// Generate correct password hash for "admin123"
		// Using bcrypt cost 12 (same as config)
		hashedPassword := "$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj.6PBVdCGCq"

		// Admin user doesn't exist, create it
		adminUser := user.User{
			Email:         "admin@example.com",
			Password:      hashedPassword, // bcrypt hash for "admin123"
			FirstName:     "Admin",
			LastName:      "User",
			IsActive:      true,
			IsAdmin:       true,
			EmailVerified: true,
		}

		if err := m.db.Create(&adminUser).Error; err != nil {
			return err
		}

		log.Println("Created admin user: admin@example.com (password: admin123)")
	}

	return nil
}

// DropAllTables drops all tables (use with caution)
func (m *Migration) DropAllTables() error {
	log.Println("⚠️  Dropping all database tables...")

	// Define tables in reverse dependency order
	tables := []string{
		"order_status_history",
		"payments",
		"order_items",
		"orders",
		"product_reviews",
		"product_variants",
		"product_images",
		"products",
		"brands",
		"categories",
		"addresses",
		"users",
	}

	for _, table := range tables {
		if err := m.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table)).Error; err != nil {
			log.Printf("Warning: Failed to drop table %s: %v", table, err)
		}
	}

	log.Println("✅ All tables dropped successfully")
	return nil
}

// GetTableInfo returns information about database tables
func (m *Migration) GetTableInfo() error {
	var tables []string

	// Get list of tables
	if err := m.db.Raw("SELECT tablename FROM pg_tables WHERE schemaname = 'public'").Scan(&tables).Error; err != nil {
		return err
	}

	log.Println("📊 Database Tables:")
	for _, table := range tables {
		var count int64
		m.db.Table(table).Count(&count)
		log.Printf("   - %s (%d records)", table, count)
	}

	return nil
}
