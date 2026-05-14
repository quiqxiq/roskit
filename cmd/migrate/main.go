package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-routeros/routeros/v3"

	"github.com/quiqxiq/roskit/internal/config"
	"github.com/quiqxiq/roskit/internal/models"
	"github.com/quiqxiq/roskit/internal/repository"
	"github.com/quiqxiq/roskit/internal/roskit/execution"
	"github.com/quiqxiq/roskit/internal/roskit/orchestrator"
	roskitservice "github.com/quiqxiq/roskit/internal/roskit/adapter/service"
	"github.com/quiqxiq/roskit/pkg/database"
	"github.com/quiqxiq/roskit/pkg/encrypt"
	"gorm.io/gorm"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	subCmd := os.Args[1]

	switch subCmd {
	case "up":
		db, err := database.Connect(cfg.PostgresDSN())
		if err != nil {
			log.Fatalf("failed to connect database: %v", err)
		}
		if err := database.AutoMigrate(db); err != nil {
			log.Fatalf("migration failed: %v", err)
		}
		log.Println("migrations applied successfully")
		return

	case "import":
		importCmd := flag.NewFlagSet("import", flag.ExitOnError)
		configFile := importCmd.String("config-file", "", "Path to config.php")
		dryRun := importCmd.Bool("dry-run", false, "Dry run mode")
		routerName := importCmd.String("router", "", "Specific router name to import (optional)")
		importCmd.Parse(os.Args[2:])

		if *configFile == "" {
			log.Fatalf("--config-file is required for import")
		}

		args := ImportArgs{
			ConfigFile: *configFile,
			DryRun:     *dryRun,
			RouterName: *routerName,
		}

		logger := slog.Default()
		pool := execution.NewPool(logger)
		engine := orchestrator.New(orchestrator.Config{Logger: logger})
		bridge := roskitservice.NewBridge(engine.Dispatcher())

		var db *gorm.DB
		if !args.DryRun {
			db, err = database.Connect(cfg.PostgresDSN())
			if err != nil {
				log.Fatalf("failed to connect database: %v", err)
			}
		}

		runImport(db, pool, bridge, cfg, args)

	case "sync-profiles":
		syncCmd := flag.NewFlagSet("sync-profiles", flag.ExitOnError)
		routerName := syncCmd.String("router", "", "Specific router name to sync (optional)")
		syncCmd.Parse(os.Args[2:])

		logger := slog.Default()
		pool := execution.NewPool(logger)
		engine := orchestrator.New(orchestrator.Config{Logger: logger})
		bridge := roskitservice.NewBridge(engine.Dispatcher())

		db, err := database.Connect(cfg.PostgresDSN())
		if err != nil {
			log.Fatalf("failed to connect database: %v", err)
		}

		runSyncProfiles(db, pool, bridge, *routerName, cfg)

	case "sync-scripts":
		syncCmd := flag.NewFlagSet("sync-scripts", flag.ExitOnError)
		routerName := syncCmd.String("router", "", "Specific router name to sync (optional)")
		dryRun := syncCmd.Bool("dry-run", false, "Preview changes without modifying routers")
		syncCmd.Parse(os.Args[2:])

		logger := slog.Default()
		pool := execution.NewPool(logger)
		engine := orchestrator.New(orchestrator.Config{Logger: logger})
		bridge := roskitservice.NewBridge(engine.Dispatcher())

		db, err := database.Connect(cfg.PostgresDSN())
		if err != nil {
			log.Fatalf("failed to connect database: %v", err)
		}

		runSyncScripts(db, pool, bridge, *routerName, *dryRun, cfg)

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Usage:
  ./migrate up
  ./migrate import --config-file=/data/config.php [--dry-run] [--router=name]
  ./migrate sync-profiles [--router=name]
  ./migrate sync-scripts [--router=name] [--dry-run]`)
}

type ImportArgs struct {
	ConfigFile string
	DryRun     bool
	RouterName string
}

type summaryStats struct {
	routersFound    int
	routersImported int
	routersSkipped  int
	salesTotal      int
	salesImported   int
	salesSkipped    int
	salesErrors     int
	profilesSynced  int
}

type routerWithConfig struct {
	router   models.Router
	settings models.Settings
}

func tenantSlugFromSession(name string) string {
	lower := strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	last := byte('-')
	for i := 0; i < len(lower); i++ {
		c := lower[i]
		switch {
		case (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9'):
			b.WriteByte(c)
			last = c
		default:
			if last != '-' {
				b.WriteByte('-')
				last = '-'
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		slug = "imported"
	}
	return slug
}

func replyToMaps(reply *routeros.Reply) []map[string]string {
	result := make([]map[string]string, 0, len(reply.Re))
	for _, sentence := range reply.Re {
		m := make(map[string]string, len(sentence.Map))
		for k, v := range sentence.Map {
			m[k] = v
		}
		result = append(result, m)
	}
	return result
}

func parseMikroTikTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Now()
	}
	formats := []string{
		"jan/02/2006 15:04:05",
		"Jan/02/2006 15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		time.RFC3339,
	}
	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t
		}
	}
	return time.Now()
}

func parseInt64(s string) int64 {
	if s == "" {
		return 0
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return v
}

func runImport(db *gorm.DB, pool *execution.Pool, bridge *roskitservice.Bridge, cfg *config.Config, args ImportArgs) {
	fmt.Println("Starting import...")

	file, err := os.Open(args.ConfigFile)
	if err != nil {
		log.Fatalf("failed to open config file: %v", err)
	}
	defer file.Close()

	var routers []routerWithConfig
	var sessions []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "$data['") {
			continue
		}

		sessionEnd := strings.Index(line[7:], "']")
		if sessionEnd == -1 {
			continue
		}
		sessionName := line[7 : 7+sessionEnd]

		if args.RouterName != "" && args.RouterName != sessionName {
			continue
		}

		valStart := strings.Index(line, "=")
		if valStart == -1 {
			continue
		}

		payload := strings.TrimSpace(line[valStart+1:])
		payload = strings.TrimPrefix(payload, `"`)
		payload = strings.TrimPrefix(payload, `'`)
		payload = strings.TrimSuffix(payload, `";`)
		payload = strings.TrimSuffix(payload, `';`)

		ipStr, rest, _ := strings.Cut(payload, "!")
		userStr, rest, _ := strings.Cut(rest, "@|@")
		passStr, rest, _ := strings.Cut(rest, "#|#")
		hotspotStr, rest, _ := strings.Cut(rest, "%")
		dnsStr, rest, _ := strings.Cut(rest, "^")
		currencyStr, rest, _ := strings.Cut(rest, "&")
		phoneStr, rest, _ := strings.Cut(rest, "*")
		emailStr, rest, _ := strings.Cut(rest, "(")
		infoStr, rest, _ := strings.Cut(rest, ")")
		idleStr, rest, _ := strings.Cut(rest, "=")
		reportStr, rest, _ := strings.Cut(rest, "@!@")
		tokenStr, _, _ := strings.Cut(rest, "#!#")

		cleartextPassword := ""
		decoded, err := encrypt.DecodeBlah(passStr)
		if err == nil && len(decoded) > 0 && isPrintable(decoded) {
			cleartextPassword = decoded
		} else {
			dec2, err2 := encrypt.DecryptLegacyPHPConfig(passStr)
			if err2 == nil && len(dec2) > 0 {
				cleartextPassword = dec2
			} else {
				cleartextPassword = passStr
			}
		}

		encryptedPassword, err := encrypt.Encrypt(cleartextPassword, cfg.AESEncKey)
		if err != nil {
			log.Printf("warning: encrypt failed for session %s: %v", sessionName, err)
			encryptedPassword = passStr
		}

		port := 8728

		r := models.Router{
			Name:                 sessionName,
			IPAddress:            ipStr,
			APIPort:              port,
			APIUsername:          userStr,
			APIPasswordEncrypted: encryptedPassword,
			Status:               models.RouterStatusUnknown,
		}
		s := models.Settings{
			HotspotName:  hotspotStr,
			DNSName:      dnsStr,
			Currency:     currencyStr,
			Phone:        phoneStr,
			Email:        emailStr,
			InfoLP:       infoStr,
			IdleTimeout:  30,
			ReportMode:   reportStr,
			WebhookToken: tokenStr,
		}
		if idleInt, err := strconv.Atoi(strings.TrimSpace(idleStr)); err == nil && idleInt > 0 {
			s.IdleTimeout = idleInt
		}
		routers = append(routers, routerWithConfig{router: r, settings: s})
		sessions = append(sessions, sessionName)
	}

	summary := summaryStats{
		routersFound: len(routers),
	}

	if args.DryRun {
		fmt.Printf("Dry run mode: Found %d routers: %v\n", summary.routersFound, sessions)
		printSummary(summary)
		return
	}

	ctx := context.Background()

	for i := range routers {
		rw := &routers[i]

		// Upsert the global Settings singleton (use data from first router if not set).
		var existingSettings models.Settings
		if db.First(&existingSettings, 1).Error != nil {
			rw.settings.ID = 0
			if err := db.Create(&rw.settings).Error; err != nil {
				log.Printf("warning: failed to create settings: %v", err)
			}
		}

		var r models.Router
		err := db.Where("name = ?", rw.router.Name).First(&r).Error
		if err == gorm.ErrRecordNotFound {
			if err := db.Create(&rw.router).Error; err != nil {
				log.Printf("failed to create router %s: %v", rw.router.Name, err)
				summary.routersSkipped++
				continue
			}
			r = rw.router
			summary.routersImported++
		} else if err != nil {
			log.Printf("failed to check router %s: %v", rw.router.Name, err)
			summary.routersSkipped++
			continue
		} else {
			summary.routersSkipped++
		}

		cleartextPassword, err := encrypt.Decrypt(r.APIPasswordEncrypted, cfg.AESEncKey)
		if err != nil {
			log.Printf("failed to decrypt password for router %s", r.Name)
			continue
		}
		port := r.APIPort
		if port == 0 {
			port = 8728
		}
		pool.Register(execution.ConnConfig{
			RouterID: fmt.Sprintf("%d", r.ID),
			Address:  fmt.Sprintf("%s:%d", r.IPAddress, port),
			Username: r.APIUsername,
			Password: cleartextPassword,
		})
	}

	ctxStart, cancelStart := context.WithTimeout(ctx, 10*time.Second)
	defer cancelStart()
	pool.Start(ctxStart)

	for _, rw := range routers {
		var dbR models.Router
		if err := db.Where("name = ?", rw.router.Name).First(&dbR).Error; err != nil {
			continue
		}

		rID := fmt.Sprintf("%d", dbR.ID)
		time.Sleep(1 * time.Second)

		reply, err := bridge.Run(ctx, rID, "/system/script/print", "?comment=mikhmon")
		if err != nil {
			log.Printf("failed to fetch scripts for router %s: %v", rw.router.Name, err)
			continue
		}

		scriptMaps := replyToMaps(reply)
		summary.salesTotal += len(scriptMaps)

		var batch []*models.VoucherSale

		for _, data := range scriptMaps {
			name := data["name"]
			fields := strings.Split(name, "-|-")
			if len(fields) < 8 {
				continue
			}

			datetimeStr := fields[0] + " " + strings.ReplaceAll(fields[1], "%3A", ":")
			soldAt := parseMikroTikTime(datetimeStr)

			price := parseInt64(fields[3])

			keyStr := rw.router.Name + fields[2] + fields[0] + fields[1]
			hash := sha256.Sum256([]byte(keyStr))
			idempKey := hex.EncodeToString(hash[:])

			var exists int64
			db.Model(&models.VoucherSale{}).Where("idempotency_key = ?", idempKey).Count(&exists)
			if exists > 0 {
				summary.salesSkipped++
				continue
			}

			rid := dbR.ID
			sale := &models.VoucherSale{
				SoldAt:         soldAt,
				Username:       fields[2],
				Price:          price,
				IPAddress:      fields[4],
				MACAddress:     fields[5],
				Validity:       fields[6],
				ProfileName:    fields[7],
				RouterID:       &rid,
				IdempotencyKey: idempKey,
			}
			batch = append(batch, sale)

			if len(batch) >= 500 {
				if err := db.CreateInBatches(batch, 500).Error; err != nil {
					summary.salesErrors += len(batch)
				} else {
					summary.salesImported += len(batch)
				}
				batch = nil
			}
		}

		if len(batch) > 0 {
			if err := db.CreateInBatches(batch, 500).Error; err != nil {
				summary.salesErrors += len(batch)
			} else {
				summary.salesImported += len(batch)
			}
		}
	}

	pool.Stop()
	printSummary(summary)
}

