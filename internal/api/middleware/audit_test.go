package middleware

import (
	"context"
	"errors"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/quiqxiq/roskit/internal/models"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// captureAuditRepo records every Create call so tests can assert what the
// background worker drained.
type captureAuditRepo struct {
	mu      sync.Mutex
	entries []*models.AuditLog
	delay   time.Duration
	failN   int32 // number of initial calls to fail
	calls   int32
}

func (c *captureAuditRepo) Create(ctx context.Context, log *models.AuditLog) error {
	if c.delay > 0 {
		select {
		case <-time.After(c.delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if atomic.AddInt32(&c.calls, 1) <= atomic.LoadInt32(&c.failN) {
		return errors.New("simulated failure")
	}
	c.mu.Lock()
	c.entries = append(c.entries, log)
	c.mu.Unlock()
	return nil
}

func (c *captureAuditRepo) ListByTenant(_ context.Context, _ uint, _ int) ([]*models.AuditLog, error) {
	return nil, nil
}

func (c *captureAuditRepo) ListByUser(_ context.Context, _ uint, _ uint, _ int) ([]*models.AuditLog, error) {
	return nil, nil
}

func (c *captureAuditRepo) snapshot() []*models.AuditLog {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]*models.AuditLog, len(c.entries))
	copy(out, c.entries)
	return out
}

func newGinCtx() *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	return c
}

func TestAuditLogger_DrainsOnShutdown(t *testing.T) {
	repo := &captureAuditRepo{}
	a := NewAuditLogger(repo)

	c := newGinCtx()
	c.Set("userID", uint(7))

	for i := 0; i < 20; i++ {
		a.LogAuth(c, "login", "alice")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown returned %v", err)
	}

	got := repo.snapshot()
	if len(got) != 20 {
		t.Fatalf("expected 20 audit entries persisted, got %d", len(got))
	}
	for _, e := range got {
		if e.UserID == nil || *e.UserID != 7 {
			t.Errorf("UserID not propagated: %+v", e.UserID)
		}
	}
}

func TestAuditLogger_ShutdownIsIdempotent(t *testing.T) {
	a := NewAuditLogger(&captureAuditRepo{})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := a.Shutdown(ctx); err != nil {
		t.Fatalf("first shutdown: %v", err)
	}
	if err := a.Shutdown(ctx); err != nil {
		t.Fatalf("second shutdown should be no-op, got %v", err)
	}
}

func TestAuditLogger_DropsOnFullChannelWithoutBlocking(t *testing.T) {
	// Slow repo means the worker can't drain fast; we then submit far more
	// entries than the channel can hold and assert Log() never blocks.
	repo := &captureAuditRepo{delay: 50 * time.Millisecond}
	a := NewAuditLogger(repo)

	c := newGinCtx()

	// Cap submission time well below what `auditChannelSize * 50ms` would
	// take if Log() blocked — Log must drop excess entries instead.
	deadline := time.Now().Add(500 * time.Millisecond)
	submitted := 0
	for time.Now().Before(deadline) {
		a.LogAuth(c, "spam", "x")
		submitted++
		if submitted > auditChannelSize*4 {
			break
		}
	}

	if submitted < auditChannelSize*2 {
		t.Fatalf("Log() appears to have blocked: only submitted %d in 500ms", submitted)
	}

	// Shutdown with generous timeout; some entries will land, some dropped.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = a.Shutdown(ctx)
}
