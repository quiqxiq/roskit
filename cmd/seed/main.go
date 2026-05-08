package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	casbinx "github.com/quiqxiq/roskit/internal/casbin"
	"github.com/quiqxiq/roskit/internal/config"
	"github.com/quiqxiq/roskit/internal/models"
	"github.com/quiqxiq/roskit/pkg/database"
	"github.com/quiqxiq/roskit/pkg/encrypt"
)

// ─── Seed data ────────────────────────────────────────────────────────────────

var seedTenants = []struct {
	Name        string
	Slug        string
	HotspotName string
	DNSName     string
	Currency    string
	Plan        models.TenantPlan
	Routers     []seedRouter
	Users       []seedUser
}{
	{
		Name:        "Tenant Alpha",
		Slug:        "alpha",
		HotspotName: "Alpha Hotspot",
		DNSName:     "alpha.roskit.local",
		Currency:    "Rp",
		Plan:        models.TenantPlanStarter,
		Routers: []seedRouter{
			{Name: "router-alpha", IPAddress: "192.168.233.1", APIPort: 8728, APIUsername: "admin", Password: "r00t"},
		},
		Users: []seedUser{
			{Username: "owner.alpha", Password: "ownerpass", Role: models.UserRoleOwner},
			{Username: "admin.alpha", Password: "adminpass", Role: models.UserRoleAdmin},
			{Username: "staff.alpha", Password: "staffpass", Role: models.UserRoleStaff},
		},
	},
	{
		Name:        "Tenant Beta",
		Slug:        "beta",
		HotspotName: "Beta Hotspot",
		DNSName:     "beta.roskit.local",
		Currency:    "Rp",
		Plan:        models.TenantPlanFree,
		Routers: []seedRouter{
			{Name: "router-beta", IPAddress: "192.168.230.2", APIPort: 8728, APIUsername: "admin", Password: "r00t"},
		},
		Users: []seedUser{
			{Username: "owner.beta", Password: "ownerpass", Role: models.UserRoleOwner},
			{Username: "admin.beta", Password: "adminpass", Role: models.UserRoleAdmin},
			{Username: "staff.beta", Password: "staffpass", Role: models.UserRoleStaff},
		},
	},
}

