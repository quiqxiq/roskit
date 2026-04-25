package definition

import (
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/core/command"
)

func init() {
	registerFirewall()
}

func registerFirewall() {
	// -------------------------------------------------------------------------
	// POLL — /ip/firewall/filter
	// Rules berubah hanya saat admin edit. follow=true tapi perubahan jarang.
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("ip/firewall/filter/print", 10*time.Minute))

	// -------------------------------------------------------------------------
	// STREAM — /ip/firewall/address-list (follow=true in JSON)
	// Diisi dinamis oleh firewall rules (add-src-to-address-list).
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef("ip/firewall/address-list/print", "", "firewall_address_list"))

	// -------------------------------------------------------------------------
	// POLL — /ip/firewall/mangle
	// Seperti filter, berubah hanya saat admin edit.
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("ip/firewall/mangle/print", 10*time.Minute))

	// -------------------------------------------------------------------------
	// POLL — /ip/firewall/raw
	// Seperti filter.
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("ip/firewall/raw/print", 10*time.Minute))

	// -------------------------------------------------------------------------
	// STREAM — /ip/firewall/connection (follow=true in JSON)
	// Very high-frequency: ribuan koneksi berubah per detik.
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef("ip/firewall/connection/print", "", "firewall_connection"))

	// -------------------------------------------------------------------------
	// QUERIES
	// -------------------------------------------------------------------------
	command.Register(command.QueryDef("ip/firewall/filter/find"))
	command.Register(command.QueryDef("ip/firewall/address-list/find"))
	command.Register(command.QueryDef("ip/firewall/mangle/find"))
	command.Register(command.QueryDef("ip/firewall/connection/find"))

	// -------------------------------------------------------------------------
	// MUTATIONS — filter
	// -------------------------------------------------------------------------
	for _, verb := range []string{"add", "set", "remove", "enable", "disable", "move"} {
		command.Register(command.MutationDef("ip/firewall/filter/" + verb))
	}
	command.Register(command.MutationDef("ip/firewall/filter/reset-counters"))

	// -------------------------------------------------------------------------
	// MUTATIONS — address-list
	// -------------------------------------------------------------------------
	for _, verb := range []string{"add", "set", "remove", "enable", "disable"} {
		command.Register(command.MutationDef("ip/firewall/address-list/" + verb))
	}

	// -------------------------------------------------------------------------
	// MUTATIONS — mangle
	// -------------------------------------------------------------------------
	for _, verb := range []string{"add", "set", "remove", "enable", "disable", "move"} {
		command.Register(command.MutationDef("ip/firewall/mangle/" + verb))
	}

	// -------------------------------------------------------------------------
	// MUTATIONS — raw
	// -------------------------------------------------------------------------
	for _, verb := range []string{"add", "set", "remove", "enable", "disable", "move"} {
		command.Register(command.MutationDef("ip/firewall/raw/" + verb))
	}

	// -------------------------------------------------------------------------
	// MUTATIONS — connection
	// -------------------------------------------------------------------------
	command.Register(command.MutationDef("ip/firewall/connection/remove"))
}
