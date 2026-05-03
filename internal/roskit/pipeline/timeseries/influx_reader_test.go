package timeseries

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseQueryResponse_ColumnarFormat(t *testing.T) {
	body := mustJSON(map[string]any{
		"columns": []string{"time", "cpu_load", "free_memory"},
		"types":   []string{"TIMESTAMP", "DOUBLE", "INTEGER"},
		"values": [][]any{
			{"2024-01-15T10:30:00Z", 12.5, float64(104857600)},
		},
	})

	rows, err := parseQueryResponse(body)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "2024-01-15T10:30:00Z", rows[0]["time"])
	assert.Equal(t, 12.5, rows[0]["cpu_load"])
	assert.Equal(t, int64(104857600), rows[0]["free_memory"])
}

func TestParseQueryResponse_ArrayFormat(t *testing.T) {
	body := mustJSON([]map[string]any{
		{"router_id": "r1", "value": 42.0},
	})

	rows, err := parseQueryResponse(body)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "r1", rows[0]["router_id"])
}

func TestParseQueryResponse_EmptyValues(t *testing.T) {
	body := mustJSON(map[string]any{
		"columns": []string{"time"},
		"types":   []string{"TIMESTAMP"},
		"values":  [][]any{},
	})

	rows, err := parseQueryResponse(body)
	require.NoError(t, err)
	assert.Nil(t, rows)
}

func TestParseQueryResponse_EmptyBody(t *testing.T) {
	rows, err := parseQueryResponse([]byte("{}"))
	require.NoError(t, err)
	assert.Nil(t, rows)
}

func TestParseQueryResponse_InvalidJSON(t *testing.T) {
	rows, err := parseQueryResponse([]byte("not-json"))
	require.NoError(t, err)
	assert.Nil(t, rows)
}

func TestConvertValue_IntegerType_FromFloat(t *testing.T) {
	result := convertValue(float64(1024), []string{"INTEGER"}, 0)
	assert.Equal(t, int64(1024), result)
}

func TestConvertValue_IntegerType_FromString(t *testing.T) {
	result := convertValue("256", []string{"BIGINT"}, 0)
	assert.Equal(t, int64(256), result)
}

func TestConvertValue_DoubleType_FromString(t *testing.T) {
	result := convertValue("3.14", []string{"DOUBLE"}, 0)
	assert.Equal(t, 3.14, result)
}

func TestConvertValue_TimestampType_RFC3339(t *testing.T) {
	result := convertValue("2024-01-15T10:30:00Z", []string{"TIMESTAMP"}, 0)
	assert.Equal(t, "2024-01-15T10:30:00Z", result)
}

func TestConvertValue_NilValue(t *testing.T) {
	result := convertValue(nil, []string{"INTEGER"}, 0)
	assert.Nil(t, result)
}

func TestConvertValue_NoTypeInfo(t *testing.T) {
	result := convertValue("raw", []string{}, 5)
	assert.Equal(t, "raw", result)
}

func TestConvertValue_Passthrough_UnknownType(t *testing.T) {
	result := convertValue("val", []string{"UNKNOWN"}, 0)
	assert.Equal(t, "val", result)
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
