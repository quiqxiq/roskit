package repository

import (
	"context"

	"github.com/quiqxiq/roskit/internal/models"
	"gorm.io/gorm"
)

type ProfilePriceMappingRepository interface {
	Upsert(ctx context.Context, mapping *models.ProfilePriceMapping) error
	FindByRouterAndProfile(ctx context.Context, routerID uint, profileName string) (*models.ProfilePriceMapping, error)
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
