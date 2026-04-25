package definition

import (
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/core/command"
)

func init() {
	registerInterfaces()
}

func registerInterfaces() {
	// -------------------------------------------------------------------------
	// POLL — interface/vlan (admin-only config)
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("interface/vlan/print", 5*time.Minute))

	// -------------------------------------------------------------------------
	// POLL — interface/bridge (admin-only config)
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("interface/bridge/print", 5*time.Minute))
	command.Register(command.PollDef("interface/bridge/port/print", 5*time.Minute))

	// -------------------------------------------------------------------------
	// STREAM — interface/bridge/host (MAC table, sering berubah)
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef("interface/bridge/host/print", "", "bridge_host"))

	// -------------------------------------------------------------------------
	// POLL — interface/ethernet (hardware-bound, jarang berubah)
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("interface/ethernet/print", 5*time.Minute))

	// -------------------------------------------------------------------------
	// POLL — interface/wireless legacy
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("interface/wireless/print", 5*time.Minute))

	// -------------------------------------------------------------------------
	// STREAM — interface/wireless/registration-table (client connect/disconnect)
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef(
		"interface/wireless/registration-table/print",
		"",
		"wireless_registration_table",
	))

	// -------------------------------------------------------------------------
	// POLL — interface/wifi (WiFi 6, admin-only config)
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("interface/wifi/print", 5*time.Minute))

	// -------------------------------------------------------------------------
	// STREAM — interface/wifi/registration-table (WiFi 6 client connect/disconnect)
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef(
		"interface/wifi/registration-table/print",
		"",
		"wifi_registration_table",
	))

	// -------------------------------------------------------------------------
	// POLL — interface/wireguard (admin-only config)
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("interface/wireguard/print", 5*time.Minute))

	// -------------------------------------------------------------------------
	// POLL — interface/wireguard/peers (handshake state, poll lebih sering)
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("interface/wireguard/peers/print", 2*time.Minute))

	// -------------------------------------------------------------------------
	// POLL — interface/pppoe-client (state bisa berubah saat reconnect)
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("interface/pppoe-client/print", 2*time.Minute))

	// -------------------------------------------------------------------------
	// POLL — interface/l2tp-client
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("interface/l2tp-client/print", 2*time.Minute))

	// -------------------------------------------------------------------------
	// QUERIES
	// -------------------------------------------------------------------------
	command.Register(command.QueryDef("interface/vlan/find"))
	command.Register(command.QueryDef("interface/bridge/find"))
	command.Register(command.QueryDef("interface/bridge/port/find"))
	command.Register(command.QueryDef("interface/ethernet/find"))
	command.Register(command.QueryDef("interface/wireless/find"))
	command.Register(command.QueryDef("interface/wifi/find"))
	command.Register(command.QueryDef("interface/wireguard/find"))
	command.Register(command.QueryDef("interface/wireguard/peers/find"))
	command.Register(command.QueryDef("interface/pppoe-client/find"))
	command.Register(command.QueryDef("interface/l2tp-client/find"))

	// -------------------------------------------------------------------------
	// MUTATIONS — vlan
	// -------------------------------------------------------------------------
	for _, verb := range []string{"add", "set", "remove", "enable", "disable"} {
		command.Register(command.MutationDef("interface/vlan/" + verb))
	}

	// -------------------------------------------------------------------------
	// MUTATIONS — bridge
	// -------------------------------------------------------------------------
	for _, verb := range []string{"add", "set", "remove", "enable", "disable"} {
		command.Register(command.MutationDef("interface/bridge/" + verb))
		command.Register(command.MutationDef("interface/bridge/port/" + verb))
	}

	// -------------------------------------------------------------------------
	// MUTATIONS — ethernet
	// -------------------------------------------------------------------------
	for _, verb := range []string{"set", "enable", "disable"} {
		command.Register(command.MutationDef("interface/ethernet/" + verb))
	}
	command.Register(command.MutationDef("interface/ethernet/reset-counters"))

	// -------------------------------------------------------------------------
	// MUTATIONS — wireless
	// -------------------------------------------------------------------------
	for _, verb := range []string{"set", "enable", "disable"} {
		command.Register(command.MutationDef("interface/wireless/" + verb))
		command.Register(command.MutationDef("interface/wifi/" + verb))
	}

	// -------------------------------------------------------------------------
	// MUTATIONS — wireguard
	// -------------------------------------------------------------------------
	for _, verb := range []string{"add", "set", "remove", "enable", "disable"} {
		command.Register(command.MutationDef("interface/wireguard/" + verb))
		command.Register(command.MutationDef("interface/wireguard/peers/" + verb))
	}

	// -------------------------------------------------------------------------
	// MUTATIONS — pppoe-client / l2tp-client
	// -------------------------------------------------------------------------
	for _, verb := range []string{"add", "set", "remove", "enable", "disable"} {
		command.Register(command.MutationDef("interface/pppoe-client/" + verb))
		command.Register(command.MutationDef("interface/l2tp-client/" + verb))
	}
}
