package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	casbinx "github.com/quiqxiq/roskit/internal/casbin"
	"github.com/quiqxiq/roskit/internal/models"
	appcache "github.com/quiqxiq/roskit/pkg/redis"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserInactive       = errors.New("user account is inactive")
	ErrUserAlreadyExists  = errors.New("username already exists")
	ErrInvalidRole        = errors.New("invalid role")
	ErrOldPasswordWrong   = errors.New("old password is incorrect")
	ErrSetupComplete      = errors.New("initial setup already completed")
	ErrTenantNotFound     = errors.New("tenant not found")
	ErrTenantSuspended    = errors.New("tenant is suspended")
	ErrTenantRequired     = errors.New("tenant is required for non-superadmin login")
)

type LoginResult struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	ExpiresIn    int      `json:"expires_in"`
	User         UserView `json:"user"`
}

type UserView struct {
	ID         uint            `json:"id"`
	Username   string          `json:"username"`
	Role       models.UserRole `json:"role"`
	TenantID   *uint           `json:"tenant_id"`
	TenantSlug string          `json:"tenant_slug"`
}

type Claims struct {
	UserID     uint            `json:"uid"`
	Username   string          `json:"sub"`
	Role       models.UserRole `json:"role"`
	TenantID   *uint           `json:"tid"`
	TenantSlug string          `json:"tslug"`
	TokenID    string          `json:"jti"`
	jwt.RegisteredClaims
}

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

type TenantLookup interface {
	GetBySlug(ctx context.Context, slug string) (*models.Tenant, error)
}

type AuthService struct {
	userRepo      UserRepository
	tenantRepo    TenantLookup
	enforcer      *casbin.Enforcer
	cache         *appcache.Cache
	jwtSecret     []byte
	refreshSecret []byte
}

func NewAuthService(userRepo UserRepository, tenantRepo TenantLookup, enforcer *casbin.Enforcer, cache *appcache.Cache, jwtSecret, refreshSecret []byte) *AuthService {
	return &AuthService{
		userRepo:      userRepo,
		tenantRepo:    tenantRepo,
		enforcer:      enforcer,
		cache:         cache,
		jwtSecret:     jwtSecret,
		refreshSecret: refreshSecret,
	}
}

func generateTokenID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Login authenticates a user.
// - tenantSlug == "" : treated as superadmin login (users.tenant_id IS NULL).
// - tenantSlug != "" : resolve tenant first, then scope user lookup to (tenant_id, username).
func (s *AuthService) Login(ctx context.Context, tenantSlug, username, password string) (*LoginResult, error) {
	tenantSlug = strings.TrimSpace(tenantSlug)

	var (
		user       *models.User
		tenant     *models.Tenant
		finalSlug  = models.PlatformTenantSlug
		finalTenID *uint
	)

	if tenantSlug == "" {
		u, err := s.userRepo.GetSuperAdminByUsername(ctx, username)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidCredentials, err)
		}
		user = u
	} else {
		if s.tenantRepo == nil {
			return nil, ErrTenantRequired
		}
		t, err := s.tenantRepo.GetBySlug(ctx, tenantSlug)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrTenantNotFound, err)
		}
		if t.Status == models.TenantStatusSuspended {
			return nil, ErrTenantSuspended
		}
		tenant = t
		finalSlug = t.Slug
		finalTenID = &t.ID

		u, err := s.userRepo.GetByTenantUsername(ctx, &t.ID, username)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidCredentials, err)
		}
		user = u
	}

	if !user.Active {
		return nil, ErrUserInactive
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return s.issueTokens(ctx, user, tenant, finalSlug, finalTenID)
}

