package services

import (
	"context"
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
	ID          uint       `json:"id"`
	TenantID    uint       `json:"tenant_id"`
	Name        string     `json:"name"`
	IPAddress   string     `json:"ip_address"`
	APIPort     int        `json:"api_port"`
	APIUsername string     `json:"api_username"`
	Status      string     `json:"status"`
	LastSeenAt  *time.Time `json:"last_seen_at"`
	Notes       *string    `json:"notes"`
}

func toPublicView(r *models.Router) RouterPublicView {
	return RouterPublicView{
		ID:          r.ID,
		TenantID:    r.TenantID,
		Name:        r.Name,
		IPAddress:   r.IPAddress,
		APIPort:     r.APIPort,
		APIUsername: r.APIUsername,
		Status:      r.Status.String(),
		LastSeenAt:  r.LastSeenAt,
		Notes:       r.Notes,
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
	Name        string `json:"name" binding:"required,min=1,max=100"`
	IPAddress   string `json:"ip_address" binding:"required,max=255"` // accepts IP, host, or host:port — service layer normalises
	APIPort     int    `json:"api_port" binding:"omitempty,min=1,max=65535"`
	APIUsername string `json:"api_username" binding:"required,min=1,max=64"`
	Password    string `json:"password" binding:"required,min=1,max=128"`
	Notes       string `json:"notes" binding:"max=500"`
}

type UpdateRouterRequest struct {
	Name        string `json:"name" binding:"omitempty,min=1,max=100"`
	IPAddress   string `json:"ip_address" binding:"omitempty,max=255"`
	APIPort     int    `json:"api_port" binding:"omitempty,min=1,max=65535"`
	APIUsername string `json:"api_username" binding:"omitempty,min=1,max=64"`
	Password    string `json:"password" binding:"omitempty,min=1,max=128"`
	Notes       string `json:"notes" binding:"max=500"`
}

type MigrationResult struct {
	Total    int `json:"total"`
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
	Errors   int `json:"errors"`
}

// RouterRepository is the local interface used by RouterService.
type RouterRepository interface {
	Create(ctx context.Context, router *models.Router) error
	GetByID(ctx context.Context, tenantID, id uint) (*models.Router, error)
	GetByIDAny(ctx context.Context, id uint) (*models.Router, error)
	GetByName(ctx context.Context, tenantID uint, name string) (*models.Router, error)
	List(ctx context.Context, tenantID uint) ([]*models.Router, error)
	ListAll(ctx context.Context) ([]*models.Router, error)
	Update(ctx context.Context, router *models.Router) error
	Delete(ctx context.Context, tenantID, id uint) error
	UpdateStatus(ctx context.Context, routerID uint, status models.RouterStatus) error
	UpdateLastSeen(ctx context.Context, routerID uint, t time.Time) error
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


func (s *RouterService) WatchAndSyncStatus(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	prev := make(map[string]string)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			statuses := s.engine.Status()
			for routerIDStr, stateStr := range statuses {
				last, seen := prev[routerIDStr]
				if seen && last == stateStr {
					continue
				}
				prev[routerIDStr] = stateStr

				var routerID uint
				if _, err := fmt.Sscanf(routerIDStr, "%d", &routerID); err != nil {
					continue
				}

				status := models.RouterStatus(stateStr)
				if err := s.repo.UpdateStatus(ctx, routerID, status); err != nil {
					s.logger.Warn("failed to update router status in db",
						"router_id", routerIDStr, "status", stateStr, "error", err)
					continue
				}

				if status == models.RouterStatusConnected {
					now := time.Now()
					if err := s.repo.UpdateLastSeen(ctx, routerID, now); err != nil {
						s.logger.Warn("failed to update router last_seen_at",
							"router_id", routerIDStr, "error", err)
					}
				}
				s.logger.Info("router status updated", "router_id", routerIDStr, "status", stateStr)
			}
		}
	}
}

// SeedEngineFromDB registers all routers across all tenants in the engine at startup.
// This is platform-level — the engine is shared, indexed by router ID.
func (s *RouterService) SeedEngineFromDB(ctx context.Context) {
	routers, err := s.repo.ListAll(ctx)
	if err != nil {
		s.logger.Error("failed to load routers for engine seeding", "error", err)
		return
	}

	registered := 0
	for _, r := range routers {
		password, err := encrypt.Decrypt(r.APIPasswordEncrypted, s.aesKey)
		if err != nil {
			s.logger.Error("failed to decrypt router password, skipping",
				"id", r.ID, "name", r.Name, "error", err)
			continue
		}
		port := r.APIPort
		if port == 0 {
			port = 8728
		}
		if err := s.engine.AddRouter(ctx, execution.ConnConfig{
			RouterID: fmt.Sprintf("%d", r.ID),
			Address:  fmt.Sprintf("%s:%d", r.IPAddress, port),
			Username: r.APIUsername,
			Password: password,
		}); err != nil {
			s.logger.Error("failed to register router in engine",
				"id", r.ID, "name", r.Name, "error", err)
			continue
		}
		registered++
	}

	if registered > 0 {
		s.logger.Info("engine seeded from database", "count", registered, "total", len(routers))
	}
}

