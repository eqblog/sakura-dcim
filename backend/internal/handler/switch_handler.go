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

		// Ports
		g.GET("/:id/ports", h.ListPorts)
		g.POST("/:id/ports", h.CreatePort)
		g.PUT("/:id/ports/:portId", h.UpdatePort)
		g.DELETE("/:id/ports/:portId", h.DeletePort)

		// Provisioning & status
		g.POST("/:id/ports/:portId/provision", h.ProvisionPort)
		g.GET("/:id/ports/:portId/status", h.GetPortStatus)
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
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: err.Error()})
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
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: err.Error()})
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
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: err.Error()})
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
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: err.Error()})
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
