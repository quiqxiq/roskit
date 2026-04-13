// Package main demonstrates the usage of the roskit telemetry library.
// This example shows how to:
//  1. Initialize infrastructure (Redis, InfluxDB)
//  2. Configure routers and stream specs
//  3. Start the collector engine
//  4. Read cached data from Redis (AI agent pattern — no router hit)
//  5. Gracefully shutdown on OS signal
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/quiqxiq/roskit/internal/collector"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/internal/repository"
	"github.com/quiqxiq/roskit/internal/spec"
	"github.com/quiqxiq/roskit/internal/usecase"
)

// getEnv returns the value of an environment variable or a fallback default.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	// ─── Load .env ──────────────────────────────────────────────────────────────
	// Load from docker/.env (relative to project root).
	_ = godotenv.Load("docker/.env")

	// ─── Logger Setup ────────────────────────────────────────────────────────────
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	// ─── Infrastructure Setup ────────────────────────────────────────────────────

	// Initialize Redis repository (cache + pub/sub).
	redisRepo, err := repository.NewRedisRepository(repository.RedisConfig{
		Addr:     "localhost:6380",
		Password: "",
		DB:       0,
	}, logger)
	if err != nil {
		logger.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer redisRepo.Close()

	// Initialize InfluxDB writer (time-series persistence).
	influxWriter, err := repository.NewInfluxDBWriter(repository.InfluxDBConfig{
		Host:          getEnv("INFLUXDB3_HOST", "http://localhost:8181"),
		Token:         getEnv("INFLUXDB3_AUTH_TOKEN", ""),
		Database:      getEnv("INFLUXDB3_DATABASE", "roskit"),
		BatchSize:     5000,
		FlushInterval: time.Second,
	}, logger)
	if err != nil {
		logger.Error("failed to connect to InfluxDB", "error", err)
		os.Exit(1)
	}
	defer influxWriter.Close()

	// ─── Business Logic (Use Case) ──────────────────────────────────────────────

	// Create the telemetry use case with dependency injection.
	// The use case depends on interfaces, not concrete implementations.
	uc := usecase.NewTelemetryUseCase(
		influxWriter, // TimeSeriesWriter interface
		redisRepo,    // CacheRepository interface
		redisRepo,    // PubSubPublisher interface (same Redis, different interface)
		logger,
	)

	// ─── Collector Engine ────────────────────────────────────────────────────────

	// Create the collector engine.
	engine := collector.NewEngine(uc, logger)

	// Define stream specs — each spec is a separate data source.
	// Adding new specs is as simple as creating a new file in internal/spec/.
	defaultSpecs := []spec.StreamSpec{
		// ── System Health ──
		spec.NewSystemResourceSpec(),       // /system/resource/print interval (CPU/memory/disk)
		spec.NewSystemHealthSpec(),         // /system/health/print interval (temperature/voltage)
		spec.NewLogSpec(),                  // /log/print follow (router logs)

		// ── Interface & Bandwidth ──
		spec.NewInterfaceStatsSpec(),       // /interface/print follow (bandwidth)
		spec.NewQueueSimpleStatsSpec(),     // /queue/simple/print stats (simple queue)
		spec.NewQueueTreeStatsSpec(),       // /queue/tree/print stats (tree queue)

		// ── Routing Protocols ──
		spec.NewBGPSessionSpec(),           // /routing/bgp/session/print follow (BGP peers)
		spec.NewOSPFNeighborSpec(),         // /routing/ospf/neighbor/print follow (OSPF adjacency)

		// ── Network Layer ──
		spec.NewIPAddressSpec(),            // /ip/address/print follow (IP assignments)
		spec.NewIPRouteSpec(),              // /ip/route/print follow (routing table)
		spec.NewARPTableSpec(),             // /ip/arp/print follow (ARP entries)
		spec.NewIPNeighborSpec(),           // /ip/neighbor/print follow (LLDP/CDP/MNDP)

		// ── Firewall & Security ──
		spec.NewFirewallFilterSpec(),       // /ip/firewall/filter/print stats (rule counters)
		spec.NewFirewallNATSpec(),          // /ip/firewall/nat/print stats (NAT counters)
		spec.NewFirewallConnectionSpec(),   // /ip/firewall/connection/print follow (conntrack)

		// ── User Sessions ──
		spec.NewHotspotActiveSpec(),        // /ip/hotspot/active/print follow (hotspot users)
		spec.NewPPPActiveSpec(),            // /ppp/active/print follow (PPPoE sessions)
		spec.NewPPPProfileSpec(),           // /ppp/profile/print follow (profile config)
		spec.NewPPPSecretSpec(),            // /ppp/secret/print follow (user accounts)
		spec.NewDHCPLeaseSpec(),            // /ip/dhcp-server/lease/print follow (DHCP leases)

		// ── Wireless ──
		spec.NewWirelessRegistrationSpec(), // /interface/wireless/registration-table/print follow
	}

	// Register routers. In production, these would come from a config file or database.
	// Each router has its own reconnect/health check configuration.
	engine.AddRouter(domain.RouterConfig{
		ID:       "core-01",
		Address:  getEnv("MIKROTIK_ADDRESS", "192.168.233.1:8728"),
		Username: getEnv("MIKROTIK_USERNAME", "admin"),
		Password: getEnv("MIKROTIK_PASSWORD", "r00t"),
		UseTLS:   false,

		// Per-router connection tuning
		ReconnectInterval:    5 * time.Second,  // Start retrying after 5s on disconnect.
		MaxReconnectInterval: 60 * time.Second, // Cap backoff at 60s.
		HealthCheckInterval:  30 * time.Second, // Ping router every 30s to detect stale connections.
		DialTimeout:          10 * time.Second, // Give up connecting after 10s.
	}, defaultSpecs)

	// You can add more routers — each gets its own persistent connection in the pool
	// with independent reconnect settings:
	// engine.AddRouter(domain.RouterConfig{
	// 	ID:       "edge-01",
	// 	Address:  "10.0.0.1:8729",
	// 	Username: "monitor",
	// 	Password: "secret",
	// 	UseTLS:   true,
	//
	// 	// Edge router — aggressive reconnect (mission-critical link).
	// 	ReconnectInterval:    2 * time.Second,
	// 	MaxReconnectInterval: 30 * time.Second,
	// 	HealthCheckInterval:  15 * time.Second,
	// 	DialTimeout:          5 * time.Second,
	// }, defaultSpecs)

	// ─── Start Engine ────────────────────────────────────────────────────────────

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start: pool connects all routers → health monitor starts → collectors begin streaming.
	engine.Start(ctx)

	logger.Info("roskit telemetry engine started",
		"routers", engine.RouterIDs(),
		"specs", len(defaultSpecs),
	)

	// Log connection pool status.
	for routerID, state := range engine.Status() {
		logger.Info("connection pool status",
			"router", routerID,
			"state", state.String(),
		)
	}


	// ─── AI Agent Pattern: Read from Redis Cache ─────────────────────────────────
	// Demonstrate reading cached data directly from Redis without hitting the router.
	// An AI agent or dashboard can call these methods to get instant snapshots.

	go func() {
		// Wait for some data to be collected.
		time.Sleep(5 * time.Second)

		logger.Info("─── AI Agent: Reading cached data from Redis ───")

		// Read system resource snapshot.
		sysData, err := redisRepo.GetSnapshot(ctx, "roskit:core-01:system_resource:system")
		if err != nil {
			logger.Error("failed to read system resource cache", "error", err)
		} else if len(sysData) > 0 {
			logger.Info("cached system resource",
				"cpu_load", sysData["cpu_load"],
				"free_memory", sysData["free_memory"],
				"board_name", sysData["board_name"],
				"version", sysData["version"],
			)
		}

		// Read all interface stats (scan for keys matching pattern).
		readAllInterfaceStats(ctx, redisRepo, logger)
	}()

	// ─── Pub/Sub Subscriber Example ──────────────────────────────────────────────
	// Demonstrate subscribing to real-time telemetry events.

	go func() {
		pubsub := redisRepo.Client().Subscribe(ctx, "roskit:telemetry:core-01")
		defer pubsub.Close()

		logger.Info("subscribed to Pub/Sub channel", "channel", "roskit:telemetry:core-01")

		ch := pubsub.Channel()
		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					return
				}
				// In production, parse the JSON and react accordingly.
				logger.Debug("pub/sub event received",
					"channel", msg.Channel,
					"payload_len", len(msg.Payload),
				)
			case <-ctx.Done():
				return
			}
		}
	}()

	// ─── Graceful Shutdown ───────────────────────────────────────────────────────

	// Wait for interrupt signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	logger.Info("received shutdown signal", "signal", sig)

	// Cancel context and stop engine.
	cancel()
	engine.Stop()

	// Flush remaining InfluxDB data.
	flushCtx, flushCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer flushCancel()
	if err := influxWriter.Flush(flushCtx); err != nil {
		logger.Error("failed to flush InfluxDB", "error", err)
	}

	logger.Info("roskit telemetry engine shutdown complete")
}

