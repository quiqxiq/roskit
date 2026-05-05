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
	Name        string
	IPAddress   string
	APIPort     int
	APIUsername string
	Password    string
	HotspotName string
	DNSName     string
	Currency    string
}{
	{
		Name:        "router-alpha",
		IPAddress:   "192.168.233.1",
		APIPort:     8728,
		APIUsername: "admin",
		Password:    "",
		HotspotName: "Alpha Hotspot",
		DNSName:     "alpha.local",
		Currency:    "Rp",
	},
	{
		Name:        "router-beta",
		IPAddress:   "192.168.230.2",
		APIPort:     8728,
		APIUsername: "admin",
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

	ctx := context.Background()
	seeded := 0

	for _, sr := range sampleRouters {
		var existing models.Router
		err := db.WithContext(ctx).
			Where("name = ?", sr.Name).
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

		txErr := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			router := &models.Router{
				Name:                 sr.Name,
				IPAddress:            sr.IPAddress,
				APIPort:              port,
				APIUsername:          sr.APIUsername,
				APIPasswordEncrypted: encPass,
				Status:               models.RouterStatusUnknown,
			}
			if err := tx.Create(router).Error; err != nil {
				return err
			}
			currency := sr.Currency
			if currency == "" {
				currency = "Rp"
			}
			hotspotConfig := &models.HotspotConfig{
				RouterID:    router.ID,
				HotspotName: sr.HotspotName,
				DNSName:     sr.DNSName,
				Currency:    currency,
				IdleTimeout: 30,
				ReportMode:  "disable",
			}
			return tx.Create(hotspotConfig).Error
		})
		if txErr != nil {
			log.Printf("  error creating %s: %v", sr.Name, txErr)
			continue
		}

		seeded++
		fmt.Printf("  created: %s (ip=%s)\n", sr.Name, sr.IPAddress)
	}

	fmt.Println("  ═══════ Seed Summary ═══════")
	fmt.Printf("  Total sample routers: %d\n", len(sampleRouters))
	fmt.Printf("  Seeded: %d\n", seeded)
	fmt.Printf("  Skipped: %d\n", len(sampleRouters)-seeded)
	fmt.Println("  ════════════════════════════")
}
