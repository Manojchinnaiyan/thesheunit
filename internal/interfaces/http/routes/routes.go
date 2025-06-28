// internal/interfaces/http/routes/routes.go - Complete and Fixed
package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/your-org/ecommerce-backend/internal/config"
	"github.com/your-org/ecommerce-backend/internal/interfaces/http/handlers"
	"github.com/your-org/ecommerce-backend/internal/interfaces/http/middleware"
	"gorm.io/gorm"
)

// SetupRoutes sets up all application routes
func SetupRoutes(rg *gin.RouterGroup, db *gorm.DB, redisClient *redis.Client, cfg *config.Config) {
	// Setup all route groups
	SetupAuthRoutes(rg, db, redisClient, cfg)
	SetupUserRoutes(rg, db, redisClient, cfg)
	SetupProductRoutes(rg, db, redisClient, cfg)
	SetupOrderRoutes(rg, db, redisClient, cfg)
	SetupPaymentRoutes(rg, db, redisClient, cfg)
	SetupAdminRoutes(rg, db, redisClient, cfg)
	SetupInventoryRoutes(rg, db, redisClient, cfg)
}

// SetupInventoryRoutes sets up inventory related routes
func SetupInventoryRoutes(rg *gin.RouterGroup, db *gorm.DB, redisClient *redis.Client, cfg *config.Config) {
	inventoryHandler := handlers.NewInventoryHandler(db, cfg)

	// Public inventory endpoints
	inventory := rg.Group("/inventory")
	{
		inventory.GET("/stock-level/:productId", inventoryHandler.GetStockLevel)
		inventory.GET("/warehouses", inventoryHandler.GetWarehouses)
		inventory.GET("/warehouses/default", inventoryHandler.GetDefaultWarehouse)
	}

	// Protected inventory endpoints (require authentication)
	inventoryAuth := rg.Group("/inventory")
	inventoryAuth.Use(middleware.AuthMiddleware(cfg))
	{
		inventoryAuth.POST("/reserve", inventoryHandler.ReserveStock)
		inventoryAuth.POST("/release", inventoryHandler.ReleaseReservation)
		inventoryAuth.POST("/fulfill", inventoryHandler.FulfillReservation)
	}
}

// SetupAuthRoutes sets up authentication related routes
func SetupAuthRoutes(rg *gin.RouterGroup, db *gorm.DB, redisClient *redis.Client, cfg *config.Config) {
	authHandler := handlers.NewAuthHandler(db, redisClient, cfg)

	auth := rg.Group("/auth")
	{
		// Public auth endpoints
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.RefreshToken)
		auth.POST("/forgot-password", authHandler.ForgotPassword)
		auth.POST("/reset-password", authHandler.ResetPassword)
		auth.GET("/verify-email", authHandler.VerifyEmail)
		auth.POST("/resend-verification", authHandler.ResendVerification)

		// Protected auth endpoints
		protected := auth.Group("")
		protected.Use(middleware.AuthMiddleware(cfg))
		{
			protected.POST("/logout", authHandler.Logout)
			protected.GET("/profile", authHandler.GetProfile)
			protected.PUT("/profile", authHandler.UpdateProfile)
			protected.PUT("/change-password", authHandler.ChangePassword)
			protected.GET("/me", authHandler.GetCurrentUser)
			protected.GET("/validate", authHandler.ValidateToken)
		}
	}
}

