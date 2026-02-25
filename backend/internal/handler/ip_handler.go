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

		// Static routes BEFORE /:id to avoid conflict
		g.GET("/assignable", h.ListAssignablePools)
		g.POST("/auto-assign", h.AutoAssign)
		g.GET("/by-server/:serverId", h.ListAddressesByServer)

		g.GET("/:id", h.GetPool)
		g.PUT("/:id", h.UpdatePool)
		g.DELETE("/:id", h.DeletePool)

		// Addresses under pool
		g.GET("/:id/addresses", h.ListAddresses)
		g.POST("/:id/addresses", h.CreateAddress)
		g.PUT("/:id/addresses/:addrId", h.UpdateAddress)
		g.DELETE("/:id/addresses/:addrId", h.DeleteAddress)

		// Children (subdivision)
		g.GET("/:id/children", h.ListChildPools)
		g.POST("/:id/generate", h.GeneratePoolIPs)

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

func (h *IPHandler) ListAssignablePools(c *gin.Context) {
	pools, err := h.svc.ListAllAssignablePools(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: "failed to list assignable pools"})
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
	var req struct {
		domain.IPPool
		ReserveGateway bool `json:"reserve_gateway"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: formatValidationError(err)})
		return
	}
	pool := &req.IPPool
	result, err := h.svc.CreatePool(c.Request.Context(), pool)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	// Auto-generate IPs for ip_pool type
	if pool.PoolType == "ip_pool" {
		_ = h.svc.GeneratePoolIPs(c.Request.Context(), result.ID, req.ReserveGateway)
		result, _ = h.svc.GetPool(c.Request.Context(), result.ID)
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

func (h *IPHandler) ListAddressesByServer(c *gin.Context) {
	serverID, err := uuid.Parse(c.Param("serverId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid server ID"})
		return
	}
	addrs, err := h.svc.ListAddressesByServer(c.Request.Context(), serverID)
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

func (h *IPHandler) ListChildPools(c *gin.Context) {
	parentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid pool ID"})
		return
	}
	pools, err := h.svc.ListChildPools(c.Request.Context(), parentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: "failed to list child pools"})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: pools})
}

func (h *IPHandler) GeneratePoolIPs(c *gin.Context) {
	poolID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid pool ID"})
		return
	}
	var req struct {
		ReserveGateway bool `json:"reserve_gateway"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.svc.GeneratePoolIPs(c.Request.Context(), poolID, req.ReserveGateway); err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Message: "IPs generated"})
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

func (h *IPHandler) AutoAssign(c *gin.Context) {
	var req struct {
		ServerID string  `json:"server_id" binding:"required"`
		PoolID   *string `json:"pool_id"`
		VRF      string  `json:"vrf"`
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
	var poolID *uuid.UUID
	if req.PoolID != nil && *req.PoolID != "" {
		pid, err := uuid.Parse(*req.PoolID)
		if err != nil {
			c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid pool_id"})
			return
		}
		poolID = &pid
	}
	addr, err := h.svc.AutoAssign(c.Request.Context(), serverID, poolID, req.VRF)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: addr})
}
