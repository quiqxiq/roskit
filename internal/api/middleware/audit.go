package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roskit/internal/models"
	"github.com/quiqxiq/roskit/internal/repository"
)

// auditChannelSize bounds how many entries can buffer between Log() and the
// background worker. When full, Log drops oldest entries to keep the API
// hot path non-blocking — the alternative (blocking) would let DB latency
// stall every request.
const auditChannelSize = 1024

// auditWriteTimeout caps each repo.Create call. Any single slow DB write
// can't stall the worker beyond this.
const auditWriteTimeout = 5 * time.Second

// AuditLogger persists audit events asynchronously through a single worker
// goroutine fed by a buffered channel. This replaces the previous
// per-request goroutine pattern, which lost entries on shutdown and could
// pile up unbounded under DB pressure.
type AuditLogger struct {
	repo   repository.AuditRepository
	logger *slog.Logger

	ch        chan *models.AuditLog
	done      chan struct{}
	closeOnce sync.Once
	workerWG  sync.WaitGroup
}

func NewAuditLogger(repo repository.AuditRepository) *AuditLogger {
	a := &AuditLogger{
		repo:   repo,
		logger: slog.Default().With("component", "audit"),
		ch:     make(chan *models.AuditLog, auditChannelSize),
		done:   make(chan struct{}),
	}
	a.workerWG.Add(1)
	go a.worker()
	return a
}

// worker drains a.ch until it is closed, then exits. The single-goroutine
// design serialises DB writes (one connection at a time) and gives a clean
// shutdown signal.
func (a *AuditLogger) worker() {
	defer a.workerWG.Done()
	for entry := range a.ch {
		ctx, cancel := context.WithTimeout(context.Background(), auditWriteTimeout)
		if err := a.repo.Create(ctx, entry); err != nil {
			a.logger.Warn("audit log write failed",
				"action", entry.Action, "entity", entry.Entity, "error", err)
		}
		cancel()
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

	// Non-blocking enqueue: drop on overflow rather than stall the request.
	// We surface drops in logs so an operator notices before correlation gaps
	// become a problem.
	select {
	case a.ch <- entry:
	case <-a.done:
		// Logger is shutting down; quietly drop.
	default:
		a.logger.Warn("audit channel full, dropping entry",
			"action", action, "entity", entity)
	}
}

// Shutdown stops accepting new entries, drains the channel, and waits for
// the worker to finish. Idempotent. Should be called from main()'s signal
// handler before the process exits to avoid losing buffered audit entries.
func (a *AuditLogger) Shutdown(ctx context.Context) error {
	if a == nil {
		return nil
	}
	a.closeOnce.Do(func() {
		close(a.done)
		close(a.ch)
	})

	doneCh := make(chan struct{})
	go func() {
		a.workerWG.Wait()
		close(doneCh)
	}()

	select {
	case <-doneCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
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
