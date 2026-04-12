package collector

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/go-routeros/routeros/v3"
	"github.com/quiqxiq/roskit/internal/domain"
)

// ConnState represents the current state of a router connection.
type ConnState int

const (
	// ConnStateDisconnected means the connection is not established.
	ConnStateDisconnected ConnState = iota
	// ConnStateConnecting means a dial is in progress.
	ConnStateConnecting
	// ConnStateConnected means the connection is alive and usable.
	ConnStateConnected
)

// String returns a human-readable representation of the connection state.
func (s ConnState) String() string {
	switch s {
	case ConnStateDisconnected:
		return "disconnected"
	case ConnStateConnecting:
		return "connecting"
	case ConnStateConnected:
		return "connected"
	default:
		return "unknown"
	}
}

// RouterConn manages a single persistent connection to a MikroTik router.
// It handles the connection lifecycle including:
//   - Initial dial with authentication
//   - Health monitoring via /system/identity/print ping
//   - Automatic reconnection with per-router configurable backoff
//   - Thread-safe client access for multiple consumers
type RouterConn struct {
	config domain.RouterConfig
	logger *slog.Logger

	client   *routeros.Client
	state    ConnState
	mu       sync.RWMutex
	lastDrop time.Time // when the connection last dropped

	// reconnect state
	reconnectDelay time.Duration
}

// newRouterConn creates a new managed connection for a router.
func newRouterConn(config domain.RouterConfig, logger *slog.Logger) *RouterConn {
	// Apply defaults to any zero-valued config fields.
	config.Defaults()

	return &RouterConn{
		config:         config,
		state:          ConnStateDisconnected,
		reconnectDelay: config.ReconnectInterval,
		logger: logger.With(
			"router", config.ID,
			"addr", config.Address,
		),
	}
}

// Connect establishes the TCP connection and authenticates.
// If already connected, this is a no-op.
func (rc *RouterConn) Connect(ctx context.Context) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.state == ConnStateConnected && rc.client != nil {
		return nil // Already connected.
	}

	rc.state = ConnStateConnecting
	rc.logger.Info("connecting to router",
		"dial_timeout", rc.config.DialTimeout,
	)

	dialCtx, cancel := context.WithTimeout(ctx, rc.config.DialTimeout)
	defer cancel()

	client, err := rc.dial(dialCtx)
	if err != nil {
		rc.state = ConnStateDisconnected
		return fmt.Errorf("connect %s: %w", rc.config.ID, err)
	}

	rc.client = client
	rc.state = ConnStateConnected
	rc.reconnectDelay = rc.config.ReconnectInterval // Reset backoff on successful connect.

	rc.logger.Info("connection established")

	return nil
}

// ReconnectWithBackoff attempts to reconnect using exponential backoff
// based on the per-router ReconnectInterval and MaxReconnectInterval.
// It blocks until connected or the context is cancelled.
func (rc *RouterConn) ReconnectWithBackoff(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := rc.Connect(ctx)
		if err == nil {
			return nil // Successfully reconnected.
		}

		rc.mu.Lock()
		delay := rc.reconnectDelay
		// Exponential backoff: double the delay, capped at MaxReconnectInterval.
		rc.reconnectDelay = min(rc.reconnectDelay*2, rc.config.MaxReconnectInterval)
		rc.mu.Unlock()

		rc.logger.Warn("reconnection failed, retrying",
			"error", err,
			"retry_in", delay,
			"max_interval", rc.config.MaxReconnectInterval,
		)

		select {
		case <-time.After(delay):
			// Continue to next attempt.
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// AcquireAsync returns the client in Async mode for streaming.
// The caller gets exclusive ownership of the client for the duration of the streaming session.
// When the session ends (specs stop), the caller should call Release() to
// close the async client cleanly so the pool can establish a fresh connection.
func (rc *RouterConn) AcquireAsync() (*routeros.Client, error) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.state != ConnStateConnected || rc.client == nil {
		return nil, fmt.Errorf("router %s: not connected (state=%s)", rc.config.ID, rc.state)
	}

	client := rc.client
	client.Async()

	rc.logger.Debug("async client acquired")
	return client, nil
}

// Release closes the current connection and marks it as disconnected.
// Called by the collector after a streaming session ends (either cleanly or due to error).
// The pool will then reconnect automatically on the next health check cycle.
func (rc *RouterConn) Release() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.client != nil {
		rc.client.Close()
		rc.client = nil
	}
	rc.state = ConnStateDisconnected
	rc.lastDrop = time.Now()
	rc.logger.Debug("connection released")
}

// IsAlive checks if the connection is still responsive by sending a
// lightweight /system/identity/print command.
func (rc *RouterConn) IsAlive(ctx context.Context) bool {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	if rc.state != ConnStateConnected || rc.client == nil {
		return false
	}

	// Send a lightweight ping command.
	cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := rc.client.RunContext(cmdCtx, "/system/identity/print")
	if err != nil {
		rc.logger.Warn("health check failed", "error", err)
		return false
	}

	return true
}

// State returns the current connection state.
func (rc *RouterConn) State() ConnState {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.state
}

// Config returns the router configuration.
func (rc *RouterConn) Config() domain.RouterConfig {
	return rc.config
}

// Close permanently closes the connection.
func (rc *RouterConn) Close() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.client != nil {
		rc.client.Close()
		rc.client = nil
	}
	rc.state = ConnStateDisconnected
}

