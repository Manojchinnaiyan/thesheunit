// internal/interfaces/http/middleware/cors.go
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/your-org/ecommerce-backend/internal/config"
)

// CORS returns a middleware that handles Cross-Origin Resource Sharing
func CORS(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		if isOriginAllowed(origin, cfg.Security.CORSAllowedOrigins) {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		// Set CORS headers
		c.Header("Access-Control-Allow-Methods", strings.Join(cfg.Security.CORSAllowedMethods, ", "))
		c.Header("Access-Control-Allow-Headers", strings.Join(cfg.Security.CORSAllowedHeaders, ", "))
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400") // 24 hours

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// isOriginAllowed checks if the origin is in the allowed list
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
		// Handle wildcard subdomains (e.g., *.example.com)
		if strings.HasPrefix(allowed, "*.") {
			domain := strings.TrimPrefix(allowed, "*.")
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}
	return false
}
