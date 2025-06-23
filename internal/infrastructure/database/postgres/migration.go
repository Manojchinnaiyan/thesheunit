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

	// Define all models that need migration in dependency order
	models := []interface{}{
		// User domain - Base tables
		&user.User{},
		&user.Address{},

		// Product domain - Base tables
		&product.Category{},
		&product.Brand{},
		&product.Product{},
		&product.ProductImage{},
		&product.ProductVariant{},
		&product.ProductReview{},

		// Cart domain
		&cart.CartItem{},

		// Order domain - Dependent tables
		&order.Order{},
		&order.OrderItem{},
		&order.Payment{}, // Payment table for Razorpay integration
		&order.OrderStatusHistory{},
	}

	// Run auto-migration for each model
	for _, model := range models {
		log.Printf("Migrating model: %T", model)
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
		"CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_users_email_verified ON users(email_verified)",

		// Product indexes
		"CREATE INDEX IF NOT EXISTS idx_products_category_active ON products(category_id, is_active)",
		"CREATE INDEX IF NOT EXISTS idx_products_featured ON products(is_featured, is_active)",
		"CREATE INDEX IF NOT EXISTS idx_products_price ON products(price)",
		"CREATE INDEX IF NOT EXISTS idx_products_created_at ON products(created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_products_sku ON products(sku)",
		"CREATE INDEX IF NOT EXISTS idx_products_slug ON products(slug)",

		// Category indexes
		"CREATE INDEX IF NOT EXISTS idx_categories_parent_active ON categories(parent_id, is_active)",
		"CREATE INDEX IF NOT EXISTS idx_categories_slug ON categories(slug)",
		"CREATE INDEX IF NOT EXISTS idx_categories_sort_order ON categories(sort_order)",

		// Product variant indexes
		"CREATE INDEX IF NOT EXISTS idx_product_variants_product_active ON product_variants(product_id, is_active)",
		"CREATE INDEX IF NOT EXISTS idx_product_variants_sku ON product_variants(sku)",

		// Product image indexes
		"CREATE INDEX IF NOT EXISTS idx_product_images_product_primary ON product_images(product_id, is_primary)",
		"CREATE INDEX IF NOT EXISTS idx_product_images_sort_order ON product_images(product_id, sort_order)",

		// Cart indexes
		"CREATE INDEX IF NOT EXISTS idx_cart_items_user_product ON cart_items(user_id, product_id)",
		"CREATE INDEX IF NOT EXISTS idx_cart_items_user_variant ON cart_items(user_id, product_variant_id)",
		"CREATE INDEX IF NOT EXISTS idx_cart_items_created_at ON cart_items(created_at DESC)",

		// Order indexes
		"CREATE INDEX IF NOT EXISTS idx_orders_user_status ON orders(user_id, status)",
		"CREATE INDEX IF NOT EXISTS idx_orders_status_created ON orders(status, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_orders_payment_status ON orders(payment_status)",
		"CREATE INDEX IF NOT EXISTS idx_orders_order_number ON orders(order_number)",
		"CREATE INDEX IF NOT EXISTS idx_orders_email ON orders(email)",
		"CREATE INDEX IF NOT EXISTS idx_orders_total_amount ON orders(total_amount)",
		"CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at DESC)",

		// Order items indexes
		"CREATE INDEX IF NOT EXISTS idx_order_items_order ON order_items(order_id)",
		"CREATE INDEX IF NOT EXISTS idx_order_items_product ON order_items(product_id)",
		"CREATE INDEX IF NOT EXISTS idx_order_items_variant ON order_items(product_variant_id)",

		// Payment indexes - CRITICAL FOR PAYMENT INTEGRATION
		"CREATE INDEX IF NOT EXISTS idx_payments_order_id ON payments(order_id)",
		"CREATE INDEX IF NOT EXISTS idx_payments_provider_id ON payments(payment_provider_id)",
		"CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status)",
		"CREATE INDEX IF NOT EXISTS idx_payments_method ON payments(payment_method)",
		"CREATE INDEX IF NOT EXISTS idx_payments_gateway ON payments(gateway)",
		"CREATE INDEX IF NOT EXISTS idx_payments_order_status ON payments(order_id, status)",
		"CREATE INDEX IF NOT EXISTS idx_payments_method_status ON payments(payment_method, status)",
		"CREATE INDEX IF NOT EXISTS idx_payments_created_at ON payments(created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_payments_processed_at ON payments(processed_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_payments_amount ON payments(amount)",

		// Order status history indexes
		"CREATE INDEX IF NOT EXISTS idx_order_status_history_order ON order_status_history(order_id, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_order_status_history_status ON order_status_history(status)",
		"CREATE INDEX IF NOT EXISTS idx_order_status_history_created_by ON order_status_history(created_by)",

		// Address indexes
		"CREATE INDEX IF NOT EXISTS idx_addresses_user_type ON addresses(user_id, type)",
		"CREATE INDEX IF NOT EXISTS idx_addresses_user_default ON addresses(user_id, is_default)",

		// Product review indexes
		"CREATE INDEX IF NOT EXISTS idx_product_reviews_product ON product_reviews(product_id, is_approved)",
		"CREATE INDEX IF NOT EXISTS idx_product_reviews_user ON product_reviews(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_product_reviews_rating ON product_reviews(rating)",
	}

	successCount := 0
	failCount := 0

	for _, indexSQL := range indexes {
		if err := m.db.Exec(indexSQL).Error; err != nil {
			log.Printf("⚠️ Failed to create index: %v", err)
			failCount++
		} else {
			successCount++
		}
	}

	log.Printf("✅ Created %d indexes successfully (%d failed)", successCount, failCount)
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

	// Create test user for development
	if err := m.seedTestUser(); err != nil {
		return fmt.Errorf("failed to seed test user: %w", err)
	}

	// Seed test products for payment testing
	if err := m.seedTestProducts(); err != nil {
		return fmt.Errorf("failed to seed test products: %w", err)
	}

	log.Println("✅ Initial data seeded successfully")
	return nil
}

