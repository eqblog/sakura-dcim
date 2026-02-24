package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/service"
)

type TenantHandler struct {
	tenantService *service.TenantService
}

func NewTenantHandler(tenantService *service.TenantService) *TenantHandler {
	return &TenantHandler{tenantService: tenantService}
}

func (h *TenantHandler) RegisterRoutes(r *gin.RouterGroup) {
	tenants := r.Group("/tenants")
	{
		tenants.GET("", h.List)
		tenants.POST("", h.Create)
		tenants.GET("/:id", h.Get)
		tenants.PUT("/:id", h.Update)
		tenants.DELETE("/:id", h.Delete)
		tenants.GET("/:id/children", h.ListChildren)
		tenants.GET("/:id/tree", h.GetHierarchy)
	}
}

func (h *TenantHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var parentID *uuid.UUID
	if pidStr := c.Query("parent_id"); pidStr != "" {
		pid, err := uuid.Parse(pidStr)
		if err == nil {
			parentID = &pid
		}
	}

	result, err := h.tenantService.List(c.Request.Context(), parentID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: "failed to list tenants"})
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: result})
}

func (h *TenantHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid tenant ID"})
		return
	}

	tenant, err := h.tenantService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, domain.APIResponse{Success: false, Error: "tenant not found"})
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: tenant})
}

func (h *TenantHandler) Create(c *gin.Context) {
	var req service.TenantCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}

	tenant, err := h.tenantService.Create(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, domain.APIResponse{Success: true, Data: tenant})
}

func (h *TenantHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid tenant ID"})
		return
	}

	var req service.TenantUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}

	tenant, err := h.tenantService.Update(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: tenant})
}

func (h *TenantHandler) ListChildren(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid tenant ID"})
		return
	}

	children, err := h.tenantService.ListChildren(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: "failed to list children"})
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: children})
}

func (h *TenantHandler) GetHierarchy(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid tenant ID"})
		return
	}

	tree, err := h.tenantService.GetHierarchy(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: "failed to get hierarchy"})
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: tree})
}

func (h *TenantHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid tenant ID"})
		return
	}

	if err := h.tenantService.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: "failed to delete tenant"})
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Message: "tenant deleted"})
}
