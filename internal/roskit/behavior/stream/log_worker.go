package stream

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/execution"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/pubsub"
)

var logFilters = []string{"all", "hotspot", "ppp"}

type LogWorker struct {
	pool   *execution.Pool
	pubsub pubsub.Publisher
	logger *slog.Logger
}

func NewLogWorker(pool *execution.Pool, ps pubsub.Publisher, logger *slog.Logger) *LogWorker {
	if logger == nil {
		logger = slog.Default()
	}
	return &LogWorker{pool: pool, pubsub: ps, logger: logger}
}

// Start streams /log/print follow for the given filter with exponential backoff on failure.
// filter: "all" (no topic filter), "hotspot", or "ppp" (maps to ?topics=pppoe internally).
func (w *LogWorker) Start(ctx context.Context, routerID, filter string) {
	w.logger.Info("log worker starting", "router_id", routerID, "filter", filter)

	for attempt := 0; ; attempt++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := w.runStream(ctx, routerID, filter); err != nil {
			w.logger.Debug("log stream ended, retrying",
				"router_id", routerID, "filter", filter,
				"err", err, "attempt", attempt+1)

			shift := attempt
			if shift > DefaultBackoffShiftCap {
				shift = DefaultBackoffShiftCap
			}
			delay := DefaultBackoffBase << uint(shift)
			if delay > DefaultBackoffMax {
				delay = DefaultBackoffMax
			}

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return
			}
		} else {
			return
		}
	}
}

func (w *LogWorker) runStream(ctx context.Context, routerID, filter string) error {
	conn, err := w.pool.BorrowAsync(ctx, routerID)
	if err != nil {
		return fmt.Errorf("borrow stream: %w", err)
	}

	sentence := []string{"/log/print", "=follow"}
	switch filter {
	case "hotspot":
		sentence = append(sentence, "?topics=hotspot,info,debug")
	case "ppp":
		sentence = append(sentence, "?topics=pppoe,info,debug")
	}

	reply, err := conn.ListenArgsQueueContext(ctx, sentence, 200)
	if err != nil {
		return fmt.Errorf("listen log/%s: %w", filter, err)
	}

	ch := reply.Chan()
	channel := pubsub.FormatLogChannel(routerID, filter)

	for {
		select {
		case <-ctx.Done():
			cancelCtx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
			reply.CancelContext(cancelCtx)
			cancelFn()
			return nil

		case sentence, ok := <-ch:
			if !ok {
				if err := reply.Err(); err != nil {
					return fmt.Errorf("log stream/%s closed: %w", filter, err)
				}
				return fmt.Errorf("log stream/%s channel closed", filter)
			}

			fields := sentence.Map
			if len(fields) == 0 {
				continue
			}

			w.pubsub.Publish(ctx, channel, pubsub.Message{
				Type:        "log",
				RouterID:    routerID,
				Measurement: "log",
				Fields:      fields,
				Timestamp:   time.Now(),
			})
		}
	}
}

// LogManager manages per-router log stream workers for all filters.
type LogManager struct {
	pool   *execution.Pool
	pubsub pubsub.Publisher
	logger *slog.Logger

	mu      sync.Mutex
	cancels map[string]context.CancelFunc
}

func NewLogManager(pool *execution.Pool, ps pubsub.Publisher, logger *slog.Logger) *LogManager {
	if logger == nil {
		logger = slog.Default()
	}
	return &LogManager{
		pool:    pool,
		pubsub:  ps,
		logger:  logger,
		cancels: make(map[string]context.CancelFunc),
	}
}

func (m *LogManager) StartAll(ctx context.Context, routerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	w := NewLogWorker(m.pool, m.pubsub, m.logger)
	for _, filter := range logFilters {
		key := routerID + ":" + filter
		if _, exists := m.cancels[key]; exists {
			continue
		}
		workerCtx, cancel := context.WithCancel(ctx)
		m.cancels[key] = cancel
		f := filter
		go w.Start(workerCtx, routerID, f)
	}
}

func (m *LogManager) StopAll(routerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	prefix := routerID + ":"
	for key, cancel := range m.cancels {
		if strings.HasPrefix(key, prefix) {
			cancel()
			delete(m.cancels, key)
		}
	}
}

func (m *LogManager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, cancel := range m.cancels {
		cancel()
		delete(m.cancels, key)
	}
	m.cancels = make(map[string]context.CancelFunc)
}
