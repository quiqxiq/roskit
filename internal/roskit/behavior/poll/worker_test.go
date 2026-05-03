//go:build mikrotik

package poll_test

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/behavior"
	"github.com/quiqxiq/roskit/internal/roskit/behavior/poll"
	"github.com/quiqxiq/roskit/internal/roskit/core/command"
	_ "github.com/quiqxiq/roskit/internal/roskit/core/definition"
	"github.com/quiqxiq/roskit/internal/roskit/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type capturingPollSink struct {
	mu     sync.Mutex
	events []behavior.PollEvent
}

func (s *capturingPollSink) OnPoll(_ context.Context, event behavior.PollEvent) error {
	s.mu.Lock()
	s.events = append(s.events, event)
	s.mu.Unlock()
	return nil
}

func (s *capturingPollSink) Events() []behavior.PollEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.events
}

func TestPollWorker_SystemResource(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)
	pool, cleanup := testhelpers.NewTestPool(t)
	defer cleanup()

	routerID := testhelpers.RouterID()
	sink := &capturingPollSink{}
	worker := poll.NewWorker(pool, sink, slog.Default())

	meta := &command.CommandMeta{
		Path:            "system/resource/print",
		Type:            command.CommandTypePoll,
		PollInterval:    1 * time.Second,
		Measurement:     "system_resource",
		Category:        "system",
		SupportsInterval: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- worker.Start(ctx, routerID, meta)
	}()

	time.Sleep(3 * time.Second)

	events := sink.Events()
	require.NotEmpty(t, events, "expected at least one poll event")

	var successEvents int
	for _, ev := range events {
		assert.Equal(t, routerID, ev.RouterID)
		assert.Equal(t, "system_resource", ev.Meta.Measurement)
		assert.NoError(t, ev.Err)
		if len(ev.Rows) > 0 {
			successEvents++
		}
	}
	assert.Greater(t, successEvents, 0, "expected at least one successful poll with rows")

	cancel()
}

func TestPollWorker_Cancel(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)
	pool, cleanup := testhelpers.NewTestPool(t)
	defer cleanup()

	routerID := testhelpers.RouterID()
	sink := &capturingPollSink{}
	worker := poll.NewWorker(pool, sink, slog.Default())

	meta := &command.CommandMeta{
		Path:         "system/resource/print",
		Type:         command.CommandTypePoll,
		PollInterval: 10 * time.Second,
		Measurement:  "system_resource",
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- worker.Start(ctx, routerID, meta)
	}()

	time.Sleep(500 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("worker did not stop after cancel")
	}
}

func TestConcurrentRunner_RunAll(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)
	pool, cleanup := testhelpers.NewTestPool(t)
	defer cleanup()

	routerID := testhelpers.RouterID()
	sink := &capturingPollSink{}
	runner := poll.NewConcurrentRunner(pool, sink, slog.Default())

	metas := []*command.CommandMeta{
		{Path: "system/resource/print", Type: command.CommandTypePoll, Measurement: "system_resource", Category: "system"},
		{Path: "system/identity/print", Type: command.CommandTypePoll, Measurement: "system_identity", Category: "system"},
		{Path: "system/routerboard/print", Type: command.CommandTypePoll, Measurement: "system_routerboard", Category: "system"},
	}

	ctx := context.Background()
	runner.RunAll(ctx, routerID, metas)

	events := sink.Events()
	require.Len(t, events, 3, "expected one event per meta")

	measurements := map[string]bool{}
	for _, ev := range events {
		assert.NoError(t, ev.Err, "poll event for %s had error", ev.Meta.Measurement)
		assert.Equal(t, routerID, ev.RouterID)
		measurements[ev.Meta.Measurement] = true
	}
	assert.True(t, measurements["system_resource"])
	assert.True(t, measurements["system_identity"])
	assert.True(t, measurements["system_routerboard"])
}
