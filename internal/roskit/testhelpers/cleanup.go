//go:build mikrotik

package testhelpers

import (
	"context"
	"strings"

	"github.com/quiqxiq/roskit/internal/roskit/adapter/service"
)

func CleanupHotspotUsers(ctx context.Context, bridge *service.Bridge, routerID, prefix string) {
	users, err := bridge.ListHotspotUsers(ctx, routerID, "")
	if err != nil {
		return
	}
	for _, u := range users {
		name := u["name"]
		if strings.HasPrefix(name, prefix) {
			bridge.RemoveHotspotUser(ctx, routerID, u[".id"])
		}
	}
}

func CleanupHotspotProfiles(ctx context.Context, bridge *service.Bridge, routerID, prefix string) {
	profiles, err := bridge.ListHotspotProfiles(ctx, routerID)
	if err != nil {
		return
	}
	for _, p := range profiles {
		name := p["name"]
		if strings.HasPrefix(name, prefix) {
			bridge.RemoveHotspotProfile(ctx, routerID, p[".id"])
		}
	}
}

func CleanupSchedulers(ctx context.Context, bridge *service.Bridge, routerID, prefix string) {
	schedulers, err := bridge.ListSchedulers(ctx, routerID)
	if err != nil {
		return
	}
	for _, s := range schedulers {
		name := s["name"]
		if strings.HasPrefix(name, prefix) {
			bridge.RemoveScheduler(ctx, routerID, s[".id"])
		}
	}
}

func CleanupScripts(ctx context.Context, bridge *service.Bridge, routerID, prefix string) {
	scripts, err := bridge.ListScripts(ctx, routerID)
	if err != nil {
		return
	}
	for _, s := range scripts {
		name := s["name"]
		if strings.HasPrefix(name, prefix) {
			bridge.RemoveScript(ctx, routerID, s[".id"])
		}
	}
}

func CleanupAll(ctx context.Context, bridge *service.Bridge, routerID, prefix string) {
	CleanupHotspotUsers(ctx, bridge, routerID, prefix)
	CleanupHotspotProfiles(ctx, bridge, routerID, prefix)
	CleanupSchedulers(ctx, bridge, routerID, prefix)
	CleanupScripts(ctx, bridge, routerID, prefix)
}
