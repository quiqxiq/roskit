package service

import (
	"context"
	"fmt"
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/core/model"
)

func (b *Bridge) ListInterfaces(ctx context.Context, routerID string) ([]map[string]string, error) {
	return b.Query(ctx, routerID, "interface/print")
}

func (b *Bridge) ListIPPools(ctx context.Context, routerID string) ([]map[string]string, error) {
	return b.Query(ctx, routerID, "ip/pool/print")
}

func (b *Bridge) ListNATRules(ctx context.Context, routerID string) ([]map[string]string, error) {
	return b.Query(ctx, routerID, "ip/firewall/nat/print")
}

func (b *Bridge) ListQueues(ctx context.Context, routerID string) ([]map[string]string, error) {
	return b.Query(ctx, routerID, "queue/simple/print")
}

func (b *Bridge) ListParentQueues(ctx context.Context, routerID string) ([]map[string]string, error) {
	return b.Query(ctx, routerID, "queue/simple/print", "?dynamic=false")
}

func (b *Bridge) FindARPByAddress(ctx context.Context, routerID, addr string) (map[string]string, error) {
	return b.QueryOne(ctx, routerID, "ip/arp/print", "?address="+addr)
}

func (b *Bridge) FindDHCPLeaseByAddress(ctx context.Context, routerID, addr string) (map[string]string, error) {
	return b.QueryOne(ctx, routerID, "ip/dhcp-server/lease/print", "?address="+addr)
}

func (b *Bridge) FindQueueByName(ctx context.Context, routerID, name string) (map[string]string, error) {
	return b.QueryOne(ctx, routerID, "queue/simple/print", "?name="+name)
}

func (b *Bridge) FindSchedulerByName(ctx context.Context, routerID, name string) (map[string]string, error) {
	return b.QueryOne(ctx, routerID, "system/scheduler/print", "?name="+name)
}

func (b *Bridge) FindScriptByName(ctx context.Context, routerID, name string) (map[string]string, error) {
	return b.QueryOne(ctx, routerID, "system/script/print", "?name="+name)
}

func (b *Bridge) ListDHCPLeases(ctx context.Context, routerID string) ([]map[string]string, error) {
	return b.Query(ctx, routerID, "ip/dhcp-server/lease/print")
}

// InterfaceTrafficStream opens a live /interface/monitor-traffic stream for iface.
// RouterOS sends one update per second with rx/tx counters.
// The returned channel closes when ctx is cancelled or the connection drops.
func (b *Bridge) InterfaceTrafficStream(ctx context.Context, routerID, iface string) (<-chan map[string]string, error) {
	reply, err := b.dispatcher.RunListen(ctx, routerID, []string{
		"/interface/monitor-traffic",
		"=interface=" + iface,
	})
	if err != nil {
		return nil, fmt.Errorf("interface traffic stream: %w", err)
	}

	out := make(chan map[string]string, 16)
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
				stats := model.NewInterfaceStatsFromReply(s.Map)
				if stats.Name == "" {
					stats.Name = iface
				}
				select {
				case out <- stats.ToCacheData(time.Now()):
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}

