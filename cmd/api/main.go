package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/quiqxiq/roskit/internal/api"
	"github.com/quiqxiq/roskit/internal/config"
	"github.com/quiqxiq/roskit/internal/repository"
	"github.com/quiqxiq/roskit/internal/roskit/orchestrator"
	roskitservice "github.com/quiqxiq/roskit/internal/roskit/adapter/service"
	roskitcache "github.com/quiqxiq/roskit/internal/roskit/pipeline/cache"
	roskitpubsub "github.com/quiqxiq/roskit/internal/roskit/pipeline/pubsub"
	roskittimeseries "github.com/quiqxiq/roskit/internal/roskit/pipeline/timeseries"
	"github.com/quiqxiq/roskit/internal/services"
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

	var tsWriter roskittimeseries.Writer = roskittimeseries.NoopWriter{}
	var tsReader roskittimeseries.Reader = roskittimeseries.NoopReader{}

	if cfg.InfluxURL != "" {
		influxW, influxErr := roskittimeseries.NewInfluxWriter(roskittimeseries.InfluxWriterConfig{
			URL:      cfg.InfluxURL,
			Token:    cfg.InfluxToken,
			Database: cfg.InfluxDatabase,
		}, logger)
		if influxErr != nil {
			logger.Warn("InfluxDB writer init failed, time-series disabled", "error", influxErr)
		} else {
			defer influxW.Close()
			tsWriter = influxW
		}

		influxR, influxRErr := roskittimeseries.NewInfluxReader(roskittimeseries.InfluxReaderConfig{
			URL:      cfg.InfluxURL,
			Token:    cfg.InfluxToken,
			Database: cfg.InfluxDatabase,
		})
		if influxRErr != nil {
			logger.Warn("InfluxDB reader init failed", "error", influxRErr)
		} else {
			tsReader = influxR
		}
	}

	var roskitCache roskitcache.Repository = roskitcache.NoopRepository{}
	var roskitPubSub roskitpubsub.Publisher = roskitpubsub.NoopPublisher{}
	var roskitSubscriber roskitpubsub.Subscriber = roskitpubsub.NoopSubscriber{}

	redisRepo, redisErr := roskitcache.NewRedisRepository(roskitcache.RedisConfig{
		Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	}, logger)
	if redisErr != nil {
		logger.Warn("Redis repository init failed, telemetry cache/pubsub disabled", "error", redisErr)
	} else {
		defer redisRepo.Close()
		roskitCache = redisRepo
	}

	redisPub, redisPubErr := roskitpubsub.NewRedisPublisher(roskitcache.RedisConfig{
		Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	}, logger)
	if redisPubErr != nil {
		logger.Warn("Redis publisher init failed, pubsub disabled", "error", redisPubErr)
	} else {
		defer redisPub.Close()
		roskitPubSub = redisPub
	}

	subscriber, subErr := roskitpubsub.NewRedisSubscriber(roskitcache.RedisConfig{
		Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	}, logger)
	if subErr != nil {
		logger.Warn("Redis subscriber init failed, SSE disabled", "error", subErr)
	} else {
		defer subscriber.Close()
		roskitSubscriber = subscriber
	}

	engine := orchestrator.New(orchestrator.Config{
		Logger:     logger,
		Cache:      roskitCache,
		TimeSeries: tsWriter,
		PubSub:     roskitPubSub,
	})

	bridge := roskitservice.NewBridge(engine.Dispatcher(), roskitCache)

	routerRepo := repository.NewRouterRepo(db)
	routerSvc := services.NewRouterService(routerRepo, engine, cache, cfg.AESEncKey)
	routerSvc.SeedEngineFromDB(context.Background())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine.Start(ctx)

	saleRepo := repository.NewSaleRepo(db)
	profileRepo := repository.NewProfilePriceMappingRepo(db)

	backgroundWorker := worker.New(db, cache, bridge, saleRepo, profileRepo, cfg, nil)
	go backgroundWorker.Start(ctx)

	router := api.NewRouter(cfg, db, cache, bridge, routerSvc, tsReader, roskitSubscriber)

	srv := &http.Server{
		Addr:        fmt.Sprintf(":%d", cfg.Port),
		Handler:     router,
		ReadTimeout: 15 * time.Second,
		IdleTimeout: 120 * time.Second,
	}

	go func() {
		log.Printf("API server starting on :%d", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")
	engine.Stop()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutCancel()

	if err := srv.Shutdown(shutCtx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("server exited")
}
