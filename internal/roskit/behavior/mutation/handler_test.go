//go:build mikrotik

package mutation_test

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/quiqxiq/roskit/internal/roskit/behavior/mutation"
	"github.com/quiqxiq/roskit/internal/roskit/core/command"
	_ "github.com/quiqxiq/roskit/internal/roskit/core/definition"
	"github.com/quiqxiq/roskit/internal/roskit/execution"
	"github.com/quiqxiq/roskit/internal/roskit/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMutation_AddHotspotUser(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)
	pool, cleanup := testhelpers.NewTestPool(t)
	defer cleanup()

	routerID := testhelpers.RouterID()
	h := mutation.NewHandler(pool, nil, nil, slog.Default())

	userName := testhelpers.UniqueName("mut-test")
	meta := command.Lookup("ip/hotspot/user/add")
	require.NotNil(t, meta)

	reply, err := h.Execute(context.Background(), routerID, meta,
		fmt.Sprintf("=name=%s", userName),
		"=password=pass",
		"=profile=default",
	)
	require.NoError(t, err)
	require.NotNil(t, reply)

	exec := execution.NewExecutor(pool)
	exec.Run(context.Background(), routerID,
		"/ip/hotspot/user/remove",
		fmt.Sprintf("=.id=%s", userName),
	)
}

func TestMutation_RemoveHotspotUser(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)
	pool, cleanup := testhelpers.NewTestPool(t)
	defer cleanup()

	routerID := testhelpers.RouterID()
	exec := execution.NewExecutor(pool)

	userName := testhelpers.UniqueName("mut-rm")
	_, err := exec.Run(context.Background(), routerID,
		"/ip/hotspot/user/add",
		fmt.Sprintf("=name=%s", userName),
		"=password=pass",
		"=profile=default",
	)
	require.NoError(t, err)

	h := mutation.NewHandler(pool, nil, nil, slog.Default())
	removeMeta := command.Lookup("ip/hotspot/user/remove")
	require.NotNil(t, removeMeta)

	_, err = h.Execute(context.Background(), routerID, removeMeta,
		fmt.Sprintf("=.id=%s", userName),
	)
	assert.NoError(t, err)
}

func TestMutation_NilMeta(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)
	pool, cleanup := testhelpers.NewTestPool(t)
	defer cleanup()

	routerID := testhelpers.RouterID()
	h := mutation.NewHandler(pool, nil, nil, slog.Default())

	_, err := h.Execute(context.Background(), routerID, nil)
	assert.Error(t, err)
}

func TestMutation_AddRemoveIPBinding(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)
	pool, cleanup := testhelpers.NewTestPool(t)
	defer cleanup()

	routerID := testhelpers.RouterID()
	h := mutation.NewHandler(pool, nil, nil, slog.Default())

	mac := "AA:BB:CC:DD:EE:FF"
	addMeta := command.Lookup("ip/hotspot/ip-binding/add")
	require.NotNil(t, addMeta)

	reply, err := h.Execute(context.Background(), routerID, addMeta,
		fmt.Sprintf("=mac-address=%s", mac),
		"=type=bypassed",
		"=comment=test-binding",
	)
	require.NoError(t, err)
	require.NotNil(t, reply)

	removeMeta := command.Lookup("ip/hotspot/ip-binding/remove")
	require.NotNil(t, removeMeta)

	_, err = h.Execute(context.Background(), routerID, removeMeta,
		fmt.Sprintf("=.id=%s", mac),
	)
	assert.NoError(t, err)
}
