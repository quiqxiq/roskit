package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/quiqxiq/roskit/internal/roskit/behavior"
	bstream "github.com/quiqxiq/roskit/internal/roskit/behavior/stream"
	_ "github.com/quiqxiq/roskit/internal/roskit/core/definition"
	"github.com/quiqxiq/roskit/internal/roskit/execution"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/pubsub"
)

func main() {
	_ = godotenv.Load()

	host := envOrDefault("MIKROTIK_HOST", "192.168.230.2:8728")
	user := envOrDefault("MIKROTIK_USER", "admin")
	pass := envOrDefault("MIKROTIK_PASS", "r00t")
	rid := envOrDefault("MIKROTIK_ID", "test-cli")

	logger := slog.Default()
	cfg := execution.ConnConfig{
		RouterID:    rid,
		Address:     host,
		Username:    user,
		Password:    pass,
		DialTimeout: 10 * time.Second,
	}

	fmt.Println("╔══════════════════════════════════════════════╗")
	fmt.Println("║       Roskit Live Test — Real Router         ║")
	fmt.Println("╠══════════════════════════════════════════════╣")
	fmt.Printf("║  Host   : %-35s║\n", host)
	fmt.Printf("║  User   : %-35s║\n", user)
	fmt.Printf("║  Router : %-35s║\n", rid)
	fmt.Println("╚══════════════════════════════════════════════╝")
	fmt.Println()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\n🛑 Shutting down...")
		cancel()
	}()

	pool := execution.NewPool(logger)
	pool.Register(cfg)
	pool.Start(ctx)

	waitCtx, waitCancel := context.WithTimeout(ctx, 15*time.Second)
	defer waitCancel()
	if err := pool.WaitConnected(waitCtx, rid); err != nil {
		fmt.Fprintf(os.Stderr, "✗ Failed to connect: %v\n", err)
		pool.Stop()
		os.Exit(1)
	}
	fmt.Println("✓ Connected to router")
	fmt.Println()

	// ──────────────────────────────────────────────────────────
	// Test 1: Interface list & traffic monitoring
	// ──────────────────────────────────────────────────────────
	fmt.Println("┌─ TEST 1: Interface Traffic Monitor ─────────────")
	testInterfaceTraffic(ctx, pool, rid)
	fmt.Println("└─────────────────────────────────────────────────")
	fmt.Println()

	// ──────────────────────────────────────────────────────────
	// Test 2: Log streaming (hotspot filter)
	// ──────────────────────────────────────────────────────────
	fmt.Println("┌─ TEST 2: Log Stream (hotspot filter) ───────────")
	testLogStream(ctx, pool, rid)
	fmt.Println("└─────────────────────────────────────────────────")
	fmt.Println()

	pool.Stop()
	fmt.Println("✓ All tests completed.")
}

func testInterfaceTraffic(ctx context.Context, pool *execution.Pool, routerID string) {
	logPub := &consoleLogPublisher{}
	ifaceMon := bstream.NewInterfaceMonitorManager(pool, &noopSink{}, slog.Default())

	conn, err := pool.Borrow(ctx, routerID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ✗ borrow failed: %v\n", err)
		return
	}
	defer pool.Return(routerID, conn)

	reply, err := conn.RunContext(ctx, "/interface/print")
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ✗ /interface/print failed: %v\n", err)
		return
	}

	var names []string
	for _, s := range reply.Re {
		if name, ok := s.Map["name"]; ok && name != "" {
			names = append(names, name)
		}
	}

	if len(names) == 0 {
		fmt.Println("  (no interfaces found)")
		return
	}

	fmt.Printf("  Found %d interfaces: %v\n", len(names), names)

	for _, iface := range names {
		displayMonitorSnapshot(ctx, pool, routerID, iface)
	}

	ifaceMon.SyncInterfaces(ctx, routerID, names)
	time.Sleep(2 * time.Second)

	count := ifaceMon.ActiveCount()
	fmt.Printf("  Active monitors: %d (expected %d)\n", count, len(names))

	ifaceMon.StopAll(routerID)
	_ = logPub
}

func displayMonitorSnapshot(ctx context.Context, pool *execution.Pool, routerID, iface string) {
	conn, err := pool.Borrow(ctx, routerID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ✗ borrow for %s failed: %v\n", iface, err)
		return
	}
	defer pool.Return(routerID, conn)

	reply, err := conn.RunContext(ctx, "/interface/monitor-traffic", "=interface="+iface, "=once")
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ✗ monitor-traffic %s failed: %v\n", iface, err)
		return
	}

	for _, s := range reply.Re {
		m := s.Map
		fmt.Printf("  [%-15s] rx=%s bps  tx=%s bps  rx-pkt=%s/s  tx-pkt=%s/s\n",
			m["name"],
			m["rx-bits-per-second"],
			m["tx-bits-per-second"],
			m["rx-packets-per-second"],
			m["tx-packets-per-second"],
		)
	}
}

func testLogStream(ctx context.Context, pool *execution.Pool, routerID string) {
	pub := &consoleLogPublisher{}
	logMgr := bstream.NewLogManager(pool, pub, slog.Default())

	logCtx, logCancel := context.WithTimeout(ctx, 10*time.Second)
	defer logCancel()

	fmt.Println("  Starting hotspot log stream (10s) ...")
	logMgr.StartAll(logCtx, routerID)

	select {
	case <-logCtx.Done():
		pub.mu.Lock()
		count := pub.count
		pub.mu.Unlock()
		fmt.Printf("  Received %d log lines in 10s\n", count)
	}

	pub.mu.Lock()
	recent := pub.recent
	pub.mu.Unlock()
	if len(recent) > 0 {
		fmt.Println("  Recent log lines:")
		start := len(recent) - 5
		if start < 0 {
			start = 0
		}
		for _, line := range recent[start:] {
			fmt.Printf("    %s\n", line)
		}
	}
}

type noopSink struct{}

func (n *noopSink) OnEvent(_ context.Context, _ behavior.StreamEvent) error { return nil }

type consoleLogPublisher struct {
	mu     sync.Mutex
	count  int
	recent []string
}

func (p *consoleLogPublisher) Publish(_ context.Context, channel string, msg pubsub.Message) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.count++
	var detail string
	switch msg.Type {
	case "log":
		if t, ok := msg.Fields["time"]; ok {
			detail += t + " "
		}
		if topics, ok := msg.Fields["topics"]; ok {
			detail += "[" + topics + "] "
		}
		if m, ok := msg.Fields["message"]; ok {
			detail += m
		}
	default:
		detail = fmt.Sprintf("measurement=%s fields=%d", msg.Measurement, len(msg.Fields))
	}
	line := fmt.Sprintf("[%s] %s", msg.Timestamp.Format("15:04:05"), detail)
	p.recent = append(p.recent, line)
	if len(p.recent) > 20 {
		p.recent = p.recent[1:]
	}
	return nil
}

func (p *consoleLogPublisher) Close() error { return nil }

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
