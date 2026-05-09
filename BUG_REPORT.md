# Bug Report — Roskit API

**Tanggal Testing:** 2026-05-09  
**Script Testing:** `scratch_test.ps1`  
**Target:** `http://localhost:8080/api/v1`  
**Metode:** Black-box HTTP testing + Static code review  
**Build Status:** ✅ `go build ./...` — LULUS (exit code 0)

---

## Ringkasan Eksekusi Test (`scratch_test.ps1`)

| Test Case | Hasil | Status |
|---|---|---|
| Superadmin login | ✅ Token diterima | LULUS |
| Superadmin GET /users (X-Tenant-Slug: alpha) | ❌ `forbidden` | **GAGAL** |
| Superadmin GET /users (X-Tenant-Slug: beta) | ❌ `forbidden` | **GAGAL** |
| Admin Alpha login | ✅ Token diterima | LULUS |
| Admin Alpha GET /users | ✅ 3 users ditemukan | LULUS |
| Admin Alpha GET /users/:id | ✅ User ditemukan | LULUS |
| Staff Alpha GET /users | ❌ `forbidden` | LULUS (expected) |
| Admin Beta GET /users | ✅ 3 users ditemukan | LULUS |
| Admin Beta GET /users/:id | ✅ User ditemukan | LULUS |
| Staff Beta GET /users | ❌ `forbidden` | LULUS (expected) |
| Cross-tenant: Admin Alpha → Beta | ❌ **Diizinkan!** | **GAGAL (KRITIS)** |

---

## Daftar Bug

| ID | Severity | Kategori | File | Status |
|---|---|---|---|---|
| BUG-001 | 🔴 Critical | Security / AuthZ | `middleware/tenant.go` | Terbukti dari test |
| BUG-002 | 🔴 High | Authorization / Casbin | `casbin/policies.go` + `model.conf` | Terbukti dari test |
| BUG-003 | 🟡 Medium | Logic / Silent Failure | `handlers/user_handler.go` | Code review |
| BUG-004 | 🟡 Medium | Logic / Auth | `handlers/auth_handler.go` | Code review |
| BUG-005 | 🟡 Medium | Logic / Race | `services/router_service.go` | Code review |
| BUG-006 | 🟡 Medium | Logic / Security | `services/tenant_service.go` | Code review |
| BUG-007 | 🟢 Low | Logic / Data | `services/report_service.go` | Code review |
| BUG-008 | 🟢 Low | Logic / Export | `services/hotspot_service.go` | Code review |
| BUG-009 | 🟢 Low | Logic / Cache | `services/voucher_service.go` | Code review |
| BUG-010 | 🟢 Low | Consistency | `casbin/policies.go` | Code review |

---

## Detail Bug

---

### 🔴 BUG-001 — Cross-Tenant Access Tidak Diblokir (KRITIS)

**Severity:** Critical | **File:** `internal/api/middleware/tenant.go:64–70`

**Output test:**
```
=== Testing Cross-Tenant: Admin Alpha reading Beta ===
FAIL: Allowed access! Found 3 users.
```

**Deskripsi:**  
`admin.alpha` berhasil membaca daftar user tenant `beta` hanya dengan mengganti header `X-Tenant-Slug: beta`. Isolasi data antar-tenant tidak berjalan untuk role non-superadmin.

**Root Cause:**
```go
if role != models.UserRoleSuperAdmin {
    jwtTID, _ := c.Get("jwtTenantID")         // _ membuang ok
    if tid, ok := jwtTID.(uint); ok && tid != tenant.ID {  // jika ok=false → TIDAK DIBLOKIR
        c.AbortWithStatusJSON(403, ...)
        return
    }
}
```

Kondisi `if tid, ok := jwtTID.(uint); ok && tid != tenant.ID` hanya memblokir jika **keduanya benar**: type assertion sukses DAN ID berbeda. Jika type assertion gagal (`ok=false`), blok abort terlewati dan request diteruskan tanpa pengecekan apapun.

