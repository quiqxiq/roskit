package repository

import (
	"context"

	"github.com/quiqxiq/roskit/internal/models"
	"gorm.io/gorm"
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id uint) (*models.User, error)
	GetByTenantUsername(ctx context.Context, tenantID *uint, username string) (*models.User, error)
	GetSuperAdminByUsername(ctx context.Context, username string) (*models.User, error)
	List(ctx context.Context, tenantID uint) ([]*models.User, error)
	ListSuperAdmins(ctx context.Context) ([]*models.User, error)
	Update(ctx context.Context, user *models.User) error
	UpdateLastLogin(ctx context.Context, userID uint) error
	Delete(ctx context.Context, id uint) error
	Count(ctx context.Context) (int64, error)
	CountByTenant(ctx context.Context, tenantID uint) (int64, error)
}

type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserRepo) GetByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByTenantUsername finds a user scoped to a tenant. Pass tenantID=nil to find
// a platform-level superadmin (tenant_id IS NULL).
func (r *UserRepo) GetByTenantUsername(ctx context.Context, tenantID *uint, username string) (*models.User, error) {
	var user models.User
	q := r.db.WithContext(ctx).Where("username = ?", username)
	if tenantID == nil {
		q = q.Where("tenant_id IS NULL")
	} else {
		q = q.Where("tenant_id = ?", *tenantID)
	}
	if err := q.First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) GetSuperAdminByUsername(ctx context.Context, username string) (*models.User, error) {
	return r.GetByTenantUsername(ctx, nil, username)
}

func (r *UserRepo) List(ctx context.Context, tenantID uint) ([]*models.User, error) {
	var users []*models.User
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("id ASC").
		Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepo) ListSuperAdmins(ctx context.Context) ([]*models.User, error) {
	var users []*models.User
	if err := r.db.WithContext(ctx).
		Where("tenant_id IS NULL").
		Order("id ASC").
		Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepo) Update(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *UserRepo) UpdateLastLogin(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", userID).
		Update("last_login_at", gorm.Expr("NOW()")).Error
}

func (r *UserRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.User{}, id).Error
}

func (r *UserRepo) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.User{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *UserRepo) CountByTenant(ctx context.Context, tenantID uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("tenant_id = ?", tenantID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
