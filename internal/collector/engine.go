package collector

import (
	"context"
	"log/slog"
	"sync"

	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/internal/spec"
	"github.com/quiqxiq/roskit/internal/usecase"
)

// Engine is the top-level collector that manages the connection pool and
// per-router data collection. It provides a unified lifecycle management
// interface for starting, stopping, and dynamically adding/removing routers.
//
// Architecture:
//
//	Engine
//	  ├── ConnPool (owns persistent connections)
//	  │     ├── RouterConn["core-01"] → persistent TCP connection
//	  │     ├── RouterConn["edge-01"] → persistent TCP connection
//	  │     └── RouterConn["ap-01"]   → persistent TCP connection
//	  │
//	  └── Collectors (borrows connections from pool)
//	        ├── RouterCollector["core-01"] → acquires/releases from pool
//	        ├── RouterCollector["edge-01"] → acquires/releases from pool
//	        └── RouterCollector["ap-01"]   → acquires/releases from pool
type Engine struct {
	pool       *ConnPool
	usecase    *usecase.TelemetryUseCase
	logger     *slog.Logger
	collectors map[string]*RouterCollector
	mu         sync.RWMutex
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// NewEngine creates a new collector engine with a connection pool.
// The engine uses the provided TelemetryUseCase for event processing
// and manages a ConnPool that handles persistent router connections.
func NewEngine(uc *usecase.TelemetryUseCase, logger *slog.Logger) *Engine {
	return &Engine{
		pool:       NewConnPool(logger),
		usecase:    uc,
		logger:     logger,
		collectors: make(map[string]*RouterCollector),
	}
}

// AddRouter registers a router with the specified stream specs.
// This registers the router in both the connection pool (for persistent
// connection management) and creates a collector (for spec execution).
//
// The router will start collecting when Start() is called.
func (e *Engine) AddRouter(config domain.RouterConfig, specs []spec.StreamSpec) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Remove existing if replacing.
	if _, exists := e.collectors[config.ID]; exists {
		e.logger.Warn("router already registered, replacing", "router", config.ID)
		e.collectors[config.ID].Stop()
		e.pool.Unregister(config.ID)
	}

	// Register in pool first (connection management).
	e.pool.Register(config)

	// Create collector (spec execution) — it borrows from the pool.
	rc := NewRouterCollector(config, specs, e.usecase, e.pool, e.logger)
	e.collectors[config.ID] = rc

	e.logger.Info("router registered",
		"router", config.ID,
		"address", config.Address,
		"specs", len(specs),
	)
}

// RemoveRouter stops collection for the specified router, closes its
// persistent connection, and removes it from the engine.
func (e *Engine) RemoveRouter(routerID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if rc, exists := e.collectors[routerID]; exists {
		rc.Stop()
		delete(e.collectors, routerID)
	}

	e.pool.Unregister(routerID)
	e.logger.Info("router removed", "router", routerID)
}

// Start begins data collection for all registered routers.
// It first starts the connection pool (establishes persistent connections
// and begins health monitoring), then starts all collectors.
func (e *Engine) Start(ctx context.Context) {
	var childCtx context.Context
	childCtx, e.cancel = context.WithCancel(ctx)

	// Start the connection pool first — this establishes all connections.
	e.pool.Start(childCtx)

	e.mu.RLock()
	defer e.mu.RUnlock()

	e.logger.Info("starting collector engine",
		"routers", len(e.collectors),
	)

	// Start all collectors — they acquire connections from the pool.
	for id, rc := range e.collectors {
		e.wg.Add(1)
		go func(id string, rc *RouterCollector) {
			defer e.wg.Done()
			rc.Start(childCtx)
		}(id, rc)
	}
}

// Stop gracefully shuts down all collectors and the connection pool.
// Order: cancel context → wait for collectors → stop pool.
func (e *Engine) Stop() {
	e.logger.Info("stopping collector engine")

	// Cancel all contexts.
	if e.cancel != nil {
		e.cancel()
	}

	// Wait for all collector goroutines to finish.
	e.wg.Wait()

	// Stop the connection pool (closes all connections, stops health monitor).
	e.pool.Stop()

	e.logger.Info("collector engine stopped")
}

// Pool returns the connection pool for external status monitoring.
func (e *Engine) Pool() *ConnPool {
	return e.pool
}

// RouterIDs returns a list of all registered router IDs.
func (e *Engine) RouterIDs() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	ids := make([]string, 0, len(e.collectors))
	for id := range e.collectors {
		ids = append(ids, id)
	}
	return ids
}

// Status returns the connection state of all managed routers.
// Useful for dashboards and health endpoints.
func (e *Engine) Status() map[string]ConnState {
	return e.pool.Status()
}
