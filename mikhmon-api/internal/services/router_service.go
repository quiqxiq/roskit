package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	routeros "github.com/go-routeros/routeros/v3"

	"github.com/quiqxiq/roskit/internal/models"
	"github.com/quiqxiq/roskit/internal/roskit/execution"
	"github.com/quiqxiq/roskit/internal/roskit/orchestrator"
	"github.com/quiqxiq/roskit/pkg/encrypt"
	appcache "github.com/quiqxiq/roskit/pkg/redis"
)

type RouterPublicView struct {
	ID          uint   `json:"id"`
	SessionName string `json:"session_name"`
	IP          string `json:"ip"`
	Username    string `json:"username"`
	HotspotName string `json:"hotspot_name"`
	DNSName     string `json:"dns_name"`
	Currency    string `json:"currency"`
	Phone       string `json:"phone"`
	Email       string `json:"email"`
	InfoLP      string `json:"info_lp"`
	IdleTimeout string `json:"idle_timeout"`
	ReportMode  string `json:"report_mode"`
}

func toPublicView(r *models.Router) RouterPublicView {
	return RouterPublicView{
		ID:          r.ID,
		SessionName: r.SessionName,
		IP:          r.IP,
		Username:    r.Username,
		HotspotName: r.HotspotName,
		DNSName:     r.DNSName,
		Currency:    r.Currency,
		Phone:       r.Phone,
		Email:       r.Email,
		InfoLP:      r.InfoLP,
		IdleTimeout: r.IdleTimeout,
		ReportMode:  r.ReportMode,
	}
}

type ConnectionTestResult struct {
	Connected bool   `json:"connected"`
	Latency   int64  `json:"latency_ms"`
	Version   string `json:"routeros_version"`
	Board     string `json:"board_model"`
	Identity  string `json:"identity"`
	Error     string `json:"error,omitempty"`
}

type CreateRouterRequest struct {
	SessionName string `json:"session_name" binding:"required"`
	IP          string `json:"ip" binding:"required"`
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required"`
	HotspotName string `json:"hotspot_name"`
	DNSName     string `json:"dns_name"`
	Currency    string `json:"currency"`
	Phone       string `json:"phone"`
	Email       string `json:"email"`
	InfoLP      string `json:"info_lp"`
	IdleTimeout string `json:"idle_timeout"`
	ReportMode  string `json:"report_mode"`
}

