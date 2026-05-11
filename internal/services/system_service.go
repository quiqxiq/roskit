package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/quiqxiq/roskit/internal/repository"
	roskitservice "github.com/quiqxiq/roskit/internal/roskit/adapter/service"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/timeseries"
	appcache "github.com/quiqxiq/roskit/pkg/redis"
)

type SystemService struct {
	bridge     *roskitservice.Bridge
	tsReader   timeseries.Reader
	routerRepo *repository.RouterRepo
	cache      *appcache.Cache
	logger     *slog.Logger
}

func NewSystemService(bridge *roskitservice.Bridge, tsReader timeseries.Reader, routerRepo *repository.RouterRepo, cache *appcache.Cache) *SystemService {
	return &SystemService{
		bridge:     bridge,
		tsReader:   tsReader,
		routerRepo: routerRepo,
		cache:      cache,
		logger:     slog.Default().With("component", "system-svc"),
	}
}

func (s *SystemService) GetSystemResource(ctx context.Context, routerID uint) (map[string]string, error) {
	return s.bridge.GetSystemResource(ctx, fmt.Sprintf("%d", routerID))
}

func (s *SystemService) GetSystemLog(ctx context.Context, routerID uint, limit int) ([]map[string]string, error) {
	data, err := s.bridge.GetSystemLog(ctx, fmt.Sprintf("%d", routerID), "")
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(data) > limit {
		data = data[:limit]
	}
	return data, nil
}

func (s *SystemService) GetSystemClock(ctx context.Context, routerID uint) (map[string]string, error) {
	return s.bridge.GetSystemClock(ctx, fmt.Sprintf("%d", routerID))
}

func (s *SystemService) GetSystemIdentity(ctx context.Context, routerID uint) (map[string]string, error) {
	return s.bridge.GetSystemIdentity(ctx, fmt.Sprintf("%d", routerID))
}

func (s *SystemService) GetRouterboard(ctx context.Context, routerID uint) (map[string]string, error) {
	return s.bridge.GetRouterboard(ctx, fmt.Sprintf("%d", routerID))
}

func (s *SystemService) Reboot(ctx context.Context, routerID uint) error {
	return s.bridge.Reboot(ctx, fmt.Sprintf("%d", routerID))
}

func (s *SystemService) Shutdown(ctx context.Context, routerID uint) error {
	return s.bridge.Shutdown(ctx, fmt.Sprintf("%d", routerID))
}

func (s *SystemService) GetExpireMonitorStatus(ctx context.Context, routerID uint) (map[string]string, error) {
	return s.bridge.CheckExpireMonitor(ctx, fmt.Sprintf("%d", routerID))
}

func (s *SystemService) DeployExpireMonitor(ctx context.Context, routerID uint, interval string) error {
	return s.bridge.DeployExpireMonitor(ctx, fmt.Sprintf("%d", routerID), interval)
}

func (s *SystemService) RemoveExpireMonitor(ctx context.Context, routerID uint) error {
	return s.bridge.RemoveExpireMonitor(ctx, fmt.Sprintf("%d", routerID))
}

func (s *SystemService) ListSchedulers(ctx context.Context, routerID uint) ([]map[string]string, error) {
	return s.bridge.ListSchedulers(ctx, fmt.Sprintf("%d", routerID))
}

func (s *SystemService) AddScheduler(ctx context.Context, routerID uint, params map[string]string) (map[string]string, error) {
	rID := fmt.Sprintf("%d", routerID)
	_, err := s.bridge.AddScheduler(ctx, rID, params)
	if err != nil {
		return nil, err
	}
	return map[string]string{"message": "scheduler created"}, nil
}

func (s *SystemService) UpdateScheduler(ctx context.Context, routerID uint, schedulerID string, params map[string]string) error {
	rID := fmt.Sprintf("%d", routerID)
	return s.bridge.SetScheduler(ctx, rID, schedulerID, params)
}

func (s *SystemService) DeleteScheduler(ctx context.Context, routerID uint, schedulerID string) error {
	rID := fmt.Sprintf("%d", routerID)
	return s.bridge.RemoveScheduler(ctx, rID, schedulerID)
}

func (s *SystemService) EnableSchedulerByID(ctx context.Context, routerID uint, schedulerID string) error {
	rID := fmt.Sprintf("%d", routerID)
	return s.bridge.EnableScheduler(ctx, rID, schedulerID)
}

