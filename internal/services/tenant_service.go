package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"

	"github.com/quiqxiq/roskit/internal/models"
	"github.com/quiqxiq/roskit/internal/repository"
	"gorm.io/gorm"
)

var slugRe = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,98}[a-z0-9])?$`)

// CreateTenantRequest is the data needed to bootstrap a new tenant.
// Settings start empty (defaults applied) — caller can update via UpdateSettings later.
type CreateTenantRequest struct {
	Name        string             `json:"name" binding:"required"`
	Slug        string             `json:"slug" binding:"required"`
	Plan        models.TenantPlan  `json:"plan"`
	HotspotName string             `json:"hotspot_name"`
	DNSName     string             `json:"dns_name"`
	Currency    string             `json:"currency"`
	Phone       string             `json:"phone"`
	Email       string             `json:"email"`
	IdleTimeout int                `json:"idle_timeout"`
	ReportMode  string             `json:"report_mode"`
}

type UpdateTenantSettingsRequest struct {
	HotspotName *string `json:"hotspot_name"`
	DNSName     *string `json:"dns_name"`
	Currency    *string `json:"currency"`
	Phone       *string `json:"phone"`
	Email       *string `json:"email"`
	InfoLP      *string `json:"info_lp"`
	IdleTimeout *int    `json:"idle_timeout"`
	ReportMode  *string `json:"report_mode"`
	Timezone    *string `json:"timezone"`
}

type TenantView struct {
	ID       uint                  `json:"id"`
	Name     string                `json:"name"`
	Slug     string                `json:"slug"`
	Plan     models.TenantPlan     `json:"plan"`
	Status   models.TenantStatus   `json:"status"`
	Settings *models.TenantSettings `json:"settings,omitempty"`
}

type TenantService struct {
	db           *gorm.DB
	tenantRepo   repository.TenantRepository
	settingsRepo repository.TenantSettingsRepository
	templateRepo repository.TemplateRepository
	logger       *slog.Logger
}

func NewTenantService(
	db *gorm.DB,
	tenantRepo repository.TenantRepository,
	settingsRepo repository.TenantSettingsRepository,
	templateRepo repository.TemplateRepository,
) *TenantService {
	return &TenantService{
		db:           db,
		tenantRepo:   tenantRepo,
		settingsRepo: settingsRepo,
		templateRepo: templateRepo,
		logger:       slog.Default().With("component", "tenant-svc"),
	}
}

// Create provisions a tenant with empty settings and copies global default templates.
// Runs in a single DB transaction so partial state is impossible.
func (s *TenantService) Create(ctx context.Context, req CreateTenantRequest) (*models.Tenant, error) {
	slug := strings.ToLower(strings.TrimSpace(req.Slug))
	if !slugRe.MatchString(slug) {
		return nil, fmt.Errorf("invalid slug: must be lowercase alphanumeric with optional dashes")
	}
	if slug == models.PlatformTenantSlug {
		return nil, fmt.Errorf("slug %q is reserved", slug)
	}
	plan := req.Plan
	if plan == "" {
		plan = models.TenantPlanFree
	}
	currency := req.Currency
	if currency == "" {
		currency = "Rp"
	}
	idleTimeout := req.IdleTimeout
	if idleTimeout == 0 {
		idleTimeout = 30
	}
	reportMode := req.ReportMode
	if reportMode == "" {
		reportMode = "disable"
	}

	tenant := &models.Tenant{
		Name:   strings.TrimSpace(req.Name),
		Slug:   slug,
		Plan:   plan,
		Status: models.TenantStatusActive,
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(tenant).Error; err != nil {
			return fmt.Errorf("create tenant: %w", err)
		}

		settings := &models.TenantSettings{
			TenantID:     tenant.ID,
			HotspotName:  req.HotspotName,
			DNSName:      req.DNSName,
			Currency:     currency,
			Phone:        req.Phone,
			Email:        req.Email,
			IdleTimeout:  idleTimeout,
			ReportMode:   reportMode,
			WebhookToken: generateTenantToken(),
		}
		if err := tx.Create(settings).Error; err != nil {
			return fmt.Errorf("create tenant settings: %w", err)
		}

		var globals []models.PrintTemplate
		if err := tx.Where("tenant_id IS NULL").Find(&globals).Error; err != nil {
			return fmt.Errorf("load global templates: %w", err)
		}
		for _, g := range globals {
			tid := tenant.ID
			cp := models.PrintTemplate{
				TenantID: &tid,
				Name:     g.Name,
				Type:     g.Type,
				Part:     g.Part,
				Content:  g.Content,
			}
			if err := tx.Create(&cp).Error; err != nil {
				return fmt.Errorf("copy template %s/%s: %w", g.Type, g.Part, err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.logger.Info("tenant created", "id", tenant.ID, "slug", tenant.Slug)
	return s.tenantRepo.GetByID(ctx, tenant.ID)
}

func (s *TenantService) GetByID(ctx context.Context, id uint) (*models.Tenant, error) {
	return s.tenantRepo.GetByID(ctx, id)
}

func (s *TenantService) GetBySlug(ctx context.Context, slug string) (*models.Tenant, error) {
	return s.tenantRepo.GetBySlug(ctx, slug)
}

func (s *TenantService) List(ctx context.Context) ([]*models.Tenant, error) {
	return s.tenantRepo.List(ctx)
}

func (s *TenantService) UpdateName(ctx context.Context, id uint, name string) error {
	tenant, err := s.tenantRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	tenant.Name = strings.TrimSpace(name)
	return s.tenantRepo.Update(ctx, tenant)
}

func (s *TenantService) UpdateSettings(ctx context.Context, tenantID uint, req UpdateTenantSettingsRequest) (*models.TenantSettings, error) {
	settings, err := s.settingsRepo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if req.HotspotName != nil {
		settings.HotspotName = *req.HotspotName
	}
	if req.DNSName != nil {
		settings.DNSName = *req.DNSName
	}
	if req.Currency != nil && *req.Currency != "" {
		settings.Currency = *req.Currency
	}
	if req.Phone != nil {
		settings.Phone = *req.Phone
	}
	if req.Email != nil {
		settings.Email = *req.Email
	}
	if req.InfoLP != nil {
		settings.InfoLP = *req.InfoLP
	}
	if req.IdleTimeout != nil && *req.IdleTimeout > 0 {
		settings.IdleTimeout = *req.IdleTimeout
	}
	if req.ReportMode != nil && *req.ReportMode != "" {
		settings.ReportMode = *req.ReportMode
	}
	if req.Timezone != nil {
		settings.Timezone = *req.Timezone
	}
	if err := s.settingsRepo.Upsert(ctx, settings); err != nil {
		return nil, err
	}
	return settings, nil
}

func (s *TenantService) GetSettings(ctx context.Context, tenantID uint) (*models.TenantSettings, error) {
	return s.settingsRepo.GetByTenantID(ctx, tenantID)
}

// UploadLogo writes a logo file to disk and updates tenant_settings.logo_path.
func (s *TenantService) UploadLogo(ctx context.Context, tenantID uint, fileData []byte, filename string) (string, error) {
	dir := "uploads/logos"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("mkdir logos: %w", err)
	}
	path := fmt.Sprintf("%s/tenant-%d.png", dir, tenantID)
	if err := os.WriteFile(path, fileData, 0644); err != nil {
		return "", fmt.Errorf("write logo: %w", err)
	}
	if err := s.settingsRepo.UpdateLogo(ctx, tenantID, path); err != nil {
		return "", err
	}
	return path, nil
}

func (s *TenantService) GetLogoPath(ctx context.Context, tenantID uint) (string, error) {
	settings, err := s.settingsRepo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return "", err
	}
	if settings.LogoPath == "" {
		return "", fmt.Errorf("logo not found")
	}
	return settings.LogoPath, nil
}

func (s *TenantService) Suspend(ctx context.Context, id uint) error {
	return s.tenantRepo.UpdateStatus(ctx, id, models.TenantStatusSuspended)
}

func (s *TenantService) Activate(ctx context.Context, id uint) error {
	return s.tenantRepo.UpdateStatus(ctx, id, models.TenantStatusActive)
}

// HardDelete unscope-deletes the tenant, triggering FK CASCADE on all child tables.
func (s *TenantService) HardDelete(ctx context.Context, id uint) error {
	return s.tenantRepo.HardDelete(ctx, id)
}

func generateTenantToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
