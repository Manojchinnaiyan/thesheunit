// cmd/api/main.go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/your-org/ecommerce-backend/internal/config"
	_ "github.com/your-org/ecommerce-backend/internal/domain/order"
	"github.com/your-org/ecommerce-backend/internal/infrastructure/database/postgres"
	"github.com/your-org/ecommerce-backend/internal/infrastructure/database/redis"
	"github.com/your-org/ecommerce-backend/internal/interfaces/http"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("🚀 Starting %s v%s in %s mode", cfg.App.Name, cfg.App.Version, cfg.App.Environment)

	// Connect to database
	db, err := postgres.NewConnection(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Connect to Redis
	redisClient, err := redis.NewConnection(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Health check
	if err := db.Health(); err != nil {
		log.Fatalf("Database health check failed: %v", err)
	}

	if err := redisClient.Health(); err != nil {
		log.Fatalf("Redis health check failed: %v", err)
	}

	// Run database migrations
	migration := postgres.NewMigration(db.GetDB())

	if err := migration.RunAutoMigrations(); err != nil {
		log.Fatalf("Database migration failed: %v", err)
	}

	if err := migration.CreateIndexes(); err != nil {
		log.Printf("Warning: Index creation failed: %v", err)
	}

	// Seed initial data in development
	if cfg.IsDevelopment() {
		if err := migration.SeedInitialData(); err != nil {
			log.Printf("Warning: Data seeding failed: %v", err)
		}
		migration.GetTableInfo()
	}

	log.Println("✅ All systems operational!")

	// Create and start HTTP server
	server := http.NewServer(cfg, db.GetDB(), redisClient.GetClient())

	// Start server in a goroutine
	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("👋 Shutting down gracefully...")

	// Give server 30 seconds to shutdown gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Stop(ctx); err != nil {
		log.Printf("Failed to shutdown HTTP server gracefully: %v", err)
	}

	log.Println("✅ Server shutdown completed")
}
