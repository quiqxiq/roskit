package services

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/quiqxiq/roskit/internal/models"
	roskitservice "github.com/quiqxiq/roskit/internal/roskit/adapter/service"
	appcache "github.com/quiqxiq/roskit/pkg/redis"
)

type VoucherGenerateParams struct {
	Qty        int    `json:"qty"`
	Server     string `json:"server"`
	UserType   string `json:"user_type"`
	NameLength int    `json:"name_length"`
	Prefix     string `json:"prefix"`
	CharSet    string `json:"char_set"`
	Profile    string `json:"profile"`
	TimeLimit  string `json:"time_limit"`
	DataLimit  int64  `json:"data_limit"`
	Comment    string `json:"comment"`
	Gencode    string `json:"gencode"`
}

type VoucherGenerateResult struct {
	Count    int                                `json:"count"`
	Gencode  string                             `json:"gencode"`
	Profile  string                             `json:"profile"`
	Vouchers []roskitservice.GeneratedVoucher   `json:"vouchers"`
}

type RecordSaleParams struct {
	Username    string    `json:"username"`
	ProfileName string    `json:"profile_name"`
	Price       int64     `json:"price"`
	Server      string    `json:"server"`
	IPAddress   string    `json:"ip_address"`
	MACAddress  string    `json:"mac_address"`
	SoldAt      time.Time `json:"sold_at"`
}

type ImportResult struct {
	Total    int `json:"total"`
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
	Errors   int `json:"errors"`
}

type VoucherService struct {
	bridge     *roskitservice.Bridge
	saleRepo   SaleRepository
	routerRepo RouterRepository
	cache      *appcache.Cache
	logger     *slog.Logger
}

func NewVoucherService(bridge *roskitservice.Bridge, saleRepo SaleRepository, routerRepo RouterRepository, cache *appcache.Cache) *VoucherService {
	return &VoucherService{
		bridge:     bridge,
		saleRepo:   saleRepo,
		routerRepo: routerRepo,
		cache:      cache,
		logger:     slog.Default().With("component", "voucher-svc"),
	}
}

func (s *VoucherService) GenerateVoucher(ctx context.Context, routerID uint, params VoucherGenerateParams) (*VoucherGenerateResult, error) {
	_, err := s.routerRepo.GetByID(ctx, routerID)
	if err != nil {
		return nil, fmt.Errorf("router not found: %w", err)
	}

	if params.NameLength <= 0 {
		params.NameLength = 6
	}
	if params.CharSet == "" {
		params.CharSet = "mix"
	}
	if params.UserType == "" {
		params.UserType = "vc"
	}

	comment := roskitservice.GenerateVoucherComment(params.UserType, params.Gencode, "", params.Comment)

	rosParams := roskitservice.VoucherParams{
		Qty:        params.Qty,
		Server:     params.Server,
		UserType:   params.UserType,
		NameLength: params.NameLength,
		Prefix:     params.Prefix,
		CharSet:    params.CharSet,
		Profile:    params.Profile,
		TimeLimit:  params.TimeLimit,
		DataLimit:  fmt.Sprintf("%d", params.DataLimit),
		Comment:    comment,
		Gencode:    params.Gencode,
	}

	vouchers, err := s.bridge.GenerateAndCreateVouchers(ctx, fmt.Sprintf("%d", routerID), rosParams)
	if err != nil {
		return nil, fmt.Errorf("generate vouchers: %w", err)
	}

	if s.cache != nil {
		key := appcache.VoucherSessionKey(routerID, params.Gencode)
		result := &VoucherGenerateResult{
			Count:    len(vouchers),
			Gencode:  params.Gencode,
			Profile:  params.Profile,
			Vouchers: vouchers,
		}
		if err := s.cache.SetJSON(ctx, key, result, appcache.TTLVoucherSession); err != nil {
			s.logger.Warn("failed to cache voucher session", "error", err)
		}
	}

	return &VoucherGenerateResult{
		Count:    len(vouchers),
		Gencode:  params.Gencode,
		Profile:  params.Profile,
		Vouchers: vouchers,
	}, nil
}

func (s *VoucherService) GetCachedVouchers(ctx context.Context, routerID uint, gencode string) (*VoucherGenerateResult, error) {
	if s.cache == nil {
		return nil, fmt.Errorf("cache not available")
	}

	key := appcache.VoucherSessionKey(routerID, gencode)
	var result VoucherGenerateResult
	found, err := s.cache.GetJSON(ctx, key, &result)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("voucher session not found or expired")
	}
	return &result, nil
}

