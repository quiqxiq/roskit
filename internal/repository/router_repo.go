package repository

import (
	"context"

	"github.com/quiqxiq/roskit/internal/models"
	"gorm.io/gorm"
)

type RouterRepository interface {
	Create(ctx context.Context, router *models.Router) error
	GetByID(ctx context.Context, id uint) (*models.Router, error)
	GetBySessionName(ctx context.Context, sessionName string) (*models.Router, error)
	List(ctx context.Context) ([]*models.Router, error)
	Update(ctx context.Context, router *models.Router) error
	Delete(ctx context.Context, id uint) error
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

func (r *RouterRepo) GetByID(ctx context.Context, id uint) (*models.Router, error) {
	var router models.Router
	if err := r.db.WithContext(ctx).First(&router, id).Error; err != nil {
		return nil, err
	}
	return &router, nil
}

func (r *RouterRepo) GetBySessionName(ctx context.Context, sessionName string) (*models.Router, error) {
	var router models.Router
	if err := r.db.WithContext(ctx).
		Where("session_name = ? AND deleted_at IS NULL", sessionName).
		First(&router).Error; err != nil {
		return nil, err
	}
	return &router, nil
}

func (r *RouterRepo) List(ctx context.Context) ([]*models.Router, error) {
	var routers []*models.Router
	if err := r.db.WithContext(ctx).Find(&routers).Error; err != nil {
		return nil, err
	}
	return routers, nil
}

func (r *RouterRepo) Update(ctx context.Context, router *models.Router) error {
	return r.db.WithContext(ctx).Save(router).Error
}

func (r *RouterRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Router{}, id).Error
}

func (r *RouterRepo) UpdateTimezone(ctx context.Context, routerID string, tz string) error {
	return r.db.WithContext(ctx).Model(&models.Router{}).
		Where("id = ?", routerID).
		Update("timezone", tz).Error
}
