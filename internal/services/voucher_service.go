package services

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/quiqxiq/roskit/internal/models"
	"github.com/quiqxiq/roskit/internal/repository"
	roskitservice "github.com/quiqxiq/roskit/internal/roskit/adapter/service"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
	appcache "github.com/quiqxiq/roskit/pkg/redis"
)

type VoucherGenerateParams struct {
	Qty        int    `json:"qty" binding:"required,min=1,max=1000"`
	Server     string `json:"server" binding:"omitempty,max=64"`
	UserType   string `json:"user_type" binding:"omitempty,max=32"`
	NameLength int    `json:"name_length" binding:"omitempty,min=3,max=32"`
	Prefix     string `json:"prefix" binding:"omitempty,max=16"`
	CharSet    string `json:"char_set" binding:"omitempty,max=64"`
	Profile    string `json:"profile" binding:"required,min=1,max=64"`
	TimeLimit  string `json:"time_limit" binding:"omitempty,max=64"`
	DataLimit  int64  `json:"data_limit" binding:"omitempty,min=0"`
	Comment    string `json:"comment" binding:"omitempty,max=200"`
	Gencode    string `json:"gencode" binding:"omitempty,max=64"`
}

type VoucherGenerateResult struct {
	Count    int                              `json:"count"`
	Gencode  string                           `json:"gencode"`
	Profile  string                           `json:"profile"`
	Vouchers []roskitservice.GeneratedVoucher `json:"vouchers"`
}

type RecordSaleParams struct {
	Username    string    `json:"username" binding:"required,min=1,max=64"`
	ProfileName string    `json:"profile_name" binding:"omitempty,max=64"`
	Price       int64     `json:"price" binding:"omitempty,min=0"`
	Validity    string    `json:"validity" binding:"omitempty,max=20"`
	Server      string    `json:"server" binding:"omitempty,max=64"`
	IPAddress   string    `json:"ip_address" binding:"omitempty,max=45"`
	MACAddress  string    `json:"mac_address" binding:"omitempty,max=17"`
	SoldAt      time.Time `json:"sold_at"`
}

type ImportResult struct {
	Total    int `json:"total"`
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
	Errors   int `json:"errors"`
}

type ProfilePriceMappingRepository interface {
	FindByRouterAndProfile(ctx context.Context, routerID uint, profileName string) (*models.ProfilePriceMapping, error)
}

type ResolvedVoucher struct {
	Username  string
	Password  string
	Profile   string
	Comment   string
	TimeLimit string
	DataLimit string
	UserMode  string
	Price     string
	Validity  string
}

type VoucherService struct {
	bridge      *roskitservice.Bridge
	saleRepo    SaleRepository
	routerRepo  RouterRepository
	profileRepo ProfilePriceMappingRepository
	settingsRepo repository.SettingsRepository
	cache       *appcache.Cache
	logger      *slog.Logger
}

func NewVoucherService(
	bridge *roskitservice.Bridge,
	saleRepo SaleRepository,
	routerRepo RouterRepository,
	profileRepo ProfilePriceMappingRepository,
	settingsRepo repository.SettingsRepository,
	cache *appcache.Cache,
) *VoucherService {
	return &VoucherService{
		bridge:      bridge,
		saleRepo:    saleRepo,
		routerRepo:  routerRepo,
		profileRepo: profileRepo,
		settingsRepo: settingsRepo,
		cache:       cache,
		logger:      slog.Default().With("component", "voucher-svc"),
	}
}

func (s *VoucherService) resolveProfilePrice(ctx context.Context, routerID uint, profileName string) (price, sellingPrice int64, validity string) {
	if profileName != "" {
		if m, err := s.profileRepo.FindByRouterAndProfile(ctx, routerID, profileName); err == nil && m != nil {
			return m.Price, m.SellingPrice, m.Validity
		}
	}
	if profileName == "" {
		return 0, 0, ""
	}
	profiles, err := s.bridge.Query(ctx, fmt.Sprintf("%d", routerID), "ip/hotspot/user/profile/print", "?name="+profileName)
	if err != nil || len(profiles) == 0 {
		return 0, 0, ""
	}
	meta := roskitservice.ParseOnLoginPut(profiles[0]["on-login"])
	if meta == nil {
		return 0, 0, ""
	}
	p, _ := strconv.ParseInt(meta.Price, 10, 64)
	sp, _ := strconv.ParseInt(meta.SellingPrice, 10, 64)
	return p, sp, meta.Validity
}

