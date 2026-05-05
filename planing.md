# Roskit — Schema & Migration Refactor (Pure GORM)

## Konteks

Kamu adalah AI coding agent yang bekerja pada repo Go bernama **roskit** (branch: `improve`).
Repo ini adalah backend ISP management system untuk hotspot MikroTik (RT-RW Net).

Stack: Go 1.24, Gin, GORM v2, PostgreSQL, Redis, `go-routeros/v3`.

---

## Strategi

**Satu sumber kebenaran: GORM models.**
- GORM AutoMigrate adalah satu-satunya yang mengelola schema.
- Semua SQL files di `migrations/` adalah dead code — hapus semuanya.
- Tidak perlu library tambahan seperti `golang-migrate` atau `goose`.

---

## Daftar Masalah yang Harus Diperbaiki

### MASALAH 1 — SQL Files adalah Dead Code

Direktori `migrations/` berisi 7 file `.up.sql` dan `migrations.go` yang hanya mendeklarasikan `embed.FS` tapi tidak pernah dibaca oleh siapapun. Grep konfirmasi:

```
grep -rn "migrations.FS" → hanya muncul di deklarasinya sendiri
```

Yang benar-benar dieksekusi saat `./migrate up` adalah:
```go
// cmd/migrate/main.go
database.AutoMigrate(db)  // ← ini
```

**Fix:** Hapus seluruh direktori `migrations/` beserta semua isinya.

---

### MASALAH 2 — Index Salah pada `ProfilePriceMapping`

GORM model saat ini:
```go
ProfileName string `gorm:"uniqueIndex;size:100;not null"`  // ← unique HANYA profile_name
RouterID    uint   `gorm:"index;not null"`
```

Ini membuat unique index hanya pada `profile_name` saja, sehingga dua router berbeda tidak bisa punya profile dengan nama yang sama (misal "10Mbps"). Sistem multi-router langsung rusak.

**Fix:** Gunakan composite uniqueIndex via nama index yang sama:
```go
RouterID     uint   `gorm:"not null;uniqueIndex:udx_router_profile"`
ProfileName  string `gorm:"size:100;not null;uniqueIndex:udx_router_profile"`
```

---

### MASALAH 3 — Soft-Delete + Non-Partial Unique Index

`SystemUser` menggunakan `gorm.DeletedAt` (soft delete), tapi:
```go
Username string `gorm:"uniqueIndex;size:100;not null"`
```

GORM akan buat `UNIQUE INDEX (username)` tanpa kondisi, sehingga user yang sudah di-soft-delete memblokir pembuatan user baru dengan username yang sama selamanya.

GORM v2 mendukung partial index via tag `where`:
```go
Username string `gorm:"size:100;not null;uniqueIndex:idx_system_users_username,where:deleted_at IS NULL"`
```

**Fix:** Tambahkan `,where:deleted_at IS NULL` pada uniqueIndex untuk semua model yang punya `gorm.DeletedAt`.

Model `Router` juga perlu partial unique index pada field `Name` (pengganti `session_name`).

---

### MASALAH 4 — `idempotency_key` Harus NOT NULL

```go
IdempotencyKey string `gorm:"size:200;uniqueIndex"`  // ← tidak ada not null
```

Tanpa `not null`, string kosong `""` bisa masuk dan akan conflict di unique index saat ada record kedua tanpa key. Idempotency key pada voucher sale selalu di-generate sebelum insert, jadi field ini wajib ada.

**Fix:**
```go
IdempotencyKey string `gorm:"type:char(64);not null;uniqueIndex" json:"idempotency_key"`
```

Gunakan `char(64)` karena key-nya adalah SHA-256 hex (fixed 64 karakter).

---

### MASALAH 5 — `cmd/seed/main.go` Memanggil AutoMigrate

```go
// cmd/seed/main.go
if err := database.AutoMigrate(db); err != nil { ...
```

Migration dan seeding adalah concern berbeda. Seed command tidak boleh mengubah schema.

**Fix:** Hapus pemanggilan `database.AutoMigrate(db)` dari seed command.

---

### MASALAH 6 — Schema `routers` Mencampur Dua Concern

Tabel `routers` saat ini mencampur **koneksi infrastruktur MikroTik** dan **konfigurasi bisnis hotspot**:

