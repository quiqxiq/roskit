package core

import "strings"

// IsRouterOSPermanentError returns true for errors that indicate a RouterOS
// command or feature does not exist on the target device. These errors will
// never resolve without a firmware/package change, so retrying is pointless.
func IsRouterOSPermanentError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "no such command prefix") ||
		strings.Contains(s, "no such command or directory") ||
		strings.Contains(s, "missing =")
}
