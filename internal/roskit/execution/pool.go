// Package execution manages RouterOS TCP connections.
// It is a pure I/O layer — no knowledge of behaviors, specs, or pipeline.
//
// Dependency rule: execution/ → nothing internal (only go-routeros + stdlib)
package execution

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/go-routeros/routeros/v3"
)

// ConnConfig holds parameters for one router connection.
type ConnConfig struct {
	RouterID string
	Address  string // host:port, e.g. "192.168.1.1:8728"
	Username string
	Password string

	DialTimeout    time.Duration // default 10s
	ReadTimeout    time.Duration // default 15s
	WriteTimeout   time.Duration // default 15s
	HealthInterval time.Duration // default 30s
}

func (c ConnConfig) withDefaults() ConnConfig {
	if c.DialTimeout == 0 {
		c.DialTimeout = 10 * time.Second
	}
	if c.ReadTimeout == 0 {
		c.ReadTimeout = 15 * time.Second
	}
	if c.WriteTimeout == 0 {
		c.WriteTimeout = 15 * time.Second
	}
	if c.HealthInterval == 0 {
		c.HealthInterval = 30 * time.Second
	}
	return c
}

// RouterConn is a managed connection to one MikroTik router.
// It holds TWO persistent connections, each with its own TCP socket,
// async mode (tag-multiplexing), watchAsync goroutine, and auto-reconnect:
//
//   - client: persistent conn for poll / query / mutation (tag-multiplexed)
//   - stream: persistent conn for ALL stream workers (tag-multiplexed)
//
// Both connections remain open for the lifetime of the application.
// Commands never create new TCP connections — they reuse the existing one
// via getConn(). When a connection drops, the watchAsync goroutine detects
// it and triggers auto-reconnect with exponential backoff.
type RouterConn struct {
	mu     sync.RWMutex
	cfg    ConnConfig
	client *PersistentConn
	stream *PersistentConn
	logger *slog.Logger
}

func newRouterConn(cfg ConnConfig, logger *slog.Logger) *RouterConn {
	resolved := cfg.withDefaults()
	return &RouterConn{
		cfg:    resolved,
		client: newPersistentConn(resolved, RoleClient, logger),
		stream: newPersistentConn(resolved, RoleStream, logger),
		logger: logger.With("router_id", cfg.RouterID),
	}
}

// Connect dials and authenticates both persistent connections.
// Idempotent — noop for connections already established.
func (rc *RouterConn) Connect(ctx context.Context) error {
	if err := rc.client.Connect(ctx); err != nil {
		return fmt.Errorf("client conn: %w", err)
	}
	if err := rc.stream.Connect(ctx); err != nil {
		return fmt.Errorf("stream conn: %w", err)
	}
	return nil
}

// ConnectWithBackoff retries connection with exponential backoff until ctx is cancelled.
func (rc *RouterConn) ConnectWithBackoff(ctx context.Context) error {
	if err := rc.client.ConnectWithBackoff(ctx); err != nil {
		return fmt.Errorf("client conn: %w", err)
	}
	if err := rc.stream.ConnectWithBackoff(ctx); err != nil {
		return fmt.Errorf("stream conn: %w", err)
	}
	return nil
}

// Close closes both persistent connections. Idempotent.
func (rc *RouterConn) Close() {
	rc.stream.Close()
	rc.client.Close()
}

// BorrowStream returns the persistent stream connection's underlying client.
// All stream workers share this single connection via tag-multiplexing.
// Callers must NOT close the returned client.
func (rc *RouterConn) BorrowStream() (*PersistentConn, error) {
	if rc.stream.State() != ConnStateConnected {
		return nil, fmt.Errorf("stream conn not connected for %s", rc.cfg.RouterID)
	}
	return rc.stream, nil
}

// RunContext executes a command on the client connection.
// Safe for concurrent use — the client is in async mode so go-routeros
// handles tag-based multiplexing internally.
func (rc *RouterConn) RunContext(ctx context.Context, sentence ...string) (*routeros.Reply, error) {
	return rc.client.RunContext(ctx, sentence...)
}

func (rc *RouterConn) Client() *PersistentConn {
	return rc.client
}

func (rc *RouterConn) Stream() *PersistentConn {
	return rc.stream
}

// IsAlive checks health of the client connection.
func (rc *RouterConn) IsAlive(ctx context.Context) bool {
	return rc.client.IsAlive(ctx)
}

// State returns the worst state between client and stream connections.
func (rc *RouterConn) State() ConnState {
	cs := rc.client.State()
	ss := rc.stream.State()
	if cs == ConnStateConnected && ss == ConnStateConnected {
		return ConnStateConnected
	}
	if cs == ConnStateAuthFailed || ss == ConnStateAuthFailed {
		return ConnStateAuthFailed
	}
	if cs == ConnStateConnecting || ss == ConnStateConnecting {
		return ConnStateConnecting
	}
	return ConnStateDisconnected
}

// =============================================================================
// Pool — manages multiple RouterConn instances
// =============================================================================

// Pool manages connections to multiple routers.
// Each router gets two persistent connections (client + stream) that remain
// open for the application lifetime. Commands reuse existing connections
// via tag-multiplexing — no new TCP connections are created per command.
type Pool struct {
	mu      sync.RWMutex
	conns   map[string]*RouterConn
	logger  *slog.Logger

	appCtx       context.Context
	healthCtx    context.Context
	healthCancel context.CancelFunc
}

// NewPool creates an empty connection pool.
func NewPool(logger *slog.Logger) *Pool {
	if logger == nil {
		logger = slog.Default()
	}
	return &Pool{
		conns:  make(map[string]*RouterConn),
		logger: logger,
	}
}

