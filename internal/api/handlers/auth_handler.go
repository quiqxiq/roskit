package handlers

import (
	stderrors "errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/quiqxiq/roskit/internal/api/middleware"
	"github.com/quiqxiq/roskit/internal/models"
	"github.com/quiqxiq/roskit/internal/services"
	"github.com/quiqxiq/roskit/pkg/errors"
)

type AuthHandler struct {
	svc       *services.AuthService
	tenantSvc *services.TenantService
	audit     *middleware.AuditLogger
}

func NewAuthHandler(svc *services.AuthService, tenantSvc *services.TenantService, audit *middleware.AuditLogger) *AuthHandler {
	return &AuthHandler{svc: svc, tenantSvc: tenantSvc, audit: audit}
}

type loginRequest struct {
	Tenant   string `json:"tenant"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

type setupRequest struct {
	TenantName string `json:"tenant_name" binding:"required"`
	TenantSlug string `json:"tenant_slug" binding:"required"`
	Username   string `json:"username" binding:"required,min=3"`
	Password   string `json:"password" binding:"required,min=6"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid request body"})
		return
	}

	result, err := h.svc.Login(c.Request.Context(), req.Tenant, req.Username, req.Password)
	if err != nil {
		switch {
		case stderrors.Is(err, services.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, gin.H{"data": nil, "error": "invalid username or password"})
		case stderrors.Is(err, services.ErrUserInactive):
			c.JSON(http.StatusForbidden, gin.H{"data": nil, "error": err.Error()})
		case stderrors.Is(err, services.ErrTenantSuspended):
			c.JSON(http.StatusForbidden, gin.H{"data": nil, "error": err.Error()})
		case stderrors.Is(err, services.ErrTenantNotFound):
			c.JSON(http.StatusUnauthorized, gin.H{"data": nil, "error": "invalid username or password"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid request body"})
		return
	}

	result, err := h.svc.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"data": nil, "error": "invalid or expired refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	tokenID, _ := c.Get("tokenID")
	userID, _ := c.Get("userID")
	tokenIDStr, _ := tokenID.(string)
	userIDStr := ""
	if id, ok := userID.(uint); ok {
		userIDStr = fmt.Sprintf("%d", id)
	}

	var req logoutRequest
	_ = c.ShouldBindJSON(&req)

	err := h.svc.Logout(c.Request.Context(), tokenIDStr, userIDStr, req.RefreshToken, time.Now().Add(15*time.Minute))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "logged out"}, "error": nil})
	h.audit.LogAuth(c, "auth.logout", "")
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, _ := c.Get("userID")
	username, _ := c.Get("username")
	role, _ := c.Get("role")
	tenantSlug, _ := c.Get("tenantSlug")

	uid, _ := userID.(uint)
	uname, _ := username.(string)
	roleVal, _ := role.(models.UserRole)
	slug, _ := tenantSlug.(string)

	var tenantID *uint
	if v, ok := c.Get("tenantID"); ok {
		if id, ok := v.(uint); ok {
			t := id
			tenantID = &t
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": services.UserView{
			ID:         uid,
			Username:   uname,
			Role:       roleVal,
			TenantID:   tenantID,
			TenantSlug: slug,
		},
		"error": nil,
	})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, _ := c.Get("userID")
	uid, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"data": nil, "error": "user not authenticated"})
		return
	}

	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid request body"})
		return
	}

	err := h.svc.ChangePassword(c.Request.Context(), uid, req.OldPassword, req.NewPassword)
	if err != nil {
		switch err {
		case services.ErrOldPasswordWrong:
			c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "password changed"}, "error": nil})
}

func (h *AuthHandler) Setup(c *gin.Context) {
	count, err := h.svc.UserCount(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "internal error"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusForbidden, gin.H{"data": nil, "error": errors.NewForbidden("setup already completed").Message})
		return
	}

	var req setupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid request body"})
		return
	}

	tenant, err := h.tenantSvc.Create(c.Request.Context(), services.CreateTenantRequest{
		Name: req.TenantName,
		Slug: req.TenantSlug,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	tid := tenant.ID
	user, err := h.svc.CreateUser(c.Request.Context(), &tid, tenant.Slug, req.Username, req.Password, models.UserRoleOwner)
	if err != nil {
		switch err {
		case services.ErrInvalidRole:
			c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		case services.ErrUserAlreadyExists:
			c.JSON(http.StatusConflict, gin.H{"data": nil, "error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "internal error"})
		}
		return
	}

	loginResult, err := h.svc.Login(c.Request.Context(), tenant.Slug, req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "auto-login after setup failed"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": gin.H{
			"tenant": gin.H{
				"id":   tenant.ID,
				"name": tenant.Name,
				"slug": tenant.Slug,
			},
			"user": gin.H{
				"id":       user.ID,
				"username": user.Username,
				"role":     user.Role,
			},
			"access_token":  loginResult.AccessToken,
			"refresh_token": loginResult.RefreshToken,
			"expires_in":    loginResult.ExpiresIn,
		},
		"error": nil,
	})
	h.audit.LogAuth(c, "auth.setup", req.Username)
}
