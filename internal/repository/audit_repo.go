package repository

import (
	"context"
	"time"

	"github.com/quiqxiq/roskit/internal/models"
	"gorm.io/gorm"
)

type AuditRepository interface {
	Create(ctx context.Context, log *models.AuditLog) error
	ListByUser(ctx context.Context, userID uint, limit int) ([]*models.AuditLog, error)
}

type auditRepo struct {
	db *gorm.DB
}

func NewAuditRepo(db *gorm.DB) AuditRepository {
	return &auditRepo{db: db}
}

func (r *auditRepo) Create(ctx context.Context, log *models.AuditLog) error {
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *auditRepo) ListByUser(ctx context.Context, userID uint, limit int) ([]*models.AuditLog, error) {
	if limit <= 0 {
		limit = 50
	}
	var logs []*models.AuditLog
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}
