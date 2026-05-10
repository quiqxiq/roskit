# Rencana Simplifikasi: Hapus Multi-Tenant → Single-Instance Multi-Router

## Latar Belakang

Saat ini sistem menggunakan arsitektur multi-tenant penuh (SaaS) yang kompleks:
- Setiap resource (Router, User, VoucherSale) di-scope ke `tenant_id`
- Ada `Tenant` model, `TenantSettings`, `TenantService`, middleware Casbin dengan domain tenant
- Auth menggunakan JWT yang menyertakan `tenant_id` dan `tenant_slug`
- Role: `superadmin`, `owner`, `admin`, `staff`

**Tujuan perubahan:** Simplifikasi menjadi single-instance app (satu instalasi = satu bisnis) dengan multi-router. Tidak ada lagi konsep "tenant" — hanya ada satu set konfigurasi (settings), satu set user, dan banyak router.

---

## Pertanyaan Terbuka

> [!IMPORTANT]
> Perlu konfirmasi sebelum eksekusi:

1. **Role apa yang dipertahankan?**
   - Saran: `admin` (akses penuh) + `staff` (operasional harian saja). `owner` dan `superadmin` dihapus karena tidak relevan tanpa multi-tenant.

2. **Casbin — tetap atau dihapus?**
   - Saran: Casbin bisa **dihapus** dan diganti middleware RBAC sederhana berbasis field `role` di JWT. Ini menghilangkan kompleksitas policy + gorm-adapter.

3. **Data lama — migrasi atau fresh start?**
   - Jika ada data production yang perlu dipertahankan, diperlukan SQL migration script.
   - Jika fresh start, cukup drop + recreate schema.

4. **Webhook token untuk on-login RouterOS** saat ini disimpan di `tenant_settings.webhook_token`. Setelah perubahan disimpan di `settings.webhook_token`. Apakah script RouterOS yang sudah terpasang perlu diupdate?

---

## Gambaran Arsitektur Baru

```
Sebelum:                          Sesudah:
─────────────────────────────     ─────────────────────────────
Tenant (ID, name, slug, plan)     ← HAPUS
  └── TenantSettings              → Settings (satu baris singleton)
  └── User (tenant_id FK)         → User (tanpa tenant_id)
  └── Router (tenant_id FK)       → Router (tanpa tenant_id)
  └── VoucherSale (tenant_id FK)  → VoucherSale (tanpa tenant_id)
  └── AuditLog (tenant_id FK)     → AuditLog (tanpa tenant_id)
  └── PrintTemplate (tenant_id)   → PrintTemplate (global only)

Auth JWT sebelum:                 Auth JWT sesudah:
  { uid, role, tid, tslug }       { uid, role }

Middleware stack sebelum:         Middleware stack sesudah:
  Auth → Tenant → Casbin          Auth → RoleGuard (simple)
```

---

## Perubahan yang Diusulkan

---

### Layer 1: Models (`internal/models/`)

#### [DELETE] `tenant.go`
Hapus seluruh file. `PlatformTenantSlug` juga dihapus.

#### [DELETE] `tenant_settings.go`
Hapus, diganti dengan file baru.

#### [NEW] `settings.go`
```go
type Settings struct {
    ID           uint      // primaryKey
    HotspotName  string
    DNSName      string
    Currency     string    // default: "Rp"
    Phone        string
    Email        string
    InfoLP       string
    IdleTimeout  int       // default: 30
    ReportMode   string    // "disable" | "enable"
    WebhookToken string    // json:"-"
    LogoPath     string
    Timezone     string
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```
Tabel: `settings` (singleton — selalu ID=1, dibuat via AutoMigrate + seed).

#### [MODIFY] `user.go`
- Hapus field `TenantID *uint`
- Hapus `UserRoleSuperAdmin` dan `UserRoleOwner`
- Pertahankan: `UserRoleAdmin = "admin"`, `UserRoleStaff = "staff"`
- Ubah unique index menjadi `uniqueIndex:idx_users_username,where:deleted_at IS NULL`

#### [MODIFY] `router.go`
- Hapus field `TenantID uint`
- Ubah unique index `idx_routers_tenant_name` → `idx_routers_name`

