package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/service"
)

type BandwidthHandler struct {
	svc *service.BandwidthService
}

func NewBandwidthHandler(svc *service.BandwidthService) *BandwidthHandler {
	return &BandwidthHandler{svc: svc}
}

func (h *BandwidthHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/servers/:id/bandwidth", h.GetServerBandwidth)
	r.GET("/switches/:id/bandwidth", h.GetSwitchBandwidth)
}

func (h *BandwidthHandler) GetSwitchBandwidth(c *gin.Context) {
	switchID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid switch ID"})
		return
	}
	bw, err := h.svc.GetSwitchBandwidth(c.Request.Context(), switchID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: bw})
}

func (h *BandwidthHandler) GetServerBandwidth(c *gin.Context) {
	serverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid server ID"})
		return
	}
	period := c.DefaultQuery("period", "hourly")
	summaries, err := h.svc.GetServerBandwidth(c.Request.Context(), serverID, period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: summaries})
}