// seedCategories creates default product categories
func (m *Migration) seedCategories() error {
	log.Println("🏷️ Seeding categories...")

	categories := []product.Category{
		{
			Name:        "Electronics",
			Slug:        "electronics",
			Description: "Electronic devices, gadgets, and accessories",
			SortOrder:   1,
			IsActive:    true,
		},
		{
			Name:        "Clothing",
			Slug:        "clothing",
			Description: "Fashion, apparel, and accessories",
			SortOrder:   2,
			IsActive:    true,
		},
		{
			Name:        "Books",
			Slug:        "books",
			Description: "Books, eBooks, and educational materials",
			SortOrder:   3,
			IsActive:    true,
		},
		{
			Name:        "Home & Garden",
			Slug:        "home-garden",
			Description: "Home improvement, furniture, and garden supplies",
			SortOrder:   4,
			IsActive:    true,
		},
		{
			Name:        "Sports & Outdoors",
			Slug:        "sports-outdoors",
			Description: "Sports equipment, outdoor gear, and fitness products",
			SortOrder:   5,
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
			log.Printf("✅ Created category: %s", category.Name)
		} else {
			log.Printf("⏭️ Category already exists: %s", category.Name)
		}
	}

	return nil
}

// seedAdminUser creates default admin user
func (m *Migration) seedAdminUser() error {
	log.Println("👤 Seeding admin user...")

	var existing user.User
	result := m.db.Where("email = ?", "admin@example.com").First(&existing)
	if result.Error != nil {
		// Password hash for "admin123" (bcrypt cost 12)
		hashedPassword := "$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj.6PBVdCGCq"

		adminUser := user.User{
			Email:         "admin@example.com",
			Password:      hashedPassword,
			FirstName:     "Admin",
			LastName:      "User",
			IsActive:      true,
			IsAdmin:       true,
			EmailVerified: true,
		}

		if err := m.db.Create(&adminUser).Error; err != nil {
			return err
		}

		log.Println("✅ Created admin user: admin@example.com (password: admin123)")
	} else {
		log.Println("⏭️ Admin user already exists")
	}

	return nil
}

