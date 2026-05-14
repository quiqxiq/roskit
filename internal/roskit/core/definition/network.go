package definition

import (
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/core/command"
)

func init() {
	registerNetwork()
}

func registerNetwork() {
	// -------------------------------------------------------------------------
	// STREAM — interface list (follow=true in JSON)
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef("interface/print", "name", "interface"))

	// -------------------------------------------------------------------------
	// STREAM — /interface/monitor-traffic
	// Handled by InterfaceMonitorManager per-interface (not auto-registered).
	// -------------------------------------------------------------------------
	_ = command.MonitorDef("interface/monitor-traffic", "interface_traffic")

	// -------------------------------------------------------------------------
	// STREAM — /ip/address (follow=true in JSON)
	// Berubah saat DHCP renew, failover, atau admin edit.
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef("ip/address/print", "address", "address"))

	// -------------------------------------------------------------------------
	// STREAM — /ip/route (follow=true in JSON)
	// Berubah dengan dynamic routing (OSPF/BGP) dan failover.
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef("ip/route/print", "", "route"))

	// -------------------------------------------------------------------------
	// STREAM — ARP (follow=true in JSON)
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef("ip/arp/print", "address", "arp"))

	// -------------------------------------------------------------------------
	// STREAM — /ip/neighbor (follow=true in JSON)
	// Perangkat di-discover terus via CDP/LLDP.
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef("ip/neighbor/print", "", "neighbor"))

	// -------------------------------------------------------------------------
	// STREAM — IP Pool (follow=true in JSON)
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef("ip/pool/print", "name", "ip_pool"))

	// -------------------------------------------------------------------------
	// STREAM — /ip/pool/used (follow=true in JSON)
	// Berubah tiap kali hotspot/PPP assign atau release IP.
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef("ip/pool/used/print", "", "pool_used"))

	// -------------------------------------------------------------------------
	// STREAM — DHCP lease (follow=true in JSON)
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef(
		"ip/dhcp-server/lease/print",
		"address",
		"dhcp_lease",
	))

	// -------------------------------------------------------------------------
	// STREAM — /ip/dns/static (follow=true in JSON)
	// Bisa di-update dinamis (split-DNS, dynamic blocklists).
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef("ip/dns/static/print", "", "dns_static"))

	// -------------------------------------------------------------------------
	// STREAM — NAT rules (follow=true in JSON)
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef("ip/firewall/nat/print", "", "firewall_nat"))

	// -------------------------------------------------------------------------
	// STREAM — Simple queues (follow=true in JSON)
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef("queue/simple/print", "name", "queue_simple"))

	// -------------------------------------------------------------------------
	// POLL — /ip/dhcp-server (instances)
	// Admin jarang ubah server instances; follow=true tapi jarang berubah.
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("ip/dhcp-server/print", 5*time.Minute))
	command.Register(command.PollDef("ip/dhcp-server/network/print", 10*time.Minute))
	command.Register(command.PollDef("ip/dhcp-server/option/print", 30*time.Minute))

	// -------------------------------------------------------------------------
	// POLL — /ip/dhcp-client
	// State bisa berubah saat renew/rebind.
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("ip/dhcp-client/print", 2*time.Minute))

	// -------------------------------------------------------------------------
	// POLL — /ip/dns
	// Global DNS settings sangat jarang berubah.
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("ip/dns/print", 30*time.Minute))

	// -------------------------------------------------------------------------
	// POLL — /ip/service
	// Port settings sangat jarang berubah.
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("ip/service/print", 30*time.Minute))

	// -------------------------------------------------------------------------
	// POLL — /ip/vrf
	// VRF config sangat jarang berubah.
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("ip/vrf/print", 30*time.Minute))

	// -------------------------------------------------------------------------
	// QUERIES — interface
	// -------------------------------------------------------------------------
	command.Register(command.QueryDef("interface/find"))
	command.Register(command.QueryDef("interface/get"))

	// -------------------------------------------------------------------------
	// QUERIES — ip network
	// -------------------------------------------------------------------------
	command.Register(command.QueryDef("ip/address/find"))
	command.Register(command.QueryDef("ip/address/get"))
	command.Register(command.QueryDef("ip/route/find"))
	command.Register(command.QueryDef("ip/arp/find"))
	command.Register(command.QueryDef("ip/neighbor/find"))
	command.Register(command.QueryDef("ip/pool/find"))
	command.Register(command.QueryDef("ip/pool/used/find"))
	command.Register(command.QueryDef("ip/dhcp-server/lease/find"))
	command.Register(command.QueryDef("ip/dhcp-server/find"))
	command.Register(command.QueryDef("ip/dns/static/find"))
	command.Register(command.QueryDef("ip/service/find"))
	command.Register(command.QueryDef("ip/vrf/find"))
	command.Register(command.QueryDef("queue/simple/find"))
	command.Register(command.QueryDef("queue/simple/get"))

	// -------------------------------------------------------------------------
	// MUTATIONS — interface
	// -------------------------------------------------------------------------
	command.Register(command.MutationDef("interface/set"))
	command.Register(command.MutationDef("interface/enable"))
	command.Register(command.MutationDef("interface/disable"))

	// -------------------------------------------------------------------------
	// MUTATIONS — ip/address
	// -------------------------------------------------------------------------
	for _, verb := range []string{"add", "set", "remove", "enable", "disable"} {
		command.Register(command.MutationDef("ip/address/" + verb))
	}

	// -------------------------------------------------------------------------
	// MUTATIONS — ip/route
	// -------------------------------------------------------------------------
	for _, verb := range []string{"add", "set", "remove", "enable", "disable"} {
		command.Register(command.MutationDef("ip/route/" + verb))
	}

	// -------------------------------------------------------------------------
	// MUTATIONS — ip/arp
	// -------------------------------------------------------------------------
	command.Register(command.MutationDef("ip/arp/remove"))

	// -------------------------------------------------------------------------
	// MUTATIONS — ip/dhcp-server
	// -------------------------------------------------------------------------
	command.Register(command.MutationDef("ip/dhcp-server/lease/add"))
	command.Register(command.MutationDef("ip/dhcp-server/lease/remove"))
	for _, verb := range []string{"add", "set", "remove", "enable", "disable"} {
		command.Register(command.MutationDef("ip/dhcp-server/" + verb))
	}
	for _, verb := range []string{"add", "set", "remove"} {
		command.Register(command.MutationDef("ip/dhcp-server/network/" + verb))
		command.Register(command.MutationDef("ip/dhcp-server/option/" + verb))
	}

	// -------------------------------------------------------------------------
	// MUTATIONS — ip/dhcp-client
	// -------------------------------------------------------------------------
	for _, verb := range []string{"add", "set", "remove", "enable", "disable", "renew", "release"} {
		command.Register(command.MutationDef("ip/dhcp-client/" + verb))
	}

	// -------------------------------------------------------------------------
	// MUTATIONS — ip/dns
	// -------------------------------------------------------------------------
	command.Register(command.MutationDef("ip/dns/set"))
	for _, verb := range []string{"add", "set", "remove", "enable", "disable"} {
		command.Register(command.MutationDef("ip/dns/static/" + verb))
	}
	command.Register(command.MutationDef("ip/dns/cache/flush"))

	// -------------------------------------------------------------------------
	// MUTATIONS — ip/service
	// -------------------------------------------------------------------------
	for _, verb := range []string{"set", "enable", "disable"} {
		command.Register(command.MutationDef("ip/service/" + verb))
	}

	// -------------------------------------------------------------------------
	// MUTATIONS — ip/vrf
	// -------------------------------------------------------------------------
	for _, verb := range []string{"add", "set", "remove", "enable", "disable"} {
		command.Register(command.MutationDef("ip/vrf/" + verb))
	}

	// -------------------------------------------------------------------------
	// MUTATIONS — ip/firewall/nat
	// -------------------------------------------------------------------------
	for _, verb := range []string{"add", "set", "remove", "enable", "disable", "move"} {
		command.Register(command.MutationDef("ip/firewall/nat/" + verb))
	}

	// -------------------------------------------------------------------------
	// MUTATIONS — queue/simple
	// -------------------------------------------------------------------------
	command.Register(command.MutationDef("queue/simple/add"))
	command.Register(command.MutationDef("queue/simple/set"))
	command.Register(command.MutationDef("queue/simple/remove"))
}
