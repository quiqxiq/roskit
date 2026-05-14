//go:build mikrotik

package stream_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/adapter/service"
	"github.com/quiqxiq/roskit/internal/roskit/behavior/stream"
	"github.com/quiqxiq/roskit/internal/roskit/core/command"
	_ "github.com/quiqxiq/roskit/internal/roskit/core/definition"
	"github.com/quiqxiq/roskit/internal/roskit/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamWorker_SystemResource_ReceivesData(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	pool, poolCleanup := testhelpers.NewTestPool(t)
	t.Cleanup(poolCleanup)

	sink := testhelpers.NewMemStreamSink()
	worker := stream.NewWorker(pool, sink, slog.Default())

	meta := command.Lookup("system/resource/print")
	require.NotNil(t, meta, "system/resource/print must be registered")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	rid := testhelpers.RouterID()
	go worker.Start(ctx, rid, meta)

	select {
	case event := <-sink.Ch:
		assert.Equal(t, "system_resource", event.Meta.Measurement)
		assert.Contains(t, event.Fields, "version")
		assert.Contains(t, event.Fields, "uptime")
	case <-ctx.Done():
		t.Fatal("timeout: no stream event received for system/resource")
	}
}

func TestStreamWorker_HotspotUser_ReceivesUpdate(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	pool, poolCleanup := testhelpers.NewTestPool(t)
	t.Cleanup(poolCleanup)

	sink := testhelpers.NewMemStreamSink()
	worker := stream.NewWorker(pool, sink, slog.Default())

	meta := command.Lookup("ip/hotspot/user/print")
	require.NotNil(t, meta, "ip/hotspot/user/print must be registered")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	rid := testhelpers.RouterID()
	go worker.Start(ctx, rid, meta)

	time.Sleep(2 * time.Second)

	bridge, bCleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(bCleanup)
	prefix := testhelpers.UniqueName("stream")
	t.Cleanup(func() { testhelpers.CleanupAll(context.Background(), bridge, rid, prefix) })

	name := testhelpers.UniqueName("stream")
	_, err := bridge.AddHotspotUser(ctx, rid, map[string]string{
		"name":     name,
		"password": "pass123",
		"profile":  "default",
	})
	require.NoError(t, err)

	timeout := time.After(10 * time.Second)
	for {
		select {
		case event := <-sink.Ch:
			if event.Meta.Measurement == "hotspot_user" {
				return
			}
		case <-timeout:
			t.Fatal("timeout: no hotspot_user stream event received after mutation")
		}
	}
}

func TestStreamWorker_PPPSecret_ReceivesUpdate(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	pool, poolCleanup := testhelpers.NewTestPool(t)
	t.Cleanup(poolCleanup)

	sink := testhelpers.NewMemStreamSink()
	worker := stream.NewWorker(pool, sink, slog.Default())

	meta := command.Lookup("ppp/secret/print")
	require.NotNil(t, meta, "ppp/secret/print must be registered")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	rid := testhelpers.RouterID()
	go worker.Start(ctx, rid, meta)

	time.Sleep(2 * time.Second)

	bridge, bCleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(bCleanup)
	prefix := testhelpers.UniqueName("stream")
	t.Cleanup(func() { testhelpers.CleanupAll(context.Background(), bridge, rid, prefix) })

	name := testhelpers.UniqueName("stream")
	_, err := bridge.AddPPPSecret(ctx, rid, map[string]string{
		"name":     name,
		"password": "secret123",
		"profile":  "default",
		"comment":  prefix,
	})
	require.NoError(t, err)

	timeout := time.After(10 * time.Second)
	for {
		select {
		case event := <-sink.Ch:
			if event.Meta.Measurement == "ppp_secret" {
				return
			}
		case <-timeout:
			t.Fatal("timeout: no ppp_secret stream event received after mutation")
		}
	}
}

func TestStreamWorker_Interface_ReceivesData(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	pool, poolCleanup := testhelpers.NewTestPool(t)
	t.Cleanup(poolCleanup)

	sink := testhelpers.NewMemStreamSink()
	worker := stream.NewWorker(pool, sink, slog.Default())

	meta := command.Lookup("interface/print")
	require.NotNil(t, meta, "interface/print must be registered")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	rid := testhelpers.RouterID()
	go worker.Start(ctx, rid, meta)

	select {
	case event := <-sink.Ch:
		assert.Equal(t, "interface", event.Meta.Measurement)
		assert.Contains(t, event.Fields, "name")
	case <-ctx.Done():
		t.Fatal("timeout: no interface stream event received")
	}
}

var _ = &service.Bridge{}
