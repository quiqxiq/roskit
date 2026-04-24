package mutation

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-routeros/routeros/v3"
	"github.com/quiqxiq/roskit/internal/roskit/behavior"
	"github.com/quiqxiq/roskit/internal/roskit/core/command"
	"github.com/quiqxiq/roskit/internal/roskit/execution"
)

type Handler struct {
	pool   *execution.Pool
	query  behavior.QueryHandler
	logger *slog.Logger
}

func NewHandler(pool *execution.Pool, query behavior.QueryHandler, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{pool: pool, query: query, logger: logger}
}

func (h *Handler) Execute(ctx context.Context, routerID string, meta *command.CommandMeta, args ...string) (*routeros.Reply, error) {
	if meta == nil {
		return nil, fmt.Errorf("mutation: nil meta")
	}

	exec := execution.NewExecutor(h.pool)
	path := meta.Path
	sentence := append([]string{"/" + path}, args...)
	return exec.Run(ctx, routerID, sentence...)
}

var _ behavior.MutationHandler = (*Handler)(nil)
