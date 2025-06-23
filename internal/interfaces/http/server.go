// internal/interfaces/http/server.go - Fixed version
package http

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/your-org/ecommerce-backend/internal/config"
	"github.com/your-org/ecommerce-backend/internal/interfaces/http/middleware"
	"github.com/your-org/ecommerce-backend/internal/interfaces/http/routes"
	"gorm.io/gorm"
)

// Server represents the HTTP server
type Server struct {
	config      *config.Config
	gin         *gin.Engine
	httpServer  *http.Server
	db          *gorm.DB
	redisClient *redis.Client
}

// NewServer creates a new HTTP server instance
func NewServer(cfg *config.Config, db *gorm.DB, redisClient *redis.Client) *Server {
	return &Server{
		config:      cfg,
		db:          db,
		redisClient: redisClient,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Set Gin mode based on environment
	if s.config.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Create Gin engine
	s.gin = gin.New()

	// Setup middleware
	s.setupMiddleware()

	// Setup routes
	s.setupRoutes()

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:         ":" + s.config.Server.Port,
		Handler:      s.gin,
		ReadTimeout:  s.config.Server.ReadTimeout,
		WriteTimeout: s.config.Server.WriteTimeout,
		IdleTimeout:  s.config.Server.IdleTimeout,
	}

	log.Printf("🚀 HTTP Server starting on port %s", s.config.Server.Port)
	log.Printf("🌐 API Base URL: http://localhost:%s/api/v1", s.config.Server.Port)
	log.Printf("📊 Health Check: http://localhost:%s/health", s.config.Server.Port)

	// Start server
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	return nil
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	log.Println("🛑 Shutting down HTTP server...")

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	log.Println("✅ HTTP server stopped gracefully")
	return nil
}

// setupMiddleware configures all middleware for the server
func (s *Server) setupMiddleware() {
	// Recovery middleware - recover from panics
	s.gin.Use(gin.Recovery())

	// Custom logger middleware
	s.gin.Use(middleware.Logger(s.config))

	// Request ID middleware
	s.gin.Use(middleware.RequestID())

	// CORS middleware
	s.gin.Use(middleware.CORS(s.config))

	// Security headers middleware
	s.gin.Use(middleware.SecurityHeaders())

	// Rate limiting middleware
	s.gin.Use(middleware.RateLimit(s.config, s.redisClient))

	// Request size limit middleware
	s.gin.Use(middleware.RequestSizeLimit(10 << 20)) // 10MB limit

	// Timeout middleware
	s.gin.Use(middleware.Timeout(30 * time.Second))
}

// setupRoutes configures all routes for the server - FIXED
func (s *Server) setupRoutes() {
	// Health check endpoint (no auth required)
	s.gin.GET("/health", s.healthCheck)
	s.gin.GET("/ready", s.readinessCheck)

	// API v1 routes
	apiV1 := s.gin.Group("/api/v1")

	// Setup all routes using the consolidated function
	routes.SetupRoutes(apiV1, s.db, s.redisClient, s.config)

	// API documentation and root endpoint
	if s.config.IsDevelopment() {
		s.gin.Static("/docs", "./docs")
		s.gin.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message":     "E-commerce API",
				"version":     s.config.App.Version,
				"environment": s.config.App.Environment,
				"docs":        "/docs",
				"health":      "/health",
				"endpoints": gin.H{
					"auth":     "/api/v1/auth",
					"products": "/api/v1/products",
					"orders":   "/api/v1/orders",
					"cart":     "/api/v1/cart",
					"payment":  "/api/v1/payment",
					"webhooks": "/api/v1/webhooks",
					"admin":    "/api/v1/admin",
				},
			})
		})
	}
}

// healthCheck handles health check requests
func (s *Server) healthCheck(c *gin.Context) {
	// Check database health
	sqlDB, err := s.db.DB()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  "database connection error",
		})
		return
	}

	if err := sqlDB.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  "database ping failed",
		})
		return
	}

	// Check Redis health
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := s.redisClient.Ping(ctx).Err(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  "redis ping failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":      "healthy",
		"timestamp":   time.Now().UTC(),
		"version":     s.config.App.Version,
		"environment": s.config.App.Environment,
	})
}

// readinessCheck handles readiness check requests
func (s *Server) readinessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ready",
		"timestamp": time.Now().UTC(),
		"uptime":    time.Since(time.Now()).String(), // You'd track actual uptime
	})
}
