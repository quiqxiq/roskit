package stream

import (
	"context"
	"log/slog"
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/behavior"
	"github.com/quiqxiq/roskit/internal/roskit/core/command"
	"github.com/quiqxiq/roskit/internal/roskit/execution"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/cache"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/pubsub"
)

type LogWorker struct {
	pool   *execution.Pool
	pubsub pubsub.Publisher
	logger *slog.Logger
}

func NewLogWorker(pool *execution.Pool, ps pubsub.Publisher, logger *slog.Logger) *LogWorker {
	if logger == nil {
		logger = slog.Default()
	}
	return &LogWorker{pool: pool, pubsub: ps, logger: logger}
}

func (w *LogWorker) Start(ctx context.Context, routerID, filter string, interval time.Duration) {
	if interval == 0 {
		interval = 5 * time.Second
	}

	w.logger.Info("log worker starting",
		"router_id", routerID, "filter", filter, "interval", interval)

	var lastID string

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rows, err := w.pollLogs(ctx, routerID, filter)
			if err != nil {
				w.logger.Debug("log worker poll error",
					"router_id", routerID, "filter", filter, "err", err)
				continue
			}

			if len(rows) == 0 {
				continue
			}

			newRows, newLastID := filterNewLogs(rows, lastID)
			if newLastID != "" {
				lastID = newLastID
			}

			for _, row := range newRows {
				channel := cache.FormatLogChannel(routerID, filter)
				w.pubsub.Publish(ctx, channel, pubsub.Message{
					Type:        "log",
					RouterID:    routerID,
					Measurement: "system_log",
					Fields:      row,
					Timestamp:   time.Now(),
				})
			}
		}
	}
}

func filterNewLogs(rows []map[string]string, lastID string) ([]map[string]string, string) {
	if lastID == "" {
		if len(rows) > 0 {
			return nil, rows[len(rows)-1][".id"]
		}
		return nil, ""
	}

	var newRows []map[string]string
	found := false
	for _, row := range rows {
		if found {
			newRows = append(newRows, row)
		} else if row[".id"] == lastID {
			found = true
		}
	}

	if !found {
		if len(rows) > 0 {
			return nil, rows[len(rows)-1][".id"]
		}
		return nil, lastID
	}

	newLastID := lastID
	if len(newRows) > 0 {
		newLastID = newRows[len(newRows)-1][".id"]
	}
	return newRows, newLastID
}

func (w *LogWorker) pollLogs(ctx context.Context, routerID, filter string) ([]map[string]string, error) {
	conn, err := w.pool.Borrow(ctx, routerID)
	if err != nil {
		return nil, err
	}
	defer w.pool.Return(routerID, conn)

	meta := command.Lookup("log/print")
	if meta == nil {
		return nil, nil
	}

	sentence := []string{"/" + meta.Path}
	if filter != "" && filter != "all" {
		switch filter {
		case "hotspot":
			sentence = append(sentence, "?topics=hotspot,info,debug")
		case "ppp":
			sentence = append(sentence, "?topics=ppp,info,debug")
		}
	}

	reply, err := conn.RunContext(ctx, sentence...)
	if err != nil {
		return nil, err
	}

	results := make([]map[string]string, 0, len(reply.Re))
	for _, s := range reply.Re {
		row := make(map[string]string, len(s.Map))
		for k, v := range s.Map {
			row[k] = v
		}
		results = append(results, row)
	}
	return results, nil
}

var _ behavior.StreamHandler = (*Worker)(nil)
