//go:build mikrotik

package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- LogStream ---

// TestBridge_LogStream_All verifies that /log/print =follow delivers at least
// one entry. RouterOS replays buffered entries first so we expect immediate data.
func TestBridge_LogStream_All(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ch, err := bridge.LogStream(ctx, testhelpers.RouterID(), "")
	require.NoError(t, err, "LogStream should start without error")

	select {
	case entry, ok := <-ch:
		require.True(t, ok, "channel should not close immediately")
		assert.NotEmpty(t, entry["topics"], "log entry must have topics field")
		assert.NotEmpty(t, entry["message"], "log entry must have message field")
		t.Logf("log entry: topics=%s time=%s message=%s", entry["topics"], entry["time"], entry["message"])
	case <-ctx.Done():
		t.Fatal("timeout: no log entries received within 30s — check if RouterOS is producing logs")
	}
}

// TestBridge_LogStream_HotspotFilter verifies the stream starts and is
// filtered by hotspot topics. We only assert the stream opens without error
// since hotspot activity may not exist on every test router.
func TestBridge_LogStream_HotspotFilter(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ch, err := bridge.LogStream(ctx, testhelpers.RouterID(), "hotspot")
	require.NoError(t, err, "LogStream hotspot filter should start without error")

	// Drain any entries that arrive; don't fail if none come — hotspot may be idle.
	received := 0
	for {
		select {
		case entry, ok := <-ch:
			if !ok {
				t.Logf("hotspot log stream closed after %d entries", received)
				return
			}
			received++
			t.Logf("hotspot log entry: topics=%s message=%s", entry["topics"], entry["message"])
			if received >= 3 {
				cancel()
			}
		case <-ctx.Done():
			t.Logf("hotspot log stream: received %d entries before timeout", received)
			return
		}
	}
}

// TestBridge_LogStream_Cancel verifies that cancelling the context causes
// the output channel to close promptly.
func TestBridge_LogStream_Cancel(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithCancel(context.Background())

	ch, err := bridge.LogStream(ctx, testhelpers.RouterID(), "")
	require.NoError(t, err)

	// Cancel right after at least one entry arrives (confirms stream is live).
	select {
	case _, ok := <-ch:
		require.True(t, ok)
		cancel()
	case <-time.After(30 * time.Second):
		cancel()
		t.Fatal("timeout: no log entries before cancel — stream may be stalled")
	}

	// Channel must close within 5s of cancel.
	select {
	case _, ok := <-ch:
		if ok {
			// Drain a few more — that's fine. Eventually it should close.
			drainCtx, done := context.WithTimeout(context.Background(), 5*time.Second)
			defer done()
			for {
				select {
				case _, open := <-ch:
					if !open {
						return
					}
				case <-drainCtx.Done():
					t.Fatal("channel did not close within 5s after context cancel")
				}
			}
		}
	case <-time.After(5 * time.Second):
		t.Fatal("channel did not close within 5s after context cancel")
	}
}

// --- InterfaceTrafficStream ---

// TestBridge_InterfaceTrafficStream_ReceivesStats verifies that
// /interface/monitor-traffic delivers at least one update with expected fields.
func TestBridge_InterfaceTrafficStream_ReceivesStats(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	// Pick the first available interface.
	ifaces, err := bridge.ListInterfaces(context.Background(), testhelpers.RouterID())
	require.NoError(t, err, "must be able to list interfaces")
	require.NotEmpty(t, ifaces, "router must have at least one interface")
	iface := ifaces[0]["name"]
	require.NotEmpty(t, iface, "interface name must not be empty")
	t.Logf("streaming traffic for interface: %s", iface)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ch, err := bridge.InterfaceTrafficStream(ctx, testhelpers.RouterID(), iface)
	require.NoError(t, err, "InterfaceTrafficStream should start without error")

	select {
	case stat, ok := <-ch:
		require.True(t, ok, "channel should not close immediately")
		assert.Equal(t, iface, stat["name"], "name field must match requested interface")
		assert.Contains(t, stat, "rx-bits-per-second", "must contain rx-bits-per-second")
		assert.Contains(t, stat, "tx-bits-per-second", "must contain tx-bits-per-second")
		t.Logf("traffic stat: name=%s rx=%s tx=%s",
			stat["name"], stat["rx-bits-per-second"], stat["tx-bits-per-second"])
	case <-ctx.Done():
		t.Fatalf("timeout: no traffic stats for interface %q within 15s", iface)
	}
}

// TestBridge_InterfaceTrafficStream_Cancel verifies the stream stops cleanly.
func TestBridge_InterfaceTrafficStream_Cancel(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ifaces, err := bridge.ListInterfaces(context.Background(), testhelpers.RouterID())
	require.NoError(t, err)
	require.NotEmpty(t, ifaces)
	iface := ifaces[0]["name"]

	ctx, cancel := context.WithCancel(context.Background())

	ch, err := bridge.InterfaceTrafficStream(ctx, testhelpers.RouterID(), iface)
	require.NoError(t, err)

	// Wait for one stat, then cancel.
	select {
	case _, ok := <-ch:
		require.True(t, ok)
		cancel()
	case <-time.After(15 * time.Second):
		cancel()
		t.Fatalf("timeout: no traffic stat for %q", iface)
	}

	// Channel must close within 5s.
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal("InterfaceTrafficStream channel did not close within 5s after cancel")
	}
}

// TestBridge_LogAndTraffic_Concurrent runs both streams simultaneously to
// verify that tag-multiplexing on the client connection handles concurrent
// listeners without data corruption or stream stall.
func TestBridge_LogAndTraffic_Concurrent(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ifaces, err := bridge.ListInterfaces(context.Background(), testhelpers.RouterID())
	require.NoError(t, err)
	require.NotEmpty(t, ifaces)
	iface := ifaces[0]["name"]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rid := testhelpers.RouterID()

	logCh, err := bridge.LogStream(ctx, rid, "")
	require.NoError(t, err, "log stream must start")

	trafficCh, err := bridge.InterfaceTrafficStream(ctx, rid, iface)
	require.NoError(t, err, "traffic stream must start")

	gotLog := false
	gotTraffic := false

	for !gotLog || !gotTraffic {
		select {
		case entry, ok := <-logCh:
			require.True(t, ok, "log channel closed unexpectedly")
			assert.NotEmpty(t, entry["topics"])
			gotLog = true
			t.Logf("concurrent: log entry topics=%s", entry["topics"])

		case stat, ok := <-trafficCh:
			require.True(t, ok, "traffic channel closed unexpectedly")
			assert.Equal(t, iface, stat["name"])
			gotTraffic = true
			t.Logf("concurrent: traffic stat rx=%s tx=%s", stat["rx-bits-per-second"], stat["tx-bits-per-second"])

		case <-ctx.Done():
			t.Fatalf("timeout — gotLog=%v gotTraffic=%v: both streams must deliver data concurrently", gotLog, gotTraffic)
		}
	}
}
