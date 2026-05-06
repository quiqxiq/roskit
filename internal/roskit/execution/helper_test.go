package execution

import (
	"errors"
	"testing"
)

func TestConnStateString(t *testing.T) {
	tests := []struct {
		state ConnState
		want  string
	}{
		{ConnStateConnected, "connected"},
		{ConnStateConnecting, "connecting"},
		{ConnStateAuthFailed, "auth_failed"},
		{ConnStateDisconnected, "disconnected"},
		{ConnState(99), "disconnected"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			if got := tc.state.String(); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "unrelated", err: errors.New("connection refused"), want: false},
		{name: "empty message", err: errors.New(""), want: false},

		// Known RouterOS auth-failure variants — keep this list as the
		// canonical snapshot. If a new variant appears in the wild, add it
		// here AND update isAuthError so retry/backoff logic short-circuits.
		{name: "cannot log in lower", err: errors.New("cannot log in"), want: true},
		{name: "cannot log in mixed", err: errors.New("Cannot Log In"), want: true},
		{name: "cannot log in suffix", err: errors.New("from API: cannot log in"), want: true},
		{name: "invalid user name or password", err: errors.New("invalid user name or password"), want: true},
		{name: "invalid user name or password upper", err: errors.New("INVALID USER NAME OR PASSWORD"), want: true},
		{name: "authentication failed", err: errors.New("authentication failed"), want: true},
		{name: "authentication failed wrapped", err: errors.New("routeros: authentication failed (101)"), want: true},

		// Edge cases — substrings that should NOT match.
		{name: "log in unrelated", err: errors.New("user must log in periodically"), want: false},
		{name: "password ok", err: errors.New("password expires soon"), want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isAuthError(tc.err); got != tc.want {
				t.Errorf("isAuthError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
