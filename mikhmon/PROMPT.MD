# PROMPT MIGRASI MIKHMON → GO (REVISI v3)
## Berdasarkan: Mikhmon V3 + V4 (PHP) + mikhmon-api (Go project — state terbaru)

---

## PERUBAHAN SIGNIFIKAN DARI v2 KE v3

Analisis Go project terbaru menunjukkan progress yang jauh lebih maju dari sebelumnya.
Beberapa hal yang di v2 dianggap "belum ada" ternyata sudah selesai di v3:

| Komponen | Asumsi di v2 | Realita di v3 | Impact ke Prompt |
|----------|-------------|---------------|-----------------|
| HotspotCookie | Belum ada | Domain + Spec + Tool selesai | Step 8 tidak perlu implement backend cookie lagi |
| IPBinding | Belum ada | Domain + Spec + Tool selesai | Step 8 tidak perlu implement backend binding lagi |
| QuickPrint | Belum ada | Domain + Spec + Tool selesai | Step baru diperlukan untuk handler |
| System Shutdown/Reboot | Hanya disebutkan | Tool selesai (tool/system.go) | Step 8 tinggal buat handler |
| Redis-first read | Belum diimplementasi | Semua spec ad-hoc pakai Redis-first | Pattern caching di service layer berubah |
| Secondary index Redis | Belum ada | Selesai (SetIndex/GetByIndex/DeleteIndex) | Cache key lookup by name O(1) |
| CacheRepository 10 method | Partial | Selesai lengkap | Executor sudah implements CachedCommandRunner |
| HotspotServerSpec | Polling | Streaming =follow | Server list realtime, bukan periodic |
| Cleanup methods | Belum ada | 17 methods incl. cleanup di hotspot.go | RemoveUser sekarang include cleanup |
| TTL 5min auto-expire | Belum ada | Selesai di telemetry.go | Data stale auto-bersih saat router disconnect |

**Implikasi terbesar:** Service layer tidak perlu implement cache invalidation manual untuk
data RouterOS. Cache dikelola sepenuhnya oleh streaming telemetry pipeline.
Service layer hanya perlu membaca dari cache (via Executor.GetSnapshot/ScanByMeasurement)
dan menulis ke RouterOS (via Executor.Run/Add/Set/Remove).

---

## BAGIAN 1: PETA STATUS IMPLEMENTASI (AKURAT — STATE TERBARU)

### Sudah Selesai — Jangan Disentuh

```
internal/routeros/
  domain/       — 17 entity types ✓
    config.go, events.go (+ IndexName/IndexKey), sales_record.go
    system_resource.go (+ Clock + Identity + Routerboard + Health)
    system_scheduler.go, hotspot_user.go, hotspot_active.go
    hotspot_profile.go, hotspot_server.go, hotspot_host.go
    hotspot_cookie.go ✓ NEW, ip_binding.go ✓ NEW
    ppp_secret.go, ppp_active.go, interface_stats.go
    dhcp_lease.go, quick_print.go ✓ NEW

  collector/    — pool.go, router.go, engine.go ✓
    (11 StreamSpec + 1 PollSpec sekarang)

  spec/         — 20 files ✓
    spec.go (StreamSpec + PollSpec + CacheReader + CachedCommandRunner interfaces)
    hotspot_user.go (Redis-first), hotspot_active.go (Redis-first)
    hotspot_profile.go (Redis-first), hotspot_host.go (Redis-first)
    hotspot_cookie.go ✓ NEW (Redis-first), hotspot_server.go ✓ =follow (Redis-first)
    ip_binding.go ✓ NEW (Redis-first), ppp_secret.go, ppp_active.go
    interface_traffic.go, system_resource_poll.go, dhcp_lease.go
    system.go, scheduler.go, report.go, voucher.go (Redis-first)
    quick_print.go ✓ NEW, network.go (FindARP, FindDHCPLease, FindQueue, FindScheduler, FindScript)

  tool/         — 8 files ✓
    executor.go (CachedCommandRunner impl — wraps Engine + CacheReader)
    hotspot.go (17 methods incl. cleanup: RemoveIPBindingWithCleanup,
                RemoveHotspotUserWithCleanup, RemoveHotspotProfileWithCleanup,
                DisconnectUser, RemoveCookie, EnableIPBinding, DisableIPBinding)
    voucher.go, script.go, scheduler.go
    system.go (+ Shutdown, Reboot) ✓ NEW
    report.go, quick_print.go ✓ NEW

  usecase/      — telemetry.go (TTL 5min + secondary index), aggregator.go ✓
  repository/   — redis.go (10 methods + pipeline), influxdb.go, noop.go ✓
  utils/        — parser.go (+ FormatIndexKey) ✓

internal/models/      — 6 GORM models ✓
migrations/           — 001, 002 SQL ✓
internal/api/router.go — route definitions ✓
internal/api/middleware/auth.go — CORS ✓ (Auth+Logger masih stub)
pkg/database/postgres.go, pkg/redis/client.go, pkg/errors/errors.go ✓
docker/ — complete ✓
```

### Stub — Harus Diimplementasi (Target Utama)

```
internal/api/handlers/
  auth_handler.go         — Login, Logout, Me, ChangePassword
  router_handler.go       — CRUD + TestConnection + Migrate
  hotspot_handler.go      — 16 methods CRUD user/profile/active/host/server
  voucher_handler.go      — Generate, Cache, RecordSale, ImportSales
  report_handler.go       — Daily, Monthly, Summary, Export CSV/Excel
  event_handler.go        — OnLoginEvent (HealthCheck sudah selesai)

internal/services/
  auth_service.go         — Login, ValidateToken, CreateUser, ChangePassword
  hotspot_service.go      — 16 methods
  report_service.go       — GetDailyReport, GetMonthlyReport, GetDailySummary
  router_service.go       — CRUD + TestConnection + Migrate
  voucher_service.go      — Generate, RecordSale, Import

internal/repository/
  router_repo.go          — CRUD via GORM
  sale_repo.go            — CRUD + query by date/month + aggregates
  user_repo.go            — CRUD + GetByUsername
  template_repo.go        — CRUD + seed defaults

internal/api/middleware/
  auth.go                 — AuthMiddleware + RequireRole (JWT)
  logger.go               — Request/response logging

pkg/encrypt/encrypt.go   — AES-256-GCM + legacy PHP decrypt
pkg/redis/cache.go       — Application-level cache (BUKAN RouterOS cache)
cmd/worker/main.go       — Background task runner
```

### Belum Ada — Harus Dibuat (Handler + Service baru)

```
internal/api/handlers/
  system_handler.go       — resource, log, reboot, shutdown, scheduler
  ppp_handler.go          — PPP secrets, active, profiles
  network_handler.go      — interfaces, traffic, pools, queues, NAT
  quick_print_handler.go  — Quick print package CRUD
  ws_handler.go           — WebSocket bridge Redis Pub/Sub → client

internal/services/
  system_service.go       — wrap spec/tool system functions
  ppp_service.go          — wrap spec/tool PPP functions

internal/api/router.go (tambahan routes):
  /system/*, /ppp/*, /network/*, /quick-print/*, /ws

migrations/
  003_schema_corrections.sql
```

---

## BAGIAN 2: POLA ARSITEKTUR KRITIS (WAJIB DIPAHAMI AGENT)

### 2.1 Dua Jenis Cache — JANGAN Dicampur

Ini adalah sumber kebingungan terbesar. Ada dua cache system yang berbeda:

**Cache A — RouterOS Telemetry Cache (SUDAH ADA, JANGAN DISENTUH)**
```
internal/routeros/repository/redis.go
  Interface: CacheRepository (10 methods)
  Key pattern: roskit:{routerID}:{measurement}:{entityID}
  Secondary index: roskit:{routerID}:idx:{measurement}
  TTL: 5 menit (auto-refresh oleh streaming)
  Dikelola: TelemetryUseCase (otomatis saat stream)
  Diakses: spec ad-hoc functions via CachedCommandRunner
  Purpose: Cache data RouterOS operasional (users, profiles, active sessions, dll)
```

**Cache B — Application Cache (STUB, HARUS DIIMPLEMENTASI)**
```
pkg/redis/cache.go
  Interface: Cache struct (Get, Set, Delete, Invalidate)
  Key pattern: mikhmon:{resource}:{id}
  TTL: manual per use case (sales 60s, system 10s, dll)
  Dikelola: Service layer (manual set/invalidate)
  Purpose: Cache hasil query PostgreSQL, aggregasi sales, dashboard summary
```

**Aturan penggunaan:**
- Data dari RouterOS → baca lewat `spec.*` functions (otomatis Redis-first)
- Data dari PostgreSQL → baca lewat repository, cache hasil di Cache B
- Jangan pernah buat code yang cache data RouterOS manual di service layer

### 2.2 Cara Service Mengakses Data RouterOS

Service layer tidak boleh memanggil RouterOS API secara langsung.
Harus melalui `tool.Executor` yang implements `CachedCommandRunner`:

```go
// BENAR — lewat Executor yang sudah implements CachedCommandRunner
func (s *HotspotService) ListUsers(ctx context.Context, routerID uint, profile string) ([]domain.HotspotUser, error) {
    // spec.ListHotspotUsers menerima CachedCommandRunner
    // Executor mengimplementasi interface ini
    users, err := spec.ListHotspotUsers(ctx, s.executor, routerID, profile)
    return users, err
}

// SALAH — jangan bypass lewat pool langsung
func (s *HotspotService) ListUsers(ctx context.Context, routerID uint, profile string) ([]domain.HotspotUser, error) {
    conn, _ := s.pool.Get(routerID)
    reply, _ := conn.ExecuteContext(ctx, "/ip/hotspot/user/print")
    // ... manual parsing (sudah ada di spec, jangan duplikasi)
}
```

