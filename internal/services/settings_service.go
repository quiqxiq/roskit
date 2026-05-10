package services

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/quiqxiq/roskit/internal/models"
	"github.com/quiqxiq/roskit/internal/repository"
)

type UpdateSettingsRequest struct {
	HotspotName string `json:"hotspot_name"`
	DNSName     string `json:"dns_name"`
	Currency    string `json:"currency"    binding:"omitempty,max=10"`
	Phone       string `json:"phone"       binding:"omitempty,max=20"`
	Email       string `json:"email"       binding:"omitempty,email"`
	InfoLP      string `json:"info_lp"`
	IdleTimeout int    `json:"idle_timeout" binding:"omitempty,min=0"`
	ReportMode  string `json:"report_mode"  binding:"omitempty,oneof=disable enable"`
	Timezone    string `json:"timezone"    binding:"omitempty,max=50"`
}

type SettingsService struct {
	repo   repository.SettingsRepository
	logger *slog.Logger
}

func NewSettingsService(repo repository.SettingsRepository) *SettingsService {
	return &SettingsService{
		repo:   repo,
		logger: slog.Default().With("component", "settings-svc"),
	}
}

func (s *SettingsService) Get(ctx context.Context) (*models.Settings, error) {
	settings, err := s.repo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("get settings: %w", err)
	}
	return settings, nil
}

func (s *SettingsService) Update(ctx context.Context, req UpdateSettingsRequest) (*models.Settings, error) {
	settings, err := s.Get(ctx)
	if err != nil {
		return nil, err
	}

	if req.HotspotName != "" {
		settings.HotspotName = req.HotspotName
	}
	if req.DNSName != "" {
		settings.DNSName = req.DNSName
	}
	if req.Currency != "" {
		settings.Currency = req.Currency
	}
	if req.Phone != "" {
		settings.Phone = req.Phone
	}
	if req.Email != "" {
		settings.Email = req.Email
	}
	if req.InfoLP != "" {
		settings.InfoLP = req.InfoLP
	}
	if req.IdleTimeout > 0 {
		settings.IdleTimeout = req.IdleTimeout
	}
	if req.ReportMode != "" {
		settings.ReportMode = req.ReportMode
	}
	if req.Timezone != "" {
		settings.Timezone = req.Timezone
	}

	if err := s.repo.Upsert(ctx, settings); err != nil {
		return nil, fmt.Errorf("save settings: %w", err)
	}
	return settings, nil
}

func (s *SettingsService) UploadLogo(ctx context.Context, data []byte, filename string) (string, error) {
	dir := "uploads/logos"
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		ext = ".png"
	}
	logoPath := fmt.Sprintf("%s/logo%s", dir, ext)

	if err := writeFile(logoPath, data); err != nil {
		return "", fmt.Errorf("save logo: %w", err)
	}

	if err := s.repo.UpdateLogo(ctx, logoPath); err != nil {
		return "", fmt.Errorf("update logo path: %w", err)
	}

	s.logger.Info("logo uploaded", "path", logoPath)
	return logoPath, nil
}

func (s *SettingsService) GetLogoPath(ctx context.Context) (string, error) {
	settings, err := s.repo.Get(ctx)
	if err != nil {
		return "", fmt.Errorf("settings not found")
	}
	if settings.LogoPath == "" {
		return "", fmt.Errorf("logo not set")
	}
	return settings.LogoPath, nil
}
