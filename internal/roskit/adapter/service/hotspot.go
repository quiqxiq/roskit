package service

import (
	"context"
	"fmt"
)

func (b *Bridge) ListHotspotUsers(ctx context.Context, routerID, profile string) ([]map[string]string, error) {
	filters := []string{}
	if profile != "" {
		filters = append(filters, "?profile="+profile)
	}
	return b.Query(ctx, routerID, "ip/hotspot/user/print", filters...)
}

func (b *Bridge) GetHotspotUser(ctx context.Context, routerID, idOrName string) (map[string]string, error) {
	if len(idOrName) > 0 && idOrName[:1] == "*" {
		return b.QueryOne(ctx, routerID, "ip/hotspot/user/print", "?.id="+idOrName)
	}
	return b.QueryOne(ctx, routerID, "ip/hotspot/user/print", "?name="+idOrName)
}

func (b *Bridge) GetHotspotUserCount(ctx context.Context, routerID string) (int, error) {
	users, err := b.Query(ctx, routerID, "ip/hotspot/user/print")
	if err != nil {
		return 0, err
	}
	count := len(users)
	if count > 1 {
		return count - 1, nil
	}
	return 0, nil
}

func (b *Bridge) AddHotspotUser(ctx context.Context, routerID string, params map[string]string) (string, error) {
	return b.mutateAdd(ctx, routerID, "ip/hotspot/user/add", params)
}

func (b *Bridge) RemoveHotspotUser(ctx context.Context, routerID, id string) error {
	_, err := b.Mutate(ctx, routerID, "ip/hotspot/user/remove", "=.id="+id)
	return err
}

func (b *Bridge) SetHotspotUser(ctx context.Context, routerID, id string, params map[string]string) error {
	args := []string{"=.id=" + id}
	for k, v := range params {
		args = append(args, "="+k+"="+v)
	}
	_, err := b.Mutate(ctx, routerID, "ip/hotspot/user/set", args...)
	return err
}

func (b *Bridge) EnableHotspotUser(ctx context.Context, routerID, id string) error {
	_, err := b.Mutate(ctx, routerID, "ip/hotspot/user/enable", "=numbers="+id)
	return err
}

func (b *Bridge) DisableHotspotUser(ctx context.Context, routerID, id string) error {
	_, err := b.Mutate(ctx, routerID, "ip/hotspot/user/disable", "=numbers="+id)
	return err
}

func (b *Bridge) ResetUserCounters(ctx context.Context, routerID, id string) error {
	_, err := b.Mutate(ctx, routerID, "ip/hotspot/user/reset-counters", "=.id="+id)
	return err
}

func (b *Bridge) RemoveHotspotUserWithCleanup(ctx context.Context, routerID, id string) error {
	data, err := b.GetHotspotUser(ctx, routerID, id)
	if err == nil && len(data) > 0 {
		user := data["name"]
		if user != "" {
			script, err := b.FindScriptByName(ctx, routerID, user)
			if err == nil && script[".id"] != "" {
				b.Mutate(ctx, routerID, "system/script/remove", "=.id="+script[".id"])
			}
			scheduler, err := b.FindSchedulerByName(ctx, routerID, user)
			if err == nil && scheduler[".id"] != "" {
				b.Mutate(ctx, routerID, "system/scheduler/remove", "=.id="+scheduler[".id"])
			}
		}
	}
	_, err = b.Mutate(ctx, routerID, "ip/hotspot/user/remove", "=.id="+id)
	return err
}

func (b *Bridge) ListHotspotActive(ctx context.Context, routerID string) ([]map[string]string, error) {
	return b.Query(ctx, routerID, "ip/hotspot/active/print")
}

func (b *Bridge) GetActiveSession(ctx context.Context, routerID, id string) (map[string]string, error) {
	return b.QueryOne(ctx, routerID, "ip/hotspot/active/print", "?.id="+id)
}

func (b *Bridge) GetActiveSessionCount(ctx context.Context, routerID string) (int, error) {
	sessions, err := b.Query(ctx, routerID, "ip/hotspot/active/print")
	if err != nil {
		return 0, err
	}
	return len(sessions), nil
}

func (b *Bridge) RemoveHotspotActive(ctx context.Context, routerID, id string) error {
	_, err := b.Mutate(ctx, routerID, "ip/hotspot/active/remove", "=.id="+id)
	return err
}

