package stream

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-routeros/routeros/v3"
	"github.com/quiqxiq/roskit/internal/roskit/behavior"
	"github.com/quiqxiq/roskit/internal/roskit/core/command"
	"github.com/quiqxiq/roskit/internal/roskit/core/parser"
	"github.com/quiqxiq/roskit/internal/roskit/execution"
)

const defaultQueueSize = 500

type Worker struct {
	pool   *execution.Pool
	sink   behavior.StreamSink
	logger *slog.Logger
}

func NewWorker(pool *execution.Pool, sink behavior.StreamSink, logger *slog.Logger) *Worker {
	if logger == nil {
		logger = slog.Default()
	}
	return &Worker{pool: pool, sink: sink, logger: logger}
}

func (w *Worker) Start(ctx context.Context, routerID string, meta *command.CommandMeta) error {
	w.logger.Info("stream worker starting",
		"router_id", routerID,
		"measurement", meta.Measurement,
		"path", meta.Path,
	)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		if err := w.acquireAndStream(ctx, routerID, meta); err != nil {
			w.logger.Error("stream session ended",
				"router_id", routerID,
				"measurement", meta.Measurement,
				"error", err,
			)

			select {
			case <-time.After(5 * time.Second):
			case <-ctx.Done():
				return nil
			}
		} else {
			return nil
		}
	}
}

func (w *Worker) acquireAndStream(ctx context.Context, routerID string, meta *command.CommandMeta) error {
	client, err := w.pool.BorrowAsync(ctx, routerID)
	if err != nil {
		return fmt.Errorf("borrow async: %w", err)
	}

	w.logger.Info("stream worker acquired async conn",
		"router_id", routerID,
		"measurement", meta.Measurement,
	)

	streamErr := w.runStream(ctx, routerID, meta, client)

	// On error the shared async conn may be broken — invalidate so the next
	// BorrowAsync re-dials. On clean ctx cancellation streamErr is nil, so
	// the healthy conn is preserved for other workers.
	if streamErr != nil {
		w.pool.InvalidateAsync(routerID)
	}

	return streamErr
}

func (w *Worker) runStream(ctx context.Context, routerID string, meta *command.CommandMeta, client *routeros.Client) error {
	sentence := command.BuildStreamSentence(meta)
	reply, err := client.ListenArgsQueueContext(ctx, sentence, defaultQueueSize)
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
