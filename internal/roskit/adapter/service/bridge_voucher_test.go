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

func TestBridge_GenerateAndCreateVouchers_VC(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("t")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	before, err := bridge.ListHotspotUsers(ctx, rid, "")
	require.NoError(t, err)

	params := service.VoucherParams{
		Qty:        3,
		UserType:   "vc",
		CharSet:    "mix",
		NameLength: 6,
		Prefix:     prefix,
		Profile:    "default",
		Comment:    prefix,
	}

	vouchers, err := bridge.GenerateAndCreateVouchers(ctx, rid, params)
	require.NoError(t, err)
	require.Len(t, vouchers, 3)

	for _, v := range vouchers {
		assert.Equal(t, v.Username, v.Password)
		assert.Contains(t, v.Username, prefix)
	}

	after, err := bridge.ListHotspotUsers(ctx, rid, "")
	require.NoError(t, err)
	assert.Equal(t, len(before)+3, len(after))
}

func TestBridge_GenerateAndCreateVouchers_UP(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("t")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	params := service.VoucherParams{
		Qty:        3,
		UserType:   "up",
		CharSet:    "mix",
		NameLength: 6,
		Prefix:     prefix,
		Profile:    "default",
		Comment:    prefix,
	}

	vouchers, err := bridge.GenerateAndCreateVouchers(ctx, rid, params)
	require.NoError(t, err)
	require.Len(t, vouchers, 3)

	for _, v := range vouchers {
		assert.NotEqual(t, v.Username, v.Password)
		assert.Contains(t, v.Username, prefix)
	}
}
