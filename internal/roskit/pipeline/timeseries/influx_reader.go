package timeseries

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type InfluxReaderConfig struct {
	URL      string
	Token    string
	Database string
}

type InfluxReader struct {
	cfg    InfluxReaderConfig
	client *http.Client
}

func NewInfluxReader(cfg InfluxReaderConfig) (*InfluxReader, error) {
	if cfg.Database == "" {
		cfg.Database = "mikhmon"
	}
	return &InfluxReader{
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (r *InfluxReader) QueryRange(ctx context.Context, measurement, routerID string, from, to time.Time, step string) ([]map[string]interface{}, error) {
	query := fmt.Sprintf(
		`SELECT mean("cpu-load") as cpu_load, mean("free-memory") as free_memory, mean("total-memory") as total_memory, mean("free-hdd-space") as free_hdd_space, mean("total-hdd-space") as total_hdd_space FROM "%s" WHERE "router_id" = '%s' AND time >= '%s' AND time <= '%s' GROUP BY time(%s) fill(null) ORDER BY time ASC`,
		measurement,
		routerID,
		from.UTC().Format(time.RFC3339Nano),
		to.UTC().Format(time.RFC3339Nano),
		step,
	)

	url := fmt.Sprintf("%s/api/v3/query_sql?db=%s", r.cfg.URL, r.cfg.Database)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBufferString(query))
	if err != nil {
		return nil, fmt.Errorf("influxdb reader: create request: %w", err)
	}
	if r.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+r.cfg.Token)
	}
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Accept", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("influxdb reader: query failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, fmt.Errorf("influxdb reader: read response: %w", err)
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("influxdb reader: status %d: %s", resp.StatusCode, string(body))
	}

	return parseQueryResponse(body)
}

func parseQueryResponse(body []byte) ([]map[string]interface{}, error) {
	var raw struct {
		Columns []string `json:"columns"`
		Types   []string `json:"types"`
		Values  [][]interface{} `json:"values"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		var fallback []map[string]interface{}
		if err2 := json.Unmarshal(body, &fallback); err2 == nil {
			return fallback, nil
		}
		return nil, nil
	}

	if len(raw.Columns) == 0 || len(raw.Values) == 0 {
		return nil, nil
	}

	results := make([]map[string]interface{}, 0, len(raw.Values))
	for _, row := range raw.Values {
		entry := make(map[string]interface{}, len(raw.Columns))
		for i, col := range raw.Columns {
			if i < len(row) {
				entry[col] = convertValue(row[i], raw.Types, i)
			}
		}
		results = append(results, entry)
	}

	return results, nil
}

func convertValue(v interface{}, types []string, idx int) interface{} {
	if v == nil {
		return nil
	}
	if idx >= len(types) {
		return v
	}

	switch strings.ToUpper(types[idx]) {
	case "INTEGER", "BIGINT", "INT":
		if f, ok := v.(float64); ok {
			return int64(f)
		}
		if s, ok := v.(string); ok {
			if n, err := strconv.ParseInt(s, 10, 64); err == nil {
				return n
			}
		}
	case "DOUBLE", "FLOAT":
		if s, ok := v.(string); ok {
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				return f
			}
		}
	case "TIMESTAMP", "TIMESTAMP WITH TIME ZONE":
		if s, ok := v.(string); ok {
			if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
				return t.Format(time.RFC3339)
			}
		}
	}
	return v
}

var _ Reader = (*InfluxReader)(nil)
