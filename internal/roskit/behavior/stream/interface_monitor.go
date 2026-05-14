package stream

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/behavior"
	"github.com/quiqxiq/roskit/internal/roskit/core/command"
	"github.com/quiqxiq/roskit/internal/roskit/core/parser"
	"github.com/quiqxiq/roskit/internal/roskit/execution"
)

type InterfaceMonitorManager struct {
	pool     *execution.Pool
	sink     behavior.StreamSink
	logger   *slog.Logger

	mu       sync.Mutex
	monitors map[monitorKey]context.CancelFunc
}

type monitorKey struct {
	routerID   string
	interfaceName string
}

func NewInterfaceMonitorManager(pool *execution.Pool, sink behavior.StreamSink, logger *slog.Logger) *InterfaceMonitorManager {
	if logger == nil {
		logger = slog.Default()
	}
	return &InterfaceMonitorManager{
		pool:     pool,
		sink:     sink,
		logger:   logger,
		monitors: make(map[monitorKey]context.CancelFunc),
	}
}

func (m *InterfaceMonitorManager) SyncInterfaces(ctx context.Context, routerID string, interfaceNames []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	activeSet := make(map[string]bool, len(interfaceNames))
	for _, name := range interfaceNames {
		activeSet[name] = true
	}

	for key, cancel := range m.monitors {
		if key.routerID == routerID && !activeSet[key.interfaceName] {
			cancel()
			delete(m.monitors, key)
			m.logger.Info("interface monitor stopped",
				"router_id", routerID, "interface", key.interfaceName)
		}
	}

	meta := &command.CommandMeta{
		Type:            command.CommandTypeStream,
		Path:            "interface/monitor-traffic",
		Measurement:     "interface_traffic",
		WriteTimeSeries: true,
	}

	for _, name := range interfaceNames {
		key := monitorKey{routerID: routerID, interfaceName: name}
		if _, exists := m.monitors[key]; exists {
			continue
		}

		monitorCtx, cancel := context.WithCancel(ctx)
		m.monitors[key] = cancel

		go m.runMonitor(monitorCtx, routerID, name, meta)
		m.logger.Info("interface monitor started",
			"router_id", routerID, "interface", name)
	}
}

func (m *InterfaceMonitorManager) StopAll(routerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, cancel := range m.monitors {
		if key.routerID == routerID {
			cancel()
			delete(m.monitors, key)
		}
	}
}

func (m *InterfaceMonitorManager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, cancel := range m.monitors {
		cancel()
	}
	m.monitors = make(map[monitorKey]context.CancelFunc)
}

func (m *InterfaceMonitorManager) ActiveCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.monitors)
}

func (m *InterfaceMonitorManager) runMonitor(ctx context.Context, routerID, ifaceName string, meta *command.CommandMeta) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := m.acquireAndStream(ctx, routerID, ifaceName, meta); err != nil {
			m.logger.Debug("interface monitor stream ended",
				"router_id", routerID, "interface", ifaceName, "err", err)

			select {
			case <-time.After(5 * time.Second):
			case <-ctx.Done():
				return
			}
		} else {
			return
		}
	}
}

func (m *InterfaceMonitorManager) acquireAndStream(ctx context.Context, routerID, ifaceName string, meta *command.CommandMeta) error {
	streamConn, err := m.pool.BorrowAsync(ctx, routerID)
	if err != nil {
		return fmt.Errorf("borrow stream: %w", err)
	}

	return m.runStream(ctx, routerID, ifaceName, meta, streamConn)
}

func (m *InterfaceMonitorManager) runStream(ctx context.Context, routerID, ifaceName string, meta *command.CommandMeta, streamConn *execution.PersistentConn) error {
	sentence := command.BuildMonitorSentence(meta, map[string]string{
		"interface": ifaceName,
	})

	reply, err := streamConn.ListenArgsQueueContext(ctx, sentence, 100)
	if err != nil {
		return fmt.Errorf("listen interface_traffic/%s: %w", ifaceName, err)
	}

	ch := reply.Chan()
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
					return fmt.Errorf("interface_traffic/%s closed: %w", ifaceName, err)
				}
				return fmt.Errorf("interface_traffic/%s channel closed", ifaceName)
			}

			pairs := sentence.Map
			if len(pairs) == 0 {
				continue
			}

			result := parser.ParseStreamSentence(routerID, meta, pairs)
			if result == nil {
				continue
			}

			fields := result.CacheData
			if fields == nil {
				fields = pairs
			}

			if fields["name"] == "" {
				fields["name"] = ifaceName
			}

			event := behavior.StreamEvent{
				Meta:       meta,
				RouterID:   routerID,
				Fields:     fields,
				IsDead:     false,
				ReceivedAt: time.Now(),
			}

			if err := m.sink.OnEvent(ctx, event); err != nil {
				m.logger.Warn("interface monitor: sink error",
					"interface", ifaceName, "err", err)
			}
		}
	}
}
