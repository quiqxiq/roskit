package definition

import (
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/core/command"
)

func init() {
	registerSystem()
	registerNetwork()
	registerPPP()
}

func registerSystem() {
	// -------------------------------------------------------------------------
	// POLL — system commands
	// JSON confirms: NONE of these have "follow" arg → CommandTypePoll
	// They support "interval" → used for ticker-based polling in behavior/poll/
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("system/resource/print", 60*time.Second))
	command.Register(command.PollDef("system/identity/print", 5*time.Minute))
	command.Register(command.PollDef("system/clock/print", 60*time.Second))
	command.Register(command.PollDef("system/health/print", 30*time.Second))

	// system/routerboard/print — NOT in RouterOS JSON schema 7.20.8
	// (may be CHR-only or older path). Register manually as poll, long interval.
	command.Register(command.CommandMeta{
		Path:         "system/routerboard/print",
		Type:         command.CommandTypePoll,
		PollInterval: 10 * time.Minute, // hardware info rarely changes
		CacheTTL:     15 * time.Minute,
		Measurement:  "system_routerboard",
		Category:     "system",
	})

	// -------------------------------------------------------------------------
	// STREAM — /system/scheduler (has follow in JSON)
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef("system/scheduler/print", "name", "system_scheduler"))

	// -------------------------------------------------------------------------
	// STREAM — /system/script (has follow in JSON)
	// Used for sales records AND quick-print packages.
	// Measurement is generic; consumer filters by script name pattern.
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef("system/script/print", "name", "system_script"))

	// -------------------------------------------------------------------------
	// MUTATIONS — system
	// -------------------------------------------------------------------------
	command.Register(command.MutationDef("system/scheduler/add"))
	command.Register(command.MutationDef("system/scheduler/set"))
	command.Register(command.MutationDef("system/scheduler/remove"))
	command.Register(command.MutationDef("system/script/add"))
	command.Register(command.MutationDef("system/script/set"))
	command.Register(command.MutationDef("system/script/remove"))
	command.Register(command.MutationDef("system/script/run"))
	command.Register(command.MutationDef("system/shutdown"))
	command.Register(command.MutationDef("system/reboot"))

	// system/log/print — NOT in 7.20.8 JSON (may be legacy path)
	// Treat as a direct-query action (no cache, no stream).
	command.Register(command.CommandMeta{
		Path:        "system/log/print",
		Type:        command.CommandTypeQuery,
		Measurement: "system_log",
		Category:    "system",
	})
}

func registerNetwork() {
	// -------------------------------------------------------------------------
	// STREAM — interface list (follow=true in JSON)
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef("interface/print", "name", "interface"))

	// -------------------------------------------------------------------------
	// STREAM — /interface/monitor-traffic
	// JSON: follow=false, interval=true — it's a monitor-type (continuous push)
	// -------------------------------------------------------------------------
	command.Register(command.MonitorDef("interface/monitor-traffic", "interface_traffic"))

	// -------------------------------------------------------------------------
	// STREAM — DHCP lease (follow=true in JSON)
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef(
		"ip/dhcp-server/lease/print",
		"address", // secondary index: lookup by IP address
		"dhcp_lease",
	))

	// -------------------------------------------------------------------------
	// STREAM — ARP (follow=true in JSON)
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef("ip/arp/print", "address", "arp"))

	// -------------------------------------------------------------------------
	// STREAM — IP Pool (follow=true in JSON)
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef("ip/pool/print", "name", "ip_pool"))

	// -------------------------------------------------------------------------
	// STREAM — NAT rules (follow=true in JSON)
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef("ip/firewall/nat/print", "", "firewall_nat"))

	// -------------------------------------------------------------------------
	// STREAM — Simple queues (follow=true in JSON)
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef("queue/simple/print", "name", "queue_simple"))

	// -------------------------------------------------------------------------
	// QUERIES — network
	// -------------------------------------------------------------------------
	command.Register(command.QueryDef("interface/find"))
	command.Register(command.QueryDef("interface/get"))
	command.Register(command.QueryDef("ip/arp/find"))
	command.Register(command.QueryDef("ip/dhcp-server/lease/find"))
	command.Register(command.QueryDef("ip/pool/find"))
	command.Register(command.QueryDef("queue/simple/find"))
	command.Register(command.QueryDef("queue/simple/get"))

	// -------------------------------------------------------------------------
	// MUTATIONS — network
	// -------------------------------------------------------------------------
	command.Register(command.MutationDef("ip/dhcp-server/lease/add"))
	command.Register(command.MutationDef("ip/dhcp-server/lease/remove"))
	command.Register(command.MutationDef("ip/arp/remove"))
	command.Register(command.MutationDef("queue/simple/add"))
	command.Register(command.MutationDef("queue/simple/set"))
	command.Register(command.MutationDef("queue/simple/remove"))
	command.Register(command.MutationDef("interface/set"))
	command.Register(command.MutationDef("interface/enable"))
	command.Register(command.MutationDef("interface/disable"))
}

func registerPPP() {
	// -------------------------------------------------------------------------
	// STREAM — PPP secrets (follow=true in JSON)
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef("ppp/secret/print", "name", "ppp_secret"))

	// -------------------------------------------------------------------------
	// STREAM — PPP active sessions (follow=true in JSON)
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef("ppp/active/print", "name", "ppp_active"))

	// -------------------------------------------------------------------------
	// MUTATIONS — PPP
	// -------------------------------------------------------------------------
	for _, verb := range []string{"add", "set", "remove", "enable", "disable"} {
		command.Register(command.MutationDef("ppp/secret/" + verb))
	}
	command.Register(command.MutationDef("ppp/active/remove"))

	// -------------------------------------------------------------------------
	// QUERIES — PPP
	// -------------------------------------------------------------------------
	command.Register(command.QueryDef("ppp/secret/find"))
	command.Register(command.QueryDef("ppp/active/find"))
	command.Register(command.QueryDef("ppp/profile/print"))
}