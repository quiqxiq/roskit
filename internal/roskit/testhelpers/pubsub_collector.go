//go:build mikrotik

package testhelpers

import (
	"context"
	"sync"

	"github.com/quiqxiq/roskit/internal/roskit/behavior"
)

type MemStreamSink struct {
	mu     sync.Mutex
	Events []behavior.StreamEvent
	Ch     chan behavior.StreamEvent
}

func NewMemStreamSink() *MemStreamSink {
	return &MemStreamSink{Ch: make(chan behavior.StreamEvent, 200)}
}

func (m *MemStreamSink) OnEvent(_ context.Context, event behavior.StreamEvent) error {
	m.mu.Lock()
	m.Events = append(m.Events, event)
	m.mu.Unlock()
	select {
	case m.Ch <- event:
	default:
	}
	return nil
}

var _ behavior.StreamSink = (*MemStreamSink)(nil)
