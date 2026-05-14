//go:build mikrotik

package execution

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConn_Connect_Success(t *testing.T) {
	skipWithoutMikroTik(t)
	cfg := loadRouterConfig().withDefaults()
	pc := newPersistentConn(cfg, RoleClient, slog.Default())
	err := pc.Connect(context.Background())
	require.NoError(t, err)
	assert.Equal(t, ConnStateConnected, pc.State())
	pc.Close()
}

func TestConn_Connect_WrongHost(t *testing.T) {
	skipWithoutMikroTik(t)
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
	skipWithoutMikroTik(t)
	cfg := loadRouterConfig().withDefaults()
	pc := newPersistentConn(cfg, RoleClient, slog.Default())
	err := pc.Connect(context.Background())
	require.NoError(t, err)
	defer pc.Close()
	reply, err := pc.RunContext(context.Background(), "/system/identity/print")
	require.NoError(t, err)
	assert.NotEmpty(t, reply.Re)
}

func TestConn_Close_State(t *testing.T) {
	skipWithoutMikroTik(t)
	cfg := loadRouterConfig().withDefaults()
	pc := newPersistentConn(cfg, RoleClient, slog.Default())
	err := pc.Connect(context.Background())
	require.NoError(t, err)
	pc.Close()
	assert.Equal(t, ConnStateDisconnected, pc.State())
}

func TestRouterConn_ConnectBoth(t *testing.T) {
	skipWithoutMikroTik(t)
	cfg := loadRouterConfig()
	rc := newRouterConn(cfg, slog.Default())
	err := rc.Connect(context.Background())
	require.NoError(t, err)
	defer rc.Close()
	assert.Equal(t, ConnStateConnected, rc.State())
	stream, err := rc.BorrowStream()
	require.NoError(t, err)
	assert.NotNil(t, stream)
}

func TestPersistentConn_ConnectAndRun(t *testing.T) {
	skipWithoutMikroTik(t)
	cfg := loadRouterConfig().withDefaults()
	pc := newPersistentConn(cfg, RoleClient, slog.Default())
	err := pc.Connect(context.Background())
	require.NoError(t, err)
	defer pc.Close()

	reply, err := pc.RunContext(context.Background(), "/system/identity/print")
	require.NoError(t, err)
	assert.NotEmpty(t, reply.Re)
}

func TestPersistentConn_ReconnectAfterDisconnect(t *testing.T) {
	skipWithoutMikroTik(t)
	cfg := loadRouterConfig().withDefaults()
	pc := newPersistentConn(cfg, RoleClient, slog.Default())
	err := pc.Connect(context.Background())
	require.NoError(t, err)
	defer pc.Close()

	assert.Equal(t, ConnStateConnected, pc.State())

	client := pc.Client()
	require.NotNil(t, client)
	client.Close()

	deadline := time.After(15 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for reconnect")
		case <-ticker.C:
			if pc.State() == ConnStateConnected {
				reply, err := pc.RunContext(context.Background(), "/system/identity/print")
				require.NoError(t, err)
				assert.NotEmpty(t, reply.Re)
				return
			}
		}
	}
}

func TestRouterConn_ConcurrentBorrowBorrowAsync(t *testing.T) {
	skipWithoutMikroTik(t)
	cfg := loadRouterConfig()
	pool := NewPool(slog.Default())
	pool.Register(cfg)
	ctx := context.Background()
	pool.Start(ctx)
	defer pool.Stop()

	waitCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	err := pool.WaitConnected(waitCtx, cfg.RouterID)
	require.NoError(t, err)

	rc, err := pool.Borrow(ctx, cfg.RouterID)
	require.NoError(t, err)
	reply, err := rc.RunContext(ctx, "/system/identity/print")
	require.NoError(t, err)
	assert.NotEmpty(t, reply.Re)

	stream, err := pool.BorrowAsync(ctx, cfg.RouterID)
	require.NoError(t, err)
	reply2, err := stream.RunContext(ctx, "/interface/print")
	require.NoError(t, err)
	assert.NotEmpty(t, reply2.Re)
}

func TestPool_RegisterUnregisterReconnect(t *testing.T) {
	skipWithoutMikroTik(t)
	cfg := loadRouterConfig()
	pool := NewPool(slog.Default())
	pool.Register(cfg)
	ctx := context.Background()
	pool.Start(ctx)
	defer pool.Stop()

	waitCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	err := pool.WaitConnected(waitCtx, cfg.RouterID)
	require.NoError(t, err)

	assert.Equal(t, ConnStateConnected, pool.Status()[cfg.RouterID])

	pool.Unregister(cfg.RouterID)
	_, err = pool.Borrow(ctx, cfg.RouterID)
	assert.Error(t, err)

	pool.Register(cfg)
	pool.LaunchOne(cfg.RouterID)
	waitCtx2, cancel2 := context.WithTimeout(ctx, 15*time.Second)
	defer cancel2()
	err = pool.WaitConnected(waitCtx2, cfg.RouterID)
	require.NoError(t, err)

	rc, err := pool.Borrow(ctx, cfg.RouterID)
	require.NoError(t, err)
	reply, err := rc.RunContext(ctx, "/system/identity/print")
	require.NoError(t, err)
	assert.NotEmpty(t, reply.Re)
}

func TestPool_MultipleConcurrentStreams(t *testing.T) {
	skipWithoutMikroTik(t)
	cfg := loadRouterConfig()
	pool := NewPool(slog.Default())
	pool.Register(cfg)
	ctx := context.Background()
	pool.Start(ctx)
	defer pool.Stop()

	waitCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	err := pool.WaitConnected(waitCtx, cfg.RouterID)
	require.NoError(t, err)

	pc, err := pool.BorrowAsync(ctx, cfg.RouterID)
	require.NoError(t, err)

	streamCtx, streamCancel := context.WithCancel(ctx)
	defer streamCancel()

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reply, err := pc.ListenArgsQueueContext(streamCtx, []string{"/interface/print"}, 10)
			if !assert.NoError(t, err) {
				return
			}
			ch := reply.Chan()
			select {
			case sentence, ok := <-ch:
				if !ok {
					assert.Fail(t, "stream channel closed before receiving data")
					return
				}
				assert.NotNil(t, sentence)
			case <-time.After(10 * time.Second):
				assert.Fail(t, "timed out waiting for stream data")
			}
		}()
	}

	rc, err := pool.Borrow(ctx, cfg.RouterID)
	require.NoError(t, err)
	reply, err := rc.RunContext(ctx, "/system/identity/print")
	require.NoError(t, err)
	assert.NotEmpty(t, reply.Re)

	wg.Wait()
}
