package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/service"
	ws "github.com/sakura-dcim/sakura-dcim/backend/internal/websocket"
)

type AgentHandler struct {
	agentService *service.AgentService
	hub          *ws.Hub
}

func NewAgentHandler(agentService *service.AgentService, hub *ws.Hub) *AgentHandler {
	return &AgentHandler{
		agentService: agentService,
		hub:          hub,
	}
}

func (h *AgentHandler) RegisterRoutes(r *gin.RouterGroup) {
	agents := r.Group("/agents")
	{
		agents.GET("", h.List)
		agents.POST("", h.Create)
		agents.GET("/:id", h.Get)
		agents.PUT("/:id", h.Update)
		agents.DELETE("/:id", h.Delete)
	}
}

func (h *AgentHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	result, err := h.agentService.List(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: "failed to list agents"})
		return
	}

	onlineIDs := h.hub.OnlineAgentIDs()
	onlineSet := make(map[uuid.UUID]bool, len(onlineIDs))
	for _, id := range onlineIDs {
		onlineSet[id] = true
	}
	for i := range result.Items {
		if onlineSet[result.Items[i].ID] {
			result.Items[i].Status = domain.AgentStatusOnline
		}
	}

	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: result})
}

func (h *AgentHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid agent ID"})
		return
	}

	agent, err := h.agentService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, domain.APIResponse{Success: false, Error: "agent not found"})
		return
	}

	if h.hub.IsAgentOnline(id) {
		agent.Status = domain.AgentStatusOnline
	}

	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: agent})
}

func (h *AgentHandler) Create(c *gin.Context) {
	var req domain.AgentCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: formatValidationError(err)})
		return
	}

	resp, err := h.agentService.Create(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: "failed to create agent"})
		return
	}

	c.JSON(http.StatusCreated, domain.APIResponse{Success: true, Data: resp})
}

func (h *AgentHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid agent ID"})
		return
	}

	var req struct {
		Name         string   `json:"name" binding:"required"`
		Location     string   `json:"location"`
		Capabilities []string `json:"capabilities"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: formatValidationError(err)})
		return
	}

	agent, err := h.agentService.Update(c.Request.Context(), id, req.Name, req.Location, req.Capabilities)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: agent})
}

func (h *AgentHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid agent ID"})
		return
	}

	if err := h.agentService.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: "failed to delete agent"})
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Message: "agent deleted"})
}