### 2.3 Cara Service Menulis ke RouterOS

Untuk mutasi (Add/Set/Remove), service memanggil `tool.*` functions:

```go
func (s *HotspotService) AddUser(ctx context.Context, routerID uint, params AddUserParams) (*domain.HotspotUser, error) {
    // tool.AddHotspotUser memanggil /ip/hotspot/user/add
    // Setelah berhasil, streaming =follow otomatis update Redis cache
    result, err := tool.AddHotspotUser(ctx, s.executor, routerID, params)
    // TIDAK perlu manual cache invalidation — streaming handles it
    return result, err
}
```

### 2.4 Executor Initialization

Executor adalah jembatan antara service dan RouterOS + Redis cache:

```go
// Di api/router.go saat wiring DI:
cacheRepo := routerosRepo.NewRedisRepository(rdb)
executor := tool.NewExecutor(engine, cacheRepo)

// Inject ke semua services yang butuh RouterOS access:
hotspotSvc := services.NewHotspotService(executor, profileRepo, cache)
voucherSvc := services.NewVoucherService(executor, saleRepo, cache)
```

### 2.5 TTL dan Cache Invalidation

Karena telemetry pipeline auto-manage cache RouterOS dengan TTL 5 menit:
- Jika router **connected**: cache selalu fresh (streaming terus refresh TTL)
- Jika router **disconnected**: data expire otomatis setelah 5 menit
- Service layer **tidak perlu** dan **tidak boleh** invalidate RouterOS cache secara manual

Yang perlu manual invalidation hanya Cache B (PostgreSQL results):
```go
// Setelah voucher sale event masuk:
s.cache.Invalidate(ctx,
    salesKey(routerID, "today"),
    salesKey(routerID, "month"),
)
```

---

## BAGIAN 3: SYSTEM INSTRUCTION (Selalu disertakan di setiap sesi)

```
You are continuing development of "mikhmon-api", a Go backend project that is
already substantially implemented. ALWAYS read existing files before modifying.

═══════════════════════════════════════════════════════════════
CRITICAL: WHAT IS COMPLETE — DO NOT RECREATE OR MODIFY
═══════════════════════════════════════════════════════════════

internal/routeros/ — ENTIRE SUBSYSTEM IS COMPLETE (~5,200+ lines, 70+ files)
  domain/    17 entity types including HotspotCookie, IPBinding, QuickPrintPackage
  collector/ connection pool, per-router streaming, engine
  spec/      20 files — ALL ad-hoc read functions use Redis-first pattern
             Key interface: CachedCommandRunner (CommandRunner + CacheReader)
  tool/      8 files — executor.go implements CachedCommandRunner
             hotspot.go has 17 methods including cleanup variants
             system.go has Shutdown() and Reboot()
             quick_print.go manages QuickPrint packages
  usecase/   telemetry (TTL 5min + secondary index), aggregator
  repository/ CacheRepository with 10 methods, InfluxDB, NoOp
  utils/     parser + FormatCacheKey + FormatPubSubChannel + FormatIndexKey

═══════════════════════════════════════════════════════════════
ARCHITECTURAL RULES — NON-NEGOTIABLE
═══════════════════════════════════════════════════════════════

RULE 1: RouterOS data access
  Services MUST access RouterOS data via spec.* functions (Redis-first).
  spec.* functions accept CachedCommandRunner interface.
  tool.Executor implements CachedCommandRunner.
  DO NOT create direct pool connections in service layer.

RULE 2: Two cache systems — never confuse them
  A) internal/routeros/repository (CacheRepository):
     RouterOS telemetry cache. Key: roskit:{routerID}:{measurement}:{id}
     Managed automatically by streaming pipeline. DO NOT touch in services.
  B) pkg/redis/cache.go (Cache, stub):
     Application cache for PostgreSQL query results.
     Key: mikhmon:{resource}:{id}. Managed manually by service layer.

RULE 3: Cache invalidation
  RouterOS data: NEVER invalidate manually — streaming TTL handles it (5min).
  PostgreSQL results: invalidate manually after write operations via Cache B.

RULE 4: Cleanup operations
  For Remove operations, use the WithCleanup variants in tool/hotspot.go:
  RemoveHotspotUserWithCleanup, RemoveHotspotProfileWithCleanup, RemoveIPBindingWithCleanup
  These handle cascading cleanup (scripts, schedulers, ARP, DHCP) automatically.

RULE 5: Executor construction
  tool.NewExecutor(engine, cacheRepo) — cacheRepo is routerosRepo.CacheRepository
  This is injected into all services that need RouterOS access.
  Wire this in internal/api/router.go during DI setup.

═══════════════════════════════════════════════════════════════
MIKHMON DOMAIN KNOWLEDGE
═══════════════════════════════════════════════════════════════

Sales records stored in RouterOS /system/script:
  name = "date-|-time-|-user-|-price-|-ip-|-mac-|-validity-|-profile-|-comment"
  owner = "MonthYear" (e.g., "Apr2026"), comment = "mikhmon"

Profile pricing in on-login :put statement:
  :put (",{expmode},{price},{validity},{sprice},,{lockuser},{lockserver},")

Voucher types: vc (username=password), up (username≠password)

QuickPrint packages stored in /system/script with specific comment prefix.
  spec.quick_print.go handles parsing.

on-login script for recording must use HTTP POST (not RouterOS script):
  /tool/fetch sends form data (NOT JSON) to /api/v1/events/on-login

RouterOS script variable: $"mac-address" (not $mac), $user (not $username)

═══════════════════════════════════════════════════════════════
WHAT NEEDS IMPLEMENTATION (target of these prompts)
═══════════════════════════════════════════════════════════════

STUBS (implement existing signatures):
  internal/api/handlers/ — 6 files (all stubs)
  internal/services/ — 5 files (all stubs)
  internal/repository/ — 4 files (all stubs)
  internal/api/middleware/ — auth.go (Auth+Logger stubs)
  pkg/encrypt/encrypt.go — AES + legacy PHP decrypt (stubs)
  pkg/redis/cache.go — application cache wrapper (stub)

NEW FILES (create):
  internal/api/handlers/system_handler.go
  internal/api/handlers/ppp_handler.go
  internal/api/handlers/network_handler.go
  internal/api/handlers/quick_print_handler.go
  internal/api/handlers/ws_handler.go
  internal/services/system_service.go
  internal/services/ppp_service.go
  migrations/003_schema_corrections.sql

═══════════════════════════════════════════════════════════════
CODING STANDARDS
═══════════════════════════════════════════════════════════════

- All functions accept context.Context as first param
- Response: {"data": ..., "error": null} or {"data": null, "error": "msg"}
- Use AppError from pkg/errors for all error responses
- Repository: pure GORM, no raw SQL unless aggregation requires it
- Service: thin orchestration — spec/tool calls + repo calls + cache B ops
- Handler: thin controller — bind, validate, call service, return JSON
- No global variables, no panics in goroutines without recovery
- go build ./... must pass after every step
```

---

## BAGIAN 4: PROMPT STEP-BY-STEP

---

### STEP 1 — Audit dan Verifikasi State

```
STEP 1: CODEBASE AUDIT

Verify the actual state of the project before writing any code.

TASK A: Compilation check
  Run: go build ./...
  If errors exist, list them by file. Do not fix them yet — just document.

TASK B: Verify key interfaces exist
  Read internal/routeros/spec/spec.go.
  Confirm these interfaces are defined:
    - CommandRunner
    - CacheReader (GetSnapshot, ScanByMeasurement, CountByMeasurement, GetByIndex)
    - CachedCommandRunner (combines both)
  Confirm tool.Executor implements CachedCommandRunner.

TASK C: Inventory all stub functions with their dependency needs
  For each stub file (handlers, services, repositories, middleware, encrypt, cache):
    1. List all function signatures
    2. For each function, identify:
       - spec.* functions it needs (read from RouterOS via cache)
       - tool.* functions it needs (write to RouterOS)
       - repository methods it needs (PostgreSQL)
       - cache B methods it needs (application cache)

TASK D: Schema gap analysis
  Compare internal/models/*.go against migrations/001_initial_schema.sql.
  Identify missing columns:
    voucher_sales: server, selling_price, idempotency_key
    print_templates: router_id, part

TASK E: Route completeness check
  Read internal/api/router.go.
  List routes that exist but have no corresponding handler method yet:
    Expected missing: /system/*, /ppp/*, /network/*, /quick-print/*, /ws

TASK F: Dependency check
  Read go.mod. Confirm these are present:
    - github.com/go-routeros/routeros/v3
    - gorm.io/gorm, gorm.io/driver/postgres
    - github.com/gin-gonic/gin
    - github.com/redis/go-redis/v9
    - github.com/influxdata/influxdb-client-go/v2
  Note which are missing (jwt, excelize, websocket if not present).

Output: gap report document. Do NOT write any code in this step.
Confirm findings before proceeding to Step 2.
```

---

### STEP 2 — Schema Corrections dan Migration 003

