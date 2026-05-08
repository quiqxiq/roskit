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
	"github.com/quiqxiq/roskit/internal/api/middleware"
	casbinx "github.com/quiqxiq/roskit/internal/casbin"
	"github.com/quiqxiq/roskit/internal/config"
	"github.com/quiqxiq/roskit/internal/repository"
	roskitservice "github.com/quiqxiq/roskit/internal/roskit/adapter/service"
	"github.com/quiqxiq/roskit/internal/roskit/orchestrator"
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
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
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

	// ===== Casbin =====
	enforcer, err := casbinx.NewEnforcer(db)
	if err != nil {
		log.Fatalf("failed to init casbin enforcer: %v", err)
	}
	if err := casbinx.SeedPolicies(enforcer); err != nil {
		logger.Warn("casbin policy seed failed", "error", err)
	}

	// ===== Repositories (top-level wiring) =====
	tenantRepo := repository.NewTenantRepo(db)
	tenantSettingsRepo := repository.NewTenantSettingsRepo(db)
	userRepo := repository.NewUserRepo(db)
	templateRepo := repository.NewTemplateRepo(db)
	saleRepo := repository.NewSaleRepo(db)
	profileRepo := repository.NewProfilePriceMappingRepo(db)

	// ===== Services that auth depends on =====
	tenantSvc := services.NewTenantService(db, tenantRepo, tenantSettingsRepo, templateRepo)
	authSvc := services.NewAuthService(
		userRepo,
		tenantRepo,
		enforcer,
		cache,
		[]byte(cfg.JWTSecret),
		[]byte(cfg.JWTRefreshSecret),
	)

	var setupLoggingFn func(ctx context.Context, routerID string)
	var syncTimezoneFn func(ctx context.Context, routerID string)

	engine := orchestrator.New(orchestrator.Config{
		Logger:     logger,
		Cache:      roskitCache,
		TimeSeries: tsWriter,
		PubSub:     roskitPubSub,
		OnRouterConnect: func(ctx context.Context, routerID string) {
			if setupLoggingFn != nil {
				setupLoggingFn(ctx, routerID)
			}
			if syncTimezoneFn != nil {
				syncTimezoneFn(ctx, routerID)
			}
		},
	})

	bridge := roskitservice.NewBridge(engine.Dispatcher(), roskitCache)

	setupLoggingFn = func(ctx context.Context, routerID string) {
		if err := bridge.SetupLogging(ctx, routerID); err != nil {
			logger.Warn("auto setup logging failed", "router_id", routerID, "error", err)
		}
	}

	routerRepo := repository.NewRouterRepo(db)

	syncTimezoneFn = func(ctx context.Context, routerID string) {
		clock, err := bridge.GetSystemClock(ctx, routerID)
		if err != nil {
			logger.Warn("failed to query system clock for timezone", "router_id", routerID, "error", err)
			return
		}
		tz := clock["time-zone-name"]
		if tz == "" {
			return
		}
		if err := routerRepo.UpdateTimezone(ctx, routerID, tz); err != nil {
			logger.Warn("failed to save router timezone", "router_id", routerID, "error", err)
		}
	}

	routerSvc := services.NewRouterService(routerRepo, engine, cache, cfg.AESEncKey)
	routerSvc.SeedEngineFromDB(context.Background())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine.Start(ctx)

	go routerSvc.WatchAndSyncStatus(ctx)

	backgroundWorker := worker.New(db, cache, saleRepo)
	go backgroundWorker.Start(ctx)

	auditRepo := repository.NewAuditRepo(db)
	auditLogger := middleware.NewAuditLogger(auditRepo)

	router := api.NewRouter(
		cfg, db, cache, bridge,
		routerSvc, tsReader, roskitSubscriber, profileRepo,
		tenantRepo, tenantSettingsRepo, tenantSvc, authSvc, enforcer,
		auditLogger,
	)

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

	// Drain buffered audit entries before exiting. Bounded by shutCtx so a
	// stuck DB cannot block the process from terminating.
	if err := auditLogger.Shutdown(shutCtx); err != nil {
		log.Printf("audit logger shutdown: %v", err)
	}

	log.Println("server exited")
}
