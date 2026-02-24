package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/service"
)

type IPMIHandler struct {
	ipmiService *service.IPMIService
}

func NewIPMIHandler(ipmiService *service.IPMIService) *IPMIHandler {
	return &IPMIHandler{ipmiService: ipmiService}
}

func (h *IPMIHandler) RegisterPowerRoutes(r *gin.RouterGroup) {
	r.POST("/servers/:id/power", h.PowerAction)
	r.GET("/servers/:id/power", h.PowerStatus)
}

func (h *IPMIHandler) RegisterSensorRoutes(r *gin.RouterGroup) {
	r.GET("/servers/:id/sensors", h.Sensors)
}

// PowerAction handles POST /servers/:id/power
// Body: {"action": "on|off|reset|cycle"}
func (h *IPMIHandler) PowerAction(c *gin.Context) {
	serverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid server ID"})
		return
	}

	var req struct {
		Action string `json:"action" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "action is required (on/off/reset/cycle)"})
		return
	}

	result, err := h.ipmiService.PowerAction(c.Request.Context(), serverID, req.Action)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{
		Success: true,
		Message: "Power " + req.Action + " command sent",
		Data:    result,
	})
}

// PowerStatus handles GET /servers/:id/power
func (h *IPMIHandler) PowerStatus(c *gin.Context) {
	serverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid server ID"})
		return
	}

	status, err := h.ipmiService.GetPowerStatus(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{
		Success: true,
		Data: map[string]string{
			"status": status,
		},
	})
}

// Sensors handles GET /servers/:id/sensors
func (h *IPMIHandler) Sensors(c *gin.Context) {
	serverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid server ID"})
		return
	}

	sensors, err := h.ipmiService.GetSensors(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"sensors": sensors,
		},
	})
}
