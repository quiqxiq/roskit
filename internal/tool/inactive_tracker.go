package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// InactiveTracker tracks which PPP and Hotspot users are configured but currently disconnected.
// Because RouterOS does not provide a single query for inactive users, this tool calculates
// the difference between active sessions and configured secrets/users, updating Redis dynamically.
type InactiveTracker struct {
	rdb    *redis.Client
	logger *slog.Logger
	mu     sync.Mutex

	// pending handles debouncing of recalculation to avoid overwhelming Redis on massive reconnects.
	pending map[string]bool
}

// NewInactiveTracker creates a new tracker tool using the given Redis client.
func NewInactiveTracker(rdb *redis.Client, logger *slog.Logger) *InactiveTracker {
	return &InactiveTracker{
		rdb:     rdb,
		logger:  logger.With("component", "inactive_tracker"),
		pending: make(map[string]bool),
	}
}

// Start begins subscribing to telemetry events and processing.
// This blocks until ctx is canceled.
func (t *InactiveTracker) Start(ctx context.Context) {
	pubsub := t.rdb.PSubscribe(ctx, "roskit:telemetry:*")
	defer pubsub.Close()

	ch := pubsub.Channel()
	t.logger.Info("Inactive tracker started listening to telemetry channels")

	for {
		select {
		case <-ctx.Done():
			t.logger.Info("Inactive tracker stopped")
			return
		case msg := <-ch:
			// Minimal unmarshal just to get router and measurement
			var evt struct {
				RouterID    string `json:"router_id"`
				Measurement string `json:"measurement"`
			}
			if err := json.Unmarshal([]byte(msg.Payload), &evt); err != nil {
				continue
			}

			switch evt.Measurement {
			case "ppp_secret", "ppp_active":
				t.scheduleRecalculation(ctx, evt.RouterID, "ppp")
			case "hotspot_user", "hotspot_active":
				t.scheduleRecalculation(ctx, evt.RouterID, "hotspot")
			}
		}
	}
}

// scheduleRecalculation debounces the heavy difference calculation
func (t *InactiveTracker) scheduleRecalculation(ctx context.Context, routerID, service string) {
	taskKey := routerID + ":" + service
	t.mu.Lock()
	if t.pending[taskKey] {
		t.mu.Unlock()
		return
	}
	t.pending[taskKey] = true
	t.mu.Unlock()

	// Wait before calculating to debounce rapid events
	go func() {
		time.Sleep(1 * time.Second)

		t.mu.Lock()
		t.pending[taskKey] = false
		t.mu.Unlock()

		if service == "ppp" {
			t.recalculatePPP(ctx, routerID)
		} else if service == "hotspot" {
			t.recalculateHotspot(ctx, routerID)
		}
	}()
}

func (t *InactiveTracker) recalculatePPP(ctx context.Context, routerID string) {
	err := t.calculateDiff(ctx, routerID,
		fmt.Sprintf("roskit:%s:ppp_secret:*", routerID),
		fmt.Sprintf("roskit:%s:ppp_active:*", routerID),
		fmt.Sprintf("roskit:%s:derived:ppp_inactive", routerID),
	)
	if err != nil {
		t.logger.Error("Failed to recalculate inactive PPP", "router", routerID, "error", err)
	}
}

func (t *InactiveTracker) recalculateHotspot(ctx context.Context, routerID string) {
	// Active hotspot sessions specify user under 'user' key instead of 'name' typically,
	// but we'll fetch both just in case, our model has `name`. Wait, hotspot active model mapping: user->name
	err := t.calculateDiff(ctx, routerID,
		fmt.Sprintf("roskit:%s:hotspot_user:*", routerID),
		fmt.Sprintf("roskit:%s:hotspot_active:*", routerID),
		fmt.Sprintf("roskit:%s:derived:hotspot_inactive", routerID),
	)
	if err != nil {
		t.logger.Error("Failed to recalculate inactive Hotspot", "router", routerID, "error", err)
	}
}

// calculateDiff computes configured that are not active and writes them to a specific redis key.
// It assumes the active user key contains a "name" or "user" field identifying the user.
func (t *InactiveTracker) calculateDiff(ctx context.Context, routerID, configPattern, activePattern, destKey string) error {
	// 1. Fetch all configured users
	cfgKeys, err := t.rdb.Keys(ctx, configPattern).Result()
	if err != nil {
		return err
	}

	// 2. Fetch all active sessions
	actKeys, err := t.rdb.Keys(ctx, activePattern).Result()
	if err != nil {
		return err
	}

	// Fetch mapping configName -> profile/details via Pipeline for speed
	pipe := t.rdb.Pipeline()
	cfgCmds := make(map[string]*redis.MapStringStringCmd)
	for _, key := range cfgKeys {
		cfgCmds[key] = pipe.HGetAll(ctx, key)
	}

	actCmds := make(map[string]*redis.MapStringStringCmd)
	for _, key := range actKeys {
		actCmds[key] = pipe.HGetAll(ctx, key)
	}
	_, _ = pipe.Exec(ctx)

	// Build active set
	activeNames := make(map[string]bool)
	for _, cmd := range actCmds {
		res := cmd.Val()
		name := res["name"]
		if name == "" {
			name = res["user"] // Fallback for some hotspot structures
		}
		if name != "" {
			activeNames[name] = true
		}
	}

	// Calculate difference
	inactiveList := make(map[string]interface{})
	for _, cmd := range cfgCmds {
		res := cmd.Val()
		name := res["name"]
		// If a configured user is empty or disabled, we might skip them or count them as well.
		// For now we include them if they have a name.
		if name != "" && !activeNames[name] {
			// Convert to JSON to store inside Redis Hash
			jsonData, _ := json.Marshal(res)
			inactiveList[name] = string(jsonData)
		}
	}

	// Save to derived key
	// Clean previous state first
	t.rdb.Del(ctx, destKey)

	if len(inactiveList) > 0 {
		return t.rdb.HSet(ctx, destKey, inactiveList).Err()
	}

	return nil
}
