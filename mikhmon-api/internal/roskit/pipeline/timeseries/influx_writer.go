package timeseries

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	defaultBatchSize    = 500
	defaultFlushInterval = 5 * time.Second
)

type InfluxWriterConfig struct {
	URL      string
	Token    string
	Database string
}

type InfluxWriter struct {
	cfg    InfluxWriterConfig
	client *http.Client
	logger *slog.Logger

	mu      sync.Mutex
	buf     bytes.Buffer
	flushCh chan struct{}
	done    chan struct{}
}

func NewInfluxWriter(cfg InfluxWriterConfig, logger *slog.Logger) (*InfluxWriter, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.Database == "" {
		cfg.Database = "mikhmon"
	}
	w := &InfluxWriter{
		cfg:     cfg,
		client:  &http.Client{Timeout: 10 * time.Second},
		logger:  logger,
		flushCh: make(chan struct{}, 1),
		done:    make(chan struct{}),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/health", cfg.URL)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("influxdb writer: health check %s: %w", cfg.URL, err)
	}
	resp.Body.Close()

	logger.Info("InfluxDB writer initialized", "url", cfg.URL, "database", cfg.Database)

	go w.backgroundFlush()
	return w, nil
}

func (w *InfluxWriter) WritePoint(_ context.Context, point Point) error {
	line := w.formatLineProtocol(point)
	if line == "" {
		return nil
	}
	w.mu.Lock()
	w.buf.WriteString(line)
	w.buf.WriteByte('\n')
	shouldFlush := w.buf.Len() >= defaultBatchSize
	w.mu.Unlock()

	if shouldFlush {
		select {
		case w.flushCh <- struct{}{}:
		default:
		}
	}
	return nil
}

func (w *InfluxWriter) Flush(ctx context.Context) error {
	w.mu.Lock()
	if w.buf.Len() == 0 {
		w.mu.Unlock()
		return nil
	}
	data := make([]byte, w.buf.Len())
	copy(data, w.buf.Bytes())
	w.buf.Reset()
	w.mu.Unlock()

	return w.write(ctx, data)
}

func (w *InfluxWriter) Close() error {
	close(w.done)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = w.Flush(ctx)
	return nil
}

var reservedFields = map[string]bool{
	"time": true,
}

func (w *InfluxWriter) formatLineProtocol(p Point) string {
	var sb strings.Builder
	sb.WriteString(escapeMeasurement(p.Measurement))

	tagSet := make(map[string]bool, len(p.Tags))
	for k, v := range p.Tags {
		tagSet[k] = true
		sb.WriteByte(',')
		sb.WriteString(escapeTag(k))
		sb.WriteByte('=')
		sb.WriteString(escapeTag(v))
	}

	first := true
	for k, v := range p.Fields {
		if tagSet[k] || reservedFields[k] {
			continue
		}
		if first {
			sb.WriteByte(' ')
			first = false
		} else {
			sb.WriteByte(',')
		}
		sb.WriteString(escapeTag(k))
		sb.WriteByte('=')
		sb.WriteString(formatValue(v))
	}

	if first {
		return ""
	}

	if !p.Timestamp.IsZero() {
		sb.WriteByte(' ')
		sb.WriteString(fmt.Sprintf("%d", p.Timestamp.UnixNano()))
	}

	return sb.String()
}

func (w *InfluxWriter) write(ctx context.Context, data []byte) error {
	url := fmt.Sprintf("%s/api/v3/write_lp?db=%s&precision=ns", w.cfg.URL, w.cfg.Database)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("influxdb writer: create request: %w", err)
	}
	if w.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+w.cfg.Token)
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("influxdb writer: write failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		w.logger.Warn("influxdb writer: write rejected",
			"status", resp.StatusCode, "body", string(body))
		return fmt.Errorf("influxdb writer: status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (w *InfluxWriter) backgroundFlush() {
	ticker := time.NewTicker(defaultFlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = w.Flush(context.Background())
		case <-w.flushCh:
			_ = w.Flush(context.Background())
		case <-w.done:
			return
		}
	}
}

func escapeMeasurement(s string) string {
	return strings.ReplaceAll(s, " ", "\\ ")
}

func escapeTag(s string) string {
	s = strings.ReplaceAll(s, " ", "\\ ")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "=", "\\=")
	return s
}

func formatValue(v interface{}) string {
	switch val := v.(type) {
	case float64:
		return fmt.Sprintf("%g", val)
	case int64:
		return fmt.Sprintf("%di", val)
	case int:
		return fmt.Sprintf("%di", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case string:
		return fmt.Sprintf("%q", val)
	default:
		return fmt.Sprintf("%q", fmt.Sprintf("%v", v))
	}
}

var _ Writer = (*InfluxWriter)(nil)
