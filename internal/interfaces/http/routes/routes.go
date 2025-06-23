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

	products := rg.Group("/products")
	products.Use(middleware.OptionalAuthMiddleware(cfg)) // Optional auth for personalization
	{
		products.GET("", productHandler.GetProducts)
		products.GET("/:id", productHandler.GetProduct)
		products.GET("/slug/:slug", productHandler.GetProductBySlug)
		products.GET("/search", productHandler.SearchProducts)

		products.GET("/categories", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "List categories endpoint - Coming soon"})
		})

		products.GET("/categories/:id", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Get category endpoint - Coming soon"})
		})

		products.GET("/brands", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "List brands endpoint - Coming soon"})
		})
	}
}

// SetupOrderRoutes sets up order related routes
func SetupOrderRoutes(rg *gin.RouterGroup, db *gorm.DB, redisClient *redis.Client, cfg *config.Config) {
	orders := rg.Group("/orders")
	orders.Use(middleware.AuthMiddleware(cfg)) // All order routes require authentication
	{
		orders.GET("", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "List orders endpoint - Coming soon"})
		})

		orders.POST("", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Create order endpoint - Coming soon"})
		})

		orders.GET("/:id", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Get order endpoint - Coming soon"})
		})

		orders.PUT("/:id/cancel", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Cancel order endpoint - Coming soon"})
		})
	}

	// Cart routes (can work with guest sessions or authenticated users)
	cart := rg.Group("/cart")
	cart.Use(middleware.OptionalAuthMiddleware(cfg))
	{
		cart.GET("", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Get cart endpoint - Coming soon"})
		})

		cart.POST("/items", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Add to cart endpoint - Coming soon"})
		})

		cart.PUT("/items/:id", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Update cart item endpoint - Coming soon"})
		})

		cart.DELETE("/items/:id", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Remove from cart endpoint - Coming soon"})
		})

		cart.DELETE("", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Clear cart endpoint - Coming soon"})
		})
	}

	// Checkout routes require authentication
	checkout := rg.Group("/checkout")
	checkout.Use(middleware.AuthMiddleware(cfg))
	{
		checkout.POST("", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Checkout endpoint - Coming soon"})
		})

		checkout.POST("/payment", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Process payment endpoint - Coming soon"})
		})
	}
}

// SetupAdminRoutes sets up admin related routes
func SetupAdminRoutes(rg *gin.RouterGroup, db *gorm.DB, redisClient *redis.Client, cfg *config.Config) {
	productHandler := handlers.NewProductHandler(db, cfg)

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
		}

		// Order management
		orders := admin.Group("/orders")
		{
			orders.GET("", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Admin list orders endpoint - Coming soon"})
			})

			orders.PUT("/:id/status", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Admin update order status endpoint - Coming soon"})
			})
		}

		// User management
		users := admin.Group("/users")
		{
			users.GET("", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Admin list users endpoint - Coming soon"})
			})

			users.PUT("/:id/status", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Admin update user status endpoint - Coming soon"})
			})
		}

		// Analytics
		analytics := admin.Group("/analytics")
		{
			analytics.GET("/dashboard", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Admin dashboard endpoint - Coming soon"})
			})

			analytics.GET("/sales", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Admin sales analytics endpoint - Coming soon"})
			})
		}
	}
}
