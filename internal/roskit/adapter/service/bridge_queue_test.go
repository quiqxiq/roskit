//go:build mikrotik

package service_test

import (
	"context"
	"testing"

	"github.com/quiqxiq/roskit/internal/roskit/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBridge_ListQueues_Fields(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	queues, err := bridge.ListQueues(ctx, rid)
	require.NoError(t, err)
	assert.NotNil(t, queues)
}

func TestBridge_AddSetRemoveQueue(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("q")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	name := testhelpers.UniqueName("q")
	reply, err := bridge.Mutate(ctx, rid, "queue/simple/add",
		"=name="+name,
		"=target=192.168.88.254/32",
		"=max-limit=1M/1M",
		"=comment="+prefix,
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

	_, err = bridge.Mutate(ctx, rid, "queue/simple/set",
		"=.id="+retID,
		"=max-limit=2M/2M",
	)
	require.NoError(t, err)

	found, err := bridge.QueryOne(ctx, rid, "queue/simple/print", "?.id="+retID)
	require.NoError(t, err)
	assert.Equal(t, name, found["name"])

	_, err = bridge.Mutate(ctx, rid, "queue/simple/remove", "=.id="+retID)
	require.NoError(t, err)
}

func TestBridge_ListQueueTree(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	tree, err := bridge.Query(ctx, rid, "queue/tree/print")
	require.NoError(t, err)
	assert.NotNil(t, tree)
}

func TestBridge_ListQueueTypes(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	types, err := bridge.Query(ctx, rid, "queue/type/print")
	require.NoError(t, err)
	assert.NotEmpty(t, types)
}
