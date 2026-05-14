//go:build mikrotik

package service_test

import (
	"context"
	"testing"

	"github.com/quiqxiq/roskit/internal/roskit/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBridge_ListPPPSecrets(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	secrets, err := bridge.ListPPPSecrets(ctx, rid)
	require.NoError(t, err)
	assert.NotNil(t, secrets)
}

func TestBridge_AddGetRemovePPPSecret(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("ppp")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	name := testhelpers.UniqueName("ppp")
	id, err := bridge.AddPPPSecret(ctx, rid, map[string]string{
		"name":     name,
		"password": "secret123",
		"profile":  "default",
		"comment":  prefix,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	secret, err := bridge.GetPPPSecret(ctx, rid, id)
	require.NoError(t, err)
	assert.Equal(t, name, secret["name"])

	err = bridge.RemovePPPSecret(ctx, rid, id)
	require.NoError(t, err)
}

func TestBridge_SetPPPSecret(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("ppp")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	name := testhelpers.UniqueName("ppp")
	id, err := bridge.AddPPPSecret(ctx, rid, map[string]string{
		"name":     name,
		"password": "secret123",
		"profile":  "default",
		"comment":  prefix,
	})
	require.NoError(t, err)

	err = bridge.SetPPPSecret(ctx, rid, id, map[string]string{
		"comment": "updated-ppp",
	})
	require.NoError(t, err)

	secret, err := bridge.GetPPPSecret(ctx, rid, id)
	require.NoError(t, err)
	assert.Equal(t, "updated-ppp", secret["comment"])
}

func TestBridge_EnableDisablePPPSecret(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("ppp")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	name := testhelpers.UniqueName("ppp")
	id, err := bridge.AddPPPSecret(ctx, rid, map[string]string{
		"name":     name,
		"password": "secret123",
		"profile":  "default",
		"comment":  prefix,
	})
	require.NoError(t, err)

	err = bridge.DisablePPPSecret(ctx, rid, id)
	require.NoError(t, err)

	secret, err := bridge.GetPPPSecret(ctx, rid, id)
	require.NoError(t, err)
	assert.True(t, secret["disabled"] == "true" || secret["disabled"] == "yes")

	err = bridge.EnablePPPSecret(ctx, rid, id)
	require.NoError(t, err)

	secret, err = bridge.GetPPPSecret(ctx, rid, id)
	require.NoError(t, err)
	assert.True(t, secret["disabled"] == "false" || secret["disabled"] == "no")
}

func TestBridge_ListPPPActive(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	sessions, err := bridge.ListPPPActive(ctx, rid)
	require.NoError(t, err)
	assert.NotNil(t, sessions)
}

func TestBridge_ListPPPProfiles(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	profiles, err := bridge.ListPPPProfiles(ctx, rid)
	require.NoError(t, err)
	assert.NotEmpty(t, profiles)
}

func TestBridge_AddSetRemovePPPProfile(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("ppp")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	name := testhelpers.UniqueName("ppp")
	id, err := bridge.AddPPPProfile(ctx, rid, map[string]string{
		"name":       name,
		"rate-limit": "1M/1M",
		"comment":    prefix,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	profile, err := bridge.GetPPPProfile(ctx, rid, id)
	require.NoError(t, err)
	assert.Equal(t, name, profile["name"])

	err = bridge.SetPPPProfile(ctx, rid, id, map[string]string{
		"rate-limit": "2M/2M",
	})
	require.NoError(t, err)

	err = bridge.RemovePPPProfile(ctx, rid, id)
	require.NoError(t, err)
}

func TestBridge_ListInactivePPPSecrets(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("ppp")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	name := testhelpers.UniqueName("ppp")
	_, err := bridge.AddPPPSecret(ctx, rid, map[string]string{
		"name":     name,
		"password": "secret123",
		"profile":  "default",
		"comment":  prefix,
	})
	require.NoError(t, err)

	inactive, err := bridge.ListInactivePPPSecrets(ctx, rid)
	require.NoError(t, err)

	found := false
	for _, s := range inactive {
		if s["name"] == name {
			found = true
			break
		}
	}
	assert.True(t, found, "newly created PPP secret should be in inactive list")
}

func TestBridge_GetInactivePPPSecretCount(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("ppp")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	_, err := bridge.AddPPPSecret(ctx, rid, map[string]string{
		"name":     testhelpers.UniqueName("ppp"),
		"password": "secret123",
		"profile":  "default",
		"comment":  prefix,
	})
	require.NoError(t, err)

	count, err := bridge.GetInactivePPPSecretCount(ctx, rid)
	require.NoError(t, err)
	assert.Greater(t, count, 0)
}
