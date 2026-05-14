package execution

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/go-routeros/routeros/v3"
)

const (
	reconnectBaseDelay = 2 * time.Second
	reconnectMaxDelay  = 30 * time.Second
)

type PersistentConn struct {
	mu          sync.RWMutex
	cfg         ConnConfig
	role        ConnectionRole
	roleCfg     RoleConfig
	conn        *routeros.Client
	state       ConnState
	closed      bool
	asyncCtx    context.Context
	asyncCancel context.CancelFunc
	logger      *slog.Logger
}

func newPersistentConn(cfg ConnConfig, role ConnectionRole, logger *slog.Logger) *PersistentConn {
	roleCfg, ok := DefaultRoleConfigs[role]
	if !ok {
		roleCfg = DefaultRoleConfigs[RoleClient]
	}
	asyncCtx, asyncCancel := context.WithCancel(context.Background())
	return &PersistentConn{
		cfg:         cfg,
		role:        role,
		roleCfg:     roleCfg,
		state:       ConnStateDisconnected,
		asyncCtx:    asyncCtx,
		asyncCancel: asyncCancel,
		logger:      logger.With("router_id", cfg.RouterID, "role", string(role)),
	}
}

func (pc *PersistentConn) dial(ctx context.Context) (*routeros.Client, error) {
	dialCtx, cancel := context.WithTimeout(ctx, pc.cfg.DialTimeout)
	defer cancel()
	return routeros.DialContext(dialCtx, pc.cfg.Address, pc.cfg.Username, pc.cfg.Password)
}

func (pc *PersistentConn) Connect(ctx context.Context) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pc.closed {
		return fmt.Errorf("persistent conn %s/%s is closed", pc.cfg.RouterID, pc.role)
	}
	if pc.state == ConnStateConnected && pc.conn != nil {
		return nil
	}

	pc.state = ConnStateConnecting
	pc.logger.Info("connecting", "address", pc.cfg.Address)

	conn, err := pc.dial(ctx)
	if err != nil {
		pc.state = ConnStateDisconnected
		return fmt.Errorf("dial %s (%s): %w", pc.cfg.Address, pc.role, err)
	}

	errCh := conn.Async()
	conn.Queue = pc.roleCfg.QueueSize
	pc.conn = conn
	pc.state = ConnStateConnected
	pc.logger.Info("connected", "address", pc.cfg.Address, "queue", pc.roleCfg.QueueSize)

	go pc.watchAsync(errCh)
	return nil
}

func (pc *PersistentConn) ConnectWithBackoff(ctx context.Context) error {
	for attempt := 0; ; attempt++ {
		err := pc.Connect(ctx)
		if err == nil {
			return nil
		}

		if isAuthError(err) {
			pc.mu.Lock()
			pc.state = ConnStateAuthFailed
			pc.mu.Unlock()
			pc.logger.Error("authentication failed, not retrying",
				"address", pc.cfg.Address,
			)
			return ErrAuthFailed
		}

		shift := attempt
		if shift > 5 {
			shift = 5
		}
		delay := reconnectBaseDelay * time.Duration(1<<uint(shift))
		if delay > reconnectMaxDelay {
			delay = reconnectMaxDelay
		}

		pc.logger.Warn("connect failed, retrying",
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

func (pc *PersistentConn) getConn() (*routeros.Client, error) {
	pc.mu.RLock()
	conn := pc.conn
	pc.mu.RUnlock()
	if conn == nil {
		return nil, fmt.Errorf("not connected to %s (%s)", pc.cfg.RouterID, pc.role)
	}
	return conn, nil
}

func (pc *PersistentConn) RunContext(ctx context.Context, sentence ...string) (*routeros.Reply, error) {
	conn, err := pc.getConn()
	if err != nil {
		return nil, err
	}
	return conn.RunContext(ctx, sentence...)
}

func (pc *PersistentConn) ListenArgsQueueContext(ctx context.Context, sentence []string, queueSize int) (*routeros.ListenReply, error) {
	conn, err := pc.getConn()
	if err != nil {
		return nil, err
	}
	return conn.ListenArgsQueueContext(ctx, sentence, queueSize)
}

func (pc *PersistentConn) Client() *routeros.Client {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return pc.conn
}

func (pc *PersistentConn) State() ConnState {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return pc.state
}

func (pc *PersistentConn) watchAsync(errCh <-chan error) {
	select {
	case err := <-errCh:
		if err != nil {
			pc.logger.Warn("async loop ended", "err", err)
		}
	case <-pc.asyncCtx.Done():
		return
	}

	pc.mu.Lock()
	wasClosed := pc.closed
	if pc.conn != nil {
		pc.conn.Close()
		pc.conn = nil
	}
	pc.state = ConnStateDisconnected
	pc.mu.Unlock()

	if !wasClosed {
		go pc.reconnect()
	}
}

func (pc *PersistentConn) reconnect() {
	pc.logger.Info("reconnecting")
	err := pc.ConnectWithBackoff(pc.asyncCtx)
	if err != nil {
		pc.logger.Error("reconnect gave up", "err", err)
	}
}

func (pc *PersistentConn) Close() {
	pc.mu.Lock()
	pc.closed = true
	conn := pc.conn
	pc.conn = nil
	pc.state = ConnStateDisconnected
	pc.mu.Unlock()

	pc.asyncCancel()
	if conn != nil {
		conn.Close()
	}
	pc.logger.Info("closed")
}
