package definition

import (
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/core/command"
)

func init() {
	registerUserManager()
}

func registerUserManager() {
	command.Register(command.PollDef("user-manager/user/print", 2*time.Minute))
	command.Register(command.PollDef("user-manager/profile/print", 10*time.Minute))
	command.Register(command.PollDef("user-manager/router/print", 10*time.Minute))

	command.Register(command.QueryDef("user-manager/user/find"))

	for _, verb := range []string{"add", "set", "remove"} {
		command.Register(command.MutationDef("user-manager/user/" + verb))
		command.Register(command.MutationDef("user-manager/profile/" + verb))
	}
}
