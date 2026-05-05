package models

import "github.com/quiqxiq/roskit/internal/roskit/execution"

type RouterStatus string

const (
	RouterStatusUnknown      RouterStatus = "unknown"
	RouterStatusConnecting   RouterStatus = "connecting"
	RouterStatusConnected    RouterStatus = "connected"
	RouterStatusDisconnected RouterStatus = "disconnected"
	RouterStatusAuthFailed   RouterStatus = "auth_failed"
)

func (s RouterStatus) String() string { return string(s) }

func (s RouterStatus) IsTerminal() bool {
	return s == RouterStatusAuthFailed
}

func (s RouterStatus) IsHealthy() bool {
	return s == RouterStatusConnected
}

func RouterStatusFromConnState(state execution.ConnState) RouterStatus {
	switch state {
	case execution.ConnStateConnected:
		return RouterStatusConnected
	case execution.ConnStateConnecting:
		return RouterStatusConnecting
	case execution.ConnStateAuthFailed:
		return RouterStatusAuthFailed
	default:
		return RouterStatusDisconnected
	}
}
