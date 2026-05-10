package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/quiqxiq/roskit/internal/models"
	"github.com/quiqxiq/roskit/internal/services"
)

type UserHandler struct {
	authSvc *services.AuthService
}

func NewUserHandler(authSvc *services.AuthService) *UserHandler {
	return &UserHandler{authSvc: authSvc}
}

type createUserRequest struct {
	Username string          `json:"username" binding:"required,min=3,max=64"`
	Password string          `json:"password" binding:"required,min=6,max=128"`
	Role     models.UserRole `json:"role" binding:"required"`
}

type updateUserRequest struct {
	Username *string          `json:"username" binding:"omitempty,min=3,max=64"`
	Password *string          `json:"password" binding:"omitempty,min=6,max=128"`
	Role     *models.UserRole `json:"role"`
	Active   *bool            `json:"active"`
}

func (h *UserHandler) List(c *gin.Context) {
	users, err := h.authSvc.ListUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to list users"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users, "error": nil})
}

func (h *UserHandler) Create(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid request body: " + err.Error()})
		return
	}
	if !req.Role.Valid() {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid role"})
		return
	}

	user, err := h.authSvc.CreateUser(c.Request.Context(), req.Username, req.Password, req.Role)
	if err != nil {
		switch err {
		case services.ErrInvalidRole:
			c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		case services.ErrUserAlreadyExists:
			c.JSON(http.StatusConflict, gin.H{"data": nil, "error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": user, "error": nil})
}

func (h *UserHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid user id"})
		return
	}
	user, err := h.authSvc.GetUser(c.Request.Context(), uint(id))
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"data": nil, "error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": user, "error": nil})
}

func (h *UserHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid user id"})
		return
	}

	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid request body: " + err.Error()})
		return
	}

	user, err := h.authSvc.UpdateUser(c.Request.Context(), uint(id), req.Username, req.Password, req.Role, req.Active)
	if err != nil {
		switch err {
		case services.ErrInvalidRole:
			c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid role"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": user, "error": nil})
}

func (h *UserHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid user id"})
		return
	}
	if err := h.authSvc.DeleteUser(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "user deleted"}, "error": nil})
}
