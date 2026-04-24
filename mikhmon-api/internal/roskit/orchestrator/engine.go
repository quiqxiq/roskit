package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/quiqxiq/roskit/internal/roskit/behavior/mutation"
	"github.com/quiqxiq/roskit/internal/roskit/behavior/poll"
	"github.com/quiqxiq/roskit/internal/roskit/behavior/query"
	bstream "github.com/quiqxiq/roskit/internal/roskit/behavior/stream"
	"github.com/quiqxiq/roskit/internal/roskit/core/command"
	_ "github.com/quiqxiq/roskit/internal/roskit/core/definition"
	"github.com/quiqxiq/roskit/internal/roskit/execution"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/cache"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/event"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/pubsub"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/timeseries"
)

type Engine struct {
	mu sync.RWMutex

	pool      *execution.Pool
	streams   *bstream.Manager
	polls     *poll.Scheduler
	processor *event.Processor
	dispatch  *Dispatcher

	logger *slog.Logger
}

type Config struct {
	Logger     *slog.Logger
	Cache      cache.Repository
	TimeSeries timeseries.Writer
	PubSub     pubsub.Publisher
}

func New(cfg Config) *Engine {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Cache == nil {
		cfg.Cache = cache.NoopRepository{}
	}
	if cfg.TimeSeries == nil {
		cfg.TimeSeries = timeseries.NoopWriter{}
	}
	if cfg.PubSub == nil {
		cfg.PubSub = pubsub.NoopPublisher{}
	}

	pool := execution.NewPool(cfg.Logger)
	processor := event.NewProcessor(cfg.Cache, cfg.TimeSeries, cfg.PubSub, cfg.Logger)
	streamMgr := bstream.NewManager(pool, processor, cfg.Logger)
	pollSched := poll.NewScheduler(pool, processor, cfg.Logger)
	queryH := query.NewHandler(pool, cfg.Cache, cfg.Logger)
	mutateH := mutation.NewHandler(pool, queryH, cfg.Logger)

	dispatcher := NewDispatcher(
		bstream.NewWorker(pool, processor, cfg.Logger),
		&bstream.PollManager{},
		queryH,
		mutateH,
		pool,
	)

	return &Engine{
		pool:      pool,
		streams:   streamMgr,
		polls:     pollSched,
		processor: processor,
		dispatch:  dispatcher,
		logger:    cfg.Logger,
	}
}

func (e *Engine) AddRouter(ctx context.Context, cfg execution.ConnConfig) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.pool.Register(cfg)

	go func() {
		conn, err := e.pool.Borrow(ctx, cfg.RouterID)
		if err != nil {
			return
		}
		_ = conn
		e.startRouterWorkers(ctx, cfg.RouterID)
	}()

	return nil
}

func (e *Engine) RemoveRouter(routerID string) {
	e.streams.StopAll(routerID)
	e.polls.StopAll(routerID)
	e.pool.Unregister(routerID)
}

func (e *Engine) Start(ctx context.Context) {
	e.pool.Start(ctx)

	go func() {
		for routerID := range e.pool.Status() {
			e.startRouterWorkers(ctx, routerID)
		}
	}()
}

func (e *Engine) Stop() {
	e.streams.Shutdown()
	e.polls.Shutdown()
	e.pool.Stop()
	e.processor = nil
}

func (e *Engine) Dispatcher() *Dispatcher {
	return e.dispatch
}

func (e *Engine) Status() map[string]string {
	states := e.pool.Status()
	out := make(map[string]string, len(states))
	for id, state := range states {
		out[id] = state.String()
	}
	return out
}

func (e *Engine) ExecuteCommand(ctx context.Context, routerID string, sentence ...string) error {
	conn, err := e.pool.Borrow(ctx, routerID)
	if err != nil {
		return fmt.Errorf("engine: %w", err)
	}
	defer e.pool.Return(routerID, conn)

	_, err = conn.RunContext(ctx, sentence...)
	return err
}

func (e *Engine) startRouterWorkers(ctx context.Context, routerID string) {
	for _, meta := range command.ByType(command.CommandTypeStream) {
		e.streams.Start(ctx, routerID, meta)
	}

	for _, meta := range command.ByType(command.CommandTypePoll) {
		e.polls.Start(ctx, routerID, meta)
	}

	e.logger.Info("engine: workers started", "router_id", routerID,
		"streams", e.streams.ActiveCount(),
	)
}
