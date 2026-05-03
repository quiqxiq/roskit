//go:build mikrotik

package stream_test

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/behavior"
	"github.com/quiqxiq/roskit/internal/roskit/behavior/stream"
	"github.com/quiqxiq/roskit/internal/roskit/core/command"
	_ "github.com/quiqxiq/roskit/internal/roskit/core/definition"
	"github.com/quiqxiq/roskit/internal/roskit/execution"
	"github.com/quiqxiq/roskit/internal/roskit/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type capturingStreamSink struct {
	mu     sync.Mutex
	events []behavior.StreamEvent
}

func (s *capturingStreamSink) OnEvent(_ context.Context, event behavior.StreamEvent) error {
	s.mu.Lock()
	s.events = append(s.events, event)
	s.mu.Unlock()
	return nil
}

func (s *capturingStreamSink) Events() []behavior.StreamEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.events
}

func TestStreamWorker_ReceiveEvents(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)
	pool, cleanup := testhelpers.NewTestPool(t)
	defer cleanup()

	routerID := testhelpers.RouterID()
	sink := &capturingStreamSink{}
	worker := stream.NewWorker(pool, sink, slog.Default())

	meta := command.Lookup("ip/hotspot/user/print")
	require.NotNil(t, meta)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- worker.Start(ctx, routerID, meta)
	}()

	time.Sleep(2 * time.Second)

	userName := testhelpers.UniqueName("stream-test")
	exec := execution.NewExecutor(pool)
	_, err := exec.Run(context.Background(), routerID,
		"/ip/hotspot/user/add",
		"=name="+userName,
		"=password=pass",
		"=profile=default",
	)
	require.NoError(t, err)

	defer func() {
		exec.Run(context.Background(), routerID,
			"/ip/hotspot/user/remove",
			"=.id="+userName,
		)
	}()

	time.Sleep(3 * time.Second)

	events := sink.Events()
	require.NotEmpty(t, events, "expected at least one stream event")

	found := false
	for _, ev := range events {
		assert.Equal(t, routerID, ev.RouterID)
		assert.Equal(t, meta.Measurement, ev.Meta.Measurement)
		if ev.Fields["name"] == userName {
			found = true
		}
	}
	assert.True(t, found, "expected event for user %s", userName)

	cancel()
}

func TestStreamWorker_Cancel(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)
	pool, cleanup := testhelpers.NewTestPool(t)
	defer cleanup()

	routerID := testhelpers.RouterID()
	sink := &capturingStreamSink{}
	worker := stream.NewWorker(pool, sink, slog.Default())

	meta := command.Lookup("ip/hotspot/user/print")
	require.NotNil(t, meta)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- worker.Start(ctx, routerID, meta)
	}()

	time.Sleep(1 * time.Second)
	cancel()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("worker did not stop after cancel")
	}
}

func TestStreamManager_Multiple(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)
	pool, cleanup := testhelpers.NewTestPool(t)
	defer cleanup()

	routerID := testhelpers.RouterID()
	sink := &capturingStreamSink{}
	mgr := stream.NewManager(pool, sink, slog.Default())

	paths := []string{
		"ip/hotspot/user/print",
		"ip/hotspot/active/print",
		"ip/hotspot/print",
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, p := range paths {
		meta := command.Lookup(p)
		require.NotNil(t, meta, "command %q not found", p)
		mgr.Start(ctx, routerID, meta)
	}

	time.Sleep(2 * time.Second)
	assert.Greater(t, mgr.ActiveCount(), 0)

	mgr.StopAll(routerID)
	time.Sleep(2 * time.Second)
	assert.Equal(t, 0, mgr.ActiveCount())
}
