package middleware

import (
	"bytes"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/repository"
	"go.uber.org/zap"
)

// AuditLog middleware logs all state-changing API calls
func AuditLog(auditRepo repository.AuditLogRepository, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only audit state-changing methods
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		// Read request body for logging
		var bodyBytes []byte
		if c.Request.Body != nil {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		c.Next()

		// Log after handler execution
		userID := GetUserID(c)
		tenantID := GetTenantID(c)

		var tenantIDPtr *uuid.UUID
		if tenantID != uuid.Nil {
			tenantIDPtr = &tenantID
		}
		var userIDPtr *uuid.UUID
		if userID != uuid.Nil {
			userIDPtr = &userID
		}

		log := &domain.AuditLog{
			ID:        uuid.New(),
			TenantID:  tenantIDPtr,
			UserID:    userIDPtr,
			Action:    c.Request.Method + " " + c.FullPath(),
			IPAddress: c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
			Details:   map[string]any{"status": c.Writer.Status(), "body_size": len(bodyBytes)},
		}

		if err := auditRepo.Create(c.Request.Context(), log); err != nil {
			logger.Error("failed to create audit log", zap.Error(err))
		}
	}
}
