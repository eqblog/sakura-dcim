package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/pkg/crypto"
)

const (
	ContextKeyUserID   = "user_id"
	ContextKeyTenantID = "tenant_id"
	ContextKeyRoleID   = "role_id"
	ContextKeyClaims   = "claims"
)

// Auth middleware validates JWT access tokens
func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			c.Abort()
			return
		}

		claims, err := crypto.ParseAccessToken(parts[1], jwtSecret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID in token"})
			c.Abort()
			return
		}

		tenantID, err := uuid.Parse(claims.TenantID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid tenant ID in token"})
			c.Abort()
			return
		}

		c.Set(ContextKeyUserID, userID)
		c.Set(ContextKeyTenantID, tenantID)
		c.Set(ContextKeyRoleID, claims.RoleID)
		c.Set(ContextKeyClaims, claims)

		c.Next()
	}
}

// GetUserID extracts the user ID from the Gin context
func GetUserID(c *gin.Context) uuid.UUID {
	v, _ := c.Get(ContextKeyUserID)
	id, _ := v.(uuid.UUID)
	return id
}

// GetTenantID extracts the tenant ID from the Gin context
func GetTenantID(c *gin.Context) uuid.UUID {
	v, _ := c.Get(ContextKeyTenantID)
	id, _ := v.(uuid.UUID)
	return id
}
