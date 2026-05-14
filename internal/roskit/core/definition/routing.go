package definition

import (
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/core/command"
)

func init() {
	registerRouting()
}

func registerRouting() {
	// -------------------------------------------------------------------------
	// POLL — routing/ospf/instance (admin-only config)
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("routing/ospf/instance/print", 5*time.Minute))

	// -------------------------------------------------------------------------
	// POLL — routing/ospf/interface (mapping config, jarang berubah)
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("routing/ospf/interface/print", 5*time.Minute))

	// -------------------------------------------------------------------------
	// POLL — routing/bgp/connection (admin-only config)
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("routing/bgp/connection/print", 5*time.Minute))

	// -------------------------------------------------------------------------
	// QUERIES
	// -------------------------------------------------------------------------
	command.Register(command.QueryDef("routing/ospf/instance/find"))
	command.Register(command.QueryDef("routing/bgp/connection/find"))

	// -------------------------------------------------------------------------
	// MUTATIONS — routing/ospf
	// -------------------------------------------------------------------------
	for _, verb := range []string{"add", "set", "remove", "enable", "disable"} {
		command.Register(command.MutationDef("routing/ospf/instance/" + verb))
	}
	for _, verb := range []string{"add", "set", "remove"} {
		command.Register(command.MutationDef("routing/ospf/area/" + verb))
	}

	// -------------------------------------------------------------------------
	// MUTATIONS — routing/bgp
	// -------------------------------------------------------------------------
	for _, verb := range []string{"add", "set", "remove", "enable", "disable"} {
		command.Register(command.MutationDef("routing/bgp/connection/" + verb))
	}
}