```go
type Router struct {
    // Koneksi infrastruktur — OK di sini
    SessionName string
    IP          string
    Username    string
    PasswordEnc string

    // Konfigurasi bisnis — seharusnya di tabel terpisah
    HotspotName string
    DNSName     string
    Currency    string
    Phone       string
    Email       string
    InfoLP      string
    IdleTimeout string
    ReportMode  string
    Token       string
    Timezone    string
    LogoPath    string
}
```

**Fix:** Pisahkan menjadi dua model — `Router` (infra koneksi) dan `HotspotConfig` (konfigurasi bisnis), relasi 1:1.

---

## Model Baru yang Diinginkan

### `internal/models/router_status.go` — FILE BARU

Buat file terpisah untuk tipe status agar bisa di-import dari layer lain tanpa circular dependency:

```go
package models

// RouterStatus merepresentasikan state koneksi RouterOS API.
// Nilai-nilai ini selaras langsung dengan execution.ConnState
// di internal/roskit/execution/helper.go.
type RouterStatus string

const (
    // RouterStatusUnknown adalah state awal saat router baru ditambahkan
    // ke database dan belum pernah ada percobaan koneksi sama sekali.
    RouterStatusUnknown RouterStatus = "unknown"

    // RouterStatusConnecting berarti pool sedang aktif mencoba membangun
    // koneksi TCP ke RouterOS API — baik koneksi pertama maupun
    // reconnect setelah koneksi putus (exponential backoff aktif).
    RouterStatusConnecting RouterStatus = "connecting"

    // RouterStatusConnected berarti kedua persistent connection
    // (client + stream) aktif, terautentikasi, dan siap menerima command.
    // Selaras dengan execution.ConnStateConnected.
    RouterStatusConnected RouterStatus = "connected"

    // RouterStatusDisconnected berarti koneksi pernah aktif tapi saat ini
    // tidak tersambung — host tidak dapat dijangkau, TCP timeout, atau
    // koneksi di-drop oleh router. Auto-reconnect dengan backoff sedang
    // berjalan di background.
    RouterStatusDisconnected RouterStatus = "disconnected"

    // RouterStatusAuthFailed berarti TCP berhasil terhubung tapi
    // RouterOS menolak kredensial (username/password salah).
    // Kondisi ini permanen — tidak akan auto-retry sampai kredensial
    // diperbarui secara manual. Selaras dengan execution.ConnStateAuthFailed.
    RouterStatusAuthFailed RouterStatus = "auth_failed"
)

// String implements the Stringer interface.
func (s RouterStatus) String() string { return string(s) }

// IsTerminal returns true untuk status yang tidak akan berubah
// secara otomatis tanpa intervensi manual (saat ini hanya auth_failed).
func (s RouterStatus) IsTerminal() bool {
    return s == RouterStatusAuthFailed
}

// IsHealthy returns true hanya jika koneksi benar-benar aktif.
func (s RouterStatus) IsHealthy() bool {
    return s == RouterStatusConnected
}
```

Tambahkan juga fungsi konversi dari `execution.ConnState` ke `RouterStatus`.
Letakkan di file yang sama atau di file adapter, **bukan** di package `execution`
(untuk menjaga agar execution layer tidak tahu tentang models):

```go
// RouterStatusFromConnState mengkonversi execution.ConnState ke RouterStatus
// untuk disimpan ke database. Dipanggil oleh health-check goroutine
// setiap kali state koneksi berubah.
//
// Catatan: RouterStatusUnknown tidak punya padanan di ConnState karena
// ConnState hanya ada selama pool berjalan — Unknown hanya ada di DB
// sebelum router pernah di-register ke pool.
func RouterStatusFromConnState(state execution.ConnState) RouterStatus {
    switch state {
    case execution.ConnStateConnected:
        return RouterStatusConnected
    case execution.ConnStateConnecting:
        return RouterStatusConnecting
    case execution.ConnStateAuthFailed:
        return RouterStatusAuthFailed
    default: // ConnStateDisconnected
        return RouterStatusDisconnected
    }
}
```

### `internal/models/router.go`

