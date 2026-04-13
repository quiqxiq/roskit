package tool

import (
	"context"
	"fmt"

	"github.com/go-routeros/routeros/v3"
	"github.com/quiqxiq/roskit/internal/collector"
)

// Executor provides a high-level API to execute ad-hoc RouterOS commands
// (CRUD operations) through the existing connection pool (Engine).
type Executor struct {
	engine *collector.Engine
}

// NewExecutor creates a new Tool Executor.
func NewExecutor(engine *collector.Engine) *Executor {
	return &Executor{engine: engine}
}

// Run executes an arbitrary command on the specified router.
func (e *Executor) Run(ctx context.Context, routerID string, command ...string) (*routeros.Reply, error) {
	return e.engine.ExecuteCommand(ctx, routerID, command...)
}

// Enable enables a configuration item using its internal ID.
// Example: Enable("core-01", "/ppp/secret", "*1A")
func (e *Executor) Enable(ctx context.Context, routerID, path, id string) error {
	cmd := fmt.Sprintf("%s/enable", path)
	_, err := e.Run(ctx, routerID, cmd, "=numbers="+id)
	return err
}

// Disable disables a configuration item using its internal ID.
// Example: Disable("core-01", "/ppp/secret", "*1A")
func (e *Executor) Disable(ctx context.Context, routerID, path, id string) error {
	cmd := fmt.Sprintf("%s/disable", path)
	_, err := e.Run(ctx, routerID, cmd, "=numbers="+id)
	return err
}

// Remove removes a configuration item using its internal ID.
// Example: Remove("core-01", "/ppp/secret", "*1A")
func (e *Executor) Remove(ctx context.Context, routerID, path, id string) error {
	cmd := fmt.Sprintf("%s/remove", path)
	_, err := e.Run(ctx, routerID, cmd, "=numbers="+id)
	return err
}

// Add adds a new configuration item.
// Example: Add("core-01", "/ppp/secret", map[string]string{"name":"user1", "password":"123", "profile":"default"})
func (e *Executor) Add(ctx context.Context, routerID, path string, params map[string]string) (*routeros.Reply, error) {
	cmd := []string{fmt.Sprintf("%s/add", path)}
	for k, v := range params {
		cmd = append(cmd, fmt.Sprintf("=%s=%s", k, v))
	}
	return e.Run(ctx, routerID, cmd...)
}

// Set updates an existing configuration item.
// Example: Set("core-01", "/ppp/secret", "*1A", map[string]string{"password":"newpassword"})
func (e *Executor) Set(ctx context.Context, routerID, path, id string, params map[string]string) error {
	cmd := []string{fmt.Sprintf("%s/set", path), "=numbers=" + id}
	for k, v := range params {
		cmd = append(cmd, fmt.Sprintf("=%s=%s", k, v))
	}
	_, err := e.Run(ctx, routerID, cmd...)
	return err
}
