package services

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/quiqxiq/roskit/internal/models"
	"github.com/quiqxiq/roskit/internal/repository"
	"github.com/xuri/excelize/v2"

	appcache "github.com/quiqxiq/roskit/pkg/redis"
)

type DailyReport struct {
	Date  string           `json:"date"`
	Sales []models.VoucherSale `json:"sales"`
	Total int64            `json:"total"`
	Count int              `json:"count"`
}

type MonthlyReport struct {
	Year   int                  `json:"year"`
	Month  int                  `json:"month"`
	Daily  []DailySummaryRow    `json:"daily"`
	Total  int64                `json:"total"`
	Count  int64                `json:"count"`
}

type DailySummaryRow = repository.DailySummary

type ResumeReport struct {
	Year   int                `json:"year"`
	Months []MonthSummaryRow  `json:"months"`
	Total  int64              `json:"total"`
	Count  int64              `json:"count"`
}

type MonthSummaryRow struct {
	Month int    `json:"month"`
	Count int64  `json:"count"`
	Total int64  `json:"total"`
}

type DashboardSummary struct {
	TodayCount int64 `json:"today_count"`
	TodaySum   int64 `json:"today_sum"`
	MonthCount int64 `json:"month_count"`
	MonthSum   int64 `json:"month_sum"`
}

type SaleFilters struct {
	Profile string
	Server  string
	Search  string
}

type SaleRepository = repository.SaleRepository

type ReportService struct {
	saleRepo SaleRepository
	cache    *appcache.Cache
	logger   *slog.Logger
}

func NewReportService(saleRepo SaleRepository, cache *appcache.Cache) *ReportService {
	return &ReportService{
		saleRepo: saleRepo,
		cache:    cache,
		logger:   slog.Default().With("component", "report-svc"),
	}
}

func (s *ReportService) GetDailyReport(ctx context.Context, routerID uint, date time.Time, filters SaleFilters) (*DailyReport, error) {
	isPast := date.Before(time.Now().Truncate(24 * time.Hour))
	ttl := appcache.TTLSalesDay
	if isPast {
		ttl = appcache.TTLSalesPast
	}
	cacheKey := appcache.SalesKey(routerID, "day:"+date.Format("2006-01-02"))

	return appcache.GetOrSetJSON(s.cache, ctx, cacheKey, ttl, func() (*DailyReport, error) {
		sales, err := s.saleRepo.GetByDay(ctx, routerID, date)
		if err != nil {
			return nil, err
		}

		filtered := applySaleFilters(sales, filters)

		var total int64
		for _, sale := range filtered {
			total += sale.Price
		}

		return &DailyReport{
			Date:  date.Format("2006-01-02"),
			Sales: pointerSliceToValues(filtered),
			Total: total,
			Count: len(filtered),
		}, nil
	})
}

func (s *ReportService) GetMonthlyReport(ctx context.Context, routerID uint, year, month int) (*MonthlyReport, error) {
	cacheKey := appcache.SalesKey(routerID, fmt.Sprintf("month:%d-%02d", year, month))
	ttl := appcache.TTLSalesMonth

	return appcache.GetOrSetJSON(s.cache, ctx, cacheKey, ttl, func() (*MonthlyReport, error) {
		tm := time.Month(month)
		sales, err := s.saleRepo.GetByMonth(ctx, routerID, year, tm)
		if err != nil {
			return nil, err
		}

		var total int64
		for _, sale := range sales {
			total += sale.Price
		}

		from := time.Date(year, tm, 1, 0, 0, 0, 0, time.UTC)
		to := from.AddDate(0, 1, 0)
		daily, err := s.saleRepo.GetDailySummary(ctx, routerID, from, to)
		if err != nil {
			return nil, err
		}

		rows := make([]DailySummaryRow, len(daily))
		for i, d := range daily {
			rows[i] = DailySummaryRow{Date: d.Date, Count: d.Count, Sum: d.Sum}
		}

		return &MonthlyReport{
			Year:   year,
			Month:  month,
			Daily:  rows,
			Total:  total,
			Count:  int64(len(sales)),
		}, nil
	})
}

func (s *ReportService) GetResumeReport(ctx context.Context, routerID uint, year int) (*ResumeReport, error) {
	cacheKey := appcache.SalesKey(routerID, fmt.Sprintf("resume:%d", year))
	ttl := appcache.TTLSalesPast

	return appcache.GetOrSetJSON(s.cache, ctx, cacheKey, ttl, func() (*ResumeReport, error) {
		summary, err := s.saleRepo.GetMonthlySummary(ctx, routerID, year)
		if err != nil {
			return nil, err
		}

		months := make([]MonthSummaryRow, 12)
		var totalCount, totalSum int64
		for _, sm := range summary {
			idx := int(sm.Month) - 1
			if idx >= 0 && idx < 12 {
				months[idx] = MonthSummaryRow{Month: int(sm.Month), Count: sm.Count, Total: sm.Sum}
			}
			totalCount += sm.Count
			totalSum += sm.Sum
		}

		for i := 0; i < 12; i++ {
			if months[i].Count == 0 && months[i].Total == 0 {
				months[i].Month = i + 1
			}
		}

		return &ResumeReport{
			Year:   year,
			Months: months,
			Total:  totalSum,
			Count:  totalCount,
		}, nil
	})
}

