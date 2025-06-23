// internal/infrastructure/database/redis/connection.go
package redis

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/your-org/ecommerce-backend/internal/config"
)

// Client wraps the Redis client
type Client struct {
	Redis *redis.Client
}

// NewConnection creates a new Redis connection
func NewConnection(cfg *config.Config) (*Client, error) {
	// Create Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.GetRedisAddr(),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,

		// Connection timeouts
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,

		// Pool timeouts
		PoolTimeout: 4 * time.Second,
	})

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Println("✅ Redis connection established successfully")

	return &Client{
		Redis: rdb,
	}, nil
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.Redis.Close()
}

// GetClient returns the Redis client instance
func (c *Client) GetClient() *redis.Client {
	return c.Redis
}

// Health checks the Redis connection health
func (c *Client) Health() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return c.Redis.Ping(ctx).Err()
}

// Set stores a key-value pair with expiration
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.Redis.Set(ctx, key, value, expiration).Err()
}

// Get retrieves a value by key
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.Redis.Get(ctx, key).Result()
}

// Del deletes one or more keys
func (c *Client) Del(ctx context.Context, keys ...string) error {
	return c.Redis.Del(ctx, keys...).Err()
}

// Exists checks if key exists
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	count, err := c.Redis.Exists(ctx, key).Result()
	return count > 0, err
}

// SetJSON stores a JSON value
func (c *Client) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.Redis.Set(ctx, key, value, expiration).Err()
}

// GetJSON retrieves a JSON value
func (c *Client) GetJSON(ctx context.Context, key string, dest interface{}) error {
	val, err := c.Redis.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	// You would unmarshal JSON here - we'll implement this later
	_ = val
	_ = dest
	return nil
}
