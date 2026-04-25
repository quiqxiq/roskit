package definition

import "github.com/quiqxiq/roskit/internal/roskit/core/command"

func init() {
	registerTools()
}

func registerTools() {
	command.Register(command.StreamDef("tool/netwatch/print", "", "tool_netwatch"))

	command.Register(command.QueryDef("tool/netwatch/find"))

	for _, verb := range []string{"add", "set", "remove", "enable", "disable"} {
		command.Register(command.MutationDef("tool/netwatch/" + verb))
	}
}
