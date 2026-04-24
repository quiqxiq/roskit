package behavior

import (
	"context"
	"time"

	"github.com/go-routeros/routeros/v3"
	"github.com/quiqxiq/roskit/internal/roskit/core/command"
)

type StreamEvent struct {
	Meta       *command.CommandMeta
	RouterID   string
	Fields     map[string]string
	IsDead     bool
	ReceivedAt time.Time
}

type PollEvent struct {
	Meta     *command.CommandMeta
	RouterID string
	Rows     []map[string]string
	PollTime time.Time
	Err      error
}

type StreamSink interface {
	OnEvent(ctx context.Context, event StreamEvent) error
}

type PollSink interface {
	OnPoll(ctx context.Context, event PollEvent) error
}

type StreamHandler interface {
	Start(ctx context.Context, routerID string, meta *command.CommandMeta) error
}

type PollHandler interface {
	Start(ctx context.Context, routerID string, meta *command.CommandMeta) error
}

type QueryHandler interface {
	Query(ctx context.Context, routerID string, meta *command.CommandMeta, filters ...string) ([]map[string]string, error)
	QueryOne(ctx context.Context, routerID string, meta *command.CommandMeta, filters ...string) (map[string]string, error)
	GetByIndex(ctx context.Context, routerID string, meta *command.CommandMeta, name string) (map[string]string, error)
	Count(ctx context.Context, routerID string, meta *command.CommandMeta) (int, error)
}

type MutationHandler interface {
	Execute(ctx context.Context, routerID string, meta *command.CommandMeta, args ...string) (*routeros.Reply, error)
}

type Dispatcher interface {
	Meta(path string) *command.CommandMeta
	Stream(ctx context.Context, routerID, path string) (context.CancelFunc, error)
	StopStream(routerID, path string)
	Query(ctx context.Context, routerID, path string, filters ...string) ([]map[string]string, error)
	Mutate(ctx context.Context, routerID, path string, args ...string) (*routeros.Reply, error)
}
