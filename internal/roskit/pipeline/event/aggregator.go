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

const aggregatorCacheTTL = 10 * time.Minute
const aggregatorDebounce = 100 * time.Millisecond

type routerAggState struct {
	pppSecrets     map[string]struct{}
	pppActives     map[string]struct{}
	hotspotUsers   map[string]struct{}
	hotspotActives map[string]struct{}

	pppDirty     bool
	hotspotDirty bool
	pppTimer     *time.Timer
	hotspotTimer *time.Timer
}

func newRouterAggState() *routerAggState {
	return &routerAggState{
		pppSecrets:     make(map[string]struct{}),
		pppActives:     make(map[string]struct{}),
		hotspotUsers:   make(map[string]struct{}),
		hotspotActives: make(map[string]struct{}),
	}
}

type InactiveAggregator struct {
	cache  cache.Repository
	pubsub pubsub.Publisher
	logger *slog.Logger

	mu      sync.Mutex
	routers map[string]*routerAggState
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
		cache:   c,
		pubsub:  ps,
		logger:  logger,
		routers: make(map[string]*routerAggState),
	}
}

func (a *InactiveAggregator) OnEvent(ctx context.Context, event behavior.StreamEvent) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	s := a.ensureRouter(event.RouterID)

	switch event.Meta.Measurement {
	case "ppp_secret":
		name := event.Fields["name"]
		if name == "" {
			return nil
		}
		if event.IsDead {
			delete(s.pppSecrets, name)
		} else {
			s.pppSecrets[name] = struct{}{}
		}
		a.schedulePPP(ctx, event.RouterID, s)

	case "ppp_active":
		name := event.Fields["name"]
		if name == "" {
			return nil
		}
		if event.IsDead {
			delete(s.pppActives, name)
		} else {
			s.pppActives[name] = struct{}{}
		}
		a.schedulePPP(ctx, event.RouterID, s)

	case "hotspot_user":
		name := event.Fields["name"]
		if name == "" {
			return nil
		}
		if event.IsDead {
			delete(s.hotspotUsers, name)
		} else {
			s.hotspotUsers[name] = struct{}{}
		}
		a.scheduleHotspot(ctx, event.RouterID, s)

	case "hotspot_active":
		user := event.Fields["user"]
		if user == "" {
			return nil
		}
		if event.IsDead {
			delete(s.hotspotActives, user)
		} else {
			s.hotspotActives[user] = struct{}{}
		}
		a.scheduleHotspot(ctx, event.RouterID, s)
	}

	return nil
}

func (a *InactiveAggregator) schedulePPP(ctx context.Context, routerID string, s *routerAggState) {
	if s.pppTimer != nil {
		s.pppTimer.Stop()
	}
	s.pppTimer = time.AfterFunc(aggregatorDebounce, func() {
		a.recalculate(ctx, routerID, "ppp")
	})
}

func (a *InactiveAggregator) scheduleHotspot(ctx context.Context, routerID string, s *routerAggState) {
	if s.hotspotTimer != nil {
		s.hotspotTimer.Stop()
	}
	s.hotspotTimer = time.AfterFunc(aggregatorDebounce, func() {
		a.recalculate(ctx, routerID, "hotspot")
	})
}

func (a *InactiveAggregator) ensureRouter(routerID string) *routerAggState {
	if a.routers[routerID] == nil {
		a.routers[routerID] = newRouterAggState()
	}
	return a.routers[routerID]
}

func (a *InactiveAggregator) recalculate(ctx context.Context, routerID, category string) {
	a.mu.Lock()
	s := a.routers[routerID]
	if s == nil {
		a.mu.Unlock()
		return
	}

	var inactiveNames []string
	var key string

	switch category {
	case "ppp":
		key = fmt.Sprintf("roskit:%s:ppp_inactive", routerID)
		for name := range s.pppSecrets {
			if _, active := s.pppActives[name]; !active {
				inactiveNames = append(inactiveNames, name)
			}
		}
	case "hotspot":
		key = fmt.Sprintf("roskit:%s:hotspot_inactive", routerID)
		for name := range s.hotspotUsers {
			if _, active := s.hotspotActives[name]; !active {
				inactiveNames = append(inactiveNames, name)
			}
		}
	default:
		a.mu.Unlock()
		return
	}
	a.mu.Unlock()

	jsonPayload, _ := json.Marshal(inactiveNames)
	finalData := map[string]string{
		"names":     string(jsonPayload),
		"count":     fmt.Sprintf("%d", len(inactiveNames)),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	if err := a.cache.SetSnapshot(ctx, key, finalData, aggregatorCacheTTL); err != nil {
		a.logger.Error("aggregator: cache write failed",
			"category", category, "router_id", routerID, "err", err)
	} else {
		a.logger.Debug("aggregator: inactive recalculated",
			"category", category, "router_id", routerID, "count", len(inactiveNames))
	}

	channel := cache.FormatPubSubChannel(routerID)
	a.pubsub.Publish(ctx, channel, pubsub.Message{
		Type:        "update",
		RouterID:    routerID,
		Measurement: category + "_inactive",
		Fields: map[string]string{
			"count": fmt.Sprintf("%d", len(inactiveNames)),
		},
		Timestamp: time.Now(),
	})
}

func (a *InactiveAggregator) GetInactiveHotspotUsers(routerID string) []string {
	a.mu.Lock()
	defer a.mu.Unlock()

	s := a.routers[routerID]
	if s == nil {
		return nil
	}
	var inactive []string
	for name := range s.hotspotUsers {
		if _, active := s.hotspotActives[name]; !active {
			inactive = append(inactive, name)
		}
	}
	return inactive
}

func (a *InactiveAggregator) GetInactivePPPSecrets(routerID string) []string {
	a.mu.Lock()
	defer a.mu.Unlock()

	s := a.routers[routerID]
	if s == nil {
		return nil
	}
	var inactive []string
	for name := range s.pppSecrets {
		if _, active := s.pppActives[name]; !active {
			inactive = append(inactive, name)
		}
	}
	return inactive
}

var _ behavior.StreamSink = (*InactiveAggregator)(nil)
