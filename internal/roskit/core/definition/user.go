package definition

import (
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/core/command"
)

func init() {
	registerUsers()
}

func registerUsers() {
	command.Register(command.PollDef("user/print", 30*time.Minute))
	command.Register(command.StreamDef("user/active/print", "", "user_active"))
	command.Register(command.PollDef("user/group/print", 30*time.Minute))

	command.Register(command.QueryDef("user/find"))
	command.Register(command.QueryDef("user/active/find"))

	for _, verb := range []string{"add", "set", "remove", "enable", "disable"} {
		command.Register(command.MutationDef("user/" + verb))
	}
}
