//go:build mikrotik

package service_test

import (
	"context"
	"testing"

	"github.com/quiqxiq/roskit/internal/roskit/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBridge_GetSystemClock(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	clock, err := bridge.GetSystemClock(ctx, rid)
	require.NoError(t, err)
	assert.Contains(t, clock, "time")
	assert.Contains(t, clock, "date")
}

func TestBridge_GetSystemHealth(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	health, err := bridge.GetSystemHealth(ctx, rid)
	if err != nil {
		t.Skipf("system/health not supported on this device: %v", err)
	}
	assert.NotNil(t, health)
}

func TestBridge_GetRouterboard(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	rb, err := bridge.GetRouterboard(ctx, rid)
	if err != nil {
		t.Skipf("system/routerboard not supported on this device: %v", err)
	}
	assert.NotNil(t, rb)
}

func TestBridge_GetSystemLog(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	logs, err := bridge.GetSystemLog(ctx, rid, "")
	require.NoError(t, err)
	assert.NotNil(t, logs)
}

func TestBridge_GetSystemLog_Filtered(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	logs, err := bridge.GetSystemLog(ctx, rid, "system,info")
	require.NoError(t, err)
	assert.NotNil(t, logs)
}

func TestBridge_EnableDisableScheduler(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("sys")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	name := testhelpers.UniqueName("sys")
	id, err := bridge.AddScheduler(ctx, rid, map[string]string{
		"name":     name,
		"interval": "1h",
		"on-event": ":put test",
		"comment":  prefix,
	})
	require.NoError(t, err)

	err = bridge.DisableScheduler(ctx, rid, id)
	require.NoError(t, err)

	err = bridge.EnableScheduler(ctx, rid, id)
	require.NoError(t, err)

	_ = bridge.RemoveScheduler(ctx, rid, id)
}

func TestBridge_RunScript(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("sys")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	name := testhelpers.UniqueName("sys")
	id, err := bridge.AddScript(ctx, rid, map[string]string{
		"name":    name,
		"source":  ":put hello",
		"comment": prefix,
	})
	require.NoError(t, err)

	err = bridge.RunScript(ctx, rid, id)
	require.NoError(t, err)

	_ = bridge.RemoveScript(ctx, rid, id)
}

func TestBridge_SetupLogging(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	err := bridge.SetupLogging(ctx, rid)
	require.NoError(t, err)

	err = bridge.SetupLogging(ctx, rid)
	require.NoError(t, err)
}
