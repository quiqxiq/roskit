package handlers_test

// HotspotHandler full integration tests (ListUsers, AddUser, etc.) require a
// real MikroTik router because Bridge depends on *orchestrator.Dispatcher (concrete).
// HTTP-layer parsing tests are here; bridge-backed tests are in bridge_hotspot_test.go.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roskit/internal/api/handlers"
	"github.com/quiqxiq/roskit/internal/services"
	"github.com/stretchr/testify/assert"
)

func newNilHotspotHandler() *handlers.HotspotHandler {
	// nil bridge is safe as long as handler returns before calling svc methods
	return handlers.NewHotspotHandler(services.NewHotspotService(nil, nil, nil, nil, nil))
}

func TestHotspotHandler_ListUsers_InvalidRouterID(t *testing.T) {
	h := newNilHotspotHandler()
	router := gin.New()
	router.GET("/routers/:routerId/hotspot/users", h.ListUsers)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/routers/abc/hotspot/users", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHotspotHandler_GetUser_InvalidRouterID(t *testing.T) {
	h := newNilHotspotHandler()
	router := gin.New()
	router.GET("/routers/:routerId/hotspot/users/:id", h.GetUser)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/routers/bad/hotspot/users/user1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHotspotHandler_GetUserCount_InvalidRouterID(t *testing.T) {
	h := newNilHotspotHandler()
	router := gin.New()
	router.GET("/routers/:routerId/hotspot/users/count", h.GetUserCount)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/routers/x/hotspot/users/count", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