```
STEP 2: SCHEMA CORRECTIONS

Based on gap analysis from Step 1, create migrations/003_schema_corrections.sql.

A. VoucherSale corrections:
   ALTER TABLE voucher_sales
     ADD COLUMN IF NOT EXISTS server VARCHAR(100),
     ADD COLUMN IF NOT EXISTS selling_price BIGINT NOT NULL DEFAULT 0,
     ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(200);
   CREATE UNIQUE INDEX IF NOT EXISTS idx_voucher_sales_idempotency
     ON voucher_sales(idempotency_key) WHERE idempotency_key IS NOT NULL;
   CREATE INDEX IF NOT EXISTS idx_voucher_sales_server
     ON voucher_sales(server) WHERE server IS NOT NULL;

B. PrintTemplate corrections:
   ALTER TABLE print_templates
     ADD COLUMN IF NOT EXISTS router_id BIGINT REFERENCES routers(id) ON DELETE CASCADE,
     ADD COLUMN IF NOT EXISTS part VARCHAR(10) CHECK (part IN ('header', 'row', 'footer'));

C. Update internal/models/voucher_sale.go:
   Add fields: Server string, SellingPrice int64 (default 0), IdempotencyKey string
   Add GORM tags: gorm:"index" for Server, gorm:"uniqueIndex" for IdempotencyKey
   Keep existing RouterID FK and all other fields.

D. Update internal/models/print_template.go:
   Add fields: RouterID uint (FK), Part string
   Add GORM tag: gorm:"check:chk_part,part IN ('header','row','footer')"

E. Update pkg/database/postgres.go AutoMigrate:
   Ensure updated models are in the AutoMigrate call.

F. Verify: go build ./... passes.
```

---

### STEP 3 — pkg/encrypt dan pkg/redis/cache.go

```
STEP 3: SHARED PACKAGES

A. IMPLEMENT pkg/encrypt/encrypt.go

The file has 4 stub functions. Implement all:

1. Encrypt(plaintext, keyBase64 string) (string, error)
   - Decode keyBase64: base64.StdEncoding.DecodeString(keyBase64) → 32 bytes
   - AES-256-GCM: aes.NewCipher → cipher.NewGCM
   - Random nonce: 12 bytes via crypto/rand
   - Seal into nonce+ciphertext
   - Return: base64.StdEncoding.EncodeToString(nonce + ciphertext)

2. Decrypt(ciphertext, keyBase64 string) (string, error)
   - Reverse of Encrypt

3. DecryptLegacyPHPConfig(encrypted string) (string, error)
   PHP Mikhmon V3 algorithm (XOR key=128):
   a. base64.StdDecoding(encrypted) → bytes
   b. XOR each byte with 128
   c. Return as string

4. DecodeBlah(encoded string) (string, error)
   PHP Mikhmon V4 algorithm (triple base64 + XOR key=10):
   a. base64.StdDecoding(encoded) → b1
   b. XOR each byte of b1 with 10 → b2
   c. base64.StdDecoding(b2) → b3
   d. base64.StdDecoding(b3) → plaintext

Write encrypt_test.go:
   - Test Encrypt/Decrypt roundtrip
   - Test DecryptLegacyPHPConfig with a known V3-encrypted password
   - Test DecodeBlah with a known V4-encrypted password
   (use hardcoded test vectors — compute expected values manually)

B. IMPLEMENT pkg/redis/cache.go

IMPORTANT DISTINCTION: This is application-level cache (Cache B) for PostgreSQL
query results. It is COMPLETELY SEPARATE from the RouterOS telemetry cache
(internal/routeros/repository/redis.go). Do not confuse the two.

Replace stub with full implementation:

type Cache struct {
    client *redis.Client
}

func NewCache(client *redis.Client) *Cache

TTL Constants:
   TTLSalesDay    = 60 * time.Second    // today's sales (invalidated on new sale)
   TTLSalesMonth  = 5 * time.Minute     // monthly aggregates
   TTLSalesPast   = 1 * time.Hour       // past date sales (immutable)
   TTLDashboard   = 10 * time.Second    // dashboard summary
   TTLTemplates   = 5 * time.Minute     // print templates

Key helpers (MUST use "mikhmon:" prefix to not collide with RouterOS "roskit:" keys):
   func SalesKey(routerID uint, period string) string → "mikhmon:sales:{routerID}:{period}"
   func DashboardKey(routerID uint) string → "mikhmon:dashboard:{routerID}"
   func TemplateKey(routerID uint, name, part string) string → "mikhmon:template:{routerID}:{name}:{part}"
   func AuthRevokedKey(tokenID string) string → "mikhmon:auth:revoked:{tokenID}"
   func AuthRefreshKey(userID, tokenID string) string → "mikhmon:auth:refresh:{userID}:{tokenID}"
   func VoucherSessionKey(routerID uint, gencode string) string → "mikhmon:vsession:{routerID}:{gencode}"
   TTLVoucherSession = 2 * time.Hour

Methods:
   Get(ctx, key string) (string, bool, error)
      HGET → (value, exists, error). Returns ("", false, nil) if key not found.

   Set(ctx, key, value string, ttl time.Duration) error
      SET key value EX ttl. ttl=0 means no expiry.

   Delete(ctx, keys ...string) error
      DEL multi-key in one call.

   SetJSON(ctx, key string, value any, ttl time.Duration) error
      Marshal → Set

   GetJSON(ctx, key string, dest any) (bool, error)
      Get → Unmarshal. Returns (false, nil) on miss.

   GetOrSetJSON[T any](ctx, key string, ttl time.Duration, fetch func() (T, error)) (T, error)
      Cache-aside pattern:
      1. Try GetJSON → if found return
      2. Call fetch()
      3. SetJSON with ttl
      4. On Redis error: log warning, call fetch() directly (degrade gracefully)
      Note: Never return error if fetch() succeeds, even if Redis fails.

   Invalidate(ctx, keys ...string) error
      Alias for Delete.

C. Verify: go build ./... passes.
```

---

### STEP 4 — Auth Foundation

```
STEP 4: AUTHENTICATION

Add dependency: go get github.com/golang-jwt/jwt/v5
Add dependency: go get golang.org/x/crypto (for bcrypt, likely already present)

A. IMPLEMENT internal/repository/user_repo.go

Interface:
  type UserRepository interface {
    Create(ctx, user *models.SystemUser) error
    GetByID(ctx, id uint) (*models.SystemUser, error)
    GetByUsername(ctx, username string) (*models.SystemUser, error)
    List(ctx) ([]*models.SystemUser, error)
    Update(ctx, user *models.SystemUser) error
    UpdateLastLogin(ctx, userID uint) error
    Delete(ctx, id uint) error
    Count(ctx) (int64, error)
  }

GORM implementation: all queries use db.WithContext(ctx).
GetByUsername: use Where("username = ? AND deleted_at IS NULL") with soft delete.

B. IMPLEMENT internal/services/auth_service.go

Dependencies: UserRepository, *cache.Cache, jwtSecret []byte, refreshSecret []byte

type LoginResult struct {
  AccessToken  string    `json:"access_token"`
  RefreshToken string    `json:"refresh_token"`
  ExpiresIn    int       `json:"expires_in"` // seconds
  User         UserView  `json:"user"`
}

type Claims struct {
  UserID   uint   `json:"uid"`
  Username string `json:"sub"`
  Role     string `json:"role"`
  TokenID  string `json:"jti"`
  jwt.RegisteredClaims
}

Implement:

Login(ctx, username, password string) (*LoginResult, error)
  1. userRepo.GetByUsername
  2. Check user.Active == true
  3. bcrypt.CompareHashAndPassword(user.PasswordHash, password)
  4. Generate tokenID = uuid or random 16 bytes hex
  5. Access token: Claims{UserID, Username, Role, TokenID}, exp 15min, sign with jwtSecret
  6. Refresh token: Claims{UserID, TokenID: refreshTokenID}, exp 7d, sign with refreshSecret
  7. cache.Set(ctx, AuthRefreshKey(userID, refreshTokenID), "1", 7*24*time.Hour)
  8. userRepo.UpdateLastLogin(ctx, user.ID)
  9. Return LoginResult

ValidateToken(ctx, tokenString string) (*Claims, error)
  1. jwt.ParseWithClaims with jwtSecret
  2. cache.Get(ctx, AuthRevokedKey(claims.TokenID)) → if found: return Unauthorized
  3. Return claims

ValidateRefreshToken(ctx, tokenString string) (*Claims, error)
  Similar but use refreshSecret, check refresh key in Redis.

Logout(ctx, accessTokenID, userID, refreshTokenID string, accessTokenExpiry time.Time) error
  1. Remaining TTL = accessTokenExpiry - time.Now()
  2. cache.Set(ctx, AuthRevokedKey(accessTokenID), "1", remainingTTL)
  3. cache.Delete(ctx, AuthRefreshKey(userID, refreshTokenID))

RefreshToken(ctx, refreshToken string) (*LoginResult, error)
  1. ValidateRefreshToken
  2. Verify refresh key in Redis still exists
  3. Logout old refresh token
  4. Login again (issue new pair)

CreateUser(ctx, username, password, role string) (*models.SystemUser, error)
  1. Check role is valid: owner, admin, operator
  2. bcrypt.GenerateFromPassword(cost=12)
  3. userRepo.Create

ChangePassword(ctx, userID uint, oldPass, newPass string) error
  1. Load user, verify old password
  2. Hash new password, update

C. IMPLEMENT internal/api/middleware/auth.go (JWT parts)

AuthMiddleware(authSvc *services.AuthService) gin.HandlerFunc:
  1. Extract: Authorization: Bearer <token>
  2. authSvc.ValidateToken(ctx, token)
  3. Set in Gin context: c.Set("userID", claims.UserID)
                         c.Set("username", claims.Username)
                         c.Set("role", claims.Role)
                         c.Set("tokenID", claims.TokenID)
  4. On failure: c.AbortWithStatusJSON(401, AppError)

RequireRole(roles ...string) gin.HandlerFunc:
  1. Get "role" from Gin context
  2. Check against allowed roles
  3. On failure: c.AbortWithStatusJSON(403, AppError)

LoggerMiddleware:
  Before: record start time, request ID (UUID)
  After: log JSON: {method, path, status, latency_ms, client_ip, user_id, request_id}
  Use log/slog (Go 1.21+ standard library).

D. IMPLEMENT internal/api/handlers/auth_handler.go

POST /api/v1/auth/login → no middleware
  Bind JSON: {username, password}
  Call authSvc.Login
  Return: LoginResult

POST /api/v1/auth/refresh → no middleware
  Bind JSON: {refresh_token}
  Call authSvc.RefreshToken
  Return: LoginResult

POST /api/v1/auth/logout → RequireAuth
  Extract tokenID + refreshTokenID from body
  Call authSvc.Logout

GET /api/v1/auth/me → RequireAuth
  Return user from context

PUT /api/v1/auth/password → RequireAuth
  Bind: {old_password, new_password}
  Call authSvc.ChangePassword

POST /api/v1/auth/setup → no middleware (first-run only)
  Call userRepo.Count — if > 0, return 403
  Bind: {username, password}
  Call authSvc.CreateUser(ctx, username, password, "owner")
  Return: created user + login tokens

E. Update internal/api/router.go to wire auth:
  Pass authSvc to middleware and handlers.
  Add /auth/setup, /auth/refresh routes.

F. Verify: go build ./... passes.
   Test: POST /api/v1/auth/setup → POST /api/v1/auth/login → GET /api/v1/auth/me
```

