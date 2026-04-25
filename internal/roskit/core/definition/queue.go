package definition

import (
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/core/command"
)

func init() {
	registerQueues()
}

func registerQueues() {
	// -------------------------------------------------------------------------
	// POLL — queue/tree (admin-only hierarchy config)
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("queue/tree/print", 10*time.Minute))

	// -------------------------------------------------------------------------
	// POLL — queue/interface (aggregate stats per interface, not per-packet)
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("queue/interface/print", 1*time.Minute))

	// -------------------------------------------------------------------------
	// POLL — queue/type (algorithm config, sangat jarang berubah)
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("queue/type/print", 30*time.Minute))

	// -------------------------------------------------------------------------
	// QUERIES
	// -------------------------------------------------------------------------
	command.Register(command.QueryDef("queue/tree/find"))
	command.Register(command.QueryDef("queue/interface/find"))
	command.Register(command.QueryDef("queue/type/find"))

	// -------------------------------------------------------------------------
	// MUTATIONS — queue/tree
	// -------------------------------------------------------------------------
	for _, verb := range []string{"add", "set", "remove", "enable", "disable"} {
		command.Register(command.MutationDef("queue/tree/" + verb))
	}
	command.Register(command.MutationDef("queue/tree/reset-counters"))

	// -------------------------------------------------------------------------
	// MUTATIONS — queue/type
	// -------------------------------------------------------------------------
	for _, verb := range []string{"add", "set", "remove"} {
		command.Register(command.MutationDef("queue/type/" + verb))
	}
}
