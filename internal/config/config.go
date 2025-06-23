// internal/config/config.go
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for our application
type Config struct {
	App      AppConfig
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Security SecurityConfig
	External ExternalConfig
	Upload   UploadConfig
	Logging  LoggingConfig
}

// AppConfig contains application-level configuration
type AppConfig struct {
	Name        string
	Version     string
	Environment string
	Debug       bool
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DatabaseConfig contains database connection configuration
type DatabaseConfig struct {
	Host           string
	Port           string
	Name           string
	User           string
	Password       string
	SSLMode        string
	MaxOpenConns   int
	MaxIdleConns   int
	MaxLifetime    time.Duration
	MigrationsPath string
}

// RedisConfig contains Redis configuration
type RedisConfig struct {
	Host         string
	Port         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
}

// JWTConfig contains JWT token configuration
type JWTConfig struct {
	Secret               string
	AccessTokenExpiry    time.Duration
	RefreshTokenExpiry   time.Duration
	RefreshTokenRotation bool
}

// SecurityConfig contains security-related configuration
type SecurityConfig struct {
	BcryptCost         int
	RateLimitPerMinute int
	RateLimitBurst     int
	CORSAllowedOrigins []string
	CORSAllowedMethods []string
	CORSAllowedHeaders []string
	TrustedProxies     []string
}

// ExternalConfig contains external service configurations
type ExternalConfig struct {
	Stripe  StripeConfig
	Email   EmailConfig
	Storage StorageConfig
}

// StripeConfig contains Stripe payment configuration
type StripeConfig struct {
	SecretKey      string
	PublishableKey string
	WebhookSecret  string
	Environment    string
}

// EmailConfig contains email service configuration
type EmailConfig struct {
	Provider  string
	APIKey    string
	FromEmail string
	FromName  string
	SMTPHost  string
	SMTPPort  int
	SMTPUser  string
	SMTPPass  string
}

// StorageConfig contains file storage configuration
type StorageConfig struct {
	Provider    string
	LocalPath   string
	S3Bucket    string
	S3Region    string
	S3AccessKey string
	S3SecretKey string
	CDNBaseURL  string
}

// UploadConfig contains file upload configuration
type UploadConfig struct {
	MaxSize           int64
	AllowedExtensions []string
	ImageMaxWidth     int
	ImageMaxHeight    int
	ThumbnailWidth    int
	ThumbnailHeight   int
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string
	Format string
	File   string
}

// Load loads configuration from environment variables and .env file
func Load() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found, using environment variables")
	}

	config := &Config{
		App: AppConfig{
			Name:        getEnv("APP_NAME", "E-commerce Backend"),
			Version:     getEnv("APP_VERSION", "1.0.0"),
			Environment: getEnv("APP_ENV", "development"),
			Debug:       getEnvAsBool("APP_DEBUG", true),
		},
		Server: ServerConfig{
			Port:         getEnv("APP_PORT", "8080"),
			ReadTimeout:  getEnvAsDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getEnvAsDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  getEnvAsDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},
		Database: DatabaseConfig{
			Host:           getEnv("DB_HOST", "localhost"),
			Port:           getEnv("DB_PORT", "5432"),
			Name:           getEnv("DB_NAME", "ecommerce_db"),
			User:           getEnv("DB_USER", "ecommerce_user"),
			Password:       getEnv("DB_PASSWORD", "ecommerce_password"),
			SSLMode:        getEnv("DB_SSL_MODE", "disable"),
			MaxOpenConns:   getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:   getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
			MaxLifetime:    getEnvAsDuration("DB_MAX_LIFETIME", 300*time.Second),
			MigrationsPath: getEnv("DB_MIGRATIONS_PATH", "internal/infrastructure/database/postgres/migrations"),
		},
		Redis: RedisConfig{
			Host:         getEnv("REDIS_HOST", "localhost"),
			Port:         getEnv("REDIS_PORT", "6379"),
			Password:     getEnv("REDIS_PASSWORD", ""),
			DB:           getEnvAsInt("REDIS_DB", 0),
			PoolSize:     getEnvAsInt("REDIS_POOL_SIZE", 10),
			MinIdleConns: getEnvAsInt("REDIS_MIN_IDLE_CONNS", 5),
		},
		JWT: JWTConfig{
			Secret:               getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-in-production"),
			AccessTokenExpiry:    getEnvAsDuration("JWT_ACCESS_EXPIRE", 24*time.Hour),
			RefreshTokenExpiry:   getEnvAsDuration("JWT_REFRESH_EXPIRE", 7*24*time.Hour),
			RefreshTokenRotation: getEnvAsBool("JWT_REFRESH_ROTATION", true),
		},
		Security: SecurityConfig{
			BcryptCost:         getEnvAsInt("BCRYPT_COST", 12),
			RateLimitPerMinute: getEnvAsInt("RATE_LIMIT_PER_MINUTE", 100),
			RateLimitBurst:     getEnvAsInt("RATE_LIMIT_BURST", 50),
			CORSAllowedOrigins: getEnvAsSlice("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000", "http://localhost:3001"}),
			CORSAllowedMethods: getEnvAsSlice("CORS_ALLOWED_METHODS", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
			CORSAllowedHeaders: getEnvAsSlice("CORS_ALLOWED_HEADERS", []string{"Origin", "Content-Type", "Accept", "Authorization"}),
			TrustedProxies:     getEnvAsSlice("TRUSTED_PROXIES", []string{}),
		},
		External: ExternalConfig{
			Stripe: StripeConfig{
				SecretKey:      getEnv("STRIPE_SECRET_KEY", ""),
				PublishableKey: getEnv("STRIPE_PUBLISHABLE_KEY", ""),
				WebhookSecret:  getEnv("STRIPE_WEBHOOK_SECRET", ""),
				Environment:    getEnv("STRIPE_ENVIRONMENT", "test"),
			},
			Email: EmailConfig{
				Provider:  getEnv("EMAIL_PROVIDER", "sendgrid"),
				APIKey:    getEnv("SENDGRID_API_KEY", ""),
				FromEmail: getEnv("FROM_EMAIL", "noreply@example.com"),
				FromName:  getEnv("FROM_NAME", "E-commerce Store"),
				SMTPHost:  getEnv("SMTP_HOST", ""),
				SMTPPort:  getEnvAsInt("SMTP_PORT", 587),
				SMTPUser:  getEnv("SMTP_USER", ""),
				SMTPPass:  getEnv("SMTP_PASS", ""),
			},
			Storage: StorageConfig{
				Provider:    getEnv("STORAGE_PROVIDER", "local"),
				LocalPath:   getEnv("STORAGE_LOCAL_PATH", "./uploads"),
				S3Bucket:    getEnv("S3_BUCKET", ""),
				S3Region:    getEnv("S3_REGION", "us-east-1"),
				S3AccessKey: getEnv("S3_ACCESS_KEY", ""),
				S3SecretKey: getEnv("S3_SECRET_KEY", ""),
				CDNBaseURL:  getEnv("CDN_BASE_URL", ""),
			},
		},
		Upload: UploadConfig{
			MaxSize:           getEnvAsInt64("UPLOAD_MAX_SIZE", 10485760), // 10MB
			AllowedExtensions: getEnvAsSlice("UPLOAD_ALLOWED_EXTENSIONS", []string{"jpg", "jpeg", "png", "gif", "pdf"}),
			ImageMaxWidth:     getEnvAsInt("IMAGE_MAX_WIDTH", 2048),
			ImageMaxHeight:    getEnvAsInt("IMAGE_MAX_HEIGHT", 2048),
			ThumbnailWidth:    getEnvAsInt("THUMBNAIL_WIDTH", 300),
			ThumbnailHeight:   getEnvAsInt("THUMBNAIL_HEIGHT", 300),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "debug"),
			Format: getEnv("LOG_FORMAT", "json"),
			File:   getEnv("LOG_FILE", "logs/app.log"),
		},
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate JWT secret
	if len(c.JWT.Secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters long")
	}

	// Validate database configuration
	if c.Database.Host == "" {
		return fmt.Errorf("DB_HOST is required")
	}
	if c.Database.Name == "" {
		return fmt.Errorf("DB_NAME is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("DB_USER is required")
	}

	// Validate Redis configuration
	if c.Redis.Host == "" {
		return fmt.Errorf("REDIS_HOST is required")
	}

	// Validate server port
	if c.Server.Port == "" {
		return fmt.Errorf("APP_PORT is required")
	}

	return nil
}

// IsDevelopment returns true if the application is running in development mode
func (c *Config) IsDevelopment() bool {
	return c.App.Environment == "development"
}

// IsProduction returns true if the application is running in production mode
func (c *Config) IsProduction() bool {
	return c.App.Environment == "production"
}

// GetDatabaseDSN returns the database connection string
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Name,
		c.Database.SSLMode,
	)
}

// GetRedisAddr returns the Redis address
func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%s", c.Redis.Host, c.Redis.Port)
}

// Helper functions for environment variable parsing

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}
