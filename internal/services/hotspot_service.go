package services

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/quiqxiq/roskit/internal/config"
	"github.com/quiqxiq/roskit/internal/repository"
	roskitservice "github.com/quiqxiq/roskit/internal/roskit/adapter/service"
	appcache "github.com/quiqxiq/roskit/pkg/redis"
)

type ProfileParams struct {
	Name         string `json:"name"`
	RateLimit    string `json:"rate_limit"`
	AddressPool  string `json:"address_pool"`
	SharedUsers  string `json:"shared_users"`
	ParentQueue  string `json:"parent_queue"`
	Price        int64  `json:"price"`
	SellingPrice int64  `json:"selling_price"`
	Validity     string `json:"validity"`
	ExpireMode   string `json:"expire_mode"`
	LockUser     string `json:"lock_user"`
	LockServer   string `json:"lock_server"`
}

type ProfileWithMeta struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	AddressPool       string `json:"address_pool"`
	RateLimit         string `json:"rate_limit"`
	SharedUsers       string `json:"shared_users"`
	StatusAutoRefresh string `json:"status_autorefresh"`
	OnLogin           string `json:"on_login"`
	OnLogout          string `json:"on_logout"`
	ParentQueue       string `json:"parent_queue"`
	ExpMode           string `json:"exp_mode"`
	Price             string `json:"price"`
	SellingPrice      string `json:"selling_price"`
	Validity          string `json:"validity"`
	NoExpiry          bool   `json:"no_expiry"`
	LockUser          string `json:"lock_user"`
	LockServer        string `json:"lock_server"`
}

func enrichProfile(data map[string]string, meta *roskitservice.OnLoginMetadata) ProfileWithMeta {
	p := ProfileWithMeta{
		ID:                data[".id"],
		Name:              data["name"],
		AddressPool:       data["address-pool"],
		RateLimit:         data["rate-limit"],
		SharedUsers:       data["shared-users"],
		StatusAutoRefresh: data["status-autorefresh"],
		OnLogin:           data["on-login"],
		OnLogout:          data["on-logout"],
		ParentQueue:       data["parent-queue"],
	}
	if meta != nil {
		p.ExpMode = meta.ExpMode
		p.Price = meta.Price
		p.SellingPrice = meta.SellingPrice
		p.Validity = meta.Validity
		p.NoExpiry = meta.NoExpiry
		p.LockUser = meta.LockUser
		p.LockServer = meta.LockServer
	}
	return p
}

type HotspotService struct {
	bridge       *roskitservice.Bridge
	cache        *appcache.Cache
	cfg          *config.Config
	settingsRepo repository.TenantSettingsRepository
	routerRepo   RouterRepository
	logger       *slog.Logger
}

func NewHotspotService(
	bridge *roskitservice.Bridge,
	cache *appcache.Cache,
	cfg *config.Config,
	settingsRepo repository.TenantSettingsRepository,
	routerRepo RouterRepository,
) *HotspotService {
	return &HotspotService{
		bridge:       bridge,
		cache:        cache,
		cfg:          cfg,
		settingsRepo: settingsRepo,
		routerRepo:   routerRepo,
		logger:       slog.Default().With("component", "hotspot-svc"),
	}
}

func routerIDStr(routerID uint) string {
	return fmt.Sprintf("%d", routerID)
}

func (s *HotspotService) ListUsers(ctx context.Context, routerID uint, profile string) ([]map[string]string, error) {
	if s.cache != nil && profile == "" {
		key := appcache.HotspotUsersKey(routerID)
		var cached []map[string]string
		found, _ := s.cache.GetJSON(ctx, key, &cached)
		if found {
			return cached, nil
		}
		users, err := s.bridge.ListHotspotUsers(ctx, routerIDStr(routerID), "")
		if err != nil {
			return nil, err
		}
		_ = s.cache.SetJSON(ctx, key, users, appcache.TTL30s)
		return users, nil
	}
	return s.bridge.ListHotspotUsers(ctx, routerIDStr(routerID), profile)
}

func (s *HotspotService) GetUser(ctx context.Context, routerID uint, idOrName string) (map[string]string, error) {
	return s.bridge.GetHotspotUser(ctx, routerIDStr(routerID), idOrName)
}

func (s *HotspotService) GetUserCount(ctx context.Context, routerID uint, profile string) (int, error) {
	if profile != "" {
		users, err := s.bridge.ListHotspotUsers(ctx, routerIDStr(routerID), profile)
		if err != nil {
			return 0, err
		}
		return len(users), nil
	}
	return s.bridge.GetHotspotUserCount(ctx, routerIDStr(routerID))
}

