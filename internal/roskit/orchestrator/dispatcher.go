package orchestrator

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-routeros/routeros/v3"
	"github.com/quiqxiq/roskit/internal/roskit/behavior"
	"github.com/quiqxiq/roskit/internal/roskit/behavior/mutation"
	"github.com/quiqxiq/roskit/internal/roskit/behavior/query"
	"github.com/quiqxiq/roskit/internal/roskit/behavior/stream"
	"github.com/quiqxiq/roskit/internal/roskit/core/command"
	"github.com/quiqxiq/roskit/internal/roskit/execution"
)

type Dispatcher struct {
	mu       sync.Mutex
	streams  map[streamKey]context.CancelFunc

	streamHandler  behavior.StreamHandler
	pollManager    *stream.PollManager
	queryHandler   behavior.QueryHandler
	mutateHandler  behavior.MutationHandler
	pool           *execution.Pool
}

type streamKey struct {
	routerID string
	path     string
}

func NewDispatcher(
	sh behavior.StreamHandler,
	pm *stream.PollManager,
	qh behavior.QueryHandler,
	mh behavior.MutationHandler,
	pool *execution.Pool,
) *Dispatcher {
	return &Dispatcher{
		streams:        make(map[streamKey]context.CancelFunc),
		streamHandler:  sh,
		pollManager:    pm,
		queryHandler:   qh,
		mutateHandler:  mh,
		pool:           pool,
	}
}

func (d *Dispatcher) Meta(path string) *command.CommandMeta {
	return command.Lookup(path)
}

func (d *Dispatcher) Stream(ctx context.Context, routerID, path string) (context.CancelFunc, error) {
	meta := command.Lookup(path)
	if meta == nil {
		return nil, fmt.Errorf("dispatcher: unknown path %q", path)
	}
	if !meta.IsStream() {
		return nil, fmt.Errorf("dispatcher: %q has type %s, not stream", path, meta.Type)
	}

	key := streamKey{routerID, meta.Path}

	d.mu.Lock()
	if existing, ok := d.streams[key]; ok {
		d.mu.Unlock()
		return existing, nil
	}

	streamCtx, cancelFn := context.WithCancel(ctx)
	d.streams[key] = cancelFn
	d.mu.Unlock()

	go func() {
		defer func() {
			d.mu.Lock()
			delete(d.streams, key)
			d.mu.Unlock()
		}()
		_ = d.streamHandler.Start(streamCtx, routerID, meta)
	}()

	return cancelFn, nil
}

func (d *Dispatcher) StopStream(routerID, path string) {
	meta := command.Lookup(path)
	if meta == nil {
		return
	}
	key := streamKey{routerID, meta.Path}
	d.mu.Lock()
	if cancel, ok := d.streams[key]; ok {
		cancel()
		delete(d.streams, key)
	}
	d.mu.Unlock()
}

func (d *Dispatcher) Query(ctx context.Context, routerID, path string, filters ...string) ([]map[string]string, error) {
	meta := command.Lookup(path)
	if meta == nil {
		return nil, fmt.Errorf("dispatcher: unknown path %q", path)
	}
	if meta.Type != command.CommandTypeQuery && meta.Type != command.CommandTypePoll && meta.Type != command.CommandTypeStream {
		return nil, fmt.Errorf("dispatcher: %q has type %s, not query/poll/stream", path, meta.Type)
	}
	return d.queryHandler.Query(ctx, routerID, meta, filters...)
}

func (d *Dispatcher) Mutate(ctx context.Context, routerID, path string, args ...string) (*routeros.Reply, error) {
	meta := command.Lookup(path)
	if meta == nil {
		return nil, fmt.Errorf("dispatcher: unknown path %q", path)
	}
	if meta.Type != command.CommandTypeMutation && meta.Type != command.CommandTypeAction {
		return nil, fmt.Errorf("dispatcher: %q has type %s, not mutation/action", path, meta.Type)
	}
	return d.mutateHandler.Execute(ctx, routerID, meta, args...)
}

func (d *Dispatcher) ActiveStreams() []streamKey {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]streamKey, 0, len(d.streams))
	for k := range d.streams {
		out = append(out, k)
	}
	return out
}

func (d *Dispatcher) Run(ctx context.Context, routerID string, sentence ...string) (*routeros.Reply, error) {
	if d.pool == nil {
		return nil, fmt.Errorf("dispatcher: no pool")
	}
	conn, err := d.pool.Borrow(ctx, routerID)
	if err != nil {
		return nil, fmt.Errorf("dispatcher: %w", err)
	}
	defer d.pool.Return(routerID, conn)
	return conn.RunContext(ctx, sentence...)
}

// RunListen starts a listen (streaming) command on the client connection and
// returns a ListenReply whose channel delivers each !re sentence as it arrives.
// Callers must call reply.CancelContext when done to release the router-side listener.
func (d *Dispatcher) RunListen(ctx context.Context, routerID string, sentence []string) (*routeros.ListenReply, error) {
	if d.pool == nil {
		return nil, fmt.Errorf("dispatcher: no pool")
	}
	conn, err := d.pool.Borrow(ctx, routerID)
	if err != nil {
		return nil, fmt.Errorf("dispatcher: %w", err)
	}
	defer d.pool.Return(routerID, conn)
	return conn.Client().ListenArgsQueueContext(ctx, sentence, 64)
}

func (d *Dispatcher) PoolStatus() map[string]execution.ConnState {
	if d.pool == nil {
		return nil
	}
	return d.pool.Status()
}

var _ behavior.Dispatcher = (*Dispatcher)(nil)

var (
	_ = mutation.Handler{}
	_ = query.Handler{}
	_ = stream.PollManager{}
)
