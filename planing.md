# Roskit — Multi-Tenant SaaS + Casbin RBAC
## Planning Document

---

## Ringkasan Keputusan Desain

| Topik | Keputusan | Alasan |
|---|---|---|
| Super admin akses tenant | Header `X-Tenant-Slug` per request | Lebih fleksibel dari re-login, tidak perlu generate token baru per tenant |
| Super admin storage | Tabel `users`, `tenant_id = NULL` | Satu tabel auth, satu flow login, tidak perlu tabel terpisah |
| Global template | `tenant_id = NULL` di `print_templates`, di-copy saat tenant dibuat | Copy lebih simpel dari inheritance, tidak ada dependency runtime ke global |
| Template lama `router_id = 0` | **Dihapus** — ini hack, bukan design | NULL lebih eksplisit dan aman dari angka sentinel |
| Cascade delete | Tenant hard delete → semua CASCADE. Router soft delete, data sales tetap ada | Sales adalah data finansial, jangan ikut terhapus kalau satu router dihapus |
| Casbin super admin | Policy `p, superadmin, *, /api/v1/*, *` + bypass domain check | Konsisten dengan domain-based model, tidak perlu special-case di middleware |

---

## Phase 1 — Database Schema

### Struktur Relasi

```
tenants (1)
├── tenant_settings (1:1, CASCADE)
├── users (1:N, CASCADE)         ← tenant_id nullable, NULL = superadmin
├── routers (1:N, CASCADE)
│   └── profile_price_mappings (1:N, CASCADE dari router)
├── voucher_sales (1:N, CASCADE) ← tenant_id langsung, bukan via router
├── print_templates (1:N, CASCADE) ← tenant_id nullable, NULL = global default
└── audit_logs (1:N, CASCADE)
```

---

### Model: `Tenant`

```go
// internal/models/tenant.go
package models

import (
    "time"
    "gorm.io/gorm"
)

type TenantStatus string
type TenantPlan   string

const (
    TenantStatusActive    TenantStatus = "active"
    TenantStatusTrial     TenantStatus = "trial"
    TenantStatusSuspended TenantStatus = "suspended"

    TenantPlanFree    TenantPlan = "free"
    TenantPlanStarter TenantPlan = "starter"
    TenantPlanPro     TenantPlan = "pro"
)

type Tenant struct {
    ID        uint           `gorm:"primaryKey"                   json:"id"`
    Name      string         `gorm:"size:100;not null;uniqueIndex" json:"name"`
    // Slug dipakai sebagai domain identifier di Casbin dan URL.
    // Format: lowercase, hanya huruf/angka/strip. Contoh: "bintang-net"
    Slug      string         `gorm:"size:100;not null;uniqueIndex" json:"slug"`
    Plan      TenantPlan     `gorm:"type:varchar(20);not null;default:free" json:"plan"`
    Status    TenantStatus   `gorm:"type:varchar(20);not null;default:active" json:"status"`
    CreatedAt time.Time      `                                    json:"created_at"`
    UpdatedAt time.Time      `                                    json:"updated_at"`
    // DeletedAt menggunakan soft delete. Hard delete dilakukan secara eksplisit
    // via service layer untuk memastikan semua CASCADE berjalan sebelum baris tenant dihapus.
    DeletedAt gorm.DeletedAt `gorm:"index"                       json:"-"`

    Settings *TenantSettings `gorm:"foreignKey:TenantID;constraint:OnDelete:CASCADE" json:"settings,omitempty"`
    Users    []User          `gorm:"foreignKey:TenantID;constraint:OnDelete:CASCADE" json:"-"`
    Routers  []Router        `gorm:"foreignKey:TenantID;constraint:OnDelete:CASCADE" json:"-"`
}
```

**Catatan hard delete tenant:**
Karena GORM soft delete tidak men-trigger FK CASCADE, proses hapus tenant harus:
1. Set `status = suspended` (soft, bisa di-undo)
2. Jika benar-benar ingin hapus: `db.Unscoped().Delete(&tenant)` — ini trigger CASCADE ke semua child table

---

### Model: `TenantSettings`