func (s *HotspotService) ListInactiveHotspotUsers(ctx context.Context, routerID string) ([]map[string]string, error) {
	return s.bridge.ListInactiveHotspotUsers(ctx, routerID)
}

func (s *HotspotService) GetInactiveHotspotUserCount(ctx context.Context, routerID string) (int, error) {
	return s.bridge.GetInactiveHotspotUserCount(ctx, routerID)
}

func (s *HotspotService) AddUser(ctx context.Context, routerID uint, params map[string]string) (map[string]string, error) {
	_, err := s.bridge.AddHotspotUser(ctx, routerIDStr(routerID), params)
	if err != nil {
		return nil, err
	}
	name := params["name"]
	if name == "" {
		return params, nil
	}
	return s.bridge.GetHotspotUser(ctx, routerIDStr(routerID), name)
}

func (s *HotspotService) UpdateUser(ctx context.Context, routerID uint, id string, params map[string]string) (map[string]string, error) {
	if err := s.bridge.SetHotspotUser(ctx, routerIDStr(routerID), id, params); err != nil {
		return nil, err
	}
	return s.bridge.GetHotspotUser(ctx, routerIDStr(routerID), id)
}

func (s *HotspotService) RemoveUser(ctx context.Context, routerID uint, id string) error {
	return s.bridge.RemoveHotspotUserWithCleanup(ctx, routerIDStr(routerID), id)
}

func (s *HotspotService) ListProfiles(ctx context.Context, routerID uint) ([]ProfileWithMeta, error) {
	profiles, err := s.bridge.ListHotspotProfiles(ctx, routerIDStr(routerID))
	if err != nil {
		return nil, err
	}
	result := make([]ProfileWithMeta, len(profiles))
	for i, p := range profiles {
		meta := roskitservice.ParseOnLoginPut(p["on-login"])
		result[i] = enrichProfile(p, meta)
	}
	return result, nil
}

func (s *HotspotService) GetProfile(ctx context.Context, routerID uint, idOrName string) (*ProfileWithMeta, error) {
	profile, err := s.bridge.GetHotspotProfile(ctx, routerIDStr(routerID), idOrName)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, fmt.Errorf("profile not found")
	}
	meta := roskitservice.ParseOnLoginPut(profile["on-login"])
	enriched := enrichProfile(profile, meta)
	return &enriched, nil
}

func (s *HotspotService) AddProfile(ctx context.Context, tenantID, routerID uint, params ProfileParams) (*ProfileWithMeta, error) {
	olParams := s.resolveOnLoginParams(ctx, tenantID, routerID, params)
	onLogin := roskitservice.GenerateOnLoginScript(olParams)

	rosParams := map[string]string{
		"name":     params.Name,
		"on-login": onLogin,
	}
	if params.RateLimit != "" {
		rosParams["rate-limit"] = params.RateLimit
	}
	if params.AddressPool != "" {
		rosParams["address-pool"] = params.AddressPool
	}
	if params.SharedUsers != "" {
		rosParams["shared-users"] = params.SharedUsers
	}
	if params.ParentQueue != "" {
		rosParams["parent-queue"] = params.ParentQueue
	}

	_, err := s.bridge.AddHotspotProfile(ctx, routerIDStr(routerID), rosParams)
	if err != nil {
		return nil, fmt.Errorf("add profile: %w", err)
	}

	profile, err := s.bridge.GetHotspotProfile(ctx, routerIDStr(routerID), params.Name)
	if err != nil {
		return nil, err
	}
	meta := roskitservice.ParseOnLoginPut(profile["on-login"])
	enriched := enrichProfile(profile, meta)
	return &enriched, nil
}

func (s *HotspotService) UpdateProfile(ctx context.Context, tenantID, routerID uint, id string, params ProfileParams) (*ProfileWithMeta, error) {
	olParams := s.resolveOnLoginParams(ctx, tenantID, routerID, params)
	onLogin := roskitservice.GenerateOnLoginScript(olParams)

	rosParams := map[string]string{
		"on-login": onLogin,
	}
	if params.Name != "" {
		rosParams["name"] = params.Name
	}
	if params.RateLimit != "" {
		rosParams["rate-limit"] = params.RateLimit
	}
	if params.AddressPool != "" {
		rosParams["address-pool"] = params.AddressPool
	}
	if params.SharedUsers != "" {
		rosParams["shared-users"] = params.SharedUsers
	}
	if params.ParentQueue != "" {
		rosParams["parent-queue"] = params.ParentQueue
	}

	if err := s.bridge.SetHotspotProfile(ctx, routerIDStr(routerID), id, rosParams); err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}

	profile, err := s.bridge.GetHotspotProfile(ctx, routerIDStr(routerID), id)
	if err != nil {
		return nil, err
	}
	meta := roskitservice.ParseOnLoginPut(profile["on-login"])
	enriched := enrichProfile(profile, meta)
	return &enriched, nil
}