```go
package models

import (
    "time"
    "gorm.io/gorm"
)

type Router struct {
    ID                   uint           `gorm:"primaryKey"                                                              json:"id"`
    Name                 string         `gorm:"size:100;not null;uniqueIndex:idx_routers_name,where:deleted_at IS NULL" json:"name"`
    IPAddress            string         `gorm:"size:45;not null"                                                        json:"ip_address"`
    APIPort              int            `gorm:"not null;default:8728"                                                   json:"api_port"`
    APIUsername          string         `gorm:"size:100;not null"                                                       json:"api_username"`
    APIPasswordEncrypted string         `gorm:"not null"                                                                json:"-"`
    SSHPort              *int           `gorm:"default:22"                                                              json:"ssh_port"`
    SSHUsername          *string        `gorm:"size:50"                                                                 json:"ssh_username"`
    SSHPasswordEncrypted *string        `                                                                               json:"-"`
    // Status merepresentasikan state koneksi RouterOS API saat ini.
    // Nilai valid: lihat konstanta RouterStatus di router_status.go.
    // Di-update oleh health-check goroutine setiap kali state berubah,
    // bukan oleh request HTTP.
    Status               RouterStatus   `gorm:"type:varchar(20);not null;default:unknown"                               json:"status"`
    // LastSeenAt di-update setiap kali koneksi berhasil (status → connected).
    LastSeenAt           *time.Time     `                                                                               json:"last_seen_at"`
    Notes                *string        `gorm:"type:text"                                                               json:"notes"`
    CreatedAt            time.Time      `                                                                               json:"created_at"`
    UpdatedAt            time.Time      `                                                                               json:"updated_at"`
    DeletedAt            gorm.DeletedAt `gorm:"index"                                                                   json:"-"`

    HotspotConfig *HotspotConfig `gorm:"foreignKey:RouterID" json:"hotspot_config,omitempty"`
}
```

### `internal/models/hotspot_config.go` — FILE BARU

```go
package models

import "time"

type HotspotConfig struct {
    ID           uint      `gorm:"primaryKey"                      json:"id"`
    RouterID     uint      `gorm:"not null;uniqueIndex"             json:"router_id"`
    HotspotName  string    `gorm:"size:100"                        json:"hotspot_name"`
    DNSName      string    `gorm:"size:100"                        json:"dns_name"`
    Currency     string    `gorm:"size:10;not null;default:Rp"     json:"currency"`
    Phone        string    `gorm:"size:20"                         json:"phone"`
    Email        string    `gorm:"size:100"                        json:"email"`
    InfoLP       string    `gorm:"type:text"                       json:"info_lp"`
    IdleTimeout  int       `gorm:"not null;default:30"             json:"idle_timeout"`
    ReportMode   string    `gorm:"size:20;not null;default:disable" json:"report_mode"`
    WebhookToken string    `gorm:"size:255"                        json:"-"`
    LogoPath     string    `gorm:"type:text;not null;default:''"   json:"logo_path"`
    Timezone     string    `gorm:"size:50;not null;default:''"     json:"timezone"`
    CreatedAt    time.Time `                                        json:"created_at"`
    UpdatedAt    time.Time `                                        json:"updated_at"`

    Router Router `gorm:"foreignKey:RouterID" json:"-"`
}
```

### `internal/models/profile_price_mapping.go`

```go
package models

type ProfilePriceMapping struct {
    ID           uint   `gorm:"primaryKey"                               json:"id"`
    RouterID     uint   `gorm:"not null;uniqueIndex:udx_router_profile"  json:"router_id"`
    ProfileName  string `gorm:"size:100;not null;uniqueIndex:udx_router_profile" json:"profile_name"`
    Price        int64  `gorm:"not null;default:0"                       json:"price"`
    SellingPrice int64  `gorm:"not null;default:0"                       json:"selling_price"`
    Validity     string `gorm:"size:20"                                  json:"validity"`
    ExpMode      string `gorm:"size:10"                                  json:"exp_mode"`
    LockUser     bool   `gorm:"not null;default:false"                   json:"lock_user"`
    LockServer   bool   `gorm:"not null;default:false"                   json:"lock_server"`

    Router Router `gorm:"foreignKey:RouterID" json:"-"`
}
```

### `internal/models/voucher_sale.go`