// seedTestUser creates test user for development
func (m *Migration) seedTestUser() error {
	log.Println("👤 Seeding test user...")

	var existing user.User
	result := m.db.Where("email = ?", "test1@example.com").First(&existing)
	if result.Error != nil {
		// Password hash for "SecurePass1!!" (bcrypt cost 12)
		hashedPassword := "$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj.6PBVdCGCq"

		testUser := user.User{
			Email:         "test1@example.com",
			Password:      hashedPassword,
			FirstName:     "Test",
			LastName:      "User",
			Phone:         "+919876543210",
			IsActive:      true,
			IsAdmin:       false,
			EmailVerified: true,
		}

		if err := m.db.Create(&testUser).Error; err != nil {
			return err
		}

		log.Println("✅ Created test user: test1@example.com (password: SecurePass1!!)")
	} else {
		log.Println("⏭️ Test user already exists")
	}

	return nil
}

// seedTestProducts creates test products for payment integration testing
func (m *Migration) seedTestProducts() error {
	log.Println("🛍️ Seeding test products...")

	// Check if we already have products
	var productCount int64
	m.db.Model(&product.Product{}).Count(&productCount)

	if productCount >= 3 {
		log.Println("⏭️ Test products already exist")
		return nil
	}

	testProducts := []product.Product{
		{
			SKU:               "PAY-TEST-001",
			Name:              "Premium Gaming Laptop",
			Slug:              "premium-gaming-laptop",
			Description:       "High-performance gaming laptop with latest processors, dedicated graphics, and premium build quality. Perfect for gaming, content creation, and professional work.",
			ShortDesc:         "High-performance gaming laptop with dedicated graphics",
			Price:             199999, // ₹1999.99
			ComparePrice:      249999, // ₹2499.99
			CostPrice:         150000, // ₹1500.00
			CategoryID:        1,      // Electronics
			Weight:            2500,   // 2.5 kg
			Dimensions:        "35x25x2",
			IsActive:          true,
			IsFeatured:        true,
			IsDigital:         false,
			RequiresShipping:  true,
			TrackQuantity:     true,
			Quantity:          25,
			LowStockThreshold: 5,
			SeoTitle:          "Premium Gaming Laptop - High Performance",
			SeoDescription:    "Buy the best gaming laptop with latest technology and premium features",
			Tags:              "gaming,laptop,computer,electronics,high-performance",
		},
		{
			SKU:               "PAY-TEST-002",
			Name:              "Wireless Gaming Mouse",
			Slug:              "wireless-gaming-mouse",
			Description:       "Ergonomic wireless gaming mouse with high-precision sensor, customizable buttons, and RGB lighting. Designed for competitive gaming and long-duration use.",
			ShortDesc:         "Wireless gaming mouse with precision sensor and RGB lighting",
			Price:             7999, // ₹79.99
			ComparePrice:      9999, // ₹99.99
			CostPrice:         5000, // ₹50.00
			CategoryID:        1,    // Electronics
			Weight:            120,  // 120g
			Dimensions:        "12x6x4",
			IsActive:          true,
			IsFeatured:        false,
			IsDigital:         false,
			RequiresShipping:  true,
			TrackQuantity:     true,
			Quantity:          50,
			LowStockThreshold: 10,
			SeoTitle:          "Wireless Gaming Mouse - Precision Control",
			SeoDescription:    "Professional wireless gaming mouse with customizable features",
			Tags:              "gaming,mouse,wireless,computer,accessories",
		},
		{
			SKU:               "PAY-TEST-003",
			Name:              "Bluetooth Noise-Cancelling Headphones",
			Slug:              "bluetooth-noise-cancelling-headphones",
			Description:       "Premium wireless headphones with active noise cancellation, superior sound quality, and long battery life. Perfect for music, calls, and immersive audio experience.",
			ShortDesc:         "Premium wireless headphones with active noise cancellation",
			Price:             15999, // ₹159.99
			ComparePrice:      19999, // ₹199.99
			CostPrice:         12000, // ₹120.00
			CategoryID:        1,     // Electronics
			Weight:            300,   // 300g
			Dimensions:        "18x16x8",
			IsActive:          true,
			IsFeatured:        true,
			IsDigital:         false,
			RequiresShipping:  true,
			TrackQuantity:     true,
			Quantity:          30,
			LowStockThreshold: 8,
			SeoTitle:          "Bluetooth Noise-Cancelling Headphones - Premium Audio",
			SeoDescription:    "Experience superior sound quality with our premium wireless headphones",
			Tags:              "headphones,bluetooth,wireless,audio,music,noise-cancelling",
		},
	}

	for _, prod := range testProducts {
		var existing product.Product
		result := m.db.Where("sku = ?", prod.SKU).First(&existing)
		if result.Error != nil {
			if err := m.db.Create(&prod).Error; err != nil {
				log.Printf("⚠️ Failed to create test product %s: %v", prod.SKU, err)
			} else {
				log.Printf("✅ Created test product: %s", prod.Name)
			}
		} else {
			log.Printf("⏭️ Product already exists: %s", prod.Name)
		}
	}

	return nil
}