**Perbaikan:**
```go
if role != models.UserRoleSuperAdmin {
    jwtTID, ok := c.Get("jwtTenantID")
    if !ok {
        c.AbortWithStatusJSON(403, gin.H{"data": nil, "error": "tenant context missing"})
        return
    }
    tid, ok := jwtTID.(uint)
    if !ok || tid != tenant.ID {
        c.AbortWithStatusJSON(403, gin.H{"data": nil, "error": "tenant mismatch"})
        return
    }
}
```

---

### 🔴 BUG-002 — Superadmin `forbidden` di Semua Endpoint (TINGGI)

**Severity:** High | **File:** `internal/casbin/policies.go:10` + `internal/casbin/model.conf:14`

**Output test:**
```
=== Testing Superadmin (to Alpha) (superadmin) ===
Login SUCCESS. Token: eyJhbGciOiJIUzI1NiIs...
ERROR: {"data":null,"error":"forbidden"}
```

**Deskripsi:**  
Superadmin login sukses tetapi selalu mendapat `403 forbidden` di semua endpoint, termasuk `GET /users` yang seharusnya diizinkan penuh.

**Root Cause — ketidakcocokan `keyMatch2` vs wildcard `*`:**

Model Casbin (`model.conf`) menggunakan `keyMatch2`:
```ini
m = g(r.sub, p.sub, r.dom) && ... && keyMatch2(r.obj, p.obj) && ...
```

Policy superadmin:
```go
{"superadmin", "*", "/api/v1/*", "*"},
```

`keyMatch2` dari Casbin mencocokkan pola `:param` gaya Gin, **bukan** glob `*`. Path `/api/v1/*` dengan tanda bintang literal **tidak** mencocokkan `/api/v1/users` — `keyMatch2` hanya mencocokkan satu level path parameter bernama (`:sesuatu`). Akibatnya tidak ada policy superadmin yang pernah match.

**Perbaikan — Opsi 1 (rekomendasi):** Ganti matcher agar mendukung glob menggunakan `keyMatch`:
```ini
[matchers]
m = g(r.sub, p.sub, r.dom) && (p.dom == "*" || r.dom == p.dom) && (keyMatch2(r.obj, p.obj) || keyMatch(r.obj, p.obj)) && (r.act == p.act || p.act == "*")
```

**Perbaikan — Opsi 2:** Daftarkan policy superadmin secara eksplisit per endpoint (verbose tetapi aman):
```go
{"superadmin", "*", "/api/v1/users", "*"},
{"superadmin", "*", "/api/v1/users/:id", "*"},
// ... semua endpoint lain
```

---

### 🟡 BUG-003 — Password Reset Admin Gagal Diam-Diam (SEDANG)

**Severity:** Medium | **File:** `internal/api/handlers/user_handler.go:143–149`

**Deskripsi:**  
`PUT /users/:id` dengan field `password` selalu mengembalikan `200 OK`, tetapi password user **tidak pernah berubah**. Handler memanggil `ChangePassword` dengan `oldPass=""` yang selalu gagal karena validasi bcrypt, kemudian error-nya diabaikan dengan `_ = err`.

**Kode bermasalah:**
```go
if req.Password != nil && *req.Password != "" {
    if err := h.authSvc.ChangePassword(c.Request.Context(), user.ID, "", *req.Password); err != nil {
        // ChangePassword requires old password — skip for admin path; do raw update via repo
        _ = err   // ← ERROR SELALU TERJADI, SELALU DIABAIKAN
    }
}
// Lalu userRepo.Update() dipanggil — tapi PasswordHash tidak berubah karena ChangePassword gagal
```

`ChangePassword` di `auth_service.go:360` **selalu** membandingkan password lama via bcrypt — memanggil dengan `""` akan selalu gagal.

