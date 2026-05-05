package repository

import (
	"context"
	"time"

	"github.com/quiqxiq/roskit/internal/models"
	"gorm.io/gorm"
)

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
	UpdateTimezone(ctx context.Context, routerID string, tz string) error
}

type RouterRepo struct {
	db *gorm.DB
}

func NewRouterRepo(db *gorm.DB) *RouterRepo {
	return &RouterRepo{db: db}
}

func (r *RouterRepo) Create(ctx context.Context, router *models.Router) error {
	return r.db.WithContext(ctx).Create(router).Error
}

func (r *RouterRepo) GetByID(ctx context.Context, tenantID, id uint) (*models.Router, error) {
	var router models.Router
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		First(&router).Error; err != nil {
		return nil, err
	}
	return &router, nil
}

// GetByIDAny fetches a router without tenant constraint. Reserved for the engine/orchestrator
// layer which operates across tenants (e.g. seeding the pool). Handlers must not use this.
func (r *RouterRepo) GetByIDAny(ctx context.Context, id uint) (*models.Router, error) {
	var router models.Router
	if err := r.db.WithContext(ctx).First(&router, id).Error; err != nil {
		return nil, err
	}
	return &router, nil
}

func (r *RouterRepo) GetByName(ctx context.Context, tenantID uint, name string) (*models.Router, error) {
	var router models.Router
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND name = ? AND deleted_at IS NULL", tenantID, name).
		First(&router).Error; err != nil {
		return nil, err
	}
	return &router, nil
}

func (r *RouterRepo) List(ctx context.Context, tenantID uint) ([]*models.Router, error) {
	var routers []*models.Router
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("id ASC").
		Find(&routers).Error; err != nil {
		return nil, err
	}
	return routers, nil
}

// ListAll returns routers across all tenants; used by engine startup & cross-tenant reports.
func (r *RouterRepo) ListAll(ctx context.Context) ([]*models.Router, error) {
	var routers []*models.Router
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&routers).Error; err != nil {
		return nil, err
	}
	return routers, nil
}

func (r *RouterRepo) Update(ctx context.Context, router *models.Router) error {
	return r.db.WithContext(ctx).Save(router).Error
}

func (r *RouterRepo) Delete(ctx context.Context, tenantID, id uint) error {
	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		Delete(&models.Router{}).Error
}

func (r *RouterRepo) UpdateStatus(ctx context.Context, routerID uint, status models.RouterStatus) error {
	return r.db.WithContext(ctx).Model(&models.Router{}).
		Where("id = ?", routerID).
		Update("status", status).Error
}

func (r *RouterRepo) UpdateLastSeen(ctx context.Context, routerID uint, t time.Time) error {
	return r.db.WithContext(ctx).Model(&models.Router{}).
		Where("id = ?", routerID).
		Update("last_seen_at", t).Error
}

// UpdateTimezone writes timezone to the router's tenant_settings row (not routers table).
// routerID arrives as string from the orchestrator/engine layer.
func (r *RouterRepo) UpdateTimezone(ctx context.Context, routerID string, tz string) error {
	return r.db.WithContext(ctx).
		Model(&models.TenantSettings{}).
		Where("tenant_id IN (SELECT tenant_id FROM routers WHERE id = ?)", routerID).
		Update("timezone", tz).Error
}
