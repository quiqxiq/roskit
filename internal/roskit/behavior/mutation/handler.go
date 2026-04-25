package mutation

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/go-routeros/routeros/v3"
	"github.com/quiqxiq/roskit/internal/roskit/behavior"
	"github.com/quiqxiq/roskit/internal/roskit/core/command"
	"github.com/quiqxiq/roskit/internal/roskit/execution"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/cache"
)

type Handler struct {
	pool   *execution.Pool
	query  behavior.QueryHandler
	cache  cache.Repository
	logger *slog.Logger
}

func NewHandler(pool *execution.Pool, query behavior.QueryHandler, c cache.Repository, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{pool: pool, query: query, cache: c, logger: logger}
}

func (h *Handler) Execute(ctx context.Context, routerID string, meta *command.CommandMeta, args ...string) (*routeros.Reply, error) {
	if meta == nil {
		return nil, fmt.Errorf("mutation: nil meta")
	}

	exec := execution.NewExecutor(h.pool)
	path := meta.Path
	sentence := append([]string{"/" + path}, args...)
	reply, err := exec.Run(ctx, routerID, sentence...)
	if err != nil {
		return nil, err
	}

	h.invalidateCache(ctx, routerID, meta.Path)
	return reply, nil
}

// invalidateCache deletes all cache entries for the stream measurement that corresponds
// to this mutation path. Derives the measurement by stripping the mutation verb and
// looking up the equivalent print path.
func (h *Handler) invalidateCache(ctx context.Context, routerID, mutationPath string) {
	if h.cache == nil {
		return
	}
	measurement := streamMeasurementFor(mutationPath)
	if measurement == "" {
		return
	}

	if err := h.cache.DeleteByMeasurement(ctx, routerID, measurement); err != nil {
		h.logger.Warn("mutation: cache invalidation failed",
			"router_id", routerID, "measurement", measurement, "err", err)
	}

	indexKey := cache.FormatIndexKey(routerID, measurement)
	if err := h.cache.DeleteSnapshot(ctx, indexKey); err != nil {
		h.logger.Warn("mutation: index invalidation failed",
			"router_id", routerID, "index_key", indexKey, "err", err)
	}
}

// streamMeasurementFor derives the InfluxDB/Redis measurement name for the stream
// that is affected by a mutation. Strips the verb (last path segment) and looks up
// the matching stream definition.
//
// Example: "ip/hotspot/user/add" → lookup "ip/hotspot/user/print" → "hotspot_user"
func streamMeasurementFor(mutationPath string) string {
	parts := strings.Split(mutationPath, "/")
	if len(parts) < 2 {
		return ""
	}
	streamPath := strings.Join(parts[:len(parts)-1], "/") + "/print"
	meta := command.Lookup(streamPath)
	if meta == nil {
		return ""
	}
	return meta.Measurement
}

var _ behavior.MutationHandler = (*Handler)(nil)
