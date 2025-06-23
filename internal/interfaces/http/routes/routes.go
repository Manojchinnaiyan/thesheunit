// internal/interfaces/http/routes/routes.go
package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/your-org/ecommerce-backend/internal/config"
	"github.com/your-org/ecommerce-backend/internal/interfaces/http/handlers"
	"github.com/your-org/ecommerce-backend/internal/interfaces/http/middleware"
	"gorm.io/gorm"
)

// SetupAuthRoutes sets up authentication related routes
func SetupAuthRoutes(rg *gin.RouterGroup, db *gorm.DB, redisClient *redis.Client, cfg *config.Config) {
	authHandler := handlers.NewAuthHandler(db, cfg)

	auth := rg.Group("/auth")
	{
		// Public auth endpoints
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.RefreshToken)
		auth.POST("/forgot-password", authHandler.ForgotPassword)
		auth.POST("/reset-password", authHandler.ResetPassword)

		// Protected auth endpoints
		protected := auth.Group("")
		protected.Use(middleware.AuthMiddleware(cfg))
		{
			protected.POST("/logout", authHandler.Logout)
			protected.GET("/profile", authHandler.GetProfile)
			protected.PUT("/profile", authHandler.UpdateProfile)
			protected.PUT("/change-password", authHandler.ChangePassword)
			protected.GET("/verify-email", authHandler.VerifyEmail)
			protected.POST("/resend-verification", authHandler.ResendVerification)
			protected.GET("/me", authHandler.GetCurrentUser)
			protected.GET("/validate", authHandler.ValidateToken)
		}
	}
}

// SetupUserRoutes sets up user related routes
func SetupUserRoutes(rg *gin.RouterGroup, db *gorm.DB, redisClient *redis.Client, cfg *config.Config) {
	users := rg.Group("/users")
	users.Use(middleware.AuthMiddleware(cfg)) // All user routes require authentication
	{
		users.GET("/addresses", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "User addresses endpoint - Coming soon"})
		})

		users.POST("/addresses", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Create address endpoint - Coming soon"})
		})

		users.PUT("/addresses/:id", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Update address endpoint - Coming soon"})
		})

		users.DELETE("/addresses/:id", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Delete address endpoint - Coming soon"})
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

// Updated routes/routes.go - Add to SetupOrderRoutes function

// SetupOrderRoutes sets up order related routes
func SetupOrderRoutes(rg *gin.RouterGroup, db *gorm.DB, redisClient *redis.Client, cfg *config.Config) {
	cartHandler := handlers.NewCartHandler(db, redisClient, cfg)
	orderHandler := handlers.NewOrderHandler(db, redisClient, cfg)

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
		checkout.POST("", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Use POST /orders to create order from cart"})
		})

		checkout.POST("/payment", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Process payment endpoint - Coming soon"})
		})

		checkout.GET("/shipping-methods", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "Available shipping methods",
				"data": []gin.H{
					{"id": "standard", "name": "Standard Shipping", "price": 999, "days": "5-7"},
					{"id": "express", "name": "Express Shipping", "price": 1999, "days": "2-3"},
					{"id": "overnight", "name": "Overnight Shipping", "price": 2999, "days": "1"},
				},
			})
		})

		checkout.POST("/calculate-shipping", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Calculate shipping endpoint - Coming soon"})
		})

		checkout.POST("/apply-coupon", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Apply coupon endpoint - Coming soon"})
		})
	}

	// Wishlist routes
	wishlist := rg.Group("/wishlist")
	wishlist.Use(middleware.AuthMiddleware(cfg))
	{
		wishlist.GET("", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Get wishlist endpoint - Coming soon"})
		})

		wishlist.POST("/items", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Add to wishlist endpoint - Coming soon"})
		})

		wishlist.DELETE("/items/:id", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Remove from wishlist endpoint - Coming soon"})
		})
	}
}

// SetupAdminRoutes sets up admin related routes
func SetupAdminRoutes(rg *gin.RouterGroup, db *gorm.DB, redisClient *redis.Client, cfg *config.Config) {
	productHandler := handlers.NewProductHandler(db, cfg)
	categoryHandler := handlers.NewCategoryHandler(db, cfg)

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

		orders := admin.Group("/orders")
		orderHandler := handlers.NewOrderHandler(db, redisClient, cfg)
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

		// User management
		users := admin.Group("/users")
		{
			users.GET("", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Admin list users endpoint - Coming soon"})
			})

			users.GET("/:id", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Admin get user endpoint - Coming soon"})
			})

			users.PUT("/:id/status", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Admin update user status endpoint - Coming soon"})
			})

			users.PUT("/:id/admin", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Admin toggle user admin status endpoint - Coming soon"})
			})

			users.GET("/export", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Export users endpoint - Coming soon"})
			})
		}

		// Analytics and reporting
		analytics := admin.Group("/analytics")
		{
			analytics.GET("/dashboard", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Admin dashboard endpoint - Coming soon"})
			})

			analytics.GET("/sales", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Admin sales analytics endpoint - Coming soon"})
			})

			analytics.GET("/products", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Admin product analytics endpoint - Coming soon"})
			})

			analytics.GET("/customers", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Admin customer analytics endpoint - Coming soon"})
			})

			analytics.GET("/revenue", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Admin revenue analytics endpoint - Coming soon"})
			})
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
			uploads.POST("/image", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Upload image endpoint - Coming soon"})
			})

			uploads.DELETE("/image/:id", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Delete image endpoint - Coming soon"})
			})

			uploads.POST("/bulk-upload", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Bulk upload images endpoint - Coming soon"})
			})
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
}
