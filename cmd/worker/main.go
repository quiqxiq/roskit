package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/quiqxiq/roskit/internal/config"
	"github.com/quiqxiq/roskit/internal/repository"
	"github.com/quiqxiq/roskit/internal/roskit/orchestrator"
	roskitservice "github.com/quiqxiq/roskit/internal/roskit/adapter/service"
	roskitcache "github.com/quiqxiq/roskit/internal/roskit/pipeline/cache"
	"github.com/quiqxiq/roskit/internal/worker"
	"github.com/quiqxiq/roskit/pkg/database"
	appredis "github.com/quiqxiq/roskit/pkg/redis"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := database.Connect(cfg.PostgresDSN())
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	rdb, err := appredis.Connect(cfg.RedisURL())
	if err != nil {
		log.Fatalf("failed to connect redis: %v", err)
	}

	cache := appredis.NewCache(rdb)
	logger := slog.Default()

	var roskitCache roskitcache.Repository = roskitcache.NoopRepository{}
	redisRepo, redisErr := roskitcache.NewRedisRepository(roskitcache.RedisConfig{
		Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	}, logger)
	if redisErr != nil {
		logger.Warn("Redis repository init failed", "error", redisErr)
	} else {
		defer redisRepo.Close()
		roskitCache = redisRepo
	}

	engine := orchestrator.New(orchestrator.Config{
		Logger: logger,
		Cache:  roskitCache,
	})

	bridge := roskitservice.NewBridge(engine.Dispatcher(), roskitCache)

	saleRepo := repository.NewSaleRepo(db)
	profileRepo := repository.NewProfilePriceMappingRepo(db)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	backgroundWorker := worker.New(db, cache, bridge, saleRepo, profileRepo, cfg)
	backgroundWorker.Start(ctx)

	log.Println("worker started, waiting for events...")

	<-ctx.Done()
	log.Println("worker shutting down")
	engine.Stop()
}
