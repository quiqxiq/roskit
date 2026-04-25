package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/behavior"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/cache"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/pubsub"
)

type InactiveAggregator struct {
	cache    cache.Repository
	pubsub   pubsub.Publisher
	logger   *slog.Logger

	mu sync.RWMutex

	pppSecrets     map[string]map[string]map[string]string
	pppActives     map[string]map[string]struct{}
	hotspotUsers   map[string]map[string]map[string]string
	hotspotActives map[string]map[string]struct{}
}

func NewInactiveAggregator(
	c cache.Repository,
	ps pubsub.Publisher,
	logger *slog.Logger,
) *InactiveAggregator {
	if logger == nil {
		logger = slog.Default()
	}
	return &InactiveAggregator{
		cache:          c,
		pubsub:         ps,
		logger:         logger,
		pppSecrets:     make(map[string]map[string]map[string]string),
		pppActives:     make(map[string]map[string]struct{}),
		hotspotUsers:   make(map[string]map[string]map[string]string),
		hotspotActives: make(map[string]map[string]struct{}),
	}
}

func (a *InactiveAggregator) OnEvent(ctx context.Context, event behavior.StreamEvent) error {
	a.mu.Lock()
	a.ensureRouter(event.RouterID)

	var changed bool
	var category string

	switch event.Meta.Measurement {
	case "ppp_secret":
		name := event.Fields["name"]
		if event.IsDead {
			delete(a.pppSecrets[event.RouterID], name)
		} else {
			a.pppSecrets[event.RouterID][name] = event.Fields
		}
		category = "ppp"
		changed = true

	case "ppp_active":
		name := event.Fields["name"]
		if event.IsDead {
			delete(a.pppActives[event.RouterID], name)
		} else {
			a.pppActives[event.RouterID][name] = struct{}{}
		}
		category = "ppp"
		changed = true

	case "hotspot_user":
		name := event.Fields["name"]
		if event.IsDead {
			delete(a.hotspotUsers[event.RouterID], name)
		} else {
			a.hotspotUsers[event.RouterID][name] = event.Fields
		}
		category = "hotspot"
		changed = true

	case "hotspot_active":
		user := event.Fields["user"]
		if event.IsDead {
			delete(a.hotspotActives[event.RouterID], user)
		} else {
			a.hotspotActives[event.RouterID][user] = struct{}{}
		}
		category = "hotspot"
		changed = true
	}
	a.mu.Unlock()

	if changed {
		go a.recalculateAndCache(context.Background(), event.RouterID, category)
	}

	return nil
}

func (a *InactiveAggregator) ensureRouter(routerID string) {
	if a.pppSecrets[routerID] == nil {
		a.pppSecrets[routerID] = make(map[string]map[string]string)
		a.pppActives[routerID] = make(map[string]struct{})
		a.hotspotUsers[routerID] = make(map[string]map[string]string)
		a.hotspotActives[routerID] = make(map[string]struct{})
	}
}

func (a *InactiveAggregator) recalculateAndCache(ctx context.Context, routerID, category string) {
	a.mu.RLock()
	var totals map[string]map[string]string
	var actives map[string]struct{}
	var key string

	switch category {
	case "ppp":
		totals = a.pppSecrets[routerID]
		actives = a.pppActives[routerID]
		key = fmt.Sprintf("roskit:%s:ppp_inactive", routerID)
	case "hotspot":
		totals = a.hotspotUsers[routerID]
		actives = a.hotspotActives[routerID]
		key = fmt.Sprintf("roskit:%s:hotspot_inactive", routerID)
	default:
		a.mu.RUnlock()
		return
	}

	var payload []map[string]string
	for username, data := range totals {
		if _, isActive := actives[username]; !isActive {
			payload = append(payload, data)
		}
	}
	a.mu.RUnlock()

	jsonPayload, _ := json.Marshal(payload)
	finalData := map[string]string{
		"list":      string(jsonPayload),
		"count":     fmt.Sprintf("%d", len(payload)),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	if err := a.cache.SetSnapshot(ctx, key, finalData, 0); err != nil {
		a.logger.Error("aggregator: cache write failed",
			"category", category, "router_id", routerID, "err", err)
	} else {
		a.logger.Debug("aggregator: inactive recalculated",
			"category", category, "router_id", routerID, "count", len(payload))
	}

	channel := cache.FormatPubSubChannel(routerID)
	a.pubsub.Publish(ctx, channel, pubsub.Message{
		Type:        "update",
		RouterID:    routerID,
		Measurement: category + "_inactive",
		Fields: map[string]string{
			"count": fmt.Sprintf("%d", len(payload)),
		},
		Timestamp: time.Now(),
	})
}

func (a *InactiveAggregator) GetInactiveHotspotUsers(routerID string) []map[string]string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	users := a.hotspotUsers[routerID]
	actives := a.hotspotActives[routerID]
	var inactive []map[string]string
	for name, data := range users {
		if _, isActive := actives[name]; !isActive {
			inactive = append(inactive, data)
		}
	}
	return inactive
}

func (a *InactiveAggregator) GetInactivePPPSecrets(routerID string) []map[string]string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	secrets := a.pppSecrets[routerID]
	actives := a.pppActives[routerID]
	var inactive []map[string]string
	for name, data := range secrets {
		if _, isActive := actives[name]; !isActive {
			inactive = append(inactive, data)
		}
	}
	return inactive
}

var _ behavior.StreamSink = (*InactiveAggregator)(nil)
