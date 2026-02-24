package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/middleware"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/service"
)

type RoleHandler struct {
	roleService *service.RoleService
}

func NewRoleHandler(roleService *service.RoleService) *RoleHandler {
	return &RoleHandler{roleService: roleService}
}

func (h *RoleHandler) RegisterRoutes(r *gin.RouterGroup) {
	roles := r.Group("/roles")
	{
		roles.GET("", h.List)
		roles.POST("", h.Create)
		roles.GET("/permissions", h.ListPermissions)
		roles.GET("/:id", h.Get)
		roles.PUT("/:id", h.Update)
		roles.DELETE("/:id", h.Delete)
	}
}

func (h *RoleHandler) List(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	roles, err := h.roleService.List(c.Request.Context(), &tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: "failed to list roles"})
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: roles})
}

func (h *RoleHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid role ID"})
		return
	}

	role, err := h.roleService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, domain.APIResponse{Success: false, Error: "role not found"})
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: role})
}

func (h *RoleHandler) Create(c *gin.Context) {
	var req struct {
		Name        string   `json:"name" binding:"required"`
		Permissions []string `json:"permissions" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}

	tenantID := middleware.GetTenantID(c)
	role, err := h.roleService.Create(c.Request.Context(), &tenantID, req.Name, req.Permissions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, domain.APIResponse{Success: true, Data: role})
}

func (h *RoleHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid role ID"})
		return
	}

	var req struct {
		Name        string   `json:"name" binding:"required"`
		Permissions []string `json:"permissions" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}

	role, err := h.roleService.Update(c.Request.Context(), id, req.Name, req.Permissions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: role})
}

func (h *RoleHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid role ID"})
		return
	}

	if err := h.roleService.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Message: "role deleted"})
}

func (h *RoleHandler) ListPermissions(c *gin.Context) {
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: h.roleService.AllPermissions()})
}
