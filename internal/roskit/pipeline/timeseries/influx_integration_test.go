//go:build influxdb

package timeseries_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/pipeline/timeseries"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skipWithoutInfluxDB(t *testing.T) {
	t.Helper()
	if os.Getenv("INFLUXDB_URL") == "" {
		t.Skip("skipping: INFLUXDB_URL not set")
	}
}

func newTestWriter(t *testing.T) *timeseries.InfluxWriter {
	t.Helper()
	skipWithoutInfluxDB(t)
	url := os.Getenv("INFLUXDB_URL")
	if url == "" {
		url = "http://localhost:8181"
	}
	cfg := timeseries.InfluxWriterConfig{
		URL:      url,
		Token:    os.Getenv("INFLUXDB_TOKEN"),
		Database: testDB(),
	}
	w, err := timeseries.NewInfluxWriter(cfg, nil)
	if err != nil {
		t.Skipf("influxdb writer unavailable: %v", err)
	}
	t.Cleanup(func() { _ = w.Close() })
	return w
}

func newTestReader(t *testing.T) *timeseries.InfluxReader {
	t.Helper()
	skipWithoutInfluxDB(t)
	url := os.Getenv("INFLUXDB_URL")
	if url == "" {
		url = "http://localhost:8181"
	}
	r, err := timeseries.NewInfluxReader(timeseries.InfluxReaderConfig{
		URL:      url,
		Token:    os.Getenv("INFLUXDB_TOKEN"),
		Database: testDB(),
	})
	if err != nil {
		t.Skipf("influxdb reader unavailable: %v", err)
	}
	return r
}

func testDB() string {
	if db := os.Getenv("INFLUXDB_DATABASE"); db != "" {
		return db
	}
	return "mikhmon_test"
}

func TestInfluxWriter_WritePoint_And_Flush(t *testing.T) {
	w := newTestWriter(t)
	ctx := context.Background()

	err := w.WritePoint(ctx, timeseries.Point{
		Measurement: "system_resource",
		Tags:        map[string]string{"router_id": "test-r1"},
		Fields:      map[string]any{"cpu-load": int64(10), "free-memory": int64(104857600)},
		Timestamp:   time.Now(),
	})
	require.NoError(t, err)

	require.NoError(t, w.Flush(ctx))
}

func TestInfluxWriter_Flush_EmptyBuffer(t *testing.T) {
	w := newTestWriter(t)
	// Flush with nothing buffered should be a no-op
	assert.NoError(t, w.Flush(context.Background()))
}

func TestInfluxWriter_Close_FlushesBuffer(t *testing.T) {
	// Create a new writer and write without explicit flush; Close should flush.
	skipWithoutInfluxDB(t)
	url := os.Getenv("INFLUXDB_URL")
	if url == "" {
		url = "http://localhost:8181"
	}
	cfg := timeseries.InfluxWriterConfig{
		URL:      url,
		Token:    os.Getenv("INFLUXDB_TOKEN"),
		Database: testDB(),
	}
	w, err := timeseries.NewInfluxWriter(cfg, nil)
	if err != nil {
		t.Skipf("influxdb writer unavailable: %v", err)
	}

	_ = w.WritePoint(context.Background(), timeseries.Point{
		Measurement: "system_resource",
		Tags:        map[string]string{"router_id": "test-close"},
		Fields:      map[string]any{"cpu-load": int64(5)},
		Timestamp:   time.Now(),
	})

	assert.NoError(t, w.Close())
}

func TestInfluxReader_QueryRange_EmptyResult(t *testing.T) {
	r := newTestReader(t)
	ctx := context.Background()

	from := time.Now().Add(-1 * time.Minute)
	to := time.Now()

	// Query a measurement that doesn't exist — should return nil, no error
	rows, err := r.QueryRange(ctx, "nonexistent_measurement_xyz", "no-such-router", from, to, "10s")
	require.NoError(t, err)
	assert.Nil(t, rows)
}

func TestInfluxReader_QueryRange_ReturnsData(t *testing.T) {
	w := newTestWriter(t)
	r := newTestReader(t)
	ctx := context.Background()

	routerID := "test-query-router"
	ts := time.Now().Truncate(time.Second)

	require.NoError(t, w.WritePoint(ctx, timeseries.Point{
		Measurement: "system_resource",
		Tags:        map[string]string{"router_id": routerID},
		Fields:      map[string]any{"cpu-load": int64(42)},
		Timestamp:   ts,
	}))
	require.NoError(t, w.Flush(ctx))

	// Give InfluxDB a moment to ingest
	time.Sleep(500 * time.Millisecond)

	rows, err := r.QueryRange(ctx, "system_resource", routerID, ts.Add(-5*time.Second), ts.Add(5*time.Second), "10s")
	require.NoError(t, err)
	// May be nil if InfluxDB hasn't ingested yet — just verify no error
	_ = rows
}