func (s *ReportService) GetDashboardSummary(ctx context.Context, routerID uint) (*DashboardSummary, error) {
	cacheKey := appcache.DashboardKey(routerID)
	ttl := appcache.TTLDashboard

	return appcache.GetOrSetJSON(s.cache, ctx, cacheKey, ttl, func() (*DashboardSummary, error) {
		today, err := s.saleRepo.TodayTotal(ctx, routerID)
		if err != nil {
			s.logger.Warn("failed to get today total", "error", err)
		}
		month, err := s.saleRepo.MonthTotal(ctx, routerID)
		if err != nil {
			s.logger.Warn("failed to get month total", "error", err)
		}
		return &DashboardSummary{
			TodayCount: today.Count,
			TodaySum:   today.Sum,
			MonthCount: month.Count,
			MonthSum:   month.Sum,
		}, nil
	})
}

func (s *ReportService) ExportCSV(ctx context.Context, routerID uint, from, to time.Time, filters SaleFilters) ([]byte, error) {
	cacheKey := appcache.SalesKey(routerID, fmt.Sprintf("csv:%s:%s", from.Format("2006-01-02"), to.Format("2006-01-02")))
	ttl := appcache.TTLSalesPast

	return appcache.GetOrSetJSON(s.cache, ctx, cacheKey, ttl, func() ([]byte, error) {
		fromDay := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
		toDay := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, to.Location()).Add(24 * time.Hour)

		sales, err := s.saleRepo.GetByDay(ctx, routerID, fromDay)
		if err != nil {
			return nil, err
		}

		for day := fromDay.Add(24 * time.Hour); day.Before(toDay); day = day.Add(24 * time.Hour) {
			daySales, err := s.saleRepo.GetByDay(ctx, routerID, day)
			if err != nil {
				continue
			}
			sales = append(sales, daySales...)
		}

		filtered := applySaleFilters(sales, filters)

		var buf bytes.Buffer
		w := csv.NewWriter(&buf)
		w.Write([]string{"Date", "Time", "Username", "Profile", "Price", "Validity", "MAC", "IP", "Comment", "Server"})

		for _, sale := range filtered {
			w.Write([]string{
				sale.SoldAt.Format("2006-01-02"),
				sale.SoldAt.Format("15:04:05"),
				sale.Username,
				sale.ProfileName,
				fmt.Sprintf("%d", sale.Price),
				sale.Validity,
				sale.MACAddress,
				sale.IPAddress,
				"",
				sale.Server,
			})
		}
		w.Flush()

		if err := w.Error(); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	})
}

func (s *ReportService) ExportExcel(ctx context.Context, routerID uint, from, to time.Time, filters SaleFilters) ([]byte, error) {
	cacheKey := appcache.SalesKey(routerID, fmt.Sprintf("xlsx:%s:%s", from.Format("2006-01-02"), to.Format("2006-01-02")))
	ttl := appcache.TTLSalesPast

	return appcache.GetOrSetJSON(s.cache, ctx, cacheKey, ttl, func() ([]byte, error) {
		fromDay := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
		toDay := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, to.Location()).Add(24 * time.Hour)

		sales, err := s.saleRepo.GetByDay(ctx, routerID, fromDay)
		if err != nil {
			return nil, err
		}

		for day := fromDay.Add(24 * time.Hour); day.Before(toDay); day = day.Add(24 * time.Hour) {
			daySales, err := s.saleRepo.GetByDay(ctx, routerID, day)
			if err != nil {
				continue
			}
			sales = append(sales, daySales...)
		}

		filtered := applySaleFilters(sales, filters)

		f := excelize.NewFile()
		sheet := "Sales Report"
		index, err := f.NewSheet(sheet)
		if err != nil {
			return nil, fmt.Errorf("create sheet: %w", err)
		}
		f.SetActiveSheet(index)
		f.DeleteSheet("Sheet1")

		headers := []string{"Date", "Time", "Username", "Profile", "Price", "Validity", "MAC", "IP", "Comment", "Server"}
		headerStyle, _ := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true},
		})
		for col, h := range headers {
			cell, _ := excelize.CoordinatesToCellName(col+1, 1)
			f.SetCellValue(sheet, cell, h)
			f.SetCellStyle(sheet, cell, cell, headerStyle)
		}

		for i, sale := range filtered {
			row := i + 2
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), sale.SoldAt.Format("2006-01-02"))
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), sale.SoldAt.Format("15:04:05"))
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), sale.Username)
			f.SetCellValue(sheet, fmt.Sprintf("D%d", row), sale.ProfileName)
			f.SetCellValue(sheet, fmt.Sprintf("E%d", row), sale.Price)
			f.SetCellValue(sheet, fmt.Sprintf("F%d", row), sale.Validity)
			f.SetCellValue(sheet, fmt.Sprintf("G%d", row), sale.MACAddress)
			f.SetCellValue(sheet, fmt.Sprintf("H%d", row), sale.IPAddress)
			f.SetCellValue(sheet, fmt.Sprintf("I%d", row), "")
			f.SetCellValue(sheet, fmt.Sprintf("J%d", row), sale.Server)
		}

		buf, err := f.WriteToBuffer()
		if err != nil {
			return nil, fmt.Errorf("write excel: %w", err)
		}
		return buf.Bytes(), nil
	})
}

func applySaleFilters(sales []*models.VoucherSale, filters SaleFilters) []*models.VoucherSale {
	if filters.Profile == "" && filters.Server == "" && filters.Search == "" {
		return sales
	}
	filtered := make([]*models.VoucherSale, 0, len(sales))
	for _, sale := range sales {
		if filters.Profile != "" && sale.ProfileName != filters.Profile {
			continue
		}
		if filters.Server != "" && sale.Server != filters.Server {
			continue
		}
		if filters.Search != "" && !strings.HasPrefix(sale.Username, filters.Search) {
			continue
		}
		filtered = append(filtered, sale)
	}
	return filtered
}

func pointerSliceToValues(sales []*models.VoucherSale) []models.VoucherSale {
	result := make([]models.VoucherSale, len(sales))
	for i, s := range sales {
		result[i] = *s
	}
	return result
}
