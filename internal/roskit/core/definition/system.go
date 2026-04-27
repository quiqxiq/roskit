package definition

import (
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/core/command"
)

func init() {
	registerSystem()
}

func registerSystem() {
	// -------------------------------------------------------------------------
	// POLL — system commands
	// JSON confirms: NONE of these have "follow" arg → CommandTypePoll
	// They support "interval" → used for ticker-based polling in behavior/poll/
	// -------------------------------------------------------------------------
	// system/resource/print supports =interval=Xs (not =follow).
	// Run as stream with interval=1s so fields are included in pub-sub messages.
	systemResource := command.CommandMeta{
		Path:             "system/resource/print",
		Type:             command.CommandTypeStream,
		SupportsFollow:   false,
		SupportsInterval: true,
		CacheTTL:         10 * time.Second,
		Measurement:      "system_resource",
		Category:         "system",
		WriteTimeSeries:  true,
	}
	command.Register(systemResource)
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
	command.Register(command.MutationDef("system/scheduler/enable"))
	command.Register(command.MutationDef("system/scheduler/disable"))
	command.Register(command.MutationDef("system/script/add"))
	command.Register(command.MutationDef("system/script/set"))
	command.Register(command.MutationDef("system/script/remove"))
	command.Register(command.MutationDef("system/script/run"))
	command.Register(command.MutationDef("system/shutdown"))
	command.Register(command.MutationDef("system/reboot"))

	// log/print — correct RouterOS path (not system/log/print)
	command.Register(command.CommandMeta{
		Path:        "log/print",
		Type:        command.CommandTypeQuery,
		Measurement: "log",
		Category:    "log",
	})

	// -------------------------------------------------------------------------
	// POLL — system/resource/cpu (per-core stats, supplement system/resource)
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("system/resource/cpu/print", 30*time.Second))

	// -------------------------------------------------------------------------
	// POLL — system/logging (follow=true but admin-only config)
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("system/logging/print", 30*time.Minute))
	command.Register(command.PollDef("system/logging/action/print", 30*time.Minute))

	// -------------------------------------------------------------------------
	// POLL — system/package (hanya berubah saat upgrade)
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("system/package/print", 60*time.Minute))

	// -------------------------------------------------------------------------
	// POLL — system/ntp (sangat jarang berubah)
	// -------------------------------------------------------------------------
	command.Register(command.PollDef("system/ntp/client/print", 60*time.Minute))
	command.Register(command.CommandMeta{
		Path:         "system/ntp/client/servers/print",
		Type:         command.CommandTypePoll,
		SupportsFollow:   true,
		SupportsInterval: true,
		PollInterval: 60 * time.Minute,
		CacheTTL:     2 * time.Hour,
		Measurement:  "ntp_servers",
		Category:     "system",
	})

	// -------------------------------------------------------------------------
	// MUTATIONS — system/logging
	// -------------------------------------------------------------------------
	for _, verb := range []string{"add", "set", "remove", "enable", "disable"} {
		command.Register(command.MutationDef("system/logging/" + verb))
	}

	// -------------------------------------------------------------------------
	// MUTATIONS — system/ntp
	// -------------------------------------------------------------------------
	command.Register(command.MutationDef("system/ntp/client/set"))
	for _, verb := range []string{"add", "set", "remove"} {
		command.Register(command.MutationDef("system/ntp/client/servers/" + verb))
	}

	// -------------------------------------------------------------------------
	// QUERIES — system extensions
	// -------------------------------------------------------------------------
	command.Register(command.QueryDef("system/logging/find"))
	command.Register(command.QueryDef("system/package/find"))
}

