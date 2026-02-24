package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/repository"
	"go.uber.org/zap"
)

// sensitiveFields are redacted from audit log details
var sensitiveFields = map[string]bool{
	"password":       true,
	"root_password":  true,
	"ipmi_pass":      true,
	"ssh_pass":       true,
	"snmp_community": true,
	"secret":         true,
	"token":          true,
	"refresh_token":  true,
	"access_token":   true,
	"encryption_key": true,
}

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

		// Extract resource type and ID from the route
		resourceType, resourceID := extractResource(c)

		// Build details with sanitized request body
		details := map[string]any{
			"status":      c.Writer.Status(),
			"method":      c.Request.Method,
			"path":        c.Request.URL.Path,
			"query":       c.Request.URL.RawQuery,
			"body_size":   len(bodyBytes),
			"response_sz": c.Writer.Size(),
		}

		// Include sanitized body for non-large payloads
		if len(bodyBytes) > 0 && len(bodyBytes) < 10240 {
			sanitized := sanitizeBody(bodyBytes)
			if sanitized != nil {
				details["body"] = sanitized
			}
		}

		log := &domain.AuditLog{
			ID:           uuid.New(),
			TenantID:     tenantIDPtr,
			UserID:       userIDPtr,
			Action:       c.Request.Method + " " + c.FullPath(),
			ResourceType: resourceType,
			ResourceID:   resourceID,
			IPAddress:    c.ClientIP(),
			UserAgent:    c.Request.UserAgent(),
			Details:      details,
		}

		if err := auditRepo.Create(c.Request.Context(), log); err != nil {
			logger.Error("failed to create audit log", zap.Error(err))
		}
	}
}

// extractResource derives resource_type and resource_id from route parameters.
func extractResource(c *gin.Context) (string, *uuid.UUID) {
	path := c.FullPath()
	if path == "" {
		return "", nil
	}

	resourceMap := map[string]string{
		"/api/v1/servers":      "server",
		"/api/v1/agents":       "agent",
		"/api/v1/users":        "user",
		"/api/v1/roles":        "role",
		"/api/v1/tenants":      "tenant",
		"/api/v1/os-profiles":  "os_profile",
		"/api/v1/disk-layouts": "disk_layout",
		"/api/v1/scripts":      "script",
		"/api/v1/switches":     "switch",
		"/api/v1/ip-pools":     "ip_pool",
	}

	for prefix, resType := range resourceMap {
		if strings.HasPrefix(path, prefix) {
			idStr := c.Param("id")
			if idStr != "" {
				if id, err := uuid.Parse(idStr); err == nil {
					return resType, &id
				}
			}
			return resType, nil
		}
	}

	return "", nil
}

// sanitizeBody parses JSON body and redacts sensitive fields.
func sanitizeBody(body []byte) map[string]any {
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil
	}
	redactMap(parsed)
	return parsed
}

func redactMap(m map[string]any) {
	for key, val := range m {
		if sensitiveFields[strings.ToLower(key)] {
			m[key] = "***REDACTED***"
			continue
		}
		if nested, ok := val.(map[string]any); ok {
			redactMap(nested)
		}
	}
}