// Superadmin lives in the platform pseudo-tenant (tenantID = nil).
var seedSuperAdmin = seedUser{
	Username: "superadmin",
	Password: "superadminpass",
	Role:     models.UserRoleSuperAdmin,
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

// ─── Main ─────────────────────────────────────────────────────────────────────

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Connect(cfg.PostgresDSN())
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	enforcer, err := casbinx.NewEnforcer(db)
	if err != nil {
		log.Fatalf("casbin enforcer: %v", err)
	}
	if err := casbinx.SeedPolicies(enforcer); err != nil {
		log.Fatalf("seed casbin policies: %v", err)
	}
	fmt.Println("✔ Casbin policies seeded")

	ctx := context.Background()

	// ── 1. Superadmin (platform scope, no tenant) ─────────────────────────────
	fmt.Println("\n── Platform superadmin ──────────────────────────────────────")
	sa, created, err := ensureUser(ctx, db, nil, seedSuperAdmin)
	if err != nil {
		log.Fatalf("superadmin: %v", err)
	}
	printUser(sa, created)
	if err := casbinx.AssignRole(enforcer, sa.ID, models.PlatformTenantSlug, models.UserRoleSuperAdmin); err != nil {
		log.Printf("  WARN assign casbin superadmin: %v", err)
	}

	// ── 2. Tenants ────────────────────────────────────────────────────────────
	for _, st := range seedTenants {
		fmt.Printf("\n── Tenant: %s (%s) ─────────────────────────────────────────\n", st.Name, st.Slug)

		tenant, err := ensureTenant(ctx, db, st.Slug, st.Name, st.Plan, st.HotspotName, st.DNSName, st.Currency)
		if err != nil {
			log.Fatalf("tenant %s: %v", st.Slug, err)
		}
		fmt.Printf("  tenant id=%d  slug=%s\n", tenant.ID, tenant.Slug)

		// Routers
		fmt.Println("  [routers]")
		for _, sr := range st.Routers {
			r, created, err := ensureRouter(ctx, db, tenant.ID, sr, cfg.AESEncKey)
			if err != nil {
				log.Printf("  WARN router %s: %v", sr.Name, err)
				continue
			}
			printRouter(r, created)
		}

		// Users + Casbin
		fmt.Println("  [users]")
		for _, su := range st.Users {
			u, created, err := ensureUser(ctx, db, &tenant.ID, su)
			if err != nil {
				log.Printf("  WARN user %s: %v", su.Username, err)
				continue
			}
			printUser(u, created)
			if err := casbinx.AssignRole(enforcer, u.ID, tenant.Slug, su.Role); err != nil {
				log.Printf("  WARN assign casbin role %s -> %s: %v", su.Username, su.Role, err)
			}
		}
	}

	fmt.Println("\n════════════════════════════════════════")
	fmt.Println("  Seed complete ✔")
	fmt.Println("════════════════════════════════════════")
	fmt.Println()
	fmt.Println("  Default credentials (CHANGE in production):")
	fmt.Printf("  %-16s  %-12s  %s\n", "Username", "Password", "Role")
	fmt.Printf("  %-16s  %-12s  %s\n", "────────────────", "────────────", "──────────")
	fmt.Printf("  %-16s  %-12s  %s\n", "superadmin", "superadminpass", "superadmin")
	for _, st := range seedTenants {
		for _, u := range st.Users {
			fmt.Printf("  %-16s  %-12s  %s (%s)\n", u.Username, u.Password, u.Role, st.Slug)
		}
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func ensureTenant(ctx context.Context, db *gorm.DB, slug, name string, plan models.TenantPlan, hotspotName, dnsName, currency string) (*models.Tenant, error) {
	var existing models.Tenant
	err := db.WithContext(ctx).Where("slug = ?", slug).First(&existing).Error
	if err == nil {
		return &existing, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	tenant := &models.Tenant{
		Name:   name,
		Slug:   slug,
		Plan:   plan,
		Status: models.TenantStatusActive,
	}

	txErr := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(tenant).Error; err != nil {
			return fmt.Errorf("create tenant: %w", err)
		}
		settings := &models.TenantSettings{
			TenantID:     tenant.ID,
			HotspotName:  hotspotName,
			DNSName:      dnsName,
			Currency:     currency,
			IdleTimeout:  30,
			ReportMode:   "disable",
			WebhookToken: generateToken(),
		}
		return tx.Create(settings).Error
	})
	if txErr != nil {
		return nil, txErr
	}
	return tenant, nil
}

func ensureRouter(ctx context.Context, db *gorm.DB, tenantID uint, sr seedRouter, aesKey string) (*models.Router, bool, error) {
	var existing models.Router
	err := db.WithContext(ctx).
		Where("tenant_id = ? AND name = ?", tenantID, sr.Name).
		First(&existing).Error
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
		TenantID:             tenantID,
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

func ensureUser(ctx context.Context, db *gorm.DB, tenantID *uint, su seedUser) (*models.User, bool, error) {
	var existing models.User
	q := db.WithContext(ctx).Where("username = ?", su.Username)
	if tenantID != nil {
		q = q.Where("tenant_id = ?", *tenantID)
	} else {
		q = q.Where("tenant_id IS NULL")
	}
	err := q.First(&existing).Error
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
		TenantID:     tenantID,
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
	tenantStr := "platform"
	if u.TenantID != nil {
		tenantStr = fmt.Sprintf("tenant=%d", *u.TenantID)
	}
	fmt.Printf("    [%s] user %-20s  role=%-12s  %s (id=%d)\n",
		action, u.Username, u.Role, tenantStr, u.ID)
}

func printRouter(r *models.Router, created bool) {
	action := "skip"
	if created {
		action = "created"
	}
	fmt.Printf("    [%s] router %-20s  ip=%-18s  (id=%d)\n",
		action, r.Name, r.IPAddress, r.ID)
}

func generateToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