**Perbaikan:** Tambahkan metode `AdminResetPassword` di `AuthService` yang melewati validasi password lama:
```go
// auth_service.go
func (s *AuthService) AdminResetPassword(ctx context.Context, userID uint, newPass string) error {
    user, err := s.userRepo.GetByID(ctx, userID)
    if err != nil {
        return fmt.Errorf("user not found: %w", err)
    }
    hash, err := bcrypt.GenerateFromPassword([]byte(newPass), 12)
    if err != nil {
        return fmt.Errorf("hash password: %w", err)
    }
    user.PasswordHash = string(hash)
    return s.userRepo.Update(ctx, user)
}

// user_handler.go
if req.Password != nil && *req.Password != "" {
    if err := h.authSvc.AdminResetPassword(c.Request.Context(), user.ID, *req.Password); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to update password"})
        return
    }
}
```

---

### 🟡 BUG-004 — Audit Log `auth.logout` Dicatat Setelah Response Dikirim (SEDANG)

**Severity:** Medium | **File:** `internal/api/handlers/auth_handler.go:116–117`

**Deskripsi:**  
Di handler `Logout`, `LogAuth` dipanggil **setelah** `c.JSON()` mengirim response:

```go
c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "logged out"}, "error": nil})
h.audit.LogAuth(c, "auth.logout", "")   // ← dipanggil setelah response
```

Secara teknis, karena `AuditLogger.Log()` bersifat **non-blocking** (channel), ini tidak menyebabkan crash. Namun, jika `AuditLogger` sedang shutdown, `audit.done` sudah tertutup, dan panggilan `c.Get()` di dalam `Log()` akan mengakses `*gin.Context` yang mungkin sudah di-reset oleh Gin untuk request selanjutnya.

Lebih konseptual: pola yang benar adalah memanggil audit **sebelum** response, atau gunakan `defer`. Bandingkan dengan `auth.setup` di baris 261 yang juga punya masalah yang sama.

**Perbaikan:**
```go
h.audit.LogAuth(c, "auth.logout", "")   // pindahkan ke atas
c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "logged out"}, "error": nil})
```

---

### 🟡 BUG-005 — Race Condition saat Update Router: Engine Re-register Menggunakan Password Terenkripsi (SEDANG)

**Severity:** Medium | **File:** `internal/services/router_service.go:349–365`

**Deskripsi:**  
Di `UpdateRouter`, ketika kredensial router berubah, password yang sudah **terenkripsi** bisa masuk ke engine jika dekripsi gagal diam-diam:

```go
if credsChanged {
    routerID := fmt.Sprintf("%d", id)
    plainPass := passwordPlain
    if plainPass == "" {
        dec, err := encrypt.Decrypt(router.APIPasswordEncrypted, s.aesKey)
        if err == nil {  // ← jika err != nil, plainPass tetap ""
            plainPass = dec
        }
    }
    s.engine.RemoveRouter(routerID)
    _ = s.engine.AddRouter(ctx, execution.ConnConfig{
        ...
        Password: plainPass,   // ← bisa "" jika dekripsi gagal
    })
}
```

Jika `encrypt.Decrypt` gagal (AES key berubah, data corrupt), `plainPass` tetap `""`, router di-remove dari engine, lalu di-add kembali dengan password kosong — koneksi akan selalu gagal tanpa ada error yang dikembalikan ke caller.

**Dampak tambahan:** `s.engine.RemoveRouter()` dan `s.engine.AddRouter()` dipanggil terpisah tanpa sinkronisasi — window waktu singkat dimana router tidak terdaftar di engine bisa menyebabkan request concurrent ke router tersebut gagal.

**Perbaikan:**
```go
if credsChanged {
    dec, err := encrypt.Decrypt(router.APIPasswordEncrypted, s.aesKey)
    if err != nil {
        return nil, fmt.Errorf("decrypt updated password: %w", err)  // gagal secara eksplisit
    }
    plainPass := dec
    if passwordPlain != "" {
        plainPass = passwordPlain
    }
    routerID := fmt.Sprintf("%d", id)
    s.engine.RemoveRouter(routerID)
    _ = s.engine.AddRouter(ctx, execution.ConnConfig{...Password: plainPass})
}
```

---

### 🟡 BUG-006 — Tenant Logo Disimpan dengan Ekstensi `.png` Terlepas dari Tipe File Asli (SEDANG)