func printSummary(s summaryStats) {
	fmt.Println("  ═══════ Migration Summary ═══════")
	fmt.Printf("  Routers found:    %d\n", s.routersFound)
	fmt.Printf("  Routers imported: %d (skipped: %d)\n", s.routersImported, s.routersSkipped)
	fmt.Printf("  Sales records:    %d total, %d imported, %d skipped (duplicates), %d errors\n", s.salesTotal, s.salesImported, s.salesSkipped, s.salesErrors)
	fmt.Printf("  Profiles synced:  %d\n", s.profilesSynced)
	fmt.Println("  ═══════════════════════════════")
}

func isPrintable(s string) bool {
	for _, r := range s {
		if r < 32 || r > 126 {
			return false
		}
	}
	return true
}

func runSyncProfiles(db *gorm.DB, pool *execution.Pool, bridge *roskitservice.Bridge, routerName string, cfg *config.Config) {
	fmt.Println("Starting sync-profiles...")
	ctx := context.Background()

	var routers []models.Router
	if routerName != "" {
		if err := db.Where("name = ?", routerName).Find(&routers).Error; err != nil {
			log.Fatalf("failed to find router: %v", err)
		}
	} else {
		if err := db.Find(&routers).Error; err != nil {
			log.Fatalf("failed to find routers: %v", err)
		}
	}

	for _, r := range routers {
		cleartextPassword, err := encrypt.Decrypt(r.APIPasswordEncrypted, cfg.AESEncKey)
		if err != nil {
			continue
		}
		port := r.APIPort
		if port == 0 {
			port = 8728
		}
		pool.Register(execution.ConnConfig{
			RouterID: fmt.Sprintf("%d", r.ID),
			Address:  fmt.Sprintf("%s:%d", r.IPAddress, port),
			Username: r.APIUsername,
			Password: cleartextPassword,
		})
	}

	ctxStart, cancelStart := context.WithTimeout(ctx, 5*time.Second)
	defer cancelStart()
	pool.Start(ctxStart)

	profileRepo := repository.NewProfilePriceMappingRepo(db)

	summary := summaryStats{}

	for _, r := range routers {
		rID := fmt.Sprintf("%d", r.ID)
		time.Sleep(1 * time.Second)

		profiles, err := bridge.ListHotspotProfiles(ctx, rID)
		if err != nil {
			log.Printf("failed to fetch profiles for router %s: %v", r.Name, err)
			continue
		}

		for _, prof := range profiles {
			onLogin := prof["on-login"]
			if onLogin == "" {
				continue
			}

			meta := roskitservice.ParseOnLoginPut(onLogin)
			if meta == nil {
				continue
			}

			price, _ := strconv.ParseInt(meta.Price, 10, 64)
			sprice, _ := strconv.ParseInt(meta.SellingPrice, 10, 64)
			lockUser := meta.LockUser == "Enable"
			lockServer := meta.LockServer != "Disable" && meta.LockServer != ""

			mapping := &models.ProfilePriceMapping{
				RouterID:     r.ID,
				ProfileName:  prof["name"],
				Price:        price,
				SellingPrice: sprice,
				Validity:     meta.Validity,
				ExpMode:      meta.ExpMode,
				LockUser:     lockUser,
				LockServer:   lockServer,
			}

			if err := profileRepo.Upsert(ctx, mapping); err != nil {
				log.Printf("warning: failed to upsert profile %s on router %s: %v", prof["name"], r.Name, err)
			} else {
				summary.profilesSynced++
			}
		}
	}

	pool.Stop()
	printSummary(summary)
}

