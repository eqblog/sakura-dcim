package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/service"
)

type IPHandler struct {
	svc *service.IPService
}

func NewIPHandler(svc *service.IPService) *IPHandler {
	return &IPHandler{svc: svc}
}

func (h *IPHandler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/ip-pools")
	{
		g.GET("", h.ListPools)
		g.POST("", h.CreatePool)
		g.GET("/:id", h.GetPool)
		g.PUT("/:id", h.UpdatePool)
		g.DELETE("/:id", h.DeletePool)

		// Addresses under pool
		g.GET("/:id/addresses", h.ListAddresses)
		g.POST("/:id/addresses", h.CreateAddress)
		g.PUT("/:id/addresses/:addrId", h.UpdateAddress)
		g.DELETE("/:id/addresses/:addrId", h.DeleteAddress)

		// Auto-assign
		g.POST("/:id/assign", h.AssignNextAvailable)
	}
}

func (h *IPHandler) ListPools(c *gin.Context) {
	pools, err := h.svc.ListPools(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: "failed to list pools"})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: pools})
}

func (h *IPHandler) GetPool(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid ID"})
		return
	}
	pool, err := h.svc.GetPool(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, domain.APIResponse{Success: false, Error: "pool not found"})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: pool})
}

func (h *IPHandler) CreatePool(c *gin.Context) {
	var pool domain.IPPool
	if err := c.ShouldBindJSON(&pool); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: formatValidationError(err)})
		return
	}
	result, err := h.svc.CreatePool(c.Request.Context(), &pool)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, domain.APIResponse{Success: true, Data: result})
}

func (h *IPHandler) UpdatePool(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid ID"})
		return
	}
	var pool domain.IPPool
	if err := c.ShouldBindJSON(&pool); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: formatValidationError(err)})
		return
	}
	result, err := h.svc.UpdatePool(c.Request.Context(), id, &pool)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: result})
}

func (h *IPHandler) DeletePool(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid ID"})
		return
	}
	if err := h.svc.DeletePool(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Message: "pool deleted"})
}

func (h *IPHandler) ListAddresses(c *gin.Context) {
	poolID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid pool ID"})
		return
	}
	addrs, err := h.svc.ListAddresses(c.Request.Context(), poolID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: "failed to list addresses"})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: addrs})
}

func (h *IPHandler) CreateAddress(c *gin.Context) {
	poolID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid pool ID"})
		return
	}
	var addr domain.IPAddress
	if err := c.ShouldBindJSON(&addr); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: formatValidationError(err)})
		return
	}
	result, err := h.svc.CreateAddress(c.Request.Context(), poolID, &addr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, domain.APIResponse{Success: true, Data: result})
}

func (h *IPHandler) UpdateAddress(c *gin.Context) {
	addrID, err := uuid.Parse(c.Param("addrId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid address ID"})
		return
	}
	var req domain.IPAddressUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: formatValidationError(err)})
		return
	}
	result, err := h.svc.UpdateAddress(c.Request.Context(), addrID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: result})
}

func (h *IPHandler) DeleteAddress(c *gin.Context) {
	addrID, err := uuid.Parse(c.Param("addrId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid address ID"})
		return
	}
	if err := h.svc.DeleteAddress(c.Request.Context(), addrID); err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Message: "address deleted"})
}

func (h *IPHandler) AssignNextAvailable(c *gin.Context) {
	poolID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid pool ID"})
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
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid server_id"})
		return
	}
	addr, err := h.svc.AssignNextAvailable(c.Request.Context(), poolID, serverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: addr})
}
