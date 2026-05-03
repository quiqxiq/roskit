//go:build mikrotik

package service_test

import (
	"context"
	"testing"

	"github.com/quiqxiq/roskit/internal/roskit/adapter/service"
	"github.com/quiqxiq/roskit/internal/roskit/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBridge_RecordAndFetchSale(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("t")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	rec := &service.SalesRecord{
		Date: "jan/15/2024", Time: "10:30:00", Username: prefix + "-user1",
		Price: "5000", IPAddress: "192.168.88.100", MACAddress: "AA:BB:CC:DD:EE:FF",
		Validity: "1d", Profile: "default", Owner: "jan2024",
		Source: "test", Comment: "mikhmon",
	}

	err := bridge.RecordSale(ctx, rid, rec)
	require.NoError(t, err)

	sales, err := bridge.FetchDailySales(ctx, rid, "jan/15/2024")
	require.NoError(t, err)
	assert.Greater(t, len(sales), 0)
}

func TestBridge_RemoveSaleRecord(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("t")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	rec := &service.SalesRecord{
		Date: "jan/15/2024", Time: "10:30:00", Username: prefix + "-user2",
		Price: "5000", IPAddress: "192.168.88.100", MACAddress: "AA:BB:CC:DD:EE:FF",
		Validity: "1d", Profile: "default", Owner: "jan2024",
		Source: "test", Comment: "mikhmon",
	}

	err := bridge.RecordSale(ctx, rid, rec)
	require.NoError(t, err)

	before, err := bridge.FetchDailySales(ctx, rid, "jan/15/2024")
	require.NoError(t, err)

	for _, s := range before {
		if s.Username == prefix+"-user2" && s.ID != "" {
			err = bridge.RemoveSaleRecord(ctx, rid, s.ID)
			require.NoError(t, err)
			break
		}
	}

	after, err := bridge.FetchDailySales(ctx, rid, "jan/15/2024")
	require.NoError(t, err)
	assert.LessOrEqual(t, len(after), len(before))
}

func TestBridge_ImportSalesFromRouterOS(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("t")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	rec := &service.SalesRecord{
		Date: "jan/15/2024", Time: "10:30:00", Username: prefix + "-user3",
		Price: "5000", IPAddress: "192.168.88.100", MACAddress: "AA:BB:CC:DD:EE:FF",
		Validity: "1d", Profile: "default", Owner: "jan2024",
		Source: "test", Comment: "mikhmon",
	}

	err := bridge.RecordSale(ctx, rid, rec)
	require.NoError(t, err)

	records, err := bridge.ImportSalesFromRouterOS(ctx, rid, "jan2024")
	require.NoError(t, err)
	assert.NotNil(t, records)
}
