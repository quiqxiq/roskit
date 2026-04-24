package repository

import (
	"context"
	"fmt"

	"github.com/quiqxiq/roskit/internal/models"
	"gorm.io/gorm"
)

type TemplateRepository interface {
	Create(ctx context.Context, t *models.PrintTemplate) error
	GetByID(ctx context.Context, id uint) (*models.PrintTemplate, error)
	List(ctx context.Context, routerID uint) ([]models.PrintTemplate, error)
	Update(ctx context.Context, t *models.PrintTemplate) error
	Delete(ctx context.Context, id uint) error
	GetByRouterAndType(ctx context.Context, routerID uint, templateType string) ([]models.PrintTemplate, error)
}

type TemplateRepo struct {
	db *gorm.DB
}

func NewTemplateRepo(db *gorm.DB) *TemplateRepo {
	return &TemplateRepo{db: db}
}

func (r *TemplateRepo) Create(ctx context.Context, t *models.PrintTemplate) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *TemplateRepo) GetByID(ctx context.Context, id uint) (*models.PrintTemplate, error) {
	var t models.PrintTemplate
	if err := r.db.WithContext(ctx).First(&t, id).Error; err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}
	return &t, nil
}

func (r *TemplateRepo) List(ctx context.Context, routerID uint) ([]models.PrintTemplate, error) {
	var templates []models.PrintTemplate
	if err := r.db.WithContext(ctx).
		Where("router_id = ? OR router_id IS NULL", routerID).
		Order("name ASC, part ASC").
		Find(&templates).Error; err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	return templates, nil
}

func (r *TemplateRepo) Update(ctx context.Context, t *models.PrintTemplate) error {
	return r.db.WithContext(ctx).Save(t).Error
}

func (r *TemplateRepo) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.PrintTemplate{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete template: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("template not found")
	}
	return nil
}

func (r *TemplateRepo) GetByRouterAndType(ctx context.Context, routerID uint, templateType string) ([]models.PrintTemplate, error) {
	var templates []models.PrintTemplate
	err := r.db.WithContext(ctx).
		Where("(router_id = ? OR router_id IS NULL) AND type = ?", routerID, templateType).
		Order("part ASC").
		Find(&templates).Error
	if err != nil {
		return nil, fmt.Errorf("get templates by type: %w", err)
	}
	return templates, nil
}
