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

func TestBridge_SaveGetQuickPrintPackage(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("t")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	pkg := &service.QuickPrintPackage{
		Name: prefix, Server: "all", UserMode: "vc", UserLength: "6",
		Prefix: "vc", CharMode: "mix", Profile: "default",
		TimeLimit: "", DataLimit: "", Comment: prefix,
		Validity: "1d", Price: "5000", SellingPrice: "6000", LockUser: "Disable",
	}

	err := bridge.SaveQuickPrintPackage(ctx, rid, pkg)
	require.NoError(t, err)

	got, err := bridge.GetQuickPrintPackage(ctx, rid, prefix)
	require.NoError(t, err)
	assert.Equal(t, pkg.Name, got.Name)
	assert.Equal(t, pkg.Server, got.Server)
	assert.Equal(t, pkg.UserMode, got.UserMode)
	assert.Equal(t, pkg.UserLength, got.UserLength)
	assert.Equal(t, pkg.Profile, got.Profile)
	assert.Equal(t, pkg.Validity, got.Validity)
	assert.Equal(t, pkg.Price, got.Price)
	assert.Equal(t, pkg.SellingPrice, got.SellingPrice)
	assert.Equal(t, pkg.LockUser, got.LockUser)
}

func TestBridge_ListQuickPrintPackages(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("t")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	pkg1 := &service.QuickPrintPackage{
		Name: prefix + "-a", Server: "all", UserMode: "vc", UserLength: "6",
		Prefix: "vc", CharMode: "mix", Profile: "default",
		TimeLimit: "", DataLimit: "", Comment: service.QuickPrintComment,
		Validity: "1d", Price: "5000", SellingPrice: "6000", LockUser: "Disable",
	}
	pkg2 := &service.QuickPrintPackage{
		Name: prefix + "-b", Server: "all", UserMode: "up", UserLength: "8",
		Prefix: "up", CharMode: "num", Profile: "default",
		TimeLimit: "", DataLimit: "", Comment: service.QuickPrintComment,
		Validity: "3d", Price: "10000", SellingPrice: "12000", LockUser: "Enable",
	}

	err := bridge.SaveQuickPrintPackage(ctx, rid, pkg1)
	require.NoError(t, err)
	err = bridge.SaveQuickPrintPackage(ctx, rid, pkg2)
	require.NoError(t, err)

	pkgs, err := bridge.ListQuickPrintPackages(ctx, rid)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(pkgs), 2)
}

func TestBridge_RemoveQuickPrintPackage(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("t")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	pkg := &service.QuickPrintPackage{
		Name: prefix, Server: "all", UserMode: "vc", UserLength: "6",
		Prefix: "vc", CharMode: "mix", Profile: "default",
		TimeLimit: "", DataLimit: "", Comment: prefix,
		Validity: "1d", Price: "5000", SellingPrice: "6000", LockUser: "Disable",
	}

	err := bridge.SaveQuickPrintPackage(ctx, rid, pkg)
	require.NoError(t, err)

	got, err := bridge.GetQuickPrintPackage(ctx, rid, prefix)
	require.NoError(t, err)
	require.NotEmpty(t, got.ID)

	err = bridge.RemoveQuickPrintPackage(ctx, rid, got.ID)
	require.NoError(t, err)

	_, err = bridge.GetQuickPrintPackage(ctx, rid, prefix)
	assert.Error(t, err)
}