---

### STEP 5 — Router Repository dan Service

```
STEP 5: ROUTER MANAGEMENT

A. IMPLEMENT internal/repository/router_repo.go

type RouterRepository interface {
  Create(ctx, router *models.Router) error
  GetByID(ctx, id uint) (*models.Router, error)
  GetBySessionName(ctx, name string) (*models.Router, error)
  List(ctx) ([]*models.Router, error)
  Update(ctx, router *models.Router) error
  Delete(ctx, id uint) error
}

All GORM queries use db.WithContext(ctx).
Soft delete: use GORM's DeletedAt, queries auto-exclude deleted.
GetBySessionName: explicitly filter deleted_at IS NULL (for partial unique index).

B. IMPLEMENT internal/services/router_service.go

Dependencies:
  - RouterRepository
  - *encrypt.Encryptor (wrap pkg/encrypt functions)
  - *collector.ConnPool (register/unregister router connections)
  - *cache.Cache (for invalidating dashboard cache on router changes)

type RouterPublicView struct {
  ID          uint   `json:"id"`
  SessionName string `json:"session_name"`
  IP          string `json:"ip"`
  Username    string `json:"username"`
  // PasswordEnc NEVER returned
  HotspotName string `json:"hotspot_name"`
  DNSName     string `json:"dns_name"`
  Currency    string `json:"currency"`
  Phone       string `json:"phone"`
  Email       string `json:"email"`
  InfoLP      string `json:"info_lp"`
  IdleTimeout string `json:"idle_timeout"`
  ReportMode  string `json:"report_mode"`
  // Token NEVER returned in list — only used for webhook validation
}

type ConnectionTestResult struct {
  Connected bool          `json:"connected"`
  Latency   int64         `json:"latency_ms"`
  Version   string        `json:"routeros_version"`
  Board     string        `json:"board_model"`
  Identity  string        `json:"identity"`
  Error     string        `json:"error,omitempty"`
}

CreateRouter(ctx, req):
  1. Validate: IP not empty, username not empty, password not empty
  2. TestConnection using IP + credentials (NOT via pool — direct connect)
  3. On connection failure: return error (don't save unreachable router)
  4. Encrypt password: encrypt.Encrypt(password, cfg.AESKey)
  5. Generate router token: 32 random bytes → hex string
  6. Save to DB
  7. Register in ConnPool: pool.Register(domain.RouterConfig{...})
  8. Return RouterPublicView

GetRouter(ctx, id):
  Load from DB → convert to RouterPublicView (strip password)

ListRouters(ctx):
  Load all from DB → convert each to RouterPublicView

UpdateRouter(ctx, id, req):
  1. Load existing router
  2. If IP or credentials changed: test new connection first
  3. If password changed: encrypt new password
  4. Save changes
  5. If IP/credentials changed: pool.Unregister(id) → pool.Register(new config)
  6. cache.Invalidate(ctx, DashboardKey(uint(id)))

DeleteRouter(ctx, id):
  1. pool.Unregister(id)
  2. Soft delete in DB (router.DeletedAt = now)

TestConnection(ctx, ip, username, password string):
  Create a single RouterOS connection with 5s timeout.
  Execute /system/identity/print and /system/resource/print.
  Return: latency, identity, RouterOS version, board model.
  Do NOT register this connection in the pool.

MigrateFromConfigPHP(ctx, filePath string):
  Read config file, parse line by line.
  For each $data[session] block:
    1. Parse IP (after '!')
    2. Parse username (after '@|@')
    3. Parse encrypted password (after '#|#')
    4. Auto-detect format: try DecodeBlah first (V4), fallback DecryptLegacyPHPConfig (V3)
    5. Call CreateRouter (includes connection test + pool registration)
    6. Count success vs skip (already exists) vs error
  Return: MigrationResult{Total, Imported, Skipped, Errors}

C. IMPLEMENT internal/api/handlers/router_handler.go

  POST   /api/v1/routers              → CreateRouter (RequireRole owner, admin)
  GET    /api/v1/routers              → ListRouters (RequireAuth)
  GET    /api/v1/routers/:id          → GetRouter (RequireAuth)
  PUT    /api/v1/routers/:id          → UpdateRouter (RequireRole owner, admin)
  DELETE /api/v1/routers/:id          → DeleteRouter (RequireRole owner)
  POST   /api/v1/routers/:id/test     → TestConnection (body: optional override creds)
  POST   /api/v1/routers/migrate      → MigrateFromConfigPHP (RequireRole owner)

D. STARTUP — Register routers on boot

Update cmd/api/main.go after DB init:
  var routers []models.Router
  db.Find(&routers)
  for _, r := range routers {
    pass, _ := encrypt.Decrypt(r.PasswordEnc, cfg.AESEncKey)
    pool.Register(domain.RouterConfig{
      ID: r.ID, SessionName: r.SessionName,
      Address: r.IP, Username: r.Username, Password: pass,
    })
  }
  pool.Start(ctx) // start streaming + health checks

E. Verify: go build ./... passes.
   Test: create router, list routers, delete router.
```

---

### STEP 6 — Hotspot Service dan Handler

