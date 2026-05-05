package repository

import (
	"context"

	"github.com/quiqxiq/roskit/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TenantSettingsRepository interface {
	GetByTenantID(ctx context.Context, tenantID uint) (*models.TenantSettings, error)
	GetByWebhookToken(ctx context.Context, token string) (*models.TenantSettings, error)
	Upsert(ctx context.Context, settings *models.TenantSettings) error
	UpdateLogo(ctx context.Context, tenantID uint, logoPath string) error
	UpdateTimezone(ctx context.Context, tenantID uint, tz string) error
}

type TenantSettingsRepo struct {
	db *gorm.DB
}

func NewTenantSettingsRepo(db *gorm.DB) *TenantSettingsRepo {
	return &TenantSettingsRepo{db: db}
}

func (r *TenantSettingsRepo) GetByTenantID(ctx context.Context, tenantID uint) (*models.TenantSettings, error) {
	var s models.TenantSettings
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *TenantSettingsRepo) GetByWebhookToken(ctx context.Context, token string) (*models.TenantSettings, error) {
	if token == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var s models.TenantSettings
	if err := r.db.WithContext(ctx).
		Where("webhook_token = ?", token).
		First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *TenantSettingsRepo) Upsert(ctx context.Context, settings *models.TenantSettings) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tenant_id"}},
			UpdateAll: true,
		}).
		Create(settings).Error
}

func (r *TenantSettingsRepo) UpdateLogo(ctx context.Context, tenantID uint, logoPath string) error {
	return r.db.WithContext(ctx).
		Model(&models.TenantSettings{}).
		Where("tenant_id = ?", tenantID).
		Update("logo_path", logoPath).Error
}

func (r *TenantSettingsRepo) UpdateTimezone(ctx context.Context, tenantID uint, tz string) error {
	return r.db.WithContext(ctx).
		Model(&models.TenantSettings{}).
		Where("tenant_id = ?", tenantID).
		Update("timezone", tz).Error
}
