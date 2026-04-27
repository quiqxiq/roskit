package poll

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/behavior"
	"github.com/quiqxiq/roskit/internal/roskit/core/command"
	"github.com/quiqxiq/roskit/internal/roskit/execution"
)

// ConcurrentRunner runs all provided poll commands simultaneously on a single
// router connection using go-routeros async tag-multiplexing.
type ConcurrentRunner struct {
	pool   *execution.Pool
	sink   behavior.PollSink
	logger *slog.Logger
}

func NewConcurrentRunner(pool *execution.Pool, sink behavior.PollSink, logger *slog.Logger) *ConcurrentRunner {
	if logger == nil {
		logger = slog.Default()
	}
	return &ConcurrentRunner{pool: pool, sink: sink, logger: logger}
}

// RunAll fires all metas concurrently and waits until every one completes.
// The underlying client connection is in async mode, so all commands run
// over the same TCP connection via tag multiplexing.
func (r *ConcurrentRunner) RunAll(ctx context.Context, routerID string, metas []*command.CommandMeta) {
	var wg sync.WaitGroup
	for _, meta := range metas {
		wg.Add(1)
		go func(m *command.CommandMeta) {
			defer wg.Done()
			r.runOne(ctx, routerID, m)
		}(meta)
	}
	wg.Wait()
}

func (r *ConcurrentRunner) runOne(ctx context.Context, routerID string, meta *command.CommandMeta) {
	conn, err := r.pool.Borrow(ctx, routerID)
	if err != nil {
		r.logger.Warn("concurrent poll: borrow failed",
			"router_id", routerID,
			"measurement", meta.Measurement,
			"err", err,
		)
		r.sink.OnPoll(ctx, behavior.PollEvent{ //nolint:errcheck
			Meta:     meta,
			RouterID: routerID,
			Err:      err,
			PollTime: time.Now(),
		})
		return
	}
	defer r.pool.Return(routerID, conn)

	sentence := command.BuildPollSentence(meta)
	reply, err := conn.RunContext(ctx, sentence...)
	if err != nil {
		r.logger.Warn("concurrent poll: command failed",
			"router_id", routerID,
			"measurement", meta.Measurement,
			"err", err,
		)
		r.sink.OnPoll(ctx, behavior.PollEvent{ //nolint:errcheck
			Meta:     meta,
			RouterID: routerID,
			Err:      err,
			PollTime: time.Now(),
		})
		return
	}

	rows := make([]map[string]string, 0, len(reply.Re))
	for _, s := range reply.Re {
		row := make(map[string]string, len(s.Map))
		for k, v := range s.Map {
			row[k] = v
		}
		rows = append(rows, row)
	}

	r.sink.OnPoll(ctx, behavior.PollEvent{ //nolint:errcheck
		Meta:     meta,
		RouterID: routerID,
		Rows:     rows,
		PollTime: time.Now(),
	})
}
