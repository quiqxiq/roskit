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
	// STREAM — routing/route (complete routing table, changes with dynamic routing)
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef("routing/route/print", "", "routing_route"))

	// -------------------------------------------------------------------------
	// POLL — routing/ospf/instance (admin-only config)
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("routing/ospf/instance/print", 5*time.Minute))

	// -------------------------------------------------------------------------
	// STREAM — routing/ospf/neighbor (state berubah saat link flap)
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef("routing/ospf/neighbor/print", "", "ospf_neighbor"))

	// -------------------------------------------------------------------------
	// POLL — routing/ospf/interface (mapping config, jarang berubah)
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("routing/ospf/interface/print", 5*time.Minute))

	// -------------------------------------------------------------------------
	// POLL — routing/bgp/connection (admin-only config)
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("routing/bgp/connection/print", 5*time.Minute))

	// -------------------------------------------------------------------------
	// STREAM — routing/bgp/session (state berubah saat peer flap)
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef("routing/bgp/session/print", "", "bgp_session"))

	// -------------------------------------------------------------------------
	// QUERIES
	// -------------------------------------------------------------------------
	command.Register(command.QueryDef("routing/route/find"))
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
