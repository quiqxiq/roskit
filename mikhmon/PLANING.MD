# PROMPT MIGRASI MIKHMON → GO (REVISI v2)
## Berdasarkan Analisis: Mikhmon V3 + V4 + mikhmon-api (Go project yang sudah ada)

---

## KONTEKS PENTING SEBELUM MEMBACA PROMPT

Tiga sumber telah dianalisis secara mendalam:

**Mikhmon V3** — versi PHP mature dengan fitur lengkap: PPP management, DHCP leases,
IP binding, Hotspot cookies, Traffic monitor (Highcharts), i18n (5 bahasa), theming,
Bluetooth print, Quick print, Export user ke CSV/Script, Resume report bulanan.

**Mikhmon V4** — versi PHP yang disederhanakan, sebagian fitur V3 dihilangkan,
konfigurasi dipindah ke format baru, tampilan di-redesign.

**mikhmon-api** — Go project yang sudah berjalan. RouterOS subsystem sudah ~73%
selesai (~3,590 baris). Yang belum diimplementasi: handlers, services, repositories,
auth middleware, encryption, Redis cache wrapper.

**Kesimpulan:** Prompt ini bukan "buat dari nol" — ini adalah prompt untuk
**melanjutkan dan melengkapi project yang sudah ada**, sambil menambahkan fitur
baru dari V3 yang belum ada di V4.

---

## BAGIAN 1: PETA STATUS IMPLEMENTASI

### Sudah Selesai (JANGAN disentuh kecuali ada bug)

```
internal/routeros/
  domain/       — 14 entity types ✓
  collector/    — pool.go (452 ln), router.go (324 ln), engine.go (175 ln) ✓
  spec/         — 16 command specs ✓
  tool/         — executor, hotspot, voucher, script, scheduler, report, system ✓
  usecase/      — telemetry.go, aggregator.go ✓
  repository/   — redis.go, influxdb.go, noop.go ✓
  utils/        — parser.go ✓

internal/models/      — 6 GORM models ✓
migrations/           — 001_initial_schema.sql, 002_indexes.sql ✓
internal/api/router.go — route definitions ✓
internal/api/middleware/auth.go — CORS ✓ (Auth+Logger masih stub)
pkg/database/postgres.go ✓
pkg/redis/client.go ✓
pkg/errors/errors.go ✓
docker/ — Dockerfile, docker-compose.yml, docker-compose.dev.yml ✓
```

### Stub — Harus Diimplementasi

```
internal/api/handlers/   — 6 files, semua stub
internal/services/       — 5 files, semua stub
internal/repository/     — 4 files, semua stub
internal/api/middleware/ — auth.go (Auth+Logger middleware)
pkg/encrypt/encrypt.go   — AES + legacy PHP decrypt
pkg/redis/cache.go       — Cache wrapper
cmd/worker/main.go       — Background worker
```

### Belum Ada — Harus Dibuat

```
internal/models/         — ppp_secret.go, dhcp_lease.go, hotspot_host.go (V3 features)
internal/routeros/tool/  — ppp.go, network.go, dhcp.go
migrations/              — 003_ppp_tables.sql
internal/api/handlers/   — ppp_handler.go, network_handler.go, system_handler.go
internal/services/       — ppp_service.go, system_service.go
frontend/                — React + TypeScript (belum ada sama sekali)
```

---

## BAGIAN 2: FEATURE DELTA (V3 vs V4 vs Target)

| Fitur | V3 PHP | V4 PHP | Target Go | Catatan |
|-------|--------|--------|-----------|---------|
| Hotspot User CRUD | ✓ | ✓ | ✓ already | |
| Hotspot Profile CRUD | ✓ | ✓ | ✓ already | |
| Voucher Generate | ✓ | ✓ | ✓ already | |
| Print Voucher | ✓ | ✓ | implement | |
| Hotspot Active | ✓ | ✓ | ✓ already | |
| Sales Report | ✓ | ✓ | implement | |
| Live Report | ✓ | ✓ | implement | WebSocket |
| Resume Report | ✓ | — | implement | |
| Export CSV/XLS | ✓ | ✓ | implement | |
| Export User to Script | ✓ | — | implement | RouterOS script export |
| Traffic Monitor | ✓ | ✓ | ✓ already | via InterfaceTrafficSpec |
| Hotspot Log | ✓ | ✓ | implement | |
| DHCP Leases | ✓ | — | implement | |
| PPP Secrets | ✓ | — | implement | |
| PPP Active | ✓ | — | implement | |
| PPP Profiles | ✓ | — | implement | |
| IP Binding | ✓ | — | implement | |
| Hotspot Cookies | ✓ | — | implement | |
| Hotspot Hosts | ✓ | ✓ | implement | |
| Scheduler Manager | ✓ | ✓ | ✓ already (tool) | |
| Expire Monitor | ✓ | ✓ | ✓ already (tool) | |
| Multi-router | ✓ | ✓ | implement | |
| Multi-language (i18n) | ✓ 5 lang | — | implement | frontend |
| Theming | ✓ | ✓ | implement | frontend |
| Bluetooth Print | ✓ | — | implement | PWA API |
| Quick Print | ✓ | ✓ | implement | |
| Multi-user + Role | — | — | **NEW** | |
| Audit Log | — | — | **NEW** | |
| WebSocket realtime | — | — | **NEW** | |
| API Token per router | — | — | **NEW** | |
| Dashboard Analytics | — | — | **NEW** | |

---

## BAGIAN 3: ARSITEKTUR TARGET FINAL

```
┌─────────────────────────────────────────────────────────┐
│  React Frontend (Vite + TypeScript)                     │
│  Fitur: Dashboard, Hotspot, PPP, DHCP, Report,         │
│         Traffic, Log, Settings, Print Voucher           │
└───────────────────┬─────────────────────────────────────┘
                    │ REST API + WebSocket
┌───────────────────▼─────────────────────────────────────┐
│  Gin HTTP Server (port 8080)                            │
│  Middleware: CORS → Logger → JWT Auth → Rate Limit      │
└───┬───────────────────────────────────┬─────────────────┘
    │                                   │
┌───▼──────────┐                ┌───────▼──────────────┐
│ Services     │                │ RouterOS Subsystem   │
│ (business    │                │ (SUDAH ADA)          │
│  logic)      │                │                      │
│ [STUB → IMPL]│                │ Engine               │
└───┬──────────┘                │   └── ConnPool       │
    │                           │   └── Collectors     │
┌───▼──────────┐                │       ├── Specs      │
│ Repositories │                │       └── Tools      │
│ (GORM)       │                │ UseCase (Telemetry)  │
│ [STUB → IMPL]│                └──┬───────────────────┘
└───┬──────────┘                   │
    │               ┌──────────────┼──────────────┐
    │               │              │              │
┌───▼───────┐ ┌─────▼─────┐ ┌─────▼──────┐ ┌────▼──────┐
│PostgreSQL │ │  Redis    │ │  InfluxDB  │ │MikroTik   │
│(transaksi,│ │(cache +   │ │(time-series│ │RouterOS   │
│ config,   │ │ pub/sub + │ │ traffic +  │ │TCP 8728   │
│ users)    │ │ sessions) │ │ resources) │ │           │
└───────────┘ └───────────┘ └────────────┘ └───────────┘
        ▲
        │ HTTP POST (on-login webhook)
        │ sumber: RouterOS on-login script
```

---

## BAGIAN 4: DATABASE SCHEMA FINAL

