package timeseries

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatLineProtocol_BasicMeasurement(t *testing.T) {
	w := &InfluxWriter{}
	p := Point{
		Measurement: "hotspot_user",
		Tags:        map[string]string{"router_id": "r1"},
		Fields:      map[string]any{"cpu-load": int64(12)},
	}
	line := w.formatLineProtocol(p)
	assert.Contains(t, line, "hotspot_user,")
	assert.Contains(t, line, "router_id=r1")
	assert.Contains(t, line, "cpu-load=12i")
}

func TestFormatLineProtocol_WithTimestamp(t *testing.T) {
	w := &InfluxWriter{}
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	p := Point{
		Measurement: "sys",
		Tags:        map[string]string{"id": "r1"},
		Fields:      map[string]any{"val": int64(1)},
		Timestamp:   ts,
	}
	line := w.formatLineProtocol(p)
	want := fmt.Sprintf("%d", ts.UnixNano())
	assert.True(t, strings.HasSuffix(line, " "+want), "expected line to end with %q, got %q", want, line)
}

func TestFormatLineProtocol_EscapeSpaces(t *testing.T) {
	w := &InfluxWriter{}
	p := Point{
		Measurement: "my measure",
		Tags:        map[string]string{"host name": "a b"},
		Fields:      map[string]any{"val": int64(1)},
	}
	line := w.formatLineProtocol(p)
	assert.Contains(t, line, `my\ measure`)
	assert.Contains(t, line, `host\ name=a\ b`)
}

func TestFormatLineProtocol_EscapeCommaEquals(t *testing.T) {
	w := &InfluxWriter{}
	p := Point{
		Measurement: "m",
		Tags:        map[string]string{"k,1": "v=2"},
		Fields:      map[string]any{"x": int64(1)},
	}
	line := w.formatLineProtocol(p)
	assert.Contains(t, line, `k\,1=v\=2`)
}

func TestFormatLineProtocol_EmptyFields(t *testing.T) {
	w := &InfluxWriter{}
	p := Point{
		Measurement: "m",
		Tags:        map[string]string{"id": "r1"},
		Fields:      map[string]any{},
	}
	line := w.formatLineProtocol(p)
	assert.Empty(t, line, "should return empty string when no valid fields")
}

func TestFormatLineProtocol_SkipsReservedTimeField(t *testing.T) {
	w := &InfluxWriter{}
	p := Point{
		Measurement: "m",
		Tags:        map[string]string{"id": "r1"},
		Fields:      map[string]any{"time": "2024-01-01", "val": int64(5)},
	}
	line := w.formatLineProtocol(p)
	assert.NotContains(t, line, "time=")
	assert.Contains(t, line, "val=5i")
}

func TestFormatLineProtocol_SkipsTagKeyAsField(t *testing.T) {
	w := &InfluxWriter{}
	p := Point{
		Measurement: "m",
		Tags:        map[string]string{"router_id": "r1"},
		Fields:      map[string]any{"router_id": "r1", "val": int64(1)},
	}
	line := w.formatLineProtocol(p)
	// split into "measurement,tags" and "fields [timestamp]"
	parts := strings.SplitN(line, " ", 2)
	assert.Len(t, parts, 2)
	assert.NotContains(t, parts[1], "router_id=")
	assert.Contains(t, parts[1], "val=1i")
}

func TestFormatValue_Float64(t *testing.T) {
	assert.Equal(t, "1.5", formatValue(float64(1.5)))
}

func TestFormatValue_Int64(t *testing.T) {
	assert.Equal(t, "42i", formatValue(int64(42)))
}

func TestFormatValue_Int(t *testing.T) {
	assert.Equal(t, "7i", formatValue(int(7)))
}

func TestFormatValue_BoolTrue(t *testing.T) {
	assert.Equal(t, "true", formatValue(true))
}

func TestFormatValue_BoolFalse(t *testing.T) {
	assert.Equal(t, "false", formatValue(false))
}

func TestFormatValue_String(t *testing.T) {
	assert.Equal(t, `"hello"`, formatValue("hello"))
}

func TestFormatValue_UnknownType(t *testing.T) {
	result := formatValue([]int{1, 2})
	assert.Contains(t, result, "[1 2]")
}