```
STEP 6: HOTSPOT SERVICE AND HANDLER

IMPORTANT READ BEFORE IMPLEMENTING:
- ALL read operations via spec.* functions (Redis-first)
- ALL write operations via tool.* functions
- NO manual cache invalidation for RouterOS data
- Use tool.RemoveHotspotUserWithCleanup (not RemoveHotspotUser) for full cleanup
- Executor is already wired as CachedCommandRunner

A. IMPLEMENT internal/services/hotspot_service.go

Dependencies:
  - *tool.Executor (implements CachedCommandRunner)
  - ProfilePriceRepository (for upsert on profile changes)
  - *cache.Cache (for application cache, NOT RouterOS cache)

Implement 16 methods from existing stub:

ListUsers(ctx, routerID uint, profile string):
  spec.ListHotspotUsers(ctx, executor, routerID, profile)

GetUser(ctx, routerID uint, idOrName string):
  spec.GetHotspotUser(ctx, executor, routerID, idOrName)

GetUserCount(ctx, routerID uint, profile string):
  spec.GetHotspotUserCount(ctx, executor, routerID)
  NOTE: spec returns count-1 (excludes 'admin' user — this is correct behavior)

AddUser(ctx, routerID uint, params domain.AddHotspotUserParams):
  tool.AddHotspotUser(ctx, executor, routerID, params)
  (streaming auto-updates cache after add)

UpdateUser(ctx, routerID uint, id string, params):
  tool.UpdateHotspotUser(ctx, executor, routerID, id, params)

RemoveUser(ctx, routerID uint, id string):
  tool.RemoveHotspotUserWithCleanup(ctx, executor, routerID, id)
  (cleanup: associated scripts, schedulers, etc.)

ListProfiles(ctx, routerID uint):
  profiles, err := spec.ListHotspotProfiles(ctx, executor, routerID)
  For each profile: call tool.ParseOnLoginPut(profile.OnLogin) → metadata
  Return enriched []ProfileWithMeta

GetProfile(ctx, routerID uint, idOrName string):
  profile, err := spec.GetHotspotProfile(ctx, executor, routerID, idOrName)
  metadata := tool.ParseOnLoginPut(profile.OnLogin)
  Return ProfileWithMeta

AddProfile(ctx, routerID uint, params ProfileParams):
  1. Generate on-login script: tool.GenerateOnLoginScript(params)
  2. Set params.OnLogin = generated script
  3. tool.AddHotspotProfile(ctx, executor, routerID, params)
  4. Upsert ProfilePriceMapping in PostgreSQL via profilePriceRepo
  5. Return profile with metadata

UpdateProfile(ctx, routerID uint, id string, params):
  Same flow as AddProfile but use UpdateHotspotProfile

RemoveProfile(ctx, routerID uint, id string):
  tool.RemoveHotspotProfileWithCleanup(ctx, executor, routerID, id)

ListActive(ctx, routerID uint, server string):
  spec.ListActiveSessions(ctx, executor, routerID)
  If server != "": filter by server client-side

RemoveActive(ctx, routerID uint, sessionID string):
  tool.RemoveActiveSession(ctx, executor, routerID, sessionID)

ListHosts(ctx, routerID uint):
  spec.ListHosts(ctx, executor, routerID)

RemoveHost(ctx, routerID uint, mac string):
  tool.Run(ctx, executor, routerID, "/ip/hotspot/host/remove", "=.id="+mac)

ListServers(ctx, routerID uint):
  spec.ListHotspotServers(ctx, executor, routerID)

ListCookies(ctx, routerID uint):
  spec.ListCookies(ctx, executor, routerID)

RemoveCookie(ctx, routerID uint, id string):
  tool.RemoveCookie(ctx, executor, routerID, id)

ListIPBindings(ctx, routerID uint):
  spec.ListIPBindings(ctx, executor, routerID)

AddIPBinding(ctx, routerID uint, params domain.IPBinding):
  tool.Add(ctx, executor, routerID, "/ip/hotspot/ip-binding", params.ToMap())

UpdateIPBinding(ctx, routerID uint, id string, params domain.IPBinding):
  tool.Set(ctx, executor, routerID, "/ip/hotspot/ip-binding", id, params.ToMap())

RemoveIPBinding(ctx, routerID uint, id string):
  tool.RemoveIPBindingWithCleanup(ctx, executor, routerID, id)

EnableIPBinding(ctx, routerID uint, id string):
  tool.EnableIPBinding(ctx, executor, routerID, id)

DisableIPBinding(ctx, routerID uint, id string):
  tool.DisableIPBinding(ctx, executor, routerID, id)

DisconnectUser(ctx, routerID uint, username string):
  tool.DisconnectUser(ctx, executor, routerID, username)
  (removes cookie + active session for that user)

B. IMPLEMENT internal/api/handlers/hotspot_handler.go

All 16+ endpoints. Parse routerID as uint. Bind request body where needed.

Endpoints from existing router.go (keep path structure):
  GET    /hotspot/users?profile=&search= → ListUsers
  POST   /hotspot/users                  → AddUser
  GET    /hotspot/users/:id              → GetUser
  PUT    /hotspot/users/:id              → UpdateUser
  DELETE /hotspot/users/:id              → RemoveUser

  GET    /hotspot/profiles               → ListProfiles
  POST   /hotspot/profiles               → AddProfile
  GET    /hotspot/profiles/:id           → GetProfile
  PUT    /hotspot/profiles/:id           → UpdateProfile
  DELETE /hotspot/profiles/:id           → RemoveProfile

  GET    /hotspot/active?server=         → ListActive
  DELETE /hotspot/active/:id             → RemoveActive (disconnect)
  POST   /hotspot/active/:id/disconnect  → DisconnectUser (clean disconnect)

  GET    /hotspot/hosts                  → ListHosts
  DELETE /hotspot/hosts/:id              → RemoveHost

  GET    /hotspot/servers                → ListServers

  GET    /hotspot/cookies                → ListCookies (NEW route)
  DELETE /hotspot/cookies/:id            → RemoveCookie (NEW route)

  GET    /hotspot/bindings               → ListIPBindings (NEW route)
  POST   /hotspot/bindings               → AddIPBinding (NEW route)
  PUT    /hotspot/bindings/:id           → UpdateIPBinding (NEW route)
  DELETE /hotspot/bindings/:id           → RemoveIPBinding (NEW route)
  POST   /hotspot/bindings/:id/enable    → EnableIPBinding (NEW route)
  POST   /hotspot/bindings/:id/disable   → DisableIPBinding (NEW route)

  GET    /hotspot/users/export?profile=&format=csv|script → ExportUsers

ExportUsers:
  format=script: generate RouterOS script (/ip hotspot user add name=... password=... ...)
  format=csv: return CSV file

Add new routes to internal/api/router.go.

C. ProfileParams struct (for Add/Update profile request body):
  Name, RateLimit, AddressPool, SharedUsers, ParentQueue string
  Price, SellingPrice int64
  Validity string  // e.g. "1d", "7d", "30d"
  ExpireMode string // ntf, ntfc, rem, remc, 0
  LockUser, LockServer bool

D. Verify: go build ./... passes.
```

---

### STEP 7 — Voucher Generation dan Sale Recording

```
STEP 7: VOUCHER AND EVENT HANDLER

A. IMPLEMENT internal/repository/sale_repo.go

type SaleRepository interface {
  Create(ctx, sale *models.VoucherSale) error
  CreateBatch(ctx, sales []*models.VoucherSale) error
  ExistsByIdempotencyKey(ctx, key string) (bool, error)
  GetByDateRange(ctx, routerID uint, from, to time.Time, filters SaleFilters) ([]*models.VoucherSale, error)
  GetByDay(ctx, routerID uint, date time.Time) ([]*models.VoucherSale, error)
  GetByMonth(ctx, routerID uint, year int, month time.Month) ([]*models.VoucherSale, error)
  GetDailySummary(ctx, routerID uint, from, to time.Time) ([]DailySummary, error)
  GetMonthlySummary(ctx, routerID uint, year int) ([]MonthlySummary, error)
  TodayTotal(ctx, routerID uint) (TotalResult, error)
  MonthTotal(ctx, routerID uint) (TotalResult, error)
}

type SaleFilters struct {
  Profile string
  Server  string
  Search  string  // username prefix search
}

type TotalResult struct {
  Count int64 `json:"count"`
  Sum   int64 `json:"sum"`
}

ExistsByIdempotencyKey:
  Use db.Select("1").Where("idempotency_key = ?", key).First(&exists).Error
  Check errors.Is(err, gorm.ErrRecordNotFound) for "not found"

CreateBatch: use db.CreateInBatches(sales, 500) for large imports.

B. IMPLEMENT internal/services/voucher_service.go

Dependencies:
  - *tool.Executor
  - SaleRepository
  - RouterRepository
  - ProfilePriceMappingRepository (read price by profile)
  - *cache.Cache

Implement:

GenerateVoucher(ctx, routerID uint, params VoucherGenerateParams) (*VoucherGenerateResult, error):
  1. Fetch router to verify exists
  2. tool.GenerateVoucherBatch(ctx, executor, routerID, params) — ALREADY HANDLES DEDUP
  3. Store voucher batch in Redis session:
     key = VoucherSessionKey(routerID, params.Gencode)
     cache.SetJSON(ctx, key, batchResult, TTLVoucherSession)
  4. Return: VoucherGenerateResult{Count, Gencode, Profile, Vouchers}

GetCachedVouchers(ctx, routerID uint, gencode string):
  cache.GetJSON(ctx, VoucherSessionKey(routerID, gencode), &batch)

RecordSale(ctx, routerID uint, params RecordSaleParams) error:
  Manual sale recording (for admin use)
  Create VoucherSale directly in DB

ImportSalesFromRouterOS(ctx, routerID uint) (*ImportResult, error):
  1. spec.ImportSalesFromRouterOS(ctx, executor, routerID) → []domain.SalesRecord
  2. For each record:
     a. Generate idempotency_key: SHA256(sessionName + record.Username + record.Date + record.Time)
     b. saleRepo.ExistsByIdempotencyKey → skip if exists
     c. Parse price: strconv.ParseInt(record.Price, 10, 64)
     d. Build VoucherSale{...}
  3. saleRepo.CreateBatch(ctx, newSales)
  4. cache.Invalidate(ctx, SalesKey(routerID, "today"), SalesKey(routerID, "month"))
  5. Return ImportResult{Total, Imported, Skipped, Errors}

C. IMPLEMENT event_handler.go — OnLoginEvent endpoint

POST /api/v1/events/on-login (NO JWT auth — called by RouterOS)

CRITICAL: RouterOS /tool/fetch sends application/x-www-form-urlencoded, NOT JSON.
Use c.PostForm() to read fields, NOT c.BindJSON().

type OnLoginPayload struct {
  RouterSession string  // from c.PostForm("router_session")
  Server        string  // from c.PostForm("server")
  Username      string  // from c.PostForm("username")
  MAC           string  // from c.PostForm("mac")
  IP            string  // from c.PostForm("ip")
  Date          string  // from c.PostForm("date") — "Apr/15/2026"
  Time          string  // from c.PostForm("time") — "10:33:21"
  Profile       string  // from c.PostForm("profile") — added to RouterOS script
}

Security validation:
  1. router := routerRepo.GetBySessionName(ctx, payload.RouterSession)
  2. If router.Token != "" && c.GetHeader("X-Router-Token") != router.Token: return 401
  3. Optional: check client IP matches router.IP

Processing:
  1. Look up ProfilePriceMapping: price = profilePriceRepo.GetByRouterAndProfile(router.ID, profile)
     If not found: use price=0 (record anyway)
  2. Parse sale time: utils.ParseMikroTikDate(payload.Date) + parse payload.Time → time.Time
  3. idempotency_key = SHA256(fmt.Sprintf("%d|%s|%s|%s", router.ID, payload.Username, payload.Date, payload.Time))
  4. saleRepo.ExistsByIdempotencyKey(ctx, key) → skip if exists (return 200 anyway)
  5. saleRepo.Create(ctx, &VoucherSale{
       RouterID: router.ID, Server: payload.Server,
       Username: payload.Username, ProfileName: payload.Profile,
       MACAddress: payload.MAC, IPAddress: payload.IP,
       Price: price.Price, SellingPrice: price.SellingPrice,
       Validity: price.Validity, SoldAt: saleTime,
       IdempotencyKey: key,
     })
  6. cache.Invalidate(ctx, SalesKey(router.ID, "today"), SalesKey(router.ID, "month"), DashboardKey(router.ID))
  7. Return 200 (RouterOS ignores body)
  8. All errors: log but ALWAYS return 200 (RouterOS retry on non-200 causes duplicate events)

ROUTEROS SCRIPT TEMPLATE UPDATE:
The on-login script HTTP fetch call must include profile field.
Update internal/routeros/tool/script.go GenerateOnLoginScript:
Add &profile=[/ip hotspot user get [find where name=$user] profile] to the http-data param.

D. IMPLEMENT internal/api/handlers/voucher_handler.go

  POST /vouchers/generate   → GenerateVoucher (RequireRole owner, admin, operator)
  POST /vouchers/cache      → GetCachedVouchers (requires: routerId + body: {gencode})
  GET  /vouchers/print-data → GetPrintData (fetch template + router info + cached vouchers)
  POST /vouchers/sales      → RecordSale (RequireRole owner, admin)
  POST /vouchers/import     → ImportSalesFromRouterOS (RequireRole owner, admin)

PrintData response:
  {header_template, row_template, footer_template, vouchers, router_info{hotspot_name, dns_name, currency, ...}}

E. Verify: go build ./... passes.
   Test: generate vouchers → cache them → POST on-login event → check sale record.
```

