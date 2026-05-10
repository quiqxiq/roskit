package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/quiqxiq/roskit/internal/config"
	"github.com/quiqxiq/roskit/internal/models"
	"github.com/quiqxiq/roskit/pkg/database"
	"github.com/quiqxiq/roskit/pkg/encrypt"
)

var seedRouters = []seedRouter{
	{Name: "router-main", IPAddress: "192.168.88.1", APIPort: 8728, APIUsername: "admin", Password: ""},
}

var seedUsers = []seedUser{
	{Username: "admin", Password: "adminpass", Role: models.UserRoleAdmin},
	{Username: "staff", Password: "staffpass", Role: models.UserRoleStaff},
}

type seedRouter struct {
	Name        string
	IPAddress   string
	APIPort     int
	APIUsername string
	Password    string
}

type seedUser struct {
	Username string
	Password string
	Role     models.UserRole
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Connect(cfg.PostgresDSN())
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	ctx := context.Background()

	// ── 1. Settings singleton ─────────────────────────────────────────────────
	fmt.Println("\n── Settings ─────────────────────────────────────────────────────")
	settings, settingsCreated, err := ensureSettings(ctx, db)
	if err != nil {
		log.Fatalf("settings: %v", err)
	}
	action := "skip"
	if settingsCreated {
		action = "created"
	}
	fmt.Printf("  [%s] settings id=%d  webhook_token=%s\n", action, settings.ID, settings.WebhookToken)

	// ── 2. Routers ────────────────────────────────────────────────────────────
	fmt.Println("\n── Routers ─────────────────────────────────────────────────────")
	for _, sr := range seedRouters {
		r, created, err := ensureRouter(ctx, db, sr, cfg.AESEncKey)
		if err != nil {
			log.Printf("  WARN router %s: %v", sr.Name, err)
			continue
		}
		printRouter(r, created)
	}

	// ── 3. Users ─────────────────────────────────────────────────────────────
	fmt.Println("\n── Users ───────────────────────────────────────────────────────")
	for _, su := range seedUsers {
		u, created, err := ensureUser(ctx, db, su)
		if err != nil {
			log.Printf("  WARN user %s: %v", su.Username, err)
			continue
		}
		printUser(u, created)
	}

	fmt.Println("\n════════════════════════════════════════")
	fmt.Println("  Seed complete ✔")
	fmt.Println("════════════════════════════════════════")
	fmt.Println()
	fmt.Println("  Default credentials (CHANGE in production):")
	fmt.Printf("  %-16s  %-12s  %s\n", "Username", "Password", "Role")
	fmt.Printf("  %-16s  %-12s  %s\n", "────────────────", "────────────", "──────────")
	for _, u := range seedUsers {
		fmt.Printf("  %-16s  %-12s  %s\n", u.Username, u.Password, u.Role)
	}
}

func ensureSettings(ctx context.Context, db *gorm.DB) (*models.Settings, bool, error) {
	var existing models.Settings
	err := db.WithContext(ctx).First(&existing, 1).Error
	if err == nil {
		return &existing, false, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, false, err
	}

	s := &models.Settings{
		HotspotName:  "My Hotspot",
		DNSName:      "hotspot.local",
		Currency:     "Rp",
		IdleTimeout:  30,
		ReportMode:   "disable",
		WebhookToken: generateToken(),
		Timezone:     "Asia/Jakarta",
	}
	if err := db.WithContext(ctx).Create(s).Error; err != nil {
		return nil, false, err
	}
	return s, true, nil
}

func ensureRouter(ctx context.Context, db *gorm.DB, sr seedRouter, aesKey string) (*models.Router, bool, error) {
	var existing models.Router
	err := db.WithContext(ctx).Where("name = ?", sr.Name).First(&existing).Error
	if err == nil {
		return &existing, false, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, false, err
	}

	encPass, err := encrypt.Encrypt(sr.Password, aesKey)
	if err != nil {
		return nil, false, fmt.Errorf("encrypt password: %w", err)
	}

	port := sr.APIPort
	if port == 0 {
		port = 8728
	}

	router := &models.Router{
		Name:                 sr.Name,
		IPAddress:            sr.IPAddress,
		APIPort:              port,
		APIUsername:          sr.APIUsername,
		APIPasswordEncrypted: encPass,
		Status:               models.RouterStatusUnknown,
	}
	if err := db.WithContext(ctx).Create(router).Error; err != nil {
		return nil, false, err
	}
	return router, true, nil
}

func ensureUser(ctx context.Context, db *gorm.DB, su seedUser) (*models.User, bool, error) {
	var existing models.User
	err := db.WithContext(ctx).Where("username = ?", su.Username).First(&existing).Error
	if err == nil {
		return &existing, false, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, false, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(su.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, false, fmt.Errorf("hash password: %w", err)
	}

	user := &models.User{
		Username:     su.Username,
		PasswordHash: string(hash),
		Role:         su.Role,
		Active:       true,
	}
	if err := db.WithContext(ctx).Create(user).Error; err != nil {
		return nil, false, err
	}
	return user, true, nil
}

func printUser(u *models.User, created bool) {
	action := "skip"
	if created {
		action = "created"
	}
	fmt.Printf("    [%s] user %-20s  role=%-12s  (id=%d)\n", action, u.Username, u.Role, u.ID)
}

func printRouter(r *models.Router, created bool) {
	action := "skip"
	if created {
		action = "created"
	}
	fmt.Printf("    [%s] router %-20s  ip=%-18s  (id=%d)\n", action, r.Name, r.IPAddress, r.ID)
}

func generateToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
