package worker

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"

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
	routers, err := w.routerRepo.ListAll(ctx)
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
	routers, err := w.routerRepo.ListAll(ctx)
	if err != nil {
		slog.Error("SalesCacheWarmup failed to list routers", "error", err)
		return
	}

	for _, router := range routers {
		rid := router.ID
		today, err := w.saleRepo.TodayTotal(ctx, router.TenantID, &rid)
		if err == nil {
			w.cache.SetJSON(ctx, redis.SalesKey(router.ID, "today"), today, 5*time.Minute)
		}

		month, err := w.saleRepo.MonthTotal(ctx, router.TenantID, &rid)
		if err == nil {
			w.cache.SetJSON(ctx, redis.SalesKey(router.ID, "month"), month, 5*time.Minute)
		}
	}
}

func (w *Worker) voucherSessionCleanup(ctx context.Context) {
	slog.Info("Running VoucherSessionCleanup task")

	client := w.cache.Client()
	if client == nil {
		return
	}

	pattern := "mikhmon:vsession:*"
	var cursor uint64
	var deleted int
	for {
		keys, nextCursor, err := client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			slog.Error("VoucherSessionCleanup scan failed", "error", err)
			break
		}
		if len(keys) > 0 {
			ttls, err := client.Pipelined(ctx, func(pipe goredis.Pipeliner) error {
				for _, k := range keys {
					pipe.TTL(ctx, k)
				}
				return nil
			})
			if err == nil {
				var toDelete []string
				for i, cmd := range ttls {
					if ttlCmd, ok := cmd.(*goredis.DurationCmd); ok {
						ttlVal := ttlCmd.Val()
						if ttlVal < 0 {
							toDelete = append(toDelete, keys[i])
						}
					}
				}
				if len(toDelete) > 0 {
					client.Del(ctx, toDelete...)
					deleted += len(toDelete)
				}
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	if deleted > 0 {
		slog.Info("VoucherSessionCleanup complete", "deleted", deleted)
	}
}

