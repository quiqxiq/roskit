package service

import "context"

func (b *Bridge) GetSystemResource(ctx context.Context, routerID string) (map[string]string, error) {
	return b.QueryOne(ctx, routerID, "system/resource/print")
}

func (b *Bridge) GetSystemIdentity(ctx context.Context, routerID string) (map[string]string, error) {
	return b.QueryOne(ctx, routerID, "system/identity/print")
}

func (b *Bridge) GetSystemClock(ctx context.Context, routerID string) (map[string]string, error) {
	return b.QueryOne(ctx, routerID, "system/clock/print")
}

func (b *Bridge) GetSystemHealth(ctx context.Context, routerID string) (map[string]string, error) {
	return b.QueryOne(ctx, routerID, "system/health/print")
}

func (b *Bridge) GetRouterboard(ctx context.Context, routerID string) (map[string]string, error) {
	return b.QueryOne(ctx, routerID, "system/routerboard/print")
}

func (b *Bridge) GetSystemLog(ctx context.Context, routerID string, topics string) ([]map[string]string, error) {
	if topics != "" {
		return b.Query(ctx, routerID, "system/log/print", "?topics="+topics)
	}
	return b.Query(ctx, routerID, "system/log/print")
}

func (b *Bridge) ListSchedulers(ctx context.Context, routerID string) ([]map[string]string, error) {
	return b.Query(ctx, routerID, "system/scheduler/print")
}

func (b *Bridge) ListScripts(ctx context.Context, routerID string) ([]map[string]string, error) {
	return b.Query(ctx, routerID, "system/script/print")
}

func (b *Bridge) Reboot(ctx context.Context, routerID string) error {
	_, err := b.Mutate(ctx, routerID, "system/reboot")
	return err
}

func (b *Bridge) Shutdown(ctx context.Context, routerID string) error {
	_, err := b.Mutate(ctx, routerID, "system/shutdown")
	return err
}

func (b *Bridge) SetupLogging(ctx context.Context, routerID string) error {
	existing, err := b.Query(ctx, routerID, "system/logging/print", "?prefix=->")
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return nil
	}
	_, err = b.mutateAdd(ctx, routerID, "system/logging/add", map[string]string{
		"action": "echo",
		"prefix": "->",
		"topics": "hotspot,debug,info",
	})
	return err
}