**Severity:** Medium | **File:** `internal/services/tenant_service.go:334`

**Deskripsi:**  
`UploadLogo` selalu menyimpan file dengan nama `tenant-{id}.png`, meskipun validasi MIME di handler memperbolehkan JPEG dan WebP:

```go
// tenant_service.go
path := fmt.Sprintf("%s/tenant-%d.png", dir, tenantID)   // selalu .png
```

```go
// tenant_handler.go
if !isAllowedImageMIME(data) {
    c.JSON(http.StatusBadRequest, ...)
}
// isAllowedImageMIME menerima PNG, JPEG, dan WEBP
```

**Dampak:**
1. File JPEG/WebP disimpan dengan ekstensi `.png`, menyebabkan browser tidak bisa mengidentifikasi format yang benar berdasarkan ekstensi.
2. Jika logo lama adalah JPEG dan logo baru adalah PNG (atau sebaliknya), file-nya akan ditimpa dengan nama yang sama — itu justru benar — tapi Content-Type yang dikembalikan saat `GET /tenant/logo` bergantung pada `c.File(path)` dari Gin yang mendeteksi MIME berdasarkan ekstensi, bukan konten.

**Perbaikan:**
```go
func (s *TenantService) UploadLogo(ctx context.Context, tenantID uint, fileData []byte, filename string) (string, error) {
    ext := filepath.Ext(filename)
    if ext == "" {
        ext = ".png"
    }
    path := fmt.Sprintf("%s/tenant-%d%s", dir, tenantID, ext)
    ...
}
```

---

### 🟢 BUG-007 — Export CSV/Excel Menggunakan Cache TTL Past Meskipun Rentang Waktu Mencakup Hari Ini (RENDAH)

**Severity:** Low | **File:** `internal/services/report_service.go:227–230, 263–266`

**Deskripsi:**  
`ExportCSV` dan `ExportExcel` selalu menggunakan `appcache.TTLSalesPast` (kemungkinan besar TTL panjang seperti 24 jam atau lebih) sebagai TTL cache, bahkan ketika rentang tanggal `from..to` mencakup hari ini:

```go
func (s *ReportService) ExportCSV(...) ([]byte, error) {
    cacheKey := ...
    ttl := appcache.TTLSalesPast   // ← selalu past, tidak peduli rentang tanggal
    return appcache.GetOrSetJSON(s.cache, ctx, cacheKey, ttl, func() ...)
}
```

Bandingkan dengan `GetDailyReport` yang sudah benar memeriksa apakah tanggal sudah lewat:
```go
isPast := date.Before(time.Now().Truncate(24 * time.Hour))
ttl := appcache.TTLSalesDay
if isPast {
    ttl = appcache.TTLSalesPast
}
```

**Dampak:** Export yang mencakup hari ini akan di-cache terlalu lama, sehingga transaksi baru yang terjadi di hari yang sama tidak akan terlihat dalam export sampai TTL habis.

**Perbaikan:**
```go
ttl := appcache.TTLSalesDay
if to.Before(time.Now().Truncate(24 * time.Hour)) {
    ttl = appcache.TTLSalesPast
}
```

---

### 🟢 BUG-008 — Export Script Hotspot User Tidak Meng-quote Nilai dengan Spasi (RENDAH)

**Severity:** Low | **File:** `internal/services/hotspot_service.go:593–606`

**Deskripsi:**  
Fungsi `ExportUsers` dengan format `script` menghasilkan script RouterOS menggunakan `fmt.Sprintf` tanpa quoting nilai yang mengandung spasi:

```go
sb.WriteString(fmt.Sprintf("/ip hotspot user add name=%s password=%s profile=%s",
    u["name"], u["password"], u["profile"]))
// ...
if comment, ok := u["comment"]; ok && comment != "" {
    sb.WriteString(fmt.Sprintf(" comment=\"%s\"", comment))  // comment dikutip
}
```

