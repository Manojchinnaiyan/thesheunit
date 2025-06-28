// internal/infrastructure/database/postgres/migration.go
package postgres

import (
	"fmt"
	"log"

	"github.com/your-org/ecommerce-backend/internal/domain/cart"
	"github.com/your-org/ecommerce-backend/internal/domain/inventory"
	"github.com/your-org/ecommerce-backend/internal/domain/order"
	"github.com/your-org/ecommerce-backend/internal/domain/product"
	"github.com/your-org/ecommerce-backend/internal/domain/upload"
	"github.com/your-org/ecommerce-backend/internal/domain/user"
	"github.com/your-org/ecommerce-backend/internal/domain/wishlist"
	"golang.org/x/crypto/bcrypt"
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

		// ADD THESE INVENTORY MODELS HERE:
		&inventory.Warehouse{},
		&inventory.InventoryItem{},
		&inventory.InventoryMovement{},
		&inventory.StockAlert{},
		&inventory.StockReservation{},

		// Cart domain
		&cart.CartItem{},

		// Order domain - Dependent tables
		&order.Order{},
		&order.OrderItem{},
		&order.Payment{},
		&order.OrderStatusHistory{},

		// Upload domain
		&upload.UploadedFile{},
		&upload.FileUsage{},

		// Wishlist domain
		&wishlist.WishlistItem{},

		&product.ProductReview{},
		&product.ProductReviewImage{},
		&product.ProductReviewHelpful{},
		&product.ProductReviewReport{},
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
		"CREATE INDEX IF NOT EXISTS idx_product_reviews_product_approved ON product_reviews(product_id, is_approved)",
		"CREATE INDEX IF NOT EXISTS idx_product_reviews_user ON product_reviews(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_product_reviews_rating ON product_reviews(rating)",
		"CREATE INDEX IF NOT EXISTS idx_product_reviews_verified ON product_reviews(is_verified)",
		"CREATE INDEX IF NOT EXISTS idx_product_reviews_created_at ON product_reviews(created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_product_reviews_helpful_count ON product_reviews(helpful_count DESC)",
		"CREATE INDEX IF NOT EXISTS idx_product_reviews_order_id ON product_reviews(order_id)",

		// Review images indexes
		"CREATE INDEX IF NOT EXISTS idx_product_review_images_review ON product_review_images(review_id)",
		"CREATE INDEX IF NOT EXISTS idx_product_review_images_sort ON product_review_images(review_id, sort_order)",

		// Review helpful votes indexes
		"CREATE INDEX IF NOT EXISTS idx_product_review_helpful_review ON product_review_helpful(review_id)",
		"CREATE INDEX IF NOT EXISTS idx_product_review_helpful_user ON product_review_helpful(user_id)",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_product_review_helpful_unique ON product_review_helpful(review_id, user_id)",

		// Review reports indexes
		"CREATE INDEX IF NOT EXISTS idx_product_review_reports_review ON product_review_reports(review_id)",
		"CREATE INDEX IF NOT EXISTS idx_product_review_reports_user ON product_review_reports(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_product_review_reports_status ON product_review_reports(status)",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_product_review_reports_unique ON product_review_reports(review_id, user_id)",
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

	if err := m.seedTestReviews(); err != nil {
		return fmt.Errorf("failed to seed test reviews: %w", err)
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

// Replace your seedAdminUser function with this fixed version:

func (m *Migration) seedAdminUser() error {
	log.Println("👤 Seeding admin user...")

	var existing user.User
	result := m.db.Where("email = ?", "admin@example.com").First(&existing)
	if result.Error != nil {
		// Generate hash using bcrypt cost 10 (from your .env file)
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), 10)
		if err != nil {
			log.Printf("❌ Failed to hash password: %v", err)
			return fmt.Errorf("failed to hash password: %w", err)
		}

		adminUser := user.User{
			Email:         "admin@example.com",
			Password:      string(hashedPassword),
			FirstName:     "Admin",
			LastName:      "User",
			IsActive:      true,
			IsAdmin:       true,
			EmailVerified: true,
		}

		if err := m.db.Create(&adminUser).Error; err != nil {
			log.Printf("❌ Failed to create admin user: %v", err)
			return fmt.Errorf("failed to create admin user: %w", err)
		}

		log.Println("✅ Created admin user: admin@example.com (password: admin123)")

		// Verify the user was actually created
		var verifyUser user.User
		if err := m.db.Where("email = ?", "admin@example.com").First(&verifyUser).Error; err != nil {
			log.Printf("❌ Failed to verify admin user creation: %v", err)
			return fmt.Errorf("failed to verify admin user creation: %w", err)
		}
		log.Printf("✅ Verified admin user created with ID: %d", verifyUser.ID)

	} else {
		log.Printf("⏭️ Admin user already exists with ID: %d", existing.ID)
	}

	return nil
}