type UpdateRouterRequest struct {
	SessionName string `json:"session_name"`
	IP          string `json:"ip"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	HotspotName string `json:"hotspot_name"`
	DNSName     string `json:"dns_name"`
	Currency    string `json:"currency"`
	Phone       string `json:"phone"`
	Email       string `json:"email"`
	InfoLP      string `json:"info_lp"`
	IdleTimeout string `json:"idle_timeout"`
	ReportMode  string `json:"report_mode"`
}

type MigrationResult struct {
	Total    int `json:"total"`
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
	Errors   int `json:"errors"`
}

type RouterService struct {
	repo   RouterRepository
	engine *orchestrator.Engine
	cache  *appcache.Cache
	logger *slog.Logger
	aesKey string
}

func NewRouterService(repo RouterRepository, engine *orchestrator.Engine, cache *appcache.Cache, aesKey string) *RouterService {
	return &RouterService{
		repo:   repo,
		engine: engine,
		cache:  cache,
		logger: slog.Default().With("component", "router-svc"),
		aesKey: aesKey,
	}
}

func (s *RouterService) SeedEngineFromDB(ctx context.Context) {
	routers, err := s.repo.List(ctx)
	if err != nil {
		s.logger.Error("failed to load routers for engine seeding", "error", err)
		return
	}

	registered := 0
	for _, r := range routers {
		password, err := encrypt.Decrypt(r.PasswordEnc, s.aesKey)
		if err != nil {
			s.logger.Error("failed to decrypt router password, skipping",
				"id", r.ID, "session", r.SessionName, "error", err)
			continue
		}
		if err := s.engine.AddRouter(ctx, execution.ConnConfig{
			RouterID: fmt.Sprintf("%d", r.ID),
			Address:  r.IP + ":8728",
			Username: r.Username,
			Password: password,
		}); err != nil {
			s.logger.Error("failed to register router in engine",
				"id", r.ID, "session", r.SessionName, "error", err)
			continue
		}
		registered++
	}

	if registered > 0 {
		s.logger.Info("engine seeded from database", "count", registered, "total", len(routers))
	}
}

func (s *RouterService) CreateRouter(ctx context.Context, req CreateRouterRequest) (*RouterPublicView, error) {
	if req.IP == "" || req.Username == "" || req.Password == "" {
		return nil, fmt.Errorf("ip, username, and password are required")
	}

	result, err := s.TestConnection(ctx, req.IP, req.Username, req.Password)
	if err != nil {
		return nil, fmt.Errorf("connection test failed: %w", err)
	}
	if !result.Connected {
		return nil, fmt.Errorf("connection test failed: %s", result.Error)
	}

	encPass, err := encrypt.Encrypt(req.Password, s.aesKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt password: %w", err)
	}

	router := &models.Router{
		SessionName: req.SessionName,
		IP:          req.IP,
		Username:    req.Username,
		PasswordEnc: encPass,
		HotspotName: req.HotspotName,
		DNSName:     req.DNSName,
		Currency:    req.Currency,
		Phone:       req.Phone,
		Email:       req.Email,
		InfoLP:      req.InfoLP,
		IdleTimeout: req.IdleTimeout,
		ReportMode:  req.ReportMode,
		Token:       generateToken(),
	}

	if err := s.repo.Create(ctx, router); err != nil {
		return nil, fmt.Errorf("save router: %w", err)
	}

	routerID := fmt.Sprintf("%d", router.ID)
	_ = s.engine.AddRouter(ctx, execution.ConnConfig{
		RouterID: routerID,
		Address:  req.IP + ":8728",
		Username: req.Username,
		Password: req.Password,
	})

	s.logger.Info("router created and registered", "id", router.ID, "session", req.SessionName)

	view := toPublicView(router)
	return &view, nil
}

func (s *RouterService) GetRouter(ctx context.Context, id uint) (*RouterPublicView, error) {
	router, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	view := toPublicView(router)
	return &view, nil
}

func (s *RouterService) ListRouters(ctx context.Context) ([]RouterPublicView, error) {
	routers, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	views := make([]RouterPublicView, len(routers))
	for i, r := range routers {
		views[i] = toPublicView(r)
	}
	return views, nil
}

func (s *RouterService) UpdateRouter(ctx context.Context, id uint, req UpdateRouterRequest) (*RouterPublicView, error) {
	router, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	ip := router.IP
	username := router.Username
	password := router.PasswordEnc
	passwordPlain := ""
	credsChanged := false

	if req.IP != "" && req.IP != router.IP {
		ip = req.IP
		credsChanged = true
	}
	if req.Username != "" && req.Username != router.Username {
		username = req.Username
		credsChanged = true
	}
	if req.Password != "" {
		encPass, err := encrypt.Encrypt(req.Password, s.aesKey)
		if err != nil {
			return nil, fmt.Errorf("encrypt password: %w", err)
		}
		password = encPass
		passwordPlain = req.Password
		credsChanged = true
	}

	if credsChanged {
		testPass := passwordPlain
		if testPass == "" {
			dec, err := encrypt.Decrypt(router.PasswordEnc, s.aesKey)
			if err != nil {
				return nil, fmt.Errorf("decrypt existing password: %w", err)
			}
			testPass = dec
		}
		testResult, err := s.TestConnection(ctx, ip, username, testPass)
		if err != nil {
			return nil, fmt.Errorf("connection test with new credentials failed: %w", err)
		}
		if !testResult.Connected {
			return nil, fmt.Errorf("connection test failed: %s", testResult.Error)
		}
	}

	if req.SessionName != "" {
		router.SessionName = req.SessionName
	}
	router.IP = ip
	router.Username = username
	router.PasswordEnc = password
	if req.HotspotName != "" {
		router.HotspotName = req.HotspotName
	}
	if req.DNSName != "" {
		router.DNSName = req.DNSName
	}
	if req.Currency != "" {
		router.Currency = req.Currency
	}
	if req.Phone != "" {
		router.Phone = req.Phone
	}
	if req.Email != "" {
		router.Email = req.Email
	}
	if req.InfoLP != "" {
		router.InfoLP = req.InfoLP
	}
	if req.IdleTimeout != "" {
		router.IdleTimeout = req.IdleTimeout
	}
	if req.ReportMode != "" {
		router.ReportMode = req.ReportMode
	}

	if err := s.repo.Update(ctx, router); err != nil {
		return nil, fmt.Errorf("update router: %w", err)
	}

	if credsChanged {
		routerID := fmt.Sprintf("%d", id)
		plainPass := passwordPlain
		if plainPass == "" {
			dec, err := encrypt.Decrypt(router.PasswordEnc, s.aesKey)
			if err == nil {
				plainPass = dec
			}
		}
		s.engine.RemoveRouter(routerID)
		_ = s.engine.AddRouter(ctx, execution.ConnConfig{
			RouterID: routerID,
			Address:  ip + ":8728",
			Username: username,
			Password: plainPass,
		})
	}

	if s.cache != nil {
		_ = s.cache.Delete(ctx, appcache.DashboardKey(id))
	}

	view := toPublicView(router)
	return &view, nil
}

func (s *RouterService) DeleteRouter(ctx context.Context, id uint) error {
	router, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	routerID := fmt.Sprintf("%d", id)
	s.engine.RemoveRouter(routerID)

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete router: %w", err)
	}

	if s.cache != nil {
		_ = s.cache.Delete(ctx, appcache.DashboardKey(id))
	}

	s.logger.Info("router deleted", "id", id, "session", router.SessionName)
	return nil
}

func (s *RouterService) TestConnection(ctx context.Context, ip, username, password string) (*ConnectionTestResult, error) {
	start := time.Now()

	client, err := routeros.DialContext(ctx, ip+":8728", username, password)
	if err != nil {
		return &ConnectionTestResult{
			Connected: false,
			Error:     fmt.Sprintf("dial failed: %s", err),
		}, nil
	}
	defer client.Close()

	latency := time.Since(start).Milliseconds()

	reply, err := client.RunContext(ctx, "/system/identity/print")
	if err != nil {
		return &ConnectionTestResult{
			Connected: true,
			Latency:   latency,
			Error:     fmt.Sprintf("identity print failed: %s", err),
		}, nil
	}

	identity := ""
	board := ""
	version := ""

	for _, re := range reply.Re {
		for k, v := range re.Map {
			if k == "name" {
				identity = v
			}
			if k == "board-name" {
				board = v
			}
		}
	}

	resReply, err := client.RunContext(ctx, "/system/resource/print")
	if err == nil {
		for _, re := range resReply.Re {
			if v, ok := re.Map["version"]; ok {
				version = v
			}
		}
	}

	return &ConnectionTestResult{
		Connected: true,
		Latency:   latency,
		Version:   version,
		Board:     board,
		Identity:  identity,
	}, nil
}

func (s *RouterService) MigrateFromConfigPHP(ctx context.Context, filePath string) (*MigrationResult, error) {
	data, err := readFileContent(filePath)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	result := &MigrationResult{}
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "$data[") {
			continue
		}

		result.Total++

		ip := extractDelimited(line, "!", "")
		username := extractDelimited(line, "@|@", "")
		encPass := extractDelimited(line, "#|#", "")
		sessionName := extractField(line)

		if ip == "" || username == "" || encPass == "" {
			result.Errors++
			continue
		}

		password := ""
		if dec, err := encrypt.DecodeBlah(encPass); err == nil {
			password = dec
		} else if dec, err := encrypt.DecryptLegacyPHPConfig(encPass); err == nil {
			password = dec
		} else {
			result.Errors++
			s.logger.Warn("failed to decrypt password", "session", sessionName, "error", err)
			continue
		}

		req := CreateRouterRequest{
			SessionName: sessionName,
			IP:          ip,
			Username:    username,
			Password:    password,
		}

		_, err = s.CreateRouter(ctx, req)
		if err != nil {
			if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
				result.Skipped++
				s.logger.Warn("router already exists, skipping", "session", sessionName)
			} else {
				result.Errors++
				s.logger.Error("failed to import router", "session", sessionName, "error", err)
			}
			continue
		}

		result.Imported++
	}

	s.logger.Info("migration complete",
		"total", result.Total,
		"imported", result.Imported,
		"skipped", result.Skipped,
		"errors", result.Errors,
	)

	return result, nil
}

type RouterRepository interface {
	Create(ctx context.Context, router *models.Router) error
	GetByID(ctx context.Context, id uint) (*models.Router, error)
	GetBySessionName(ctx context.Context, sessionName string) (*models.Router, error)
	List(ctx context.Context) ([]*models.Router, error)
	Update(ctx context.Context, router *models.Router) error
	Delete(ctx context.Context, id uint) error
}

func generateToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func extractDelimited(s, delim, fallback string) string {
	idx := strings.Index(s, delim)
	if idx < 0 {
		return fallback
	}
	after := s[idx+len(delim):]
	end := strings.Index(after, delim)
	if end < 0 {
		return after
	}
	return after[:end]
}

func extractField(line string) string {
	start := strings.Index(line, "'")
	if start < 0 {
		return ""
	}
	start++
	end := strings.Index(line[start:], "'")
	if end < 0 {
		return line[start:]
	}
	return line[start : start+end]
}

func readFileContent(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}
