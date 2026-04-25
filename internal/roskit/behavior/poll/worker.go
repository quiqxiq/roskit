package poll

import (
	"context"
	"log/slog"
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/behavior"
	"github.com/quiqxiq/roskit/internal/roskit/core/command"
	"github.com/quiqxiq/roskit/internal/roskit/execution"
)

type Worker struct {
	pool   *execution.Pool
	sink   behavior.PollSink
	logger *slog.Logger
}

func NewWorker(pool *execution.Pool, sink behavior.PollSink, logger *slog.Logger) *Worker {
	if logger == nil {
		logger = slog.Default()
	}
	return &Worker{pool: pool, sink: sink, logger: logger}
}

func (w *Worker) Start(ctx context.Context, routerID string, meta *command.CommandMeta) error {
	interval := command.PollIntervalOrDefault(meta, 60*time.Second)
	w.logger.Info("poll worker starting",
		"router_id", routerID,
		"measurement", meta.Measurement,
		"interval", interval,
	)

	w.doPoll(ctx, routerID, meta)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("poll worker stopped",
				"router_id", routerID,
				"measurement", meta.Measurement,
			)
			return nil
		case <-ticker.C:
			w.doPoll(ctx, routerID, meta)
		}
	}
}

func (w *Worker) doPoll(ctx context.Context, routerID string, meta *command.CommandMeta) {
	conn, err := w.pool.Borrow(ctx, routerID)
	if err != nil {
		w.logger.Warn("poll: borrow failed",
			"router_id", routerID,
			"measurement", meta.Measurement,
			"error", err,
		)
		w.sink.OnPoll(ctx, behavior.PollEvent{
			Meta:     meta,
			RouterID: routerID,
			Err:      err,
			PollTime: time.Now(),
		})
		return
	}
	defer w.pool.Return(routerID, conn)

	sentence := command.BuildPollSentence(meta)
	reply, err := conn.RunContext(ctx, sentence...)
	if err != nil {
		w.logger.Warn("poll: command failed",
			"router_id", routerID,
			"measurement", meta.Measurement,
			"error", err,
		)
		w.sink.OnPoll(ctx, behavior.PollEvent{
			Meta:     meta,
			RouterID: routerID,
			Err:      err,
			PollTime: time.Now(),
		})
		return
	}

	var rows []map[string]string
	for _, s := range reply.Re {
		row := make(map[string]string, len(s.Map))
		for k, v := range s.Map {
			row[k] = v
		}
		rows = append(rows, row)
	}

	w.sink.OnPoll(ctx, behavior.PollEvent{
		Meta:     meta,
		RouterID: routerID,
		Rows:     rows,
		PollTime: time.Now(),
	})
}

var _ behavior.PollHandler = (*Worker)(nil)
