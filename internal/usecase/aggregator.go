package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/internal/repository"
)

// InactiveAggregator maintains in-memory state of active vs total users
// to calculate and cache inactive users in raw real-time.
// It hooks into the Usecase layer and listens to parsed streams.
type InactiveAggregator struct {
	cache  repository.CacheRepository
	logger *slog.Logger

	mu sync.RWMutex

	// Nested maps to store state: [routerID][username]UserCacheData
	pppSecrets     map[string]map[string]map[string]string
	pppActives     map[string]map[string]struct{}
	hotspotUsers   map[string]map[string]map[string]string
	hotspotActives map[string]map[string]struct{}
}

// NewInactiveAggregator initializes a thread-safe aggregator.
func NewInactiveAggregator(cache repository.CacheRepository, logger *slog.Logger) *InactiveAggregator {
	return &InactiveAggregator{
		cache:          cache,
		logger:         logger,
		pppSecrets:     make(map[string]map[string]map[string]string),
		pppActives:     make(map[string]map[string]struct{}),
		hotspotUsers:   make(map[string]map[string]map[string]string),
		hotspotActives: make(map[string]map[string]struct{}),
	}
}

// EnsureRouter maps are initialized. Expects lock to be held.
func (a *InactiveAggregator) ensureRouter(routerID string) {
	if a.pppSecrets[routerID] == nil {
		a.pppSecrets[routerID] = make(map[string]map[string]string)
		a.pppActives[routerID] = make(map[string]struct{})
		a.hotspotUsers[routerID] = make(map[string]map[string]string)
		a.hotspotActives[routerID] = make(map[string]struct{})
	}
}

// Update feeds an event into the aggregator state machine.
// It recalculates inactive users only if relevant states changed.
func (a *InactiveAggregator) Update(ctx context.Context, event *domain.TelemetryEvent) {
	if event.Type != domain.EventUpdate && event.Type != domain.EventDead {
		return
	}

	a.mu.Lock()
	a.ensureRouter(event.RouterID)

	var changed bool
	var category string

	switch event.Measurement {
	case "ppp_secret": // Total configured PPP users
		name := event.CacheData["name"]
		if name == "" {
			name = event.Tags["name"]
		}
		if event.Type == domain.EventUpdate {
			a.pppSecrets[event.RouterID][name] = event.CacheData
		} else {
			delete(a.pppSecrets[event.RouterID], name)
		}
		category = "ppp"
		changed = true

	case "ppp_active": // Active PPP sessions
		name := event.Fields["name"].(string)
		if name == "" { // Fallback to tags if field mapping changes
			name = event.Tags["name"]
		}
		if event.Type == domain.EventUpdate {
			a.pppActives[event.RouterID][name] = struct{}{}
		} else {
			delete(a.pppActives[event.RouterID], name)
		}
		category = "ppp"
		changed = true

	case "hotspot_user": // Total configured Hotspot users
		name := event.CacheData["name"]
		if name == "" {
			name = event.Tags["name"]
		}
		if event.Type == domain.EventUpdate {
			a.hotspotUsers[event.RouterID][name] = event.CacheData
		} else {
			delete(a.hotspotUsers[event.RouterID], name)
		}
		category = "hotspot"
		changed = true

	case "hotspot_active": // Active Hotspot sessions
		user := event.Fields["user"].(string)
		if user == "" {
			user = event.Tags["user"]
		}
		if event.Type == domain.EventUpdate {
			a.hotspotActives[event.RouterID][user] = struct{}{}
		} else {
			delete(a.hotspotActives[event.RouterID], user)
		}
		category = "hotspot"
		changed = true
	}
	a.mu.Unlock()

	// If a relevant map changed, run calculation async so it doesn't block stream parsing.
	if changed {
		go a.recalculateAndCache(context.Background(), event.RouterID, category)
	}
}

// recalculateAndCache diffs total vs active to find inactive, then pushes to Redis.
func (a *InactiveAggregator) recalculateAndCache(ctx context.Context, routerID, category string) {
	a.mu.RLock()
	var totals map[string]map[string]string
	var actives map[string]struct{}
	var key string

	if category == "ppp" {
		totals = a.pppSecrets[routerID]
		actives = a.pppActives[routerID]
		key = fmt.Sprintf("roskit:%s:ppp_inactive", routerID)
	} else if category == "hotspot" {
		totals = a.hotspotUsers[routerID]
		actives = a.hotspotActives[routerID]
		key = fmt.Sprintf("roskit:%s:hotspot_inactive", routerID)
	} else {
		a.mu.RUnlock()
		return
	}

	// Calculate inactive (Users in total BUT NOT in actives).
	// We build a map[string]string because Redis HSET expects field-value pairs.
	// Field: username format, Value: JSON of the user configuration block.
	inactiveData := make(map[string]string)
	for username, data := range totals {
		if _, isActive := actives[username]; !isActive {
			// They are inactive, serialize their config payload.
			jsonBytes, err := json.Marshal(data)
			if err == nil {
				inactiveData[username] = string(jsonBytes)
			}
		}
	}
	a.mu.RUnlock()

	// Important limitation: we cannot just pass all to HSET if we want to remove
	// users who became active. We should replace the entire hash or calculate diffs.
	// Since inactive lists can be large, wiping and re-writing completely or doing HDELs is tricky.
	// The safest pattern is to override the Hash completely, so old fields are removed.
	// We'll delete the entire key first, then HSET the exact list in an atomic transaction-like way,
	// or we can just send it as a JSON string under a summary key.
	// We will use one cache key holding a single field "list" with the JSON array for easy frontend consumption.

	var payload []map[string]string
	for _, v := range inactiveData {
		var item map[string]string
		_ = json.Unmarshal([]byte(v), &item)
		payload = append(payload, item)
	}

	jsonPayload, _ := json.Marshal(payload)
	finalData := map[string]string{
		"list":      string(jsonPayload),
		"count":     fmt.Sprintf("%d", len(payload)),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	if err := a.cache.SetSnapshot(ctx, key, finalData); err != nil {
		a.logger.Error("failed to set inactive aggregator snapshot", "category", category, "error", err)
	} else {
		a.logger.Debug("inactive users recalculated", "category", category, "inactive_count", len(payload))
	}
}
