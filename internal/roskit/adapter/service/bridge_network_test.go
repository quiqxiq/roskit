//go:build mikrotik

package service_test

import (
	"context"
	"testing"

	"github.com/quiqxiq/roskit/internal/roskit/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBridge_ListIPPools(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	pools, err := bridge.ListIPPools(ctx, rid)
	require.NoError(t, err)
	assert.NotNil(t, pools)
}

func TestBridge_ListNATRules(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	rules, err := bridge.ListNATRules(ctx, rid)
	require.NoError(t, err)
	assert.NotNil(t, rules)
}

func TestBridge_ListQueues(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	queues, err := bridge.ListQueues(ctx, rid)
	require.NoError(t, err)
	assert.NotNil(t, queues)
}

func TestBridge_FindARPByAddress(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	_, err := bridge.FindARPByAddress(ctx, rid, "192.168.88.254")
	if err != nil {
		assert.Error(t, err)
	}
}

func TestBridge_ListDHCPLeases(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	leases, err := bridge.ListDHCPLeases(ctx, rid)
	require.NoError(t, err)
	assert.NotNil(t, leases)
}

func TestBridge_FindSchedulerByName(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("t")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	result, err := bridge.FindSchedulerByName(ctx, rid, prefix+"-nonexistent")
	if err != nil {
		assert.Error(t, err)
	} else {
		assert.Nil(t, result)
	}
}
