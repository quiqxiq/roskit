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

func CleanupPPPSecrets(ctx context.Context, bridge *service.Bridge, routerID, prefix string) {
	secrets, err := bridge.ListPPPSecrets(ctx, routerID)
	if err != nil {
		return
	}
	for _, s := range secrets {
		name := s["name"]
		if strings.HasPrefix(name, prefix) {
			bridge.RemovePPPSecret(ctx, routerID, s[".id"])
		}
	}
}

func CleanupPPPProfiles(ctx context.Context, bridge *service.Bridge, routerID, prefix string) {
	profiles, err := bridge.ListPPPProfiles(ctx, routerID)
	if err != nil {
		return
	}
	for _, p := range profiles {
		name := p["name"]
		if strings.HasPrefix(name, prefix) {
			bridge.RemovePPPProfile(ctx, routerID, p[".id"])
		}
	}
}

func CleanupIPBindings(ctx context.Context, bridge *service.Bridge, routerID, prefix string) {
	bindings, err := bridge.ListIPBindings(ctx, routerID)
	if err != nil {
		return
	}
	for _, b := range bindings {
		comment := b["comment"]
		if strings.HasPrefix(comment, prefix) {
			bridge.RemoveIPBinding(ctx, routerID, b[".id"])
		}
	}
}

func CleanupWalledGarden(ctx context.Context, bridge *service.Bridge, routerID, prefix string) {
	wg, err := bridge.ListWalledGarden(ctx, routerID)
	if err != nil {
		return
	}
	for _, w := range wg {
		comment := w["comment"]
		if strings.HasPrefix(comment, prefix) {
			bridge.RemoveWalledGarden(ctx, routerID, w[".id"])
		}
	}
}

func CleanupQueues(ctx context.Context, bridge *service.Bridge, routerID, prefix string) {
	queues, err := bridge.ListQueues(ctx, routerID)
	if err != nil {
		return
	}
	for _, q := range queues {
		name := q["name"]
		if strings.HasPrefix(name, prefix) {
			bridge.Run(ctx, routerID, "/queue/simple/remove", "=.id="+q[".id"])
		}
	}
}

func CleanupAll(ctx context.Context, bridge *service.Bridge, routerID, prefix string) {
	CleanupHotspotUsers(ctx, bridge, routerID, prefix)
	CleanupHotspotProfiles(ctx, bridge, routerID, prefix)
	CleanupSchedulers(ctx, bridge, routerID, prefix)
	CleanupScripts(ctx, bridge, routerID, prefix)
	CleanupPPPSecrets(ctx, bridge, routerID, prefix)
	CleanupPPPProfiles(ctx, bridge, routerID, prefix)
	CleanupIPBindings(ctx, bridge, routerID, prefix)
	CleanupWalledGarden(ctx, bridge, routerID, prefix)
	CleanupQueues(ctx, bridge, routerID, prefix)
}
