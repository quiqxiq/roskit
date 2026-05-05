package repository

import (
	"context"
	"fmt"

	"github.com/quiqxiq/roskit/internal/models"
	"gorm.io/gorm"
)

type TemplateRepository interface {
	Create(ctx context.Context, t *models.PrintTemplate) error
	GetByID(ctx context.Context, tenantID *uint, id uint) (*models.PrintTemplate, error)
	List(ctx context.Context, tenantID uint) ([]models.PrintTemplate, error)
	ListGlobal(ctx context.Context) ([]models.PrintTemplate, error)
	Update(ctx context.Context, t *models.PrintTemplate) error
	Delete(ctx context.Context, tenantID *uint, id uint) error
	GetByTenantAndType(ctx context.Context, tenantID uint, templateType string) ([]models.PrintTemplate, error)
	GetGlobalByType(ctx context.Context, templateType string) ([]models.PrintTemplate, error)
	CountGlobal(ctx context.Context) (int64, error)
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

// GetByID scopes lookup by tenant. Pass tenantID=nil to fetch a global default template.
func (r *TemplateRepo) GetByID(ctx context.Context, tenantID *uint, id uint) (*models.PrintTemplate, error) {
	var t models.PrintTemplate
	q := r.db.WithContext(ctx).Where("id = ?", id)
	if tenantID == nil {
		q = q.Where("tenant_id IS NULL")
	} else {
		q = q.Where("tenant_id = ?", *tenantID)
	}
	if err := q.First(&t).Error; err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}
	return &t, nil
}

func (r *TemplateRepo) List(ctx context.Context, tenantID uint) ([]models.PrintTemplate, error) {
	var templates []models.PrintTemplate
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("name ASC, part ASC").
		Find(&templates).Error; err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	return templates, nil
}

func (r *TemplateRepo) ListGlobal(ctx context.Context) ([]models.PrintTemplate, error) {
	var templates []models.PrintTemplate
	if err := r.db.WithContext(ctx).
		Where("tenant_id IS NULL").
		Order("name ASC, part ASC").
		Find(&templates).Error; err != nil {
		return nil, fmt.Errorf("list global templates: %w", err)
	}
	return templates, nil
}

func (r *TemplateRepo) Update(ctx context.Context, t *models.PrintTemplate) error {
	return r.db.WithContext(ctx).Save(t).Error
}

func (r *TemplateRepo) Delete(ctx context.Context, tenantID *uint, id uint) error {
	q := r.db.WithContext(ctx).Where("id = ?", id)
	if tenantID == nil {
		q = q.Where("tenant_id IS NULL")
	} else {
		q = q.Where("tenant_id = ?", *tenantID)
	}
	result := q.Delete(&models.PrintTemplate{})
	if result.Error != nil {
		return fmt.Errorf("delete template: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("template not found")
	}
	return nil
}

func (r *TemplateRepo) GetByTenantAndType(ctx context.Context, tenantID uint, templateType string) ([]models.PrintTemplate, error) {
	var templates []models.PrintTemplate
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND type = ?", tenantID, templateType).
		Order("part ASC").
		Find(&templates).Error
	if err != nil {
		return nil, fmt.Errorf("get templates by type: %w", err)
	}
	return templates, nil
}

func (r *TemplateRepo) GetGlobalByType(ctx context.Context, templateType string) ([]models.PrintTemplate, error) {
	var templates []models.PrintTemplate
	err := r.db.WithContext(ctx).
		Where("tenant_id IS NULL AND type = ?", templateType).
		Order("part ASC").
		Find(&templates).Error
	if err != nil {
		return nil, fmt.Errorf("get global templates by type: %w", err)
	}
	return templates, nil
}

func (r *TemplateRepo) CountGlobal(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.PrintTemplate{}).
		Where("tenant_id IS NULL").
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
