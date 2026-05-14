package stream

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/behavior"
	"github.com/quiqxiq/roskit/internal/roskit/core"
	"github.com/quiqxiq/roskit/internal/roskit/core/command"
	"github.com/quiqxiq/roskit/internal/roskit/core/parser"
	"github.com/quiqxiq/roskit/internal/roskit/execution"
)

// Defaults applied when WorkerConfig fields are zero.
const (
	DefaultQueueSize        = 500
	DefaultBackoffBase      = 5 * time.Second
	DefaultBackoffMax       = 30 * time.Second
	DefaultBackoffShiftCap  = 5
)

// WorkerConfig tunes the stream Worker. All fields are optional; zero values
// fall back to the Default* constants above so callers can override only what
// they need.
type WorkerConfig struct {
	QueueSize       int           // RouterOS reply channel buffer; default 500
	BackoffBase     time.Duration // initial reconnect delay; default 5s
	BackoffMax      time.Duration // reconnect delay cap; default 30s
	BackoffShiftCap int           // exponent cap for shift; default 5 (5s<<5 = 160s pre-cap)
}

func (c WorkerConfig) withDefaults() WorkerConfig {
	if c.QueueSize <= 0 {
		c.QueueSize = DefaultQueueSize
	}
	if c.BackoffBase <= 0 {
		c.BackoffBase = DefaultBackoffBase
	}
	if c.BackoffMax <= 0 {
		c.BackoffMax = DefaultBackoffMax
	}
	if c.BackoffShiftCap <= 0 {
		c.BackoffShiftCap = DefaultBackoffShiftCap
	}
	return c
}

type Worker struct {
	pool   *execution.Pool
	sink   behavior.StreamSink
	logger *slog.Logger
	cfg    WorkerConfig
}

// NewWorker constructs a Worker with default config.
func NewWorker(pool *execution.Pool, sink behavior.StreamSink, logger *slog.Logger) *Worker {
	return NewWorkerWithConfig(pool, sink, logger, WorkerConfig{})
}

// NewWorkerWithConfig constructs a Worker with explicit tuning. Pass a zero
// WorkerConfig{} to get the defaults; or override individual fields.
func NewWorkerWithConfig(pool *execution.Pool, sink behavior.StreamSink, logger *slog.Logger, cfg WorkerConfig) *Worker {
	if logger == nil {
		logger = slog.Default()
	}
	return &Worker{pool: pool, sink: sink, logger: logger, cfg: cfg.withDefaults()}
}

func (w *Worker) Start(ctx context.Context, routerID string, meta *command.CommandMeta) error {
	w.logger.Info("stream worker starting",
		"router_id", routerID,
		"measurement", meta.Measurement,
		"path", meta.Path,
	)

	for attempt := 0; ; attempt++ {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		state := w.pool.RouterState(routerID)
		if state == execution.ConnStateAuthFailed {
			w.logger.Info("stream worker stopping: router auth failed",
				"router_id", routerID,
				"measurement", meta.Measurement,
			)
			return nil
		}

		if err := w.acquireAndStream(ctx, routerID, meta); err != nil {
			if core.IsRouterOSPermanentError(err) {
				w.logger.Info("stream worker stopping: RouterOS feature unavailable",
					"router_id", routerID,
					"measurement", meta.Measurement,
					"error", err,
				)
				return nil
			}

			w.logger.Warn("stream session ended",
				"router_id", routerID,
				"measurement", meta.Measurement,
				"error", err,
				"attempt", attempt+1,
			)

			shift := attempt
			if shift > w.cfg.BackoffShiftCap {
				shift = w.cfg.BackoffShiftCap
			}
			delay := w.cfg.BackoffBase << uint(shift)
			if delay > w.cfg.BackoffMax {
				delay = w.cfg.BackoffMax
			}

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil
			}
		} else {
			return nil
		}
	}
}

func (w *Worker) acquireAndStream(ctx context.Context, routerID string, meta *command.CommandMeta) error {
	streamConn, err := w.pool.BorrowAsync(ctx, routerID)
	if err != nil {
		return fmt.Errorf("borrow stream: %w", err)
	}

	w.logger.Info("stream worker acquired persistent conn",
		"router_id", routerID,
		"measurement", meta.Measurement,
	)

	return w.runStream(ctx, routerID, meta, streamConn)
}

func (w *Worker) runStream(ctx context.Context, routerID string, meta *command.CommandMeta, streamConn *execution.PersistentConn) error {
	sentence := command.BuildStreamSentence(meta)
	reply, err := streamConn.ListenArgsQueueContext(ctx, sentence, w.cfg.QueueSize)
	if err != nil {
		return fmt.Errorf("listen %s: %w", meta.Measurement, err)
	}

	ch := reply.Chan()
	for {
		select {
		case <-ctx.Done():
			cancelCtx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
			reply.CancelContext(cancelCtx)
			cancelFn()
			return nil

		case sentence, ok := <-ch:
			if !ok {
				if err := reply.Err(); err != nil {
					return fmt.Errorf("stream %s closed: %w", meta.Measurement, err)
				}
				return fmt.Errorf("stream %s channel closed", meta.Measurement)
			}

			pairs := sentence.Map
			if len(pairs) == 0 {
				continue
			}

			result := parser.ParseStreamSentence(routerID, meta, pairs)
			if result == nil {
				continue
			}

			fields := result.CacheData
			if fields == nil {
				fields = pairs
			}

			event := behavior.StreamEvent{
				Meta:       meta,
				RouterID:   routerID,
				Fields:     fields,
				IsDead:     result.IsDead,
				ReceivedAt: time.Now(),
			}

			if err := w.sink.OnEvent(ctx, event); err != nil {
				w.logger.Warn("stream: process event error",
					"measurement", meta.Measurement,
					"error", err,
				)
			}
		}
	}
}

var _ behavior.StreamHandler = (*Worker)(nil)
