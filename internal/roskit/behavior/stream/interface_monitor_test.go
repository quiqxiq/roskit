//go:build mikrotik

package stream_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/behavior/stream"
	_ "github.com/quiqxiq/roskit/internal/roskit/core/definition"
	"github.com/quiqxiq/roskit/internal/roskit/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInterfaceMonitorManager_SyncInterfaces_StartsMonitors(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)
	pool, cleanup := testhelpers.NewTestPool(t)
	defer cleanup()

	routerID := testhelpers.RouterID()
	sink := &capturingStreamSink{}
	mgr := stream.NewInterfaceMonitorManager(pool, sink, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mgr.SyncInterfaces(ctx, routerID, []string{"ether1"})

	time.Sleep(500 * time.Millisecond)
	assert.Equal(t, 1, mgr.ActiveCount())

	mgr.StopAll(routerID)
	time.Sleep(500 * time.Millisecond)
	assert.Equal(t, 0, mgr.ActiveCount())
}

func TestInterfaceMonitorManager_SyncInterfaces_Idempotent(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)
	pool, cleanup := testhelpers.NewTestPool(t)
	defer cleanup()

	routerID := testhelpers.RouterID()
	sink := &capturingStreamSink{}
	mgr := stream.NewInterfaceMonitorManager(pool, sink, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mgr.SyncInterfaces(ctx, routerID, []string{"ether1"})
	mgr.SyncInterfaces(ctx, routerID, []string{"ether1"}) // second call — should not add duplicate

	time.Sleep(500 * time.Millisecond)
	assert.Equal(t, 1, mgr.ActiveCount(), "duplicate SyncInterfaces should not spawn extra monitors")

	mgr.Shutdown()
}

func TestInterfaceMonitorManager_SyncInterfaces_RemovesStale(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)
	pool, cleanup := testhelpers.NewTestPool(t)
	defer cleanup()

	routerID := testhelpers.RouterID()
	sink := &capturingStreamSink{}
	mgr := stream.NewInterfaceMonitorManager(pool, sink, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mgr.SyncInterfaces(ctx, routerID, []string{"ether1", "ether2"})
	time.Sleep(300 * time.Millisecond)
	require.Equal(t, 2, mgr.ActiveCount())

	// Sync to only ether1 — ether2 should be stopped
	mgr.SyncInterfaces(ctx, routerID, []string{"ether1"})
	time.Sleep(300 * time.Millisecond)
	assert.Equal(t, 1, mgr.ActiveCount())

	mgr.Shutdown()
}

func TestInterfaceMonitorManager_Shutdown_StopsAll(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)
	pool, cleanup := testhelpers.NewTestPool(t)
	defer cleanup()

	routerID := testhelpers.RouterID()
	sink := &capturingStreamSink{}
	mgr := stream.NewInterfaceMonitorManager(pool, sink, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mgr.SyncInterfaces(ctx, routerID, []string{"ether1"})
	time.Sleep(300 * time.Millisecond)
	assert.Greater(t, mgr.ActiveCount(), 0)

	mgr.Shutdown()
	assert.Equal(t, 0, mgr.ActiveCount())
}
