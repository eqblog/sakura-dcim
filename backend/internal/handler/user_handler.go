package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/middleware"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/service"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

func (h *UserHandler) RegisterRoutes(r *gin.RouterGroup) {
	users := r.Group("/users")
	{
		users.GET("", h.List)
		users.POST("", h.Create)
		users.GET("/:id", h.Get)
		users.PUT("/:id", h.Update)
		users.DELETE("/:id", h.Delete)
	}
}

func (h *UserHandler) List(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	result, err := h.userService.List(c.Request.Context(), tenantID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: "failed to list users"})
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: result})
}

func (h *UserHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid user ID"})
		return
	}

	user, err := h.userService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, domain.APIResponse{Success: false, Error: "user not found"})
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: user})
}

func (h *UserHandler) Create(c *gin.Context) {
	var req domain.UserCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: formatValidationError(err)})
		return
	}

	tenantID := middleware.GetTenantID(c)
	user, err := h.userService.Create(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, domain.APIResponse{Success: true, Data: user})
}

func (h *UserHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid user ID"})
		return
	}

	var req domain.UserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: formatValidationError(err)})
		return
	}

	user, err := h.userService.Update(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Data: user})
}

func (h *UserHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid user ID"})
		return
	}

	if err := h.userService.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: "failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Message: "user deleted"})
}
