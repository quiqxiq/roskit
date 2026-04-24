package timeseries

import (
	"context"
	"time"
)

type Reader interface {
	QueryRange(ctx context.Context, measurement, routerID string, from, to time.Time, step string) ([]map[string]interface{}, error)
}

type NoopReader struct{}

func (NoopReader) QueryRange(_ context.Context, _, _ string, _, _ time.Time, _ string) ([]map[string]interface{}, error) {
	return nil, nil
}

var _ Reader = NoopReader{}
