package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

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
)

type LoginResult struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	ExpiresIn    int      `json:"expires_in"`
	User         UserView `json:"user"`
}

type UserView struct {
	ID       uint            `json:"id"`
	Username string          `json:"username"`
	Role     models.UserRole `json:"role"`
}

type Claims struct {
	UserID   uint            `json:"uid"`
	Username string          `json:"sub"`
	Role     models.UserRole `json:"role"`
	TokenID  string          `json:"jti"`
	jwt.RegisteredClaims
}

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id uint) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	List(ctx context.Context) ([]*models.User, error)
	Update(ctx context.Context, user *models.User) error
	UpdateLastLogin(ctx context.Context, userID uint) error
	Delete(ctx context.Context, id uint) error
	Count(ctx context.Context) (int64, error)
}

type AuthService struct {
	userRepo      UserRepository
	cache         *appcache.Cache
	jwtSecret     []byte
	refreshSecret []byte
}

func NewAuthService(userRepo UserRepository, cache *appcache.Cache, jwtSecret, refreshSecret []byte) *AuthService {
	return &AuthService{
		userRepo:      userRepo,
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

func (s *AuthService) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !user.Active {
		return nil, ErrUserInactive
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return s.issueTokens(ctx, user)
}

func (s *AuthService) issueTokens(ctx context.Context, user *models.User) (*LoginResult, error) {
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
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		TokenID:  tokenID,
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
		UserID:  user.ID,
		TokenID: refreshTokenID,
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
			ID:       user.ID,
			Username: user.Username,
			Role:     user.Role,
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

	return s.issueTokens(ctx, user)
}

func (s *AuthService) CreateUser(ctx context.Context, username, password string, role models.UserRole) (*models.User, error) {
	if !role.Valid() {
		return nil, ErrInvalidRole
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &models.User{
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

func (s *AuthService) AdminResetPassword(ctx context.Context, userID uint, newPass string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
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

func (s *AuthService) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

func (s *AuthService) ListUsers(ctx context.Context) ([]*models.User, error) {
	return s.userRepo.List(ctx)
}

func (s *AuthService) GetUser(ctx context.Context, id uint) (*models.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

func (s *AuthService) UpdateUser(ctx context.Context, id uint, username, password *string, role *models.UserRole, active *bool) (*models.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	if username != nil {
		user.Username = *username
	}
	if active != nil {
		user.Active = *active
	}
	if role != nil && *role != user.Role {
		if !role.Valid() {
			return nil, ErrInvalidRole
		}
		user.Role = *role
	}
	if password != nil && *password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(*password), 12)
		if err != nil {
			return nil, fmt.Errorf("hash password: %w", err)
		}
		user.PasswordHash = string(hash)
	}
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	return user, nil
}

func (s *AuthService) DeleteUser(ctx context.Context, id uint) error {
	return s.userRepo.Delete(ctx, id)
}
