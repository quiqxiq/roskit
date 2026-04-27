package definition

import (
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/core/command"
)

// init registers all hotspot-related RouterOS commands.
// Classification is derived from RouterOS 7.20.8 JSON schema:
//   - All hotspot "print" commands have "follow" arg → CommandTypeStream
//   - Query/mutation commands registered as-needed (lazy — only register what's used)
func init() {
	registerHotspot()
}

func registerHotspot() {
	// -------------------------------------------------------------------------
	// STREAM — /ip/hotspot/user
	// JSON confirms: follow=true, follow-only=true, interval=true, count-only=true
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef(
		"ip/hotspot/user/print",
		"name", // secondary index: lookup user by name O(1)
		"hotspot_user",
	))

	// -------------------------------------------------------------------------
	// STREAM — /ip/hotspot/user/profile
	// JSON confirms: follow=true
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef(
		"ip/hotspot/user/profile/print",
		"name",
		"hotspot_profile",
	))

	// -------------------------------------------------------------------------
	// STREAM — /ip/hotspot/active
	// JSON confirms: follow=true
	// No index field — active sessions identified by ID only (changes frequently)
	// -------------------------------------------------------------------------
	hsActive := command.StreamDef("ip/hotspot/active/print", "", "hotspot_active")
	hsActive.WriteTimeSeries = true
	command.Register(hsActive)

	// -------------------------------------------------------------------------
	// STREAM — /ip/hotspot (server list)
	// JSON confirms: follow=true
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef(
		"ip/hotspot/print",
		"name",
		"hotspot_server",
	))

	// -------------------------------------------------------------------------
	// STREAM — /ip/hotspot/host
	// JSON confirms: follow=true
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef(
		"ip/hotspot/host/print",
		"", // no stable index — MAC changes on reconnect
		"hotspot_host",
	))

	// -------------------------------------------------------------------------
	// STREAM — /ip/hotspot/cookie
	// JSON confirms: follow=true
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef(
		"ip/hotspot/cookie/print",
		"", // lookup by user requires client-side filter
		"hotspot_cookie",
	))

	// -------------------------------------------------------------------------
	// STREAM — /ip/hotspot/ip-binding
	// JSON confirms: follow=true
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef(
		"ip/hotspot/ip-binding/print",
		"", // no single stable field — MAC or address both work but neither is canonical
		"ip_binding",
	))

	// -------------------------------------------------------------------------
	// MUTATIONS — /ip/hotspot/user
	// Registered here so the registry can validate commands before execution.
	// -------------------------------------------------------------------------
	for _, verb := range []string{"add", "set", "remove", "enable", "disable", "reset-counters"} {
		command.Register(command.MutationDef("ip/hotspot/user/" + verb))
	}

	// -------------------------------------------------------------------------
	// MUTATIONS — /ip/hotspot/user/profile
	// -------------------------------------------------------------------------
	for _, verb := range []string{"add", "set", "remove", "reset"} {
		command.Register(command.MutationDef("ip/hotspot/user/profile/" + verb))
	}

	// -------------------------------------------------------------------------
	// MUTATIONS — /ip/hotspot/active
	// -------------------------------------------------------------------------
	command.Register(command.MutationDef("ip/hotspot/active/remove"))
	command.Register(command.MutationDef("ip/hotspot/active/login"))

	// -------------------------------------------------------------------------
	// MUTATIONS — /ip/hotspot/host
	// -------------------------------------------------------------------------
	command.Register(command.MutationDef("ip/hotspot/host/remove"))

	// -------------------------------------------------------------------------
	// MUTATIONS — /ip/hotspot/cookie
	// -------------------------------------------------------------------------
	command.Register(command.MutationDef("ip/hotspot/cookie/remove"))

	// -------------------------------------------------------------------------
	// MUTATIONS — /ip/hotspot/ip-binding
	// -------------------------------------------------------------------------
	for _, verb := range []string{"add", "set", "remove", "enable", "disable"} {
		command.Register(command.MutationDef("ip/hotspot/ip-binding/" + verb))
	}

	// -------------------------------------------------------------------------
	// MUTATIONS — /ip/hotspot (server-level)
	// -------------------------------------------------------------------------
	for _, verb := range []string{"add", "set", "remove", "enable", "disable", "reset-html"} {
		command.Register(command.MutationDef("ip/hotspot/" + verb))
	}

	// -------------------------------------------------------------------------
	// QUERIES — find/get (used for specific lookups, short-cached)
	// -------------------------------------------------------------------------
	command.Register(command.QueryDef("ip/hotspot/user/find"))
	command.Register(command.QueryDef("ip/hotspot/user/get"))
	command.Register(command.QueryDef("ip/hotspot/user/profile/find"))
	command.Register(command.QueryDef("ip/hotspot/user/profile/get"))
	command.Register(command.QueryDef("ip/hotspot/active/find"))
	command.Register(command.QueryDef("ip/hotspot/active/get"))

	// -------------------------------------------------------------------------
	// HOT PATH: walled-garden (streamed but lower priority)
	// -------------------------------------------------------------------------
	command.Register(command.CommandMeta{
		Path:              "ip/hotspot/walled-garden/print",
		Type:              command.CommandTypeStream,
		SupportsFollow:    true,
		SupportsInterval:  true,
		SupportsCountOnly: true,
		CacheTTL:          5 * time.Minute,
		Measurement:       "walled_garden",
		Category:          "hotspot",
	})

	// -------------------------------------------------------------------------
	// POLL — /ip/hotspot/profile (server-level profiles, beda dengan user profile)
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("ip/hotspot/profile/print", 5*time.Minute))

	// -------------------------------------------------------------------------
	// STREAM — /ip/hotspot/walled-garden/ip (IP-based walled garden)
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef("ip/hotspot/walled-garden/ip/print", "", "walled_garden_ip"))

	// -------------------------------------------------------------------------
	// POLL — /ip/hotspot/service-port (sangat jarang berubah)
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("ip/hotspot/service-port/print", 30*time.Minute))

	// -------------------------------------------------------------------------
	// MUTATIONS — /ip/hotspot/profile
	// -------------------------------------------------------------------------
	for _, verb := range []string{"add", "set", "remove"} {
		command.Register(command.MutationDef("ip/hotspot/profile/" + verb))
	}

	// -------------------------------------------------------------------------
	// MUTATIONS — /ip/hotspot/walled-garden/ip
	// -------------------------------------------------------------------------
	command.Register(command.MutationDef("ip/hotspot/walled-garden/ip/add"))
	command.Register(command.MutationDef("ip/hotspot/walled-garden/ip/remove"))
}