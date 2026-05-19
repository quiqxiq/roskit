package definition

import (
	"github.com/quiqxiq/roskit/internal/roskit/core/command"
)

func init() {
	registerNetwork()
}

func registerNetwork() {
	// STREAM — used by Bridge methods and SSE handlers
	command.Register(command.StreamDef("interface/print", "name", "interface"))
	command.Register(command.StreamDef("ip/pool/print", "name", "ip_pool"))
	command.Register(command.StreamDef("ip/arp/print", "address", "arp"))
	command.Register(command.StreamDef("ip/dhcp-server/lease/print", "address", "dhcp_lease"))
	command.Register(command.StreamDef("ip/firewall/nat/print", "", "firewall_nat"))
	command.Register(command.StreamDef("queue/simple/print", "name", "queue_simple"))

	// interface/monitor-traffic is NOT registered in the command registry.
	// It is managed directly by InterfaceMonitorManager and bridge.InterfaceTrafficStream()
	// via dispatcher.RunListen() — no registry lookup required.

	// MUTATIONS — interface
	command.Register(command.MutationDef("interface/set"))
	command.Register(command.MutationDef("interface/enable"))
	command.Register(command.MutationDef("interface/disable"))

	// MUTATIONS — arp
	command.Register(command.MutationDef("ip/arp/remove"))

	// MUTATIONS — dhcp lease (ReleaseDHCPLease uses lease/remove)
	command.Register(command.MutationDef("ip/dhcp-server/lease/add"))
	command.Register(command.MutationDef("ip/dhcp-server/lease/remove"))

	// MUTATIONS — firewall nat
	for _, verb := range []string{"add", "set", "remove", "enable", "disable", "move"} {
		command.Register(command.MutationDef("ip/firewall/nat/" + verb))
	}

	// MUTATIONS — queue/simple
	command.Register(command.MutationDef("queue/simple/add"))
	command.Register(command.MutationDef("queue/simple/set"))
	command.Register(command.MutationDef("queue/simple/remove"))
}