func (b *Bridge) DisconnectUser(ctx context.Context, routerID, activeID string) error {
	data, err := b.GetActiveSession(ctx, routerID, activeID)
	if err != nil {
		return err
	}
	user := data["user"]

	cookies, err := b.Query(ctx, routerID, "ip/hotspot/cookie/print")
	if err == nil {
		for _, c := range cookies {
			if c["user"] == user && c[".id"] != "" {
				b.Mutate(ctx, routerID, "ip/hotspot/cookie/remove", "=.id="+c[".id"])
				break
			}
		}
	}

	return b.RemoveHotspotActive(ctx, routerID, activeID)
}

func (b *Bridge) GetHotspotProfile(ctx context.Context, routerID, idOrName string) (map[string]string, error) {
	if len(idOrName) > 0 && idOrName[:1] == "*" {
		return b.QueryOne(ctx, routerID, "ip/hotspot/user/profile/print", "?.id="+idOrName)
	}
	return b.QueryOne(ctx, routerID, "ip/hotspot/user/profile/print", "?name="+idOrName)
}

func (b *Bridge) ListHotspotProfiles(ctx context.Context, routerID string) ([]map[string]string, error) {
	return b.Query(ctx, routerID, "ip/hotspot/user/profile/print")
}

func (b *Bridge) AddHotspotProfile(ctx context.Context, routerID string, params map[string]string) (string, error) {
	return b.mutateAdd(ctx, routerID, "ip/hotspot/user/profile/add", params)
}

func (b *Bridge) SetHotspotProfile(ctx context.Context, routerID, id string, params map[string]string) error {
	args := []string{"=.id=" + id}
	for k, v := range params {
		args = append(args, "="+k+"="+v)
	}
	_, err := b.Mutate(ctx, routerID, "ip/hotspot/user/profile/set", args...)
	return err
}

func (b *Bridge) RemoveHotspotProfile(ctx context.Context, routerID, id string) error {
	_, err := b.Mutate(ctx, routerID, "ip/hotspot/user/profile/remove", "=.id="+id)
	return err
}

func (b *Bridge) RemoveHotspotProfileWithCleanup(ctx context.Context, routerID, id, profileName string) error {
	scheduler, err := b.FindSchedulerByName(ctx, routerID, profileName)
	if err == nil && scheduler[".id"] != "" {
		b.Mutate(ctx, routerID, "system/scheduler/remove", "=.id="+scheduler[".id"])
	}
	_, err = b.Mutate(ctx, routerID, "ip/hotspot/user/profile/remove", "=.id="+id)
	return err
}

func (b *Bridge) ListHotspotServers(ctx context.Context, routerID string) ([]map[string]string, error) {
	return b.Query(ctx, routerID, "ip/hotspot/print")
}

func (b *Bridge) ListHosts(ctx context.Context, routerID string) ([]map[string]string, error) {
	return b.Query(ctx, routerID, "ip/hotspot/host/print")
}

func (b *Bridge) ListCookies(ctx context.Context, routerID string) ([]map[string]string, error) {
	return b.Query(ctx, routerID, "ip/hotspot/cookie/print")
}

func (b *Bridge) ListIPBindings(ctx context.Context, routerID string) ([]map[string]string, error) {
	return b.Query(ctx, routerID, "ip/hotspot/ip-binding/print")
}

func (b *Bridge) AddIPBinding(ctx context.Context, routerID string, params map[string]string) (string, error) {
	return b.mutateAdd(ctx, routerID, "ip/hotspot/ip-binding/add", params)
}

func (b *Bridge) RemoveIPBinding(ctx context.Context, routerID, id string) error {
	_, err := b.Mutate(ctx, routerID, "ip/hotspot/ip-binding/remove", "=.id="+id)
	return err
}

func (b *Bridge) RemoveIPBindingWithCleanup(ctx context.Context, routerID, id, mac, addr string) error {
	if err := b.RemoveIPBinding(ctx, routerID, id); err != nil {
		return err
	}

	if mac != "" {
		queue, err := b.FindQueueByName(ctx, routerID, mac)
		if err == nil && queue[".id"] != "" {
			b.Mutate(ctx, routerID, "queue/simple/remove", "=.id="+queue[".id"])
		}
		scheduler, err := b.FindSchedulerByName(ctx, routerID, mac)
		if err == nil && scheduler[".id"] != "" {
			b.Mutate(ctx, routerID, "system/scheduler/remove", "=.id="+scheduler[".id"])
		}
	}

	if addr != "" {
		arp, err := b.FindARPByAddress(ctx, routerID, addr)
		if err == nil && arp[".id"] != "" {
			b.Mutate(ctx, routerID, "ip/arp/remove", "=.id="+arp[".id"])
		}
		lease, err := b.FindDHCPLeaseByAddress(ctx, routerID, addr)
		if err == nil && lease[".id"] != "" {
			b.Mutate(ctx, routerID, "ip/dhcp-server/lease/remove", "=.id="+lease[".id"])
		}
	}

	return nil
}

