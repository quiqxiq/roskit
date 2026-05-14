//go:build mikrotik

package service_test

import (
	"context"
	"testing"

	"github.com/quiqxiq/roskit/internal/roskit/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBridge_ListHotspotUsers(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("t")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	users, err := bridge.ListHotspotUsers(ctx, rid, "")
	require.NoError(t, err)
	assert.NotNil(t, users)
}

func TestBridge_AddGetRemoveHotspotUser(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("t")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	name := testhelpers.UniqueName("t")
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

	err = bridge.RemoveHotspotUser(ctx, rid, id)
	require.NoError(t, err)

	_, err = bridge.GetHotspotUser(ctx, rid, name)
	assert.Error(t, err)
}

func TestBridge_SetHotspotUser(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("t")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	name := testhelpers.UniqueName("t")
	id, err := bridge.AddHotspotUser(ctx, rid, map[string]string{
		"name":     name,
		"password": "pass123",
		"profile":  "default",
	})
	require.NoError(t, err)

	err = bridge.SetHotspotUser(ctx, rid, id, map[string]string{
		"comment": "updated-comment",
	})
	require.NoError(t, err)

	user, err := bridge.GetHotspotUser(ctx, rid, id)
	require.NoError(t, err)
	assert.Equal(t, "updated-comment", user["comment"])

	_ = bridge.RemoveHotspotUser(ctx, rid, id)
}

func TestBridge_EnableDisableHotspotUser(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("t")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	name := testhelpers.UniqueName("t")
	id, err := bridge.AddHotspotUser(ctx, rid, map[string]string{
		"name":     name,
		"password": "pass123",
		"profile":  "default",
	})
	require.NoError(t, err)

	err = bridge.DisableHotspotUser(ctx, rid, id)
	require.NoError(t, err)

	user, err := bridge.GetHotspotUser(ctx, rid, id)
	require.NoError(t, err)
	assert.True(t, user["disabled"] == "true" || user["disabled"] == "yes")

	err = bridge.EnableHotspotUser(ctx, rid, id)
	require.NoError(t, err)

	user, err = bridge.GetHotspotUser(ctx, rid, id)
	require.NoError(t, err)
	assert.True(t, user["disabled"] == "false" || user["disabled"] == "no")

	_ = bridge.RemoveHotspotUser(ctx, rid, id)
}

func TestBridge_GetHotspotUserCount(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("t")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	for i := range 3 {
		name := testhelpers.UniqueName("t")
		_, err := bridge.AddHotspotUser(ctx, rid, map[string]string{
			"name":     name,
			"password": "pass",
			"profile":  "default",
		})
		require.NoError(t, err, "failed to add user %d", i)
	}

	count, err := bridge.GetHotspotUserCount(ctx, rid)
	require.NoError(t, err)
	assert.Greater(t, count, 0)
}

