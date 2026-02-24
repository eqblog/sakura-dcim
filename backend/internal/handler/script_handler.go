package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/service"
)

type ScriptHandler struct {
	svc *service.ScriptService
}

func NewScriptHandler(svc *service.ScriptService) *ScriptHandler {
	return &ScriptHandler{svc: svc}
}

func (h *ScriptHandler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/scripts")
	{
		g.GET("", h.List)
		g.POST("", h.Create)
		g.GET("/:id", h.Get)
		g.PUT("/:id", h.Update)
		g.DELETE("/:id", h.Delete)
	}
}

func (h *ScriptHandler) List(c *gin.Context) {
	scripts, err := h.svc.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: "failed to list scripts"})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: scripts})
}

func (h *ScriptHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid ID"})
		return
	}
	script, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, domain.APIResponse{Success: false, Error: "script not found"})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: script})
}

func (h *ScriptHandler) Create(c *gin.Context) {
	var script domain.Script
	if err := c.ShouldBindJSON(&script); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	result, err := h.svc.Create(c.Request.Context(), &script)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, domain.APIResponse{Success: true, Data: result})
}

func (h *ScriptHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid ID"})
		return
	}
	var script domain.Script
	if err := c.ShouldBindJSON(&script); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	result, err := h.svc.Update(c.Request.Context(), id, &script)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: result})
}

func (h *ScriptHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid ID"})
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Message: "script deleted"})
}
