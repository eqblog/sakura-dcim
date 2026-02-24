package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/repository"
)

type AuditHandler struct {
	auditRepo repository.AuditLogRepository
}

func NewAuditHandler(auditRepo repository.AuditLogRepository) *AuditHandler {
	return &AuditHandler{auditRepo: auditRepo}
}

func (h *AuditHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/audit-logs", h.List)
}

func (h *AuditHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	params := domain.AuditLogListParams{
		Page:     page,
		PageSize: pageSize,
	}

	if action := c.Query("action"); action != "" {
		params.Action = action
	}

	if resourceType := c.Query("resource_type"); resourceType != "" {
		params.ResourceType = resourceType
	}

	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if uid, err := uuid.Parse(userIDStr); err == nil {
			params.UserID = &uid
		}
	}

	if startStr := c.Query("start_time"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			params.StartTime = &t
		}
	}

	if endStr := c.Query("end_time"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			params.EndTime = &t
		}
	}

	result, err := h.auditRepo.List(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: "failed to list audit logs"})
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: result})
}