```go
// internal/models/tenant_settings.go
type TenantSettings struct {
    ID           uint      `gorm:"primaryKey"           json:"id"`
    TenantID     uint      `gorm:"not null;uniqueIndex;constraint:OnDelete:CASCADE" json:"tenant_id"`
    HotspotName  string    `gorm:"size:100"             json:"hotspot_name"`
    DNSName      string    `gorm:"size:100"             json:"dns_name"`
    Currency     string    `gorm:"size:10;not null;default:Rp" json:"currency"`
    Phone        string    `gorm:"size:20"              json:"phone"`
    Email        string    `gorm:"size:100"             json:"email"`
    InfoLP       string    `gorm:"type:text"            json:"info_lp"`
    IdleTimeout  int       `gorm:"not null;default:30"  json:"idle_timeout"`
    ReportMode   string    `gorm:"size:20;not null;default:disable" json:"report_mode"`
    WebhookToken string    `gorm:"size:255"             json:"-"`
    LogoPath     string    `gorm:"type:text;not null;default:''" json:"logo_path"`
    Timezone     string    `gorm:"size:50;not null;default:''" json:"timezone"`
    CreatedAt    time.Time `                             json:"created_at"`
    UpdatedAt    time.Time `                             json:"updated_at"`
}
```

---

### Model: `User` (rename dari `SystemUser`)

`tenant_id` nullable: `NULL` berarti user ini adalah platform-level (superadmin).

Uniqueness:
- User biasa: `(tenant_id, username) WHERE deleted_at IS NULL` — composite partial
- Superadmin: `username WHERE tenant_id IS NULL AND deleted_at IS NULL` — partial terpisah

```go
// internal/models/user.go
type UserRole string

const (
    UserRoleOwner      UserRole = "owner"      // full access dalam tenant
    UserRoleAdmin      UserRole = "admin"       // manage semua kecuali tenant settings
    UserRoleStaff      UserRole = "staff"       // operasional harian
    UserRoleSuperAdmin UserRole = "superadmin"  // akses semua tenant, platform-level
)

type User struct {
    ID           uint           `gorm:"primaryKey" json:"id"`
    // TenantID nullable: NULL = superadmin (platform-level user, tidak terikat tenant)
    TenantID     *uint          `gorm:"index;constraint:OnDelete:CASCADE" json:"tenant_id"`
    // Composite uniqueIndex: (tenant_id, username) WHERE deleted_at IS NULL
    // Untuk superadmin (tenant_id NULL): username unik di antara sesama superadmin
    TenantID2    *uint          `gorm:"column:tenant_id;uniqueIndex:idx_users_tenant_username,where:deleted_at IS NULL" json:"-"`
    Username     string         `gorm:"size:100;not null;uniqueIndex:idx_users_tenant_username,where:deleted_at IS NULL" json:"username"`
    PasswordHash string         `gorm:"size:255;not null" json:"-"`
    Role         UserRole       `gorm:"type:varchar(20);not null;default:staff" json:"role"`
    Active       bool           `gorm:"not null;default:true" json:"active"`
    LastLoginAt  *time.Time     `                             json:"last_login_at"`
    CreatedAt    time.Time      `                             json:"created_at"`
    UpdatedAt    time.Time      `                             json:"updated_at"`
    DeletedAt    gorm.DeletedAt `gorm:"index"                json:"-"`
}
```

**Catatan:** GORM composite uniqueIndex dengan nullable field memerlukan perhatian khusus.
Cara yang benar untuk composite partial index dengan nullable:
```go
TenantID *uint  `gorm:"uniqueIndex:idx_users_tenant_username,where:deleted_at IS NULL"`
Username string `gorm:"uniqueIndex:idx_users_tenant_username,where:deleted_at IS NULL"`
```
PostgreSQL akan memperlakukan NULL sebagai nilai unik tersendiri dalam unique index,
sehingga multiple superadmin (tenant_id=NULL) tetap harus punya username berbeda.

---

### Model: `Router`

Hapus relasi `HotspotConfig`. Tambah `TenantID`.

