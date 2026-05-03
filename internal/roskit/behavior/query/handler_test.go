//go:build mikrotik

package query_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/quiqxiq/roskit/internal/roskit/behavior/query"
	"github.com/quiqxiq/roskit/internal/roskit/core/command"
	_ "github.com/quiqxiq/roskit/internal/roskit/core/definition"
	"github.com/quiqxiq/roskit/internal/roskit/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuery_NoCache_ReadsFromAPI(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)
	pool, cleanup := testhelpers.NewTestPool(t)
	defer cleanup()

	routerID := testhelpers.RouterID()
	h := query.NewHandler(pool, nil, slog.Default())

	meta := command.Lookup("ip/hotspot/user/print")
	require.NotNil(t, meta)

	results, err := h.Query(context.Background(), routerID, meta)
	require.NoError(t, err)
	assert.NotNil(t, results)
}

func TestQuery_QueryOne(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)
	pool, cleanup := testhelpers.NewTestPool(t)
	defer cleanup()

	routerID := testhelpers.RouterID()
	h := query.NewHandler(pool, nil, slog.Default())

	meta := command.Lookup("ip/hotspot/user/print")
	require.NotNil(t, meta)

	result, err := h.QueryOne(context.Background(), routerID, meta, "?name=admin")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "admin", result["name"])
}

func TestQuery_QueryOne_NotFound(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)
	pool, cleanup := testhelpers.NewTestPool(t)
	defer cleanup()

	routerID := testhelpers.RouterID()
	h := query.NewHandler(pool, nil, slog.Default())

	meta := command.Lookup("ip/hotspot/user/print")
	require.NotNil(t, meta)

	result, err := h.QueryOne(context.Background(), routerID, meta, "?name=nonexistent_user_xyz")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestQuery_Filters(t *testing.T) {
	testhelpers.SkipWithoutMikroTik(t)
	pool, cleanup := testhelpers.NewTestPool(t)
	defer cleanup()

	routerID := testhelpers.RouterID()
	h := query.NewHandler(pool, nil, slog.Default())

	meta := command.Lookup("ip/hotspot/user/print")
	require.NotNil(t, meta)

	results, err := h.Query(context.Background(), routerID, meta, "?name=admin")
	require.NoError(t, err)
	require.NotEmpty(t, results)

	for _, row := range results {
		assert.Equal(t, "admin", row["name"])
	}
}
