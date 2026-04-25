package orchestrator

import (
	"context"

	"github.com/quiqxiq/roskit/internal/roskit/execution"
)

type RouterManager struct {
	engine *Engine
}

func NewRouterManager(engine *Engine) *RouterManager {
	return &RouterManager{engine: engine}
}

func (rm *RouterManager) AddRouter(ctx context.Context, cfg execution.ConnConfig) error {
	return rm.engine.AddRouter(ctx, cfg)
}

func (rm *RouterManager) RemoveRouter(routerID string) {
	rm.engine.RemoveRouter(routerID)
}

func (rm *RouterManager) Status() map[string]string {
	return rm.engine.Status()
}