---
### It's Not done yet ###
### STEP 8 — Report Service dan Handler

```
STEP 8: REPORT SYSTEM

Add dependency if not present: go get github.com/xuri/excelize/v2

A. IMPLEMENT internal/services/report_service.go

Dependencies: SaleRepository, *cache.Cache

GetDailyReport(ctx, routerID uint, date time.Time, filters SaleFilters) (*DailyReport, error):
  isPast := date.Before(time.Now().Truncate(24*time.Hour))
  ttl := TTLSalesDay
  if isPast { ttl = TTLSalesPast }
  cacheKey := SalesKey(routerID, "day:"+date.Format("2006-01-02"))
  cache.GetOrSetJSON(ctx, cacheKey, ttl, func() (*DailyReport, error) {
    sales, _ := saleRepo.GetByDay(ctx, routerID, date)
    // apply filters client-side (profile, server, search)
    total := sum of filtered sales Price
    return &DailyReport{Sales: filtered, Total: total, Count: len(filtered)}
  })

GetMonthlyReport(ctx, routerID uint, year, month int) (*MonthlyReport, error):
  Similar cache pattern.
  Fetch via saleRepo.GetDailySummary for chart data.

GetResumeReport(ctx, routerID uint, year int) (*ResumeReport, error):
  From V3: yearly breakdown month by month.
  saleRepo.GetMonthlySummary(ctx, routerID, year)
  Return: []MonthSummary{Month, Count, Total} for 12 months.

GetDashboardSummary(ctx, routerID uint) (*DashboardSummary, error):
  cache.GetOrSetJSON(ctx, DashboardKey(routerID), TTLDashboard, func() (*DashboardSummary, error) {
    today, _ := saleRepo.TodayTotal(ctx, routerID)
    month, _ := saleRepo.MonthTotal(ctx, routerID)
    return &DashboardSummary{TodayCount: today.Count, TodaySum: today.Sum, MonthCount: month.Count, MonthSum: month.Sum}
  })

ExportCSV(ctx, routerID uint, from, to time.Time, filters SaleFilters) ([]byte, error):
  Use encoding/csv.
  Columns: Date, Time, Username, Profile, Price, Validity, MAC, IP, Comment, Server

ExportExcel(ctx, routerID uint, from, to time.Time, filters SaleFilters) ([]byte, error):
  Use excelize.
  Same columns as CSV. Add header row with bold styling.
  Sheet name: "Sales Report"

B. IMPLEMENT internal/api/handlers/report_handler.go

  GET /reports/daily     → query: ?date=2026-04-15&profile=&server=&search=
  GET /reports/monthly   → query: ?year=2026&month=4
  GET /reports/resume    → query: ?year=2026  (V3 feature)
  GET /reports/summary   → dashboard totals (today + month)
  GET /reports/export/csv   → query: ?from=2026-04-01&to=2026-04-30&profile=&server=
  GET /reports/export/excel → same query params

CSV export: set Content-Disposition: attachment; filename="mikhmon-report-{month}.csv"
Excel export: filename="mikhmon-report-{month}.xlsx"

C. Verify: go build ./... passes.
```

---

### STEP 9 — System, PPP, Network, QuickPrint Handlers

```
STEP 9: SYSTEM AND NETWORK HANDLERS

A. CREATE internal/services/system_service.go

Dependencies: *tool.Executor

Methods wrapping existing spec + tool functions:

GetDashboardInfo(ctx, routerID uint) → spec.GetDashboardInfo
GetSystemResource(ctx, routerID uint) → spec.GetSystemResource
GetSystemLog(ctx, routerID uint, limit int) → spec.GetSystemLog
GetSystemClock(ctx, routerID uint) → spec.GetSystemClock
GetSystemIdentity(ctx, routerID uint) → spec.GetSystemIdentity
GetRouterboard(ctx, routerID uint) → spec.GetRouterboard
Reboot(ctx, routerID uint) → tool.Reboot
Shutdown(ctx, routerID uint) → tool.Shutdown
GetExpireMonitorStatus(ctx, routerID uint) → spec.CheckExpireMonitor
DeployExpireMonitor(ctx, routerID uint, interval string) → tool.DeployExpireMonitor
RemoveExpireMonitor(ctx, routerID uint) → tool.RemoveExpireMonitor
ListSchedulers(ctx, routerID uint) → spec.ListSchedulers

B. CREATE internal/services/ppp_service.go

Dependencies: *tool.Executor

ListSecrets(ctx, routerID uint) → spec.ListPPPSecrets (via ScanByMeasurement)
GetSecret(ctx, routerID uint, id string) → spec.GetPPPSecret
AddSecret(ctx, routerID uint, params domain.PPPSecret) → tool.Add /ppp/secret
UpdateSecret(ctx, routerID uint, id string, params) → tool.Set /ppp/secret
RemoveSecret(ctx, routerID uint, id string) → tool.Remove /ppp/secret
ListActive(ctx, routerID uint) → spec.ListPPPActive (via ScanByMeasurement)
DisconnectActive(ctx, routerID uint, id string) → tool.Remove /ppp/active
ListProfiles(ctx, routerID uint) → direct API /ppp/profile/print

C. CREATE internal/api/handlers/system_handler.go

Add routes to router.go:
  sys := protected.Group("/routers/:routerId/system")

  GET  /system/resource     → GetSystemResource (cache 10s via service)
  GET  /system/log          → GetSystemLog?limit=50 (cache 30s)
  GET  /system/clock        → GetSystemClock (no cache)
  GET  /system/identity     → GetSystemIdentity (cache 5min)
  GET  /system/routerboard  → GetRouterboard (cache 5min)
  GET  /system/dashboard    → GetDashboardInfo (combines resource + identity + hotspot counts)
  POST /system/reboot       → Reboot (RequireRole owner, admin — write audit log)
  POST /system/shutdown     → Shutdown (RequireRole owner — write audit log)

  GET  /system/expire-monitor         → GetExpireMonitorStatus
  POST /system/expire-monitor/deploy  → DeployExpireMonitor (body: {interval})
  POST /system/expire-monitor/remove  → RemoveExpireMonitor
  GET  /system/schedulers             → ListSchedulers

D. CREATE internal/api/handlers/ppp_handler.go

  ppp := protected.Group("/routers/:routerId/ppp")
  GET    /ppp/secrets      → ListSecrets
  POST   /ppp/secrets      → AddSecret
  PUT    /ppp/secrets/:id  → UpdateSecret
  DELETE /ppp/secrets/:id  → RemoveSecret
  GET    /ppp/active       → ListActive
  DELETE /ppp/active/:id   → DisconnectActive
  GET    /ppp/profiles     → ListProfiles

E. CREATE internal/api/handlers/network_handler.go

  net := protected.Group("/routers/:routerId/network")
  GET /network/interfaces          → spec.ListInterfaces (cache 60s)
  GET /network/traffic/:iface      → latest from Redis (roskit:{routerID}:interface_traffic:{iface})
  GET /network/pools               → spec.ListIPPools (cache 5min)
  GET /network/queues              → spec.ListParentQueues (cache 5min)
  GET /network/nat                 → spec.ListNATRules (cache 5min)

  Traffic endpoint:
    Read directly from Redis: cacheRepo.GetSnapshot(ctx, formatCacheKey(routerID, "interface_traffic", ifaceName))
    Return cached stats (already written by streaming, updated every 1s)

  DHCP:
  GET /network/dhcp/leases         → spec.ListDHCPLeases via ScanByMeasurement (cache 10s)
  DELETE /network/dhcp/:id/release → tool.Run /ip/dhcp-server/lease/release

F. CREATE internal/api/handlers/quick_print_handler.go

  qp := protected.Group("/routers/:routerId/quick-print")
  GET    /quick-print           → spec.ListQuickPrintPackages
  POST   /quick-print           → tool.SaveQuickPrintPackage
  GET    /quick-print/:name     → spec.GetQuickPrintPackageByName
  PUT    /quick-print/:name     → tool.UpdateQuickPrintPackage
  DELETE /quick-print/:name     → tool.RemoveQuickPrintPackage

G. Verify: go build ./... passes.
```

---

### STEP 10 — Template, WebSocket, Audit