```go
// internal/models/router.go
type Router struct {
    ID                   uint           `gorm:"primaryKey" json:"id"`
    TenantID             uint           `gorm:"not null;index;constraint:OnDelete:CASCADE;uniqueIndex:idx_routers_tenant_name,where:deleted_at IS NULL" json:"tenant_id"`
    Name                 string         `gorm:"size:100;not null;uniqueIndex:idx_routers_tenant_name,where:deleted_at IS NULL" json:"name"`
    IPAddress            string         `gorm:"size:45;not null"          json:"ip_address"`
    APIPort              int            `gorm:"not null;default:8728"     json:"api_port"`
    APIUsername          string         `gorm:"size:100;not null"         json:"api_username"`
    APIPasswordEncrypted string         `gorm:"column:password;not null"  json:"-"`
    SSHPort              *int           `gorm:"default:22"                json:"ssh_port"`
    SSHUsername          *string        `gorm:"size:50"                   json:"ssh_username"`
    SSHPasswordEncrypted *string        `gorm:"column:ssh_password"       json:"-"`
    Status               RouterStatus   `gorm:"type:varchar(20);not null;default:unknown" json:"status"`
    LastSeenAt           *time.Time     `                                 json:"last_seen_at"`
    Notes                *string        `gorm:"type:text"                 json:"notes"`
    CreatedAt            time.Time      `                                 json:"created_at"`
    UpdatedAt            time.Time      `                                 json:"updated_at"`
    DeletedAt            gorm.DeletedAt `gorm:"index"                    json:"-"`
}
```

---

### Model: `VoucherSale`

Tambah `TenantID` langsung (denormalisasi) untuk query laporan cross-router per-tenant.
`RouterID` tetap ada tapi `ON DELETE SET NULL` — kalau router dihapus, data sales tidak hilang.

```go
// internal/models/voucher_sale.go
type VoucherSale struct {
    ID             uint      `gorm:"primaryKey"                                      json:"id"`
    TenantID       uint      `gorm:"not null;index;constraint:OnDelete:CASCADE"       json:"tenant_id"`
    // RouterID SET NULL jika router dihapus — data penjualan tetap ada untuk laporan
    RouterID       *uint     `gorm:"index;constraint:OnDelete:SET NULL"               json:"router_id"`
    SoldAt         time.Time `gorm:"not null;index"                                  json:"sold_at"`
    Username       string    `gorm:"size:100;not null;index"                         json:"username"`
    ProfileName    string    `gorm:"size:100;not null;index"                         json:"profile_name"`
    Price          int64     `gorm:"not null;default:0"                              json:"price"`
    SellingPrice   int64     `gorm:"not null;default:0"                              json:"selling_price"`
    Server         string    `gorm:"size:100;index"                                  json:"server"`
    IPAddress      string    `gorm:"size:45"                                         json:"ip_address"`
    MACAddress     string    `gorm:"size:17"                                         json:"mac_address"`
    Validity       string    `gorm:"size:20"                                         json:"validity"`
    IdempotencyKey string    `gorm:"type:char(64);not null;uniqueIndex"              json:"idempotency_key"`
    CreatedAt      time.Time `                                                        json:"created_at"`
}
```

**Kenapa `RouterID` nullable + SET NULL, bukan CASCADE?**
Router bisa di-soft-delete atau bahkan di-hard-delete tanpa harus menghilangkan
riwayat penjualan. Data finansial harus tetap ada untuk laporan. Tenantnya masih
ada — yang perlu dicatat adalah: laporan menunjukkan `router_id = NULL` berarti
router sudah tidak ada, tapi transaksinya nyata.

---

### Model: `PrintTemplate`

Hapus `RouterID`. Tambah `TenantID` nullable.
`tenant_id = NULL` = template default global, hanya bisa dikelola superadmin.

```go
// internal/models/print_template.go
type PrintTemplate struct {
    ID        uint   `gorm:"primaryKey" json:"id"`
    // TenantID nullable: NULL = global default (superadmin only)
    // Saat tenant dibuat, global defaults di-copy ke tenant (bukan inheritance runtime)
    TenantID  *uint  `gorm:"index;constraint:OnDelete:CASCADE" json:"tenant_id"`
    Name      string `gorm:"size:100;not null" json:"name"`
    Type      string `gorm:"size:20;not null"  json:"type"`
    Part      string `gorm:"size:10;check:chk_part,part IN ('header','row','footer')" json:"part"`
    Content   string `gorm:"type:text;not null" json:"content"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

**Perubahan dari `router_id = 0` hack:**
Code lama menggunakan `WHERE router_id = ? OR router_id = 0` sebagai cara membedakan
global vs per-router. Ini diganti dengan `tenant_id IS NULL` yang eksplisit dan aman.