// readAllInterfaceStats scans Redis for all cached interface stats and prints them.
// This demonstrates how an external application (AI agent, dashboard) can retrieve
// the latest state of all interfaces without querying the MikroTik router.
func readAllInterfaceStats(ctx context.Context, repo *repository.RedisRepository, logger *slog.Logger) {
	// Scan for all interface stat keys.
	var cursor uint64
	var keys []string

	for {
		var err error
		var batch []string
		batch, cursor, err = repo.Client().Scan(ctx, cursor, "roskit:*:interface_stats:*", 100).Result()
		if err != nil {
			logger.Error("failed to scan interface keys", "error", err)
			return
		}
		keys = append(keys, batch...)
		if cursor == 0 {
			break
		}
	}

	for _, key := range keys {
		data, err := repo.GetSnapshot(ctx, key)
		if err != nil {
			logger.Error("failed to read interface cache", "key", key, "error", err)
			continue
		}

		// Extract interface name from key.
		parts := strings.Split(key, ":")
		ifaceName := parts[len(parts)-1]

		fmt.Printf("  Interface %s: rx=%s bytes, tx=%s bytes, rx_drop=%s, tx_drop=%s\n",
			ifaceName,
			data["rx_byte"],
			data["tx_byte"],
			data["rx_drop"],
			data["tx_drop"],
		)
	}
}
