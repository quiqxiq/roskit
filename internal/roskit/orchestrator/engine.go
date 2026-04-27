package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

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

	cfg       Config
	appCtx    context.Context // set by Start; used by AddRouter to launch workers
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
	// DisableStreams suppresses stream workers in startRouterWorkers.
	// Set to true in cmd/worker so only poll connections are established.
	DisableStreams bool
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
	mutateH := mutation.NewHandler(pool, queryH, cfg.Cache, cfg.Logger)

	dispatcher := NewDispatcher(
		bstream.NewWorker(pool, processor, cfg.Logger),
		&bstream.PollManager{},
		queryH,
		mutateH,
		pool,
	)

	return &Engine{
		cfg:       cfg,
		pool:      pool,
		streams:   streamMgr,
		polls:     pollSched,
		processor: processor,
		dispatch:  dispatcher,
		logger:    cfg.Logger,
	}
}

func (e *Engine) AddRouter(_ context.Context, cfg execution.ConnConfig) error {
	e.mu.Lock()
	appCtx := e.appCtx
	e.pool.Register(cfg)
	e.mu.Unlock()

	// Called before Start() (e.g. seed phase) — Start() will connect and launch workers.
	if appCtx == nil {
		return nil
	}

	e.pool.LaunchOne(cfg.RouterID)

	go func() {
		waitCtx, cancel := context.WithTimeout(appCtx, 30*time.Second)
		defer cancel()
		if err := e.pool.WaitConnected(waitCtx, cfg.RouterID); err != nil {
			e.logger.Warn("engine: router did not connect in time",
				"router_id", cfg.RouterID, "err", err)
			return
		}
		e.startRouterWorkers(appCtx, cfg.RouterID)
	}()

	return nil
}

func (e *Engine) RemoveRouter(routerID string) {
	e.streams.StopAll(routerID)
	e.polls.StopAll(routerID)
	// Unregister runs async: Close() can block if a poll/health goroutine holds
	// the connection read lock during a network operation. Cancelling the workers
	// above is sufficient to stop activity; the TCP teardown can happen in the bg.
	go e.pool.Unregister(routerID)
}

func (e *Engine) Start(ctx context.Context) {
	e.mu.Lock()
	e.appCtx = ctx
	e.mu.Unlock()

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
	if !e.cfg.DisableStreams {
		for _, meta := range command.ByType(command.CommandTypeStream) {
			e.streams.Start(ctx, routerID, meta)
		}
	}
	// Polls are driven by cmd/worker's ConcurrentRunner every 30 s.
	// The engine no longer starts per-command poll workers.

	e.logger.Info("engine: workers started", "router_id", routerID,
		"streams", e.streams.ActiveCount(),
	)
}

// NewConcurrentPollRunner returns a runner that executes all registered poll
// commands concurrently for a given router using the engine's pool and processor.
func (e *Engine) NewConcurrentPollRunner() *poll.ConcurrentRunner {
	return poll.NewConcurrentRunner(e.pool, e.processor, e.logger)
}
