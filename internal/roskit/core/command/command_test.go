package command

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Register test-only commands into the global registry.
// The definition packages are not imported here, so we seed our own.
func init() {
	Register(StreamDef("test/stream/print", "name", "test_stream"))
	Register(MutationDef("test/mutation/add"))
	Register(QueryDef("test/query/print"))
	Register(PollDef("test/poll/print", 30*time.Second))
}

// --- normalizePath ---

func TestNormalizePath(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"ip/hotspot/user/print", "ip/hotspot/user/print"},
		{"/ip/hotspot/user/print", "ip/hotspot/user/print"},
		{"//double", "/double"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizePath(tt.input))
		})
	}
}

// --- pathToMeasurement ---

func TestPathToMeasurement(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"ip/hotspot/user/print", "hotspot_user"},
		{"ip/hotspot/active/print", "hotspot_active"},
		{"ip/hotspot/user/profile/print", "user_profile"},
		{"system/resource/print", "system_resource"},
		{"interface/monitor-traffic", "interface"},
		{"ip/dhcp-server/lease/print", "dhcp_server_lease"},
		{"ip/arp/print", "arp"},
		{"ppp/secret/print", "ppp_secret"},
		{"queue/simple/print", "queue_simple"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, pathToMeasurement(tt.input))
		})
	}
}

// --- pathToCategory ---

func TestPathToCategory(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"ip/hotspot/user/print", "hotspot"},
		{"ip/dhcp-server/lease/print", "dhcp_server"},
		{"system/resource/print", "system"},
		{"ppp/secret/print", "ppp"},
		{"interface/print", "interface"},
		{"queue/simple/print", "queue"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, pathToCategory(tt.input))
		})
	}
}

// --- Lookup / MustLookup / ByType / ByCategory ---

func TestLookup_NotFound(t *testing.T) {
	result := Lookup("nonexistent/path/that/does/not/exist")
	assert.Nil(t, result)
}

func TestLookup_WithLeadingSlash(t *testing.T) {
	// Any registered path should be found with or without leading slash.
	all := All()
	require.NotEmpty(t, all, "registry must have at least one registered command")
	path := all[0].Path
	assert.NotNil(t, Lookup(path))
	assert.NotNil(t, Lookup("/"+path))
}

func TestAll_ReturnsAtLeastOne(t *testing.T) {
	all := All()
	assert.NotEmpty(t, all)
}

func TestByType_Stream(t *testing.T) {
	streams := ByType(CommandTypeStream)
	for _, m := range streams {
		assert.Equal(t, CommandTypeStream, m.Type)
	}
	assert.NotEmpty(t, streams)
}

func TestByType_Mutation(t *testing.T) {
	mutations := ByType(CommandTypeMutation)
	for _, m := range mutations {
		assert.Equal(t, CommandTypeMutation, m.Type)
	}
	assert.NotEmpty(t, mutations)
}

func TestByCategory_Returns(t *testing.T) {
	results := ByCategory("test")
	assert.NotEmpty(t, results)
	for _, m := range results {
		assert.Equal(t, "test", m.Category)
	}
}

// --- CommandType methods ---

func TestCommandType_String(t *testing.T) {
	tests := []struct {
		ct       CommandType
		expected string
	}{
		{CommandTypeStream, "stream"},
		{CommandTypePoll, "poll"},
		{CommandTypeQuery, "query"},
		{CommandTypeMutation, "mutation"},
		{CommandTypeAction, "action"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.ct.String())
	}
}

func TestCommandType_IsRealtime(t *testing.T) {
	assert.True(t, CommandTypeStream.IsRealtime())
	assert.False(t, CommandTypePoll.IsRealtime())
	assert.False(t, CommandTypeQuery.IsRealtime())
	assert.False(t, CommandTypeMutation.IsRealtime())
}

func TestCommandType_IsCacheable(t *testing.T) {
	assert.True(t, CommandTypeStream.IsCacheable())
	assert.True(t, CommandTypePoll.IsCacheable())
	assert.False(t, CommandTypeMutation.IsCacheable())
}

// --- CommandMeta methods ---

func TestCommandMeta_RouterOSPath(t *testing.T) {
	m := CommandMeta{Path: "ip/hotspot/user/print"}
	assert.Equal(t, "/ip/hotspot/user/print", m.RouterOSPath())
}

func TestCommandMeta_TypeCheckers(t *testing.T) {
	stream := CommandMeta{Type: CommandTypeStream}
	assert.True(t, stream.IsStream())
	assert.False(t, stream.IsPoll())
	assert.False(t, stream.IsMutation())

	poll := CommandMeta{Type: CommandTypePoll}
	assert.True(t, poll.IsPoll())

	mutation := CommandMeta{Type: CommandTypeMutation}
	assert.True(t, mutation.IsMutation())
}

// --- Sentence builders ---

func TestBuildStreamSentence_WithFollow(t *testing.T) {
	meta := &CommandMeta{
		Path:             "ip/hotspot/user/print",
		SupportsFollow:   true,
		SupportsInterval: false,
	}
	sentence := BuildStreamSentence(meta)
	assert.Equal(t, "/ip/hotspot/user/print", sentence[0])
	assert.Contains(t, sentence, "=follow")
}

