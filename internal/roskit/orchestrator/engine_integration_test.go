//go:build mikrotik

package orchestrator_test

import (
	"context"
	"testing"
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/adapter/service"
	_ "github.com/quiqxiq/roskit/internal/roskit/core/definition"
	"github.com/quiqxiq/roskit/internal/roskit/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_QueryReturnsLiveData(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	engine, cleanup := testhelpers.NewTestEngine(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	bridge := service.NewBridge(engine.Dispatcher())

	res, err := bridge.GetSystemResource(ctx, rid)
	require.NoError(t, err)
	assert.Contains(t, res, "version")
	assert.Contains(t, res, "uptime")
	assert.NotEmpty(t, res["version"])
}

func TestEngine_MutateAndQueryConsistency(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	engine, cleanup := testhelpers.NewTestEngine(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	bridge := service.NewBridge(engine.Dispatcher())
	prefix := testhelpers.UniqueName("eng")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	name := testhelpers.UniqueName("eng")
	id, err := bridge.AddHotspotUser(ctx, rid, map[string]string{
		"name":     name,
		"password": "pass123",
		"profile":  "default",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	user, err := bridge.GetHotspotUser(ctx, rid, name)
	require.NoError(t, err)
	assert.Equal(t, name, user["name"])
	assert.Equal(t, "default", user["profile"])

	err = bridge.SetHotspotUser(ctx, rid, id, map[string]string{
		"comment": "engine-test-updated",
	})
	require.NoError(t, err)

	updated, err := bridge.GetHotspotUser(ctx, rid, id)
	require.NoError(t, err)
	assert.Equal(t, "engine-test-updated", updated["comment"])

	err = bridge.RemoveHotspotUser(ctx, rid, id)
	require.NoError(t, err)

	_, err = bridge.GetHotspotUser(ctx, rid, name)
	assert.Error(t, err, "deleted user should not be found")
}

func TestEngine_MultiDomainQuery(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	engine, cleanup := testhelpers.NewTestEngine(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rid := testhelpers.RouterID()
	bridge := service.NewBridge(engine.Dispatcher())

	type domainTest struct {
		name string
		path string
	}
	domains := []domainTest{
		{"system_resource", "system/resource/print"},
		{"system_identity", "system/identity/print"},
		{"interfaces", "interface/print"},
		{"ip_addresses", "ip/address/print"},
		{"ip_routes", "ip/route/print"},
		{"ip_pools", "ip/pool/print"},
		{"firewall_filter", "ip/firewall/filter/print"},
		{"firewall_nat", "ip/firewall/nat/print"},
		{"dhcp_leases", "ip/dhcp-server/lease/print"},
		{"hotspot_users", "ip/hotspot/user/print"},
		{"hotspot_profiles", "ip/hotspot/user/profile/print"},
		{"ppp_secrets", "ppp/secret/print"},
		{"ppp_profiles", "ppp/profile/print"},
		{"schedulers", "system/scheduler/print"},
		{"scripts", "system/script/print"},
	}

	for _, d := range domains {
		t.Run(d.name, func(t *testing.T) {
			result, err := bridge.Query(ctx, rid, d.path)
			require.NoError(t, err, "query %s failed", d.path)
			assert.NotNil(t, result, "%s returned nil", d.path)
		})
	}
}
