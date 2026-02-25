package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/service"
)

type ProvisionHandler struct {
	svc *service.ProvisionService
}

func NewProvisionHandler(svc *service.ProvisionService) *ProvisionHandler {
	return &ProvisionHandler{svc: svc}
}

func (h *ProvisionHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/servers/:id/provision/preflight", h.Preflight)
	r.POST("/servers/:id/provision", h.Provision)
}

// Preflight handles GET /servers/:id/provision/preflight
func (h *ProvisionHandler) Preflight(c *gin.Context) {
	serverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid server ID"})
		return
	}

	result, err := h.svc.Preflight(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: result})
}

// Provision handles POST /servers/:id/provision
func (h *ProvisionHandler) Provision(c *gin.Context) {
	serverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid server ID"})
		return
	}

	var req domain.ProvisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: formatValidationError(err)})
		return
	}

	task, err := h.svc.Provision(c.Request.Context(), serverID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: task})
}
