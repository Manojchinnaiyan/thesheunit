// internal/interfaces/http/middleware/logger.go
package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/your-org/ecommerce-backend/internal/config"
)

// Logger returns a gin.HandlerFunc that logs HTTP requests
func Logger(cfg *config.Config) gin.HandlerFunc {
	// Configure logrus
	logger := logrus.New()

	// Set log format based on config
	if cfg.Logging.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		})
	}

	// Set log level
	level, err := logrus.ParseLevel(cfg.Logging.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Create log entry
		entry := logger.WithFields(logrus.Fields{
			"request_id":    param.Keys["request_id"],
			"timestamp":     param.TimeStamp.Format(time.RFC3339),
			"method":        param.Method,
			"path":          param.Path,
			"status_code":   param.StatusCode,
			"latency":       param.Latency,
			"client_ip":     param.ClientIP,
			"user_agent":    param.Request.UserAgent(),
			"response_size": param.BodySize,
		})

		// Add error if present
		if param.ErrorMessage != "" {
			entry = entry.WithField("error", param.ErrorMessage)
		}

		// Log based on status code
		if param.StatusCode >= 500 {
			entry.Error("HTTP request completed with server error")
		} else if param.StatusCode >= 400 {
			entry.Warn("HTTP request completed with client error")
		} else {
			entry.Info("HTTP request completed successfully")
		}

		return ""
	})
}
