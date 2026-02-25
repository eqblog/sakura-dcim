package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/service"
)

type DiskLayoutHandler struct {
	svc *service.DiskLayoutService
}

func NewDiskLayoutHandler(svc *service.DiskLayoutService) *DiskLayoutHandler {
	return &DiskLayoutHandler{svc: svc}
}

func (h *DiskLayoutHandler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/disk-layouts")
	{
		g.GET("", h.List)
		g.POST("", h.Create)
		g.GET("/:id", h.Get)
		g.PUT("/:id", h.Update)
		g.DELETE("/:id", h.Delete)
	}
}

func (h *DiskLayoutHandler) List(c *gin.Context) {
	layouts, err := h.svc.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: "failed to list disk layouts"})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: layouts})
}

func (h *DiskLayoutHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid ID"})
		return
	}
	layout, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, domain.APIResponse{Success: false, Error: "disk layout not found"})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: layout})
}

func (h *DiskLayoutHandler) Create(c *gin.Context) {
	var layout domain.DiskLayout
	if err := c.ShouldBindJSON(&layout); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: formatValidationError(err)})
		return
	}
	result, err := h.svc.Create(c.Request.Context(), &layout)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, domain.APIResponse{Success: true, Data: result})
}

func (h *DiskLayoutHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid ID"})
		return
	}
	var layout domain.DiskLayout
	if err := c.ShouldBindJSON(&layout); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: formatValidationError(err)})
		return
	}
	result, err := h.svc.Update(c.Request.Context(), id, &layout)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: result})
}

func (h *DiskLayoutHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid ID"})
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Message: "disk layout deleted"})
}
