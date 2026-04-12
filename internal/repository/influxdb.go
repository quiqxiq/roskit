package repository

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
)

// InfluxDBConfig holds configuration for the InfluxDB 3 Core connection.
type InfluxDBConfig struct {
	// Host is the InfluxDB server URL (e.g., "http://localhost:8181").
	Host string
	// Token is the authentication token. Empty for no-auth dev setups.
	Token string
	// Database is the target database name.
	Database string
	// BatchSize is the number of points to buffer before flushing (default: 5000).
	BatchSize int
	// FlushInterval is how often to flush buffered points (default: 1s).
	FlushInterval time.Duration
}

// influxPoint is an internal representation of a buffered data point.
type influxPoint struct {
	measurement string
	tags        map[string]string
	fields      map[string]interface{}
	timestamp   time.Time
}

// InfluxDBWriter implements TimeSeriesWriter using InfluxDB 3 Core.
// It provides an async batching layer on top of the synchronous InfluxDB 3 client
// for high-throughput telemetry ingestion.
type InfluxDBWriter struct {
	client *influxdb3.Client
	config InfluxDBConfig
	logger *slog.Logger

	// Batching internals
	buffer []influxPoint
	mu     sync.Mutex
	done   chan struct{}
	wg     sync.WaitGroup
}

// NewInfluxDBWriter creates a new InfluxDB 3 Core writer with async batching.
func NewInfluxDBWriter(cfg InfluxDBConfig, logger *slog.Logger) (*InfluxDBWriter, error) {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 5000
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = time.Second
	}

	client, err := influxdb3.New(influxdb3.ClientConfig{
		Host:     cfg.Host,
		Token:    cfg.Token,
		Database: cfg.Database,
	})
	if err != nil {
		return nil, fmt.Errorf("influxdb3: failed to create client: %w", err)
	}

	w := &InfluxDBWriter{
		client: client,
		config: cfg,
		logger: logger,
		buffer: make([]influxPoint, 0, cfg.BatchSize),
		done:   make(chan struct{}),
	}

	// Start the background flush goroutine.
	w.wg.Add(1)
	go w.flushLoop()

	logger.Info("InfluxDB writer initialized",
		"host", cfg.Host,
		"database", cfg.Database,
		"batch_size", cfg.BatchSize,
		"flush_interval", cfg.FlushInterval,
	)

	return w, nil
}

// WritePoint buffers a data point for async batch writing.
// Points are flushed when the buffer reaches BatchSize or FlushInterval expires.
func (w *InfluxDBWriter) WritePoint(_ context.Context, measurement string, tags map[string]string, fields map[string]interface{}, ts time.Time) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Sort tags lexicographically for InfluxDB indexing performance.
	w.buffer = append(w.buffer, influxPoint{
		measurement: measurement,
		tags:        tags,
		fields:      fields,
		timestamp:   ts,
	})

	// Flush if buffer is full.
	if len(w.buffer) >= w.config.BatchSize {
		return w.flushLocked()
	}

	return nil
}

// Flush forces all buffered points to be written to InfluxDB immediately.
func (w *InfluxDBWriter) Flush(_ context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.flushLocked()
}

// Close stops the background flush goroutine, flushes remaining points, and closes the client.
func (w *InfluxDBWriter) Close() error {
	close(w.done)
	w.wg.Wait()

	// Final flush.
	w.mu.Lock()
	if err := w.flushLocked(); err != nil {
		w.logger.Error("final flush failed", "error", err)
	}
	w.mu.Unlock()

	return w.client.Close()
}

// flushLoop runs in the background and periodically flushes the buffer.
func (w *InfluxDBWriter) flushLoop() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.mu.Lock()
			if err := w.flushLocked(); err != nil {
				w.logger.Error("periodic flush failed", "error", err)
			}
			w.mu.Unlock()
		case <-w.done:
			return
		}
	}
}

// flushLocked writes all buffered points to InfluxDB. Must be called while holding w.mu.
func (w *InfluxDBWriter) flushLocked() error {
	if len(w.buffer) == 0 {
		return nil
	}

	points := make([]*influxdb3.Point, 0, len(w.buffer))
	for _, p := range w.buffer {
		point := influxdb3.NewPoint(p.measurement, p.tags, p.fields, p.timestamp)
		points = append(points, point)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := w.client.WritePoints(ctx, points); err != nil {
		w.logger.Error("failed to write points to InfluxDB",
			"count", len(points),
			"error", err,
		)
		return fmt.Errorf("influxdb3: write failed: %w", err)
	}

	w.logger.Debug("flushed points to InfluxDB", "count", len(points))

	// Reset buffer, reusing underlying array.
	w.buffer = w.buffer[:0]
	return nil
}
