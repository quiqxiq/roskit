//go:build mikrotik

package execution

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutor_Run_IdentityPrint(t *testing.T) {
	skipWithoutMikroTik(t)
	pool, cleanup := newTestPool(t)
	defer cleanup()
	rid := routerID()
	exec := NewExecutor(pool)
	reply, err := exec.Run(context.Background(), rid, "/system/identity/print")
	require.NoError(t, err)
	assert.NotEmpty(t, reply.Re)
}

func TestExecutor_AddSetRemoveHotspotUser(t *testing.T) {
	skipWithoutMikroTik(t)
	pool, cleanup := newTestPool(t)
	defer cleanup()
	rid := routerID()
	exec := NewExecutor(pool)
	name := uniqueName("test-exec")
	reply, err := exec.Add(context.Background(), rid, "ip/hotspot/user", map[string]string{
		"name":     name,
		"password": "testpass123",
		"profile":  "default",
	})
	require.NoError(t, err)
	id := exec.ExtractID(reply)
	require.NotEmpty(t, id)
	defer exec.Remove(context.Background(), rid, "ip/hotspot/user", id)
	err = exec.Set(context.Background(), rid, "ip/hotspot/user", id, map[string]string{
		"comment": "integration test",
	})
	require.NoError(t, err)
	err = exec.Remove(context.Background(), rid, "ip/hotspot/user", id)
	require.NoError(t, err)
}

func TestExecutor_EnableDisable(t *testing.T) {
	skipWithoutMikroTik(t)
	pool, cleanup := newTestPool(t)
	defer cleanup()
	rid := routerID()
	exec := NewExecutor(pool)
	name := uniqueName("test-exec")
	reply, err := exec.Add(context.Background(), rid, "ip/hotspot/user", map[string]string{
		"name":     name,
		"password": "testpass123",
		"profile":  "default",
	})
	require.NoError(t, err)
	id := exec.ExtractID(reply)
	require.NotEmpty(t, id)
	defer exec.Remove(context.Background(), rid, "ip/hotspot/user", id)
	err = exec.Disable(context.Background(), rid, "ip/hotspot/user", id)
	require.NoError(t, err)
	err = exec.Enable(context.Background(), rid, "ip/hotspot/user", id)
	require.NoError(t, err)
	err = exec.Remove(context.Background(), rid, "ip/hotspot/user", id)
	require.NoError(t, err)
}

func TestExecutor_ExtractID(t *testing.T) {
	skipWithoutMikroTik(t)
	pool, cleanup := newTestPool(t)
	defer cleanup()
	rid := routerID()
	exec := NewExecutor(pool)
	name := uniqueName("test-exec")
	reply, err := exec.Add(context.Background(), rid, "ip/hotspot/user", map[string]string{
		"name":     name,
		"password": "testpass123",
		"profile":  "default",
	})
	require.NoError(t, err)
	id := exec.ExtractID(reply)
	defer exec.Remove(context.Background(), rid, "ip/hotspot/user", id)
	assert.NotEmpty(t, id)
}

func TestExecutor_Run_InvalidCommand(t *testing.T) {
	skipWithoutMikroTik(t)
	pool, cleanup := newTestPool(t)
	defer cleanup()
	rid := routerID()
	exec := NewExecutor(pool)
	_, err := exec.Run(context.Background(), rid, "/nonexistent/path")
	assert.Error(t, err)
}