func (b *Bridge) EnableIPBinding(ctx context.Context, routerID, id string) error {
	args := []string{"=.id=" + id, "=disabled=no"}
	_, err := b.Mutate(ctx, routerID, "ip/hotspot/ip-binding/set", args...)
	return err
}

func (b *Bridge) DisableIPBinding(ctx context.Context, routerID, id string) error {
	args := []string{"=.id=" + id, "=disabled=yes"}
	_, err := b.Mutate(ctx, routerID, "ip/hotspot/ip-binding/set", args...)
	return err
}

func (b *Bridge) ListInactiveHotspotUsers(ctx context.Context, routerID string) ([]map[string]string, error) {
	allUsers, err := b.ListHotspotUsers(ctx, routerID, "")
	if err != nil {
		return nil, err
	}
	if len(allUsers) == 0 {
		return []map[string]string{}, nil
	}

	activeSessions, err := b.ListHotspotActive(ctx, routerID)
	if err != nil {
		return nil, err
	}

	activeMap := make(map[string]struct{}, len(activeSessions))
	for _, sess := range activeSessions {
		user := sess["user"]
		if user == "" {
			user = sess["name"]
		}
		if user != "" {
			activeMap[user] = struct{}{}
		}
	}

	var inactive []map[string]string
	for _, user := range allUsers {
		name := user["name"]
		if _, isActive := activeMap[name]; !isActive {
			inactive = append(inactive, user)
		}
	}
	return inactive, nil
}

func (b *Bridge) GetInactiveHotspotUserCount(ctx context.Context, routerID string) (int, error) {
	if b.cache != nil {
		key := fmt.Sprintf("roskit:%s:hotspot_inactive", routerID)
		data, err := b.cache.GetSnapshot(ctx, key)
		if err == nil && data != nil && data["count"] != "" {
			var count int
			if _, err := fmt.Sscanf(data["count"], "%d", &count); err == nil {
				return count, nil
			}
		}
	}

	users, err := b.ListInactiveHotspotUsers(ctx, routerID)
	if err != nil {
		return 0, err
	}
	return len(users), nil
}

func (b *Bridge) RemoveHost(ctx context.Context, routerID, id string) error {
	_, err := b.Mutate(ctx, routerID, "ip/hotspot/host/remove", "=.id="+id)
	return err
}

func (b *Bridge) RemoveCookie(ctx context.Context, routerID, id string) error {
	_, err := b.Mutate(ctx, routerID, "ip/hotspot/cookie/remove", "=.id="+id)
	return err
}

func (b *Bridge) SetIPBinding(ctx context.Context, routerID, id string, params map[string]string) error {
	args := []string{"=.id=" + id}
	for k, v := range params {
		args = append(args, "="+k+"="+v)
	}
	_, err := b.Mutate(ctx, routerID, "ip/hotspot/ip-binding/set", args...)
	return err
}

func (b *Bridge) ListWalledGarden(ctx context.Context, routerID string) ([]map[string]string, error) {
	return b.Query(ctx, routerID, "ip/hotspot/walled-garden/print")
}

func (b *Bridge) AddWalledGarden(ctx context.Context, routerID string, params map[string]string) (string, error) {
	return b.mutateAdd(ctx, routerID, "ip/hotspot/walled-garden/add", params)
}

func (b *Bridge) SetWalledGarden(ctx context.Context, routerID string, id string, params map[string]string) error {
	args := []string{"=.id=" + id}
	for k, v := range params {
		args = append(args, "="+k+"="+v)
	}
	_, err := b.Mutate(ctx, routerID, "ip/hotspot/walled-garden/set", args...)
	return err
}

func (b *Bridge) RemoveWalledGarden(ctx context.Context, routerID string, id string) error {
	_, err := b.Mutate(ctx, routerID, "ip/hotspot/walled-garden/remove", "=.id="+id)
	return err
}

func (b *Bridge) ListWalledGardenIP(ctx context.Context, routerID string) ([]map[string]string, error) {
	return b.Query(ctx, routerID, "ip/hotspot/walled-garden/ip/print")
}

func (b *Bridge) AddWalledGardenIP(ctx context.Context, routerID string, params map[string]string) (string, error) {
	return b.mutateAdd(ctx, routerID, "ip/hotspot/walled-garden/ip/add", params)
}

func (b *Bridge) RemoveWalledGardenIP(ctx context.Context, routerID string, id string) error {
	_, err := b.Mutate(ctx, routerID, "ip/hotspot/walled-garden/ip/remove", "=.id="+id)
	return err
}