// SetupUserRoutes sets up user related routes
func SetupUserRoutes(rg *gin.RouterGroup, db *gorm.DB, redisClient *redis.Client, cfg *config.Config) {
	userAddressHandler := handlers.NewUserAddressHandler(db, cfg)
	userProfileHandler := handlers.NewUserProfileHandler(db, cfg)
	users := rg.Group("/users")
	users.Use(middleware.AuthMiddleware(cfg)) // All user routes require authentication
	{
		addresses := users.Group("/addresses")
		{
			addresses.GET("", userAddressHandler.GetAddresses)                  // GET /users/addresses
			addresses.GET("/:id", userAddressHandler.GetAddress)                // GET /users/addresses/:id
			addresses.POST("", userAddressHandler.CreateAddress)                // POST /users/addresses
			addresses.PUT("/:id", userAddressHandler.UpdateAddress)             // PUT /users/addresses/:id
			addresses.DELETE("/:id", userAddressHandler.DeleteAddress)          // DELETE /users/addresses/:id
			addresses.PUT("/:id/default", userAddressHandler.SetDefaultAddress) // PUT /users/addresses/:id/default
		}
		users.GET("/profile", userProfileHandler.GetProfile)
		users.PUT("/profile", userProfileHandler.UpdateProfile)
		users.GET("/account", userProfileHandler.GetAccount)
		users.GET("/dashboard", userProfileHandler.GetDashboard)
		users.PUT("/change-password", userProfileHandler.ChangePassword)

		users.GET("/orders", func(c *gin.Context) {
			c.Redirect(http.StatusMovedPermanently, "/api/v1/orders")
		})
	}
}

// SetupProductRoutes sets up product related routes
func SetupProductRoutes(rg *gin.RouterGroup, db *gorm.DB, redisClient *redis.Client, cfg *config.Config) {
	productHandler := handlers.NewProductHandler(db, cfg)
	categoryHandler := handlers.NewCategoryHandler(db, cfg)

	products := rg.Group("/products")
	products.Use(middleware.OptionalAuthMiddleware(cfg)) // Optional auth for personalization
	{
		// Product endpoints
		products.GET("", productHandler.GetProducts)
		products.GET("/:id", productHandler.GetProduct)
		products.GET("/slug/:slug", productHandler.GetProductBySlug)
		products.GET("/search", productHandler.SearchProducts)

		// Category endpoints
		categories := products.Group("/categories")
		{
			categories.GET("", categoryHandler.GetCategories)
			categories.GET("/tree", categoryHandler.GetCategoryTree)
			categories.GET("/root", categoryHandler.GetRootCategories)
			categories.GET("/:id", categoryHandler.GetCategory)
			categories.GET("/slug/:slug", categoryHandler.GetCategoryBySlug)
			categories.GET("/:id/subcategories", categoryHandler.GetSubcategories)
		}

		// Brand endpoints (placeholder for future implementation)
		products.GET("/brands", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "List brands endpoint - Coming soon"})
		})

		products.GET("/brands/:id", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Get brand endpoint - Coming soon"})
		})
	}
}

