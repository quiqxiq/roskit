//go:build mikrotik

package orchestrator_test

import (
	"context"
	"testing"
	"github.com/quiqxiq/roskit/internal/roskit/adapter/service"
	"github.com/quiqxiq/roskit/internal/roskit/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_StartStop(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	engine, cleanup := testhelpers.NewTestEngine(t)
	t.Cleanup(cleanup)

	status := engine.Status()
	rid := testhelpers.RouterID()
	require.Contains(t, status, rid)
	assert.Contains(t, status[rid], "connected")

	engine.Stop()
}

func TestEngine_ExecuteCommand(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	engine, cleanup := testhelpers.NewTestEngine(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	err := engine.ExecuteCommand(ctx, rid, "/system/identity/print")
	require.NoError(t, err)
}

func TestEngine_Dispatcher_Query(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	engine, cleanup := testhelpers.NewTestEngine(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	results, err := engine.Dispatcher().Query(ctx, rid, "ip/hotspot/user/print")
	require.NoError(t, err)
	assert.NotNil(t, results)
}

func TestEngine_Dispatcher_Mutate(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	engine, cleanup := testhelpers.NewTestEngine(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("eng")
	bridge := service.NewBridge(engine.Dispatcher())
	t.Cleanup(func() { testhelpers.CleanupHotspotUsers(ctx, bridge, rid, prefix) })

	name := testhelpers.UniqueName("eng")
	reply, err := engine.Dispatcher().Mutate(ctx, rid, "ip/hotspot/user/add",
		"=name="+name, "=password=pass", "=profile=default",
	)
	require.NoError(t, err)
	require.NotNil(t, reply)

	retID := ""
	if reply.Done != nil {
		for _, pair := range reply.Done.List {
			if pair.Key == "ret" {
				retID = pair.Value
			}
		}
	}
	assert.NotEmpty(t, retID)

	if retID != "" {
		_, _ = engine.Dispatcher().Mutate(ctx, rid, "ip/hotspot/user/remove", "=.id="+retID)
	}
}

func TestEngine_Status(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	engine, cleanup := testhelpers.NewTestEngine(t)
	t.Cleanup(cleanup)

	status := engine.Status()
	assert.NotEmpty(t, status)

	rid := testhelpers.RouterID()
	require.Contains(t, status, rid)
	assert.Contains(t, status[rid], "connected")
}
