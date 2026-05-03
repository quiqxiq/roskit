//go:build mikrotik

package execution

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPool_RegisterAndStart(t *testing.T) {
	skipWithoutMikroTik(t)
	cfg := loadRouterConfig()
	pool := NewPool(slog.Default())
	pool.Register(cfg)
	ctx := context.Background()
	pool.Start(ctx)
	defer pool.Stop()
	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	err := pool.WaitConnected(waitCtx, cfg.RouterID)
	require.NoError(t, err)
	assert.Equal(t, ConnStateConnected, pool.Status()[cfg.RouterID])
}

func TestPool_Borrow_Success(t *testing.T) {
	skipWithoutMikroTik(t)
	pool, cleanup := newTestPool(t)
	defer cleanup()
	rid := routerID()
	conn, err := pool.Borrow(context.Background(), rid)
	require.NoError(t, err)
	assert.Equal(t, ConnStateConnected, conn.State())
}

func TestPool_Borrow_NotRegistered(t *testing.T) {
	skipWithoutMikroTik(t)
	pool := NewPool(slog.Default())
	_, err := pool.Borrow(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestPool_Borrow_NotConnected(t *testing.T) {
	skipWithoutMikroTik(t)
	cfg := loadRouterConfig()
	pool := NewPool(slog.Default())
	pool.Register(cfg)
	_, err := pool.Borrow(context.Background(), cfg.RouterID)
	assert.Error(t, err)
}

func TestPool_Unregister(t *testing.T) {
	skipWithoutMikroTik(t)
	pool, cleanup := newTestPool(t)
	defer cleanup()
	rid := routerID()
	pool.Unregister(rid)
	_, exists := pool.Status()[rid]
	assert.False(t, exists)
}

func TestPool_LaunchOne(t *testing.T) {
	skipWithoutMikroTik(t)
	cfg := loadRouterConfig()
	pool := NewPool(slog.Default())
	ctx := context.Background()
	pool.Start(ctx)
	defer pool.Stop()
	pool.Register(cfg)
	pool.LaunchOne(cfg.RouterID)
	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	err := pool.WaitConnected(waitCtx, cfg.RouterID)
	require.NoError(t, err)
	assert.Equal(t, ConnStateConnected, pool.Status()[cfg.RouterID])
}

func TestPool_Stop(t *testing.T) {
	skipWithoutMikroTik(t)
	pool, _ := newTestPool(t)
	assert.NotPanics(t, func() { pool.Stop() })
}

func TestPool_WaitConnected_Timeout(t *testing.T) {
	skipWithoutMikroTik(t)
	cfg := loadRouterConfig()
	pool := NewPool(slog.Default())
	pool.Register(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := pool.WaitConnected(ctx, cfg.RouterID)
	assert.Error(t, err)
}

func TestPool_ConcurrentBorrow(t *testing.T) {
	skipWithoutMikroTik(t)
	pool, cleanup := newTestPool(t)
	defer cleanup()
	rid := routerID()
	var wg sync.WaitGroup
	errs := make([]error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			conn, err := pool.Borrow(context.Background(), rid)
			if err != nil {
				errs[idx] = err
				return
			}
			if conn.State() != ConnStateConnected {
				errs[idx] = fmt.Errorf("unexpected state: %s", conn.State())
			}
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		assert.NoError(t, err, "goroutine %d", i)
	}
}
