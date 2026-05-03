//go:build mikrotik

package execution

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConn_Connect_Success(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)
	cfg := testhelpers.LoadRouterConfig().withDefaults()
	pc := newPersistentConn(cfg, RoleClient, slog.Default())
	err := pc.Connect(context.Background())
	require.NoError(t, err)
	assert.Equal(t, ConnStateConnected, pc.State())
	pc.Close()
}

func TestConn_Connect_WrongHost(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)
	cfg := ConnConfig{
		RouterID:    "bad",
		Address:     "192.0.2.1:8728",
		Username:    "admin",
		Password:    "",
		DialTimeout: 2 * time.Second,
	}
	pc := newPersistentConn(cfg, RoleClient, slog.Default())
	err := pc.Connect(context.Background())
	assert.Error(t, err)
	assert.Equal(t, ConnStateDisconnected, pc.State())
}

func TestConn_RunContext_IdentityPrint(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)
	cfg := testhelpers.LoadRouterConfig().withDefaults()
	pc := newPersistentConn(cfg, RoleClient, slog.Default())
	err := pc.Connect(context.Background())
	require.NoError(t, err)
	defer pc.Close()
	reply, err := pc.RunContext(context.Background(), "/system/identity/print")
	require.NoError(t, err)
	assert.NotEmpty(t, reply.Re)
}

func TestConn_Close_State(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)
	cfg := testhelpers.LoadRouterConfig().withDefaults()
	pc := newPersistentConn(cfg, RoleClient, slog.Default())
	err := pc.Connect(context.Background())
	require.NoError(t, err)
	pc.Close()
	assert.Equal(t, ConnStateDisconnected, pc.State())
}

func TestConn_IsAlive_Connected(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)
	cfg := testhelpers.LoadRouterConfig().withDefaults()
	pc := newPersistentConn(cfg, RoleClient, slog.Default())
	err := pc.Connect(context.Background())
	require.NoError(t, err)
	defer pc.Close()
	assert.True(t, pc.IsAlive(context.Background()))
}

func TestRouterConn_ConnectBoth(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)
	cfg := testhelpers.LoadRouterConfig()
	rc := newRouterConn(cfg, slog.Default())
	err := rc.Connect(context.Background())
	require.NoError(t, err)
	defer rc.Close()
	assert.Equal(t, ConnStateConnected, rc.State())
	stream, err := rc.BorrowStream()
	require.NoError(t, err)
	assert.NotNil(t, stream)
}
