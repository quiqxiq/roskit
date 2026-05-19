package definition

import (
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/core/command"
)

func init() {
	registerSystem()
}

func registerSystem() {
	// system/resource/print — interval-based stream, writes timeseries every 1s
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

	// On-demand only — queried when API is called, no background polling needed.
	command.Register(command.QueryDef("system/identity/print"))
	command.Register(command.QueryDef("system/clock/print"))
	command.Register(command.QueryDef("system/health/print"))
	command.Register(command.QueryDef("system/routerboard/print"))

	// Schedulers and scripts — streamed (push on change).
	command.Register(command.StreamDef("system/scheduler/print", "name", "system_scheduler"))
	command.Register(command.StreamDef("system/script/print", "name", "system_script"))

	// log/print — on-demand query; live streaming handled by Bridge.LogStream().
	command.Register(command.CommandMeta{
		Path:        "log/print",
		Type:        command.CommandTypeQuery,
		Measurement: "log",
		Category:    "log",
	})

	// Mutations — system lifecycle
	command.Register(command.MutationDef("system/reboot"))
	command.Register(command.MutationDef("system/shutdown"))

	// Mutations — scheduler
	command.Register(command.MutationDef("system/scheduler/add"))
	command.Register(command.MutationDef("system/scheduler/set"))
	command.Register(command.MutationDef("system/scheduler/remove"))
	command.Register(command.MutationDef("system/scheduler/enable"))
	command.Register(command.MutationDef("system/scheduler/disable"))

	// Mutations — script
	command.Register(command.MutationDef("system/script/add"))
	command.Register(command.MutationDef("system/script/set"))
	command.Register(command.MutationDef("system/script/remove"))
	command.Register(command.MutationDef("system/script/run"))

	// system/logging/add — used by Bridge.SetupLogging() only.
	command.Register(command.MutationDef("system/logging/add"))
}
