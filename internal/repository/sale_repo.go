package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/quiqxiq/roskit/internal/models"
	"gorm.io/gorm"
)

type SaleRepository interface {
	Create(ctx context.Context, sale *models.VoucherSale) error
	CreateBatch(ctx context.Context, sales []*models.VoucherSale) error
	ExistsByIdempotencyKey(ctx context.Context, key string) (bool, error)
	GetByDateRange(ctx context.Context, tenantID uint, routerID *uint, from, to time.Time, filters SaleFilters) ([]*models.VoucherSale, error)
	GetByDay(ctx context.Context, tenantID uint, routerID *uint, date time.Time) ([]*models.VoucherSale, error)
	GetByMonth(ctx context.Context, tenantID uint, routerID *uint, year int, month time.Month) ([]*models.VoucherSale, error)
	GetDailySummary(ctx context.Context, tenantID uint, routerID *uint, from, to time.Time) ([]DailySummary, error)
	GetMonthlySummary(ctx context.Context, tenantID uint, routerID *uint, year int) ([]MonthlySummary, error)
	TodayTotal(ctx context.Context, tenantID uint, routerID *uint) (TotalResult, error)
	MonthTotal(ctx context.Context, tenantID uint, routerID *uint) (TotalResult, error)
}

type SaleFilters struct {
	Profile string
	Server  string
	Search  string
}

type DailySummary struct {
	Date  time.Time `json:"date"`
	Count int64     `json:"count"`
	Sum   int64     `json:"sum"`
}

type MonthlySummary struct {
	Month time.Month `json:"month"`
	Count int64      `json:"count"`
	Sum   int64      `json:"sum"`
}

type TotalResult struct {
	Count int64 `json:"count"`
	Sum   int64 `json:"sum"`
}

type SaleRepo struct {
	db *gorm.DB
}

func NewSaleRepo(db *gorm.DB) *SaleRepo {
	return &SaleRepo{db: db}
}

func (r *SaleRepo) Create(ctx context.Context, sale *models.VoucherSale) error {
	return r.db.WithContext(ctx).Create(sale).Error
}

func (r *SaleRepo) CreateBatch(ctx context.Context, sales []*models.VoucherSale) error {
	if len(sales) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(sales, 500).Error
}

func (r *SaleRepo) ExistsByIdempotencyKey(ctx context.Context, key string) (bool, error) {
	var exists bool
	err := r.db.WithContext(ctx).
		Model(&models.VoucherSale{}).
		Select("1").
		Where("idempotency_key = ?", key).
		Limit(1).
		Scan(&exists).Error
	return exists, err
}

// scopedQuery returns a *gorm.DB pre-filtered by tenant and optionally by router.
func (r *SaleRepo) scopedQuery(ctx context.Context, tenantID uint, routerID *uint) *gorm.DB {
	q := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	if routerID != nil {
		q = q.Where("router_id = ?", *routerID)
	}
	return q
}

func (r *SaleRepo) GetByDateRange(ctx context.Context, tenantID uint, routerID *uint, from, to time.Time, filters SaleFilters) ([]*models.VoucherSale, error) {
	q := r.scopedQuery(ctx, tenantID, routerID).
		Where("sold_at >= ? AND sold_at < ?", from, to)

	if filters.Profile != "" {
		q = q.Where("profile_name = ?", filters.Profile)
	}
	if filters.Server != "" {
		q = q.Where("server = ?", filters.Server)
	}
	if filters.Search != "" {
		q = q.Where("username LIKE ?", filters.Search+"%")
	}

	var sales []*models.VoucherSale
	if err := q.Order("sold_at DESC").Find(&sales).Error; err != nil {
		return nil, err
	}
	return sales, nil
}

