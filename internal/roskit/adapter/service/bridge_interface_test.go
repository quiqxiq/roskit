//go:build mikrotik

package service_test

import (
	"context"
	"testing"

	"github.com/quiqxiq/roskit/internal/roskit/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBridge_GetInterfaceTraffic(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	ifaces, err := bridge.ListInterfaces(ctx, rid)
	require.NoError(t, err)
	require.NotEmpty(t, ifaces)

	ifName := ifaces[0]["name"]
	reply, err := bridge.Run(ctx, rid,
		"/interface/monitor-traffic", "=interface="+ifName, "=once=")
	require.NoError(t, err)
	require.NotEmpty(t, reply.Re)

	data := reply.Re[0].Map
	assert.Contains(t, data, "rx-bits-per-second")
	assert.Contains(t, data, "tx-bits-per-second")
}

func TestBridge_ListVLANs(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	vlans, err := bridge.Query(ctx, rid, "interface/vlan/print")
	require.NoError(t, err)
	assert.NotNil(t, vlans)
}

func TestBridge_ListBridges(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	bridges, err := bridge.Query(ctx, rid, "interface/bridge/print")
	require.NoError(t, err)
	assert.NotNil(t, bridges)
}

func TestBridge_ListBridgePorts(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	ports, err := bridge.Query(ctx, rid, "interface/bridge/port/print")
	require.NoError(t, err)
	assert.NotNil(t, ports)
}

func TestBridge_ListEthernetInterfaces(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	ethers, err := bridge.Query(ctx, rid, "interface/ethernet/print")
	require.NoError(t, err)
	assert.NotNil(t, ethers)
	if len(ethers) > 0 {
		assert.Contains(t, ethers[0], "name")
	}
}

func TestBridge_ListAddresses(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	addrs, err := bridge.Query(ctx, rid, "ip/address/print")
	require.NoError(t, err)
	assert.NotNil(t, addrs)
	if len(addrs) > 0 {
		assert.Contains(t, addrs[0], "address")
		assert.Contains(t, addrs[0], "interface")
	}
}

func TestBridge_ListRoutes(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	routes, err := bridge.Query(ctx, rid, "ip/route/print")
	require.NoError(t, err)
	assert.NotNil(t, routes)
}

func TestBridge_ListNeighbors(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	neighbors, err := bridge.Query(ctx, rid, "ip/neighbor/print")
	require.NoError(t, err)
	assert.NotNil(t, neighbors)
}

func TestBridge_ListDNSStatic(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	dns, err := bridge.Query(ctx, rid, "ip/dns/static/print")
	require.NoError(t, err)
	assert.NotNil(t, dns)
}

func TestBridge_ListServices(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	services, err := bridge.Query(ctx, rid, "ip/service/print")
	require.NoError(t, err)
	assert.NotEmpty(t, services)
	if len(services) > 0 {
		assert.Contains(t, services[0], "name")
		assert.Contains(t, services[0], "port")
	}
}

func TestBridge_ListPoolUsed(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	used, err := bridge.Query(ctx, rid, "ip/pool/used/print")
	require.NoError(t, err)
	assert.NotNil(t, used)
}

func TestBridge_ListARPEntries(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	arps, err := bridge.Query(ctx, rid, "ip/arp/print")
	require.NoError(t, err)
	assert.NotNil(t, arps)
}
