package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/middleware"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/service"
)

type ServerHandler struct {
	serverService *service.ServerService
}

func NewServerHandler(serverService *service.ServerService) *ServerHandler {
	return &ServerHandler{serverService: serverService}
}

func (h *ServerHandler) RegisterRoutes(r *gin.RouterGroup) {
	servers := r.Group("/servers")
	{
		servers.GET("", h.List)
		servers.POST("", h.Create)
		servers.GET("/:id", h.Get)
		servers.PUT("/:id", h.Update)
		servers.DELETE("/:id", h.Delete)
	}
}

func (h *ServerHandler) List(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	params := domain.ServerListParams{
		TenantID: &tenantID,
		Search:   c.Query("search"),
		Page:     page,
		PageSize: pageSize,
	}

	if status := c.Query("status"); status != "" {
		s := domain.ServerStatus(status)
		params.Status = &s
	}

	if agentID := c.Query("agent_id"); agentID != "" {
		id, err := uuid.Parse(agentID)
		if err == nil {
			params.AgentID = &id
		}
	}

	result, err := h.serverService.List(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: "failed to list servers"})
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: result})
}

func (h *ServerHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid server ID"})
		return
	}

	server, err := h.serverService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, domain.APIResponse{Success: false, Error: "server not found"})
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: server})
}

func (h *ServerHandler) Create(c *gin.Context) {
	var req domain.ServerCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: formatValidationError(err)})
		return
	}

	tenantID := middleware.GetTenantID(c)
	server, err := h.serverService.Create(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, domain.APIResponse{Success: true, Data: server})
}

func (h *ServerHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid server ID"})
		return
	}

	var req domain.ServerUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: formatValidationError(err)})
		return
	}

	server, err := h.serverService.Update(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: server})
}

func (h *ServerHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid server ID"})
		return
	}

	if err := h.serverService.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: "failed to delete server"})
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Message: "server deleted"})
}
