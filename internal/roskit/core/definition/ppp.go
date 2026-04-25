package definition

import "github.com/quiqxiq/roskit/internal/roskit/core/command"

func init() {
	registerPPP()
}

func registerPPP() {
	// -------------------------------------------------------------------------
	// STREAM — PPP secrets (follow=true in JSON)
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef("ppp/secret/print", "name", "ppp_secret"))

	// -------------------------------------------------------------------------
	// STREAM — PPP active sessions (follow=true in JSON)
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef("ppp/active/print", "name", "ppp_active"))

	// -------------------------------------------------------------------------
	// STREAM — PPP profiles (follow=true in JSON)
	// Profiles bisa berubah saat admin edit rate limits.
	// -------------------------------------------------------------------------
	command.Register(command.StreamDef("ppp/profile/print", "name", "ppp_profile"))

	// -------------------------------------------------------------------------
	// MUTATIONS — PPP secret
	// -------------------------------------------------------------------------
	for _, verb := range []string{"add", "set", "remove", "enable", "disable"} {
		command.Register(command.MutationDef("ppp/secret/" + verb))
	}

	// -------------------------------------------------------------------------
	// MUTATIONS — PPP active
	// -------------------------------------------------------------------------
	command.Register(command.MutationDef("ppp/active/remove"))

	// -------------------------------------------------------------------------
	// MUTATIONS — PPP profile
	// -------------------------------------------------------------------------
	for _, verb := range []string{"add", "set", "remove", "enable", "disable"} {
		command.Register(command.MutationDef("ppp/profile/" + verb))
	}

	// -------------------------------------------------------------------------
	// QUERIES — PPP
	// -------------------------------------------------------------------------
	command.Register(command.QueryDef("ppp/secret/find"))
	command.Register(command.QueryDef("ppp/active/find"))
	command.Register(command.QueryDef("ppp/profile/find"))
}