// Also fix your seedTestUser function:
func (m *Migration) seedTestUser() error {
	log.Println("👤 Seeding test user...")

	var existing user.User
	result := m.db.Where("email = ?", "test1@example.com").First(&existing)
	if result.Error != nil {
		// Generate hash using bcrypt cost 10 (from your .env file)
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("test123"), 10)
		if err != nil {
			log.Printf("❌ Failed to hash password: %v", err)
			return fmt.Errorf("failed to hash password: %w", err)
		}

		testUser := user.User{
			Email:         "test1@example.com",
			Password:      string(hashedPassword),
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

		log.Println("✅ Created test user: test1@example.com (password: test123)")
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

func (m *Migration) seedTestReviews() error {
	log.Println("⭐ Seeding test reviews...")

	// Check if reviews already exist
	var reviewCount int64
	m.db.Model(&product.ProductReview{}).Count(&reviewCount)
	if reviewCount > 0 {
		log.Println("⏭️ Test reviews already exist")
		return nil
	}

	// Get existing users and products for creating reviews
	var users []user.User
	m.db.Where("email IN (?)", []string{"admin@example.com", "test1@example.com"}).Find(&users)

	var products []product.Product
	m.db.Where("sku LIKE ?", "PAY-TEST-%").Find(&products)

	if len(users) == 0 || len(products) == 0 {
		log.Println("⚠️ No users or products found for creating reviews")
		return nil
	}

	// Sample reviews data
	testReviews := []product.ProductReview{
		{
			ProductID:    products[0].ID, // Gaming laptop
			UserID:       users[0].ID,    // Admin user
			Rating:       5,
			Title:        "Excellent gaming laptop!",
			Content:      "This laptop exceeded my expectations. The performance is outstanding for both gaming and professional work. The build quality feels premium and the display is crystal clear. I've been using it for 2 months now and it handles everything I throw at it.",
			Pros:         "Exceptional performance, excellent display quality, solid build, great cooling system",
			Cons:         "A bit heavy for travel, battery life could be better during intensive gaming",
			IsVerified:   true,
			IsApproved:   true,
			HelpfulCount: 12,
			IsReported:   false,
		},
		{
			ProductID:    products[0].ID, // Gaming laptop
			UserID:       users[1].ID,    // Test user
			Rating:       4,
			Title:        "Great value for money",
			Content:      "Solid gaming laptop with impressive specs for the price. Runs all modern games smoothly on high settings. The keyboard is comfortable for long typing sessions. Some minor heating issues during extended gaming but overall very satisfied.",
			Pros:         "Good performance-to-price ratio, comfortable keyboard, runs games smoothly",
			Cons:         "Gets warm during intensive use, trackpad could be more responsive",
			IsVerified:   true,
			IsApproved:   true,
			HelpfulCount: 8,
			IsReported:   false,
		},
		{
			ProductID:    products[1].ID, // Gaming mouse (if exists)
			UserID:       users[0].ID,    // Admin user
			Rating:       5,
			Title:        "Perfect gaming mouse",
			Content:      "This mouse is exactly what I was looking for. The sensor is incredibly precise, the ergonomics are comfortable even during long gaming sessions, and the RGB lighting adds a nice touch to my setup. Highly recommended for both gaming and productivity work.",
			Pros:         "Excellent sensor accuracy, comfortable grip, customizable RGB lighting, responsive clicks",
			Cons:         "None so far after 3 months of use",
			IsVerified:   true,
			IsApproved:   true,
			HelpfulCount: 15,
			IsReported:   false,
		},
		{
			ProductID:    products[1].ID, // Gaming mouse
			UserID:       users[1].ID,    // Test user
			Rating:       4,
			Title:        "Good mouse, minor issues",
			Content:      "Overall a good gaming mouse with nice features. The wireless connection is stable and the battery life is decent. However, the scroll wheel feels a bit loose and the side buttons could be more tactile.",
			Pros:         "Stable wireless connection, good battery life, comfortable for most hand sizes",
			Cons:         "Scroll wheel feels loose, side buttons lack tactile feedback",
			IsVerified:   false, // Not verified purchase
			IsApproved:   true,
			HelpfulCount: 3,
			IsReported:   false,
		},
	}

	// Add third product reviews if it exists
	if len(products) >= 3 {
		additionalReviews := []product.ProductReview{
			{
				ProductID:    products[2].ID, // Headphones
				UserID:       users[0].ID,    // Admin user
				Rating:       5,
				Title:        "Outstanding audio quality",
				Content:      "These headphones deliver exceptional sound quality with deep bass and crystal-clear highs. The noise cancellation works perfectly for my daily commute and the battery lasts all day. Comfortable for extended wear.",
				Pros:         "Excellent sound quality, effective noise cancellation, comfortable fit, long battery life",
				Cons:         "A bit pricey but worth the investment",
				IsVerified:   true,
				IsApproved:   true,
				HelpfulCount: 9,
				IsReported:   false,
			},
			{
				ProductID:    products[2].ID, // Headphones
				UserID:       users[1].ID,    // Test user
				Rating:       3,
				Title:        "Good but not great",
				Content:      "Decent headphones with good sound quality but I expected more from the noise cancellation feature. The build quality is solid but the ear cups get uncomfortable after 2-3 hours of use.",
				Pros:         "Good sound quality, solid build, decent battery life",
				Cons:         "Noise cancellation could be better, ear cups get uncomfortable during long use",
				IsVerified:   true,
				IsApproved:   true,
				HelpfulCount: 2,
				IsReported:   false,
			},
		}
		testReviews = append(testReviews, additionalReviews...)
	}

	// Create reviews
	createdCount := 0
	for _, review := range testReviews {
		// Check if review already exists (same user and product)
		var existing product.ProductReview
		result := m.db.Where("user_id = ? AND product_id = ?", review.UserID, review.ProductID).First(&existing)
		if result.Error != nil {
			// Review doesn't exist, create it
			if err := m.db.Create(&review).Error; err != nil {
				log.Printf("⚠️ Failed to create review for product %d by user %d: %v", review.ProductID, review.UserID, err)
			} else {
				createdCount++
				log.Printf("✅ Created review: '%s' for product %d", review.Title, review.ProductID)

				// Create some sample review images for the first review
				if createdCount == 1 {
					reviewImages := []product.ProductReviewImage{
						{
							ReviewID:  review.ID,
							ImageURL:  "https://example.com/review-images/laptop-setup.jpg",
							Caption:   "My gaming setup with the new laptop",
							SortOrder: 1,
						},
						{
							ReviewID:  review.ID,
							ImageURL:  "https://example.com/review-images/laptop-performance.jpg",
							Caption:   "Performance benchmarks",
							SortOrder: 2,
						},
					}

					for _, img := range reviewImages {
						if err := m.db.Create(&img).Error; err != nil {
							log.Printf("⚠️ Failed to create review image: %v", err)
						}
					}
				}
			}
		} else {
			log.Printf("⏭️ Review already exists for product %d by user %d", review.ProductID, review.UserID)
		}
	}

	// Create some helpful votes for the reviews
	if err := m.seedReviewHelpfulVotes(); err != nil {
		log.Printf("⚠️ Failed to seed helpful votes: %v", err)
	}

	log.Printf("✅ Created %d test reviews successfully", createdCount)
	return nil
}

// seedReviewHelpfulVotes creates sample helpful votes for reviews
func (m *Migration) seedReviewHelpfulVotes() error {
	log.Println("👍 Seeding review helpful votes...")

	// Get existing reviews and users
	var reviews []product.ProductReview
	m.db.Limit(3).Find(&reviews) // Get first 3 reviews

	var users []user.User
	m.db.Where("email IN (?)", []string{"admin@example.com", "test1@example.com"}).Find(&users)

	if len(reviews) == 0 || len(users) == 0 {
		log.Println("⚠️ No reviews or users found for creating helpful votes")
		return nil
	}

	// Create some helpful votes
	helpfulVotes := []product.ProductReviewHelpful{}

	for i, review := range reviews {
		for j, user := range users {
			// Don't let users vote on their own reviews
			if review.UserID != user.ID {
				vote := product.ProductReviewHelpful{
					ReviewID:  review.ID,
					UserID:    user.ID,
					IsHelpful: (i+j)%3 != 0, // Mix of helpful and not helpful votes
				}
				helpfulVotes = append(helpfulVotes, vote)
			}
		}
	}

	createdVotes := 0
	for _, vote := range helpfulVotes {
		// Check if vote already exists
		var existing product.ProductReviewHelpful
		result := m.db.Where("review_id = ? AND user_id = ?", vote.ReviewID, vote.UserID).First(&existing)
		if result.Error != nil {
			// Vote doesn't exist, create it
			if err := m.db.Create(&vote).Error; err != nil {
				log.Printf("⚠️ Failed to create helpful vote: %v", err)
			} else {
				createdVotes++
			}
		}
	}

	log.Printf("✅ Created %d helpful votes", createdVotes)
	return nil
}
