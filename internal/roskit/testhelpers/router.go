//go:build mikrotik

package testhelpers

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/adapter/service"
	"github.com/quiqxiq/roskit/internal/roskit/execution"
	"github.com/quiqxiq/roskit/internal/roskit/orchestrator"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/cache"
)

func LoadRouterConfig() execution.ConnConfig {
	host := os.Getenv("MIKROTIK_HOST")
	user := os.Getenv("MIKROTIK_USER")
	pass := os.Getenv("MIKROTIK_PASS")
	id := os.Getenv("MIKROTIK_ID")
	if id == "" {
		id = "test"
	}
	if host == "" {
		host = "192.168.233.1:8728"
	}
	if user == "" {
		user = "admin"
	}
	if pass == "" {
		pass = "r00t"
	}
	return execution.ConnConfig{
		RouterID:        id,
		Address:         host,
		Username:        user,
		Password:        pass,
		DialTimeout:     10 * time.Second,
		HealthInterval:  60 * time.Second,
	}
}

func SkipWithoutMikroTik(t *testing.T) {
	t.Helper()
	if os.Getenv("MIKROTIK_HOST") == "" {
		t.Skip("skipping: MIKROTIK_HOST not set")
	}
}

func RouterID() string {
	id := os.Getenv("MIKROTIK_ID")
	if id == "" {
		return "test"
	}
	return id
}

func NewTestPool(t *testing.T) (*execution.Pool, func()) {
	t.Helper()
	cfg := LoadRouterConfig()
	pool := execution.NewPool(slog.Default())
	pool.Register(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	pool.Start(ctx)

	waitCtx, waitCancel := context.WithTimeout(ctx, 15*time.Second)
	defer waitCancel()
	if err := pool.WaitConnected(waitCtx, cfg.RouterID); err != nil {
		cancel()
		pool.Stop()
		t.Fatalf("pool did not connect: %v", err)
	}

	cleanup := func() {
		cancel()
		pool.Stop()
	}
	return pool, cleanup
}

func NewTestEngine(t *testing.T) (*orchestrator.Engine, func()) {
	t.Helper()
	cfg := LoadRouterConfig()
	engine := orchestrator.New(orchestrator.Config{
		Logger: slog.Default(),
	})
	ctx, cancel := context.WithCancel(context.Background())
	engine.AddRouter(ctx, cfg)
	engine.Start(ctx)

	time.Sleep(3 * time.Second)

	cleanup := func() {
		engine.Stop()
		cancel()
	}
	return engine, cleanup
}

func NewTestBridge(t *testing.T) (*service.Bridge, func()) {
	t.Helper()
	cfg := LoadRouterConfig()
	engine := orchestrator.New(orchestrator.Config{
		Logger: slog.Default(),
	})
	ctx, cancel := context.WithCancel(context.Background())
	engine.AddRouter(ctx, cfg)
	engine.Start(ctx)

	time.Sleep(3 * time.Second)

	bridge := service.NewBridge(engine.Dispatcher(), cache.NoopRepository{})

	cleanup := func() {
		engine.Stop()
		cancel()
	}
	return bridge, cleanup
}

func UniqueName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano()%1000000)
}
