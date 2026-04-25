package command

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Registry holds CommandMeta for all known RouterOS paths.
// Thread-safe after initialization — Register should only be called at startup.
type Registry struct {
	mu       sync.RWMutex
	commands map[string]*CommandMeta // key = normalized path (no leading slash)
}

var global = &Registry{
	commands: make(map[string]*CommandMeta),
}

// Register adds a CommandMeta to the global registry.
// Panics on duplicate path — this is intentional (catches definition conflicts at startup).
func Register(meta CommandMeta) {
	global.mu.Lock()
	defer global.mu.Unlock()

	path := normalizePath(meta.Path)
	if _, exists := global.commands[path]; exists {
		panic(fmt.Sprintf("routeros/command: duplicate registration for path %q", path))
	}

	// Auto-fill Measurement from path if not set.
	if meta.Measurement == "" {
		meta.Measurement = pathToMeasurement(path)
	}

	// Auto-fill Category from path if not set.
	if meta.Category == "" {
		meta.Category = pathToCategory(path)
	}

	// Normalize path.
	meta.Path = path
	global.commands[path] = &meta
}

// Lookup returns the CommandMeta for the given path.
// path may include or omit the leading slash.
// Returns nil if not found.
func Lookup(path string) *CommandMeta {
	global.mu.RLock()
	defer global.mu.RUnlock()
	return global.commands[normalizePath(path)]
}

// MustLookup returns the CommandMeta or panics.
// Use in init() / test setup to catch missing definitions early.
func MustLookup(path string) *CommandMeta {
	m := Lookup(path)
	if m == nil {
		panic(fmt.Sprintf("routeros/command: no registration for path %q", path))
	}
	return m
}

// All returns a copy of all registered CommandMeta.
func All() []*CommandMeta {
	global.mu.RLock()
	defer global.mu.RUnlock()
	out := make([]*CommandMeta, 0, len(global.commands))
	for _, m := range global.commands {
		cp := *m
		out = append(out, &cp)
	}
	return out
}

// ByType returns all registered commands of the given type.
func ByType(t CommandType) []*CommandMeta {
	global.mu.RLock()
	defer global.mu.RUnlock()
	var out []*CommandMeta
	for _, m := range global.commands {
		if m.Type == t {
			cp := *m
			out = append(out, &cp)
		}
	}
	return out
}

// ByCategory returns all registered commands for a category.
func ByCategory(category string) []*CommandMeta {
	global.mu.RLock()
	defer global.mu.RUnlock()
	var out []*CommandMeta
	for _, m := range global.commands {
		if m.Category == category {
			cp := *m
			out = append(out, &cp)
		}
	}
	return out
}

// --- helpers ---

func normalizePath(p string) string {
	return strings.TrimPrefix(p, "/")
}

// pathToMeasurement converts a RouterOS path to a snake_case measurement name.
// "ip/hotspot/user/print"         → "hotspot_user"
// "ip/hotspot/active/print"       → "hotspot_active"
// "interface/monitor-traffic"     → "interface_traffic"
// "system/resource/print"         → "system_resource"
// "ip/dhcp-server/lease/print"    → "dhcp_lease"
func pathToMeasurement(path string) string {
	parts := strings.Split(path, "/")

	// Strip common prefixes that add noise: "ip", "interface" (when followed by specific)
	// Strip trailing verbs: "print", "monitor-traffic"
	filtered := make([]string, 0, len(parts))
	skip := map[string]bool{"ip": true}
	verbLast := map[string]bool{"print": true, "monitor-traffic": true, "monitor": true}

	for i, p := range parts {
		if i == 0 && skip[p] {
			continue
		}
		if i == len(parts)-1 && verbLast[p] {
			continue
		}
		filtered = append(filtered, strings.ReplaceAll(p, "-", "_"))
	}

	// Take the last 2 segments max to keep names short.
	if len(filtered) > 2 {
		filtered = filtered[len(filtered)-2:]
	}

	return strings.Join(filtered, "_")
}

// pathToCategory extracts the top-level domain from a path.
func pathToCategory(path string) string {
	parts := strings.Split(path, "/")
	// Skip "ip" prefix — use second segment.
	if len(parts) >= 2 && parts[0] == "ip" {
		return strings.ReplaceAll(parts[1], "-", "_")
	}
	return strings.ReplaceAll(parts[0], "-", "_")
}

// --- convenience constructors ---

// StreamDef creates a CommandMeta for a follow-based stream command (the common case).
// Use this for all "print =follow" commands.
func StreamDef(path, indexField, measurement string) CommandMeta {
	return CommandMeta{
		Path:              path,
		Type:              CommandTypeStream,
		SupportsFollow:    true,
		SupportsInterval:  true,
		SupportsCountOnly: true,
		CacheTTL:          5 * time.Minute,
		IndexField:        indexField,
		Measurement:       measurement,
	}
}

// MonitorDef creates a CommandMeta for monitor-* commands (stream, no follow flag).
// Use this for /interface/monitor-traffic, /disk/monitor-traffic, etc.
func MonitorDef(path, measurement string) CommandMeta {
	return CommandMeta{
		Path:             path,
		Type:             CommandTypeStream,
		SupportsFollow:   false,
		SupportsInterval: true,
		CacheTTL:         30 * time.Second,
		Measurement:      measurement,
	}
}

// PollDef creates a CommandMeta for poll (snapshot) commands.
// Use this for /system/resource/print, /system/identity/print, etc.
func PollDef(path string, interval time.Duration) CommandMeta {
	return CommandMeta{
		Path:             path,
		Type:             CommandTypePoll,
		SupportsFollow:   false,
		SupportsInterval: true,
		PollInterval:     interval,
		CacheTTL:         interval * 2, // sensible default
	}
}

// MutationDef creates a CommandMeta for write commands.
func MutationDef(path string) CommandMeta {
	return CommandMeta{
		Path: path,
		Type: CommandTypeMutation,
	}
}

// QueryDef creates a CommandMeta for get/find commands.
func QueryDef(path string) CommandMeta {
	return CommandMeta{
		Path:     path,
		Type:     CommandTypeQuery,
		CacheTTL: 10 * time.Second, // short cache for on-demand queries
	}
}