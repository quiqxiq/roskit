package repository

import (
	"context"

	"github.com/quiqxiq/roskit/internal/models"
	"gorm.io/gorm"
)

type TenantRepository interface {
	Create(ctx context.Context, tenant *models.Tenant) error
	GetByID(ctx context.Context, id uint) (*models.Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*models.Tenant, error)
	List(ctx context.Context) ([]*models.Tenant, error)
	Update(ctx context.Context, tenant *models.Tenant) error
	UpdateStatus(ctx context.Context, id uint, status models.TenantStatus) error
	SoftDelete(ctx context.Context, id uint) error
	HardDelete(ctx context.Context, id uint) error
	Count(ctx context.Context) (int64, error)
}

type TenantRepo struct {
	db *gorm.DB
}

func NewTenantRepo(db *gorm.DB) *TenantRepo {
	return &TenantRepo{db: db}
}

func (r *TenantRepo) Create(ctx context.Context, tenant *models.Tenant) error {
	return r.db.WithContext(ctx).Create(tenant).Error
}

func (r *TenantRepo) GetByID(ctx context.Context, id uint) (*models.Tenant, error) {
	var t models.Tenant
	if err := r.db.WithContext(ctx).Preload("Settings").First(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TenantRepo) GetBySlug(ctx context.Context, slug string) (*models.Tenant, error) {
	var t models.Tenant
	if err := r.db.WithContext(ctx).
		Preload("Settings").
		Where("slug = ?", slug).
		First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TenantRepo) List(ctx context.Context) ([]*models.Tenant, error) {
	var tenants []*models.Tenant
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&tenants).Error; err != nil {
		return nil, err
	}
	return tenants, nil
}

func (r *TenantRepo) Update(ctx context.Context, tenant *models.Tenant) error {
	return r.db.WithContext(ctx).Save(tenant).Error
}

func (r *TenantRepo) UpdateStatus(ctx context.Context, id uint, status models.TenantStatus) error {
	return r.db.WithContext(ctx).
		Model(&models.Tenant{}).
		Where("id = ?", id).
		Update("status", status).Error
}

func (r *TenantRepo) SoftDelete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Tenant{}, id).Error
}

// HardDelete bypasses soft-delete and triggers FK CASCADE on all child tables.
func (r *TenantRepo) HardDelete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Unscoped().Delete(&models.Tenant{}, id).Error
}

func (r *TenantRepo) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Tenant{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
