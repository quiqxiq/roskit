package stream

import (
	"context"
	"log/slog"

	"github.com/quiqxiq/roskit/internal/roskit/behavior"
)

type MultiSink struct {
	sinks  []behavior.StreamSink
	logger *slog.Logger
}

func NewMultiSink(sinks []behavior.StreamSink, logger *slog.Logger) *MultiSink {
	if logger == nil {
		logger = slog.Default()
	}
	return &MultiSink{sinks: sinks, logger: logger}
}

func (m *MultiSink) OnEvent(ctx context.Context, event behavior.StreamEvent) error {
	for _, s := range m.sinks {
		if err := s.OnEvent(ctx, event); err != nil {
			m.logger.Warn("multi_sink: sink error",
				"measurement", event.Meta.Measurement,
				"err", err)
		}
	}
	return nil
}

var _ behavior.StreamSink = (*MultiSink)(nil)