Schema ini **sudah ada di migrations/** untuk 6 tabel utama.
Yang perlu ditambahkan:

```sql
-- TAMBAHAN: 003_v3_features.sql

-- PPP Secrets (untuk fitur PPP dari V3)
-- CATATAN: Data ini tetap di RouterOS, tabel ini HANYA untuk
-- fitur seperti template PPP yang perlu persistensi di backend
-- (tidak mirror data RouterOS)

-- Hotspot Binding (config per-router yang disimpan di backend)
router_binding_configs (
  id, router_id, description, notes, created_at
)
-- Binding actual tetap di RouterOS

-- Perbaikan VoucherSale: tambah kolom yang hilang
ALTER TABLE voucher_sales ADD COLUMN server VARCHAR(100);
ALTER TABLE voucher_sales ADD COLUMN selling_price INT8 DEFAULT 0;
ALTER TABLE voucher_sales ADD COLUMN idempotency_key VARCHAR(200) UNIQUE;

-- Perbaikan PrintTemplate: tambah relasi router
ALTER TABLE print_templates ADD COLUMN router_id INT8 REFERENCES routers(id);
```

---

## BAGIAN 5: PROMPT STEP-BY-STEP

---

### SYSTEM INSTRUCTION (selalu disertakan di setiap sesi)

```
You are continuing development of "mikhmon-api", a Go backend project that is
already partially implemented. DO NOT recreate files that already exist.

CRITICAL RULE — READ BEFORE DOING ANYTHING:
Before modifying or creating any file, ALWAYS read the existing file first
to understand what is already there. The RouterOS subsystem (collector, spec,
tool, domain, usecase, repository) is COMPLETE. Do not touch it unless fixing a bug.

PROJECT CONTEXT:
- Existing Go project: mikhmon-api (Go 1.24, Gin 1.10, GORM 1.25, go-routeros/v3)
- Database: PostgreSQL 16 (6 tables already migrated)
- Cache: Redis 7 (client connected, cache.go is stub)
- Time-series: InfluxDB v3 (batch writer already implemented)
- RouterOS: Connection pool + streaming specs + tools ALL COMPLETE
- Frontend: React + TypeScript + Vite (not yet started, separate repo: mikhmon-ui)

WHAT IS ALREADY IMPLEMENTED (do not recreate):
- internal/routeros/ — entire subsystem, ~3,590 lines, 63 files
- internal/models/ — 6 GORM models
- migrations/ — 001 and 002 SQL files
- internal/api/router.go — all route definitions
- internal/api/middleware/auth.go — CORS middleware (Auth+Logger are stubs)
- docker/ — complete docker configuration
- cmd/api/main.go — server entry point
- cmd/migrate/main.go — migration CLI
- pkg/database/postgres.go, pkg/redis/client.go, pkg/errors/errors.go

WHAT NEEDS IMPLEMENTATION (target of these prompts):
- internal/api/handlers/ — all 6 handlers (stub → full implementation)
- internal/services/ — all 5 services (stub → full implementation)
- internal/repository/ — all 4 repositories (stub → full implementation)
- internal/api/middleware/ — auth.go JWT + logger.go
- pkg/encrypt/encrypt.go — AES-256-GCM + legacy PHP decrypt
- pkg/redis/cache.go — typed cache helpers
- New features from Mikhmon V3: PPP, DHCP, IP Binding, Cookies
- Frontend: complete React application

MIKHMON BUSINESS RULES (from source analysis):
1. RouterOS is source of truth for ALL network operational data
2. Backend stores ONLY: voucher_sales, routers, system_users, audit_logs,
   print_templates, profile_price_mapping
3. Sales records were stored in RouterOS /system/script as:
   name = "date-|-time-|-user-|-price-|-ip-|-mac-|-validity-|-profile-|-comment"
   owner = "MonthYear" (e.g., "Apr2026")
   source = "date" (e.g., "Apr/15/2026")
4. Profile pricing metadata is embedded in the on-login :put statement:
   :put (",{expmode},{price},{validity},{sprice},,{lockuser},{lockserver},")
5. Voucher types: vc (username=password), up (username≠password)
6. All RouterOS commands already have typed wrappers in internal/routeros/tool/

EXISTING TOOL FUNCTIONS (use these, do not reimplement):
- tool.AddHotspotUser, tool.UpdateHotspotUser, tool.AddHotspotProfile, etc. (hotspot.go)
- tool.GenerateVoucherBatch (voucher.go) — already has deduplication
- tool.GenerateOnLoginScript, tool.ParseOnLoginPut (script.go)
- tool.DeployExpireMonitor, tool.GenerateExpireMonitorScript (scheduler.go)
- tool.RecordSale (report.go)
- spec.FetchDailySales, spec.FetchMonthlySales, spec.ImportSalesFromRouterOS (report.go)
- spec.GetSystemResource, spec.GetDashboardInfo, spec.GetSystemLog (system.go)
- spec.ListInterfaces, spec.ListIPPools, spec.ListNATRules, spec.ListParentQueues (network.go)
- spec.CacheVouchers (voucher.go)

CODING STANDARDS:
- Repository pattern: all DB ops via GORM through repository layer
- Service layer: business logic, coordinates repo + routeros/tool
- Handler: thin — validate input, call service, return JSON
- All functions accept context.Context as first param
- Response format: {"data": ..., "error": null} or {"data": null, "error": "msg"}
- Use AppError from pkg/errors for all error responses
- No global variables, no panics without recovery in goroutines
```

---

### STEP 1 — Audit dan Gap Analysis

**Tujuan:** Verifikasi state aktual project sebelum menulis kode.

```
STEP 1: PROJECT AUDIT

Read and verify the existing codebase before any implementation begins.

TASK A: Verify existing implementations
For each file in the "already implemented" list:
1. Confirm the file exists
2. Confirm it compiles without errors (run: go build ./...)
3. Note any compilation errors

TASK B: Inventory all stub functions
For each stub file (handlers, services, repositories, middleware, pkg/encrypt, pkg/redis/cache):
1. List all function signatures
2. Mark what each function needs: DB only, RouterOS only, both, or external
3. Identify all missing imports

TASK C: Identify tool/spec functions needed by each handler
Map each HTTP endpoint (from internal/api/router.go) to:
- Which service it needs
- Which repository calls the service makes
- Which routeros/tool or routeros/spec functions the service calls

TASK D: Schema gaps
Compare current migrations/ SQL with VoucherSale model.
VoucherSale model is missing: server, selling_price, idempotency_key
PrintTemplate model is missing: router_id foreign key
Confirm whether migration 003 is needed.

TASK E: Feature gap list
Cross-reference Mikhmon V3 features against current route definitions.
List features present in V3 but NOT in current route definitions:
- PPP (secrets, active, profiles)
- DHCP leases
- IP binding
- Hotspot cookies
- Router reboot/shutdown
- Ping test
- User export to RouterOS script

Output: A structured gap report with priority ordering.
Do NOT write any implementation code in this step.
Confirm the gap report before proceeding.
```

---

### STEP 2 — Schema Fix dan Migration 003

**Tujuan:** Perbaiki schema yang tidak lengkap.

```
STEP 2: SCHEMA CORRECTIONS

Based on the gap analysis from Step 1, create migration 003.

File: migrations/003_schema_corrections.sql

Changes required:

1. VoucherSale table corrections:
   ALTER TABLE voucher_sales ADD COLUMN IF NOT EXISTS server VARCHAR(100);
   ALTER TABLE voucher_sales ADD COLUMN IF NOT EXISTS selling_price BIGINT NOT NULL DEFAULT 0;
   ALTER TABLE voucher_sales ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(200);
   CREATE UNIQUE INDEX IF NOT EXISTS idx_voucher_sales_idempotency
     ON voucher_sales(idempotency_key) WHERE idempotency_key IS NOT NULL;

2. PrintTemplate table corrections:
   ALTER TABLE print_templates ADD COLUMN IF NOT EXISTS router_id BIGINT
     REFERENCES routers(id) ON DELETE CASCADE;
   ALTER TABLE print_templates ADD COLUMN IF NOT EXISTS part VARCHAR(10)
     CHECK (part IN ('header', 'row', 'footer'));
   UPDATE print_templates SET part = 'header' WHERE part IS NULL;

3. VoucherSale model update (internal/models/voucher_sale.go):
   Add Server, SellingPrice, IdempotencyKey fields with proper GORM tags.

4. PrintTemplate model update (internal/models/print_template.go):
   Add RouterID, Part fields. Add association with Router.

5. Update pkg/database/postgres.go AutoMigrate call if needed.

6. Migration runner update (cmd/migrate/main.go):
   Ensure it runs all 3 migration files in order.

After writing, verify: go build ./... must pass.
```

---

### STEP 3 — Encryption dan Auth Foundation

**Tujuan:** Fondasi keamanan — enkripsi dan JWT.

```
STEP 3: ENCRYPTION AND AUTHENTICATION FOUNDATION

A. IMPLEMENT pkg/encrypt/encrypt.go

The file currently has stubs. Implement all 4 functions:

1. Encrypt(plaintext string, keyBase64 string) (string, error)
   Use AES-256-GCM. Key from base64-decoded env var AES_ENCRYPTION_KEY.
   Output format: base64(nonce + ciphertext)

2. Decrypt(ciphertext string, keyBase64 string) (string, error)
   Reverse of Encrypt.

3. DecryptLegacyPHPConfig(encrypted string) (string, error)
   Decrypts passwords from Mikhmon V3 config.php (XOR key=128 + base64).
   PHP original:
     for each char: result += chr(ord(char) XOR key)
     return base64_encode(result)
   Go equivalent: base64_decode → XOR each byte with 128 → string

4. DecodeBlah(encoded string) (string, error)  
   Decrypts passwords from Mikhmon V4 config.php (triple base64 + XOR key=10).
   Steps: base64_decode → XOR each byte with 10 → base64_decode → base64_decode

B. IMPLEMENT JWT in internal/services/auth_service.go

Use github.com/golang-jwt/jwt/v5 (add to go.mod if not present).

Implement AuthService struct with real logic:

type AuthService struct {
  userRepo  repository.UserRepository
  jwtSecret []byte
  rdb       *redis.Client
}

Methods to implement:

Login(ctx, username, password string) (*LoginResult, error)
  1. userRepo.GetByUsername(ctx, username)
  2. bcrypt.CompareHashAndPassword
  3. Generate JWT: access token (15min) + refresh token (7d)
  4. Store refresh token in Redis: "refresh:{userID}:{tokenID}" with 7d TTL
  5. Return LoginResult{AccessToken, RefreshToken, ExpiresIn, User}

ValidateToken(ctx, tokenString string) (*Claims, error)
  1. Parse JWT with claims
  2. Check if token revoked: Redis "revoked:{tokenID}"
  3. Return Claims{UserID, Username, Role, TokenID}

RefreshToken(ctx, refreshToken string) (*LoginResult, error)
  1. Validate refresh token
  2. Check Redis for refresh token exists
  3. Revoke old refresh token
  4. Issue new access + refresh tokens

Logout(ctx, tokenID, userID string) error
  1. Redis SET "revoked:{tokenID}" = "1" with TTL = token remaining lifetime
  2. Redis DEL "refresh:{userID}:{tokenID}"

CreateUser(ctx, username, password, role string) (*models.SystemUser, error)
  1. Check username not taken
  2. bcrypt.GenerateFromPassword (cost 12)
  3. userRepo.Create

ChangePassword(ctx, userID uint, oldPass, newPass string) error

C. IMPLEMENT internal/api/middleware/auth.go (Auth + Logger parts)

AuthMiddleware (JWT validation):
  1. Extract Bearer token from Authorization header
  2. Call authService.ValidateToken
  3. Set user info in Gin context: "userID", "username", "role"
  4. On failure: return 401 with AppError

RequireRole(roles ...string) gin.HandlerFunc:
  1. Check context "role" against allowed roles
  2. Return 403 if not authorized

LoggerMiddleware:
  Use structured logging with: method, path, status, latency, clientIP, userID (if authenticated)
  Format: JSON for production, colored for development (check cfg.AppEnv)

D. IMPLEMENT internal/repository/user_repo.go

UserRepository interface and PostgreSQL implementation:

type UserRepository interface {
  Create(ctx, user *models.SystemUser) error
  GetByID(ctx, id uint) (*models.SystemUser, error)
  GetByUsername(ctx, username string) (*models.SystemUser, error)
  List(ctx) ([]*models.SystemUser, error)
  Update(ctx, user *models.SystemUser) error
  UpdateLastLogin(ctx, userID uint) error
  Delete(ctx, id uint) error
  Count(ctx) (int64, error)  // for first-run detection
}

E. IMPLEMENT internal/api/handlers/auth_handler.go

All 4 endpoints with real logic:

POST /api/v1/auth/login
  Bind JSON → authService.Login → return tokens + user info

POST /api/v1/auth/logout
  RequireAuth → extract tokenID from claims → authService.Logout

GET /api/v1/auth/me
  RequireAuth → return user from context

PUT /api/v1/auth/password
  RequireAuth → authService.ChangePassword

POST /api/v1/auth/setup (NEW — first run only)
  Check userRepo.Count == 0, else return 403
  Create first owner account
  
After this step: go build ./... must pass.
Test: POST /api/v1/auth/login should return JWT.
```

---

### STEP 4 — Router Repository dan Service

**Tujuan:** Implementasi pengelolaan konfigurasi router.

```
STEP 4: ROUTER MANAGEMENT

A. IMPLEMENT internal/repository/router_repo.go

RouterRepository interface and PostgreSQL implementation using GORM:

type RouterRepository interface {
  Create(ctx, router *models.Router) error
  GetByID(ctx, id uint) (*models.Router, error)
  GetBySessionName(ctx, name string) (*models.Router, error)
  List(ctx) ([]*models.Router, error)
  Update(ctx, router *models.Router) error
  Delete(ctx, id uint) error
}

All GORM calls must use ctx: db.WithContext(ctx).

B. IMPLEMENT internal/services/router_service.go

RouterService depends on:
  - RouterRepository
  - *encrypt.Encryptor (for password handling)
  - *collector.ConnPool (to register/unregister router connections)

Implement all 7 methods:

CreateRouter(ctx, req CreateRouterRequest) (*models.Router, error)
  1. Validate: IP format, required fields
  2. Test connection using routeros (use spec.GetSystemIdentity or similar)
  3. Encrypt password: encryptor.Encrypt(req.Password)
  4. Save to DB
  5. Register in ConnPool: pool.Register(RouterConfig{...})
  6. Write audit log

GetRouter(ctx, id uint) (*RouterPublicView, error)
  NEVER return PasswordEnc. Return RouterPublicView (no password field).

ListRouters(ctx) ([]*RouterPublicView, error)

UpdateRouter(ctx, id uint, req UpdateRouterRequest) (*RouterPublicView, error)
  1. Load existing
  2. If password changed: encrypt new, test connection first
  3. Save changes
  4. Update ConnPool registration if IP/creds changed

DeleteRouter(ctx, id uint) error
  1. Unregister from ConnPool
  2. Soft delete in DB

TestConnection(ctx, ip, username, password string) (*ConnectionTestResult, error)
  Use pool directly or create a temp connection.
  Return: ConnectionTestResult{
    Connected bool, Latency time.Duration,
    RouterOSVersion string, BoardModel string, Identity string
  }

MigrateFromConfigPHP(ctx, filePath string) (*MigrationResult, error)
  Read PHP config file, parse each $data[...] line.
  For V3: use DecryptLegacyPHPConfig (XOR 128)
  For V4: use DecodeBlah (XOR 10 + triple base64)
  Auto-detect which format by trying both.
  Create routers in DB, register in pool.
  Return count of migrated and skipped.

C. IMPLEMENT internal/api/handlers/router_handler.go

All endpoints from router.go:
  GET    /api/v1/routers              → ListRouters
  POST   /api/v1/routers              → CreateRouter
  GET    /api/v1/routers/:id          → GetRouter
  PUT    /api/v1/routers/:id          → UpdateRouter
  DELETE /api/v1/routers/:id          → DeleteRouter
  POST   /api/v1/routers/:id/test     → TestConnection (with body: no changes needed)
  POST   /api/v1/routers/:id/reboot   → NEW: execute /system/reboot via tool.Run
  POST   /api/v1/routers/:id/ping     → NEW: execute /tool/ping via tool.Run
  POST   /api/v1/routers/migrate      → MigrateFromConfigPHP

D. STARTUP: register existing routers on boot

In cmd/api/main.go, after DB and pool are initialized:
  Load all routers from DB → register each in ConnPool → pool.Start(ctx)
  This replaces the PHP behavior of connecting to router on session select.
```

---

### STEP 5 — Redis Cache dan pkg/redis/cache.go

**Tujuan:** Implementasi cache layer yang menggantikan PHP session cache.

```
STEP 5: REDIS CACHE LAYER

A. IMPLEMENT pkg/redis/cache.go

Current file is stub. Implement Cache struct with all methods:

type Cache struct {
  client *redis.Client
}

const (
  TTLProfiles    = 30 * time.Second
  TTLUsers       = 30 * time.Second
  TTLActive      = 5 * time.Second
  TTLResource    = 10 * time.Second
  TTLLog         = 30 * time.Second
  TTLInterfaces  = 60 * time.Second
  TTLPools       = 5 * time.Minute
  TTLQueues      = 5 * time.Minute
  TTLNATRules    = 5 * time.Minute
  TTLServers     = 5 * time.Minute
  TTLSalesToday  = 60 * time.Second
  TTLSalesMonth  = 5 * time.Minute
  TTLDHCPLeases  = 10 * time.Second
  TTLPPPActive   = 5 * time.Second
)

Cache key helpers (consistent naming):
  func RouterKey(routerID uint, resource string) string
  Returns: "mikhmon:router:{routerID}:{resource}"
  
  func SalesKey(routerID uint, period string) string
  Returns: "mikhmon:sales:{routerID}:{period}"

Implement GetOrSet generic function:
  func GetOrSet[T any](
    ctx context.Context,
    c *Cache,
    key string,
    ttl time.Duration,
    fetch func() (T, error),
  ) (T, error)
  
  Logic:
  1. Try Redis GET → unmarshal → return
  2. On miss: call fetch()
  3. Marshal result → Redis SET with TTL
  4. On ANY Redis error: log warning, call fetch() directly (degrade gracefully)
  5. Never return error if fetch() succeeds, even if Redis fails

Implement Invalidate(ctx, keys ...string) error
  Multi-key DELETE in one call.

B. CACHE KEY CONSTANTS

Define all cache keys used across the application:
  Resource data:    RouterKey(id, "resource")
  Profiles:         RouterKey(id, "profiles")
  Users by profile: RouterKey(id, "users:"+profileName)
  Active sessions:  RouterKey(id, "active")
  System log:       RouterKey(id, "log")
  Interfaces:       RouterKey(id, "interfaces")
  IP Pools:         RouterKey(id, "pools")
  Parent Queues:    RouterKey(id, "queues")
  NAT Rules:        RouterKey(id, "nat")
  Hotspot Servers:  RouterKey(id, "servers")
  DHCP Leases:      RouterKey(id, "dhcp_leases")
  PPP Active:       RouterKey(id, "ppp_active")
  Sales today:      SalesKey(id, "today:"+date)
  Sales month:      SalesKey(id, "month:"+monthYear)
  JWT revoked:      "mikhmon:auth:revoked:"+tokenID
  Refresh token:    "mikhmon:auth:refresh:"+userID+":"+tokenID
```

---

### STEP 6 — Hotspot Service dan Handler

**Tujuan:** Implementasi fitur hotspot menggunakan tool functions yang sudah ada.

```
STEP 6: HOTSPOT SERVICE AND HANDLER

IMPORTANT: All RouterOS operations use existing tool functions.
DO NOT reimplement what's already in internal/routeros/tool/hotspot.go
and internal/routeros/spec/*.go

A. IMPLEMENT internal/services/hotspot_service.go

HotspotService depends on:
  - *collector.ConnPool (for RouterOS access)
  - *cache.Cache (for Redis caching)

Implement all 16 methods from the existing stub.
Each method follows this pattern:
  1. Get RouterConn from pool: pool.Get(routerID)
  2. Use cache.GetOrSet where appropriate
  3. Call spec.* or tool.* functions
  4. Invalidate cache on mutations

Key implementations:

ListUsers(ctx, routerID uint, profile string) ([]domain.HotspotUser, error)
  Cache key: RouterKey(routerID, "users:"+profile), TTL: 30s
  Fetch: spec.FetchHotspotUsersByProfile (use existing spec function)

GetUserCount(ctx, routerID uint, profile string) (int, error)
  Use cached list, return len()

AddUser(ctx, routerID uint, params domain.AddHotspotUserParams) (*domain.HotspotUser, error)
  Call tool.AddHotspotUser (already exists in tool/hotspot.go)
  Invalidate: RouterKey(routerID, "users:*")

AddProfile(ctx, routerID uint, params domain.HotspotProfileParams) (*domain.HotspotProfile, error)
  1. Call tool.GenerateOnLoginScript(params) — ALREADY EXISTS in tool/script.go
  2. Set OnLogin field on profile params
  3. Call tool.AddHotspotProfile
  4. Upsert ProfilePriceMapping in PostgreSQL
  5. Invalidate profiles cache

UpdateProfile: same flow as AddProfile but use tool.UpdateHotspotProfile

GetProfileWithParsedMetadata(ctx, routerID uint) ([]ProfileWithMeta, error)
  1. Fetch profiles from RouterOS (cached)
  2. For each profile: call tool.ParseOnLoginPut(profile.OnLogin) — ALREADY EXISTS
  3. Return combined data

B. IMPLEMENT internal/api/handlers/hotspot_handler.go

All 16 endpoints are already defined in router.go.

For each endpoint implement:
  1. Parse routerID from URL param (validate uint)
  2. Bind request body if POST/PUT
  3. Validate required fields
  4. Call service method
  5. Return JSON response

Special considerations:
  - GET /hotspot/users: optional ?profile= query param
  - POST /hotspot/profiles: body must include pricing fields (price, sellingPrice,
    validity, expMode, lockUser, lockServer, rateLimit, sharedUsers, addressPool, parentQueue)
  - POST /hotspot/vouchers/generate: use existing VoucherGenerateParams struct from services

C. VOUCHER HANDLER — implement internal/api/handlers/voucher_handler.go

GenerateVoucher endpoint:
  1. Bind VoucherGenerateParams
  2. Call voucherService.GenerateVoucher
  3. Store generated list in Redis session: "mikhmon:voucher_session:{routerID}:{gencode}"
     TTL: 2 hours (for print preview without regenerating)
  4. Return: count, comment, profile

CacheVoucher endpoint:
  1. Retrieve from Redis session using routerID + comment
  2. Return voucher list (for print preview page)

RecordSale endpoint (manual record):
  POST /api/v1/routers/:routerId/vouchers/sales
  Create VoucherSale record directly (for manual recording)

ImportSales endpoint:
  POST /api/v1/routers/:routerId/vouchers/import
  Calls spec.ImportSalesFromRouterOS — ALREADY EXISTS
  Inserts into voucher_sales table with deduplication via idempotency_key
```

---

### STEP 7 — Voucher Sale Event dan Report

**Tujuan:** Implementasi sales recording dari RouterOS webhook dan reporting.

```
STEP 7: EVENT HANDLER AND REPORT SYSTEM

A. IMPLEMENT internal/api/handlers/event_handler.go

OnLoginEvent endpoint: POST /api/v1/events/on-login
(Called by RouterOS on-login script, NOT by frontend)

Security: This endpoint does not use JWT.
Instead, validate via X-Router-Token header or source IP check.
Router token stored in routers.token field.

Processing logic:
1. Bind request body:
   type OnLoginEventPayload struct {
     RouterSession string `form:"router_session"`
     Server        string `form:"server"`
     Username      string `form:"username"`
     MAC           string `form:"mac"`
     IP            string `form:"ip"`
     Date          string `form:"date"`    // "Apr/15/2026"
     Time          string `form:"time"`    // "10:33:21"
   }
   NOTE: RouterOS /tool/fetch sends as form data, not JSON

2. Security check: routerRepo.GetBySessionName → compare token

3. Parse date: use utils.ParseMikroTikTime — ALREADY EXISTS in routeros/utils/parser.go

4. Look up ProfilePriceMapping:
   profileRepo.GetByRouterAndProfile(ctx, routerID, profile)
   If not found: use price=0 (record anyway, price can be corrected later)
   IMPORTANT: on-login event does NOT include profile name directly.
   Solution: Do a RouterOS lookup to get the user's profile name,
   OR accept profile as additional form field (add to RouterOS script template).
   Recommended: add &profile=... to the RouterOS HTTP fetch call.

5. Generate idempotency_key: SHA256(routerID + username + date + time)

6. Check for duplicate: saleRepo.ExistsByIdempotencyKey → skip if exists

7. Insert VoucherSale record

8. Invalidate Redis cache:
   cache.Invalidate(ctx, SalesKey(routerID, "today:"+todayDate), SalesKey(routerID, "month:"+monthYear))

9. Return 200 (RouterOS ignores response body anyway)

ROUTEROS SCRIPT TEMPLATE UPDATE:
The on-login script must be updated to send profile name.
Update tool/script.go GenerateOnLoginScript to include:
  &profile=[/ip hotspot user get [find where name="$user"] profile]
in the HTTP fetch URL.

B. IMPLEMENT internal/repository/sale_repo.go

SaleRepository interface and GORM implementation:

type SaleRepository interface {
  Create(ctx, sale *models.VoucherSale) error
  CreateBatch(ctx, sales []*models.VoucherSale) error
  ExistsByIdempotencyKey(ctx, key string) (bool, error)
  GetByDateRange(ctx, routerID uint, from, to time.Time) ([]*models.VoucherSale, error)
  GetByDay(ctx, routerID uint, date time.Time) ([]*models.VoucherSale, error)
  GetByMonth(ctx, routerID uint, year int, month time.Month) ([]*models.VoucherSale, error)
  GetDailySummary(ctx, routerID uint, from, to time.Time) ([]DailySummary, error)
  GetMonthlySummary(ctx, routerID uint, year int) ([]MonthlySummary, error)
  GetByProfile(ctx, routerID uint, profileName string, from, to time.Time) ([]*models.VoucherSale, error)
  GetByServer(ctx, routerID uint, server string, from, to time.Time) ([]*models.VoucherSale, error)
  TodayTotal(ctx, routerID uint) (int64, int64, error)    // count, sum
  MonthTotal(ctx, routerID uint) (int64, int64, error)    // count, sum
}

type DailySummary struct {
  Date  time.Time
  Count int64
  Total int64
}

type MonthlySummary struct {
  Year, Month int
  Count       int64
  Total       int64
}

C. IMPLEMENT internal/services/report_service.go

ReportService depends on: SaleRepository, *cache.Cache

GetDailyReport(ctx, routerID uint, date time.Time, filters ReportFilters) (*DailyReport, error)
  Cache key: SalesKey(routerID, "day:"+date.Format("2006-01-02")), TTL: 60s if past, 10s if today
  Return: DailyReport{Sales []VoucherSale, Total int64, Count int64, FilteredTotal int64}

GetMonthlyReport(ctx, routerID uint, year, month int) (*MonthlyReport, error)
  Cache key: SalesKey(routerID, "month:"+monthYear), TTL: 5min if past, 60s if current month
  Return: MonthlyReport{Days []DailySummary, Total int64, Count int64}

GetDashboardSummary(ctx, routerID uint) (*DashboardSummary, error)
  Return: today count + today total + month count + month total
  Cache 60s

ExportCSV(ctx, routerID uint, from, to time.Time) ([]byte, error)
  Format: same columns as Mikhmon V3 report (Date, Time, Username, Profile,
  Price, IP, MAC, Validity, Comment)
  Use encoding/csv

ExportExcel(ctx, routerID uint, from, to time.Time) ([]byte, error)
  Use github.com/xuri/excelize/v2

ImportFromRouterOS(ctx, routerID uint) (*ImportResult, error)
  1. Call spec.ImportSalesFromRouterOS
  2. Parse each SalesRecord
  3. Create VoucherSale with idempotency_key
  4. Batch insert (skip duplicates)
  5. Return: imported count, skipped count, error list

D. IMPLEMENT internal/api/handlers/report_handler.go

  GET /api/v1/routers/:routerId/reports/daily
    Query params: ?date=2026-04-15&profile=&server=
    
  GET /api/v1/routers/:routerId/reports/monthly
    Query params: ?year=2026&month=4
    
  GET /api/v1/routers/:routerId/reports/summary
    Returns: today + this month totals (for dashboard widget)
    
  GET /api/v1/routers/:routerId/reports/resume
    NEW: Monthly resume (from V3) — yearly breakdown
    Query params: ?year=2026
    
  GET /api/v1/routers/:routerId/reports/export/csv
    Query params: ?from=2026-04-01&to=2026-04-30
    Response: Content-Disposition: attachment; filename="report-apr2026.csv"
    
  GET /api/v1/routers/:routerId/reports/export/excel
    Same query params, returns .xlsx

  POST /api/v1/routers/:routerId/reports/import
    Import historical data from RouterOS to PostgreSQL
```

---

### STEP 8 — System Endpoints dan Fitur V3 (PPP, DHCP, Network)

**Tujuan:** Implementasi endpoint system info dan fitur-fitur dari V3 yang belum ada.

```
STEP 8: SYSTEM ENDPOINTS AND V3 FEATURES

A. NEW: internal/api/handlers/system_handler.go

System info endpoints (all use existing spec functions):

GET /api/v1/routers/:id/system/resource
  Call spec.GetSystemResource — ALREADY EXISTS
  Cache: RouterKey(id, "resource"), TTL: 10s

GET /api/v1/routers/:id/system/identity
  Call spec.GetSystemIdentity
  Cache: RouterKey(id, "identity"), TTL: 300s

GET /api/v1/routers/:id/system/clock
  Call spec.GetSystemClock
  No cache (real-time)

GET /api/v1/routers/:id/system/routerboard
  Call spec.GetRouterboard
  Cache: RouterKey(id, "routerboard"), TTL: 300s

GET /api/v1/routers/:id/system/log
  Call spec.GetSystemLog (already exists in spec/system.go)
  Cache: RouterKey(id, "log"), TTL: 30s

GET /api/v1/routers/:id/system/health
  Call spec.GetSystemHealth
  Cache 10s

GET /api/v1/routers/:id/system/dashboard
  Call spec.GetDashboardInfo (combines resource + identity + hotspot counts)
  Cache 10s

POST /api/v1/routers/:id/system/reboot
  RequireRole("owner", "admin")
  tool.Run(pool, routerID, "/system/reboot", nil)
  Write audit log

POST /api/v1/routers/:id/system/shutdown
  RequireRole("owner")
  tool.Run(pool, routerID, "/system/shutdown", nil)
  Write audit log

B. NEW: Network endpoints

GET /api/v1/routers/:id/network/interfaces
  spec.ListInterfaces — ALREADY EXISTS
  Cache TTL: 60s

GET /api/v1/routers/:id/network/traffic/:iface
  spec.InterfaceTrafficSpec — uses streaming, return latest snapshot from Redis
  NO additional cache (already in Redis via telemetry pipeline)

GET /api/v1/routers/:id/network/pools
  spec.ListIPPools — ALREADY EXISTS, Cache 5min

GET /api/v1/routers/:id/network/queues
  spec.ListParentQueues — ALREADY EXISTS, Cache 5min

GET /api/v1/routers/:id/network/nat
  spec.ListNATRules — ALREADY EXISTS, Cache 5min

C. NEW: Scheduler / Expire Monitor endpoints

GET  /api/v1/routers/:id/expire-monitor
  spec.CheckExpireMonitor — verify if "Mikhmon-Expire-Monitor" scheduler exists

POST /api/v1/routers/:id/expire-monitor/deploy
  tool.DeployExpireMonitor — ALREADY EXISTS in tool/scheduler.go

POST /api/v1/routers/:id/expire-monitor/remove
  tool.RemoveExpireMonitor — ALREADY EXISTS

PUT /api/v1/routers/:id/expire-monitor/interval
  tool.SetSchedulerInterval

GET /api/v1/routers/:id/system/schedulers
  spec.ListSchedulers — ALREADY EXISTS

D. NEW: PPP endpoints (from Mikhmon V3)

PPP data lives entirely in RouterOS. Backend only proxies with Redis cache.

Add to internal/api/router.go:
  ppp := protected.Group("/routers/:routerId/ppp")
  ppp.GET("/secrets", pppHandler.ListSecrets)
  ppp.POST("/secrets", pppHandler.AddSecret)
  ppp.PUT("/secrets/:id", pppHandler.UpdateSecret)
  ppp.DELETE("/secrets/:id", pppHandler.RemoveSecret)
  ppp.GET("/active", pppHandler.ListActive)
  ppp.DELETE("/active/:id", pppHandler.DisconnectActive)
  ppp.GET("/profiles", pppHandler.ListProfiles)

Implement internal/api/handlers/ppp_handler.go using:
  - domain.PPPSecret, domain.PPPActive (already exist in domain/)
  - spec.PPPSecretSpec, spec.PPPActiveSpec (streaming, latest in Redis)
  - tool.Run for Add/Update/Remove via:
    /ppp/secret/add, /ppp/secret/set, /ppp/secret/remove
    /ppp/active/remove

Cache: RouterKey(routerID, "ppp_secrets") TTL 30s
       RouterKey(routerID, "ppp_active") TTL 5s
       RouterKey(routerID, "ppp_profiles") TTL 60s

E. NEW: DHCP endpoint (from Mikhmon V3)

GET /api/v1/routers/:id/dhcp/leases
  domain.DHCPLease, spec.DHCPLeaseSpec — BOTH ALREADY EXIST
  Latest snapshot in Redis from telemetry pipeline
  Cache: RouterKey(routerID, "dhcp_leases") TTL 10s

F. NEW: IP Binding, Cookies (from Mikhmon V3)

GET  /api/v1/routers/:id/hotspot/bindings
POST /api/v1/routers/:id/hotspot/bindings
PUT  /api/v1/routers/:id/hotspot/bindings/:id
DEL  /api/v1/routers/:id/hotspot/bindings/:id
  RouterOS path: /ip/hotspot/ip-binding/*
  Cache TTL: 30s

GET /api/v1/routers/:id/hotspot/cookies
DEL /api/v1/routers/:id/hotspot/cookies/:id
  RouterOS path: /ip/hotspot/cookie/*
  Cache TTL: 30s

G. NEW: User Export to RouterOS Script (from Mikhmon V3)

GET /api/v1/routers/:id/hotspot/users/export
  Query params: ?profile=xxx&format=csv|script
  
  For format=script:
    Fetch users from RouterOS (with profile filter)
    Generate RouterOS script:
      /ip hotspot user add name=xxx password=xxx profile=xxx ...
    Return as text/plain with Content-Disposition attachment

  For format=csv:
    Return CSV with same structure as V3 export
```

---

### STEP 9 — WebSocket Real-Time dan Template Voucher

**Tujuan:** WebSocket untuk live data dan template management.

```
STEP 9: WEBSOCKET AND TEMPLATE MANAGEMENT

A. WebSocket untuk Real-Time Data

Add gorilla/websocket to go.mod.

Implement internal/api/handlers/ws_handler.go:

Connection endpoint: GET /api/v1/ws/:routerId

On connect:
  1. Upgrade to WebSocket
  2. Validate JWT from query param (?token=...) — WebSocket cannot set headers
  3. Subscribe to Redis Pub/Sub channels:
     - utils.FormatPubSubChannel(routerID, "hotspot_active")
     - utils.FormatPubSubChannel(routerID, "hotspot_user")
     - utils.FormatPubSubChannel(routerID, "system_resource")
     - utils.FormatPubSubChannel(routerID, "interface_traffic")
  4. Forward published events to WebSocket client as JSON

Message format sent to client:
  {
    "type": "hotspot_active" | "system_resource" | etc,
    "routerID": 1,
    "data": {...}
  }

Handle disconnect: unsubscribe from Redis, close goroutines.

Note: The telemetry pipeline already publishes to Redis Pub/Sub
via internal/routeros/repository/redis.go (PubSubPublisher).
This handler only needs to bridge Redis Pub/Sub → WebSocket.

B. TEMPLATE REPOSITORY DAN SERVICE

IMPLEMENT internal/repository/template_repo.go:

type TemplateRepository interface {
  Create(ctx, tmpl *models.PrintTemplate) error
  GetByRouterAndType(ctx, routerID uint, name, part string) (*models.PrintTemplate, error)
  ListByRouter(ctx, routerID uint) ([]*models.PrintTemplate, error)
  Update(ctx, tmpl *models.PrintTemplate) error
  ResetToDefault(ctx, routerID uint, name, part string) error
  SeedDefaults(ctx, routerID uint) error
}

Default templates must match Mikhmon V3 originals (header/row/footer × default/small/thermal).
Store defaults as embedded strings in the service layer.

IMPLEMENT template service and handler:

GET  /api/v1/routers/:id/templates
GET  /api/v1/routers/:id/templates/:name/:part
PUT  /api/v1/routers/:id/templates/:name/:part
POST /api/v1/routers/:id/templates/:name/reset
POST /api/v1/routers/:id/templates/seed          (initialize defaults)

Template variables (for reference, not validation):
$username, $password, $profile, $price, $sellingprice, $validity,
$expired, $comment, $server, $ip, $mac, $qrcode, $logo, $hotspotname,
$dnsname, $currency, $phone, $email, $infolp, $date, $time

C. Print Voucher endpoint

GET /api/v1/routers/:id/vouchers/print-data
  Query: ?comment=... (gencode comment to identify the voucher batch)
  
  1. Retrieve voucher batch from Redis session
  2. Get templates from DB (or defaults)
  3. Get router metadata (hotspot name, DNS, currency, etc.)
  4. Return: VoucherPrintData{Header, Row template, Footer, Vouchers[], RouterInfo}
  5. Frontend handles actual template rendering and print dialog

D. Audit Log

ADD audit logging to key operations:
Create a lightweight AuditLogger helper:

  func LogAction(ctx context.Context, repo AuditRepository, userID uint, 
                 action, entity, entityID, details, ip string) 
  (non-blocking: run in goroutine, log error if fails)

Operations to audit:
  - router created/updated/deleted
  - hotspot profile created/updated/deleted  
  - system user created/deleted
  - router rebooted/shutdown
  - expire monitor deployed
  - bulk voucher generated (qty + profile)

IMPLEMENT internal/repository/audit_repo.go:
  Create(ctx, log *models.AuditLog) error
  List(ctx, filters AuditFilters) ([]*models.AuditLog, error)

Add audit log endpoint:
  GET /api/v1/admin/audit-logs
  RequireRole("owner", "admin")
  Query params: ?user_id=&action=&from=&to=&page=&per_page=
```

---

### STEP 10 — Frontend React

**Tujuan:** Build complete React frontend.

```
STEP 10: FRONTEND — REACT + TYPESCRIPT + VITE

Create new repository: mikhmon-ui

A. PROJECT SETUP

Tech stack:
- Vite + React 18 + TypeScript
- TailwindCSS v3
- React Router v6 (SPA routing)
- Zustand (state management — lightweight)
- React Query / TanStack Query v5 (server state, caching, refetch)
- Axios (HTTP client, interceptors for JWT)
- Recharts (traffic charts, sales charts)
- React Hook Form + Zod (form validation)

Directory structure:
src/
  api/
    client.ts         Axios instance, interceptors, token refresh
    auth.ts
    routers.ts
    hotspot.ts
    voucher.ts
    report.ts
    system.ts
    ppp.ts
    network.ts
  components/
    ui/               base: Button, Input, Modal, Table, Badge, Spinner
    layout/           Sidebar, MobileNav, RouterSelector, Header
    hotspot/
    voucher/
    report/
    dashboard/
    settings/
  pages/
    LoginPage.tsx
    SetupPage.tsx     (first run)
    DashboardPage.tsx
    HotspotPage.tsx
    ReportPage.tsx
    LogPage.tsx
    PPPPage.tsx       (from V3)
    DHCPPage.tsx      (from V3)
    SettingsPage.tsx
    PrintVoucherPage.tsx
  hooks/
    useAuth.ts
    useRouter.ts      (selected router context)
    useWebSocket.ts
  store/
    authStore.ts      (Zustand: user, tokens)
    routerStore.ts    (Zustand: selected router)
  types/
    api.ts            (all API response types)
    routeros.ts       (RouterOS domain types)
  utils/
    formatters.ts     (currency, date, bytes)
    template.ts       (voucher template renderer)

B. AUTHENTICATION FLOW

- Login page: shown when no JWT in localStorage
- First-run setup page: shown when /api/v1/auth/setup returns 200 (no users)
- Token refresh: Axios interceptor catches 401 → refresh → retry original request
- Route protection: PrivateRoute component wraps all authenticated pages

C. ROUTER SELECTOR

Global context: selected router persists in localStorage + Zustand store.
RouterSelector dropdown in header shows all configured routers.
All data pages use selected router ID.
Show connection status dot (green/yellow/red) per router.

D. DASHBOARD PAGE

Two-column layout (desktop), single column (mobile).

Left column:
  - SystemResourceCard: board model, RouterOS version, CPU gauge, RAM gauge, uptime
  - HotspotStatsCard: registered users count, active now count
  - IncomeCard: today (count + total) + this month (count + total)

Right column:
  - TrafficChart: real-time TX/RX via WebSocket, line chart (Recharts)
    Updates every 1s from WebSocket
  - HotspotLogCard: latest log entries, auto-refresh 30s

E. HOTSPOT PAGE

Tabs: Profiles | Users | Active | Voucher | Hosts | Bindings | Cookies

Profiles tab:
  Table columns: Name, Rate Limit, Price, Selling Price, Validity, Expire Mode,
                 Shared Users, Address Pool, Users Count, Actions
  Add Profile modal: full form matching all backend fields
  Edit Profile modal: same form pre-filled
  Delete: confirmation dialog

Users tab:
  Profile filter dropdown at top
  Table: Username, Password (hidden/show), Profile, Limit Uptime, Limit Bytes,
         Comment (expire date), Status, Actions
  Quick actions: copy username+password
  Add/Edit/Remove user
  Export button: dropdown (CSV, RouterOS Script)

Active tab:
  Auto-refresh toggle (WebSocket or 5s polling)
  Table: Username, IP, MAC, Uptime, RX/TX, Server
  Disconnect button per row
  Filter by server

Voucher tab:
  Form: Profile (dropdown), Type (vc/up), Qty, Char Set, Length, Prefix,
        Time Limit, Data Limit, Comment, Server
  Generate button → show result table
  Print button → navigate to /print-voucher
  Regenerate last batch button (uses cache endpoint)

Hosts, Bindings, Cookies: simple table + delete per row

F. REPORT PAGE

Layout:
  Top: date range picker, profile filter, server filter, search input
  Middle: summary cards (qty, total income)
  Main: paginated sales table
  Footer: export buttons (CSV, Excel)

Sales table columns: Date, Time, Username, Profile, Price, Validity, MAC, IP, Comment

Resume tab (from V3):
  Yearly view: month-by-month total sales
  Chart: bar chart (Recharts) per month

G. LOG PAGE

Auto-refresh toggle (30s)
Filter input
Table: Time, Topics, Message
Pagination (50 per page)

H. PPP PAGE (from V3)

Tabs: Secrets | Active | Profiles

Secrets: Name, Password, Profile, Service, IP, Comment
Active: Name, Caller-ID, Address, Uptime, Service, Disconnect button
Profiles: Name, Rate Limit, Local/Remote Address

I. DHCP PAGE (from V3)

DHCP Leases table:
  Address, MAC, Client-ID, Hostname, Status, Expires, Server
  Release button per row

J. SETTINGS PAGE

Tabs: Router Settings | Admin Account | Expire Monitor | Template Editor | Theme

Router Settings form:
  IP, Username, Password (masked), Hotspot Name, DNS Name, Currency,
  Phone, Email, Info LP, Idle Timeout, Report Mode
  Test Connection button (live feedback with ping latency)
  Upload Logo button
  Save button

Admin Account:
  List system users (owner only)
  Change password for own account
  Add/remove users (owner only)
  Role assignment

Expire Monitor:
  Status card (installed / not installed)
  Deploy / Remove buttons
  Interval adjustment

Template Editor (from V3):
  Tab selector: default / small / thermal
  Part selector: header / row / footer
  Code editor (textarea with monospace font)
  Variable reference sidebar
  Preview button (renders with sample data)
  Save / Reset to default buttons

Theme:
  Color theme selector
  Language selector (prepare UI for i18n even if only 1 lang initially)

K. PRINT VOUCHER PAGE

Route: /print-voucher (separate page, no header/footer — print-optimized)

Query params: ?routerId=&comment=

1. Fetch print data from API
2. Render template: replace variables in header/row/footer template
3. Show print preview
4. Print button: window.print()
5. Bluetooth print button (if device supports — use Web Bluetooth API)

L. MOBILE RESPONSIVE

Bottom navigation (mobile): Dashboard | Hotspot | Voucher | Report | Settings
Sidebar navigation (desktop)
All tables must be horizontally scrollable on mobile
Touch-friendly: minimum tap target 44px
```

---

### STEP 11 — Historical Data Migration

**Tujuan:** Import transaksi lama dari RouterOS ke PostgreSQL.

```
STEP 11: DATA MIGRATION CLI

Update cmd/migrate/main.go to support subcommands:

Commands:
  up       — run SQL migrations
  down     — rollback
  version  — show current version
  import   — import data from RouterOS/config.php

A. import subcommand

Flags:
  --config-file  PATH  Path to Mikhmon V3/V4 config.php
  --router-id    ID    Only migrate specific router (optional)
  --dry-run           Show what would be imported without writing

Steps:
  1. Parse config.php (try V4 format first, fallback to V3 format)
  2. For each router session:
     a. Decrypt password (auto-detect V3 XOR-128 vs V4 XOR-10)
     b. Test RouterOS connection
     c. If connection OK: fetch all /system/script with comment=mikhmon
     d. Parse each script name: fields = strings.Split(name, "-|-")
     e. Parse date using utils.ParseMikroTikTime
     f. Build idempotency_key: SHA256(session_name + username + date + time)
     g. Skip if already exists
     h. Batch insert 500 records per transaction
  3. Print summary per router

B. sync-profiles subcommand

For each router in DB:
  1. Fetch hotspot profiles from RouterOS
  2. For each profile: tool.ParseOnLoginPut → ProfilePriceMapping
  3. Upsert into DB

C. Data validation

After import, verify:
  - Compare count: /system/script mikhmon entries vs DB voucher_sales
  - Print: total imported, total skipped (duplicates), errors

D. Clean up RouterOS (optional, with confirmation)

--cleanup flag: after successful import, remove /system/script entries
  (keeps RouterOS clean, since data is now in PostgreSQL)
  Only execute if dry-run passed and counts match.
```

---

### STEP 12 — Background Worker

**Tujuan:** Worker untuk task periodik.

```
STEP 12: BACKGROUND WORKER

Implement cmd/worker/main.go properly (currently stub).

The worker runs alongside the API server (separate process or goroutine in API).

Tasks:

A. RouterOS Collector Start
   On startup: load all routers from DB → register in collector.Engine → start
   This is the streaming/telemetry pipeline for real-time data.
   
B. Periodic Profile Price Sync
   Interval: every 5 minutes
   For each connected router:
     Fetch all profiles → parse on-login metadata → upsert ProfilePriceMapping
   This keeps pricing data in sync without manual intervention.

C. Router Health Monitor
   Already handled by ConnPool health checker (pool.go).
   Worker just ensures pool is started and handles reconnect logging.

D. Today's Sales Cache Warm-up
   On startup: for each router, fetch today's sales from DB and warm Redis cache.
   So first dashboard load is fast.

E. Stale Session Cleanup
   Interval: every 1 hour
   Clean up expired voucher print sessions from Redis:
   Pattern: "mikhmon:voucher_session:*" — delete if TTL expired naturally.
   (Redis TTL handles this, but explicit cleanup for monitoring)

Worker can either:
  a. Run as separate binary (cmd/worker/main.go): uses same DB + Redis
  b. Run as goroutines in API binary (recommended for simplicity)

Recommendation: embed in API binary using a worker manager:
  type Worker struct { db, redis, engine, config }
  func (w *Worker) Start(ctx context.Context)
```

---

### STEP 13 — Testing

**Tujuan:** Test coverage untuk komponen kritis.

```
STEP 13: TESTING

A. Unit tests (no external dependencies)

internal/routeros/tool/script_test.go
  Test GenerateOnLoginScript for ALL combinations:
  - expMode: ntf, ntfc, rem, remc, 0+price, 0 no price
  - lockUser: Enable/Disable
  - lockServer: Enable/Disable
  Assert: :put metadata line format, expiry logic presence, record line presence/absence
  Assert: HTTP fetch line present in remc/ntfc modes (new backend URL)

internal/routeros/tool/voucher_test.go
  Test GenerateVoucherBatch deduplication
  Test all 10 character modes generate correct length and charset

internal/routeros/utils/parser_test.go
  Test ParseMikroTikTime: "Apr/15/2026 10:33:21" → correct UTC time
  Test ParseDuration: "1w2d3h" → correct time.Duration

pkg/encrypt/encrypt_test.go
  Test Encrypt → Decrypt roundtrip
  Test DecryptLegacyPHPConfig: known PHP-encrypted string → known plaintext
  Test DecodeBlah: known V4-encrypted string → known plaintext

B. Integration tests (testcontainers-go)

Test file: internal/repository/sale_repo_integration_test.go
  Use testcontainers to spin up real PostgreSQL.
  Test: Create sale → ExistsByIdempotencyKey → CreateBatch dedup

Test file: internal/api/event_handler_integration_test.go
  Full flow: POST /api/v1/events/on-login → verify VoucherSale in DB → verify idempotency

Test file: internal/services/auth_service_test.go
  Use testcontainers for Redis + PostgreSQL
  Test: Login → ValidateToken → Logout → ValidateToken returns unauthorized

C. API handler tests (httptest, mock services)

Test each handler independently using mock service implementations.
Test: invalid JWT → 401, wrong role → 403, missing fields → 400, success → 200.

D. Performance test specification

Document expected load characteristics:
  - POST /api/v1/events/on-login: target 500 req/s sustained, p99 < 50ms
  - GET /api/v1/routers/:id/active: p99 < 100ms (cached)
  - GET /api/v1/routers/:id/reports/daily: p99 < 300ms (PostgreSQL query)
  - WebSocket connections: 1000 concurrent with <1% message loss
```

---

### STEP 14 — Finalisasi dan Dokumentasi

**Tujuan:** Production-ready dengan dokumentasi lengkap.

```
STEP 14: FINALIZATION AND DOCUMENTATION

A. API Documentation

Generate OpenAPI 3.0 spec.
Two options:
  1. Use swaggo/swag annotations directly in handler code (recommended)
  2. Write swagger.yaml manually
  
Serve Swagger UI at /docs in non-production mode.

B. RouterOS Integration Guide

Write RouterOS_Setup.md covering:

1. The updated on-login script format (with HTTP POST to backend):
   The key section that replaces $record:
   /tool fetch url="http://BACKEND_HOST:8080/api/v1/events/on-login" \
     http-method=post \
     http-data=("router_session=ROUTER_SESSION_NAME&server="+$server+"&username="+$user+"&mac="+$"mac-address"+"&ip="+$address+"&date="+$date+"&time="+[/system clock get time]+"&profile=[/ip hotspot user get [find where name=$user] profile]")

2. How to update existing profiles:
   Option A: Re-save each profile from Mikhmon UI (auto-generates new on-login)
   Option B: RouterOS script to update all profiles at once

3. Network requirements: router must be able to reach backend URL

4. Security: X-Router-Token header, what it is, how to configure it

C. Deployment Guide (README.md)

Step-by-step:
1. Clone both repos: mikhmon-api and mikhmon-ui
2. Configure .env (all required vars with explanations)
3. docker compose up
4. First-run setup: access /setup, create owner account
5. Add first router: Settings → Add Router, Test Connection, Save
6. Migrate historical data: docker exec mikhmon-api ./api migrate import --config-file /data/config.php
7. Sync profiles: docker exec mikhmon-api ./api migrate sync-profiles
8. Verify dashboard shows data
9. Update RouterOS: run provided update script, or re-save profiles one by one

D. .env.example update

# Server
PORT=8080
APP_ENV=production   # development | production

# Database
DATABASE_URL=postgres://mikhmon:password@postgres:5432/mikhmon?sslmode=disable

# Redis
REDIS_URL=redis://redis:6379/0

# InfluxDB (optional, for traffic time-series)
INFLUXDB_URL=http://influxdb:8086
INFLUXDB_TOKEN=your-token
INFLUXDB_ORG=mikhmon
INFLUXDB_BUCKET=telemetry

# Auth
JWT_SECRET=minimum-32-character-random-secret-here
JWT_EXPIRY_MINUTES=15
JWT_REFRESH_EXPIRY_DAYS=7

# Encryption (AES-256 for router passwords)
AES_ENCRYPTION_KEY=base64-encoded-32-byte-key-here

# CORS
ALLOWED_ORIGINS=http://localhost:3000,https://your-domain.com

# RouterOS Collector
COLLECTOR_HEALTH_INTERVAL=30s
COLLECTOR_RECONNECT_MAX_ATTEMPTS=5
COLLECTOR_CONNECTION_TIMEOUT=5s

E. Security Checklist

Pre-production verification:
[ ] AES_ENCRYPTION_KEY is unique random value (not default)
[ ] JWT_SECRET is minimum 32 chars, random
[ ] Router passwords encrypted in DB (verify: no plaintext in routers table)
[ ] CORS ALLOWED_ORIGINS does not include * in production
[ ] HTTPS configured (nginx/traefik in front of API)
[ ] /api/v1/events/on-login: router token validation active
[ ] Rate limiting active on /api/v1/auth/login
[ ] go build -ldflags="-s -w" for binary (strip debug symbols)
[ ] Docker containers run as non-root user
[ ] Postgres password not default "changeme"
[ ] Redis AUTH password set if exposed
[ ] Log level set to "info" (not "debug") in production
```

---

## PANDUAN PENGGUNAAN DENGAN CLAUDE CODE

### Cara Optimal

Setiap step dijalankan sebagai satu sesi terpisah di Claude Code.
Sebelum setiap step, tempel ulang System Instruction di awal konteks.

**Urutan yang direkomendasikan untuk hasil tercepat:**

```
Phase 1 — Core foundation (wajib sebelum lainnya):
  STEP 1 (audit) → STEP 2 (schema fix) → STEP 3 (auth)

Phase 2 — Backend fungsional:
  STEP 4 (router) → STEP 5 (cache) → STEP 6 (hotspot)

Phase 3 — Business critical:
  STEP 7 (event + report) → STEP 8 (system + V3 features)

Phase 4 — Frontend:
  STEP 10 (React)

Phase 5 — Production ready:
  STEP 9 (WebSocket + templates) → STEP 11 (migration)
  → STEP 12 (worker) → STEP 13 (testing) → STEP 14 (docs)
```

### Checkpoint Wajib Setelah Setiap Step

```bash
# Setelah setiap step backend, pastikan ini lulus:
go build ./...
go vet ./...

# Setiap step yang menyentuh handler, test endpoint ini:
curl -s http://localhost:8080/api/v1/events/health | jq .

# Setelah STEP 3 (auth):
curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"test"}' | jq .
```

### Peringatan Spesifik untuk AI Agent

Lima kesalahan yang paling sering terjadi pada skenario ini:

1. **Agent membuat ulang routeros/ subsystem** — Harus selalu dicek
   dengan membaca file yang ada sebelum membuat apapun di folder itu.

2. **Agent lupa bahwa tool.ParseOnLoginPut sudah ada** — Sering diimplementasi
   ulang di service layer. Cukup import dan panggil.

3. **Agent menggunakan form data biasa untuk event endpoint** — RouterOS
   `/tool/fetch` mengirim sebagai `application/x-www-form-urlencoded`,
   bukan JSON. Handler harus bind form, bukan JSON.

4. **Agent membuat Redis cache blocking** — `GetOrSet` harus degrade gracefully
   jika Redis tidak tersedia, bukan return error.

5. **Agent tidak memperbarui go.mod** — Setelah menambahkan library baru
   (jwt, excelize, websocket), harus jalankan `go mod tidy`.
```