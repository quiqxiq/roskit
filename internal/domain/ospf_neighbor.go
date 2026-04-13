package domain

import "time"

// OSPFNeighbor represents an OSPF neighbor adjacency.
type OSPFNeighbor struct {
	ID            string    // RouterOS internal identifier.
	Instance      string    // OSPF instance name.
	RouterID      string    // Neighbor's router ID.
	Address       string    // Neighbor's IP address.
	Interface     string    // Interface where neighbor was discovered.
	Priority      string    // Neighbor priority.
	State         string    // Adjacency state (full, 2way, init, down, etc.).
	StateChanges  string    // Number of state changes.
	Adjacency     string    // Adjacency duration.
	Timestamp     time.Time // When this sample was collected.
}

func (o *OSPFNeighbor) ToTags() map[string]string {
	return map[string]string{
		"instance":  o.Instance,
		"router_id": o.RouterID,
		"address":   o.Address,
		"interface": o.Interface,
		"state":     o.State,
	}
}

func (o *OSPFNeighbor) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"priority":      o.Priority,
		"state_changes": o.StateChanges,
		"adjacency":     o.Adjacency,
	}
}

func (o *OSPFNeighbor) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id": o.ID, "instance": o.Instance, "router_id": o.RouterID,
		"address": o.Address, "interface": o.Interface, "priority": o.Priority,
		"state": o.State, "state_changes": o.StateChanges, "adjacency": o.Adjacency,
		"timestamp": ts.Format(time.RFC3339),
	}
}
