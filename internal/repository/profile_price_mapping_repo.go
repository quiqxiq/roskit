package repository

import (
	"context"

	"github.com/quiqxiq/roskit/internal/models"
	"gorm.io/gorm"
)

type ProfilePriceMappingRepository interface {
	Upsert(ctx context.Context, mapping *models.ProfilePriceMapping) error
	FindByRouterAndProfile(ctx context.Context, routerID uint, profileName string) (*models.ProfilePriceMapping, error)
	ListByRouter(ctx context.Context, routerID uint) ([]*models.ProfilePriceMapping, error)
	DeleteByRouterAndProfile(ctx context.Context, routerID uint, profileName string) error
}

type profilePriceMappingRepo struct {
	db *gorm.DB
}

func NewProfilePriceMappingRepo(db *gorm.DB) ProfilePriceMappingRepository {
	return &profilePriceMappingRepo{db: db}
}

func (r *profilePriceMappingRepo) Upsert(ctx context.Context, mapping *models.ProfilePriceMapping) error {
	var existing models.ProfilePriceMapping
	err := r.db.WithContext(ctx).
		Where("profile_name = ? AND router_id = ?", mapping.ProfileName, mapping.RouterID).
		First(&existing).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return r.db.WithContext(ctx).Create(mapping).Error
		}
		return err
	}

	mapping.ID = existing.ID
	return r.db.WithContext(ctx).Save(mapping).Error
}

func (r *profilePriceMappingRepo) FindByRouterAndProfile(ctx context.Context, routerID uint, profileName string) (*models.ProfilePriceMapping, error) {
	var m models.ProfilePriceMapping
	err := r.db.WithContext(ctx).Where("router_id = ? AND profile_name = ?", routerID, profileName).First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *profilePriceMappingRepo) ListByRouter(ctx context.Context, routerID uint) ([]*models.ProfilePriceMapping, error) {
	var mappings []*models.ProfilePriceMapping
	err := r.db.WithContext(ctx).
		Where("router_id = ?", routerID).
		Order("profile_name ASC").
		Find(&mappings).Error
	return mappings, err
}

func (r *profilePriceMappingRepo) DeleteByRouterAndProfile(ctx context.Context, routerID uint, profileName string) error {
	result := r.db.WithContext(ctx).
		Where("router_id = ? AND profile_name = ?", routerID, profileName).
		Delete(&models.ProfilePriceMapping{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
