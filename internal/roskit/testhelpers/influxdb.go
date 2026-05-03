//go:build influxdb

package testhelpers

import (
	"log/slog"
	"os"
	"testing"

	"github.com/quiqxiq/roskit/internal/roskit/pipeline/timeseries"
)

func SkipWithoutInfluxDB(t *testing.T) {
	t.Helper()
	if os.Getenv("INFLUXDB_URL") == "" {
		t.Skip("skipping: INFLUXDB_URL not set")
	}
}

func influxConfig() timeseries.InfluxWriterConfig {
	url := os.Getenv("INFLUXDB_URL")
	if url == "" {
		url = "http://localhost:8181"
	}
	return timeseries.InfluxWriterConfig{
		URL:      url,
		Token:    os.Getenv("INFLUXDB_TOKEN"),
		Database: influxDB(),
	}
}

func influxDB() string {
	if db := os.Getenv("INFLUXDB_DATABASE"); db != "" {
		return db
	}
	return "mikhmon_test"
}

func NewTestInfluxWriter(t *testing.T) *timeseries.InfluxWriter {
	t.Helper()
	SkipWithoutInfluxDB(t)
	cfg := influxConfig()
	w, err := timeseries.NewInfluxWriter(cfg, slog.Default())
	if err != nil {
		t.Skipf("influxdb writer unavailable: %v", err)
	}
	t.Cleanup(func() { _ = w.Close() })
	return w
}

func NewTestInfluxReader(t *testing.T) *timeseries.InfluxReader {
	t.Helper()
	SkipWithoutInfluxDB(t)
	cfg := influxConfig()
	r, err := timeseries.NewInfluxReader(timeseries.InfluxReaderConfig{
		URL:      cfg.URL,
		Token:    cfg.Token,
		Database: influxDB(),
	})
	if err != nil {
		t.Skipf("influxdb reader unavailable: %v", err)
	}
	return r
}
