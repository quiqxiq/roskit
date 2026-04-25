package stream

import (
	"context"
	"log/slog"
	"sync"

	"github.com/quiqxiq/roskit/internal/roskit/behavior"
	"github.com/quiqxiq/roskit/internal/roskit/core/command"
	"github.com/quiqxiq/roskit/internal/roskit/execution"
)

type workerKey struct {
	routerID    string
	measurement string
}

type Manager struct {
	pool     *execution.Pool
	sink     behavior.StreamSink
	logger   *slog.Logger
	workers  map[workerKey]context.CancelFunc
	mu       sync.RWMutex
}

func NewManager(pool *execution.Pool, sink behavior.StreamSink, logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	return &Manager{
		pool:    pool,
		sink:    sink,
		logger:  logger,
		workers: make(map[workerKey]context.CancelFunc),
	}
}

func (m *Manager) Start(ctx context.Context, routerID string, meta *command.CommandMeta) {
	key := workerKey{routerID: routerID, measurement: meta.Measurement}

	m.mu.Lock()
	if _, exists := m.workers[key]; exists {
		m.mu.Unlock()
		return
	}

	workerCtx, cancel := context.WithCancel(ctx)
	m.workers[key] = cancel
	m.mu.Unlock()

	w := NewWorker(m.pool, m.sink, m.logger)
	go func() {
		defer func() {
			m.mu.Lock()
			delete(m.workers, key)
			m.mu.Unlock()
		}()
		if err := w.Start(workerCtx, routerID, meta); err != nil {
			m.logger.Error("stream worker exited with error",
				"router_id", routerID,
				"measurement", meta.Measurement,
				"error", err,
			)
		}
	}()
}

func (m *Manager) Stop(routerID, measurement string) {
	key := workerKey{routerID: routerID, measurement: measurement}
	m.mu.Lock()
	if cancel, ok := m.workers[key]; ok {
		cancel()
		delete(m.workers, key)
	}
	m.mu.Unlock()
}

func (m *Manager) StopAll(routerID string) {
	m.mu.Lock()
	for key, cancel := range m.workers {
		if key.routerID == routerID {
			cancel()
			delete(m.workers, key)
		}
	}
	m.mu.Unlock()
}

func (m *Manager) Shutdown() {
	m.mu.Lock()
	for key, cancel := range m.workers {
		cancel()
		delete(m.workers, key)
	}
	m.mu.Unlock()
}

func (m *Manager) ActiveCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.workers)
}

type PollManager struct{}

func (p *PollManager) Start(_ context.Context, _ string, _ *command.CommandMeta) {}