func (r *SaleRepo) GetByDay(ctx context.Context, tenantID uint, routerID *uint, date time.Time) ([]*models.VoucherSale, error) {
	from := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	to := from.Add(24 * time.Hour)

	var sales []*models.VoucherSale
	if err := r.scopedQuery(ctx, tenantID, routerID).
		Where("sold_at >= ? AND sold_at < ?", from, to).
		Order("sold_at DESC").
		Find(&sales).Error; err != nil {
		return nil, err
	}
	return sales, nil
}

func (r *SaleRepo) GetByMonth(ctx context.Context, tenantID uint, routerID *uint, year int, month time.Month) ([]*models.VoucherSale, error) {
	from := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)

	var sales []*models.VoucherSale
	if err := r.scopedQuery(ctx, tenantID, routerID).
		Where("sold_at >= ? AND sold_at < ?", from, to).
		Order("sold_at DESC").
		Find(&sales).Error; err != nil {
		return nil, err
	}
	return sales, nil
}

func (r *SaleRepo) GetDailySummary(ctx context.Context, tenantID uint, routerID *uint, from, to time.Time) ([]DailySummary, error) {
	type row struct {
		Day   time.Time
		Count int64
		Sum   int64
	}
	var rows []row
	err := r.scopedQuery(ctx, tenantID, routerID).
		Model(&models.VoucherSale{}).
		Select("DATE(sold_at) as day, COUNT(*) as count, COALESCE(SUM(price), 0) as sum").
		Where("sold_at >= ? AND sold_at < ?", from, to).
		Group("DATE(sold_at)").
		Order("day ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]DailySummary, len(rows))
	for i, row := range rows {
		result[i] = DailySummary{Date: row.Day, Count: row.Count, Sum: row.Sum}
	}
	return result, nil
}

func (r *SaleRepo) GetMonthlySummary(ctx context.Context, tenantID uint, routerID *uint, year int) ([]MonthlySummary, error) {
	type row struct {
		Month time.Month
		Count int64
		Sum   int64
	}
	var rows []row
	from := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(1, 0, 0)

	err := r.scopedQuery(ctx, tenantID, routerID).
		Model(&models.VoucherSale{}).
		Select("EXTRACT(MONTH FROM sold_at) as month, COUNT(*) as count, COALESCE(SUM(price), 0) as sum").
		Where("sold_at >= ? AND sold_at < ?", from, to).
		Group("EXTRACT(MONTH FROM sold_at)").
		Order("month ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]MonthlySummary, len(rows))
	for i, row := range rows {
		result[i] = MonthlySummary{Month: row.Month, Count: row.Count, Sum: row.Sum}
	}
	return result, nil
}

func (r *SaleRepo) TodayTotal(ctx context.Context, tenantID uint, routerID *uint) (TotalResult, error) {
	now := time.Now()
	from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	to := from.Add(24 * time.Hour)

	type row struct {
		Count int64
		Sum   int64
	}
	var r2 row
	err := r.scopedQuery(ctx, tenantID, routerID).
		Model(&models.VoucherSale{}).
		Select("COUNT(*) as count, COALESCE(SUM(price), 0) as sum").
		Where("sold_at >= ? AND sold_at < ?", from, to).
		Scan(&r2).Error
	if err != nil {
		return TotalResult{}, err
	}
	return TotalResult{Count: r2.Count, Sum: r2.Sum}, nil
}

func (r *SaleRepo) MonthTotal(ctx context.Context, tenantID uint, routerID *uint) (TotalResult, error) {
	now := time.Now()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	to := from.AddDate(0, 1, 0)

	type row struct {
		Count int64
		Sum   int64
	}
	var r2 row
	err := r.scopedQuery(ctx, tenantID, routerID).
		Model(&models.VoucherSale{}).
		Select("COUNT(*) as count, COALESCE(SUM(price), 0) as sum").
		Where("sold_at >= ? AND sold_at < ?", from, to).
		Scan(&r2).Error
	if err != nil {
		return TotalResult{}, fmt.Errorf("month total: %w", err)
	}
	return TotalResult{Count: r2.Count, Sum: r2.Sum}, nil
}
