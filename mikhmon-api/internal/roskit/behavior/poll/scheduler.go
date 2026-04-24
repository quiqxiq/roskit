package poll

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

type Scheduler struct {
	pool     *execution.Pool
	sink     behavior.PollSink
	logger   *slog.Logger
	workers  map[workerKey]context.CancelFunc
	mu       sync.RWMutex
}

func NewScheduler(pool *execution.Pool, sink behavior.PollSink, logger *slog.Logger) *Scheduler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Scheduler{
		pool:    pool,
		sink:    sink,
		logger:  logger,
		workers: make(map[workerKey]context.CancelFunc),
	}
}

func (s *Scheduler) Start(ctx context.Context, routerID string, meta *command.CommandMeta) {
	key := workerKey{routerID: routerID, measurement: meta.Measurement}

	s.mu.Lock()
	if _, exists := s.workers[key]; exists {
		s.mu.Unlock()
		return
	}

	workerCtx, cancel := context.WithCancel(ctx)
	s.workers[key] = cancel
	s.mu.Unlock()

	w := NewWorker(s.pool, s.sink, s.logger)
	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.workers, key)
			s.mu.Unlock()
		}()
		if err := w.Start(workerCtx, routerID, meta); err != nil {
			s.logger.Error("poll worker exited",
				"router_id", routerID,
				"measurement", meta.Measurement,
				"error", err,
			)
		}
	}()
}

func (s *Scheduler) StopAll(routerID string) {
	s.mu.Lock()
	for key, cancel := range s.workers {
		if key.routerID == routerID {
			cancel()
			delete(s.workers, key)
		}
	}
	s.mu.Unlock()
}

func (s *Scheduler) Shutdown() {
	s.mu.Lock()
	for key, cancel := range s.workers {
		cancel()
		delete(s.workers, key)
	}
	s.mu.Unlock()
}

func (s *Scheduler) ActiveCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.workers)
}
