package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/repository"
)

// RequirePermission middleware checks if the user has the specified permission
func RequirePermission(roleRepo repository.RoleRepository, permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleIDStr, _ := c.Get(ContextKeyRoleID)
		roleIDString, ok := roleIDStr.(string)
		if !ok || roleIDString == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "no role assigned"})
			c.Abort()
			return
		}

		roleID, err := uuid.Parse(roleIDString)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "invalid role"})
			c.Abort()
			return
		}

		role, err := roleRepo.GetByID(c.Request.Context(), roleID)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "role not found"})
			c.Abort()
			return
		}

		for _, perm := range permissions {
			if !role.HasPermission(perm) {
				c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions", "required": perm})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}