#### [MODIFY] `voucher_sale.go`
- Hapus field `TenantID uint`
- Pertahankan `RouterID *uint`

#### [MODIFY] `audit_log.go`
- Hapus field `TenantID *uint`

#### [MODIFY] `print_template.go`
- Hapus field `TenantID *uint` (semua template menjadi global)

---

### Layer 2: Repository (`internal/repository/`)

#### [DELETE] `tenant_repo.go`
#### [DELETE] `tenant_settings_repo.go`

#### [NEW] `settings_repo.go`
```go
type SettingsRepository interface {
    Get(ctx context.Context) (*models.Settings, error)
    GetByWebhookToken(ctx context.Context, token string) (*models.Settings, error)
    Upsert(ctx context.Context, s *models.Settings) error
    UpdateLogo(ctx context.Context, logoPath string) error
}
```
Implementasi `Get()` selalu mengambil baris dengan ID=1 (singleton).

#### [MODIFY] `user_repo.go`
- Hapus `GetByTenantID()`, `GetByTenantUsername()`, `GetSuperAdminByUsername()`, `List(tenantID)`, `CountByTenant()`
- Ganti dengan:
  - `GetByUsername(ctx, username)` — tanpa tenant scope
  - `List(ctx)` — semua user
  - `Count(ctx)` — tetap ada

#### [MODIFY] `router_repo.go`
- Hapus parameter `tenantID` dari semua method: `GetByID(ctx, id)`, `GetByName(ctx, name)`, `List(ctx)`, `Delete(ctx, id)`
- Hapus `GetByIDAny()` (tidak perlu lagi — semua akses tanpa tenant scope)
- Hapus `UpdateTimezone()` dari sini (pindah ke `SettingsRepo`)

#### [MODIFY] `sale_repo.go`
- Hapus parameter `tenantID` dari semua method di interface dan implementasi
- `scopedQuery()` hanya filter by `routerID` (opsional)

#### [MODIFY] `template_repo.go`
- Hapus semua filter `tenant_id`

---

### Layer 3: Services (`internal/services/`)

#### [DELETE] `tenant_service.go`

#### [NEW] `settings_service.go`
Menggantikan `TenantService` untuk operasi settings:
```go
type SettingsService struct { ... }

func (s *SettingsService) Get(ctx) (*models.Settings, error)
func (s *SettingsService) Update(ctx, req UpdateSettingsRequest) (*models.Settings, error)
func (s *SettingsService) UploadLogo(ctx, data []byte) (string, error)
func (s *SettingsService) GetLogoPath(ctx) (string, error)
```

#### [MODIFY] `auth_service.go`
- Hapus `TenantLookup` interface (tidak diperlukan)
- Hapus `tenantRepo` field dari `AuthService`
- Ubah `Login(ctx, tenantSlug, username, password)` → `Login(ctx, username, password)`
- Hapus `tenantID`/`tenantSlug` dari `Claims` struct
- Hapus `TenantID`/`TenantSlug` dari `UserView` struct
- Ubah `CreateUser()` — tidak ada tenant parameter
- Hapus `ErrTenantNotFound`, `ErrTenantSuspended`, `ErrTenantRequired`

#### [MODIFY] `router_service.go`
- Hapus `tenantID` dari semua method: `CreateRouter`, `GetRouter`, `ListRouters`, `UpdateRouter`, `DeleteRouter`
- Hapus pengecekan tenant di `MigrateFromConfigPHP`

#### [MODIFY] `hotspot_service.go`
- Hapus `tenantID` dari `AddProfile`, `UpdateProfile`, `SyncProfiles`, `resolveOnLoginParams`
- `resolveOnLoginParams` mengambil settings dari `SettingsRepo.Get()` bukan `GetByTenantID()`
- `SyncProfiles` mengambil settings dari `SettingsRepo.Get()`

#### [MODIFY] `voucher_service.go`
- Hapus `tenantID` dari `RecordSale`, `ImportSalesFromRouterOS`, `GenerateVoucher`, `GetRouterInfo`
- `resolveProfilePrice` tidak perlu tenantID