func (s *SystemService) DisableSchedulerByID(ctx context.Context, routerID uint, schedulerID string) error {
	rID := fmt.Sprintf("%d", routerID)
	return s.bridge.DisableScheduler(ctx, rID, schedulerID)
}

func (s *SystemService) GetDashboard(ctx context.Context, routerID uint) (map[string]interface{}, error) {
	rID := fmt.Sprintf("%d", routerID)

	resource, err := s.bridge.GetSystemResource(ctx, rID)
	if err != nil {
		return nil, fmt.Errorf("get resource: %w", err)
	}

	identity, err := s.bridge.GetSystemIdentity(ctx, rID)
	if err != nil {
		return nil, fmt.Errorf("get identity: %w", err)
	}

	userCount, err := s.bridge.GetHotspotUserCount(ctx, rID)
	if err != nil {
		s.logger.Warn("failed to get hotspot user count", "error", err)
	}

	activeCount, err := s.bridge.GetActiveSessionCount(ctx, rID)
	if err != nil {
		s.logger.Warn("failed to get active session count", "error", err)
	}

	result := map[string]interface{}{
		"resource":        resource,
		"identity":        identity,
		"hotspot_users":   userCount,
		"active_sessions": activeCount,
	}

	health, err := s.bridge.GetSystemHealth(ctx, rID)
	if err == nil {
		result["system_health"] = health
	}

	var summary DashboardSummary
	found, err := s.cache.GetJSON(ctx, appcache.DashboardKey(routerID), &summary)
	if err == nil && found {
		result["income"] = summary
	}

	logs, err := s.bridge.GetSystemLog(ctx, rID, "hotspot")
	if err == nil && len(logs) > 0 {
		if len(logs) > 5 {
			logs = logs[len(logs)-5:]
		}
		result["recent_logs"] = logs
	}

	return result, nil
}

func (s *SystemService) ListScripts(ctx context.Context, routerID uint) ([]map[string]string, error) {
	rID := fmt.Sprintf("%d", routerID)
	return s.bridge.ListScripts(ctx, rID)
}

func (s *SystemService) AddScript(ctx context.Context, routerID uint, params map[string]string) (map[string]string, error) {
	rID := fmt.Sprintf("%d", routerID)
	_, err := s.bridge.AddScript(ctx, rID, params)
	if err != nil {
		return nil, err
	}
	return map[string]string{"message": "script created"}, nil
}

func (s *SystemService) UpdateScript(ctx context.Context, routerID uint, scriptID string, params map[string]string) error {
	rID := fmt.Sprintf("%d", routerID)
	return s.bridge.SetScript(ctx, rID, scriptID, params)
}

func (s *SystemService) DeleteScript(ctx context.Context, routerID uint, scriptID string) error {
	rID := fmt.Sprintf("%d", routerID)
	return s.bridge.RemoveScript(ctx, rID, scriptID)
}

func (s *SystemService) RunScript(ctx context.Context, routerID uint, scriptID string) error {
	rID := fmt.Sprintf("%d", routerID)
	return s.bridge.RunScript(ctx, rID, scriptID)
}

func (s *SystemService) SetupLogging(ctx context.Context, routerID uint) error {
	rID := fmt.Sprintf("%d", routerID)
	return s.bridge.SetupLogging(ctx, rID)
}

func (s *SystemService) GetSystemResourceHistory(ctx context.Context, routerID uint, rangeStr, step string) ([]map[string]interface{}, error) {
	rangeDurations := map[string]time.Duration{
		"15m": 15 * time.Minute,
		"1h":  1 * time.Hour,
		"6h":  6 * time.Hour,
		"12h": 12 * time.Hour,
		"24h": 24 * time.Hour,
		"7d":  7 * 24 * time.Hour,
		"30d": 30 * 24 * time.Hour,
	}

	dur, ok := rangeDurations[rangeStr]
	if !ok {
		dur = time.Hour
	}

	now := time.Now()
	from := now.Add(-dur)

	if step == "" {
		switch {
		case dur <= time.Hour:
			step = "1m"
		case dur <= 6*time.Hour:
			step = "5m"
		case dur <= 24*time.Hour:
			step = "15m"
		default:
			step = "1h"
		}
	}

	return s.tsReader.QueryRange(ctx, "system_resource", fmt.Sprintf("%d", routerID), from, now, step)
}