---

### Model: `AuditLog`

Tambah `TenantID`. `UserID` dan `TenantID` sama-sama nullable karena superadmin
bisa melakukan aksi lintas tenant, dan beberapa aksi sistem tidak punya user.

```go
// internal/models/audit_log.go
type AuditLog struct {
    ID        uint      `gorm:"primaryKey"                                      json:"id"`
    TenantID  *uint     `gorm:"index;constraint:OnDelete:CASCADE"               json:"tenant_id"`
    UserID    *uint     `gorm:"index"                                           json:"user_id"`
    // UserID tidak CASCADE — log tetap ada meskipun user dihapus
    Action    string    `gorm:"size:50;not null;index"                          json:"action"`
    Entity    string    `gorm:"size:50;not null"                                json:"entity"`
    EntityID  string    `gorm:"size:50"                                         json:"entity_id"`
    Details   string    `gorm:"type:text"                                       json:"details"`
    IPAddress string    `gorm:"size:45"                                         json:"ip_address"`
    CreatedAt time.Time `gorm:"index"                                           json:"created_at"`
}
```

---

### Model Dihapus: `HotspotConfig`

Hapus `internal/models/hotspot_config.go` sepenuhnya.
Tidak ada relasi ke `Router` lagi. Fungsionalitasnya terbagi ke:
- Data koneksi → tetap di `Router`
- Konfigurasi bisnis → pindah ke `TenantSettings`

---

### Update `AutoMigrate`

Urutan penting — parent harus di-migrate sebelum child yang punya FK.

```go
func AutoMigrate(db *gorm.DB) error {
    return db.AutoMigrate(
        &models.Tenant{},              // root
        &models.TenantSettings{},      // FK → tenants
        &models.User{},                // FK → tenants (nullable)
        &models.Router{},              // FK → tenants
        &models.ProfilePriceMapping{}, // FK → routers
        &models.VoucherSale{},         // FK → tenants + routers (nullable)
        &models.PrintTemplate{},       // FK → tenants (nullable)
        &models.AuditLog{},            // FK → tenants (nullable) + users (nullable)
    )
}
```

---

## Phase 2 — Casbin RBAC

### Model File: `internal/casbin/model.conf`

```ini
[request_definition]
r = sub, dom, obj, act

[policy_definition]
p = sub, dom, obj, act

[role_definition]
g = _, _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub, r.dom) && (p.dom == "*" || r.dom == p.dom) && keyMatch2(r.obj, p.obj) && (r.act == p.act || p.act == "*")
```

**Penjelasan matcher:**
- `g(r.sub, p.sub, r.dom)` — user punya role tersebut dalam domain (tenant) tersebut
- `p.dom == "*"` — policy berlaku untuk semua tenant (dipakai oleh semua role definitions)
- `keyMatch2` — support `:param` wildcard Gin, misal `/api/v1/routers/:routerId`
- `p.act == "*"` — policy berlaku untuk semua HTTP method

---

### Policy Definitions

```
# ════════════════════════════════════════
# SUPERADMIN — akses penuh semua tenant
# ════════════════════════════════════════
p, superadmin, *, /api/v1/*, *

# ════════════════════════════════════════
# OWNER — full access dalam tenant sendiri
# ════════════════════════════════════════
p, owner, *, /api/v1/tenant,          *
p, owner, *, /api/v1/tenant/*,        *
p, owner, *, /api/v1/users,           *
p, owner, *, /api/v1/users/:id,       *
p, owner, *, /api/v1/routers,         *
p, owner, *, /api/v1/routers/:id,     *
p, owner, *, /api/v1/routers/:id/*,   *
p, owner, *, /api/v1/templates,       *
p, owner, *, /api/v1/templates/*,     *

# ════════════════════════════════════════
# ADMIN — semua kecuali tenant settings dan billing
# ════════════════════════════════════════
p, admin, *, /api/v1/users,           GET
p, admin, *, /api/v1/users,           POST
p, admin, *, /api/v1/users/:id,       GET
p, admin, *, /api/v1/users/:id,       PUT
p, admin, *, /api/v1/users/:id,       DELETE
p, admin, *, /api/v1/routers,         *
p, admin, *, /api/v1/routers/:id,     *
p, admin, *, /api/v1/routers/:id/*,   *
p, admin, *, /api/v1/templates,       *
p, admin, *, /api/v1/templates/*,     *

# ════════════════════════════════════════
# STAFF — operasional harian saja
# ════════════════════════════════════════
p, staff, *, /api/v1/routers,                          GET
p, staff, *, /api/v1/routers/:id,                      GET
p, staff, *, /api/v1/routers/:id/hotspot/users,        *
p, staff, *, /api/v1/routers/:id/hotspot/users/:uid,   *
p, staff, *, /api/v1/routers/:id/hotspot/active,       *
p, staff, *, /api/v1/routers/:id/hotspot/inactive,     GET
p, staff, *, /api/v1/routers/:id/hotspot/profiles,     GET
p, staff, *, /api/v1/routers/:id/hotspot/servers,      GET
p, staff, *, /api/v1/routers/:id/vouchers/*,           *
p, staff, *, /api/v1/routers/:id/reports/*,            GET
p, staff, *, /api/v1/templates,                        GET
p, staff, *, /api/v1/templates/:id,                    GET
```