#### [MODIFY] `report_service.go`
- Hapus `tenantID` dari semua method
- `scopedQuery` hanya by `routerID`
- Cache key tidak perlu prefix `t{tenantID}:`

---

### Layer 4: Casbin (`internal/casbin/`)

#### [DELETE] Seluruh package `internal/casbin/`
Casbin dihapus sepenuhnya. Diganti middleware RBAC sederhana.

---

### Layer 5: Middleware (`internal/api/middleware/`)

#### [DELETE] `tenant.go`
#### [DELETE] `casbin.go`

#### [NEW] `role.go`
Middleware RBAC sederhana berbasis role dari JWT:
```go
// RequireRole menolak request jika role user tidak ada dalam daftar yang diizinkan.
func RequireRole(roles ...models.UserRole) gin.HandlerFunc {
    return func(c *gin.Context) {
        roleVal, _ := c.Get("role")
        role, _ := roleVal.(models.UserRole)
        for _, r := range roles {
            if role == r {
                c.Next()
                return
            }
        }
        c.AbortWithStatusJSON(403, gin.H{"data": nil, "error": "forbidden"})
    }
}
```

#### [MODIFY] `auth.go`
- Hapus `jwtTenantID` dan `jwtTenantSlug` dari context sets
- `Claims` yang disimpan hanya `userID`, `username`, `role`, `tokenID`

#### [MODIFY] `rate_limit.go`
Tidak berubah.

#### [MODIFY] `audit.go`
- Hapus `tenantID` dari `Log()` — tidak perlu di-set ke context

---

### Layer 6: Handlers (`internal/api/handlers/`)

#### [DELETE] `tenant_handler.go`

#### [NEW] `settings_handler.go`
Menggantikan sebagian `tenant_handler.go`:
```go
// GET  /api/v1/settings
// PUT  /api/v1/settings
// POST /api/v1/settings/logo
// GET  /api/v1/settings/logo
```

#### [MODIFY] `auth_handler.go`
- `loginRequest` hapus field `Tenant`
- `Setup()` tidak perlu create tenant + owner, cukup create admin user pertama
- `setupRequest` disederhanakan — hanya `username` + `password`
- Hapus `tenantSvc` dari `AuthHandler`

#### [MODIFY] `user_handler.go`
- Hapus `tenantIDFromCtx()` calls
- `List()` mengembalikan semua user
- `Create()` tidak perlu tenant scope
- Hapus Casbin role assignment dari `Update()` dan `Delete()`

#### [MODIFY] `router_handler.go`
- Hapus `tenantID` dari semua panggilan service

#### [MODIFY] `hotspot_handler.go`
- Hapus `tenantID` dari `SyncProfiles`, `AddProfile`, `UpdateProfile`

#### [MODIFY] `voucher_handler.go`
- Hapus `tenantID` dari semua panggilan service
- `PrintData` mengambil settings dari `SettingsRepo` tanpa tenantID

#### [MODIFY] `report_handler.go`
- Hapus `tenantID` dari semua panggilan service

#### [MODIFY] `event_handler.go`
- `OnLoginEvent` mengambil `settings` via `SettingsRepo.GetByWebhookToken()` — menghapus `tenantID` dari alur

#### [MODIFY] `context.go`
- Hapus `tenantIDFromCtx()` dan `optionalTenantIDFromCtx()`

---

### Layer 7: Router (`internal/api/router.go`)

Perubahan signifikan:
- Hapus `TenantMiddleware`, `CasbinMiddleware` dari stack middleware
- Hapus semua parameter `tenantRepo`, `tenantSettingsRepo`, `tenantSvc`, `enforcer` dari `NewRouter()`
- Ganti grup `/admin` dengan endpoint `/settings`
- Pasang `RequireRole(models.UserRoleAdmin)` sebagai middleware untuk endpoint yang butuh admin
- Route structure baru:
  ```
  /api/v1/auth/login    (public)
  /api/v1/auth/refresh  (public)
  /api/v1/auth/logout   (auth only)
  /api/v1/auth/me       (auth only)
  /api/v1/auth/password (auth only)
  /api/v1/auth/setup    (public, hanya sebelum user ada)
  /api/v1/health        (public)
  /api/v1/status        (public)
  /api/v1/events/*      (public, webhook)

  /api/v1/settings      (auth + RequireRole(admin))
  /api/v1/users         (auth + RequireRole(admin))
  /api/v1/routers       (auth)
  /api/v1/routers/:id/* (auth, role check per endpoint)
  /api/v1/templates     (auth)
  ```

