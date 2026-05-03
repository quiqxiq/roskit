package command_test

import (
	"testing"

	"github.com/quiqxiq/roskit/internal/roskit/core/command"
	_ "github.com/quiqxiq/roskit/internal/roskit/core/definition"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefinitionsRegistered(t *testing.T) {
	all := command.All()
	require.NotEmpty(t, all, "no commands registered — definition import may be missing")
	assert.GreaterOrEqual(t, len(all), 100, "expected at least 100 commands registered")
}

func TestCriticalPaths(t *testing.T) {
	critical := []string{
		"ip/hotspot/user/print",
		"ip/hotspot/active/print",
		"ip/hotspot/user/profile/print",
		"ip/hotspot/user/add",
		"ip/hotspot/user/set",
		"ip/hotspot/user/remove",
		"ip/hotspot/user/enable",
		"ip/hotspot/user/disable",
		"system/resource/print",
		"system/identity/print",
		"system/scheduler/print",
		"system/scheduler/add",
		"system/scheduler/set",
		"system/scheduler/remove",
		"system/script/print",
		"system/script/add",
		"system/script/set",
		"system/script/remove",
		"interface/print",
		"ip/dhcp-server/lease/print",
		"queue/simple/print",
		"ppp/secret/print",
		"ppp/secret/add",
		"ppp/secret/set",
		"ppp/secret/remove",
	}
	for _, path := range critical {
		t.Run(path, func(t *testing.T) {
			meta := command.Lookup(path)
			require.NotNil(t, meta, "critical path %q not registered", path)
			assert.NotEmpty(t, meta.Measurement)
		})
	}
}

func TestStreamCommandsExist(t *testing.T) {
	streams := command.ByType(command.CommandTypeStream)
	assert.NotEmpty(t, streams)
	for _, m := range streams {
		assert.Equal(t, command.CommandTypeStream, m.Type)
		assert.NotEmpty(t, m.Measurement)
	}
}

func TestPollCommandsExist(t *testing.T) {
	polls := command.ByType(command.CommandTypePoll)
	assert.NotEmpty(t, polls)
}

func TestMutationCommandsExist(t *testing.T) {
	mutations := command.ByType(command.CommandTypeMutation)
	assert.NotEmpty(t, mutations)
}
