package execution

import (
	"errors"
	"strings"
)

// ConnState represents the lifecycle state of a router connection.
type ConnState uint8

const (
	ConnStateDisconnected ConnState = iota
	ConnStateConnecting
	ConnStateConnected
	ConnStateAuthFailed
)

func (s ConnState) String() string {
	switch s {
	case ConnStateConnected:
		return "connected"
	case ConnStateConnecting:
		return "connecting"
	case ConnStateAuthFailed:
		return "auth_failed"
	default:
		return "disconnected"
	}
}

var ErrAuthFailed = errors.New("routeros: authentication failed")

// isAuthError returns true when the RouterOS API rejects credentials.
// These errors will never succeed on retry.
func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "cannot log in") ||
		strings.Contains(msg, "invalid user name or password") ||
		strings.Contains(msg, "authentication failed")
}