`name`, `password`, dan `profile` tidak dikutip. Jika nama atau password voucher mengandung spasi, script RouterOS yang dihasilkan akan **invalid** dan gagal saat diimport. `comment` sudah benar dikutip, tapi tiga field lainnya tidak.

**Perbaikan:**
```go
sb.WriteString(fmt.Sprintf("/ip hotspot user add name=\"%s\" password=\"%s\" profile=\"%s\"",
    u["name"], u["password"], u["profile"]))
```

---

### 🟢 BUG-009 — `GetDashboardSummary` Menggunakan Cache Key Berbeda untuk Scope yang Sama (RENDAH)

**Severity:** Low | **File:** `internal/services/report_service.go:199–203`

**Deskripsi:**  
Fungsi `GetDashboardSummary` memiliki dua path untuk membuat cache key:

```go
cacheKey := appcache.DashboardKey(scopeCacheRouterID(routerID))   // untuk per-router
if routerID == nil {
    cacheKey = fmt.Sprintf("mikhmon:dashboard:tenant:%d", tenantID)  // untuk tenant-wide
}
```

Sementara itu, invalidasi cache di `EventHandler.OnLoginEvent` menggunakan:
```go
_ = h.cache.Invalidate(c.Request.Context(),
    appcache.SalesKey(router.ID, "today"),
    appcache.SalesKey(router.ID, "month"),
    appcache.DashboardKey(router.ID),   // ← hanya menginvalidasi per-router
)
```

Ketika event on-login terjadi, **cache dashboard tenant-wide** (`mikhmon:dashboard:tenant:{id}`) **tidak diinvalidasi**, sehingga `GetDashboardSummary` tanpa routerID bisa mengembalikan data stale setelah transaksi baru.

**Perbaikan:** Tambahkan invalidasi tenant-wide di `EventHandler.OnLoginEvent`:
```go
_ = h.cache.Invalidate(c.Request.Context(),
    appcache.SalesKey(router.ID, "today"),
    appcache.SalesKey(router.ID, "month"),
    appcache.DashboardKey(router.ID),
    fmt.Sprintf("mikhmon:dashboard:tenant:%d", tenantID),  // tambahkan ini
)
```

---

### 🟢 BUG-010 — Inkonsistensi Nama Parameter di Casbin Policy vs Router Gin (RENDAH)

**Severity:** Low | **File:** `internal/casbin/policies.go:44–53`

**Deskripsi:**  
Policy Casbin untuk `staff` menggunakan nama placeholder `:id` untuk router:
```go
{"staff", "*", "/api/v1/routers/:id/hotspot/users/:uid", "*"},
{"staff", "*", "/api/v1/routers/:id/hotspot/active", "*"},
// ...
```

Sedangkan router Gin mendefinisikan parameter sebagai `:routerId`:
```go
routerOne := protected.Group("/routers/:routerId")
```

`keyMatch2` dari Casbin **tidak peduli nama placeholder** (hanya struktur path), sehingga secara fungsional masih bekerja. Namun, inkonsistensi ini membingungkan dan berisiko saat refactoring.

Selain itu, policy staff untuk `/hotspot/users/:uid` **tidak ada** padanannya di router Gin — route yang ada adalah `/hotspot/users/:id`. Apabila Casbin di masa depan diubah ke matcher yang lebih ketat, policy ini akan invalid.

**Perbaikan:**
```go
// Selaraskan semua placeholder dengan nama yang digunakan di router.go
{"staff", "*", "/api/v1/routers/:routerId/hotspot/users", "*"},
{"staff", "*", "/api/v1/routers/:routerId/hotspot/users/:id", "*"},
{"staff", "*", "/api/v1/routers/:routerId/hotspot/active", "*"},
{"staff", "*", "/api/v1/routers/:routerId/hotspot/inactive", "GET"},
{"staff", "*", "/api/v1/routers/:routerId/hotspot/profiles", "GET"},
{"staff", "*", "/api/v1/routers/:routerId/hotspot/servers", "GET"},
{"staff", "*", "/api/v1/routers/:routerId/vouchers/*", "*"},
{"staff", "*", "/api/v1/routers/:routerId/reports/*", "GET"},
```