func runSyncScripts(db *gorm.DB, pool *execution.Pool, bridge *roskitservice.Bridge, routerName string, dryRun bool, cfg *config.Config) {
	if dryRun {
		fmt.Println("Starting sync-scripts (DRY RUN)...")
	} else {
		fmt.Println("Starting sync-scripts...")
	}
	ctx := context.Background()

	var routers []models.Router
	if routerName != "" {
		if err := db.Where("name = ?", routerName).Find(&routers).Error; err != nil {
			log.Fatalf("failed to find router: %v", err)
		}
	} else {
		if err := db.Find(&routers).Error; err != nil {
			log.Fatalf("failed to find routers: %v", err)
		}
	}

	for _, r := range routers {
		cleartextPassword, err := encrypt.Decrypt(r.APIPasswordEncrypted, cfg.AESEncKey)
		if err != nil {
			continue
		}
		port := r.APIPort
		if port == 0 {
			port = 8728
		}
		pool.Register(execution.ConnConfig{
			RouterID: fmt.Sprintf("%d", r.ID),
			Address:  fmt.Sprintf("%s:%d", r.IPAddress, port),
			Username: r.APIUsername,
			Password: cleartextPassword,
		})
	}

	ctxStart, cancelStart := context.WithTimeout(ctx, 5*time.Second)
	defer cancelStart()
	pool.Start(ctxStart)

	profileRepo := repository.NewProfilePriceMappingRepo(db)

	var totalProfiles, updatedProfiles, skippedProfiles, syncedMappings int

	for _, r := range routers {
		rID := fmt.Sprintf("%d", r.ID)
		time.Sleep(1 * time.Second)

		profiles, err := bridge.ListHotspotProfiles(ctx, rID)
		if err != nil {
			log.Printf("failed to fetch profiles for router %s: %v", r.Name, err)
			continue
		}

		var webhookToken string
		settingsRepo := repository.NewSettingsRepo(db)
		if settings, err := settingsRepo.Get(ctx); err == nil && settings != nil {
			webhookToken = settings.WebhookToken
		}

		fmt.Printf("\nRouter: %s (ID: %d)\n", r.Name, r.ID)

		for _, prof := range profiles {
			totalProfiles++
			onLogin := prof["on-login"]
			if onLogin == "" {
				continue
			}

			meta := roskitservice.ParseOnLoginPut(onLogin)
			if meta == nil {
				continue
			}

			price, _ := strconv.ParseInt(meta.Price, 10, 64)
			sprice, _ := strconv.ParseInt(meta.SellingPrice, 10, 64)
			lockUser := meta.LockUser == "Enable"
			lockServer := meta.LockServer != "Disable" && meta.LockServer != ""

			fmt.Printf("  Profile %q — expmode=%s, price=%d, validity=%s\n",
				prof["name"], meta.ExpMode, price, meta.Validity)

			mapping := &models.ProfilePriceMapping{
				RouterID:     r.ID,
				ProfileName:  prof["name"],
				Price:        price,
				SellingPrice: sprice,
				Validity:     meta.Validity,
				ExpMode:      meta.ExpMode,
				LockUser:     lockUser,
				LockServer:   lockServer,
			}

			if dryRun {
				fmt.Printf("    > ProfilePriceMapping: WILL UPSERT\n")
			} else {
				if err := profileRepo.Upsert(ctx, mapping); err != nil {
					log.Printf("    warning: failed to upsert profile %s: %v", prof["name"], err)
				} else {
					syncedMappings++
				}
			}

			newScript := roskitservice.GenerateOnLoginScript(roskitservice.OnLoginParams{
				ExpMode:      meta.ExpMode,
				Price:        meta.Price,
				SellingPrice: meta.SellingPrice,
				Validity:     meta.Validity,
				ProfileName:  prof["name"],
				LockUser:     meta.LockUser,
				LockServer:   meta.LockServer,
				APIURL:       cfg.PublicAPIURL,
				WebhookToken: webhookToken,
				RouterName:   r.Name,
			})

			if newScript == onLogin {
				fmt.Printf("    > On-login script: ALREADY UP TO DATE\n")
				skippedProfiles++
				continue
			}

			if dryRun {
				if strings.Contains(onLogin, "API_URL") || strings.Contains(onLogin, "router_session") {
					fmt.Printf("    > On-login script: WILL REPLACE (legacy placeholder detected)\n")
				} else if strings.Contains(onLogin, "/system script add") {
					fmt.Printf("    > On-login script: WILL REPLACE (legacy sales recording detected)\n")
				} else {
					fmt.Printf("    > On-login script: WILL REPLACE\n")
				}
			} else {
				profID := prof[".id"]
				if profID == "" {
					log.Printf("    warning: profile %s has no .id, skipping script replacement", prof["name"])
					skippedProfiles++
					continue
				}
				if err := bridge.SetHotspotProfile(ctx, rID, profID, map[string]string{
					"on-login": newScript,
				}); err != nil {
					log.Printf("    error: failed to update profile %s: %v", prof["name"], err)
				} else {
					updatedProfiles++
					fmt.Printf("    > On-login script: REPLACED\n")
				}
			}
		}
	}

	pool.Stop()

	fmt.Println("\n  ======= Sync-Scripts Summary =======")
	fmt.Printf("  Routers:              %d\n", len(routers))
	fmt.Printf("  Profiles scanned:     %d\n", totalProfiles)
	fmt.Printf("  Scripts replaced:     %d\n", updatedProfiles)
	fmt.Printf("  Scripts up-to-date:   %d\n", skippedProfiles)
	fmt.Printf("  Profile mappings:     %d synced\n", syncedMappings)
	if dryRun {
		fmt.Println("  Mode:                 DRY RUN (no changes made)")
	}
	fmt.Println("  ===================================")
}
