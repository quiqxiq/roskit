package api

import (
	"context"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/quiqxiq/roskit/internal/api/handlers"
	sse "github.com/quiqxiq/roskit/internal/api/handlers/sse"
	"github.com/quiqxiq/roskit/internal/api/middleware"
	"github.com/quiqxiq/roskit/internal/config"
	"github.com/quiqxiq/roskit/internal/models"
	"github.com/quiqxiq/roskit/internal/repository"
	roskitservice "github.com/quiqxiq/roskit/internal/roskit/adapter/service"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/pubsub"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/timeseries"
	"github.com/quiqxiq/roskit/internal/services"
	appcache "github.com/quiqxiq/roskit/pkg/redis"
	"gorm.io/gorm"
)

func NewRouter(
	cfg *config.Config,
	db *gorm.DB,
	rdb *appcache.Cache,
	bridge *roskitservice.Bridge,
	routerSvc *services.RouterService,
	tsReader timeseries.Reader,
	subscriber pubsub.Subscriber,
	profileRepo repository.ProfilePriceMappingRepository,
	settingsRepo repository.SettingsRepository,
	authSvc *services.AuthService,
	templateRepo repository.TemplateRepository,
	auditLogger *middleware.AuditLogger,
) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery(), middleware.LoggerMiddleware(), middleware.CORSMiddleware())

	cache := rdb

	userRepo := repository.NewUserRepo(db)
	routerRepo := repository.NewRouterRepo(db)
	saleRepo := repository.NewSaleRepo(db)

	systemSvc := services.NewSystemService(bridge, tsReader, routerRepo, cache)
	hotspotSvc := services.NewHotspotService(bridge, cache, cfg, settingsRepo, routerRepo, profileRepo)
	voucherSvc := services.NewVoucherService(bridge, saleRepo, routerRepo, profileRepo, settingsRepo, cache)
	reportSvc := services.NewReportService(saleRepo, cache)
	templateSvc := services.NewTemplateService(templateRepo)
	settingsSvc := services.NewSettingsService(settingsRepo)
	if err := templateSvc.SeedDefaults(context.Background()); err != nil {
		slog.Default().Warn("template seed defaults failed", "error", err)
	}

	routerH := handlers.NewRouterHandler(routerSvc)
	hotspotH := handlers.NewHotspotHandler(hotspotSvc)
	voucherH := handlers.NewVoucherHandler(voucherSvc, templateSvc, hotspotSvc, settingsRepo)
	reportH := handlers.NewReportHandler(reportSvc)
	authH := handlers.NewAuthHandler(authSvc, auditLogger)
	eventH := handlers.NewEventHandler(routerRepo, saleRepo, profileRepo, settingsRepo, bridge, cache)
	systemH := handlers.NewSystemHandler(systemSvc)
	pppH := handlers.NewPPPHandler(bridge)
	networkH := handlers.NewNetworkHandler(bridge)
	qpH := handlers.NewQuickPrintHandler(bridge)
	templateH := handlers.NewTemplateHandler(templateSvc, voucherSvc, settingsRepo)
	logSSEH := sse.NewLogSSEHandler(bridge)
	telemetrySSEH := sse.NewTelemetrySSEHandler(subscriber, bridge)
	pingSSEH := sse.NewPingSSEHandler(bridge)
	statusSvc := services.NewStatusService(routerRepo, bridge)
	statusH := handlers.NewStatusHandler(statusSvc)
	healthH := handlers.NewHealthHandler(db, cache, bridge)
	profileMappingH := handlers.NewProfileMappingHandler(profileRepo)
	settingsH := handlers.NewSettingsHandler(settingsSvc)
	userH := handlers.NewUserHandler(authSvc)

	_ = userRepo

	// Serve the frontend website from the same origin as the API
	engine.Static("/css", "./website/css")
	engine.Static("/js", "./website/js")
	engine.StaticFile("/", "./website/index.html")
	engine.StaticFile("/app.html", "./website/app.html")
	engine.NoRoute(func(c *gin.Context) {
		c.File("./website/index.html")
	})

	loginRL := middleware.RateLimit(10, time.Minute)
	onLoginRL := middleware.RateLimit(100, time.Minute)

	api := engine.Group("/api/v1")
	{
		api.GET("/health", healthH.Check)

		auth := api.Group("/auth")
		{
			auth.POST("/setup", loginRL, authH.Setup)
			auth.POST("/login", loginRL, authH.Login)
			auth.POST("/refresh", authH.Refresh)
			auth.POST("/logout", middleware.AuthMiddleware(authSvc), authH.Logout)
		}

		me := api.Group("")
		me.Use(middleware.AuthMiddleware(authSvc))
		{
			me.GET("/auth/me", authH.Me)
			me.PUT("/auth/password", authH.ChangePassword)
		}

		// Public on-login webhook (auth via webhook_token in body, no JWT).
		events := api.Group("/events")
		{
			events.POST("/on-login", onLoginRL, eventH.OnLoginEvent)
			events.GET("/health", eventH.HealthCheck)
		}

		// Public status check (no auth, used by login page).
		api.GET("/status", statusH.GetUserStatus)

		// All authenticated routes
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware(authSvc))
		{
			// ====== Settings (admin only) ======
			settings := protected.Group("/settings")
			settings.Use(middleware.RequireRole(models.UserRoleAdmin))
			{
				settings.GET("", settingsH.Get)
				settings.PUT("", settingsH.Update)
				settings.POST("/logo", settingsH.UploadLogo)
				settings.GET("/logo", settingsH.GetLogo)
			}

			// ====== User management (admin only) ======
			users := protected.Group("/users")
			users.Use(middleware.RequireRole(models.UserRoleAdmin))
			{
				users.GET("", userH.List)
				users.POST("", userH.Create)
				users.GET("/:id", userH.Get)
				users.PUT("/:id", userH.Update)
				users.DELETE("/:id", userH.Delete)
			}

			// ====== Templates (all authenticated) ======
			templates := protected.Group("/templates")
			{
				templates.GET("", templateH.List)
				templates.POST("", templateH.Create)
				templates.GET("/:templateId", templateH.Get)
				templates.PUT("/:templateId", templateH.Update)
				templates.DELETE("/:templateId", templateH.Delete)
				templates.POST("/render", templateH.Render)
				templates.POST("/seed-defaults", templateH.SeedDefaults)
			}

			// ====== Routers (all authenticated) ======
			routers := protected.Group("/routers")
			{
				routers.GET("", routerH.List)
				routers.POST("", routerH.Create)
				routers.POST("/migrate", routerH.MigrateConfig)
			}
			routerOne := protected.Group("/routers/:routerId")
			routerOne.Use(middleware.RouterOwnershipMiddleware(routerRepo))
			{
				routerOne.GET("", routerH.Get)
				routerOne.PUT("", routerH.Update)
				routerOne.DELETE("", routerH.Delete)
				routerOne.POST("/test", routerH.TestConnection)

				// Hotspot
				routerOne.GET("/hotspot/users", hotspotH.ListUsers)
				routerOne.GET("/hotspot/users/count", hotspotH.GetUserCount)
				routerOne.GET("/hotspot/users/export", hotspotH.ExportUsers)
				routerOne.GET("/hotspot/users/:id", hotspotH.GetUser)
				routerOne.POST("/hotspot/users", hotspotH.AddUser)
				routerOne.PUT("/hotspot/users/:id", hotspotH.UpdateUser)
				routerOne.DELETE("/hotspot/users/:id", hotspotH.RemoveUser)
				routerOne.POST("/hotspot/users/:id/reset-counters", hotspotH.ResetUserCounters)

				routerOne.GET("/hotspot/profiles", hotspotH.ListProfiles)
				routerOne.GET("/hotspot/profiles/:id", hotspotH.GetProfile)
				routerOne.POST("/hotspot/profiles", hotspotH.AddProfile)
				routerOne.PUT("/hotspot/profiles/:id", hotspotH.UpdateProfile)
				routerOne.DELETE("/hotspot/profiles/:id", hotspotH.RemoveProfile)
				routerOne.POST("/hotspot/profiles/sync", hotspotH.SyncProfiles)

				routerOne.GET("/hotspot/active", hotspotH.ListActive)
				routerOne.DELETE("/hotspot/active/:id", hotspotH.RemoveActive)
				routerOne.POST("/hotspot/active/:id/disconnect", hotspotH.DisconnectUser)

				routerOne.GET("/hotspot/inactive", hotspotH.ListInactive)
				routerOne.GET("/hotspot/inactive/count", hotspotH.GetInactiveCount)

				routerOne.GET("/hotspot/hosts", hotspotH.ListHosts)
				routerOne.DELETE("/hotspot/hosts/:id", hotspotH.RemoveHost)

				routerOne.GET("/hotspot/servers", hotspotH.ListServers)

				routerOne.GET("/hotspot/cookies", hotspotH.ListCookies)
				routerOne.DELETE("/hotspot/cookies/:id", hotspotH.RemoveCookie)

				routerOne.GET("/hotspot/bindings", hotspotH.ListIPBindings)
				routerOne.POST("/hotspot/bindings", hotspotH.AddIPBinding)
				routerOne.PUT("/hotspot/bindings/:id", hotspotH.UpdateIPBinding)
				routerOne.DELETE("/hotspot/bindings/:id", hotspotH.RemoveIPBinding)
				routerOne.POST("/hotspot/bindings/:id/enable", hotspotH.EnableIPBinding)
				routerOne.POST("/hotspot/bindings/:id/disable", hotspotH.DisableIPBinding)

				routerOne.GET("/hotspot/walled-garden", hotspotH.ListWalledGarden)
				routerOne.POST("/hotspot/walled-garden", hotspotH.AddWalledGarden)
				routerOne.DELETE("/hotspot/walled-garden/:wid", hotspotH.RemoveWalledGarden)
				routerOne.GET("/hotspot/walled-garden-ip", hotspotH.ListWalledGardenIP)
				routerOne.POST("/hotspot/walled-garden-ip", hotspotH.AddWalledGardenIP)
				routerOne.DELETE("/hotspot/walled-garden-ip/:wid", hotspotH.RemoveWalledGardenIP)

				// Vouchers
				routerOne.POST("/vouchers/generate", voucherH.Generate)
				routerOne.POST("/vouchers/cache", voucherH.CacheVoucher)
				routerOne.GET("/vouchers/print-data", voucherH.PrintData)
				routerOne.POST("/vouchers/sales", voucherH.RecordSale)
				routerOne.POST("/vouchers/import", voucherH.ImportSales)
				routerOne.POST("/vouchers/print", voucherH.PrintVouchers)

				// Reports (per-router)
				routerOne.GET("/reports/daily", reportH.GetDailyReport)
				routerOne.GET("/reports/monthly", reportH.GetMonthlyReport)
				routerOne.GET("/reports/resume", reportH.GetResumeReport)
				routerOne.GET("/reports/summary", reportH.GetDashboardSummary)
				routerOne.GET("/reports/export/csv", reportH.ExportCSV)
				routerOne.GET("/reports/export/excel", reportH.ExportExcel)

				// System
				routerOne.GET("/system/resource", systemH.GetSystemResource)
				routerOne.GET("/system/resource/history", systemH.GetSystemResourceHistory)
				routerOne.GET("/system/log", systemH.GetSystemLog)
				routerOne.GET("/system/clock", systemH.GetSystemClock)
				routerOne.GET("/system/identity", systemH.GetSystemIdentity)
				routerOne.GET("/system/routerboard", systemH.GetRouterboard)
				routerOne.POST("/system/reboot", systemH.Reboot)
				routerOne.POST("/system/shutdown", systemH.Shutdown)
				routerOne.GET("/system/expire-monitor", systemH.GetExpireMonitor)
				routerOne.POST("/system/expire-monitor/deploy", systemH.DeployExpireMonitor)
				routerOne.POST("/system/expire-monitor/remove", systemH.RemoveExpireMonitor)
				routerOne.GET("/system/schedulers", systemH.ListSchedulers)
				routerOne.POST("/system/schedulers", systemH.CreateScheduler)
				routerOne.PUT("/system/schedulers/:schedulerId", systemH.UpdateScheduler)
				routerOne.DELETE("/system/schedulers/:schedulerId", systemH.DeleteScheduler)
				routerOne.POST("/system/schedulers/:schedulerId/enable", systemH.EnableSchedulerByID)
				routerOne.POST("/system/schedulers/:schedulerId/disable", systemH.DisableSchedulerByID)
				routerOne.GET("/system/scripts", systemH.ListScripts)
				routerOne.POST("/system/scripts", systemH.CreateScript)
				routerOne.PUT("/system/scripts/:scriptId", systemH.UpdateScript)
				routerOne.DELETE("/system/scripts/:scriptId", systemH.DeleteScript)
				routerOne.POST("/system/scripts/:scriptId/run", systemH.RunScriptByID)
				routerOne.POST("/system/setup-logging", systemH.SetupLogging)
				routerOne.GET("/system/dashboard", systemH.GetDashboard)

				// PPP
				routerOne.GET("/ppp/secrets", pppH.ListSecrets)
				routerOne.POST("/ppp/secrets", pppH.AddSecret)
				routerOne.PUT("/ppp/secrets/:id", pppH.UpdateSecret)
				routerOne.DELETE("/ppp/secrets/:id", pppH.RemoveSecret)
				routerOne.GET("/ppp/active", pppH.ListActive)
				routerOne.DELETE("/ppp/active/:id", pppH.DisconnectActive)
				routerOne.GET("/ppp/inactive", pppH.ListInactive)
				routerOne.GET("/ppp/inactive/count", pppH.GetInactiveCount)
				routerOne.GET("/ppp/profiles", pppH.ListProfiles)

				// Network
				routerOne.GET("/network/interfaces", networkH.ListInterfaces)
				routerOne.GET("/network/traffic/:iface", networkH.GetInterfaceTraffic)
				routerOne.GET("/network/pools", networkH.ListPools)
				routerOne.GET("/network/queues", networkH.ListQueues)
				routerOne.GET("/network/nat", networkH.ListNATRules)
				routerOne.GET("/network/dhcp/leases", networkH.ListDHCPLeases)
				routerOne.DELETE("/network/dhcp/:id/release", networkH.ReleaseDHCPLease)

				// Quick print
				routerOne.GET("/quick-print", qpH.ListPackages)
				routerOne.POST("/quick-print", qpH.CreatePackage)
				routerOne.GET("/quick-print/:name", qpH.GetPackage)
				routerOne.PUT("/quick-print/:name", qpH.UpdatePackage)
				routerOne.DELETE("/quick-print/:name", qpH.RemovePackage)

				// Logs SSE
				routerOne.GET("/logs/stream/all", logSSEH.StreamAll)
				routerOne.GET("/logs/stream/hotspot", logSSEH.StreamHotspot)
				routerOne.GET("/logs/stream/ppp", logSSEH.StreamPPP)

				// Ping SSE
				routerOne.GET("/sse/ping", pingSSEH.Stream)

				// Telemetry SSE
				routerOne.GET("/sse/hotspot/users", telemetrySSEH.Stream("hotspot_user"))
				routerOne.GET("/sse/hotspot/active", telemetrySSEH.Stream("hotspot_active"))
				routerOne.GET("/sse/hotspot/inactive", telemetrySSEH.Stream("hotspot_inactive"))
				routerOne.GET("/sse/hotspot/bindings", telemetrySSEH.Stream("ip_binding"))
				routerOne.GET("/sse/ppp/secrets", telemetrySSEH.Stream("ppp_secret"))
				routerOne.GET("/sse/ppp/active", telemetrySSEH.Stream("ppp_active"))
				routerOne.GET("/sse/ppp/inactive", telemetrySSEH.Stream("ppp_inactive"))
				routerOne.GET("/sse/system/resource", telemetrySSEH.Stream("system_resource"))
				routerOne.GET("/sse/network/traffic/:iface", telemetrySSEH.StreamInterface)
				routerOne.GET("/sse/network/dhcp/leases", telemetrySSEH.Stream("dhcp_lease"))

				// Profile mappings
				routerOne.GET("/profile-mappings", profileMappingH.List)
				routerOne.PUT("/profile-mappings/:profileName", profileMappingH.Update)
				routerOne.DELETE("/profile-mappings/:profileName", profileMappingH.Delete)
			}
		}
	}

	_ = cfg
	return engine
}
