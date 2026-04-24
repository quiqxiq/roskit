package handlers

import (
	"crypto/sha256"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/quiqxiq/roskit/internal/models"
	"github.com/quiqxiq/roskit/internal/repository"
	appcache "github.com/quiqxiq/roskit/pkg/redis"
)

type EventHandler struct {
	routerRepo repository.RouterRepository
	saleRepo   repository.SaleRepository
	cache      *appcache.Cache
	logger     *slog.Logger
}

func NewEventHandler(routerRepo repository.RouterRepository, saleRepo repository.SaleRepository, cache *appcache.Cache) *EventHandler {
	return &EventHandler{
		routerRepo: routerRepo,
		saleRepo:   saleRepo,
		cache:      cache,
		logger:     slog.Default().With("component", "event-handler"),
	}
}

type OnLoginPayload struct {
	RouterSession string
	Server        string
	Username      string
	MAC           string
	IP            string
	Date          string
	Time          string
	Profile       string
}

func (h *EventHandler) OnLoginEvent(c *gin.Context) {
	payload := OnLoginPayload{
		RouterSession: c.PostForm("router_session"),
		Server:        c.PostForm("server"),
		Username:      c.PostForm("username"),
		MAC:           c.PostForm("mac"),
		IP:            c.PostForm("ip"),
		Date:          c.PostForm("date"),
		Time:          c.PostForm("time"),
		Profile:       c.PostForm("profile"),
	}

	if payload.Username == "" || payload.RouterSession == "" {
		c.Status(http.StatusOK)
		return
	}

	router, err := h.routerRepo.GetBySessionName(c.Request.Context(), payload.RouterSession)
	if err != nil {
		h.logger.Warn("on-login: router session not found", "session", payload.RouterSession, "error", err)
		c.Status(http.StatusOK)
		return
	}

	if router.Token != "" {
		token := c.GetHeader("X-Router-Token")
		if token == "" {
			token = c.PostForm("token")
		}
		if token != router.Token {
			h.logger.Warn("on-login: invalid token", "session", payload.RouterSession)
			c.Status(http.StatusOK)
			return
		}
	}

	idKey := fmt.Sprintf("%d|%s|%s|%s", router.ID, payload.Username, payload.Date, payload.Time)
	hash := sha256.Sum256([]byte(idKey))
	idempotencyKey := fmt.Sprintf("%x", hash)

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

	soldAt, _ := parseMikroTikDateTime(payload.Date, payload.Time)
	if soldAt.IsZero() {
		soldAt = time.Now()
	}

	price := int64(0)
	if payload.Profile != "" {
		profiles, err := h.fetchProfilePrice(c.Request.Context(), router.ID, payload.Profile)
		if err == nil && profiles != nil {
			price = profiles.Price
		}
	}

	sale := &models.VoucherSale{
		RouterID:       router.ID,
		SoldAt:         soldAt,
		Username:       payload.Username,
		ProfileName:    payload.Profile,
		Price:          price,
		Server:         payload.Server,
		IPAddress:      payload.IP,
		MACAddress:     payload.MAC,
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

func parseMikroTikDateTime(date, timeStr string) (time.Time, error) {
	if date == "" || timeStr == "" {
		return time.Time{}, fmt.Errorf("empty date or time")
	}

	parts := strings.SplitN(timeStr, ":", 3)
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("invalid time format: %s", timeStr)
	}
	hour, _ := strconv.Atoi(parts[0])
	minute, _ := strconv.Atoi(parts[1])
	second, _ := strconv.Atoi(parts[2])

	dateParts := strings.SplitN(date, "/", 3)
	if len(dateParts) != 3 {
		return time.Time{}, fmt.Errorf("invalid date format: %s", date)
	}

	months := map[string]time.Month{
		"jan": time.January, "feb": time.February, "mar": time.March,
		"apr": time.April, "may": time.May, "jun": time.June,
		"jul": time.July, "aug": time.August, "sep": time.September,
		"oct": time.October, "nov": time.November, "dec": time.December,
	}
	mon := months[strings.ToLower(dateParts[0])]
	day, _ := strconv.Atoi(dateParts[1])
	year, _ := strconv.Atoi(dateParts[2])

	return time.Date(year, mon, day, hour, minute, second, 0, time.Local), nil
}

type profilePrice struct {
	Price        int64
	SellingPrice int64
	Validity     string
}

func (h *EventHandler) fetchProfilePrice(ctx interface{ Value(any) any }, routerID uint, profileName string) (*profilePrice, error) {
	return &profilePrice{Price: 0}, nil
}
