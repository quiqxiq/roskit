package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/quiqxiq/roskit/internal/models"
	"github.com/quiqxiq/roskit/internal/repository"
	roskitservice "github.com/quiqxiq/roskit/internal/roskit/adapter/service"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
	appcache "github.com/quiqxiq/roskit/pkg/redis"
)

type EventHandler struct {
	routerRepo   repository.RouterRepository
	saleRepo     repository.SaleRepository
	profileRepo  repository.ProfilePriceMappingRepository
	settingsRepo repository.TenantSettingsRepository
	bridge       *roskitservice.Bridge
	cache        *appcache.Cache
	logger       *slog.Logger
}

func NewEventHandler(
	routerRepo repository.RouterRepository,
	saleRepo repository.SaleRepository,
	profileRepo repository.ProfilePriceMappingRepository,
	settingsRepo repository.TenantSettingsRepository,
	bridge *roskitservice.Bridge,
	cache *appcache.Cache,
) *EventHandler {
	return &EventHandler{
		routerRepo:   routerRepo,
		saleRepo:     saleRepo,
		profileRepo:  profileRepo,
		settingsRepo: settingsRepo,
		bridge:       bridge,
		cache:        cache,
		logger:       slog.Default().With("component", "event-handler"),
	}
}

type OnLoginPayload struct {
	RouterName string
	Server     string
	Username   string
	MAC        string
	IP         string
	Date       string
	Time       string
	Profile    string
}

// OnLoginEvent is a public (no-auth) endpoint hit by RouterOS /tool/fetch.
// Tenant resolution flow:
//   1. RouterOS posts X-Router-Token (or token form field)
//   2. We look up TenantSettings by webhook_token to find tenant_id
//   3. Then resolve router by (tenant_id, router_name)
func (h *EventHandler) OnLoginEvent(c *gin.Context) {
	payload := OnLoginPayload{
		RouterName: c.PostForm("router_name"),
		Server:     c.PostForm("server"),
		Username:   c.PostForm("username"),
		MAC:        c.PostForm("mac"),
		IP:         c.PostForm("ip"),
		Date:       c.PostForm("date"),
		Time:       c.PostForm("time"),
		Profile:    c.PostForm("profile"),
	}

	if payload.Username == "" || payload.RouterName == "" {
		c.Status(http.StatusOK)
		return
	}

	token := c.GetHeader("X-Router-Token")
	if token == "" {
		token = c.PostForm("token")
	}
	if token == "" {
		h.logger.Warn("on-login: missing token", "name", payload.RouterName)
		c.Status(http.StatusOK)
		return
	}

	settings, err := h.settingsRepo.GetByWebhookToken(c.Request.Context(), token)
	if err != nil || settings == nil {
		h.logger.Warn("on-login: invalid token", "name", payload.RouterName)
		c.Status(http.StatusOK)
		return
	}

	tenantID := settings.TenantID
	router, err := h.routerRepo.GetByName(c.Request.Context(), tenantID, payload.RouterName)
	if err != nil {
		h.logger.Warn("on-login: router not found", "name", payload.RouterName, "tenant_id", tenantID, "error", err)
		c.Status(http.StatusOK)
		return
	}

	soldAt, _ := parseMikroTikDateTime(payload.Date, payload.Time, settings.Timezone)
	if soldAt.IsZero() {
		soldAt = time.Now()
	}

	idempotencyKey := mikrotik.MakeSaleIdempotencyKey(router.ID, payload.Username, soldAt)

	exists, err := h.saleRepo.ExistsByIdempotencyKey(c.Request.Context(), idempotencyKey)
	if err != nil {
		h.logger.Error("on-login: idempotency check failed", "error", err)
		c.Status(http.StatusOK)
		return
	}
	if exists {
		c.Status(http.StatusOK)
		return
	}

	if settings.ReportMode == "" || settings.ReportMode == "disable" {
		c.Status(http.StatusOK)
		return
	}

	var price, sellingPrice int64
	var validity string
	if payload.Profile != "" {
		pp, err := h.fetchProfilePrice(c.Request.Context(), router.ID, payload.Profile)
		if err == nil && pp != nil {
			price = pp.Price
			sellingPrice = pp.SellingPrice
			validity = pp.Validity
		}
	}

	rid := router.ID
	sale := &models.VoucherSale{
		TenantID:       tenantID,
		RouterID:       &rid,
		SoldAt:         soldAt,
		Username:       payload.Username,
		ProfileName:    payload.Profile,
		Price:          price,
		SellingPrice:   sellingPrice,
		Server:         payload.Server,
		IPAddress:      payload.IP,
		MACAddress:     payload.MAC,
		Validity:       validity,
		IdempotencyKey: idempotencyKey,
	}

	if err := h.saleRepo.Create(c.Request.Context(), sale); err != nil {
		h.logger.Error("on-login: failed to record sale", "error", err, "username", payload.Username)
	}

	if h.cache != nil {
		_ = h.cache.Invalidate(c.Request.Context(),
			appcache.SalesKey(router.ID, "today"),
			appcache.SalesKey(router.ID, "month"),
			appcache.DashboardKey(router.ID),
		)
	}

	c.Status(http.StatusOK)
}

func (h *EventHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func parseMikroTikDateTime(date, timeStr, tz string) (time.Time, error) {
	return mikrotik.Parse(date, timeStr, mikrotik.ResolveLocation(tz))
}

type profilePrice struct {
	Price        int64
	SellingPrice int64
	Validity     string
}

func (h *EventHandler) fetchProfilePrice(ctx context.Context, routerID uint, profileName string) (*profilePrice, error) {
	mapping, err := h.profileRepo.FindByRouterAndProfile(ctx, routerID, profileName)
	if err == nil && mapping != nil {
		return &profilePrice{
			Price:        mapping.Price,
			SellingPrice: mapping.SellingPrice,
			Validity:     mapping.Validity,
		}, nil
	}

	rID := fmt.Sprintf("%d", routerID)
	profiles, err := h.bridge.Query(ctx, rID, "ip/hotspot/user/profile/print", "?name="+profileName)
	if err != nil || len(profiles) == 0 {
		return &profilePrice{Price: 0}, nil
	}
	onLogin := profiles[0]["on-login"]
	if onLogin == "" {
		return &profilePrice{Price: 0}, nil
	}
	meta := roskitservice.ParseOnLoginPut(onLogin)
	if meta == nil {
		return &profilePrice{Price: 0}, nil
	}
	price, _ := strconv.ParseInt(meta.Price, 10, 64)
	sprice, _ := strconv.ParseInt(meta.SellingPrice, 10, 64)
	return &profilePrice{Price: price, SellingPrice: sprice, Validity: meta.Validity}, nil
}
