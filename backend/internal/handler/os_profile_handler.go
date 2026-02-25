package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/service"
)

type OSProfileHandler struct {
	svc *service.OSProfileService
}

func NewOSProfileHandler(svc *service.OSProfileService) *OSProfileHandler {
	return &OSProfileHandler{svc: svc}
}

func (h *OSProfileHandler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/os-profiles")
	{
		g.GET("", h.List)
		g.POST("", h.Create)
		g.GET("/:id", h.Get)
		g.PUT("/:id", h.Update)
		g.DELETE("/:id", h.Delete)
	}
}

func (h *OSProfileHandler) List(c *gin.Context) {
	activeOnly := c.Query("active_only") == "true"
	profiles, err := h.svc.List(c.Request.Context(), activeOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: "failed to list OS profiles"})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: profiles})
}

func (h *OSProfileHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid ID"})
		return
	}
	profile, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, domain.APIResponse{Success: false, Error: "OS profile not found"})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: profile})
}

func (h *OSProfileHandler) Create(c *gin.Context) {
	var profile domain.OSProfile
	if err := c.ShouldBindJSON(&profile); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: formatValidationError(err)})
		return
	}
	result, err := h.svc.Create(c.Request.Context(), &profile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, domain.APIResponse{Success: true, Data: result})
}

func (h *OSProfileHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid ID"})
		return
	}
	var profile domain.OSProfile
	if err := c.ShouldBindJSON(&profile); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: formatValidationError(err)})
		return
	}
	result, err := h.svc.Update(c.Request.Context(), id, &profile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: result})
}

func (h *OSProfileHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid ID"})
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Message: "OS profile deleted"})
}
