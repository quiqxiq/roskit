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
) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery(), middleware.LoggerMiddleware(), middleware.CORSMiddleware())

	cache := rdb

	userRepo := repository.NewUserRepo(db)
	routerRepo := repository.NewRouterRepo(db)
	saleRepo := repository.NewSaleRepo(db)
	templateRepo := repository.NewTemplateRepo(db)

	jwtSecret := []byte(cfg.JWTSecret)
	refreshSecret := []byte(cfg.JWTRefreshSecret)
	authSvc := services.NewAuthService(userRepo, cache, jwtSecret, refreshSecret)

	systemSvc := services.NewSystemService(bridge, tsReader, routerRepo, cache)
	hotspotSvc := services.NewHotspotService(bridge, cache)
	voucherSvc := services.NewVoucherService(bridge, saleRepo, routerRepo, profileRepo, cache)
	reportSvc := services.NewReportService(saleRepo, cache)
	templateSvc := services.NewTemplateService(templateRepo)
	if err := templateSvc.SeedDefaults(context.Background()); err != nil {
		slog.Default().Warn("template seed defaults failed", "error", err)
	}

	auditRepo := repository.NewAuditRepo(db)
	auditLogger := middleware.NewAuditLogger(auditRepo)

	routerH := handlers.NewRouterHandler(routerSvc)
	hotspotH := handlers.NewHotspotHandler(hotspotSvc)
	voucherH := handlers.NewVoucherHandler(voucherSvc, templateSvc, hotspotSvc)
	reportH := handlers.NewReportHandler(reportSvc)
	authH := handlers.NewAuthHandler(authSvc, auditLogger)
	eventH := handlers.NewEventHandler(routerRepo, saleRepo, profileRepo, bridge, cache)
	systemH := handlers.NewSystemHandler(systemSvc)
	pppH := handlers.NewPPPHandler(bridge)
	networkH := handlers.NewNetworkHandler(bridge)
	qpH := handlers.NewQuickPrintHandler(bridge)
	templateH := handlers.NewTemplateHandler(templateSvc, voucherSvc)
	logSSEH := sse.NewLogSSEHandler(subscriber, bridge)
	telemetrySSEH := sse.NewTelemetrySSEHandler(subscriber)
	statusSvc := services.NewStatusService(routerRepo, bridge)
	statusH := handlers.NewStatusHandler(statusSvc)
	healthH := handlers.NewHealthHandler(db, cache, bridge)
	profileMappingH := handlers.NewProfileMappingHandler(profileRepo)

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

		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware(authSvc))
		{
			protected.GET("/auth/me", authH.Me)
			protected.PUT("/auth/password", authH.ChangePassword)

			routers := protected.Group("/routers")
			{
				routers.GET("", routerH.List)
				routers.GET("/:routerId", routerH.Get)
				routers.POST("", routerH.Create)
				routers.PUT("/:routerId", routerH.Update)
				routers.DELETE("/:routerId", routerH.Delete)
			routers.POST("/:routerId/test", routerH.TestConnection)
			routers.POST("/:routerId/logo", routerH.UploadLogo)
			routers.GET("/:routerId/logo", routerH.GetLogo)
			routers.POST("/migrate", routerH.MigrateConfig)
			}

			hotspots := protected.Group("/routers/:routerId/hotspot")
			{
				hotspots.GET("/users", hotspotH.ListUsers)
				hotspots.GET("/users/count", hotspotH.GetUserCount)
				hotspots.GET("/users/export", hotspotH.ExportUsers)
				hotspots.GET("/users/:id", hotspotH.GetUser)
				hotspots.POST("/users", hotspotH.AddUser)
				hotspots.PUT("/users/:id", hotspotH.UpdateUser)
				hotspots.DELETE("/users/:id", hotspotH.RemoveUser)
				hotspots.POST("/users/:id/reset-counters", hotspotH.ResetUserCounters)

				hotspots.GET("/profiles", hotspotH.ListProfiles)
				hotspots.GET("/profiles/:id", hotspotH.GetProfile)
				hotspots.POST("/profiles", hotspotH.AddProfile)
				hotspots.PUT("/profiles/:id", hotspotH.UpdateProfile)
				hotspots.DELETE("/profiles/:id", hotspotH.RemoveProfile)

				hotspots.GET("/active", hotspotH.ListActive)
				hotspots.DELETE("/active/:id", hotspotH.RemoveActive)
				hotspots.POST("/active/:id/disconnect", hotspotH.DisconnectUser)

				hotspots.GET("/hosts", hotspotH.ListHosts)
				hotspots.DELETE("/hosts/:id", hotspotH.RemoveHost)

				hotspots.GET("/servers", hotspotH.ListServers)

				hotspots.GET("/cookies", hotspotH.ListCookies)
				hotspots.DELETE("/cookies/:id", hotspotH.RemoveCookie)

				hotspots.GET("/bindings", hotspotH.ListIPBindings)
				hotspots.POST("/bindings", hotspotH.AddIPBinding)
				hotspots.PUT("/bindings/:id", hotspotH.UpdateIPBinding)
				hotspots.DELETE("/bindings/:id", hotspotH.RemoveIPBinding)
				hotspots.POST("/bindings/:id/enable", hotspotH.EnableIPBinding)
				hotspots.POST("/bindings/:id/disable", hotspotH.DisableIPBinding)

				hotspots.GET("/walled-garden", hotspotH.ListWalledGarden)
				hotspots.POST("/walled-garden", hotspotH.AddWalledGarden)
				hotspots.DELETE("/walled-garden/:wid", hotspotH.RemoveWalledGarden)
				hotspots.GET("/walled-garden-ip", hotspotH.ListWalledGardenIP)
				hotspots.POST("/walled-garden-ip", hotspotH.AddWalledGardenIP)
				hotspots.DELETE("/walled-garden-ip/:wid", hotspotH.RemoveWalledGardenIP)
			}

			vouchers := protected.Group("/routers/:routerId/vouchers")
			{
				vouchers.POST("/generate", voucherH.Generate)
				vouchers.POST("/cache", voucherH.CacheVoucher)
				vouchers.GET("/print-data", voucherH.PrintData)
				vouchers.POST("/sales", voucherH.RecordSale)
				vouchers.POST("/import", voucherH.ImportSales)
			vouchers.POST("/print", voucherH.PrintVouchers)
			}

			reports := protected.Group("/routers/:routerId/reports")
			{
				reports.GET("/daily", reportH.GetDailyReport)
				reports.GET("/monthly", reportH.GetMonthlyReport)
				reports.GET("/resume", reportH.GetResumeReport)
				reports.GET("/summary", reportH.GetDashboardSummary)
				reports.GET("/export/csv", reportH.ExportCSV)
				reports.GET("/export/excel", reportH.ExportExcel)
			}

			system := protected.Group("/routers/:routerId/system")
			{
				system.GET("/resource", systemH.GetSystemResource)
				system.GET("/resource/history", systemH.GetSystemResourceHistory)
				system.GET("/log", systemH.GetSystemLog)
				system.GET("/clock", systemH.GetSystemClock)
				system.GET("/identity", systemH.GetSystemIdentity)
				system.GET("/routerboard", systemH.GetRouterboard)
				system.POST("/reboot", systemH.Reboot)
				system.POST("/shutdown", systemH.Shutdown)
				system.GET("/expire-monitor", systemH.GetExpireMonitor)
				system.POST("/expire-monitor/deploy", systemH.DeployExpireMonitor)
				system.POST("/expire-monitor/remove", systemH.RemoveExpireMonitor)
				system.GET("/schedulers", systemH.ListSchedulers)
				system.POST("/schedulers", systemH.CreateScheduler)
				system.PUT("/schedulers/:schedulerId", systemH.UpdateScheduler)
				system.DELETE("/schedulers/:schedulerId", systemH.DeleteScheduler)
				system.POST("/schedulers/:schedulerId/enable", systemH.EnableSchedulerByID)
				system.POST("/schedulers/:schedulerId/disable", systemH.DisableSchedulerByID)
			system.GET("/scripts", systemH.ListScripts)
			system.POST("/scripts", systemH.CreateScript)
			system.PUT("/scripts/:scriptId", systemH.UpdateScript)
			system.DELETE("/scripts/:scriptId", systemH.DeleteScript)
			system.POST("/scripts/:scriptId/run", systemH.RunScriptByID)
			system.POST("/setup-logging", systemH.SetupLogging)
				system.GET("/dashboard", systemH.GetDashboard)
			}

			ppp := protected.Group("/routers/:routerId/ppp")
			{
				ppp.GET("/secrets", pppH.ListSecrets)
				ppp.POST("/secrets", pppH.AddSecret)
				ppp.PUT("/secrets/:id", pppH.UpdateSecret)
				ppp.DELETE("/secrets/:id", pppH.RemoveSecret)
				ppp.GET("/active", pppH.ListActive)
				ppp.DELETE("/active/:id", pppH.DisconnectActive)
				ppp.GET("/profiles", pppH.ListProfiles)
			}

			net := protected.Group("/routers/:routerId/network")
			{
				net.GET("/interfaces", networkH.ListInterfaces)
				net.GET("/traffic/:iface", networkH.GetInterfaceTraffic)
				net.GET("/pools", networkH.ListPools)
				net.GET("/queues", networkH.ListQueues)
				net.GET("/nat", networkH.ListNATRules)
				net.GET("/dhcp/leases", networkH.ListDHCPLeases)
				net.DELETE("/dhcp/:id/release", networkH.ReleaseDHCPLease)
			}

			qp := protected.Group("/routers/:routerId/quick-print")
			{
				qp.GET("", qpH.ListPackages)
				qp.POST("", qpH.CreatePackage)
				qp.GET("/:name", qpH.GetPackage)
				qp.PUT("/:name", qpH.UpdatePackage)
				qp.DELETE("/:name", qpH.RemovePackage)
			}

			logs := protected.Group("/routers/:routerId/logs")
			{
				logs.GET("/stream/all", logSSEH.StreamAll)
				logs.GET("/stream/hotspot", logSSEH.StreamHotspot)
				logs.GET("/stream/ppp", logSSEH.StreamPPP)
			}

			telemetry := protected.Group("/routers/:routerId/sse")
			{
				telemetry.GET("/hotspot/users", telemetrySSEH.Stream("hotspot_user"))
				telemetry.GET("/hotspot/active", telemetrySSEH.Stream("hotspot_active"))
				telemetry.GET("/hotspot/inactive", telemetrySSEH.Stream("hotspot_inactive"))
				telemetry.GET("/hotspot/bindings", telemetrySSEH.Stream("ip_binding"))
				telemetry.GET("/ppp/secrets", telemetrySSEH.Stream("ppp_secret"))
				telemetry.GET("/ppp/active", telemetrySSEH.Stream("ppp_active"))
				telemetry.GET("/ppp/inactive", telemetrySSEH.Stream("ppp_inactive"))
				telemetry.GET("/system/resource", telemetrySSEH.Stream("system_resource"))
				telemetry.GET("/network/traffic/:iface", telemetrySSEH.Stream("interface_traffic"))
				telemetry.GET("/network/dhcp/leases", telemetrySSEH.Stream("dhcp_lease"))
			}

			templates := protected.Group("/routers/:routerId/templates")
			{
				templates.GET("", templateH.List)
				templates.POST("", templateH.Create)
				templates.GET("/:templateId", templateH.Get)
				templates.PUT("/:templateId", templateH.Update)
				templates.DELETE("/:templateId", templateH.Delete)
				templates.POST("/render", templateH.Render)
				templates.POST("/seed-defaults", templateH.SeedDefaults)
			}
		}

		api.GET("/status", statusH.GetUserStatus)

		events := api.Group("/events")
		{
			events.POST("/on-login", onLoginRL, eventH.OnLoginEvent)
			events.GET("/health", eventH.HealthCheck)
		}

		protected.GET("/routers/:routerId/profile-mappings", profileMappingH.List)
		protected.PUT("/routers/:routerId/profile-mappings/:profileName", profileMappingH.Update)
		protected.DELETE("/routers/:routerId/profile-mappings/:profileName", profileMappingH.Delete)
	}

	return engine
}
