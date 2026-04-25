package query

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/behavior"
	"github.com/quiqxiq/roskit/internal/roskit/core/command"
	"github.com/quiqxiq/roskit/internal/roskit/execution"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/cache"
)

type Handler struct {
	pool   *execution.Pool
	cache  cache.Repository
	logger *slog.Logger
}

func NewHandler(pool *execution.Pool, c cache.Repository, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{pool: pool, cache: c, logger: logger}
}

func (h *Handler) Query(
	ctx context.Context,
	routerID string,
	meta *command.CommandMeta,
	filters ...string,
) ([]map[string]string, error) {
	if h.cache != nil && meta.Type.IsCacheable() {
		results, err := h.readFromCache(ctx, routerID, meta, filters...)
		if err == nil && len(results) > 0 {
			h.logger.Debug("query: cache hit",
				"router_id", routerID,
				"measurement", meta.Measurement,
				"count", len(results),
			)
			return results, nil
		}
		if err != nil {
			h.logger.Debug("query: cache read error, falling back to API",
				"router_id", routerID,
				"measurement", meta.Measurement,
				"err", err,
			)
		}
	}

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

func (h *Handler) GetByIndex(ctx context.Context, routerID string, meta *command.CommandMeta, name string) (map[string]string, error) {
	if h.cache == nil || meta.IndexField == "" {
		return nil, nil
	}

	indexKey := cache.FormatIndexKey(routerID, meta.Measurement)
	data, err := h.cache.GetByIndex(ctx, indexKey, name)
	if err != nil || len(data) == 0 {
		return nil, err
	}
	return data, nil
}

func (h *Handler) Count(ctx context.Context, routerID string, meta *command.CommandMeta) (int, error) {
	if h.cache == nil {
		return -1, nil
	}
	return h.cache.CountByMeasurement(ctx, routerID, meta.Measurement)
}

func (h *Handler) readFromCache(
	ctx context.Context,
	routerID string,
	meta *command.CommandMeta,
	filters ...string,
) ([]map[string]string, error) {
	all, err := h.cache.ScanByMeasurement(ctx, routerID, meta.Measurement)
	if err != nil {
		return nil, fmt.Errorf("ScanByMeasurement: %w", err)
	}

	if len(filters) == 0 {
		return all, nil
	}

	parsed := parseFilters(filters)
	if parsed == nil {
		return nil, fmt.Errorf("complex filter, fall back to API")
	}

	var filtered []map[string]string
	for _, row := range all {
		if matchesFilters(row, parsed) {
			filtered = append(filtered, row)
		}
	}
	return filtered, nil
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

	if h.cache != nil && meta.CacheTTL > 0 {
		for _, row := range results {
			id, ok := row[".id"]
			if !ok {
				continue
			}
			key := cache.FormatCacheKey(routerID, meta.Measurement, id)
			if err := h.cache.SetSnapshot(ctx, key, row, meta.CacheTTL); err != nil {
				h.logger.Warn("query: warm cache failed", "key", key, "err", err)
			}
		}
	}

	return results, nil
}

type filterPredicate struct {
	key     string
	value   string
	negate  bool
}

func parseFilters(filters []string) []filterPredicate {
	predicates := make([]filterPredicate, 0, len(filters))
	for _, f := range filters {
		if !strings.HasPrefix(f, "?") {
			continue
		}
		body := f[1:]
		negate := strings.HasPrefix(body, "!")
		if negate {
			body = body[1:]
		}
		parts := strings.SplitN(body, "=", 2)
		if len(parts) != 2 {
			return nil
		}
		predicates = append(predicates, filterPredicate{
			key:    parts[0],
			value:  parts[1],
			negate: negate,
		})
	}
	return predicates
}

func matchesFilters(row map[string]string, predicates []filterPredicate) bool {
	for _, p := range predicates {
		v, ok := row[p.key]
		matches := ok && v == p.value
		if p.negate {
			if matches {
				return false
			}
		} else {
			if !matches {
				return false
			}
		}
	}
	return true
}

var _ behavior.QueryHandler = (*Handler)(nil)

var _ = time.Second
