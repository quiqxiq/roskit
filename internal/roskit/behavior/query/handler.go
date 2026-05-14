package query

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/quiqxiq/roskit/internal/roskit/behavior"
	"github.com/quiqxiq/roskit/internal/roskit/core/command"
	"github.com/quiqxiq/roskit/internal/roskit/execution"
)

type Handler struct {
	pool   *execution.Pool
	logger *slog.Logger
}

func NewHandler(pool *execution.Pool, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{pool: pool, logger: logger}
}

func (h *Handler) Query(
	ctx context.Context,
	routerID string,
	meta *command.CommandMeta,
	filters ...string,
) ([]map[string]string, error) {
	return h.readFromAPI(ctx, routerID, meta, filters...)
}

func (h *Handler) QueryOne(
	ctx context.Context,
	routerID string,
	meta *command.CommandMeta,
	filters ...string,
) (map[string]string, error) {
	results, err := h.Query(ctx, routerID, meta, filters...)
	if err != nil || len(results) == 0 {
		return nil, err
	}
	return results[0], nil
}

func (h *Handler) readFromAPI(
	ctx context.Context,
	routerID string,
	meta *command.CommandMeta,
	filters ...string,
) ([]map[string]string, error) {
	conn, err := h.pool.Borrow(ctx, routerID)
	if err != nil {
		return nil, fmt.Errorf("borrow connection for %s: %w", meta.Path, err)
	}
	defer h.pool.Return(routerID, conn)

	sentence := command.BuildQuerySentence(meta, filters)
	reply, err := conn.RunContext(ctx, sentence...)
	if err != nil {
		return nil, fmt.Errorf("RouterOS %s: %w", meta.Path, err)
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

var _ behavior.QueryHandler = (*Handler)(nil)