```
STEP 10: TEMPLATES, WEBSOCKET, AND AUDIT

A. IMPLEMENT internal/repository/template_repo.go

type TemplateRepository interface {
  Create(ctx, tmpl *models.PrintTemplate) error
  GetByRouterAndType(ctx, routerID uint, name, part string) (*models.PrintTemplate, error)
  ListByRouter(ctx, routerID uint) ([]*models.PrintTemplate, error)
  Upsert(ctx, tmpl *models.PrintTemplate) error   // create or update
  SeedDefaults(ctx, routerID uint) error           // insert default templates if none exist
}

Default templates: embed original Mikhmon V3 template strings.
Read originals from mikhmon_v4 repository files (view/templates/).
Use Go embed: //go:embed defaults/*.html

B. Template handler

  GET  /routers/:id/templates                → ListByRouter
  GET  /routers/:id/templates/:name/:part    → GetByRouterAndType
  PUT  /routers/:id/templates/:name/:part    → Upsert
  POST /routers/:id/templates/seed           → SeedDefaults (init defaults)
  POST /routers/:id/templates/:name/reset    → Reset to embedded default

C. WebSocket handler

Add dependency: go get github.com/gorilla/websocket

CREATE internal/api/handlers/ws_handler.go

Endpoint: GET /api/v1/ws/:routerId

WebSocket upgrade:
  Upgrader := websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
      // Check origin against cfg.AllowedOrigins
    },
  }

JWT via query param (WebSocket cannot set headers):
  token := c.Query("token")
  claims, err := authSvc.ValidateToken(ctx, token)
  If err: return 401 before upgrade

On connection:
  Subscribe to Redis Pub/Sub channel: utils.FormatPubSubChannel(routerID)
  (This is "roskit:telemetry:{routerID}" — already published by telemetry pipeline)

Message relay loop:
  for msg := range pubsub.Channel() {
    // msg.Payload is JSON TelemetryEvent published by telemetry.go
    ws.WriteMessage(websocket.TextMessage, []byte(msg.Payload))
  }

On disconnect: pubsub.Unsubscribe → pubsub.Close

The telemetry pipeline already publishes events to this channel on every
RouterOS update. This handler only bridges Redis Pub/Sub → WebSocket.

Route: GET /api/v1/ws/:routerId?token=<jwt>

D. Audit Log

CREATE lightweight audit helper in a shared location:

func WriteAudit(ctx context.Context, repo AuditRepository, userID uint, action, entity, entityID, details, ip string) {
  go func() {
    // non-blocking
    log := &models.AuditLog{UserID: userID, Action: action, Entity: entity, EntityID: entityID, Details: details, IPAddress: ip}
    if err := repo.Create(context.Background(), log); err != nil {
      slog.Warn("audit log failed", "err", err)
    }
  }()
}

Add audit writes to:
  - router created/updated/deleted
  - system user created/deleted
  - profile created/updated/deleted
  - router rebooted/shutdown
  - expire monitor deployed/removed
  - voucher batch generated (details: qty + profile)

Add audit endpoints:
  GET /api/v1/admin/audit-logs?user_id=&action=&from=&to=&page=&per_page=
  RequireRole("owner", "admin")

E. System User Management

  GET    /api/v1/admin/users     → userRepo.List (RequireRole owner)
  POST   /api/v1/admin/users     → authSvc.CreateUser (RequireRole owner)
  PUT    /api/v1/admin/users/:id → update role/active (RequireRole owner)
  DELETE /api/v1/admin/users/:id → userRepo.Delete (RequireRole owner)

F. Verify: go build ./... passes.
```

---

### STEP 11 — Background Worker dan Startup

```
STEP 11: BACKGROUND WORKER

IMPLEMENT cmd/worker/main.go properly.
Embed worker functionality into API binary as background goroutines
(simpler to operate than separate binary).

CREATE internal/worker/worker.go:

type Worker struct {
  db          *gorm.DB
  cache       *cache.Cache
  executor    *tool.Executor
  saleRepo    repository.SaleRepository
  profileRepo repository.ProfilePriceMappingRepository
  cfg         *config.Config
}

Background tasks:

1. ProfilePriceSync (every 5 minutes):
   For each router in DB (that is connected in pool):
     Fetch all profiles → tool.ParseOnLoginPut each → upsert ProfilePriceMapping
   Purpose: keep pricing cache in sync without manual trigger.

2. SalesCacheWarmup (on startup, once):
   For each router in DB:
     saleRepo.TodayTotal → set in cache
     saleRepo.MonthTotal → set in cache
   Purpose: first dashboard load is fast.

3. VoucherSessionCleanup (every 1 hour):
   No-op: Redis TTL handles cleanup automatically.
   Just log count of active sessions for monitoring.

Start worker in cmd/api/main.go after all services initialized:
  worker := worker.New(db, cache, executor, saleRepo, profileRepo, cfg)
  go worker.Start(ctx)
```

---

### STEP 12 — Data Migration CLI

```
STEP 12: DATA MIGRATION CLI

Update cmd/migrate/main.go to support multiple subcommands.
Use a simple args-based CLI (no need for cobra, keep it simple):

Usage:
  ./api migrate up
  ./api migrate down
  ./api migrate version
  ./api migrate import --config-file=/data/config.php [--dry-run] [--router=name]
  ./api migrate sync-profiles [--router=name]

A. import command

func runImport(db *gorm.DB, pool *collector.ConnPool, cfg *config.Config, args ImportArgs):
  1. Read config.php flat file
  2. Parse each $data[session] line:
     - Extract IP, username, encrypted password
     - Auto-detect format: try DecodeBlah (V4), fallback DecryptLegacyPHPConfig (V3)
     - Build RouterConfig
  3. If --dry-run: print routers found, don't write
  4. For each router:
     a. Create in DB (skip if session_name already exists)
     b. Fetch all /system/script WHERE comment="mikhmon"
     c. For each script: parse name by "-|-" delimiter
     d. Parse date: utils.ParseMikroTikTime
     e. Generate idempotency_key: SHA256(sessionName + username + date + time)
     f. Skip if exists in DB
     g. Batch insert 500 per tx
  5. Print summary: total routers, imported, skipped, errors

Parse /system/script name format:
  fields := strings.Split(name, "-|-")
  // fields[0]=date, [1]=time, [2]=username, [3]=price, [4]=ip, [5]=mac, [6]=validity, [7]=profile, [8]=comment

B. sync-profiles command

func runSyncProfiles(db *gorm.DB, pool *collector.ConnPool):
  For each router in DB:
    Fetch profiles from RouterOS
    Parse on-login :put metadata via tool.ParseOnLoginPut
    Upsert ProfilePriceMapping in PostgreSQL

C. Data validation output

After each command, print:
  ═══════ Migration Summary ═══════
  Routers found:    X
  Routers imported: X (skipped: X)
  Sales records:    X total, X imported, X skipped (duplicates), X errors
  Profiles synced:  X
  ═══════════════════════════════

D. Verify: go build ./... passes.
   Test dry-run: ./api migrate import --config-file=testdata/config.php --dry-run
```

---

### STEP 13 — Frontend React

```
STEP 13: FRONTEND — mikhmon-ui

Create repository: mikhmon-ui

A. Setup
  npm create vite@latest mikhmon-ui -- --template react-ts
  cd mikhmon-ui
  npm install tailwindcss @tailwindcss/vite
  npm install react-router-dom zustand @tanstack/react-query axios
  npm install recharts
  npm install react-hook-form @hookform/resolvers zod
  npm install lucide-react

B. API Client (src/api/client.ts)

Axios instance:
  baseURL from VITE_API_URL env (default: http://localhost:8080)
  Request interceptor: inject Authorization: Bearer {accessToken}
  Response interceptor:
    On 401: try refresh token → retry original request
    On 401 after refresh: redirect to /login

Typed API modules:
  auth.ts    — login, refresh, logout, me, setup
  routers.ts — CRUD + test + migrate
  hotspot.ts — users, profiles, active, hosts, cookies, bindings, servers
  voucher.ts — generate, cache, printData, recordSale, import
  report.ts  — daily, monthly, resume, summary, exportCSV, exportExcel
  system.ts  — resource, log, dashboard, reboot, shutdown, expireMonitor, schedulers
  ppp.ts     — secrets, active, profiles
  network.ts — interfaces, traffic, pools, queues, nat, dhcpLeases
  ws.ts      — WebSocket connection helper

C. Store (Zustand)

authStore.ts:
  state: user, accessToken, refreshToken
  actions: login, logout, setTokens

routerStore.ts:
  state: selectedRouterId, selectedRouter
  actions: selectRouter

D. Pages

LoginPage.tsx:
  Form: username, password. Call auth.login. Redirect to dashboard.
  Show "First setup" link if /auth/setup returns available.

SetupPage.tsx:
  Only accessible when no users exist.
  Form: username, password, confirm password.
  Call auth.setup → auto-login → redirect dashboard.

DashboardPage.tsx:
  Left: SystemResourceCard, HotspotStatsCard (user count + active count), IncomeCard
  Right: TrafficChart (WebSocket, real-time), HotspotLogCard (30s refresh)
  Use WebSocket hook for real-time updates.

HotspotPage.tsx:
  Tabs: Profiles | Users | Active | Voucher | Hosts | Bindings | Cookies
  
  Profiles tab:
    Table: Name, Rate Limit, Price, Selling Price, Validity, Expire Mode, Users, Actions
    Add/Edit modal with all ProfileParams fields.
  
  Users tab:
    Profile dropdown filter.
    Table: Username, Password (masked/show), Profile, Comment, Status, Actions
    Add/Edit modal. Export button (CSV/Script).
  
  Active tab:
    Server filter. Auto-refresh toggle.
    Table: Username, IP, MAC, Uptime, RX/TX, Server. Disconnect button.
  
  Voucher tab:
    Generate form. Show results table. Print button.
    "Load last batch" button (GetCachedVouchers).
  
  Hosts/Bindings/Cookies: table + CRUD actions.

ReportPage.tsx:
  Tabs: Daily | Monthly | Resume
  Date picker, filters. Summary cards. Paginated table. Export buttons.

LogPage.tsx:
  Auto-refresh toggle. Filter input. Table. Pagination.

PPPPage.tsx:
  Tabs: Secrets | Active | Profiles.

DHCPPage.tsx:
  DHCP leases table with release button.

SettingsPage.tsx:
  Tabs: Router | Expire Monitor | Templates | Users | Audit Log
  
  Router tab: edit all router fields. Test connection button.
  Templates tab: selector (default/small/thermal × header/row/footer), editor, preview.
  Users tab: list system users, add/remove (owner only).
  Audit Log: paginated log table.

PrintVoucherPage.tsx:
  Route: /print-voucher?routerId=&gencode=
  No header/footer (print-optimized layout).
  Fetch print data → render template → print button.
  Template rendering: string replace all $variables with actual values.

E. WebSocket Hook (src/hooks/useWebSocket.ts)

function useRouterWebSocket(routerID: number) {
  Connects to: ws://api/api/v1/ws/{routerID}?token={accessToken}
  Parses incoming messages as TelemetryEvent
  Updates React Query cache on specific measurement types
  Auto-reconnect on disconnect (exponential backoff)
  Returns: { connected, lastEvent }
}

F. Navigation

Desktop: sidebar with icons + labels
Mobile: bottom navigation (Dashboard, Hotspot, Report, Settings)
Header: router selector dropdown + connection status + user menu

G. Build & Env

.env.example:
  VITE_API_URL=http://localhost:8080
  VITE_WS_URL=ws://localhost:8080

docker/nginx.conf: SPA routing (all routes → index.html), /api proxy → api container
```