---

### Enforcer Setup

```go
// internal/casbin/enforcer.go
package casbinx

import (
    "embed"
    "fmt"

    "github.com/casbin/casbin/v2"
    "github.com/casbin/casbin/v2/model"
    gormadapter "github.com/casbin/gorm-adapter/v3"
    "gorm.io/gorm"
)

//go:embed model.conf
var modelFS embed.FS

func NewEnforcer(db *gorm.DB) (*casbin.Enforcer, error) {
    adapter, err := gormadapter.NewAdapterByDB(db)
    if err != nil {
        return nil, fmt.Errorf("casbin adapter: %w", err)
    }

    src, _ := modelFS.ReadFile("model.conf")
    m, err := model.NewModelFromString(string(src))
    if err != nil {
        return nil, fmt.Errorf("casbin model: %w", err)
    }

    e, err := casbin.NewEnforcer(m, adapter)
    if err != nil {
        return nil, fmt.Errorf("casbin enforcer: %w", err)
    }

    return e, e.LoadPolicy()
}

// SeedPolicies menyimpan role policies ke DB. Idempotent — aman dipanggil ulang.
func SeedPolicies(e *casbin.Enforcer) error {
    policies := getRolePolicies()
    for _, p := range policies {
        if has, _ := e.HasPolicy(p[0], p[1], p[2], p[3]); !has {
            if _, err := e.AddPolicy(p[0], p[1], p[2], p[3]); err != nil {
                return err
            }
        }
    }
    return nil
}

// AssignRole memberikan role ke user dalam satu tenant.
// Untuk superadmin, tenantSlug = "__platform__"
func AssignRole(e *casbin.Enforcer, userID uint, tenantSlug string, role models.UserRole) error {
    sub := fmt.Sprintf("%d", userID)
    if has, _ := e.HasRoleForUserInDomain(sub, string(role), tenantSlug); has {
        return nil
    }
    _, err := e.AddRoleForUserInDomain(sub, string(role), tenantSlug)
    return err
}

// RevokeRoles mencabut semua role user dalam satu tenant.
func RevokeRoles(e *casbin.Enforcer, userID uint, tenantSlug string) error {
    sub := fmt.Sprintf("%d", userID)
    _, err := e.DeleteRolesForUserInDomain(sub, tenantSlug)
    return err
}
```

---

### Casbin Middleware

```go
// internal/api/middleware/casbin.go
func CasbinMiddleware(e *casbin.Enforcer) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID   := c.MustGet("userID").(uint)
        tenantSlug := c.MustGet("tenantSlug").(string)

        sub := fmt.Sprintf("%d", userID)
        obj := c.FullPath()        // template path: "/api/v1/routers/:routerId"
        act := c.Request.Method    // GET, POST, PUT, DELETE

        ok, err := e.Enforce(sub, tenantSlug, obj, act)
        if err != nil || !ok {
            c.AbortWithStatusJSON(403, gin.H{"data": nil, "error": "forbidden"})
            return
        }
        c.Next()
    }
}
```

---

## Phase 3 — Auth Refactor

### JWT Claims Baru

