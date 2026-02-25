package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/service"
)

type SwitchHandler struct {
	svc *service.SwitchService
}

func NewSwitchHandler(svc *service.SwitchService) *SwitchHandler {
	return &SwitchHandler{svc: svc}
}

func (h *SwitchHandler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/switches")
	{
		g.GET("", h.List)
		g.POST("", h.Create)
		g.GET("/:id", h.Get)
		g.PUT("/:id", h.Update)
		g.DELETE("/:id", h.Delete)

		// Command templates
		g.GET("/templates", h.GetCommandTemplates)

		// Ports
		g.GET("/:id/ports", h.ListPorts)
		g.POST("/:id/ports", h.CreatePort)
		g.PUT("/:id/ports/:portId", h.UpdatePort)
		g.DELETE("/:id/ports/:portId", h.DeletePort)

		// Provisioning & status
		g.POST("/:id/ports/:portId/provision", h.ProvisionPort)
		g.GET("/:id/ports/:portId/status", h.GetPortStatus)

		// Test & poll
		g.POST("/:id/test", h.TestConnection)
		g.POST("/:id/snmp-poll", h.PollSNMP)

		// DHCP relay
		g.POST("/:id/dhcp-relay", h.ConfigureDHCPRelay)

		// Server↔Port linkage
		g.GET("/server/:serverId/ports", h.ListPortsByServer)
		g.PUT("/ports/:portId/link", h.LinkPort)
		g.PUT("/ports/:portId/unlink", h.UnlinkPort)
	}
}

func (h *SwitchHandler) List(c *gin.Context) {
	switches, err := h.svc.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: "failed to list switches"})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: switches})
}

func (h *SwitchHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid ID"})
		return
	}
	sw, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, domain.APIResponse{Success: false, Error: "switch not found"})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: sw})
}

func (h *SwitchHandler) Create(c *gin.Context) {
	var sw domain.Switch
	if err := c.ShouldBindJSON(&sw); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: formatValidationError(err)})
		return
	}
	result, err := h.svc.Create(c.Request.Context(), &sw)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, domain.APIResponse{Success: true, Data: result})
}

func (h *SwitchHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid ID"})
		return
	}
	var sw domain.Switch
	if err := c.ShouldBindJSON(&sw); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: formatValidationError(err)})
		return
	}
	result, err := h.svc.Update(c.Request.Context(), id, &sw)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: result})
}

func (h *SwitchHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid ID"})
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Message: "switch deleted"})
}

// Port handlers

func (h *SwitchHandler) ListPorts(c *gin.Context) {
	switchID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid switch ID"})
		return
	}
	ports, err := h.svc.ListPorts(c.Request.Context(), switchID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: "failed to list ports"})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: ports})
}

func (h *SwitchHandler) CreatePort(c *gin.Context) {
	switchID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid switch ID"})
		return
	}
	var port domain.SwitchPort
	if err := c.ShouldBindJSON(&port); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: formatValidationError(err)})
		return
	}
	port.SwitchID = switchID
	result, err := h.svc.CreatePort(c.Request.Context(), &port)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, domain.APIResponse{Success: true, Data: result})
}

func (h *SwitchHandler) UpdatePort(c *gin.Context) {
	portID, err := uuid.Parse(c.Param("portId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid port ID"})
		return
	}
	var port domain.SwitchPort
	if err := c.ShouldBindJSON(&port); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: formatValidationError(err)})
		return
	}
	result, err := h.svc.UpdatePort(c.Request.Context(), portID, &port)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: result})
}

func (h *SwitchHandler) DeletePort(c *gin.Context) {
	portID, err := uuid.Parse(c.Param("portId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid port ID"})
		return
	}
	if err := h.svc.DeletePort(c.Request.Context(), portID); err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Message: "port deleted"})
}

func (h *SwitchHandler) ProvisionPort(c *gin.Context) {
	switchID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid switch ID"})
		return
	}
	portID, err := uuid.Parse(c.Param("portId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid port ID"})
		return
	}
	if err := h.svc.ProvisionPort(c.Request.Context(), switchID, portID); err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Message: "port provisioned"})
}

// GetCommandTemplates returns default CLI command templates for all supported switch vendors.
func (h *SwitchHandler) GetCommandTemplates(c *gin.Context) {
	templates := domain.DefaultSwitchTemplates()

	// Optional filter by vendor query param
	if vendor := c.Query("vendor"); vendor != "" {
		for _, t := range templates {
			if t.Vendor == vendor {
				c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: t})
				return
			}
		}
		c.JSON(http.StatusNotFound, domain.APIResponse{Success: false, Error: "vendor not found"})
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: templates})
}

// Server↔Port linkage handlers

func (h *SwitchHandler) ListPortsByServer(c *gin.Context) {
	serverID, err := uuid.Parse(c.Param("serverId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid server ID"})
		return
	}
	ports, err := h.svc.GetPortsWithSwitchInfo(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: "failed to list ports"})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: ports})
}

func (h *SwitchHandler) LinkPort(c *gin.Context) {
	portID, err := uuid.Parse(c.Param("portId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid port ID"})
		return
	}
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: formatValidationError(err)})
		return
	}
	serverID, err := uuid.Parse(req.ServerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid server ID"})
		return
	}
	if err := h.svc.LinkPortToServer(c.Request.Context(), portID, serverID); err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Message: "port linked to server"})
}

func (h *SwitchHandler) UnlinkPort(c *gin.Context) {
	portID, err := uuid.Parse(c.Param("portId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid port ID"})
		return
	}
	if err := h.svc.UnlinkPort(c.Request.Context(), portID); err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Message: "port unlinked"})
}

func (h *SwitchHandler) ConfigureDHCPRelay(c *gin.Context) {
	switchID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid switch ID"})
		return
	}
	var req domain.DHCPRelayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: formatValidationError(err)})
		return
	}
	if err := h.svc.ConfigureDHCPRelay(c.Request.Context(), switchID, &req); err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	msg := "DHCP relay configured"
	if req.Remove {
		msg = "DHCP relay removed"
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Message: msg})
}

func (h *SwitchHandler) TestConnection(c *gin.Context) {
	switchID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid switch ID"})
		return
	}
	result, err := h.svc.TestConnection(c.Request.Context(), switchID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: result})
}

func (h *SwitchHandler) PollSNMP(c *gin.Context) {
	switchID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid switch ID"})
		return
	}
	result, err := h.svc.PollSNMP(c.Request.Context(), switchID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: result})
}

func (h *SwitchHandler) GetPortStatus(c *gin.Context) {
	switchID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid switch ID"})
		return
	}
	portID, err := uuid.Parse(c.Param("portId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid port ID"})
		return
	}
	status, err := h.svc.GetPortStatus(c.Request.Context(), switchID, portID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: status})
}
