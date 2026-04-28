package execution

import "time"

type ConnectionRole string

const (
	RoleClient ConnectionRole = "client"
	RoleStream ConnectionRole = "stream"
)

type RoleConfig struct {
	QueueSize      int
	CommandTimeout time.Duration
}

var DefaultRoleConfigs = map[ConnectionRole]RoleConfig{
	RoleClient: {
		QueueSize:      100,
		CommandTimeout: 15 * time.Second,
	},
	RoleStream: {
		QueueSize:      500,
		CommandTimeout: 0,
	},
}
