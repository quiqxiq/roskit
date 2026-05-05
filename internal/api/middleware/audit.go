package middleware

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roskit/internal/models"
	"github.com/quiqxiq/roskit/internal/repository"
)

type AuditLogger struct {
	repo   repository.AuditRepository
	logger *slog.Logger
}

func NewAuditLogger(repo repository.AuditRepository) *AuditLogger {
	return &AuditLogger{
		repo:   repo,
		logger: slog.Default().With("component", "audit"),
	}
}

func (a *AuditLogger) Log(c *gin.Context, action, entity, entityID, details string) {
	if a == nil || a.repo == nil {
		return
	}

	var userID *uint
	if v, ok := c.Get("userID"); ok {
		if id, ok := v.(uint); ok {
			u := id
			userID = &u
		}
	}

	var tenantID *uint
	if v, ok := c.Get("tenantID"); ok {
		if id, ok := v.(uint); ok {
			t := id
			tenantID = &t
		}
	}

	entry := &models.AuditLog{
		TenantID:  tenantID,
		UserID:    userID,
		Action:    action,
		Entity:    entity,
		EntityID:  entityID,
		Details:   details,
		IPAddress: c.ClientIP(),
	}

	go func() {
		ctx := context.Background()
		if err := a.repo.Create(ctx, entry); err != nil {
			a.logger.Warn("audit log write failed", "action", action, "error", err)
		}
	}()
}

func (a *AuditLogger) LogRouter(c *gin.Context, action string, routerID uint, name string) {
	a.Log(c, action, "router", fmt.Sprintf("%d", routerID), name)
}

func (a *AuditLogger) LogAuth(c *gin.Context, action, username string) {
	a.Log(c, action, "auth", "", username)
}

func (a *AuditLogger) LogVoucher(c *gin.Context, action string, routerID uint, details string) {
	a.Log(c, action, "voucher", fmt.Sprintf("%d", routerID), details)
}

func (a *AuditLogger) LogSales(c *gin.Context, action string, routerID uint, details string) {
	a.Log(c, action, "sales", fmt.Sprintf("%d", routerID), details)
}

func (a *AuditLogger) LogTenant(c *gin.Context, action string, tenantID uint, name string) {
	a.Log(c, action, "tenant", fmt.Sprintf("%d", tenantID), name)
}

func (a *AuditLogger) LogUser(c *gin.Context, action string, userID uint, username string) {
	a.Log(c, action, "user", fmt.Sprintf("%d", userID), username)
}