// SetupOrderRoutes sets up order and cart related routes
func SetupOrderRoutes(rg *gin.RouterGroup, db *gorm.DB, redisClient *redis.Client, cfg *config.Config) {
	cartHandler := handlers.NewCartHandler(db, redisClient, cfg)
	orderHandler := handlers.NewOrderHandler(db, redisClient, cfg)
	checkoutHandler := handlers.NewCheckoutHandler(db, redisClient, cfg)
	wishlistHandler := handlers.NewWishlistHandler(db, redisClient, cfg)

	// Order routes - require authentication
	orders := rg.Group("/orders")
	orders.Use(middleware.AuthMiddleware(cfg))
	{
		// User order endpoints
		orders.POST("", orderHandler.CreateOrder)                         // Create order from cart
		orders.GET("", orderHandler.GetOrders)                            // Get user's orders
		orders.GET("/:id", orderHandler.GetOrder)                         // Get specific order
		orders.GET("/number/:orderNumber", orderHandler.GetOrderByNumber) // Get order by number
		orders.PUT("/:id/cancel", orderHandler.CancelOrder)               // Cancel order
		orders.GET("/:id/track", orderHandler.TrackOrder)                 // Track order
	}

	// Cart routes (can work with guest sessions or authenticated users)
	cart := rg.Group("/cart")
	cart.Use(middleware.OptionalAuthMiddleware(cfg))
	{
		cart.GET("", cartHandler.GetCart)
		cart.POST("/items", cartHandler.AddToCart)
		cart.PUT("/items/:id", cartHandler.UpdateCartItem)
		cart.DELETE("/items/:id", cartHandler.RemoveFromCart)
		cart.DELETE("", cartHandler.ClearCart)
		cart.GET("/count", cartHandler.GetCartCount)
		cart.POST("/validate", cartHandler.ValidateCart)

		// Cart merge endpoint (requires authentication)
		cartAuth := cart.Group("")
		cartAuth.Use(middleware.AuthMiddleware(cfg))
		{
			cartAuth.POST("/merge", cartHandler.MergeGuestCart)
		}
	}

	// Checkout routes require authentication
	checkout := rg.Group("/checkout")
	checkout.Use(middleware.AuthMiddleware(cfg))
	{
		// Main checkout endpoint
		checkout.GET("/summary", checkoutHandler.GetCheckoutSummary)
		checkout.POST("/validate", checkoutHandler.ValidateCheckout)

		// Shipping endpoints
		checkout.GET("/shipping-methods", checkoutHandler.GetShippingMethods)
		checkout.POST("/calculate-shipping", checkoutHandler.CalculateShipping)

		// Tax calculation
		checkout.POST("/calculate-tax", checkoutHandler.GetTaxCalculation)

		// Coupon management
		checkout.POST("/apply-coupon", checkoutHandler.ApplyCoupon)
		checkout.POST("/remove-coupon", checkoutHandler.RemoveCoupon)

		// Legacy endpoint redirect
		checkout.POST("", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message":  "Use POST /orders to create order from cart",
				"redirect": "/api/v1/orders",
			})
		})
	}

	// Wishlist routes
	wishlist := rg.Group("/wishlist")
	wishlist.Use(middleware.AuthMiddleware(cfg))
	{
		// Basic wishlist operations
		wishlist.GET("", wishlistHandler.GetWishlist)
		wishlist.GET("/count", wishlistHandler.GetWishlistCount)
		wishlist.GET("/summary", wishlistHandler.GetWishlistSummary)
		wishlist.DELETE("", wishlistHandler.ClearWishlist)

		// Item management
		wishlist.POST("/items", wishlistHandler.AddToWishlist)
		wishlist.DELETE("/items/:id", wishlistHandler.RemoveFromWishlist)
		wishlist.POST("/items/:id/move-to-cart", wishlistHandler.MoveToCart)

		// Bulk operations
		wishlist.POST("/bulk-add", wishlistHandler.BulkAddToWishlist)

		// Utility endpoints
		wishlist.GET("/check/:id", wishlistHandler.CheckItemInWishlist)
	}

	// Compare products (placeholder for future implementation)
	compare := rg.Group("/compare")
	compare.Use(middleware.OptionalAuthMiddleware(cfg))
	{
		compare.GET("", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "Product comparison endpoint - Coming soon",
				"data": gin.H{
					"products":          []gin.H{},
					"max_compare_items": 4,
					"comparison_attributes": []string{
						"price", "rating", "features", "specifications",
					},
				},
			})
		})

		compare.POST("/add/:id", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Add to comparison endpoint - Coming soon"})
		})

		compare.DELETE("/remove/:id", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Remove from comparison endpoint - Coming soon"})
		})

		compare.DELETE("", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Clear comparison endpoint - Coming soon"})
		})
	}

	// Recently viewed products (placeholder for future implementation)
	recentlyViewed := rg.Group("/recently-viewed")
	recentlyViewed.Use(middleware.OptionalAuthMiddleware(cfg))
	{
		recentlyViewed.GET("", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "Recently viewed products endpoint - Coming soon",
				"data": gin.H{
					"products":  []gin.H{},
					"max_items": 10,
				},
			})
		})

		recentlyViewed.POST("/add/:id", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Add to recently viewed endpoint - Coming soon"})
		})

		recentlyViewed.DELETE("", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Clear recently viewed endpoint - Coming soon"})
		})
	}
}

