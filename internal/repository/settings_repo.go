package repository

import (
	"context"

	"github.com/quiqxiq/roskit/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SettingsRepository interface {
	Get(ctx context.Context) (*models.Settings, error)
	GetByWebhookToken(ctx context.Context, token string) (*models.Settings, error)
	Upsert(ctx context.Context, s *models.Settings) error
	UpdateLogo(ctx context.Context, logoPath string) error
}

type SettingsRepo struct {
	db *gorm.DB
}

func NewSettingsRepo(db *gorm.DB) *SettingsRepo {
	return &SettingsRepo{db: db}
}

func (r *SettingsRepo) Get(ctx context.Context) (*models.Settings, error) {
	var s models.Settings
	if err := r.db.WithContext(ctx).First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SettingsRepo) GetByWebhookToken(ctx context.Context, token string) (*models.Settings, error) {
	if token == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var s models.Settings
	if err := r.db.WithContext(ctx).
		Where("webhook_token = ?", token).
		First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SettingsRepo) Upsert(ctx context.Context, s *models.Settings) error {
	if s.ID == 0 {
		s.ID = 1
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			UpdateAll: true,
		}).
		Create(s).Error
}

func (r *SettingsRepo) UpdateLogo(ctx context.Context, logoPath string) error {
	return r.db.WithContext(ctx).
		Model(&models.Settings{}).
		Where("id = 1").
		Update("logo_path", logoPath).Error
}
