// Package domain defines core business entities for the roskit telemetry library.
// These entities are completely decoupled from any external libraries or infrastructure.
package domain

import "time"

// RouterConfig holds the connection parameters for a MikroTik device.
type RouterConfig struct {
	// ID is a unique identifier for this router (e.g., "core-01").
	ID string
	// Address is the MikroTik API endpoint (e.g., "192.168.88.1:8728" or "192.168.88.1:8729" for TLS).
	Address string
	// Username for API authentication.
	Username string
	// Password for API authentication.
	Password string
	// UseTLS enables TLS connection to the API port (default 8729).
	UseTLS bool

	// ReconnectInterval is how long to wait before reconnecting after a connection drop.
	// The pool uses exponential backoff starting from this value, capped at MaxReconnectInterval.
	// Default: 5s if zero.
	ReconnectInterval time.Duration
	// MaxReconnectInterval is the maximum duration between reconnection attempts.
	// Default: 60s if zero.
	MaxReconnectInterval time.Duration
	// HealthCheckInterval is how often the pool verifies connection liveness
	// by sending /system/identity/print to the router.
	// Default: 30s if zero.
	HealthCheckInterval time.Duration
	// DialTimeout is the maximum time to wait for a connection to be established.
	// Default: 10s if zero.
	DialTimeout time.Duration
}

// Defaults applies default values to zero-valued config fields.
func (c *RouterConfig) Defaults() {
	if c.ReconnectInterval <= 0 {
		c.ReconnectInterval = 5 * time.Second
	}
	if c.MaxReconnectInterval <= 0 {
		c.MaxReconnectInterval = 60 * time.Second
	}
	if c.HealthCheckInterval <= 0 {
		c.HealthCheckInterval = 30 * time.Second
	}
	if c.DialTimeout <= 0 {
		c.DialTimeout = 10 * time.Second
	}
}