func (s *VoucherService) ResolveVoucherPrintData(
	ctx context.Context,
	routerID uint,
	allUsers []map[string]string,
	usernames []string,
	comment string,
) ([]ResolvedVoucher, error) {
	wanted := make(map[string]bool, len(usernames))
	for _, u := range usernames {
		wanted[u] = true
	}

	var result []ResolvedVoucher
	for _, u := range allUsers {
		name := u["name"]
		uComment := u["comment"]

		if len(wanted) > 0 {
			if !wanted[name] {
				continue
			}
		} else if !strings.HasPrefix(uComment, comment) {
			continue
		}

		userMode := "vc"
		if strings.HasPrefix(uComment, "up-") {
			userMode = "up"
		}

		_, sp, validity := s.resolveProfilePrice(ctx, routerID, u["profile"])

		result = append(result, ResolvedVoucher{
			Username:  name,
			Password:  u["password"],
			Profile:   u["profile"],
			Comment:   uComment,
			TimeLimit: u["limit-uptime"],
			DataLimit: u["limit-bytes-total"],
			UserMode:  userMode,
			Price:     fmt.Sprintf("%d", sp),
			Validity:  validity,
		})
	}
	return result, nil
}

func (s *VoucherService) GenerateVoucher(ctx context.Context, routerID uint, params VoucherGenerateParams) (*VoucherGenerateResult, error) {
	if _, err := s.routerRepo.GetByID(ctx, routerID); err != nil {
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
	if params.Gencode == "" {
		params.Gencode = fmt.Sprintf("%d", time.Now().UnixMilli())
	}

	comment := roskitservice.GenerateVoucherComment(params.UserType, params.Gencode, time.Now().Format("01.02.06"), params.Comment)

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

	result := &VoucherGenerateResult{
		Count:    len(vouchers),
		Gencode:  params.Gencode,
		Profile:  params.Profile,
		Vouchers: vouchers,
	}

	if s.cache != nil {
		key := appcache.VoucherSessionKey(routerID, params.Gencode)
		if err := s.cache.SetJSON(ctx, key, result, appcache.TTLVoucherSession); err != nil {
			s.logger.Warn("failed to cache voucher session", "error", err)
		}
	}

	return result, nil
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
	key := mikrotik.MakeSaleIdempotencyKey(routerID, params.Username, params.SoldAt)
	exists, err := s.saleRepo.ExistsByIdempotencyKey(ctx, key)
	if err != nil {
		return fmt.Errorf("check idempotency: %w", err)
	}
	if exists {
		return nil
	}

	rid := routerID
	sale := &models.VoucherSale{
		RouterID:       &rid,
		SoldAt:         params.SoldAt,
		Username:       params.Username,
		ProfileName:    params.ProfileName,
		Price:          params.Price,
		Validity:       params.Validity,
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

	tz := ""
	if s.settingsRepo != nil {
		if settings, err := s.settingsRepo.Get(ctx); err == nil && settings != nil {
			tz = settings.Timezone
		}
	}
	loc := mikrotik.ResolveLocation(tz)

	records, err := s.bridge.ImportSalesFromRouterOS(ctx, fmt.Sprintf("%d", routerID), "")
	if err != nil {
		return nil, fmt.Errorf("fetch sales from router: %w", err)
	}

	result := &ImportResult{}
	result.Total = len(records)
	now := time.Now()

	var newSales []*models.VoucherSale
	for _, rec := range records {
		soldAt, _ := mikrotik.Parse(rec.Date, rec.Time, loc)
		if soldAt.IsZero() {
			soldAt = now
		}

		key := mikrotik.MakeSaleIdempotencyKey(routerID, rec.Username, soldAt)

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

		rid := routerID
		sale := &models.VoucherSale{
			RouterID:       &rid,
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
		"router", router.Name,
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
