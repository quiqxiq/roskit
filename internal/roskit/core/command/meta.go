package command

import "time"

// CommandType describes HOW a command behaves, not what it does.
//
// Classification rules (derived from RouterOS JSON schema):
//
//   STREAM   → command has "follow" or "follow-only" arg, OR is a monitor-* command.
//              RouterOS pushes data continuously. Client uses ListenArgsContext.
//              Examples: /ip/hotspot/user/print =follow, /interface/monitor-traffic
//
//   POLL     → "print" command WITHOUT "follow" arg.
//              RouterOS returns a snapshot. Client calls periodically on a ticker.
//              Examples: /system/resource/print, /system/identity/print
//
//   QUERY    → "get" or "find" command.
//              On-demand, low-frequency, optionally short-cached.
//              Examples: /ip/hotspot/user/get, /ip/hotspot/user/find
//
//   MUTATION → "add", "set", "remove", "enable", "disable", "reset", etc.
//              Writes to RouterOS. Never cached as source of truth.
//              Examples: /ip/hotspot/user/add, /ip/firewall/filter/remove
//
//   ACTION   → One-shot commands that don't fit other categories.
//              Examples: /system/reboot, /tool/ping, /certificate/sign
type CommandType uint8

const (
	CommandTypeStream   CommandType = iota + 1
	CommandTypePoll
	CommandTypeQuery
	CommandTypeMutation
	CommandTypeAction
)

// String implements fmt.Stringer.
func (ct CommandType) String() string {
	switch ct {
	case CommandTypeStream:
		return "stream"
	case CommandTypePoll:
		return "poll"
	case CommandTypeQuery:
		return "query"
	case CommandTypeMutation:
		return "mutation"
	case CommandTypeAction:
		return "action"
	default:
		return "unknown"
	}
}

// IsRealtime returns true for behaviors that produce continuous data.
func (ct CommandType) IsRealtime() bool {
	return ct == CommandTypeStream
}

// IsCacheable returns true for behaviors where caching the result makes sense.
func (ct CommandType) IsCacheable() bool {
	return ct == CommandTypeStream || ct == CommandTypePoll || ct == CommandTypeQuery
}

// CommandMeta is the single source of truth for how a RouterOS path behaves.
// Populated from the RouterOS JSON schema — do not hand-edit individual fields.
type CommandMeta struct {
	// Path is the canonical RouterOS path WITHOUT leading slash.
	// e.g. "ip/hotspot/user/print"
	Path string

	// Type is the behavior classification.
	Type CommandType

	// SupportsFollow indicates the command accepts "=follow" or "=follow-only".
	// Only meaningful for stream commands.
	SupportsFollow bool

	// SupportsInterval indicates the command accepts "=interval=Xs".
	// Stream commands: RouterOS controls the push interval.
	// Poll commands: used to implement periodic polling.
	SupportsInterval bool

	// SupportsCountOnly indicates the command accepts "=count-only".
	// Useful for GetCount() calls without fetching full data.
	SupportsCountOnly bool

	// PollInterval is the default polling interval for CommandTypePoll.
	// Zero means use the global default (60s).
	PollInterval time.Duration

	// CacheTTL is the default Redis TTL for cached snapshots.
	// Zero means use the global default (5 minutes for stream, 30s for poll).
	CacheTTL time.Duration

	// IndexField is the entity field used as a secondary Redis index key.
	// Empty means no secondary index (lookups require full scan).
	// e.g. "name" for hotspot users, "address" for DHCP leases.
	IndexField string

	// Measurement is the InfluxDB measurement name and Redis key segment.
	// Derived from path if empty: "ip/hotspot/user/print" → "hotspot_user"
	Measurement string

	// Category groups related commands for observability/filtering.
	// e.g. "hotspot", "ppp", "system", "network", "interface"
	Category string

	// WriteTimeSeries controls whether events from this command are written to InfluxDB.
	// Only true for actual telemetry: system_resource, interface_traffic, hotspot_active.
	WriteTimeSeries bool
}

// RouterOSPath returns the path in RouterOS wire format (leading slash, slash separators).
// e.g. "ip/hotspot/user/print" → "/ip/hotspot/user/print"
func (m CommandMeta) RouterOSPath() string {
	return "/" + m.Path
}

// IsPoll is a convenience method.
func (m CommandMeta) IsPoll() bool { return m.Type == CommandTypePoll }

// IsStream is a convenience method.
func (m CommandMeta) IsStream() bool { return m.Type == CommandTypeStream }

// IsMutation is a convenience method.
func (m CommandMeta) IsMutation() bool { return m.Type == CommandTypeMutation }