// SetupPaymentRoutes sets up payment related routes
func SetupPaymentRoutes(rg *gin.RouterGroup, db *gorm.DB, redisClient *redis.Client, cfg *config.Config) {
	paymentHandler := handlers.NewPaymentHandler(db, redisClient, cfg)

	// Payment routes - require authentication
	payment := rg.Group("/payment")
	payment.Use(middleware.AuthMiddleware(cfg))
	{
		// Payment initiation and verification
		payment.POST("/initiate", paymentHandler.InitiatePayment)
		payment.POST("/verify", paymentHandler.VerifyPayment)
		payment.POST("/failure", paymentHandler.HandlePaymentFailure)
		payment.GET("/status/:orderId", paymentHandler.GetPaymentStatus)
		payment.GET("/methods", paymentHandler.GetPaymentMethods)
	}

	// Public webhook endpoints (no auth required)
	webhooks := rg.Group("/webhooks")
	{
		webhooks.POST("/razorpay", paymentHandler.RazorpayWebhook)
		// Future: webhooks.POST("/stripe", paymentHandler.StripeWebhook)
	}
}

// SetupAdminRoutes sets up admin related routes
func SetupAdminRoutes(rg *gin.RouterGroup, db *gorm.DB, redisClient *redis.Client, cfg *config.Config) {
	productHandler := handlers.NewProductHandler(db, cfg)
	categoryHandler := handlers.NewCategoryHandler(db, cfg)
	orderHandler := handlers.NewOrderHandler(db, redisClient, cfg)
	paymentHandler := handlers.NewPaymentHandler(db, redisClient, cfg)
	uploadHandler := handlers.NewUploadHandler(db, cfg)
	inventoryHandler := handlers.NewInventoryHandler(db, cfg)
	userAdminHandler := handlers.NewUserAdminHandler(db, cfg)
	analyticsHandler := handlers.NewAnalyticsHandler(db, cfg)

	admin := rg.Group("/admin")
	admin.Use(middleware.AuthMiddleware(cfg)) // Require authentication
	admin.Use(middleware.AdminMiddleware())   // Require admin privileges
	{
		// Product management
		products := admin.Group("/products")
		{
			products.GET("", productHandler.AdminGetProducts)
			products.GET("/:id", productHandler.AdminGetProduct)
			products.POST("", productHandler.AdminCreateProduct)
			products.PUT("/:id", productHandler.AdminUpdateProduct)
			products.DELETE("/:id", productHandler.AdminDeleteProduct)
			products.PUT("/:id/inventory", productHandler.AdminUpdateInventory)

			// Product bulk operations
			products.POST("/bulk-update", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Bulk update products endpoint - Coming soon"})
			})

			products.POST("/bulk-delete", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Bulk delete products endpoint - Coming soon"})
			})

			products.POST("/import", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Import products endpoint - Coming soon"})
			})

			products.GET("/export", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Export products endpoint - Coming soon"})
			})
		}

		warehouses := admin.Group("/warehouses")
		{
			warehouses.POST("", inventoryHandler.CreateWarehouse)
			warehouses.GET("", inventoryHandler.GetWarehouses)
		}

		inventory := admin.Group("/inventory")
		{
			inventory.GET("/:productId/:warehouseId", inventoryHandler.GetInventoryItem)
			inventory.POST("", inventoryHandler.CreateOrUpdateInventoryItem)
			inventory.POST("/movements", inventoryHandler.RecordStockMovement)
		}

		// Category management
		categories := admin.Group("/categories")
		{
			categories.GET("", categoryHandler.AdminGetCategories)
			categories.GET("/tree", categoryHandler.AdminGetCategoryTree)
			categories.GET("/:id", categoryHandler.AdminGetCategory)
			categories.POST("", categoryHandler.AdminCreateCategory)
			categories.PUT("/:id", categoryHandler.AdminUpdateCategory)
			categories.DELETE("/:id", categoryHandler.AdminDeleteCategory)

			// Category bulk operations
			categories.POST("/reorder", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Reorder categories endpoint - Coming soon"})
			})
		}

		// Order management
		orders := admin.Group("/orders")
		{
			orders.GET("", orderHandler.AdminGetOrders)                    // List all orders
			orders.GET("/stats", orderHandler.AdminGetOrderStats)          // Order statistics
			orders.GET("/export", orderHandler.AdminExportOrders)          // Export orders
			orders.GET("/:id", orderHandler.AdminGetOrder)                 // Get specific order
			orders.PUT("/:id/status", orderHandler.AdminUpdateOrderStatus) // Update order status
			orders.PUT("/:id/cancel", orderHandler.AdminCancelOrder)       // Cancel order
			orders.POST("/:id/refund", orderHandler.AdminRefundOrder)      // Process refund

			// Bulk operations
			orders.POST("/bulk-update", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Bulk update orders endpoint - Coming soon"})
			})

			orders.POST("/bulk-export", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Bulk export orders endpoint - Coming soon"})
			})
		}

		// Payment management
		payments := admin.Group("/payments")
		{
			payments.GET("", paymentHandler.AdminGetPayments)
			payments.POST("/:paymentId/refund", paymentHandler.AdminRefundPayment)
			payments.GET("/stats", paymentHandler.AdminGetPaymentStats)
		}

		// User management
		users := admin.Group("/users")
		{
			users.GET("", userAdminHandler.GetUsers)                    // GET /admin/users
			users.GET("/export", userAdminHandler.ExportUsers)          // GET /admin/users/export
			users.GET("/:id", userAdminHandler.GetUser)                 // GET /admin/users/:id
			users.PUT("/:id/status", userAdminHandler.UpdateUserStatus) // PUT /admin/users/:id/status
			users.PUT("/:id/admin", userAdminHandler.ToggleUserAdmin)   // PUT /admin/users/:id/admin
		}

		// Brand management (placeholder)
		brands := admin.Group("/brands")
		{
			brands.GET("", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Admin list brands endpoint - Coming soon"})
			})

			brands.POST("", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Admin create brand endpoint - Coming soon"})
			})

			brands.PUT("/:id", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Admin update brand endpoint - Coming soon"})
			})

			brands.DELETE("/:id", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Admin delete brand endpoint - Coming soon"})
			})
		}

		// Analytics and reporting
		analytics := admin.Group("/analytics")
		{
			analytics.GET("/dashboard", analyticsHandler.GetDashboard) // GET /admin/analytics/dashboard
			analytics.GET("/sales", analyticsHandler.GetSales)         // GET /admin/analytics/sales
			analytics.GET("/products", analyticsHandler.GetProducts)   // GET /admin/analytics/products
			analytics.GET("/customers", analyticsHandler.GetCustomers) // GET /admin/analytics/customers
			analytics.GET("/revenue", analyticsHandler.GetRevenue)     // GET /admin/analytics/revenue
		}

		// Settings and configuration
		settings := admin.Group("/settings")
		{
			settings.GET("", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Get admin settings endpoint - Coming soon"})
			})

			settings.PUT("", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Update admin settings endpoint - Coming soon"})
			})

			settings.GET("/shipping", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Get shipping settings endpoint - Coming soon"})
			})

			settings.PUT("/shipping", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Update shipping settings endpoint - Coming soon"})
			})

			settings.GET("/payment", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Get payment settings endpoint - Coming soon"})
			})

			settings.PUT("/payment", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Update payment settings endpoint - Coming soon"})
			})
		}

		// File upload management
		uploads := admin.Group("/uploads")
		{
			// Image upload operations
			uploads.POST("/image", uploadHandler.UploadImage)
			uploads.POST("/bulk-upload", uploadHandler.UploadMultipleImages)
			uploads.GET("/images", uploadHandler.GetImages)
			uploads.GET("/image/:id", uploadHandler.GetImage)
			uploads.PUT("/image/:id", uploadHandler.UpdateImage)
			uploads.DELETE("/image/:id", uploadHandler.DeleteImage)
			uploads.POST("/image/:id/optimize", uploadHandler.OptimizeImage)

			// Upload management
			uploads.GET("/stats", uploadHandler.GetUploadStats)
			uploads.GET("/config", uploadHandler.GetUploadConfig)
		}

		// Coupons and discounts
		coupons := admin.Group("/coupons")
		{
			coupons.GET("", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "List coupons endpoint - Coming soon"})
			})

			coupons.POST("", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Create coupon endpoint - Coming soon"})
			})

			coupons.PUT("/:id", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Update coupon endpoint - Coming soon"})
			})

			coupons.DELETE("/:id", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Delete coupon endpoint - Coming soon"})
			})
		}
	}
	rg.GET("/uploads/*filepath", uploadHandler.ServeFile)
}