// Register adds a router configuration to the pool.
// Does NOT connect — call Start() to initiate connections.
func (p *Pool) Register(cfg ConnConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.conns[cfg.RouterID] = newRouterConn(cfg, p.logger)
}

// Unregister removes and closes a router connection.
func (p *Pool) Unregister(routerID string) {
	p.mu.Lock()
	conn, ok := p.conns[routerID]
	delete(p.conns, routerID)
	p.mu.Unlock()

	if ok {
		conn.Close()
	}
}

// Start connects all registered routers and starts health monitoring.
func (p *Pool) Start(ctx context.Context) {
	p.mu.RLock()
	conns := make([]*RouterConn, 0, len(p.conns))
	for _, c := range p.conns {
		conns = append(conns, c)
	}
	p.mu.RUnlock()

	healthCtx, cancel := context.WithCancel(ctx)
	p.mu.Lock()
	p.appCtx = ctx
	p.healthCtx = healthCtx
	p.healthCancel = cancel
	p.mu.Unlock()

	for _, conn := range conns {
		go conn.client.ConnectWithBackoff(ctx) //nolint:errcheck
		go conn.stream.ConnectWithBackoff(ctx) //nolint:errcheck
		go p.healthLoop(healthCtx, conn)
	}
}

// Stop closes all connections and stops health monitoring.
func (p *Pool) Stop() {
	p.mu.Lock()
	cancel := p.healthCancel
	conns := make([]*RouterConn, 0, len(p.conns))
	for _, c := range p.conns {
		conns = append(conns, c)
	}
	p.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	for _, c := range conns {
		c.Close()
	}
}

// Borrow returns a connected RouterConn for the given routerID.
// Returns an error if the router is not registered or not connected.
// The caller MUST call Return when done.
func (p *Pool) Borrow(ctx context.Context, routerID string) (*RouterConn, error) {
	p.mu.RLock()
	conn, ok := p.conns[routerID]
	p.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("pool: router %q not registered", routerID)
	}
	if conn.client.State() != ConnStateConnected {
		return nil, fmt.Errorf("pool: router %q client is %s", routerID, conn.client.State())
	}
	return conn, nil
}

// Return releases a borrowed connection back to the pool.
// Noop — connections are persistent and never released.
func (p *Pool) Return(routerID string, conn *RouterConn) {
	// noop — persistent connections
}

// Status returns connection states for all routers.
func (p *Pool) Status() map[string]ConnState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make(map[string]ConnState, len(p.conns))
	for id, conn := range p.conns {
		out[id] = conn.State()
	}
	return out
}

// RouterState returns the connection state for a single router.
// Returns ConnStateDisconnected if the router is not registered.
func (p *Pool) RouterState(routerID string) ConnState {
	p.mu.RLock()
	conn := p.conns[routerID]
	p.mu.RUnlock()
	if conn == nil {
		return ConnStateDisconnected
	}
	return conn.State()
}

// BorrowAsync returns the persistent stream connection for the given router.
// All stream workers share this single connection via tag-multiplexing.
// Unlike the old implementation, this NEVER creates a new TCP connection —
// it returns the existing persistent stream conn.
// Callers must NOT close the returned client.
func (p *Pool) BorrowAsync(ctx context.Context, routerID string) (*PersistentConn, error) {
	p.mu.RLock()
	conn, ok := p.conns[routerID]
	p.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("pool: router %q not registered", routerID)
	}
	return conn.BorrowStream()
}

// LaunchOne starts the connect-with-backoff loop and health loop for a router
// that was registered after Start() was called. No-op if Start() has not been
// called yet (appCtx is nil).
func (p *Pool) LaunchOne(routerID string) {
	p.mu.RLock()
	appCtx := p.appCtx
	healthCtx := p.healthCtx
	conn := p.conns[routerID]
	p.mu.RUnlock()

	if appCtx == nil || conn == nil {
		return
	}
	go conn.client.ConnectWithBackoff(appCtx) //nolint:errcheck
	go conn.stream.ConnectWithBackoff(appCtx) //nolint:errcheck
	go p.healthLoop(healthCtx, conn)
}

// WaitConnected blocks until routerID reaches ConnStateConnected, or returns
// early with ErrAuthFailed if authentication permanently failed.
// Returns ctx.Err() on cancellation.
func (p *Pool) WaitConnected(ctx context.Context, routerID string) error {
	for {
		p.mu.RLock()
		conn := p.conns[routerID]
		p.mu.RUnlock()
		if conn != nil {
			switch conn.State() {
			case ConnStateConnected:
				return nil
			case ConnStateAuthFailed:
				return ErrAuthFailed
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
}

// healthLoop periodically checks a connection and reconnects if needed.
// This is a secondary safety net — PersistentConn already handles reconnect
// via its watchAsync goroutine. The health loop catches edge cases where
// the watchAsync goroutine might not have detected a problem.
func (p *Pool) healthLoop(ctx context.Context, conn *RouterConn) {
	ticker := time.NewTicker(conn.cfg.HealthInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if conn.State() == ConnStateAuthFailed {
				continue
			}
			if conn.client.State() == ConnStateConnected && !conn.client.IsAlive(ctx) {
				p.logger.Warn("health check failed on client conn, triggering reconnect",
					"router_id", conn.cfg.RouterID,
				)
				conn.client.Close()
				go conn.client.ConnectWithBackoff(ctx) //nolint:errcheck
			}
		}
	}
}