```go
package models

import "time"

type VoucherSale struct {
    ID             uint      `gorm:"primaryKey"                         json:"id"`
    RouterID       uint      `gorm:"not null;index"                     json:"router_id"`
    SoldAt         time.Time `gorm:"not null;index"                     json:"sold_at"`
    Username       string    `gorm:"size:100;not null;index"            json:"username"`
    ProfileName    string    `gorm:"size:100;not null;index"            json:"profile_name"`
    Price          int64     `gorm:"not null;default:0"                 json:"price"`
    SellingPrice   int64     `gorm:"not null;default:0"                 json:"selling_price"`
    Server         string    `gorm:"size:100;index"                     json:"server"`
    IPAddress      string    `gorm:"size:45"                            json:"ip_address"`
    MACAddress     string    `gorm:"size:17"                            json:"mac_address"`
    Validity       string    `gorm:"size:20"                            json:"validity"`
    IdempotencyKey string    `gorm:"type:char(64);not null;uniqueIndex" json:"idempotency_key"`
    CreatedAt      time.Time `                                           json:"created_at"`

    Router Router `gorm:"foreignKey:RouterID" json:"-"`
}
```

### `internal/models/system_user.go`

```go
package models

import (
    "time"
    "gorm.io/gorm"
)

type SystemUser struct {
    ID           uint           `gorm:"primaryKey"                                                                            json:"id"`
    Username     string         `gorm:"size:100;not null;uniqueIndex:idx_system_users_username,where:deleted_at IS NULL"     json:"username"`
    PasswordHash string         `gorm:"size:255;not null"                                                                     json:"-"`
    Role         string         `gorm:"size:20;not null;default:admin"                                                        json:"role"`
    Active       bool           `gorm:"not null;default:true"                                                                 json:"active"`
    LastLoginAt  *time.Time     `                                                                                             json:"last_login_at"`
    CreatedAt    time.Time      `                                                                                             json:"created_at"`
    UpdatedAt    time.Time      `                                                                                             json:"updated_at"`
    DeletedAt    gorm.DeletedAt `gorm:"index"                                                                                 json:"-"`
}
```

---

## Instruksi untuk Agent

### Langkah 1 — Hapus Direktori `migrations/`

```bash
rm -rf migrations/
```

Pastikan tidak ada import `"github.com/quiqxiq/roskit/migrations"` yang tersisa di codebase.

### Langkah 2 — Update `pkg/database/postgres.go`

Tambahkan `HotspotConfig` ke daftar `AutoMigrate`:

```go
func AutoMigrate(db *gorm.DB) error {
    return db.AutoMigrate(
        &models.Router{},
        &models.HotspotConfig{},
        &models.VoucherSale{},
        &models.ProfilePriceMapping{},
        &models.SystemUser{},
        &models.AuditLog{},
        &models.PrintTemplate{},
    )
}
```

### Langkah 3 — Buat File Model Baru

Buat `internal/models/hotspot_config.go` dengan isi seperti di atas.

### Langkah 4 — Update Semua Model yang Ada

Ganti isi file model sesuai definisi baru. Perhatikan perubahan utama:
- `router.go`: field bisnis dihapus, field koneksi diperbarui namanya
- `profile_price_mapping.go`: uniqueIndex diperbaiki jadi composite
- `voucher_sale.go`: `IdempotencyKey` menjadi `char(64);not null`
- `system_user.go`: uniqueIndex ditambahkan `where:deleted_at IS NULL`

### Langkah 5 — Hapus AutoMigrate dari `cmd/seed/main.go`

Hapus blok ini:
```go
if err := database.AutoMigrate(db); err != nil {
    log.Fatalf(...)
}
```

### Langkah 6 — Update Semua Kode yang Mereferensikan Field Lama

Jalankan grep untuk menemukan semua referensi ke field lama:
```bash
grep -rn "SessionName\|\.IP\b\|PasswordEnc\|HotspotName\|DNSName\|\.Token\b\|IdleTimeout\|ReportMode\|\.Currency\b\|\.Phone\b\|\.Email\b\|InfoLP\|LogoPath\|\.Timezone\b" --include="*.go"
```

Mapping field lama ke baru:

| Field lama (`r.X`) | Ganti dengan |
|---|---|
| `r.SessionName` | `r.Name` |
| `r.IP` | `r.IPAddress` |
| `r.Username` | `r.APIUsername` |
| `r.PasswordEnc` | `r.APIPasswordEncrypted` |
| `r.HotspotName` | `r.HotspotConfig.HotspotName` |
| `r.DNSName` | `r.HotspotConfig.DNSName` |
| `r.Currency` | `r.HotspotConfig.Currency` |
| `r.Phone` | `r.HotspotConfig.Phone` |
| `r.Email` | `r.HotspotConfig.Email` |
| `r.InfoLP` | `r.HotspotConfig.InfoLP` |
| `r.IdleTimeout` | `r.HotspotConfig.IdleTimeout` |
| `r.ReportMode` | `r.HotspotConfig.ReportMode` |
| `r.Token` | `r.HotspotConfig.WebhookToken` |
| `r.Timezone` | `r.HotspotConfig.Timezone` |
| `r.LogoPath` | `r.HotspotConfig.LogoPath` |

### Langkah 7 — Update `runImport` di `cmd/migrate/main.go`

Fungsi ini mem-parse config.php lama dan membuat struct `models.Router` dengan field campuran. Setelah refactor, pisahkan menjadi dua struct dan simpan dalam satu transaksi:

```go
err := db.Transaction(func(tx *gorm.DB) error {
    router := models.Router{
        Name:                 sessionName,
        IPAddress:            ipStr,
        APIUsername:          userStr,
        APIPasswordEncrypted: encryptedPassword,
    }
    if err := tx.Create(&router).Error; err != nil {
        return err
    }
    config := models.HotspotConfig{
        RouterID:     router.ID,
        HotspotName:  hotspotStr,
        DNSName:      dnsStr,
        Currency:     currencyStr,
        Phone:        phoneStr,
        Email:        emailStr,
        InfoLP:       infoStr,
        IdleTimeout:  idleInt, // parse string ke int
        ReportMode:   reportStr,
        WebhookToken: tokenStr,
    }
    return tx.Create(&config).Error
})
```

Query check duplikat juga perlu diupdate dari `session_name` ke `name`:
```go
// Lama
db.Where("session_name = ?", rawRouter.SessionName).First(&r)
// Baru
db.Where("name = ?", router.Name).First(&r)
```

### Langkah 8 — Update Connection Pool

Semua tempat yang membangun koneksi MikroTik menggunakan `r.IP` hardcode port:

```go
// Lama
pool.Register(execution.ConnConfig{
    Address:  r.IP + ":8728",
    Username: r.Username,
    Password: cleartextPassword,
})

// Baru — gunakan field APIPort
pool.Register(execution.ConnConfig{
    Address:  fmt.Sprintf("%s:%d", r.IPAddress, r.APIPort),
    Username: r.APIUsername,
    Password: cleartextPassword,
})
```

Query router yang butuh field HotspotConfig harus menggunakan Preload:
```go
db.Preload("HotspotConfig").Where("name = ?", routerName).Find(&routers)
```

---

## Validasi Akhir

- [ ] `go build ./...` tidak ada error kompilasi
- [ ] `go test ./...` semua test pass
- [ ] Kolom `status` di tabel `routers` hanya menerima nilai: `unknown`, `connecting`, `connected`, `disconnected`, `auth_failed`
- [ ] `RouterStatusFromConnState` menghasilkan nilai benar untuk semua 4 nilai `execution.ConnState`
- [ ] Router baru memiliki `status = "unknown"` dan `last_seen_at = NULL`
- [ ] Setelah koneksi berhasil: status → `"connected"` dan `last_seen_at` ter-update
- [ ] Setelah koneksi putus: status → `"disconnected"`, bukan tetap `"connected"`
- [ ] Kredensial salah: status → `"auth_failed"` dan tidak berubah sendiri (IsTerminal = true)
- [ ] `./migrate up` di database baru: semua tabel terbentuk dengan benar
- [ ] Tabel `routers` tidak punya kolom `hotspot_name`, `dns_name`, dll
- [ ] Tabel `hotspot_configs` terbentuk dengan `router_id` UNIQUE
- [ ] `profile_price_mappings`: dua router berbeda bisa punya profile "10Mbps" (composite index benar)
- [ ] `system_users`: user yang di-soft-delete bisa dibuat ulang dengan username sama (partial index benar)
- [ ] `voucher_sales`: insert record dengan `idempotency_key` kosong harus error
- [ ] `cmd/seed/main.go` tidak memanggil `AutoMigrate` lagi
- [ ] Direktori `migrations/` tidak ada
- [ ] Tidak ada import `"github.com/quiqxiq/roskit/migrations"` di seluruh codebase