```go
type Claims struct {
    UserID     uint     `json:"uid"`
    Username   string   `json:"sub"`
    Role       UserRole `json:"role"`
    TenantID   *uint    `json:"tid"`   // NULL untuk superadmin
    TenantSlug string   `json:"tslug"` // "__platform__" untuk superadmin
    TokenID    string   `json:"jti"`
    jwt.RegisteredClaims
}
```

---

### Login Flow

```
POST /api/v1/auth/login
{
  "tenant": "bintang-net",   // slug tenant, wajib untuk user biasa
  "username": "admin",
  "password": "..."
}
```

Superadmin login tanpa field `tenant` (atau `tenant: ""`) — sistem detect dari role.

Flow di `AuthService.Login`:
1. Kalau `tenant` kosong → cari user di mana `tenant_id IS NULL`
2. Kalau `tenant` ada → cari Tenant by slug, validasi `status != suspended`, cari User by `(tenant_id, username)`
3. Validasi password + active
4. Generate JWT dengan claims yang sesuai
5. Untuk superadmin: `TenantID = nil`, `TenantSlug = "__platform__"`

---

### Superadmin Akses Tenant Spesifik

Superadmin melewatkan header `X-Tenant-Slug` di setiap request yang butuh tenant context:

```
GET /api/v1/routers
Authorization: Bearer {superadmin-jwt}
X-Tenant-Slug: bintang-net
```

`TenantMiddleware` logic:
```go
func TenantMiddleware(tenantRepo TenantRepository) gin.HandlerFunc {
    return func(c *gin.Context) {
        role := c.MustGet("role").(UserRole)

        var tenantSlug string

        if role == UserRoleSuperAdmin {
            // Superadmin: ambil tenant dari header, boleh kosong untuk platform-level ops
            tenantSlug = c.GetHeader("X-Tenant-Slug")
            if tenantSlug == "" {
                tenantSlug = "__platform__"
            }
        } else {
            // User biasa: tenant dari JWT claim, tidak bisa di-override
            tenantSlug = c.MustGet("tenantSlug").(string)
        }

        if tenantSlug != "__platform__" {
            // Validasi tenant masih aktif — cache di Redis, TTL 5 menit
            tenant, err := tenantRepo.GetBySlug(c.Request.Context(), tenantSlug)
            if err != nil || tenant.Status == TenantStatusSuspended {
                c.AbortWithStatusJSON(403, gin.H{"error": "tenant not accessible"})
                return
            }
            c.Set("tenant", tenant)
            c.Set("tenantID", tenant.ID)
        }

        c.Set("tenantSlug", tenantSlug)
        c.Next()
    }
}
```

---

### Setup Endpoint (Bootstrap)

```
POST /api/v1/auth/setup
{
  "tenant_name": "Bintang Net",
  "tenant_slug": "bintang-net",
  "username":    "owner",
  "password":    "..."
}
```

Hanya bisa dipanggil jika belum ada tenant satupun di database.
Flow:
1. Cek `COUNT(tenants) == 0`, kalau tidak → 409
2. Create `Tenant`
3. Create `TenantSettings` kosong (bisa diisi nanti)
4. Create `User` dengan `role = owner`
5. `casbinx.AssignRole(enforcer, user.ID, tenant.Slug, UserRoleOwner)`
6. Copy global default templates ke tenant baru
7. Return JWT

---

## Phase 4 — Template: Global Default + Copy on Tenant Create

### Alur Copy Template

Saat tenant baru dibuat (via setup atau API), jalankan:

```go
// internal/services/tenant_service.go
func (s *TenantService) copyDefaultTemplates(ctx context.Context, tx *gorm.DB, tenantID uint) error {
    var globals []models.PrintTemplate
    // Global templates: tenant_id IS NULL
    if err := tx.Where("tenant_id IS NULL").Find(&globals).Error; err != nil {
        return err
    }
    for _, g := range globals {
        copy := models.PrintTemplate{
            TenantID: &tenantID, // sekarang milik tenant ini
            Name:     g.Name,
            Type:     g.Type,
            Part:     g.Part,
            Content:  g.Content,
        }
        if err := tx.Create(&copy).Error; err != nil {
            return err
        }
    }
    return nil
}
```

Setelah di-copy, template tenant berdiri sendiri — tidak terhubung ke global.
Superadmin bisa update global templates, tapi itu tidak mempengaruhi tenant yang sudah ada.