func (s *VoucherService) RecordSale(ctx context.Context, routerID uint, params RecordSaleParams) error {
	key := idempotencyKey(routerID, params.Username, params.SoldAt)
	exists, err := s.saleRepo.ExistsByIdempotencyKey(ctx, key)
	if err != nil {
		return fmt.Errorf("check idempotency: %w", err)
	}
	if exists {
		return nil
	}

	sale := &models.VoucherSale{
		RouterID:       routerID,
		SoldAt:         params.SoldAt,
		Username:       params.Username,
		ProfileName:    params.ProfileName,
		Price:          params.Price,
		Server:         params.Server,
		IPAddress:      params.IPAddress,
		MACAddress:     params.MACAddress,
		IdempotencyKey: key,
	}

	if err := s.saleRepo.Create(ctx, sale); err != nil {
		return fmt.Errorf("save sale: %w", err)
	}

	s.invalidateSalesCache(ctx, routerID)
	return nil
}

func (s *VoucherService) ImportSalesFromRouterOS(ctx context.Context, routerID uint) (*ImportResult, error) {
	router, err := s.routerRepo.GetByID(ctx, routerID)
	if err != nil {
		return nil, fmt.Errorf("router not found: %w", err)
	}

	records, err := s.bridge.ImportSalesFromRouterOS(ctx, fmt.Sprintf("%d", routerID), "")
	if err != nil {
		return nil, fmt.Errorf("fetch sales from router: %w", err)
	}

	result := &ImportResult{}
	result.Total = len(records)
	now := time.Now()

	var newSales []*models.VoucherSale
	for _, rec := range records {
		idKey := fmt.Sprintf("%s|%s|%s|%s", router.SessionName, rec.Username, rec.Date, rec.Time)
		key := sha256Sum(idKey)

		exists, err := s.saleRepo.ExistsByIdempotencyKey(ctx, key)
		if err != nil {
			result.Errors++
			continue
		}
		if exists {
			result.Skipped++
			continue
		}

		price, _ := strconv.ParseInt(rec.Price, 10, 64)
		soldAt, _ := parseMikroTikDateTime(rec.Date, rec.Time)
		if soldAt.IsZero() {
			soldAt = now
		}

		sale := &models.VoucherSale{
			RouterID:       routerID,
			SoldAt:         soldAt,
			Username:       rec.Username,
			ProfileName:    rec.Profile,
			Price:          price,
			Server:         rec.Source,
			IPAddress:      rec.IPAddress,
			MACAddress:     rec.MACAddress,
			Validity:       rec.Validity,
			IdempotencyKey: key,
		}
		newSales = append(newSales, sale)
	}

	if len(newSales) > 0 {
		if err := s.saleRepo.CreateBatch(ctx, newSales); err != nil {
			return nil, fmt.Errorf("save imported sales: %w", err)
		}
		result.Imported = len(newSales)
	}

	s.invalidateSalesCache(ctx, routerID)

	s.logger.Info("sales import complete",
		"router", router.SessionName,
		"total", result.Total,
		"imported", result.Imported,
		"skipped", result.Skipped,
		"errors", result.Errors,
	)

	return result, nil
}

func (s *VoucherService) invalidateSalesCache(ctx context.Context, routerID uint) {
	if s.cache == nil {
		return
	}
	_ = s.cache.Invalidate(ctx,
		appcache.SalesKey(routerID, "today"),
		appcache.SalesKey(routerID, "month"),
		appcache.DashboardKey(routerID),
	)
}

func (s *VoucherService) GetRouterInfo(ctx context.Context, routerID uint) (*models.Router, error) {
	return s.routerRepo.GetByID(ctx, routerID)
}

func idempotencyKey(routerID uint, username string, soldAt time.Time) string {
	input := fmt.Sprintf("%d|%s|%s", routerID, username, soldAt.Format("2006-01-02 15:04:05"))
	return sha256Sum(input)
}

func sha256Sum(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)
}

func parseMikroTikDateTime(date, timeStr string) (time.Time, error) {
	if date == "" || timeStr == "" {
		return time.Time{}, fmt.Errorf("empty date or time")
	}

	parts := strings.SplitN(timeStr, ":", 3)
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("invalid time format: %s", timeStr)
	}
	hour, _ := strconv.Atoi(parts[0])
	minute, _ := strconv.Atoi(parts[1])
	second, _ := strconv.Atoi(parts[2])

	dateParts := strings.SplitN(date, "/", 3)
	if len(dateParts) != 3 {
		return time.Time{}, fmt.Errorf("invalid date format: %s", date)
	}

	months := map[string]time.Month{
		"jan": time.January, "feb": time.February, "mar": time.March,
		"apr": time.April, "may": time.May, "jun": time.June,
		"jul": time.July, "aug": time.August, "sep": time.September,
		"oct": time.October, "nov": time.November, "dec": time.December,
	}
	mon := months[strings.ToLower(dateParts[0])]
	day, _ := strconv.Atoi(dateParts[1])
	year, _ := strconv.Atoi(dateParts[2])

	return time.Date(year, mon, day, hour, minute, second, 0, time.Local), nil
}

func ParseMikroTikDateTime(date, timeStr string) (time.Time, error) {
	return parseMikroTikDateTime(date, timeStr)
}