func (s *RouterService) CreateRouter(ctx context.Context, tenantID uint, req CreateRouterRequest) (*RouterPublicView, error) {
	if req.IPAddress == "" || req.APIUsername == "" || req.Password == "" {
		return nil, fmt.Errorf("ip_address, api_username, and password are required")
	}

	port := req.APIPort
	if port == 0 {
		port = 8728
	}

	testCtx, testCancel := context.WithTimeout(ctx, 5*time.Second)
	defer testCancel()
	result, err := s.TestConnection(testCtx, req.IPAddress, port, req.APIUsername, req.Password)
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

	var notes *string
	if req.Notes != "" {
		notes = &req.Notes
	}

	router := &models.Router{
		TenantID:             tenantID,
		Name:                 req.Name,
		IPAddress:            req.IPAddress,
		APIPort:              port,
		APIUsername:          req.APIUsername,
		APIPasswordEncrypted: encPass,
		Notes:                notes,
		Status:               models.RouterStatusUnknown,
	}

	if err := s.repo.Create(ctx, router); err != nil {
		return nil, fmt.Errorf("save router: %w", err)
	}

	routerID := fmt.Sprintf("%d", router.ID)
	_ = s.engine.AddRouter(ctx, execution.ConnConfig{
		RouterID: routerID,
		Address:  fmt.Sprintf("%s:%d", req.IPAddress, port),
		Username: req.APIUsername,
		Password: req.Password,
	})

	s.logger.Info("router created and registered", "id", router.ID, "name", req.Name, "tenant_id", tenantID)

	view := toPublicView(router)
	return &view, nil
}

func (s *RouterService) GetRouter(ctx context.Context, tenantID, id uint) (*RouterPublicView, error) {
	router, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	view := toPublicView(router)
	return &view, nil
}

func (s *RouterService) ListRouters(ctx context.Context, tenantID uint) ([]RouterPublicView, error) {
	routers, err := s.repo.List(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	views := make([]RouterPublicView, len(routers))
	for i, r := range routers {
		views[i] = toPublicView(r)
	}
	return views, nil
}

func (s *RouterService) UpdateRouter(ctx context.Context, tenantID, id uint, req UpdateRouterRequest) (*RouterPublicView, error) {
	router, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	ip := router.IPAddress
	username := router.APIUsername
	port := router.APIPort
	password := router.APIPasswordEncrypted
	passwordPlain := ""
	credsChanged := false

	if req.IPAddress != "" && req.IPAddress != router.IPAddress {
		ip = req.IPAddress
		credsChanged = true
	}
	if req.APIUsername != "" && req.APIUsername != router.APIUsername {
		username = req.APIUsername
		credsChanged = true
	}
	if req.APIPort != 0 && req.APIPort != router.APIPort {
		port = req.APIPort
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
			dec, err := encrypt.Decrypt(router.APIPasswordEncrypted, s.aesKey)
			if err != nil {
				return nil, fmt.Errorf("decrypt existing password: %w", err)
			}
			testPass = dec
		}
		testPort := port
		if testPort == 0 {
			testPort = 8728
		}
		testResult, err := s.TestConnection(ctx, ip, testPort, username, testPass)
		if err != nil {
			return nil, fmt.Errorf("connection test with new credentials failed: %w", err)
		}
		if !testResult.Connected {
			return nil, fmt.Errorf("connection test failed: %s", testResult.Error)
		}
	}

	if req.Name != "" {
		router.Name = req.Name
	}
	router.IPAddress = ip
	router.APIUsername = username
	router.APIPort = port
	router.APIPasswordEncrypted = password
	if req.Notes != "" {
		router.Notes = &req.Notes
	}

	if err := s.repo.Update(ctx, router); err != nil {
		return nil, fmt.Errorf("update router: %w", err)
	}

	if credsChanged {
		routerID := fmt.Sprintf("%d", id)
		plainPass := passwordPlain
		if plainPass == "" {
			dec, err := encrypt.Decrypt(router.APIPasswordEncrypted, s.aesKey)
			if err == nil {
				plainPass = dec
			}
		}
		s.engine.RemoveRouter(routerID)
		_ = s.engine.AddRouter(ctx, execution.ConnConfig{
			RouterID: routerID,
			Address:  fmt.Sprintf("%s:%d", ip, port),
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

func (s *RouterService) DeleteRouter(ctx context.Context, tenantID, id uint) error {
	router, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	routerID := fmt.Sprintf("%d", id)
	s.engine.RemoveRouter(routerID)

	if err := s.repo.Delete(ctx, tenantID, id); err != nil {
		return fmt.Errorf("delete router: %w", err)
	}

	if s.cache != nil {
		_ = s.cache.Delete(ctx, appcache.DashboardKey(id))
	}

	s.logger.Info("router deleted", "id", id, "name", router.Name, "tenant_id", tenantID)
	return nil
}

func (s *RouterService) TestConnection(ctx context.Context, ip string, port int, username, password string) (*ConnectionTestResult, error) {
	if port == 0 {
		port = 8728
	}
	start := time.Now()

	addr := fmt.Sprintf("%s:%d", ip, port)
	client, err := routeros.DialContext(ctx, addr, username, password)
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

func (s *RouterService) MigrateFromConfigPHP(ctx context.Context, tenantID uint, filePath string) (*MigrationResult, error) {
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
		name := extractField(line)

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
			s.logger.Warn("failed to decrypt password", "name", name, "error", err)
			continue
		}

		req := CreateRouterRequest{
			Name:        name,
			IPAddress:   ip,
			APIUsername: username,
			Password:    password,
		}

		_, err = s.CreateRouter(ctx, tenantID, req)
		if err != nil {
			if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
				result.Skipped++
				s.logger.Warn("router already exists, skipping", "name", name)
			} else {
				result.Errors++
				s.logger.Error("failed to import router", "name", name, "error", err)
			}
			continue
		}

		result.Imported++
	}

	s.logger.Info("migration complete",
		"tenant_id", tenantID,
		"total", result.Total,
		"imported", result.Imported,
		"skipped", result.Skipped,
		"errors", result.Errors,
	)

	return result, nil
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
