package service

import (
	"context"
	"fmt"
)

func (b *Bridge) ListPPPSecrets(ctx context.Context, routerID string) ([]map[string]string, error) {
	return b.Query(ctx, routerID, "ppp/secret/print")
}

func (b *Bridge) GetPPPSecret(ctx context.Context, routerID, idOrName string) (map[string]string, error) {
	if len(idOrName) > 0 && idOrName[:1] == "*" {
		return b.QueryOne(ctx, routerID, "ppp/secret/print", "?.id="+idOrName)
	}
	return b.QueryOne(ctx, routerID, "ppp/secret/print", "?name="+idOrName)
}

func (b *Bridge) AddPPPSecret(ctx context.Context, routerID string, params map[string]string) (string, error) {
	return b.mutateAdd(ctx, routerID, "ppp/secret/add", params)
}

func (b *Bridge) SetPPPSecret(ctx context.Context, routerID, id string, params map[string]string) error {
	args := []string{"=.id=" + id}
	for k, v := range params {
		args = append(args, "="+k+"="+v)
	}
	_, err := b.Mutate(ctx, routerID, "ppp/secret/set", args...)
	return err
}

func (b *Bridge) RemovePPPSecret(ctx context.Context, routerID, id string) error {
	_, err := b.Mutate(ctx, routerID, "ppp/secret/remove", "=.id="+id)
	return err
}

func (b *Bridge) EnablePPPSecret(ctx context.Context, routerID, id string) error {
	_, err := b.Mutate(ctx, routerID, "ppp/secret/enable", "=numbers="+id)
	return err
}

func (b *Bridge) DisablePPPSecret(ctx context.Context, routerID, id string) error {
	_, err := b.Mutate(ctx, routerID, "ppp/secret/disable", "=numbers="+id)
	return err
}

func (b *Bridge) ListPPPActive(ctx context.Context, routerID string) ([]map[string]string, error) {
	return b.Query(ctx, routerID, "ppp/active/print")
}

func (b *Bridge) GetPPPActive(ctx context.Context, routerID, id string) (map[string]string, error) {
	return b.QueryOne(ctx, routerID, "ppp/active/print", "?.id="+id)
}

func (b *Bridge) RemovePPPActive(ctx context.Context, routerID, id string) error {
	_, err := b.Mutate(ctx, routerID, "ppp/active/remove", "=.id="+id)
	return err
}

func (b *Bridge) ListPPPProfiles(ctx context.Context, routerID string) ([]map[string]string, error) {
	return b.Query(ctx, routerID, "ppp/profile/print")
}

func (b *Bridge) GetPPPProfile(ctx context.Context, routerID, idOrName string) (map[string]string, error) {
	if len(idOrName) > 0 && idOrName[:1] == "*" {
		return b.QueryOne(ctx, routerID, "ppp/profile/print", "?.id="+idOrName)
	}
	return b.QueryOne(ctx, routerID, "ppp/profile/print", "?name="+idOrName)
}

func (b *Bridge) AddPPPProfile(ctx context.Context, routerID string, params map[string]string) (string, error) {
	return b.mutateAdd(ctx, routerID, "ppp/profile/add", params)
}

func (b *Bridge) SetPPPProfile(ctx context.Context, routerID, id string, params map[string]string) error {
	args := []string{"=.id=" + id}
	for k, v := range params {
		args = append(args, "="+k+"="+v)
	}
	_, err := b.Mutate(ctx, routerID, "ppp/profile/set", args...)
	return err
}

func (b *Bridge) RemovePPPProfile(ctx context.Context, routerID, id string) error {
	_, err := b.Mutate(ctx, routerID, "ppp/profile/remove", "=.id="+id)
	return err
}

func (b *Bridge) EnablePPPProfile(ctx context.Context, routerID, id string) error {
	_, err := b.Mutate(ctx, routerID, "ppp/profile/enable", "=numbers="+id)
	return err
}

func (b *Bridge) DisablePPPProfile(ctx context.Context, routerID, id string) error {
	_, err := b.Mutate(ctx, routerID, "ppp/profile/disable", "=numbers="+id)
	return err
}

func (b *Bridge) ListInactivePPPSecrets(ctx context.Context, routerID string) ([]map[string]string, error) {
	allSecrets, err := b.ListPPPSecrets(ctx, routerID)
	if err != nil {
		return nil, err
	}
	if len(allSecrets) == 0 {
		return []map[string]string{}, nil
	}

	activeSessions, err := b.ListPPPActive(ctx, routerID)
	if err != nil {
		return nil, err
	}

	activeMap := make(map[string]struct{}, len(activeSessions))
	for _, sess := range activeSessions {
		name := sess["name"]
		if name != "" {
			activeMap[name] = struct{}{}
		}
	}

	var inactive []map[string]string
	for _, secret := range allSecrets {
		name := secret["name"]
		if _, isActive := activeMap[name]; !isActive {
			inactive = append(inactive, secret)
		}
	}
	return inactive, nil
}

func (b *Bridge) GetInactivePPPSecretCount(ctx context.Context, routerID string) (int, error) {
	if b.cache != nil {
		key := fmt.Sprintf("roskit:%s:ppp_inactive", routerID)
		data, err := b.cache.GetSnapshot(ctx, key)
		if err == nil && data != nil && data["count"] != "" {
			var count int
			if _, err := fmt.Sscanf(data["count"], "%d", &count); err == nil {
				return count, nil
			}
		}
	}

	list, err := b.ListInactivePPPSecrets(ctx, routerID)
	if err != nil {
		return 0, err
	}
	return len(list), nil
}
