package domain

import "time"

// BGPSession represents a BGP peer session state.
type BGPSession struct {
	ID               string    // RouterOS internal identifier.
	Name             string    // Session/peer name.
	RemoteAddress    string    // Remote peer IP address.
	RemoteAS         string    // Remote autonomous system number.
	LocalRole        string    // Local role (ebgp, ibgp).
	RemoteRole       string    // Remote role.
	State            string    // Session state (established, idle, connect, active, opensent, openconfirm).
	Uptime           string    // Session uptime.
	PrefixCount      string    // Number of prefixes received.
	EstablishedCount string    // Times session has been established.
	Disabled         bool      // Whether session is disabled.
	Timestamp        time.Time // When this sample was collected.
}

func (b *BGPSession) ToTags() map[string]string {
	return map[string]string{
		"name":           b.Name,
		"remote_address": b.RemoteAddress,
		"remote_as":      b.RemoteAS,
		"state":          b.State,
	}
}

func (b *BGPSession) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"local_role":        b.LocalRole,
		"remote_role":       b.RemoteRole,
		"uptime":            b.Uptime,
		"prefix_count":      b.PrefixCount,
		"established_count": b.EstablishedCount,
		"disabled":          b.Disabled,
	}
}

func (b *BGPSession) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id": b.ID, "name": b.Name, "remote_address": b.RemoteAddress,
		"remote_as": b.RemoteAS, "local_role": b.LocalRole, "remote_role": b.RemoteRole,
		"state": b.State, "uptime": b.Uptime, "prefix_count": b.PrefixCount,
		"established_count": b.EstablishedCount, "timestamp": ts.Format(time.RFC3339),
	}
}