func TestBridge_HotspotProfileCRUD(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("t")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	profileName := testhelpers.UniqueName("t")
	id, err := bridge.AddHotspotProfile(ctx, rid, map[string]string{
		"name":         profileName,
		"rate-limit":   "1M/1M",
		"shared-users": "1",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	profile, err := bridge.GetHotspotProfile(ctx, rid, id)
	require.NoError(t, err)
	assert.Equal(t, profileName, profile["name"])

	err = bridge.SetHotspotProfile(ctx, rid, id, map[string]string{
		"rate-limit": "2M/2M",
	})
	require.NoError(t, err)

	profiles, err := bridge.ListHotspotProfiles(ctx, rid)
	require.NoError(t, err)
	assert.NotEmpty(t, profiles)

	err = bridge.RemoveHotspotProfile(ctx, rid, id)
	require.NoError(t, err)
}

func TestBridge_IPBindingCRUD(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("t")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	id, err := bridge.AddIPBinding(ctx, rid, map[string]string{
		"mac-address": "AA:BB:CC:DD:EE:01",
		"type":        "bypassed",
		"comment":     prefix,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	bindings, err := bridge.ListIPBindings(ctx, rid)
	require.NoError(t, err)
	assert.NotEmpty(t, bindings)

	err = bridge.RemoveIPBinding(ctx, rid, id)
	require.NoError(t, err)
}

func TestBridge_WalledGardenCRUD(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("t")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	id, err := bridge.AddWalledGarden(ctx, rid, map[string]string{
		"dst-host":   prefix + ".example.com",
		"action":     "allow",
		"comment":    prefix,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	wg, err := bridge.ListWalledGarden(ctx, rid)
	require.NoError(t, err)
	assert.NotEmpty(t, wg)

	err = bridge.RemoveWalledGarden(ctx, rid, id)
	require.NoError(t, err)
}

func TestBridge_GetSystemResource(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	res, err := bridge.GetSystemResource(ctx, rid)
	require.NoError(t, err)
	assert.Contains(t, res, "version")
	assert.Contains(t, res, "uptime")
}

func TestBridge_GetSystemIdentity(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	identity, err := bridge.GetSystemIdentity(ctx, rid)
	require.NoError(t, err)
	assert.Contains(t, identity, "name")
}

func TestBridge_SchedulerCRUD(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("t")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	schedName := testhelpers.UniqueName("t")
	id, err := bridge.AddScheduler(ctx, rid, map[string]string{
		"name":     schedName,
		"interval": "1h",
		"on-event": ":put test",
		"comment":  prefix,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	schedulers, err := bridge.ListSchedulers(ctx, rid)
	require.NoError(t, err)
	assert.NotEmpty(t, schedulers)

	err = bridge.SetScheduler(ctx, rid, id, map[string]string{
		"interval": "2h",
	})
	require.NoError(t, err)

	err = bridge.RemoveScheduler(ctx, rid, id)
	require.NoError(t, err)
}

func TestBridge_ScriptCRUD(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("t")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	scriptName := testhelpers.UniqueName("t")
	id, err := bridge.AddScript(ctx, rid, map[string]string{
		"name":    scriptName,
		"source":  ":put hello",
		"comment": prefix,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	scripts, err := bridge.ListScripts(ctx, rid)
	require.NoError(t, err)
	assert.NotEmpty(t, scripts)

	err = bridge.SetScript(ctx, rid, id, map[string]string{
		"source": ":put updated",
	})
	require.NoError(t, err)

	err = bridge.RemoveScript(ctx, rid, id)
	require.NoError(t, err)
}

func TestBridge_ListInterfaces(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	ifaces, err := bridge.ListInterfaces(ctx, rid)
	require.NoError(t, err)
	assert.NotEmpty(t, ifaces)
}

func TestBridge_PPPSecretCRUD(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("t")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	secretName := testhelpers.UniqueName("t")
	id, err := bridge.AddPPPSecret(ctx, rid, map[string]string{
		"name":     secretName,
		"password": "secret123",
		"profile":  "default",
		"comment":  prefix,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	secret, err := bridge.GetPPPSecret(ctx, rid, id)
	require.NoError(t, err)
	assert.Equal(t, secretName, secret["name"])

	err = bridge.SetPPPSecret(ctx, rid, id, map[string]string{
		"comment": "updated",
	})
	require.NoError(t, err)

	err = bridge.RemovePPPSecret(ctx, rid, id)
	require.NoError(t, err)
}

func TestBridge_ListHotspotActive(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	sessions, err := bridge.ListHotspotActive(ctx, rid)
	require.NoError(t, err)
	assert.NotNil(t, sessions)
}

func TestBridge_GetActiveSessionCount(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()

	count, err := bridge.GetActiveSessionCount(ctx, rid)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, 0)
}

func TestBridge_ListInactiveHotspotUsers(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("t")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	name := testhelpers.UniqueName("t")
	_, err := bridge.AddHotspotUser(ctx, rid, map[string]string{
		"name":     name,
		"password": "pass123",
		"profile":  "default",
	})
	require.NoError(t, err)

	allUsers, err := bridge.ListHotspotUsers(ctx, rid, "")
	require.NoError(t, err)

	activeSessions, err := bridge.ListHotspotActive(ctx, rid)
	require.NoError(t, err)

	activeMap := make(map[string]struct{})
	for _, s := range activeSessions {
		if u := s["user"]; u != "" {
			activeMap[u] = struct{}{}
		}
	}

	inactiveCount := 0
	for _, u := range allUsers {
		if _, active := activeMap[u["name"]]; !active {
			inactiveCount++
		}
	}
	assert.Greater(t, inactiveCount, 0, "should have at least 1 inactive user")
}

func TestBridge_GetInactiveHotspotUserCount(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	rid := testhelpers.RouterID()
	prefix := testhelpers.UniqueName("t")
	t.Cleanup(func() { testhelpers.CleanupAll(ctx, bridge, rid, prefix) })

	_, err := bridge.AddHotspotUser(ctx, rid, map[string]string{
		"name":     testhelpers.UniqueName("t"),
		"password": "pass123",
		"profile":  "default",
	})
	require.NoError(t, err)

	count, err := bridge.GetInactiveHotspotUserCount(ctx, rid)
	require.NoError(t, err)
	assert.Greater(t, count, 0)
}

