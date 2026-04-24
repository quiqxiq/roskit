// Package execution manages RouterOS TCP connections.
// It is a pure I/O layer — no knowledge of behaviors, specs, or pipeline.
//
// Dependency rule: execution/ → nothing internal (only go-routeros + stdlib)
package execution

import (
	"context"
	"fmt"
	"log/slog"
	"math"
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

// ConnState represents the lifecycle state of a router connection.
type ConnState uint8

const (
	ConnStateDisconnected ConnState = iota
	ConnStateConnecting
	ConnStateConnected
)

func (s ConnState) String() string {
	switch s {
	case ConnStateConnected:
		return "connected"
	case ConnStateConnecting:
		return "connecting"
	default:
		return "disconnected"
	}
}

// RouterConn is a managed connection to one MikroTik router.
// It holds the underlying go-routeros client and manages its lifecycle.
type RouterConn struct {
	mu     sync.RWMutex
	cfg    ConnConfig
	client *routeros.Client
	state  ConnState
	logger *slog.Logger
}

func newRouterConn(cfg ConnConfig, logger *slog.Logger) *RouterConn {
	return &RouterConn{
		cfg:    cfg.withDefaults(),
		state:  ConnStateDisconnected,
		logger: logger.With("router_id", cfg.RouterID),
	}
}

// Connect dials and authenticates. Idempotent — noop if already connected.
func (rc *RouterConn) Connect(ctx context.Context) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.state == ConnStateConnected {
		return nil
	}

	rc.state = ConnStateConnecting
	rc.logger.Info("connecting", "address", rc.cfg.Address)

	client, err := routeros.DialContext(ctx, rc.cfg.Address, rc.cfg.Username, rc.cfg.Password)
	if err != nil {
		rc.state = ConnStateDisconnected
		return fmt.Errorf("dial %s: %w", rc.cfg.Address, err)
	}

	rc.client = client
	rc.state = ConnStateConnected
	rc.logger.Info("connected", "address", rc.cfg.Address)
	return nil
}

// ConnectWithBackoff retries connection with exponential backoff until ctx is cancelled.
func (rc *RouterConn) ConnectWithBackoff(ctx context.Context) error {
	const base = 2 * time.Second
	const maxDelay = 60 * time.Second

	for attempt := 0; ; attempt++ {
		err := rc.Connect(ctx)
		if err == nil {
			return nil
		}

		delay := time.Duration(float64(base) * math.Pow(2, float64(attempt)))
		if delay > maxDelay {
			delay = maxDelay
		}

		rc.logger.Warn("connect failed, retrying",
			"attempt", attempt+1,
			"delay", delay,
			"err", err,
		)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
}

// Close closes the underlying connection. Idempotent.
func (rc *RouterConn) Close() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	if rc.client != nil {
		rc.client.Close()
		rc.client = nil
	}
	rc.state = ConnStateDisconnected
}

// IsAlive sends a lightweight command to check connection health.
func (rc *RouterConn) IsAlive(ctx context.Context) bool {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	if rc.state != ConnStateConnected || rc.client == nil {
		return false
	}
	_, err := rc.client.RunContext(ctx, "/system/identity/print")
	return err == nil
}

// RunContext executes a synchronous command. Caller must hold pool borrow.
func (rc *RouterConn) RunContext(ctx context.Context, sentence ...string) (*routeros.Reply, error) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	if rc.state != ConnStateConnected || rc.client == nil {
		return nil, fmt.Errorf("not connected to %s", rc.cfg.RouterID)
	}
	return rc.client.RunContext(ctx, sentence...)
}

func (rc *RouterConn) Client() *routeros.Client {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.client
}

func (rc *RouterConn) AcquireAsync() (*routeros.Client, error) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	if rc.state != ConnStateConnected || rc.client == nil {
		return nil, fmt.Errorf("not connected to %s", rc.cfg.RouterID)
	}
	rc.client.Async()
	return rc.client, nil
}

func (rc *RouterConn) Release() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	if rc.client != nil {
		rc.client.Close()
		rc.client = nil
	}
	rc.state = ConnStateDisconnected
}

// State returns the current connection state (safe for concurrent access).
func (rc *RouterConn) State() ConnState {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.state
}

// =============================================================================
// Pool — manages multiple RouterConn instances
// =============================================================================

// Pool manages connections to multiple routers.
// Stream workers and poll workers borrow connections from the pool.
//
// Design decision: each RouterConn wraps a SINGLE go-routeros client.
// The RouterOS API protocol allows concurrent commands on one TCP connection
// via tag-multiplexing, but streaming (=follow) commands block that connection's
// tag slot. Therefore:
//   - Poll/query commands: share the main connection via RunContext
//   - Stream commands: get a DEDICATED connection via BorrowAsync
//
// TODO: for production scale (many concurrent streams), implement a
// secondary connection per RouterConn for streaming.
type Pool struct {
	mu      sync.RWMutex
	conns   map[string]*RouterConn
	logger  *slog.Logger

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
	p.healthCancel = cancel
	p.mu.Unlock()

	for _, conn := range conns {
		go conn.ConnectWithBackoff(ctx) //nolint:errcheck
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
//
// Note: current implementation returns the shared conn. For streaming,
// this means stream workers will block the sync command channel.
// A production improvement would be a separate async conn per worker.
func (p *Pool) Borrow(ctx context.Context, routerID string) (*RouterConn, error) {
	p.mu.RLock()
	conn, ok := p.conns[routerID]
	p.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("pool: router %q not registered", routerID)
	}
	if conn.State() != ConnStateConnected {
		return nil, fmt.Errorf("pool: router %q is %s", routerID, conn.State())
	}
	return conn, nil
}

// Return releases a borrowed connection back to the pool.
// In the current single-conn model this is a noop — but kept for API stability
// when we switch to a multi-conn pool per router.
func (p *Pool) Return(routerID string, conn *RouterConn) {
	// noop for now
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

func (p *Pool) DialSync(ctx context.Context, routerID string) (*routeros.Client, error) {
	p.mu.RLock()
	conn, ok := p.conns[routerID]
	p.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("pool: router %q not registered", routerID)
	}
	cfg := conn.cfg
	client, err := routeros.DialContext(ctx, cfg.Address, cfg.Username, cfg.Password)
	if err != nil {
		return nil, fmt.Errorf("pool dial %s: %w", routerID, err)
	}
	return client, nil
}

// healthLoop periodically checks a connection and reconnects if needed.
func (p *Pool) healthLoop(ctx context.Context, conn *RouterConn) {
	ticker := time.NewTicker(conn.cfg.HealthInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !conn.IsAlive(ctx) {
				p.logger.Warn("health check failed, reconnecting",
					"router_id", conn.cfg.RouterID,
				)
				conn.Close()
				go conn.ConnectWithBackoff(ctx) //nolint:errcheck
			}
		}
	}
}