---

### STEP 14 — Testing

```
STEP 14: TESTING

A. Unit Tests

pkg/encrypt/encrypt_test.go (already specified in Step 3)

internal/routeros/tool/script_test.go:
  Test GenerateOnLoginScript for all 6 expmode × vc/up types.
  Assert: :put metadata format, HTTP fetch presence for remc/ntfc,
          remove/notify logic for rem/ntf, price in fetch URL.

internal/routeros/utils/parser_test.go:
  Test ParseMikroTikTime("Apr/15/2026") → correct time.Time
  Test ParseDuration("1w2d3h4m5s") → correct seconds

B. Integration Tests (testcontainers-go)

internal/repository/sale_repo_integration_test.go:
  Spin up PostgreSQL container.
  Run migrations.
  Test: Create → ExistsByIdempotencyKey → CreateBatch → TodayTotal

internal/api/event_handler_integration_test.go:
  Spin up PostgreSQL + Redis.
  POST /api/v1/events/on-login (form data, not JSON).
  Verify VoucherSale in DB.
  POST same event again — verify no duplicate.
  Verify cache invalidated.

internal/services/auth_service_integration_test.go:
  Spin up PostgreSQL + Redis.
  Login → ValidateToken → Logout → ValidateToken returns 401.
  Refresh token flow.

C. API Handler Tests (httptest)

For each handler, test with mock services:
  401 on missing/invalid JWT
  403 on wrong role
  400 on missing required fields
  200 on valid request

D. Load Test Targets

Document SLOs:
  POST /api/v1/events/on-login: 500 req/s, p99 < 50ms (critical — RouterOS calls this)
  GET /api/v1/routers/:id/hotspot/active: p99 < 100ms (Redis cache hit)
  GET /api/v1/routers/:id/reports/daily: p99 < 300ms (PostgreSQL query)
  WebSocket connections: 200 concurrent with <1% message drop
  Startup time with 20 routers: < 10s (parallel connection init)
```

---

### STEP 15 — Finalisasi

```
STEP 15: FINALIZATION

A. RouterOS on-login script — final version

The updated on-login script HTTP fetch section must use form encoding:
  /tool fetch url="http://BACKEND/api/v1/events/on-login" \
    http-method=post \
    output=none \
    http-data=("router_session=SESSION_NAME&server="+$server+"&username="+$user+"&mac="+$"mac-address"+"&ip="+$address+"&date="+[/system clock get date]+"&time="+[/system clock get time]+"&profile=[/ip hotspot user get [find where name=$user] profile]")

Key notes:
- output=none: prevents blocking login while waiting for response
- $"mac-address": correct RouterOS variable name (NOT $mac)
- $user: correct (NOT $username)
- profile fetched inline via RouterOS expression

RouterOS_Setup.md — write this guide:
  1. What the on-login script does now (vs old version)
  2. How to update: re-save profiles in Mikhmon UI (auto-regenerates script)
  3. Or: use provided bulk-update RouterOS script
  4. Network requirements: RouterOS must reach backend URL (HTTP, not HTTPS unless cert valid)
  5. Security: X-Router-Token header configuration

B. Security Checklist (verify before production)

  [ ] All router passwords encrypted in DB (SELECT password_enc FROM routers — should be base64-like, never plaintext)
  [ ] JWT secrets are random 32+ chars (not default)
  [ ] AES_ENCRYPTION_KEY is random 32 bytes (not default)
  [ ] CORS ALLOWED_ORIGINS does not include * wildcard in production
  [ ] /api/v1/events/on-login: router token validation enabled for each router
  [ ] Rate limiting on /api/v1/auth/login (max 10/min per IP)
  [ ] HTTPS configured (reverse proxy in front)
  [ ] All containers run as non-root
  [ ] PostgreSQL password changed from default
  [ ] Redis AUTH set if Redis port exposed
  [ ] slog level = "info" (not "debug") in production

C. .env.example — complete with all required vars

  # Server
  PORT=8080
  APP_ENV=production

  # Database
  DATABASE_URL=postgres://mikhmon:CHANGE_ME@postgres:5432/mikhmon?sslmode=disable

  # Redis
  REDIS_URL=redis://redis:6379/0

  # InfluxDB (optional, disable by leaving blank)
  INFLUXDB_URL=http://influxdb:8086
  INFLUXDB_TOKEN=
  INFLUXDB_ORG=mikhmon
  INFLUXDB_BUCKET=telemetry

  # Auth
  JWT_SECRET=CHANGE_ME_MIN_32_CHARS_RANDOM
  JWT_REFRESH_SECRET=CHANGE_ME_MIN_32_CHARS_RANDOM_DIFFERENT

  # AES encryption for router passwords (base64-encoded 32-byte key)
  AES_ENCRYPTION_KEY=CHANGE_ME_BASE64_32_BYTES

  # CORS
  ALLOWED_ORIGINS=http://localhost:3000,https://your-domain.com

  # RouterOS Collector
  COLLECTOR_HEALTH_INTERVAL=30s
  COLLECTOR_CONNECTION_TIMEOUT=5s
  COLLECTOR_RECONNECT_MAX_ATTEMPTS=5

D. API Documentation

Use swaggo/swag annotations on all handlers.
Run: swag init -g cmd/api/main.go
Serve at /docs (Swagger UI) when APP_ENV=development.

E. Final build verification

  go build ./...
  go vet ./...
  go test ./...
  docker compose -f docker/docker-compose.yml build
  docker compose -f docker/docker-compose.yml up -d
  curl http://localhost:8080/api/v1/events/health
```

---

## APPENDIX: URUTAN IMPLEMENTASI BERDASARKAN PRIORITAS BISNIS

### Priority 1 — Sistem Berjalan (Steps 1-5)
Setelah ini: router terdaftar, login berfungsi, auth aman.

### Priority 2 — Operasional Hotspot (Steps 6-7)
Setelah ini: bisa manage user, generate voucher, dan menerima event penjualan.

### Priority 3 — Laporan Bisnis (Step 8)
Setelah ini: owner/operator bisa lihat data penjualan.

### Priority 4 — Fitur Lengkap V3 (Step 9)
Setelah ini: PPP, DHCP, system monitor semua berfungsi.

### Priority 5 — Real-time dan Print (Step 10)
Setelah ini: WebSocket dashboard, cetak voucher, template management.

### Priority 6 — Production Ready (Steps 11-15)
Migrasi data lama, frontend, testing, dokumentasi deployment.

---

## APPENDIX: 5 PERINGATAN KRITIS UNTUK AI AGENT

### ⚠️ 1 — Dua cache system terpisah
`internal/routeros/repository/redis.go` (key: `roskit:*`) = RouterOS telemetry cache.
`pkg/redis/cache.go` (key: `mikhmon:*`) = application cache untuk hasil PostgreSQL.
JANGAN pernah menulis RouterOS data ke `mikhmon:*` prefix atau sebaliknya.

### ⚠️ 2 — Form data bukan JSON di event endpoint
RouterOS `/tool/fetch` mengirim `application/x-www-form-urlencoded`.
Handler HARUS pakai `c.PostForm("field")`, BUKAN `c.BindJSON(&payload)`.
Kalau pakai BindJSON, semua event dari RouterOS akan gagal silently.

### ⚠️ 3 — Executor sudah implements CachedCommandRunner
`tool.NewExecutor(engine, cacheRepo)` adalah titik inject yang benar.
Service cukup terima `CachedCommandRunner` interface, bukan `*collector.Engine` langsung.
Dengan ini, spec.* functions bisa pakai Redis-first read otomatis.

### ⚠️ 4 — Cleanup methods untuk Remove operations
Selalu gunakan `WithCleanup` variants:
  `RemoveHotspotUserWithCleanup` — bukan `RemoveHotspotUser`
  `RemoveHotspotProfileWithCleanup` — bukan `RemoveHotspotProfile`
  `RemoveIPBindingWithCleanup` — bukan `tool.Remove /ip/hotspot/ip-binding`
Tanpa ini, orphaned scripts dan schedulers akan menumpuk di router.

### ⚠️ 5 — Jangan invalidate RouterOS cache manual
Streaming telemetry auto-manage semua RouterOS cache dengan TTL 5 menit.
Jika router disconnect, data auto-expire. Tidak perlu — dan tidak boleh —
menulis `cache.Invalidate(routerKey)` untuk data RouterOS di service layer.
Manual invalidation hanya untuk Cache B (mikhmon:* prefix, hasil PostgreSQL).
```