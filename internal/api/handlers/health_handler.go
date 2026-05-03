package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	roskitservice "github.com/quiqxiq/roskit/internal/roskit/adapter/service"
	appcache "github.com/quiqxiq/roskit/pkg/redis"
)

type HealthHandler struct {
	db     *gorm.DB
	cache  *appcache.Cache
	bridge *roskitservice.Bridge
}

func NewHealthHandler(db *gorm.DB, cache *appcache.Cache, bridge *roskitservice.Bridge) *HealthHandler {
	return &HealthHandler{db: db, cache: cache, bridge: bridge}
}

type healthStatus struct {
	Status   string         `json:"status"`
	Services serviceStatuses `json:"services"`
}

type serviceStatuses struct {
	Postgres string       `json:"postgres"`
	Redis    string       `json:"redis"`
	Engine   engineStatus `json:"engine"`
}

type engineStatus struct {
	Total     int `json:"routers_total"`
	Connected int `json:"routers_connected"`
}

func (h *HealthHandler) Check(c *gin.Context) {
	overall := "ok"

	pgStatus := "ok"
	sqlDB, err := h.db.DB()
	if err != nil || pingDB(sqlDB) != nil {
		pgStatus = "unavailable"
		overall = "degraded"
	}

	redisStatus := "ok"
	if h.cache != nil {
		if err := h.cache.Ping(c.Request.Context()); err != nil {
			redisStatus = "unavailable"
			overall = "degraded"
		}
	}

	total := 0
	connected := 0
	if h.bridge != nil {
		status := h.bridge.PoolStatus()
		total = len(status)
		for _, up := range status {
			if up {
				connected++
			}
		}
	}

	code := http.StatusOK
	if overall != "ok" {
		code = http.StatusServiceUnavailable
	}

	c.JSON(code, gin.H{
		"data": healthStatus{
			Status: overall,
			Services: serviceStatuses{
				Postgres: pgStatus,
				Redis:    redisStatus,
				Engine:   engineStatus{Total: total, Connected: connected},
			},
		},
		"error": nil,
	})
}

func pingDB(db *sql.DB) error {
	if db == nil {
		return sql.ErrNoRows
	}
	return db.Ping()
}
