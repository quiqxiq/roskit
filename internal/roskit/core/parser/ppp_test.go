package parser_test

import (
	"testing"

	"github.com/quiqxiq/roskit/internal/roskit/core/command"
	"github.com/quiqxiq/roskit/internal/roskit/core/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseStreamSentence_PPPSecret(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "ppp_secret"}
	pairs := map[string]string{
		".id": "*PS1", "name": "ppp-user1", "password": "secret123",
		"profile": "default", "service": "pppoe",
		"caller-id": "AA:BB:CC:DD:EE:FF",
		"local-address": "10.0.0.1", "remote-address": "10.0.0.100",
		"disabled": "false", "comment": "fiber customer",
	}
	result := parser.ParseStreamSentence("router1", meta, pairs)
	require.NotNil(t, result)
	assert.Equal(t, "*PS1", result.EntityID)
	assert.False(t, result.IsDead)
	assert.Equal(t, "ppp-user1", result.CacheData["name"])
	assert.Equal(t, "secret123", result.CacheData["password"])
	assert.Equal(t, "default", result.CacheData["profile"])
	assert.Equal(t, "pppoe", result.CacheData["service"])
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", result.CacheData["caller_id"])
	assert.Equal(t, "10.0.0.1", result.CacheData["local_address"])
	assert.Equal(t, "10.0.0.100", result.CacheData["remote_address"])
	assert.Equal(t, "false", result.CacheData["disabled"])
	assert.Equal(t, "fiber customer", result.CacheData["comment"])
}

func TestParseStreamSentence_PPPSecret_Disabled(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "ppp_secret"}
	pairs := map[string]string{
		".id": "*PS2", "name": "disabled-user", "password": "pwd",
		"profile": "basic", "service": "pptp",
		"disabled": "true",
	}
	result := parser.ParseStreamSentence("router1", meta, pairs)
	require.NotNil(t, result)
	assert.Equal(t, "*PS2", result.EntityID)
	assert.Equal(t, "true", result.CacheData["disabled"])
}

func TestParseStreamSentence_PPPActive(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "ppp_active"}
	pairs := map[string]string{
		".id": "*PA1", "name": "ppp-user1", "service": "pppoe",
		"caller-id": "AA:BB:CC:DD:EE:FF", "address": "10.0.0.100",
		"uptime": "2h15m", "encoding": "MPPE128",
		"session-id": "ABC123", "limit-bytes-in": "1073741824",
		"limit-bytes-out": "1073741824",
	}
	result := parser.ParseStreamSentence("router1", meta, pairs)
	require.NotNil(t, result)
	assert.Equal(t, "*PA1", result.EntityID)
	assert.False(t, result.IsDead)
	assert.Equal(t, "ppp-user1", result.CacheData["name"])
	assert.Equal(t, "pppoe", result.CacheData["service"])
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", result.CacheData["caller_id"])
	assert.Equal(t, "10.0.0.100", result.CacheData["address"])
	assert.Equal(t, "2h15m", result.CacheData["uptime"])
	assert.Equal(t, "MPPE128", result.CacheData["encoding"])
	assert.Equal(t, "ABC123", result.CacheData["session_id"])
	assert.Equal(t, "1073741824", result.CacheData["limit_bytes_in"])
	assert.Equal(t, "1073741824", result.CacheData["limit_bytes_out"])
}

func TestParseStreamSentence_PPPActive_MinimalFields(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "ppp_active"}
	pairs := map[string]string{
		".id": "*PA2", "name": "minimal-user",
	}
	result := parser.ParseStreamSentence("router1", meta, pairs)
	require.NotNil(t, result)
	assert.Equal(t, "*PA2", result.EntityID)
	assert.Equal(t, "minimal-user", result.CacheData["name"])
	assert.Equal(t, "0", result.CacheData["limit_bytes_in"])
	assert.Equal(t, "0", result.CacheData["limit_bytes_out"])
}
