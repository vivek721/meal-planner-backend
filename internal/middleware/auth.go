// Package middleware provides the Gin middleware used by the API.
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/meal-planner/backend/internal/config"
	"github.com/meal-planner/backend/internal/utils"
)

// Gin context keys set by AuthMiddleware.
const (
	ContextKeyUserID = "userID"
	ContextKeyEmail  = "email"
)

// AuthMiddleware validates JWT tokens
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header is required",
			})
			c.Abort()
			return
		}

		// Check if the header has the Bearer prefix
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization header format",
			})
			c.Abort()
			return
		}

		token := parts[1]

		// Validate token
		claims, err := utils.ValidateToken(token, cfg.JWTSecret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired token",
			})
			c.Abort()
			return
		}

		// Set user info in context
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyEmail, claims.Email)

		c.Next()
	}
}

// GetUserID retrieves the authenticated user's ID set by AuthMiddleware.
func GetUserID(c *gin.Context) (string, bool) {
	value, exists := c.Get(ContextKeyUserID)
	if !exists {
		return "", false
	}
	userID, ok := value.(string)
	if !ok || userID == "" {
		return "", false
	}
	return userID, true
}