// DropAllTables drops all tables (use with extreme caution)
func (m *Migration) DropAllTables() error {
	log.Println("⚠️ WARNING: Dropping all database tables...")

	// Define tables in reverse dependency order
	tables := []string{
		"order_status_history",
		"payments", // Payment table for Razorpay integration
		"order_items",
		"orders",
		"cart_items",
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
			log.Printf("⚠️ Failed to drop table %s: %v", table, err)
		} else {
			log.Printf("🗑️ Dropped table: %s", table)
		}
	}

	log.Println("✅ All tables dropped successfully")
	return nil
}

// GetTableInfo returns information about database tables
func (m *Migration) GetTableInfo() error {
	var tables []string

	// Get list of tables
	if err := m.db.Raw("SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename").Scan(&tables).Error; err != nil {
		return err
	}

	log.Println("📊 Database Tables Information:")
	log.Println("================================")

	totalRecords := int64(0)
	for _, table := range tables {
		var count int64
		m.db.Table(table).Count(&count)
		totalRecords += count

		status := "✅"
		if count == 0 {
			status = "📭"
		}

		log.Printf("%s %-25s | %d records", status, table, count)
	}

	log.Println("================================")
	log.Printf("📈 Total records across all tables: %d", totalRecords)
	log.Printf("🗂️ Total tables: %d", len(tables))

	return nil
}

// CleanupTestData removes test data (useful for production setup)
func (m *Migration) CleanupTestData() error {
	log.Println("🧹 Cleaning up test data...")

	// Remove test products
	result := m.db.Where("sku LIKE ?", "PAY-TEST-%").Delete(&product.Product{})
	log.Printf("🗑️ Removed %d test products", result.RowsAffected)

	// Remove test user (keep admin)
	result = m.db.Where("email = ? AND is_admin = ?", "test1@example.com", false).Delete(&user.User{})
	log.Printf("🗑️ Removed %d test users", result.RowsAffected)

	log.Println("✅ Test data cleanup completed")
	return nil
}

// VerifyPaymentIntegration verifies that payment tables are properly set up
func (m *Migration) VerifyPaymentIntegration() error {
	log.Println("🔍 Verifying payment integration setup...")

	// Check if payment table exists and has correct structure
	var payment order.Payment
	if err := m.db.First(&payment).Error; err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("payment table verification failed: %w", err)
	}

	// Check for required indexes
	requiredIndexes := []string{
		"idx_payments_order_id",
		"idx_payments_provider_id",
		"idx_payments_status",
	}

	for _, indexName := range requiredIndexes {
		var exists bool
		query := `SELECT EXISTS (
			SELECT 1 FROM pg_indexes 
			WHERE indexname = ?
		)`
		if err := m.db.Raw(query, indexName).Scan(&exists).Error; err != nil {
			log.Printf("⚠️ Could not verify index %s: %v", indexName, err)
		} else if exists {
			log.Printf("✅ Index verified: %s", indexName)
		} else {
			log.Printf("❌ Missing index: %s", indexName)
		}
	}

	log.Println("✅ Payment integration verification completed")
	return nil
}
