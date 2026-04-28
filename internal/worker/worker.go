package worker

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/quiqxiq/roskit/internal/config"
	"github.com/quiqxiq/roskit/internal/models"
	"github.com/quiqxiq/roskit/internal/repository"
	roskitservice "github.com/quiqxiq/roskit/internal/roskit/adapter/service"
	"github.com/quiqxiq/roskit/pkg/redis"
	"gorm.io/gorm"
)

type Worker struct {
	db          *gorm.DB
	cache       *redis.Cache
	bridge      *roskitservice.Bridge
	saleRepo    repository.SaleRepository
	profileRepo repository.ProfilePriceMappingRepository
	cfg         *config.Config
	routerRepo  *repository.RouterRepo
}

func New(
	db *gorm.DB,
	cache *redis.Cache,
	bridge *roskitservice.Bridge,
	saleRepo repository.SaleRepository,
	profileRepo repository.ProfilePriceMappingRepository,
	cfg *config.Config,
) *Worker {
	return &Worker{
		db:          db,
		cache:       cache,
		bridge:      bridge,
		saleRepo:    saleRepo,
		profileRepo: profileRepo,
		cfg:         cfg,
		routerRepo:  repository.NewRouterRepo(db),
	}
}

func (w *Worker) Start(ctx context.Context) {
	slog.Info("Starting background worker routines")

	go w.salesCacheWarmup(ctx)

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.profilePriceSync(ctx)
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.voucherSessionCleanup(ctx)
			}
		}
	}()
}

func (w *Worker) profilePriceSync(ctx context.Context) {
	slog.Info("Running ProfilePriceSync task")
	routers, err := w.routerRepo.List(ctx)
	if err != nil {
		slog.Error("ProfilePriceSync failed to list routers", "error", err)
		return
	}

	poolStatus := w.bridge.PoolStatus()

	for _, router := range routers {
		rID := fmt.Sprintf("%d", router.ID)

		if poolStatus != nil && !poolStatus[rID] {
			continue
		}

		profiles, err := w.bridge.ListHotspotProfiles(ctx, rID)
		if err != nil {
			slog.Error("ProfilePriceSync failed to fetch profiles", "router", router.ID, "error", err)
			continue
		}

		for _, prof := range profiles {
			onLogin := prof["on-login"]
			if onLogin == "" {
				continue
			}

			meta := roskitservice.ParseOnLoginPut(onLogin)
			if meta == nil {
				continue
			}

			price, _ := strconv.ParseInt(meta.Price, 10, 64)
			sprice, _ := strconv.ParseInt(meta.SellingPrice, 10, 64)
			lockUser := meta.LockUser == "Enable"
			lockServer := meta.LockServer != "Disable" && meta.LockServer != ""

			mapping := &models.ProfilePriceMapping{
				RouterID:     router.ID,
				ProfileName:  prof["name"],
				Price:        price,
				SellingPrice: sprice,
				Validity:     meta.Validity,
				ExpMode:      meta.ExpMode,
				LockUser:     lockUser,
				LockServer:   lockServer,
			}

			if err := w.profileRepo.Upsert(ctx, mapping); err != nil {
				slog.Error("ProfilePriceSync failed to upsert profile mapping", "router", router.ID, "profile", prof["name"], "error", err)
			}
		}
	}
}

func (w *Worker) salesCacheWarmup(ctx context.Context) {
	slog.Info("Running SalesCacheWarmup task")
	routers, err := w.routerRepo.List(ctx)
	if err != nil {
		slog.Error("SalesCacheWarmup failed to list routers", "error", err)
		return
	}

	for _, router := range routers {
		today, err := w.saleRepo.TodayTotal(ctx, router.ID)
		if err == nil {
			cacheKey := fmt.Sprintf("mikhmon:sales:today:%d", router.ID)
			w.cache.SetJSON(ctx, cacheKey, today, 5*time.Minute)
		}

		month, err := w.saleRepo.MonthTotal(ctx, router.ID)
		if err == nil {
			cacheKey := fmt.Sprintf("mikhmon:sales:month:%d", router.ID)
			w.cache.SetJSON(ctx, cacheKey, month, 5*time.Minute)
		}
	}
}

func (w *Worker) voucherSessionCleanup(ctx context.Context) {
	slog.Info("Running VoucherSessionCleanup task")

	routers, err := w.routerRepo.List(ctx)
	if err != nil {
		return
	}

	poolStatus := w.bridge.PoolStatus()

	for _, router := range routers {
		rID := fmt.Sprintf("%d", router.ID)
		if poolStatus != nil && !poolStatus[rID] {
			continue
		}

		count, err := w.bridge.GetActiveSessionCount(ctx, rID)
		if err != nil {
			continue
		}

		slog.Info("Active voucher sessions", "router", router.ID, "count", count)
	}
}

