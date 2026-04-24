package service

import (
	"context"
	"fmt"
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

func (b *Bridge) GetCachedSnapshot(ctx context.Context, key string) (map[string]string, error) {
	if b.cache == nil {
		return nil, fmt.Errorf("cache not available")
	}
	return b.cache.GetSnapshot(ctx, key)
}
