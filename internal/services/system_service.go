package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	roskitservice "github.com/quiqxiq/roskit/internal/roskit/adapter/service"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/timeseries"
)

type SystemService struct {
	bridge   *roskitservice.Bridge
	tsReader timeseries.Reader
	logger   *slog.Logger
}

func NewSystemService(bridge *roskitservice.Bridge, tsReader timeseries.Reader) *SystemService {
	return &SystemService{
		bridge:   bridge,
		tsReader: tsReader,
		logger:   slog.Default().With("component", "system-svc"),
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
	rID := fmt.Sprintf("%d", routerID)
	existing, err := s.bridge.CheckExpireMonitor(ctx, rID)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}
	args := []string{
		"=name=" + roskitservice.ExpireMonitorName,
		"=interval=" + interval,
		"=on-event=/system script run Mikhmon-Expire-Monitor",
		"=disabled=no",
	}
	_, err = s.bridge.Mutate(ctx, rID, "system/scheduler/add", args...)
	return err
}

func (s *SystemService) RemoveExpireMonitor(ctx context.Context, routerID uint) error {
	return s.bridge.RemoveExpireMonitor(ctx, fmt.Sprintf("%d", routerID))
}

func (s *SystemService) ListSchedulers(ctx context.Context, routerID uint) ([]map[string]string, error) {
	return s.bridge.ListSchedulers(ctx, fmt.Sprintf("%d", routerID))
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

	return map[string]interface{}{
		"resource":       resource,
		"identity":       identity,
		"hotspot_users":  userCount,
		"active_sessions": activeCount,
	}, nil
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
