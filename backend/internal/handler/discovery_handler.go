package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/service"
)

type DiscoveryHandler struct {
	svc *service.DiscoveryService
}

func NewDiscoveryHandler(svc *service.DiscoveryService) *DiscoveryHandler {
	return &DiscoveryHandler{svc: svc}
}

func (h *DiscoveryHandler) RegisterRoutes(viewGroup, manageGroup *gin.RouterGroup) {
	// View permissions
	viewGroup.GET("/agents/:id/discovery/status", h.GetDiscoveryStatus)
	viewGroup.GET("/discovery/servers", h.ListDiscoveredServers)
	viewGroup.GET("/discovery/servers/:id", h.GetDiscoveredServer)

	// Manage permissions
	manageGroup.POST("/agents/:id/discovery/start", h.StartDiscovery)
	manageGroup.POST("/agents/:id/discovery/stop", h.StopDiscovery)
	manageGroup.POST("/discovery/servers/:id/approve", h.ApproveServer)
	manageGroup.POST("/discovery/servers/:id/reject", h.RejectServer)
	manageGroup.DELETE("/discovery/servers/:id", h.DeleteDiscoveredServer)
}

func (h *DiscoveryHandler) StartDiscovery(c *gin.Context) {
	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid agent ID"})
		return
	}

	var req domain.DiscoveryStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: formatValidationError(err)})
		return
	}

	var userID *uuid.UUID
	if uid, exists := c.Get("user_id"); exists {
		if id, ok := uid.(uuid.UUID); ok {
			userID = &id
		}
	}

	session, err := h.svc.StartDiscovery(c.Request.Context(), agentID, userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: session})
}

func (h *DiscoveryHandler) StopDiscovery(c *gin.Context) {
	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid agent ID"})
		return
	}

	if err := h.svc.StopDiscovery(c.Request.Context(), agentID); err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Message: "discovery stopped"})
}

func (h *DiscoveryHandler) GetDiscoveryStatus(c *gin.Context) {
	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid agent ID"})
		return
	}

	status, err := h.svc.GetDiscoveryStatus(c.Request.Context(), agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: status})
}

func (h *DiscoveryHandler) ListDiscoveredServers(c *gin.Context) {
	params := domain.DiscoveredServerListParams{
		Search:   c.Query("search"),
		Page:     1,
		PageSize: 20,
	}

	if v := c.Query("agent_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			params.AgentID = &id
		}
	}
	if v := c.Query("status"); v != "" {
		s := domain.DiscoveredServerStatus(v)
		params.Status = &s
	}
	if v := c.Query("page"); v != "" {
		fmt.Sscanf(v, "%d", &params.Page)
	}
	if v := c.Query("page_size"); v != "" {
		fmt.Sscanf(v, "%d", &params.PageSize)
	}

	result, err := h.svc.ListDiscoveredServers(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: result})
}

func (h *DiscoveryHandler) GetDiscoveredServer(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid ID"})
		return
	}

	ds, err := h.svc.GetDiscoveredServer(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: ds})
}

func (h *DiscoveryHandler) ApproveServer(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid ID"})
		return
	}

	var req domain.DiscoveryApproveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: formatValidationError(err)})
		return
	}

	var userID *uuid.UUID
	if uid, exists := c.Get("user_id"); exists {
		if id, ok := uid.(uuid.UUID); ok {
			userID = &id
		}
	}
	_ = userID // could be used for approved_by field

	server, err := h.svc.ApproveServer(c.Request.Context(), id, userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: server})
}

func (h *DiscoveryHandler) RejectServer(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid ID"})
		return
	}

	if err := h.svc.RejectServer(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Message: "server rejected"})
}

func (h *DiscoveryHandler) DeleteDiscoveredServer(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid ID"})
		return
	}

	if err := h.svc.DeleteDiscoveredServer(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Message: "deleted"})
}
