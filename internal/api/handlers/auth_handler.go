package handlers

import (
	stderrors "errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/quiqxiq/roskit/internal/api/middleware"
	casbinx "github.com/quiqxiq/roskit/internal/casbin"
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
	Tenant   string `json:"tenant" binding:"omitempty,max=100"`
	Username string `json:"username" binding:"required,min=1,max=64"`
	Password string `json:"password" binding:"required,min=1,max=128"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required,min=10,max=4096"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"omitempty,max=4096"`
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required,min=1,max=128"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=128"`
}

type setupRequest struct {
	TenantName string `json:"tenant_name" binding:"required,min=2,max=100"`
	TenantSlug string `json:"tenant_slug" binding:"required,min=2,max=100"`
	Username   string `json:"username" binding:"required,min=3,max=64"`
	Password   string `json:"password" binding:"required,min=6,max=128"`
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
	ctx := c.Request.Context()

	count, err := h.svc.UserCount(ctx)
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

	// Hash before opening the transaction — bcrypt is slow (~100ms) and
	// holding a connection through it under load wastes pool capacity.
	pwHash, err := h.svc.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "internal error"})
		return
	}

	result, err := h.tenantSvc.SetupTenantWithOwner(ctx, services.SetupTenantWithOwnerParams{
		TenantName:   req.TenantName,
		TenantSlug:   req.TenantSlug,
		Username:     req.Username,
		PasswordHash: pwHash,
	})
	if err != nil {
		// Map known error shapes from SetupTenantWithOwner. Anything else
		// surfaces as 500 with the original message stripped.
		msg := err.Error()
		switch {
		case stderrors.Is(err, services.ErrUserAlreadyExists),
			containsAny(msg, "user already exists", "duplicate", "unique"):
			c.JSON(http.StatusConflict, gin.H{"data": nil, "error": "user already exists"})
		case containsAny(msg, "invalid slug", "reserved"):
			c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": msg})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "setup failed"})
		}
		return
	}

	// Casbin role grant lives outside the DB transaction because the
	// enforcer is in-memory; failure here is rare and recoverable.
	if enforcer := h.svc.Enforcer(); enforcer != nil {
		if err := casbinx.AssignRole(enforcer, result.User.ID, result.Tenant.Slug, models.UserRoleOwner); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "role assignment failed"})
			return
		}
	}

	loginResult, err := h.svc.Login(ctx, result.Tenant.Slug, req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "auto-login after setup failed"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": gin.H{
			"tenant": gin.H{
				"id":   result.Tenant.ID,
				"name": result.Tenant.Name,
				"slug": result.Tenant.Slug,
			},
			"user": gin.H{
				"id":       result.User.ID,
				"username": result.User.Username,
				"role":     result.User.Role,
			},
			"access_token":  loginResult.AccessToken,
			"refresh_token": loginResult.RefreshToken,
			"expires_in":    loginResult.ExpiresIn,
		},
		"error": nil,
	})
	h.audit.LogAuth(c, "auth.setup", req.Username)
}

// containsAny returns true when the lower-cased haystack contains any of the
// given lower-cased needles. Used to recognise wrapped error messages
// returned from gorm/postgres without imposing a custom error sentinel
// throughout the service layer.
func containsAny(haystack string, needles ...string) bool {
	lower := strings.ToLower(haystack)
	for _, n := range needles {
		if strings.Contains(lower, n) {
			return true
		}
	}
	return false
}