// dial creates a raw connection to the router.
func (rc *RouterConn) dial(ctx context.Context) (*routeros.Client, error) {
	if rc.config.UseTLS {
		return routeros.DialTLSContext(
			ctx,
			rc.config.Address,
			rc.config.Username,
			rc.config.Password,
			&tls.Config{InsecureSkipVerify: true}, //nolint:gosec // Router self-signed certs.
		)
	}
	return routeros.DialContext(ctx, rc.config.Address, rc.config.Username, rc.config.Password)
}

// ─── Connection Pool ─────────────────────────────────────────────────────────

// ConnPool manages persistent connections for multiple MikroTik routers.
// Each router gets exactly one dedicated connection that is:
//   - Established on pool start
//   - Monitored with per-router health check intervals
//   - Automatically reconnected with per-router backoff settings
//   - Closed on pool shutdown
//
// The pool is the single source of truth for connection state across the system.
type ConnPool struct {
	connections map[string]*RouterConn
	mu          sync.RWMutex
	logger      *slog.Logger

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewConnPool creates a new connection pool.
func NewConnPool(logger *slog.Logger) *ConnPool {
	return &ConnPool{
		connections: make(map[string]*RouterConn),
		logger:      logger.With("component", "connpool"),
	}
}

// Register adds a router to the connection pool.
// The connection is not established until Start() is called.
func (p *ConnPool) Register(config domain.RouterConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.connections[config.ID]; exists {
		p.logger.Warn("router already registered in pool, replacing", "router", config.ID)
		p.connections[config.ID].Close()
	}

	conn := newRouterConn(config, p.logger)
	p.connections[config.ID] = conn

	p.logger.Info("router registered in pool",
		"router", config.ID,
		"address", config.Address,
		"reconnect_interval", config.ReconnectInterval,
		"max_reconnect_interval", config.MaxReconnectInterval,
		"health_check_interval", config.HealthCheckInterval,
		"dial_timeout", config.DialTimeout,
	)
}

// Unregister removes a router from the pool and closes its connection.
func (p *ConnPool) Unregister(routerID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if conn, exists := p.connections[routerID]; exists {
		conn.Close()
		delete(p.connections, routerID)
		p.logger.Info("router unregistered from pool", "router", routerID)
	}
}

// Get returns the managed connection for a specific router.
// Returns nil if the router is not registered.
func (p *ConnPool) Get(routerID string) *RouterConn {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.connections[routerID]
}

// Start connects to all registered routers and starts per-router health monitors.
func (p *ConnPool) Start(ctx context.Context) {
	var poolCtx context.Context
	poolCtx, p.cancel = context.WithCancel(ctx)

	p.mu.RLock()
	routers := make([]*RouterConn, 0, len(p.connections))
	for _, conn := range p.connections {
		routers = append(routers, conn)
	}
	p.mu.RUnlock()

	p.logger.Info("starting connection pool", "routers", len(routers))

	// Connect to all routers concurrently.
	var connectWg sync.WaitGroup
	for _, conn := range routers {
		connectWg.Add(1)
		go func(conn *RouterConn) {
			defer connectWg.Done()
			if err := conn.Connect(poolCtx); err != nil {
				p.logger.Error("initial connection failed",
					"router", conn.config.ID,
					"error", err,
				)
			}
		}(conn)
	}
	connectWg.Wait()

	// Start a per-router health monitor goroutine.
	// Each router has its own health check interval from its config.
	for _, conn := range routers {
		p.wg.Add(1)
		go p.healthLoop(poolCtx, conn)
	}
}

// Stop closes all connections and stops all health monitors.
func (p *ConnPool) Stop() {
	p.logger.Info("stopping connection pool")

	if p.cancel != nil {
		p.cancel()
	}

	// Wait for all health loops to exit.
	p.wg.Wait()

	// Close all connections.
	p.mu.RLock()
	for _, conn := range p.connections {
		conn.Close()
	}
	p.mu.RUnlock()

	p.logger.Info("connection pool stopped")
}

// Status returns a snapshot of all router connection states.
func (p *ConnPool) Status() map[string]ConnState {
	p.mu.RLock()
	defer p.mu.RUnlock()

	status := make(map[string]ConnState, len(p.connections))
	for id, conn := range p.connections {
		status[id] = conn.State()
	}
	return status
}

// healthLoop runs per-router, periodically checks connection liveness,
// and reconnects using the router's specific backoff configuration.
func (p *ConnPool) healthLoop(ctx context.Context, conn *RouterConn) {
	defer p.wg.Done()

	ticker := time.NewTicker(conn.config.HealthCheckInterval)
	defer ticker.Stop()

	p.logger.Info("health monitor started",
		"router", conn.config.ID,
		"interval", conn.config.HealthCheckInterval,
	)

	for {
		select {
		case <-ctx.Done():
			p.logger.Debug("health monitor stopped", "router", conn.config.ID)
			return
		case <-ticker.C:
			p.checkConnection(ctx, conn)
		}
	}
}

// checkConnection verifies a single connection and reconnects if dead.
func (p *ConnPool) checkConnection(ctx context.Context, conn *RouterConn) {
	state := conn.State()

	switch state {
	case ConnStateDisconnected:
		p.logger.Info("reconnecting disconnected router",
			"router", conn.config.ID,
			"reconnect_interval", conn.config.ReconnectInterval,
		)

		if err := conn.Connect(ctx); err != nil {
			p.logger.Error("reconnection failed",
				"router", conn.config.ID,
				"error", err,
			)
		}

	case ConnStateConnected:
		// Verify the connection is actually alive.
		if !conn.IsAlive(ctx) {
			p.logger.Warn("connection lost during health check, closing",
				"router", conn.config.ID,
			)
			conn.Release()
			// Will be reconnected on the next health check cycle.
		} else {
			p.logger.Debug("health check passed", "router", conn.config.ID)
		}
	}
}