---

## Output Test Lengkap

```
=== Testing Superadmin (to Alpha) (superadmin) ===
Login SUCCESS. Token: eyJhbGciOiJIUzI1NiIs...
ERROR: {"data":null,"error":"forbidden"}
Details: {"data":null,"error":"forbidden"}

=== Testing Superadmin (to Beta) (superadmin) ===
Login SUCCESS. Token: eyJhbGciOiJIUzI1NiIs...
ERROR: {"data":null,"error":"forbidden"}
Details: {"data":null,"error":"forbidden"}

=== Testing Admin Alpha (admin.alpha) ===
Login SUCCESS. Token: eyJhbGciOiJIUzI1NiIs...
GET /users SUCCESS. Found 3 users.
GET /users/9 SUCCESS. User role: owner, Username: owner.alpha

=== Testing Staff Alpha (staff.alpha) ===
Login SUCCESS. Token: eyJhbGciOiJIUzI1NiIs...
ERROR: {"data":null,"error":"forbidden"}
Details: {"data":null,"error":"forbidden"}

=== Testing Admin Beta (admin.beta) ===
Login SUCCESS. Token: eyJhbGciOiJIUzI1NiIs...
GET /users SUCCESS. Found 3 users.
GET /users/12 SUCCESS. User role: owner, Username: owner.beta

=== Testing Staff Beta (staff.beta) ===
Login SUCCESS. Token: eyJhbGciOiJIUzI1NiIs...
ERROR: {"data":null,"error":"forbidden"}
Details: {"data":null,"error":"forbidden"}

=== Testing Cross-Tenant: Admin Alpha reading Beta ===
FAIL: Allowed access! Found 3 users.
```

---

## Prioritas Perbaikan

| ID | Severity | Dampak | Upaya | Prioritas |
|---|---|---|---|---|
| BUG-001 | 🔴 Critical | Kebocoran data antar-tenant | Rendah (ubah 1 kondisi) | **P0 — Segera** |
| BUG-002 | 🔴 High | Superadmin tidak bisa beroperasi | Sedang (perbaiki model/policy) | **P1 — Minggu ini** |
| BUG-003 | 🟡 Medium | Password reset diam-diam gagal | Sedang (tambah 1 metode) | **P2 — Sprint ini** |
| BUG-004 | 🟡 Medium | Audit log context unsafe | Rendah (pindahkan 1 baris) | **P2 — Sprint ini** |
| BUG-005 | 🟡 Medium | Router terputus jika dekripsi gagal | Sedang (tambah error handling) | **P2 — Sprint ini** |
| BUG-006 | 🟡 Medium | Logo MIME/ekstensi mismatch | Rendah (gunakan ekstensi asli) | **P3 — Backlog** |
| BUG-007 | 🟢 Low | Cache export stale untuk hari ini | Rendah (tambah TTL check) | **P3 — Backlog** |
| BUG-008 | 🟢 Low | Script RouterOS invalid jika ada spasi | Rendah (tambah quote) | **P3 — Backlog** |
| BUG-009 | 🟢 Low | Dashboard tenant-wide tidak terinvalidasi | Rendah (tambah 1 key) | **P3 — Backlog** |
| BUG-010 | 🟢 Low | Inkonsistensi nama policy | Rendah (rename placeholder) | **P4 — Backlog** |

---

## Catatan Umum

- **`go build ./...`** lulus tanpa error — semua bug bersifat runtime/logic, bukan compile-time
- **Staff yang diblokir dari `/users`** adalah perilaku yang **benar** sesuai desain — staff tidak punya policy untuk endpoint manajemen user
- **Rate limit** di-flush via `FLUSHALL` di awal test — tepat untuk kondisi testing
- Test ini tidak mencakup: endpoint router/hotspot, voucher, laporan, SSE, template, dan profile sync
- BUG-001 dan BUG-002 harus **diprioritaskan segera** sebelum deployment ke production