func (s *HotspotService) resolveOnLoginParams(ctx context.Context, tenantID, routerID uint, params ProfileParams) roskitservice.OnLoginParams {
	olParams := roskitservice.OnLoginParams{
		ExpMode:      params.ExpireMode,
		Price:        strconv.FormatInt(params.Price, 10),
		SellingPrice: strconv.FormatInt(params.SellingPrice, 10),
		Validity:     params.Validity,
		ProfileName:  params.Name,
		LockUser:     params.LockUser,
		LockServer:   params.LockServer,
	}

	if s.cfg != nil {
		olParams.APIURL = s.cfg.PublicAPIURL
	}

	if s.settingsRepo != nil {
		if settings, err := s.settingsRepo.GetByTenantID(ctx, tenantID); err == nil && settings != nil {
			olParams.WebhookToken = settings.WebhookToken
		}
	}

	if s.routerRepo != nil {
		if router, err := s.routerRepo.GetByID(ctx, tenantID, routerID); err == nil && router != nil {
			olParams.RouterName = router.Name
		}
	}

	return olParams
}

func (s *HotspotService) RemoveProfile(ctx context.Context, routerID uint, id string) error {
	profile, err := s.bridge.GetHotspotProfile(ctx, routerIDStr(routerID), id)
	if err != nil {
		return err
	}
	profileName := profile["name"]
	return s.bridge.RemoveHotspotProfileWithCleanup(ctx, routerIDStr(routerID), id, profileName)
}

func (s *HotspotService) ListActive(ctx context.Context, routerID uint, server string) ([]map[string]string, error) {
	sessions, err := s.bridge.ListHotspotActive(ctx, routerIDStr(routerID))
	if err != nil {
		return nil, err
	}
	if server != "" {
		filtered := make([]map[string]string, 0)
		for _, sess := range sessions {
			if sess["server"] == server {
				filtered = append(filtered, sess)
			}
		}
		return filtered, nil
	}
	return sessions, nil
}

func (s *HotspotService) RemoveActive(ctx context.Context, routerID uint, sessionID string) error {
	return s.bridge.RemoveHotspotActive(ctx, routerIDStr(routerID), sessionID)
}

func (s *HotspotService) DisconnectUser(ctx context.Context, routerID uint, sessionID string) error {
	return s.bridge.DisconnectUser(ctx, routerIDStr(routerID), sessionID)
}

func (s *HotspotService) ListHosts(ctx context.Context, routerID uint) ([]map[string]string, error) {
	return s.bridge.ListHosts(ctx, routerIDStr(routerID))
}

func (s *HotspotService) RemoveHost(ctx context.Context, routerID uint, id string) error {
	return s.bridge.RemoveHost(ctx, routerIDStr(routerID), id)
}

func (s *HotspotService) ListServers(ctx context.Context, routerID uint) ([]map[string]string, error) {
	return s.bridge.ListHotspotServers(ctx, routerIDStr(routerID))
}

func (s *HotspotService) ListCookies(ctx context.Context, routerID uint) ([]map[string]string, error) {
	return s.bridge.ListCookies(ctx, routerIDStr(routerID))
}

func (s *HotspotService) RemoveCookie(ctx context.Context, routerID uint, id string) error {
	return s.bridge.RemoveCookie(ctx, routerIDStr(routerID), id)
}

func (s *HotspotService) ListIPBindings(ctx context.Context, routerID uint) ([]map[string]string, error) {
	return s.bridge.ListIPBindings(ctx, routerIDStr(routerID))
}

func (s *HotspotService) AddIPBinding(ctx context.Context, routerID uint, params map[string]string) (map[string]string, error) {
	newID, err := s.bridge.AddIPBinding(ctx, routerIDStr(routerID), params)
	if err != nil {
		return nil, err
	}
	if newID == "" {
		return nil, fmt.Errorf("failed to get new IP binding ID")
	}
	items, err := s.bridge.ListIPBindings(ctx, routerIDStr(routerID))
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item[".id"] == newID {
			return item, nil
		}
	}
	return map[string]string{".id": newID}, nil
}

