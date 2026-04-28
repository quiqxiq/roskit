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

var sampleRouters = []struct {
	SessionName string
	IP          string
	Username    string
	Password    string
	HotspotName string
	DNSName     string
	Currency    string
}{
	{
		SessionName: "router-alpha",
		IP:          "192.168.233.1",
		Username:    "admin",
		Password:    "",
		HotspotName: "Alpha Hotspot",
		DNSName:     "alpha.local",
		Currency:    "Rp",
	},
	{
		SessionName: "router-beta",
		IP:          "192.168.230.2",
		Username:    "admin",
		Password:    "",
		HotspotName: "Beta Hotspot",
		DNSName:     "beta.local",
		Currency:    "Rp",
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

	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	ctx := context.Background()
	seeded := 0

	for _, sr := range sampleRouters {
		var existing models.Router
		err := db.WithContext(ctx).
			Where("session_name = ?", sr.SessionName).
			First(&existing).Error

		if err == nil {
			fmt.Printf("  skip: %s (already exists, id=%d)\n", sr.SessionName, existing.ID)
			continue
		}
		if err != gorm.ErrRecordNotFound {
			log.Printf("  error checking %s: %v", sr.SessionName, err)
			continue
		}

		encPass, err := encrypt.Encrypt(sr.Password, cfg.AESEncKey)
		if err != nil {
			log.Printf("  error encrypting password for %s: %v", sr.SessionName, err)
			continue
		}

		router := &models.Router{
			SessionName: sr.SessionName,
			IP:          sr.IP,
			Username:    sr.Username,
			PasswordEnc: encPass,
			HotspotName: sr.HotspotName,
			DNSName:     sr.DNSName,
			Currency:    sr.Currency,
			IdleTimeout: "30",
			ReportMode:  "disable",
		}

		if err := db.WithContext(ctx).Create(router).Error; err != nil {
			log.Printf("  error creating %s: %v", sr.SessionName, err)
			continue
		}

		seeded++
		fmt.Printf("  created: %s (id=%d, ip=%s)\n", sr.SessionName, router.ID, sr.IP)
	}

	fmt.Println("  ═══════ Seed Summary ═══════")
	fmt.Printf("  Total sample routers: %d\n", len(sampleRouters))
	fmt.Printf("  Seeded: %d\n", seeded)
	fmt.Printf("  Skipped: %d\n", len(sampleRouters)-seeded)
	fmt.Println("  ════════════════════════════")
}