---

### Layer 8: DI Wiring (`cmd/api/main.go`)

- Hapus: `tenantRepo`, `tenantSettingsRepo`, `tenantSvc`, `enforcer` + Casbin init
- Tambah: `settingsRepo`, `settingsSvc`
- Sederhanakan `NewRouter()` signature

---

### Layer 9: Roskit Internal (`internal/roskit/`)

> [!NOTE]
> Berdasarkan grep, **`internal/roskit/` tidak mengandung referensi ke `tenantID`** sama sekali. Subsistem ini sudah dirancang murni berdasarkan `routerID` (string). **Tidak ada perubahan yang diperlukan di dalam `internal/roskit/`.**

Yang berubah hanya dari luar roskit (service/handler layer):
- `EventHandler.OnLoginEvent` tidak lagi pass `tenantID` ke sale record
- `RouterRepo.UpdateTimezone()` tidak lagi sub-query ke `tenant_id`

---

### Layer 10: Database Migration

Script SQL yang diperlukan (jika ada data lama):

```sql
-- 1. Buat tabel settings dari tenant_settings yang sudah ada
CREATE TABLE settings AS
SELECT id, hotspot_name, dns_name, currency, phone, email,
       info_lp, idle_timeout, report_mode, webhook_token,
       logo_path, timezone, created_at, updated_at
FROM tenant_settings
LIMIT 1;

-- 2. Hapus tenant_id dari semua tabel
ALTER TABLE users DROP COLUMN tenant_id;
ALTER TABLE routers DROP COLUMN tenant_id;
ALTER TABLE voucher_sales DROP COLUMN tenant_id;
ALTER TABLE audit_logs DROP COLUMN tenant_id;
ALTER TABLE print_templates DROP COLUMN tenant_id;

-- 3. Hapus tabel lama
DROP TABLE tenant_settings;
DROP TABLE tenants;
DROP TABLE casbin_rule; -- tabel gorm-adapter
```

Untuk fresh start: GORM AutoMigrate akan menangani pembuatan semua tabel baru.

---

## Rencana Verifikasi

### Automated
```bash
go build ./...                    # harus pass
go test -race ./...               # harus pass
```

### Manual
1. `POST /api/v1/auth/setup` → buat user admin pertama
2. `POST /api/v1/auth/login` → login tanpa field `tenant`
3. `GET /api/v1/settings` → baca settings (sebelumnya `/api/v1/tenant/settings`)
4. `GET /api/v1/users` → list user tanpa tenant scope
5. `GET /api/v1/routers` → list semua router

### Perubahan API yang Breaking
| Endpoint Lama | Endpoint Baru |
|---|---|
| `POST /auth/login` body `{tenant, username, password}` | body `{username, password}` |
| `POST /auth/setup` body `{tenant_name, tenant_slug, username, password}` | body `{username, password}` |
| `GET /tenant` | `GET /settings` |
| `PUT /tenant` | `PUT /settings` |
| `GET /tenant/settings` | `GET /settings` (merged) |
| `PUT /tenant/settings` | `PUT /settings` (merged) |
| `POST /tenant/logo` | `POST /settings/logo` |
| `GET /tenant/logo` | `GET /settings/logo` |
| `GET /admin/tenants` | ← HAPUS |
| `POST /admin/tenants` | ← HAPUS |

---

## Perkiraan Scope Perubahan

| Area | File Dihapus | File Dimodifikasi | File Baru |
|---|---|---|---|
| Models | 2 | 5 | 1 |
| Repository | 2 | 4 | 1 |
| Services | 1 | 5 | 1 |
| Casbin | 3 (seluruh pkg) | — | — |
| Middleware | 2 | 2 | 1 |
| Handlers | 1 | 8 | 1 |
| Router | — | 1 | — |
| cmd/api | — | 1 | — |
| **Total** | **11** | **26** | **5** |
