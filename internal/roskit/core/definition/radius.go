package definition

import (
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/core/command"
)

func init() {
	registerRadius()
}

func registerRadius() {
	command.Register(command.PollDef("radius/print", 30*time.Minute))

	command.Register(command.QueryDef("radius/find"))

	for _, verb := range []string{"add", "set", "remove", "enable", "disable"} {
		command.Register(command.MutationDef("radius/" + verb))
	}
}
