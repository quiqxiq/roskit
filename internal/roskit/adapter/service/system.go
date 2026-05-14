package service

import (
	"context"
	"fmt"
	"time"
)

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
		return b.Query(ctx, routerID, "log/print", "?topics="+topics)
	}
	return b.Query(ctx, routerID, "log/print")
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

func (b *Bridge) EnableScheduler(ctx context.Context, routerID, id string) error {
	_, err := b.Mutate(ctx, routerID, "system/scheduler/enable", "=numbers="+id)
	return err
}

func (b *Bridge) DisableScheduler(ctx context.Context, routerID, id string) error {
	_, err := b.Mutate(ctx, routerID, "system/scheduler/disable", "=numbers="+id)
	return err
}

func (b *Bridge) AddScheduler(ctx context.Context, routerID string, params map[string]string) (string, error) {
	return b.mutateAdd(ctx, routerID, "system/scheduler/add", params)
}

func (b *Bridge) SetScheduler(ctx context.Context, routerID string, id string, params map[string]string) error {
	args := []string{"=.id=" + id}
	for k, v := range params {
		args = append(args, "="+k+"="+v)
	}
	_, err := b.Mutate(ctx, routerID, "system/scheduler/set", args...)
	return err
}

func (b *Bridge) RemoveScheduler(ctx context.Context, routerID string, id string) error {
	_, err := b.Mutate(ctx, routerID, "system/scheduler/remove", "=.id="+id)
	return err
}

func (b *Bridge) AddScript(ctx context.Context, routerID string, params map[string]string) (string, error) {
	return b.mutateAdd(ctx, routerID, "system/script/add", params)
}

func (b *Bridge) SetScript(ctx context.Context, routerID string, id string, params map[string]string) error {
	args := []string{"=.id=" + id}
	for k, v := range params {
		args = append(args, "="+k+"="+v)
	}
	_, err := b.Mutate(ctx, routerID, "system/script/set", args...)
	return err
}

func (b *Bridge) RemoveScript(ctx context.Context, routerID string, id string) error {
	_, err := b.Mutate(ctx, routerID, "system/script/remove", "=.id="+id)
	return err
}

func (b *Bridge) RunScript(ctx context.Context, routerID string, id string) error {
	_, err := b.Mutate(ctx, routerID, "system/script/run", "=.id="+id)
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

// LogStream opens a live /log/print =follow stream against routerID.
// filter selects a topic group: "" = all, "hotspot", "ppp".
// The returned channel closes when ctx is cancelled or the connection drops.
func (b *Bridge) LogStream(ctx context.Context, routerID, filter string) (<-chan map[string]string, error) {
	sentence := []string{"/log/print", "=follow"}
	switch filter {
	case "hotspot":
		sentence = append(sentence, "?topics=hotspot,info,debug")
	case "ppp":
		sentence = append(sentence, "?topics=pppoe,info,debug")
	}

	reply, err := b.dispatcher.RunListen(ctx, routerID, sentence)
	if err != nil {
		return nil, fmt.Errorf("log stream: %w", err)
	}

	out := make(chan map[string]string, 32)
	go func() {
		defer close(out)
		defer func() {
			stopCtx, stopFn := context.WithTimeout(context.Background(), 3*time.Second)
			defer stopFn()
			reply.CancelContext(stopCtx)
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case s, ok := <-reply.Chan():
				if !ok {
					return
				}
				select {
				case out <- s.Map:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}