func TestBuildStreamSentence_WithInterval(t *testing.T) {
	meta := &CommandMeta{
		Path:             "system/resource/print",
		SupportsFollow:   true,
		SupportsInterval: true,
	}
	sentence := BuildStreamSentence(meta)
	assert.Contains(t, sentence, "=interval=1s")
}

func TestBuildPollSentence(t *testing.T) {
	meta := &CommandMeta{Path: "system/identity/print"}
	sentence := BuildPollSentence(meta)
	assert.Equal(t, []string{"/system/identity/print"}, sentence)
}

func TestBuildQuerySentence_NoFilters(t *testing.T) {
	meta := &CommandMeta{Path: "ip/hotspot/user/print"}
	sentence := BuildQuerySentence(meta, nil)
	assert.Equal(t, []string{"/ip/hotspot/user/print"}, sentence)
}

func TestBuildQuerySentence_WithFilters(t *testing.T) {
	meta := &CommandMeta{Path: "ip/hotspot/user/print"}
	sentence := BuildQuerySentence(meta, []string{"?name=user001"})
	assert.Equal(t, "/ip/hotspot/user/print", sentence[0])
	assert.Equal(t, "?name=user001", sentence[1])
}

func TestBuildAddSentence(t *testing.T) {
	sentence := BuildAddSentence("ip/hotspot/user/add", map[string]string{
		"name":     "user001",
		"password": "pass001",
	})
	assert.Equal(t, "/ip/hotspot/user/add", sentence[0])
	assert.Contains(t, sentence, "=name=user001")
	assert.Contains(t, sentence, "=password=pass001")
}

func TestBuildSetSentence(t *testing.T) {
	sentence := BuildSetSentence("ip/hotspot/user/set", "*1", map[string]string{
		"disabled": "true",
	})
	assert.Equal(t, "/ip/hotspot/user/set", sentence[0])
	assert.Equal(t, "=.id=*1", sentence[1])
	assert.Contains(t, sentence, "=disabled=true")
}

func TestBuildRemoveSentence(t *testing.T) {
	sentence := BuildRemoveSentence("ip/hotspot/user", "*1")
	assert.Equal(t, []string{"/ip/hotspot/user/remove", "=.id=*1"}, sentence)
}

func TestBuildEnableSentence(t *testing.T) {
	sentence := BuildEnableSentence("ip/hotspot/user", "*1")
	assert.Equal(t, []string{"/ip/hotspot/user/enable", "=numbers=*1"}, sentence)
}

func TestBuildDisableSentence(t *testing.T) {
	sentence := BuildDisableSentence("ip/hotspot/user", "*1")
	assert.Equal(t, []string{"/ip/hotspot/user/disable", "=numbers=*1"}, sentence)
}

func TestBuildMonitorSentence(t *testing.T) {
	meta := &CommandMeta{
		Path:             "interface/monitor-traffic",
		SupportsInterval: true,
	}
	sentence := BuildMonitorSentence(meta, map[string]string{"interface": "ether1"})
	assert.Equal(t, "/interface/monitor-traffic", sentence[0])
	assert.Contains(t, sentence, "=interface=ether1")
	assert.Contains(t, sentence, "=interval=1s")
}

// --- PollIntervalOrDefault ---

func TestPollIntervalOrDefault_UsesMeta(t *testing.T) {
	meta := &CommandMeta{PollInterval: 30 * time.Second}
	result := PollIntervalOrDefault(meta, 60*time.Second)
	assert.Equal(t, 30*time.Second, result)
}

func TestPollIntervalOrDefault_UsesDefault(t *testing.T) {
	meta := &CommandMeta{PollInterval: 0}
	result := PollIntervalOrDefault(meta, 60*time.Second)
	assert.Equal(t, 60*time.Second, result)
}

// --- Convenience constructors ---

func TestStreamDef(t *testing.T) {
	m := StreamDef("ip/hotspot/user/print", "name", "hotspot_user")
	assert.Equal(t, CommandTypeStream, m.Type)
	assert.True(t, m.SupportsFollow)
	assert.Equal(t, "name", m.IndexField)
	assert.Equal(t, "hotspot_user", m.Measurement)
	assert.Greater(t, m.CacheTTL, time.Duration(0))
}

func TestPollDef(t *testing.T) {
	m := PollDef("system/resource/print", 30*time.Second)
	assert.Equal(t, CommandTypePoll, m.Type)
	assert.Equal(t, 30*time.Second, m.PollInterval)
	assert.Equal(t, 60*time.Second, m.CacheTTL) // 2x interval
}

func TestMutationDef(t *testing.T) {
	m := MutationDef("ip/hotspot/user/add")
	assert.Equal(t, CommandTypeMutation, m.Type)
}

func TestQueryDef(t *testing.T) {
	m := QueryDef("ip/hotspot/user/print")
	assert.Equal(t, CommandTypeQuery, m.Type)
	assert.Greater(t, m.CacheTTL, time.Duration(0))
}
