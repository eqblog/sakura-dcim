package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/service"
)

type ReinstallHandler struct {
	svc *service.ReinstallService
}

func NewReinstallHandler(svc *service.ReinstallService) *ReinstallHandler {
	return &ReinstallHandler{svc: svc}
}

func (h *ReinstallHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/servers/:id/reinstall", h.StartReinstall)
	r.GET("/servers/:id/reinstall/status", h.GetStatus)
}

// StartReinstall handles POST /servers/:id/reinstall
func (h *ReinstallHandler) StartReinstall(c *gin.Context) {
	serverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid server ID"})
		return
	}

	var req domain.ReinstallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: formatValidationError(err)})
		return
	}

	task, err := h.svc.StartReinstall(c.Request.Context(), serverID, &req, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: task})
}

// GetStatus handles GET /servers/:id/reinstall/status
func (h *ReinstallHandler) GetStatus(c *gin.Context) {
	serverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid server ID"})
		return
	}

	task, err := h.svc.GetInstallStatus(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, domain.APIResponse{Success: false, Error: "no active install task"})
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: task})
}
