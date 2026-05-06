package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-routeros/routeros/v3"
	"github.com/quiqxiq/roskit/internal/roskit/core/command"
	"github.com/quiqxiq/roskit/internal/roskit/orchestrator"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/cache"
)

type Bridge struct {
	dispatcher *orchestrator.Dispatcher
	cache      cache.Repository
	logger     *slog.Logger
}

func NewBridge(dispatcher *orchestrator.Dispatcher, c cache.Repository) *Bridge {
	return &Bridge{dispatcher: dispatcher, cache: c, logger: slog.Default()}
}

// logCleanupErr is a helper for best-effort cleanup mutations whose failures
// should be observable but never propagated to the caller. Cleanup is by
// definition non-fatal; orphaned MikroTik resources surface here so operators
// can investigate without breaking the primary user flow.
func (b *Bridge) logCleanupErr(ctx context.Context, routerID, op, targetID string, err error, kv ...any) {
	if err == nil {
		return
	}
	attrs := []any{
		"router_id", routerID,
		"operation", op,
		"target_id", targetID,
		"error", err,
	}
	attrs = append(attrs, kv...)
	b.logger.WarnContext(ctx, "roskit cleanup mutate failed", attrs...)
}

func (b *Bridge) Query(ctx context.Context, routerID, path string, filters ...string) ([]map[string]string, error) {
	return b.dispatcher.Query(ctx, routerID, path, filters...)
}

func (b *Bridge) QueryOne(ctx context.Context, routerID, path string, filters ...string) (map[string]string, error) {
	results, err := b.Query(ctx, routerID, path, filters...)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("roskit: %s not found on router %s", path, routerID)
	}
	return results[0], nil
}

func (b *Bridge) Mutate(ctx context.Context, routerID, path string, args ...string) (*routeros.Reply, error) {
	return b.dispatcher.Mutate(ctx, routerID, path, args...)
}

func (b *Bridge) Lookup(path string) *command.CommandMeta {
	return b.dispatcher.Meta(path)
}

func (b *Bridge) extractRetID(reply *routeros.Reply) string {
	if reply.Done != nil {
		for _, pair := range reply.Done.List {
			if pair.Key == "ret" {
				return pair.Value
			}
		}
	}
	return ""
}

func (b *Bridge) mutateAdd(ctx context.Context, routerID, path string, params map[string]string) (string, error) {
	args := []string{}
	for k, v := range params {
		args = append(args, "="+k+"="+v)
	}
	reply, err := b.Mutate(ctx, routerID, path, args...)
	if err != nil {
		return "", err
	}
	return b.extractRetID(reply), nil
}

func (b *Bridge) Run(ctx context.Context, routerID string, sentence ...string) (*routeros.Reply, error) {
	return b.dispatcher.Run(ctx, routerID, sentence...)
}

func (b *Bridge) PoolStatus() map[string]bool {
	states := b.dispatcher.PoolStatus()
	if states == nil {
		return nil
	}
	out := make(map[string]bool, len(states))
	for id, state := range states {
		out[id] = state.String() == "connected"
	}
	return out
}
