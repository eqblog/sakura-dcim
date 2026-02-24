package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/middleware"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/repository"
)

type ServerHandler struct {
	serverRepo repository.ServerRepository
}

func NewServerHandler(serverRepo repository.ServerRepository) *ServerHandler {
	return &ServerHandler{serverRepo: serverRepo}
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

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

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

	result, err := h.serverRepo.List(c.Request.Context(), params)
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

	server, err := h.serverRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, domain.APIResponse{Success: false, Error: "server not found"})
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: server})
}

func (h *ServerHandler) Create(c *gin.Context) {
	var req domain.ServerCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}

	tenantID := middleware.GetTenantID(c)
	server := &domain.Server{
		ID:        uuid.New(),
		TenantID:  &tenantID,
		AgentID:   req.AgentID,
		Hostname:  req.Hostname,
		Label:     req.Label,
		Status:    domain.ServerStatusActive,
		PrimaryIP: req.PrimaryIP,
		IPMIIP:    req.IPMIIP,
		IPMIUser:  req.IPMIUser,
		IPMIPass:  req.IPMIPass,
		Tags:      req.Tags,
		Notes:     req.Notes,
	}

	if err := h.serverRepo.Create(c.Request.Context(), server); err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: "failed to create server"})
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

	server, err := h.serverRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, domain.APIResponse{Success: false, Error: "server not found"})
		return
	}

	var req domain.ServerUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}

	if req.Hostname != nil {
		server.Hostname = *req.Hostname
	}
	if req.Label != nil {
		server.Label = *req.Label
	}
	if req.AgentID != nil {
		server.AgentID = req.AgentID
	}
	if req.PrimaryIP != nil {
		server.PrimaryIP = *req.PrimaryIP
	}
	if req.IPMIIP != nil {
		server.IPMIIP = *req.IPMIIP
	}
	if req.IPMIUser != nil {
		server.IPMIUser = *req.IPMIUser
	}
	if req.IPMIPass != nil {
		server.IPMIPass = *req.IPMIPass
	}
	if req.Tags != nil {
		server.Tags = *req.Tags
	}
	if req.Notes != nil {
		server.Notes = *req.Notes
	}

	if err := h.serverRepo.Update(c.Request.Context(), server); err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: "failed to update server"})
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

	if err := h.serverRepo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: "failed to delete server"})
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Message: "server deleted"})
}
