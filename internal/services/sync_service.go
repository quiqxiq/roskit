package services

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/quiqxiq/roskit/internal/models"
	"github.com/quiqxiq/roskit/internal/repository"
	roskitservice "github.com/quiqxiq/roskit/internal/roskit/adapter/service"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

type SyncService struct {
	routerRepo   *repository.RouterRepo
	saleRepo     repository.SaleRepository
	settingsRepo repository.SettingsRepository
	bridge       *roskitservice.Bridge
	logger       *slog.Logger
}

func NewSyncService(
	routerRepo *repository.RouterRepo,
	saleRepo repository.SaleRepository,
	settingsRepo repository.SettingsRepository,
	bridge *roskitservice.Bridge,
) *SyncService {
	return &SyncService{
		routerRepo:   routerRepo,
		saleRepo:     saleRepo,
		settingsRepo: settingsRepo,
		bridge:       bridge,
		logger:       slog.Default().With("component", "sync-svc"),
	}
}

func (s *SyncService) SyncAllActiveRouters(ctx context.Context) {
	routers, err := s.routerRepo.List(ctx)
	if err != nil {
		s.logger.Error("sync: failed to list routers", "error", err)
		return
	}

	for _, r := range routers {
		if r.Status != models.RouterStatusConnected {
			continue
		}
		if err := s.SyncRouter(ctx, r); err != nil {
			s.logger.Warn("sync: router sync failed",
				"router_id", r.ID, "error", err)
		}
	}
}

func (s *SyncService) SyncRouter(ctx context.Context, router *models.Router) error {
	routerIDStr := fmt.Sprintf("%d", router.ID)

	records, err := s.bridge.ImportSalesFromRouterOS(ctx, routerIDStr, "")
	if err != nil {
		return fmt.Errorf("fetch scripts: %w", err)
	}

	if len(records) == 0 {
		return nil
	}

	tz := ""
	if settings, err := s.settingsRepo.Get(ctx); err == nil && settings != nil {
		tz = settings.Timezone
	}
	loc := mikrotik.ResolveLocation(tz)

	synced, cleaned := 0, 0

	for _, rec := range records {
		if rec.Price == "" || rec.Price == "0" {
			continue
		}

		soldAt, _ := mikrotik.Parse(rec.Date, rec.Time, loc)
		if soldAt.IsZero() {
			soldAt = time.Now()
		}

		key := mikrotik.MakeSaleIdempotencyKey(router.ID, rec.Username, soldAt)

		exists, err := s.saleRepo.ExistsByIdempotencyKey(ctx, key)
		if err != nil {
			s.logger.Warn("sync: idempotency check failed",
				"router_id", router.ID, "username", rec.Username, "error", err)
			continue
		}

		if exists {
			s.deleteScript(ctx, routerIDStr, rec.ID)
			cleaned++
			continue
		}

		price, _ := strconv.ParseInt(rec.Price, 10, 64)
		rid := router.ID
		sale := &models.VoucherSale{
			RouterID:       &rid,
			SoldAt:         soldAt,
			Username:       rec.Username,
			ProfileName:    rec.Profile,
			Price:          price,
			Server:         rec.Source,
			IPAddress:      rec.IPAddress,
			MACAddress:     rec.MACAddress,
			Validity:       rec.Validity,
			IdempotencyKey: key,
		}

		if err := s.saleRepo.CreateBatch(ctx, []*models.VoucherSale{sale}); err != nil {
			s.logger.Error("sync: failed to insert sale",
				"router_id", router.ID, "username", rec.Username, "error", err)
			continue
		}

		s.deleteScript(ctx, routerIDStr, rec.ID)
		synced++
	}

	if synced > 0 || cleaned > 0 {
		s.logger.Info("sync complete",
			"router_id", router.ID, "synced", synced, "cleaned", cleaned)
	}

	return nil
}

func (s *SyncService) deleteScript(ctx context.Context, routerID, scriptID string) {
	if scriptID == "" {
		return
	}
	if err := s.bridge.DeleteRouterOSScript(ctx, routerID, scriptID); err != nil {
		s.logger.Warn("sync: failed to delete script",
			"router_id", routerID, "script_id", scriptID, "error", err)
	}
}
