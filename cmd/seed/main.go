package main

import (
	"context"
	"fmt"
	"log"

	"github.com/quiqxiq/roskit/internal/config"
	"github.com/quiqxiq/roskit/internal/models"
	"github.com/quiqxiq/roskit/pkg/database"
	"github.com/quiqxiq/roskit/pkg/encrypt"
	"gorm.io/gorm"
)

var sampleTenant = struct {
	Name        string
	Slug        string
	HotspotName string
	DNSName     string
	Currency    string
}{
	Name:        "Default Tenant",
	Slug:        "default",
	HotspotName: "Roskit Hotspot",
	DNSName:     "roskit.local",
	Currency:    "Rp",
}

var sampleRouters = []struct {
	Name        string
	IPAddress   string
	APIPort     int
	APIUsername string
	Password    string
}{
	{
		Name:        "router-alpha",
		IPAddress:   "192.168.233.1",
		APIPort:     8728,
		APIUsername: "admin",
		Password:    "",
	},
	{
		Name:        "router-beta",
		IPAddress:   "192.168.230.2",
		APIPort:     8728,
		APIUsername: "admin",
		Password:    "",
	},
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := database.Connect(cfg.PostgresDSN())
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	ctx := context.Background()

	tenant, err := ensureTenant(ctx, db)
	if err != nil {
		log.Fatalf("failed to ensure tenant: %v", err)
	}
	fmt.Printf("  tenant: %s (id=%d, slug=%s)\n", tenant.Name, tenant.ID, tenant.Slug)

	seeded := 0
	for _, sr := range sampleRouters {
		var existing models.Router
		err := db.WithContext(ctx).
			Where("tenant_id = ? AND name = ?", tenant.ID, sr.Name).
			First(&existing).Error

		if err == nil {
			fmt.Printf("  skip: %s (already exists, id=%d)\n", sr.Name, existing.ID)
			continue
		}
		if err != gorm.ErrRecordNotFound {
			log.Printf("  error checking %s: %v", sr.Name, err)
			continue
		}

		encPass, err := encrypt.Encrypt(sr.Password, cfg.AESEncKey)
		if err != nil {
			log.Printf("  error encrypting password for %s: %v", sr.Name, err)
			continue
		}

		port := sr.APIPort
		if port == 0 {
			port = 8728
		}

		router := &models.Router{
			TenantID:             tenant.ID,
			Name:                 sr.Name,
			IPAddress:            sr.IPAddress,
			APIPort:              port,
			APIUsername:          sr.APIUsername,
			APIPasswordEncrypted: encPass,
			Status:               models.RouterStatusUnknown,
		}
		if err := db.WithContext(ctx).Create(router).Error; err != nil {
			log.Printf("  error creating %s: %v", sr.Name, err)
			continue
		}

		seeded++
		fmt.Printf("  created: %s (ip=%s)\n", sr.Name, sr.IPAddress)
	}

	fmt.Println("  ═══════ Seed Summary ═══════")
	fmt.Printf("  Tenant: %s\n", tenant.Slug)
	fmt.Printf("  Total sample routers: %d\n", len(sampleRouters))
	fmt.Printf("  Seeded: %d\n", seeded)
	fmt.Printf("  Skipped: %d\n", len(sampleRouters)-seeded)
	fmt.Println("  ════════════════════════════")
}

func ensureTenant(ctx context.Context, db *gorm.DB) (*models.Tenant, error) {
	var existing models.Tenant
	err := db.WithContext(ctx).Where("slug = ?", sampleTenant.Slug).First(&existing).Error
	if err == nil {
		return &existing, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	tenant := &models.Tenant{
		Name:   sampleTenant.Name,
		Slug:   sampleTenant.Slug,
		Plan:   models.TenantPlanFree,
		Status: models.TenantStatusActive,
	}

	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(tenant).Error; err != nil {
			return err
		}
		settings := &models.TenantSettings{
			TenantID:    tenant.ID,
			HotspotName: sampleTenant.HotspotName,
			DNSName:     sampleTenant.DNSName,
			Currency:    sampleTenant.Currency,
			IdleTimeout: 30,
			ReportMode:  "disable",
		}
		return tx.Create(settings).Error
	})
	if err != nil {
		return nil, err
	}
	return tenant, nil
}
