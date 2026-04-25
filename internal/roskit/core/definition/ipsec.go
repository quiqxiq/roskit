package definition

import (
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/core/command"
)

func init() {
	registerIPSec()
}

func registerIPSec() {
	command.Register(command.PollDef("ip/ipsec/peer/print", 5*time.Minute))
	command.Register(command.StreamDef("ip/ipsec/active-peers/print", "", "ipsec_active_peers"))
	command.Register(command.PollDef("ip/ipsec/policy/print", 5*time.Minute))
	command.Register(command.StreamDef("ip/ipsec/installed-sa/print", "", "ipsec_installed_sa"))

	command.Register(command.QueryDef("ip/ipsec/peer/find"))
	command.Register(command.QueryDef("ip/ipsec/policy/find"))
	command.Register(command.QueryDef("ip/ipsec/active-peers/find"))

	for _, verb := range []string{"add", "set", "remove", "enable", "disable"} {
		command.Register(command.MutationDef("ip/ipsec/peer/" + verb))
		command.Register(command.MutationDef("ip/ipsec/policy/" + verb))
	}
}