### Endpoint Template Global (Superadmin Only)

```
GET    /api/v1/admin/templates          — list global default templates
POST   /api/v1/admin/templates          — buat global default baru
PUT    /api/v1/admin/templates/:id      — edit global default
DELETE /api/v1/admin/templates/:id      — hapus global default
```

Casbin policy untuk ini sudah ter-cover oleh `p, superadmin, *, /api/v1/*, *`.

---

## Phase 5 — Repository Layer

### Pola Wajib: Semua Query Harus Filter `tenant_id`

```go
// Contoh RouterRepo — pola ini berlaku untuk semua repo
func (r *RouterRepo) List(ctx context.Context, tenantID uint) ([]models.Router, error) {
    var routers []models.Router
    return routers, r.db.WithContext(ctx).
        Where("tenant_id = ?", tenantID).
        Find(&routers).Error
}

func (r *RouterRepo) GetByID(ctx context.Context, tenantID, id uint) (*models.Router, error) {
    var router models.Router
    // Selalu sertakan tenant_id — tidak boleh akses router tenant lain
    // meski router ID valid
    return &router, r.db.WithContext(ctx).
        Where("tenant_id = ? AND id = ?", tenantID, id).
        First(&router).Error
}
```

### Repo Baru yang Dibutuhkan

| Repo | Interface Utama |
|---|---|
| `TenantRepo` | `Create`, `GetByID`, `GetBySlug`, `List`, `Update`, `Delete` (hard) |
| `TenantSettingsRepo` | `GetByTenantID`, `Upsert` |
| `UserRepo` (rename) | `Create`, `GetByID`, `GetByUsername(tenantID, username)`, `List(tenantID)`, `Update`, `Delete` |

### Repo yang Diupdate

| Repo | Perubahan |
|---|---|
| `RouterRepo` | Semua method tambah `tenantID` param |
| `SaleRepo` | Semua method tambah `tenantID`, update `List` untuk cross-router query |
| `TemplateRepo` | Ganti `routerID` → `tenantID`, tambah `ListGlobal()` untuk superadmin |
| `AuditRepo` | Tambah `tenantID` param, `UserID` jadi `*uint` |
| `ProfilePriceMappingRepo` | Tidak berubah banyak — FK via routerID sudah cukup |

---

## Phase 6 — API Layer

### Middleware Chain

```go
// Endpoint tenant-scoped (semua kecuali /auth dan /admin)
protected.Use(
    middleware.AuthMiddleware(authSvc),      // 1. validasi JWT
    middleware.TenantMiddleware(tenantRepo), // 2. resolve + validasi tenant
    middleware.CasbinMiddleware(enforcer),   // 3. enforce policy
)
```

### Endpoint Baru

```
# Auth
POST /api/v1/auth/setup          — bootstrap (hanya sekali)
POST /api/v1/auth/login          — { tenant, username, password }
POST /api/v1/auth/refresh
POST /api/v1/auth/logout

# Tenant (owner only via Casbin)
GET  /api/v1/tenant              — info tenant + settings
PUT  /api/v1/tenant              — update nama tenant
PUT  /api/v1/tenant/settings     — update hotspot settings
POST /api/v1/tenant/logo         — upload logo

# User management (owner + admin via Casbin)
GET    /api/v1/users             — list users dalam tenant
POST   /api/v1/users             — buat user baru + assign Casbin role
GET    /api/v1/users/:id
PUT    /api/v1/users/:id         — update + re-assign Casbin role jika berubah
DELETE /api/v1/users/:id         — soft delete + revoke Casbin role

# Templates (pindah dari per-router ke per-tenant)
GET    /api/v1/templates
POST   /api/v1/templates
GET    /api/v1/templates/:id
PUT    /api/v1/templates/:id
DELETE /api/v1/templates/:id
POST   /api/v1/templates/render
POST   /api/v1/templates/seed-defaults  — reset ke copy global defaults

# Platform admin (superadmin only)
GET    /api/v1/admin/tenants
POST   /api/v1/admin/tenants
GET    /api/v1/admin/tenants/:id
PUT    /api/v1/admin/tenants/:id
DELETE /api/v1/admin/tenants/:id  — hard delete + cascade
POST   /api/v1/admin/tenants/:id/suspend
POST   /api/v1/admin/tenants/:id/activate
GET    /api/v1/admin/templates    — kelola global default templates
POST   /api/v1/admin/templates
PUT    /api/v1/admin/templates/:id
DELETE /api/v1/admin/templates/:id
```

