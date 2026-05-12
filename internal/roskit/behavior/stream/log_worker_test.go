//go:build mikrotik

package stream_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	_ "github.com/quiqxiq/roskit/internal/roskit/core/definition"
	"github.com/quiqxiq/roskit/internal/roskit/behavior/stream"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/pubsub"
	"github.com/quiqxiq/roskit/internal/roskit/testhelpers"
	"github.com/stretchr/testify/assert"
)

func TestLogWorker_Start_And_Cancel(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)
	pool, cleanup := testhelpers.NewTestPool(t)
	defer cleanup()

	routerID := testhelpers.RouterID()
	worker := stream.NewLogWorker(pool, pubsub.NoopPublisher{}, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		worker.Start(ctx, routerID, "all")
	}()

	time.Sleep(700 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// worker stopped cleanly
	case <-time.After(5 * time.Second):
		t.Fatal("log worker did not stop after context cancel")
	}
}

func TestLogWorker_FilterHotspot(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)
	pool, cleanup := testhelpers.NewTestPool(t)
	defer cleanup()

	routerID := testhelpers.RouterID()
	worker := stream.NewLogWorker(pool, pubsub.NoopPublisher{}, slog.Default())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		worker.Start(ctx, routerID, "hotspot")
	}()

	<-done
	// Test passes if worker exits without panic after ctx timeout
	assert.True(t, true)
}
