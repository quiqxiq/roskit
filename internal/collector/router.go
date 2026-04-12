// Package collector contains the core engine that manages the lifecycle of
// connections to MikroTik routers and orchestrates concurrent telemetry streaming.
package collector

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/go-routeros/routeros/v3"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/internal/spec"
	"github.com/quiqxiq/roskit/internal/usecase"
)

const (
	// defaultQueueSize is the buffer size for ListenArgsQueue channels.
	// A larger buffer prevents data loss during processing spikes.
	defaultQueueSize = 500
)

// RouterCollector manages the streaming lifecycle for a single MikroTik router.
// It borrows a persistent connection from the ConnPool, starts all configured
// StreamSpecs as concurrent goroutines, and releases the connection back to
// the pool when the session ends.
//
// The collector does NOT own the connection — the pool does.
// This separation ensures connections persist across streaming session restarts.
type RouterCollector struct {
	config  domain.RouterConfig
	specs   []spec.StreamSpec
	usecase *usecase.TelemetryUseCase
	pool    *ConnPool
	logger  *slog.Logger

	mu      sync.Mutex
	cancel  context.CancelFunc
	running bool
}

// NewRouterCollector creates a new collector for a single router.
// The collector uses the provided ConnPool to obtain persistent connections
// instead of dialing directly.
func NewRouterCollector(
	config domain.RouterConfig,
	specs []spec.StreamSpec,
	uc *usecase.TelemetryUseCase,
	pool *ConnPool,
	logger *slog.Logger,
) *RouterCollector {
	config.Defaults()

	return &RouterCollector{
		config:  config,
		specs:   specs,
		usecase: uc,
		pool:    pool,
		logger:  logger.With("router", config.ID, "addr", config.Address),
	}
}

// Start begins the streaming loop for this router.
// It acquires a connection from the pool, runs all specs, and on failure,
// releases the connection and retries with exponential backoff.
// The pool handles the actual reconnection — the collector just re-acquires.
// Backoff uses per-router ReconnectInterval and MaxReconnectInterval.
func (rc *RouterCollector) Start(ctx context.Context) {
	rc.mu.Lock()
	if rc.running {
		rc.mu.Unlock()
		return
	}
	rc.running = true
	var childCtx context.Context
	childCtx, rc.cancel = context.WithCancel(ctx)
	rc.mu.Unlock()

	rc.logger.Info("starting router collector",
		"reconnect_interval", rc.config.ReconnectInterval,
		"max_reconnect_interval", rc.config.MaxReconnectInterval,
	)

	delay := rc.config.ReconnectInterval
	for {
		select {
		case <-childCtx.Done():
			rc.logger.Info("router collector stopped")
			return
		default:
		}

		if err := rc.acquireAndStream(childCtx); err != nil {
			rc.logger.Error("streaming session ended",
				"error", err,
				"reconnect_in", delay,
			)

			// Wait before retrying with exponential backoff.
			select {
			case <-time.After(delay):
				delay = min(delay*2, rc.config.MaxReconnectInterval)
			case <-childCtx.Done():
				return
			}
		} else {
			// Clean exit (context cancelled) — reset delay.
			delay = rc.config.ReconnectInterval
			return
		}
	}
}

// Stop gracefully shuts down the router collector.
func (rc *RouterCollector) Stop() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.cancel != nil {
		rc.cancel()
	}
	rc.running = false
}

// acquireAndStream borrows a connection from the pool, enables async mode,
// runs all specs, then releases the connection when done.
func (rc *RouterCollector) acquireAndStream(ctx context.Context) error {
	// 1. Get the managed connection from the pool.
	conn := rc.pool.Get(rc.config.ID)
	if conn == nil {
		return fmt.Errorf("router %s not found in connection pool", rc.config.ID)
	}

	// 2. Ensure the connection is established.
	if conn.State() != ConnStateConnected {
		rc.logger.Info("waiting for pool to establish connection")
		if err := conn.Connect(ctx); err != nil {
			return fmt.Errorf("connect failed: %w", err)
		}
	}

	// 3. Acquire the client in async mode for streaming.
	client, err := conn.AcquireAsync()
	if err != nil {
		return fmt.Errorf("acquire async failed: %w", err)
	}

	rc.logger.Info("connection acquired from pool, async mode enabled")

	// 4. Run all specs concurrently on this connection.
	streamErr := rc.runAllSpecs(ctx, client)

	// 5. Release the connection back to the pool.
	// This closes the async client so the pool can create a fresh one.
	conn.Release()

	// 6. Reset backoff on successful reconnect.
	return streamErr
}

// runAllSpecs launches all specs as concurrent goroutines and waits for completion.
func (rc *RouterCollector) runAllSpecs(ctx context.Context, client *routeros.Client) error {
	var wg sync.WaitGroup
	var errCount int64
	var errMu sync.Mutex
	var lastErr error

	for _, s := range rc.specs {
		wg.Add(1)
		go func(s spec.StreamSpec) {
			defer wg.Done()
			if err := rc.runSpec(ctx, client, s); err != nil {
				rc.logger.Error("spec stream failed",
					"spec", s.Tag(),
					"error", err,
				)
				errMu.Lock()
				errCount++
				lastErr = err
				errMu.Unlock()
			}
		}(s)
	}

	// Wait for ALL spec goroutines to finish.
	wg.Wait()

	// If context was cancelled, this is a clean shutdown.
	if ctx.Err() != nil {
		rc.logger.Info("context cancelled, streams stopped")
		return nil
	}

	// If specs errored, report for reconnection.
	if lastErr != nil {
		return fmt.Errorf("%d/%d specs failed, last error: %w", errCount, len(rc.specs), lastErr)
	}

	return nil
}

// runSpec starts a single streaming spec and processes responses.
// It runs until the context is cancelled or an error occurs.
func (rc *RouterCollector) runSpec(ctx context.Context, client *routeros.Client, s spec.StreamSpec) error {
	rc.logger.Info("starting stream", "spec", s.Tag(), "command", s.Command())

	// Start the listen command with a buffered queue.
	reply, err := client.ListenArgsQueueContext(ctx, s.Command(), defaultQueueSize)
	if err != nil {
		return fmt.Errorf("ListenArgsQueueContext(%s): %w", s.Tag(), err)
	}

	// Process incoming sentences.
	ch := reply.Chan()
	for {
		select {
		case <-ctx.Done():
			// Graceful shutdown: cancel the listen command on the router.
			cancelCtx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
			reply.CancelContext(cancelCtx)
			cancelFn()
			return nil

		case sentence, ok := <-ch:
			if !ok {
				// Channel closed — connection lost.
				if err := reply.Err(); err != nil {
					return fmt.Errorf("stream %s closed with error: %w", s.Tag(), err)
				}
				return fmt.Errorf("stream %s channel closed", s.Tag())
			}

			// Parse the sentence into a telemetry event.
			event, err := s.Parse(rc.config.ID, sentence)
			if err != nil {
				rc.logger.Warn("parse error",
					"spec", s.Tag(),
					"error", err,
				)
				continue
			}

			if event == nil {
				continue
			}

			// Process the event through the usecase layer.
			if err := rc.usecase.ProcessEvent(ctx, event); err != nil {
				rc.logger.Warn("process event error",
					"spec", s.Tag(),
					"error", err,
				)
			}
		}
	}
}