---

## Urutan Implementasi

```
Phase 1 — Database (lakukan dulu, schema harus stabil sebelum lanjut)
  1a. Buat: Tenant, TenantSettings, UserRole type
  1b. Ubah: SystemUser → User (tenant_id nullable, composite unique index)
  1c. Ubah: Router (+ tenant_id, - HotspotConfig relation)
  1d. Ubah: VoucherSale (+ tenant_id, router_id nullable + SET NULL)
  1e. Ubah: PrintTemplate (- router_id, + tenant_id nullable)
  1f. Ubah: AuditLog (+ tenant_id nullable, user_id nullable)
  1g. Hapus: HotspotConfig model + file
  1h. Update AutoMigrate

Phase 2 — Casbin
  2a. go get github.com/casbin/casbin/v2 github.com/casbin/gorm-adapter/v3
  2b. Buat internal/casbin/model.conf
  2c. Buat internal/casbin/enforcer.go (NewEnforcer, SeedPolicies, AssignRole, RevokeRoles)
  2d. Buat internal/api/middleware/casbin.go

Phase 3 — Auth
  3a. Update Claims struct (+ TenantID *uint, TenantSlug)
  3b. Update AuthService.Login (+ tenant lookup, superadmin flow)
  3c. Update AuthService.Setup (+ create tenant, copy templates, assign Casbin role)
  3d. Buat TenantMiddleware
  3e. Hapus RequireRole middleware (digantikan Casbin)

Phase 4 — Repository
  4a. Buat TenantRepo + TenantSettingsRepo
  4b. Update UserRepo (rename, semua method + tenantID)
  4c. Update RouterRepo (semua method + tenantID)
  4d. Update SaleRepo (+ tenantID, update cross-router query)
  4e. Update TemplateRepo (router_id → tenant_id, tambah ListGlobal)
  4f. Update AuditRepo (+ tenantID, UserID → *uint)

Phase 5 — Service Layer
  5a. Buat TenantService (CRUD tenant, copy templates, suspend/activate)
  5b. Update AuthService (terhubung ke TenantService)
  5c. Update RouterService (+ tenantID di semua method)
  5d. Update VoucherService (+ tenantID)
  5e. Update TemplateService (router_id → tenant_id, seed defaults dari global)
  5f. Update AuditLogger (+ tenantID)

Phase 6 — API
  6a. Update router.go: middleware chain baru
  6b. Buat TenantHandler
  6c. Buat UserHandler
  6d. Pindahkan TemplateHandler dari /routers/:id/templates → /templates
  6e. Update semua handler: ambil tenantID dari context
  6f. Update cmd/seed: buat tenant + user via service layer (bukan langsung ke DB)

Phase 7 — Validasi
  [ ] go build ./... — tidak ada error kompilasi
  [ ] go test ./...
  [ ] Test isolasi: user tenant A tidak bisa akses data tenant B (harus 403)
  [ ] Test Casbin: staff tidak bisa DELETE /routers/:id (harus 403)
  [ ] Test Casbin: admin tidak bisa PUT /tenant/settings (harus 403)
  [ ] Test superadmin: bisa akses tenant A dan B dengan X-Tenant-Slug header
  [ ] Test superadmin: tanpa X-Tenant-Slug, tenantSlug = "__platform__"
  [ ] Test template copy: tenant baru dapat copy dari global defaults
  [ ] Test voucher_sale: router di-soft-delete, sales masih ada dengan router_id valid
  [ ] Test cascade: tenant di-hard-delete, semua data terhapus
  [ ] Test user soft-delete: bisa buat ulang dengan username yang sama dalam tenant yang sama
```

---

## Catatan Breaking Change

- **JWT lama invalid** setelah Phase 3 deploy — semua user harus login ulang
- **Endpoint templates** pindah dari `/routers/:id/templates` ke `/templates`
- **Login request** sekarang butuh field `tenant`
- Token lama dari sebelum refactor tidak punya `TenantID` di claims — akan ditolak oleh `TenantMiddleware`