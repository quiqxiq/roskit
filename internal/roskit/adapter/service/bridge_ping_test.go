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

func TestBridge_PingStream_FiniteCount(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rid := testhelpers.RouterID()
	ch, err := bridge.PingStream(ctx, rid, "127.0.0.1", 3)
	require.NoError(t, err)

	count := 0
	for result := range ch {
		count++
		assert.NotEmpty(t, result.Seq, "seq should be set")
		assert.NotEmpty(t, result.Host, "host should be set")
		assert.False(t, result.At.IsZero(), "At should be set")
		// status is only set by RouterOS on failure (timeout); empty means success
	}

	assert.Equal(t, 3, count, "should receive exactly 3 ping results")
}

func TestBridge_PingStream_Continuous(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)

	bridge, cleanup := testhelpers.NewTestBridge(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rid := testhelpers.RouterID()
	ch, err := bridge.PingStream(ctx, rid, "127.0.0.1", 0)
	require.NoError(t, err)

	received := 0
	for result := range ch {
		received++
		assert.NotEmpty(t, result.Seq)
		assert.NotEmpty(t, result.Host)
		if received >= 3 {
			cancel()
		}
	}

	assert.GreaterOrEqual(t, received, 1, "should receive at least 1 ping result before cancel")
}
