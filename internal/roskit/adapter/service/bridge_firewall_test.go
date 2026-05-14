//go:build mikrotik

package service_test

import (
	"context"
	"testing"

	"github.com/quiqxiq/roskit/internal/roskit/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBridge_ListFirewallFilter(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	rules, err := bridge.Query(ctx, rid, "ip/firewall/filter/print")
	require.NoError(t, err)
	assert.NotNil(t, rules)
}

func TestBridge_ListFirewallMangle(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	rules, err := bridge.Query(ctx, rid, "ip/firewall/mangle/print")
	require.NoError(t, err)
	assert.NotNil(t, rules)
}

func TestBridge_ListFirewallAddressList(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	list, err := bridge.Query(ctx, rid, "ip/firewall/address-list/print")
	require.NoError(t, err)
	assert.NotNil(t, list)
}

func TestBridge_ListFirewallNAT_FieldKeys(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	rules, err := bridge.ListNATRules(ctx, rid)
	require.NoError(t, err)
	if len(rules) > 0 {
		assert.Contains(t, rules[0], "chain")
		assert.Contains(t, rules[0], "action")
	}
}