func (s *AuthService) issueTokens(ctx context.Context, user *models.User, _ *models.Tenant, tenantSlug string, tenantID *uint) (*LoginResult, error) {
	tokenID, err := generateTokenID()
	if err != nil {
		return nil, fmt.Errorf("generate token id: %w", err)
	}
	refreshTokenID, err := generateTokenID()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token id: %w", err)
	}

	now := time.Now()
	accessExpiry := now.Add(15 * time.Minute)
	refreshExpiry := now.Add(7 * 24 * time.Hour)

	accessClaims := Claims{
		UserID:     user.ID,
		Username:   user.Username,
		Role:       user.Role,
		TenantID:   tenantID,
		TenantSlug: tenantSlug,
		TokenID:    tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
			Issuer:    "roskit-api",
		},
	}
	accessStr, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	refreshClaims := Claims{
		UserID:     user.ID,
		TenantID:   tenantID,
		TenantSlug: tenantSlug,
		TokenID:    refreshTokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(refreshExpiry),
			Issuer:    "roskit-api",
		},
	}
	refreshStr, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(s.refreshSecret)
	if err != nil {
		return nil, fmt.Errorf("sign refresh token: %w", err)
	}

	if s.cache != nil {
		_ = s.cache.Set(ctx, appcache.AuthRefreshKey(fmt.Sprintf("%d", user.ID), refreshTokenID), "1", 7*24*time.Hour)
	}
	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		return nil, fmt.Errorf("update last login: %w", err)
	}

	return &LoginResult{
		AccessToken:  accessStr,
		RefreshToken: refreshStr,
		ExpiresIn:    int(15 * time.Minute.Seconds()),
		User: UserView{
			ID:         user.ID,
			Username:   user.Username,
			Role:       user.Role,
			TenantID:   tenantID,
			TenantSlug: tenantSlug,
		},
	}, nil
}

func (s *AuthService) ValidateToken(ctx context.Context, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	if s.cache != nil {
		_, found, _ := s.cache.Get(ctx, appcache.AuthRevokedKey(claims.TokenID))
		if found {
			return nil, fmt.Errorf("token revoked")
		}
	}

	return claims, nil
}

func (s *AuthService) ValidateRefreshToken(ctx context.Context, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.refreshSecret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse refresh token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid refresh token claims")
	}
	return claims, nil
}

func (s *AuthService) Logout(ctx context.Context, accessTokenID, userID, refreshTokenID string, accessTokenExpiry time.Time) error {
	if s.cache == nil {
		return nil
	}
	remaining := time.Until(accessTokenExpiry)
	if remaining > 0 {
		_ = s.cache.Set(ctx, appcache.AuthRevokedKey(accessTokenID), "1", remaining)
	}
	_ = s.cache.Delete(ctx, appcache.AuthRefreshKey(userID, refreshTokenID))
	return nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshTokenStr string) (*LoginResult, error) {
	claims, err := s.ValidateRefreshToken(ctx, refreshTokenStr)
	if err != nil {
		return nil, err
	}

	refreshKey := appcache.AuthRefreshKey(fmt.Sprintf("%d", claims.UserID), claims.TokenID)
	if s.cache != nil {
		_, found, _ := s.cache.Get(ctx, refreshKey)
		if !found {
			return nil, fmt.Errorf("refresh token not found or expired")
		}
		_ = s.cache.Delete(ctx, refreshKey)
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	tenantSlug := claims.TenantSlug
	if tenantSlug == "" {
		tenantSlug = models.PlatformTenantSlug
	}
	return s.issueTokens(ctx, user, nil, tenantSlug, claims.TenantID)
}

// CreateUser creates a user within a tenant (tenantID != nil) or a superadmin (tenantID == nil).
// When enforcer is configured, the corresponding Casbin role grant is inserted.
func (s *AuthService) CreateUser(ctx context.Context, tenantID *uint, tenantSlug, username, password string, role models.UserRole) (*models.User, error) {
	if !role.Valid() {
		return nil, ErrInvalidRole
	}
	if role == models.UserRoleSuperAdmin && tenantID != nil {
		return nil, fmt.Errorf("superadmin must not be bound to a tenant")
	}
	if role != models.UserRoleSuperAdmin && tenantID == nil {
		return nil, fmt.Errorf("non-superadmin role requires tenant")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &models.User{
		TenantID:     tenantID,
		Username:     username,
		PasswordHash: string(hash),
		Role:         role,
		Active:       true,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return nil, ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	if s.enforcer != nil {
		slug := tenantSlug
		if role == models.UserRoleSuperAdmin {
			slug = models.PlatformTenantSlug
		}
		if err := casbinx.AssignRole(s.enforcer, user.ID, slug, role); err != nil {
			return nil, fmt.Errorf("assign casbin role: %w", err)
		}
	}

	return user, nil
}

func (s *AuthService) ChangePassword(ctx context.Context, userID uint, oldPass, newPass string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPass)); err != nil {
		return ErrOldPasswordWrong
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPass), 12)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	user.PasswordHash = string(hash)
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return nil
}

func (s *AuthService) UserCount(ctx context.Context) (int64, error) {
	return s.userRepo.Count(ctx)
}

// Enforcer returns the bound Casbin enforcer (may be nil).
func (s *AuthService) Enforcer() *casbin.Enforcer { return s.enforcer }