func (s *HotspotService) UpdateIPBinding(ctx context.Context, routerID uint, id string, params map[string]string) error {
	return s.bridge.SetIPBinding(ctx, routerIDStr(routerID), id, params)
}

func (s *HotspotService) RemoveIPBinding(ctx context.Context, routerID uint, id string) error {
	items, err := s.bridge.ListIPBindings(ctx, routerIDStr(routerID))
	if err != nil {
		return err
	}
	var mac, addr string
	for _, item := range items {
		if item[".id"] == id {
			mac = item["mac-address"]
			addr = item["address"]
			break
		}
	}
	return s.bridge.RemoveIPBindingWithCleanup(ctx, routerIDStr(routerID), id, mac, addr)
}

func (s *HotspotService) EnableIPBinding(ctx context.Context, routerID uint, id string) error {
	return s.bridge.EnableIPBinding(ctx, routerIDStr(routerID), id)
}

func (s *HotspotService) DisableIPBinding(ctx context.Context, routerID uint, id string) error {
	return s.bridge.DisableIPBinding(ctx, routerIDStr(routerID), id)
}

func (s *HotspotService) ListWalledGarden(ctx context.Context, routerID uint) ([]map[string]string, error) {
	return s.bridge.ListWalledGarden(ctx, routerIDStr(routerID))
}

func (s *HotspotService) AddWalledGarden(ctx context.Context, routerID uint, params map[string]string) (map[string]string, error) {
	newID, err := s.bridge.AddWalledGarden(ctx, routerIDStr(routerID), params)
	if err != nil {
		return nil, err
	}
	items, err := s.bridge.ListWalledGarden(ctx, routerIDStr(routerID))
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item[".id"] == newID {
			return item, nil
		}
	}
	return map[string]string{".id": newID}, nil
}

func (s *HotspotService) RemoveWalledGarden(ctx context.Context, routerID uint, id string) error {
	return s.bridge.RemoveWalledGarden(ctx, routerIDStr(routerID), id)
}

func (s *HotspotService) ListWalledGardenIP(ctx context.Context, routerID uint) ([]map[string]string, error) {
	return s.bridge.ListWalledGardenIP(ctx, routerIDStr(routerID))
}

func (s *HotspotService) AddWalledGardenIP(ctx context.Context, routerID uint, params map[string]string) (map[string]string, error) {
	newID, err := s.bridge.AddWalledGardenIP(ctx, routerIDStr(routerID), params)
	if err != nil {
		return nil, err
	}
	items, err := s.bridge.ListWalledGardenIP(ctx, routerIDStr(routerID))
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item[".id"] == newID {
			return item, nil
		}
	}
	return map[string]string{".id": newID}, nil
}

func (s *HotspotService) RemoveWalledGardenIP(ctx context.Context, routerID uint, id string) error {
	return s.bridge.RemoveWalledGardenIP(ctx, routerIDStr(routerID), id)
}

func (s *HotspotService) ResetUserCounters(ctx context.Context, routerID uint, userID string) error {
	return s.bridge.ResetUserCounters(ctx, fmt.Sprintf("%d", routerID), userID)
}

func (s *HotspotService) ExportUsers(ctx context.Context, routerID uint, profile, format string) (string, error) {
	users, err := s.bridge.ListHotspotUsers(ctx, routerIDStr(routerID), profile)
	if err != nil {
		return "", err
	}

	switch format {
	case "script":
		var sb strings.Builder
		for _, u := range users {
			sb.WriteString(fmt.Sprintf("/ip hotspot user add name=%s password=%s profile=%s",
				u["name"], u["password"], u["profile"]))
			if mac, ok := u["mac-address"]; ok && mac != "" {
				sb.WriteString(fmt.Sprintf(" mac-address=%s", mac))
			}
			if server, ok := u["server"]; ok && server != "" {
				sb.WriteString(fmt.Sprintf(" server=%s", server))
			}
			if comment, ok := u["comment"]; ok && comment != "" {
				sb.WriteString(fmt.Sprintf(" comment=\"%s\"", comment))
			}
			sb.WriteString("\n")
		}
		return sb.String(), nil

	case "csv":
		var sb strings.Builder
		sb.WriteString("name,password,profile,mac-address,server,comment,disabled\n")
		for _, u := range users {
			sb.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s\n",
				u["name"], u["password"], u["profile"],
				u["mac-address"], u["server"], u["comment"], u["disabled"]))
		}
		return sb.String(), nil

	default:
		return "", fmt.Errorf("unsupported export format: %s (use 'script' or 'csv')", format)
	}
}
