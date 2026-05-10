package worker

import (
	"context"
	"log/slog"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/quiqxiq/roskit/internal/repository"
	"github.com/quiqxiq/roskit/pkg/redis"
	"gorm.io/gorm"
)

type Worker struct {
	db         *gorm.DB
	cache      *redis.Cache
	saleRepo   repository.SaleRepository
	routerRepo *repository.RouterRepo
}

func New(
	db *gorm.DB,
	cache *redis.Cache,
	saleRepo repository.SaleRepository,
) *Worker {
	return &Worker{
		db:         db,
		cache:      cache,
		saleRepo:   saleRepo,
		routerRepo: repository.NewRouterRepo(db),
	}
}

func (w *Worker) Start(ctx context.Context) {
	slog.Info("Starting background worker routines")

	go w.salesCacheWarmup(ctx)

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

func (w *Worker) salesCacheWarmup(ctx context.Context) {
	slog.Info("Running SalesCacheWarmup task")
	routers, err := w.routerRepo.List(ctx)
	if err != nil {
		slog.Error("SalesCacheWarmup failed to list routers", "error", err)
		return
	}

	for _, router := range routers {
		rid := router.ID
		today, err := w.saleRepo.TodayTotal(ctx, &rid)
		if err == nil {
			w.cache.SetJSON(ctx, redis.SalesKey(router.ID, "today"), today, 5*time.Minute)
		}

		month, err := w.saleRepo.MonthTotal(ctx, &rid)
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
					if err := client.Del(ctx, toDelete...).Err(); err != nil {
						slog.Error("VoucherSessionCleanup delete failed", "error", err)
					} else {
						deleted += len(toDelete)
					}
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

