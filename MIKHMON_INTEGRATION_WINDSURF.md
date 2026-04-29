# MIKHMON_INTEGRATION_PLAN_WINDSURF.md

> Rencana migrasi & paritas fitur **Mikhmon v3 (PHP)** ke backend Go **roskit**. Dokumen ini bersifat plan-only — tidak ada kode yang dimodifikasi sampai disetujui.

---

## 1. Project Architecture Summary

### Stack & Module
- **Module**: `github.com/quiqxiq/roskit` · Go 1.24
- **HTTP**: Gin · **ORM**: GORM · **DB**: PostgreSQL 16
- **Realtime**: Redis 7 (cache + pub-sub) · **Time-series**: InfluxDB 3 (opsional)
- **RouterOS**: `go-routeros/v3`

### Entry Points
- `cmd/api` — HTTP server (port 8080)
- `cmd/migrate` — schema migration & import `config.php`
- `cmd/worker` — background worker
- `cmd/seed` — seed data

### Layering (top → bottom)
1. **Handler** (`internal/api/handlers/`) — bind/validate/respond. Envelope: `{"data":..., "error":...}`
2. **Service** (`internal/services/`) — orkestrasi via Bridge + repository + cache app
3. **Repository** (`internal/repository/`) — GORM, no business logic
4. **Bridge** (`internal/roskit/adapter/service/`) — fasad single-entry ke roskit
5. **Roskit core** — pure RouterOS communication, classified by behavior

### Roskit Subsystem
```
core/         — pure (command registry, parser, model). NO Redis/Influx/network.
behavior/     — stream / poll / query / mutation handlers
execution/    — TCP pool (sync conn + async conn per router)
pipeline/     — cache (Redis), event processor, pubsub, timeseries (Influx)
orchestrator/ — engine lifecycle, dispatcher, router manager
adapter/service/ — Bridge (only entry point services may use)
```

### Konvensi Wajib
- Services **HANYA** inject `*service.Bridge`. Tidak boleh sentuh `*execution.Pool` atau `*routeros.Client` langsung.
- Router ID = `string` di Bridge/roskit (`fmt.Sprintf("%d", id)`), `uint` di URL/DB.
- DB hanya menyimpan: `voucher_sales`, `routers`, `system_users`, `audit_logs`, `print_templates`, `profile_price_mappings`. **DILARANG** membuat model GORM untuk `hotspot_users`, `queues`, `interfaces`, `active_sessions` — RouterOS adalah source of truth.
- Dua sistem cache:
  - **Cache A** (`pipeline/cache`, key `roskit:{routerID}:{measurement}:{id}`) — auto, 5 min TTL, dikelola streaming pipeline.
  - **Cache B** (`pkg/redis`, key `mikhmon:{resource}:{id}`) — manual invalidate setelah mutation, dikelola service layer.
- Error: `pkg/errors.AppError` (`NewNotFound`, `NewBadRequest`, `NewUnauthorized`, `NewInternal`).
- Tanpa komentar di kode kecuali diminta. Tanpa raw SQL kecuali agregasi.

### Aliran Data
- **Write**: RouterOS → `execution.Pool` → stream/poll worker → parser → `event.Processor` → Redis HSET + InfluxDB + Redis PUBLISH
- **Read**: `Bridge.Query()` → `Dispatcher` → query handler → Redis snapshot dulu, fallback ke RouterOS live

---

## 2. Mikhmon Feature Inventory

Status: ✅ Done · ⚠️ Partial · ❌ Missing

### Auth & Multi-Router
| Fitur | Deskripsi | Status | Prioritas |
|---|---|---|---|
| Login admin | JWT access + refresh, setup admin pertama | ✅ Done | — |
| Multi-router config | CRUD router (ip, user, pass enkripsi AES) | ✅ Done | — |
| Migrasi `config.php` | Import flat-file legacy mikhmon | ✅ Done | — |
| Test koneksi router | Ping + identity + version | ✅ Done | — |
| Logo per session | Upload `img/logo-{session}.png` untuk voucher | ❌ Missing | Medium |

### Dashboard
| Fitur | Deskripsi | Status | Prioritas |
|---|---|---|---|
| System resource | CPU, RAM, HDD, uptime | ✅ Done (`GET /system/resource`) | — |
| Identity & RouterBoard | board, model, RouterOS version | ✅ Done | — |
| Hotspot count (active + total) | counter user & active | ✅ Done (off-by-one rule diikuti) | — |
| Live report (today/month) | pendapatan + jumlah voucher | ✅ Done (`/reports/summary`) | — |
| Hotspot log feed | log realtime | ✅ Done (SSE `/logs/stream/hotspot`) | — |
| Traffic chart (Tx/Rx) | Highcharts areaspline 8s interval | ✅ Done (SSE `interface_traffic`) | — |
| Resource history chart | InfluxDB 15m–30d range | ✅ Done (`/system/resource/history`) | — |

### Hotspot — User
| Fitur | Deskripsi | Status | Prioritas |
|---|---|---|---|
| List user (filter profile) | Tabel user dengan filter | ✅ Done | — |
| Add user (manual) | CRUD detail | ✅ Done | — |
| Update user (extend expired) | edit detail | ✅ Done (raw map) | Medium (helper extend belum) |
| Remove user + cleanup | hapus user + scheduler + script | ✅ Done | — |
| Generate user (bulk voucher) | qty, prefix, char-set, profile | ✅ Done | — |
| Quick Print (preset) | simpan paket di `/system/script` | ✅ Done (CRUD) | — |
| **Generate from Quick Print** | gunakan paket sebagai input generate | ⚠️ Partial | High |
| Quick User (single-shot add) | kombinasi generate+print 1 user | ❌ Missing | Low |
| Export users (CSV / .rsc) | export massal | ✅ Done | — |
| Reset counters | reset uptime/bytes counter | ⚠️ Partial (Bridge ✅, handler ❌) | Medium |
| Enable / Disable user | flag disabled=yes/no | ⚠️ Partial (Bridge ✅, handler ❌) | Medium |
| Remove by comment (batch) | hapus semua user dengan comment X | ❌ Missing | Medium |
| Remove expired (batch) | hapus user dengan comment expired | ❌ Missing | Medium |
| Users by profile (group view) | grup berdasarkan profile | ⚠️ Partial (filter ada, view khusus belum) | Low |

### Hotspot — Profile (Paket)
| Fitur | Deskripsi | Status | Prioritas |
|---|---|---|---|
| List/Get/Add/Update/Remove | CRUD lengkap | ✅ Done | — |
| `on-login` script generator | ExpMode (ntf/ntfc/rem/remc/0), Price, SellingPrice, Validity, LockUser, LockServer | ✅ Done (paritas penuh) | — |
| Parse `:put` metadata | ekstrak harga, validity dari profile lama | ✅ Done | — |
| Address pool & Parent queue | atribut profile | ✅ Done (raw forward) | — |
| Expire monitor scheduler | global scheduler `Mikhmon-Expire-Monitor` | ✅ Done (deploy/remove/check) | — |

### Hotspot — Active / Hosts / Cookies / IP Binding
| Fitur | Status | Prioritas |
|---|---|---|
| Active list (filter server) | ✅ Done | — |
| Disconnect active | ✅ Done | — |
| Hosts list / remove | ✅ Done | — |
| Cookies list / remove | ✅ Done | — |
| IP Binding CRUD + enable/disable + cleanup | ✅ Done | — |
| Hotspot servers list | ✅ Done | — |
| Walled Garden | ⚠️ Bridge `ListWalledGarden` ada, handler ❌ | Low |

### Reports — Selling
| Fitur | Status | Prioritas |
|---|---|---|
| Daily report (filter day/month/year) | ✅ Done (DB-based) | — |
| Monthly report | ✅ Done | — |
| Resume report (12-month) | ✅ Done | — |
| Filter prefix username | ✅ Done (`?search=`) | — |
| Filter server | ✅ Done (`?server=`) | — |
| Filter profile | ✅ Done (`?profile=`) | — |
| Export CSV | ✅ Done | — |
| Export Excel | ✅ Done (extra dari mikhmon) | — |
| Print HTML report | ❌ Missing | Medium |
| Remove report records | ⚠️ Bridge `RemoveSaleRecord` / `RemoveAllSalesForMonth` ada (RouterOS-side); endpoint untuk DB sales ❌ | Medium |
| User log report (session history) | ❌ Missing | Low |
| On-login event ingestion | ✅ Done (`POST /events/on-login`) + import dari RouterOS scripts | — |

### Voucher / Print Template
| Fitur | Status | Prioritas |
|---|---|---|
| Generate voucher (vc/up modes, charsets, prefix) | ✅ Done | — |
| Cache voucher batch (gencode) | ✅ Done | — |
| Print-data endpoint (info paket) | ✅ Done | — |
| Record sale (manual + idempotent) | ✅ Done | — |
| Import sales dari RouterOS scripts | ✅ Done | — |
| Print template (default/small/thermal) | ⚠️ Partial — model DB header/row/footer berbeda paradigma dari mikhmon (3 file PHP fixed) | High |
| Default templates seeded | ❌ Missing | High |
| QR code rendering | ⚠️ Partial — pakai `api.qrserver.com` (eksternal); mikhmon pakai QRious lokal | Medium |
| **Printable HTML page (full doc)** | ❌ Missing — `/templates/render` hanya kembalikan potongan HTML, tidak full document siap cetak | High |
| Voucher preview | ❌ Missing | Low |
| Bluetooth print payload (`printbt.php`) | ❌ Missing | Low |
| Voucher template editor (CodeMirror) | ⚠️ Partial — endpoint update template sudah ada, UI editor di luar scope backend | Low |

### Network / DHCP / PPP / System
| Fitur | Status | Prioritas |
|---|---|---|
| Interfaces list + traffic monitor | ✅ Done (REST + SSE) | — |
| Address pools, queues, NAT | ✅ Done (read-only list) | — |
| DHCP leases list + release | ✅ Done | — |
| PPP secrets CRUD | ✅ Done | — |
| PPP secret enable/disable | ⚠️ Partial (Bridge ✅, handler ❌) | Medium |
| PPP active + disconnect | ✅ Done | — |
| PPP profiles CRUD | ⚠️ Partial — Bridge ✅ (Add/Set/Remove/Enable/Disable), handler hanya List | Medium |
| System reboot / shutdown | ✅ Done | — |
| System schedulers — list | ✅ Done | — |
| System schedulers — full CRUD | ❌ Missing (mikhmon punya add/edit/remove/enable/disable scheduler) | Medium |
| System scripts — manage | ⚠️ Partial — hanya untuk QuickPrint/sales/expire-monitor; tidak ada CRUD generik | Low |
| Ping test (`status/ping-test.php`) | ❌ Missing | Low |

### Settings / i18n / Theme
| Fitur | Status | Prioritas |
|---|---|---|
| Setting per-router (currency, hotspot name, dns, idle, phone, email, info_lp, report_mode) | ✅ Done | — |
| Logo upload | ❌ Missing | Medium |
| Multi-language (id/en/es/tl/tr) | ❌ Missing (di-defer ke frontend) | Low |
| Theme switching | ❌ Missing (di-defer ke frontend) | Low |

---

## 3. Gap Analysis

### 3.1 Print Template & Voucher Printing — Gap Utama (High)
**Yang sudah ada**:
- Model `print_templates` (id, name, type, part [header|row|footer], content, router_id) di `@/Users/.../internal/models/print_template.go`.
- `TemplateService.Render()` me-render setiap voucher dengan susunan `header+row+footer` melalui Go `html/template`.
- `RenderParams` punya field `HotspotName`, `DNSName`, `Logo`, `UserMode`, `Currency`, `Profile`, `Validity`, `TimeLimit`, `DataLimit`, `Price`, `Comment`.
- Endpoint `POST /routers/:id/templates/render` sudah ada.

**Yang kurang**:
1. **Tidak ada full HTML document endpoint** seperti `voucher/print.php` di mikhmon. Frontend dapat HTML potongan tapi tidak ada `<html><head><style>...</style></head><body onload="window.print()">...</body></html>` yang siap dibuka di tab baru.
2. **Tidak ada default templates seeded** — DB kosong setelah migrate. User baru tidak punya template apa pun untuk dirender.
3. **QR code** memakai `api.qrserver.com` — paritas dengan mikhmon adalah QRious lokal (offline).
4. **Tidak ada flow combined**: generate vouchers → render → return printable HTML dalam satu round-trip.

**Dependensi**:
- Butuh seed migration untuk 3 template default (`default`, `small`, `thermal`) × 3 part (`header`, `row`, `footer`) = 9 baris seed.
- Butuh endpoint baru `POST /routers/:id/vouchers/print` yang menggabungkan generate (atau ambil dari cache via `gencode`) + render template + bungkus full HTML.
- QR code lokal: bundle `qrious` di `website/js/` (sudah ada `mikhmon/js/qrious.min.js` yang bisa di-copy) atau gunakan Go library `github.com/skip2/go-qrcode` untuk render PNG base64 inline di server.

### 3.2 Hotspot User — Bulk & Lifecycle Operations (Medium)
**Yang sudah ada**:
- `Bridge.EnableHotspotUser`, `DisableHotspotUser`, `ResetUserCounters` ✅
- `RemoveHotspotUserWithCleanup` ✅

**Yang kurang**:
1. Handler tidak expose enable/disable/reset untuk hotspot user (hanya untuk IP binding).
2. Tidak ada "remove by comment" batch endpoint (mikhmon `removehotspotuserbycomment.php`).
3. Tidak ada "remove expired" batch endpoint.

**Dependensi**: tidak ada — semua method Bridge sudah lengkap.

### 3.3 Quick Print — Generate Workflow (High)
**Yang sudah ada**:
- CRUD `QuickPrintPackage` di `/system/script` (Bridge `SaveQuickPrintPackage` dst.).
- Schema `#name#server#userMode#userLength#prefix#charMode#profile#timeLimit#dataLimit#comment#validity#price_sellingPrice#lockUser`.

**Yang kurang**:
1. Tidak ada endpoint `POST /routers/:id/quick-print/:name/generate` yang membaca paket lalu memanggil `voucherSvc.GenerateVoucher` dengan field paket sebagai input.
2. Frontend mikhmon `listquickprint.php` punya tombol "Generate" pada setiap baris paket — paritas backend ini belum ada.

**Dependensi**: Quick Print CRUD sudah ada; tinggal compose service-level glue.

### 3.4 Reports — Print HTML & Delete Records (Medium)
**Yang sudah ada**:
- CSV/Excel export.
- `Bridge.RemoveSaleRecord` dan `RemoveAllSalesForMonth` (RouterOS-side script).

**Yang kurang**:
1. Tidak ada endpoint print HTML report (mikhmon `report/print.php`).
2. Tidak ada endpoint `DELETE /routers/:id/reports/sales` untuk menghapus DB-side sales (atau bulk by month). Mengingat sumber kebenaran sekarang adalah DB (bukan `/system/script`), endpoint hapus DB-side wajib.

**Dependensi**: butuh `SaleRepository.DeleteByMonth` dan `DeleteByDay`.

### 3.5 PPP — Profile CRUD & Secret Lifecycle (Medium)
**Yang sudah ada**: Bridge lengkap (Add/Set/Remove/Enable/Disable secret + profile).

**Yang kurang**: Handler tidak expose:
- `POST/PUT/DELETE /ppp/profiles`
- `POST /ppp/secrets/:id/enable|disable`

**Dependensi**: tidak ada.

### 3.6 System Scheduler CRUD (Medium)
**Yang sudah ada**: `ListSchedulers`, `EnableScheduler`, `DisableScheduler`.

**Yang kurang**: Add/Set/Remove scheduler generic. Mikhmon `system/scheduler.php` punya UI lengkap.

**Dependensi**: tidak ada.

### 3.7 Logo Upload (Medium)
**Yang sudah ada**: Field `HotspotName`, `DNSName`, `Currency` di model Router.

**Yang kurang**:
1. Endpoint `POST /routers/:id/logo` (multipart upload).
2. Endpoint `GET /routers/:id/logo` (serve PNG/JPG).
3. Storage path/strategy (filesystem `./uploads/logos/{routerID}.png` atau S3-compatible). 
4. Field `LogoPath` di model Router (atau gunakan konvensi path).

**Dependensi**: keputusan strategi storage.

### 3.8 Walled Garden / Ping Test / User Log (Low)
- Walled garden: `ListWalledGarden` ada di Bridge, butuh handler.
- Ping test: butuh `Bridge.Ping(routerID, target)` baru memanggil `/ping` RouterOS dengan `count=4`.
- User log report: butuh aggregator dari `/log/print` filter `topics=hotspot,info` parsing pesan login/logout.

---

## 4. Implementation Plan

> Diurutkan berdasarkan dependensi & prioritas. Setiap task patuh layering project (Handler → Service → Bridge/Repo) dan envelope JSON `{data, error}`.

### Task 1 — Hotspot User Lifecycle Endpoints
**Scope**: Expose enable/disable/reset-counters + bulk delete by comment + bulk delete expired untuk hotspot user.
- **Files modify**:
  - `@/Users/.../internal/services/hotspot_service.go` — tambah `EnableUser`, `DisableUser`, `ResetCounters`, `RemoveByComment`, `RemoveExpired`.
  - `@/Users/.../internal/api/handlers/hotspot_handler.go` — handler baru.
  - `@/Users/.../internal/api/router.go` — daftar route:
    - `POST /routers/:id/hotspot/users/:uid/enable`
    - `POST /routers/:id/hotspot/users/:uid/disable`
    - `POST /routers/:id/hotspot/users/:uid/reset`
    - `POST /routers/:id/hotspot/users/bulk/remove-by-comment` (body: `{"comment":"..."}`)
    - `POST /routers/:id/hotspot/users/bulk/remove-expired`
- **DB**: tidak ada perubahan.
- **MikroTik via roskit**: `Bridge.EnableHotspotUser` / `DisableHotspotUser` / `ResetUserCounters`. Untuk bulk: `ListHotspotUsers` → loop → `RemoveHotspotUserWithCleanup`. Untuk expired: parsing `comment` (gunakan `ParseUserComment`) bandingkan dengan `time.Now()` lalu hapus.
- **Redis**: tidak perlu (Cache A telemetry akan refresh otomatis via stream worker).
- **Complexity**: Low.

### Task 2 — PPP Profile CRUD + Secret Enable/Disable Handlers
**Scope**: Expose CRUD profile + secret lifecycle yang sudah ada di Bridge.
- **Files modify**:
  - `@/Users/.../internal/api/handlers/ppp_handler.go` — `AddProfile`, `UpdateProfile`, `RemoveProfile`, `EnableSecret`, `DisableSecret`.
  - `@/Users/.../internal/api/router.go` — route baru di group `/routers/:id/ppp/`.
- **DB**: tidak.
- **MikroTik via roskit**: `Bridge.AddPPPProfile` / `SetPPPProfile` / `RemovePPPProfile` / `EnablePPPSecret` / `DisablePPPSecret`.
- **Redis**: tidak.
- **Complexity**: Low.

### Task 3 — Sales Report: Delete + Print HTML
**Scope**: 
- DB-side delete records (single, by-day, by-month).
- HTML print view endpoint.
- **Files create/modify**:
  - `@/Users/.../internal/repository/sale_repo.go` — tambah `Delete(id)`, `DeleteByDay(routerID, date)`, `DeleteByMonth(routerID, year, month)`.
  - `@/Users/.../internal/services/report_service.go` — tambah `DeleteSale`, `DeleteByDay`, `DeleteByMonth` + invalidate Cache B.
  - `@/Users/.../internal/services/report_service.go` — tambah `RenderPrintHTML(ctx, routerID, from, to, filters) ([]byte, error)` (Go `html/template`).
  - `@/Users/.../internal/api/handlers/report_handler.go` + router:
    - `DELETE /routers/:id/reports/sales/:saleId`
    - `DELETE /routers/:id/reports/sales?day=YYYY-MM-DD` atau `?month=YYYY-MM`
    - `GET /routers/:id/reports/print?from=...&to=...` → `text/html`
- **DB**: cuma operasi `DELETE WHERE`.
- **MikroTik**: tidak, sumber data DB.
- **Redis**: invalidate `appcache.SalesKey(routerID, "today"/"month"/"day:..."/"month:...")` + `DashboardKey`.
- **Complexity**: Medium.

### Task 4 — Quick Print → Generate Bridge
**Scope**: Endpoint `POST /routers/:id/quick-print/:name/generate?qty=N` yang merangkai paket Quick Print menjadi `VoucherGenerateParams` lalu memanggil `VoucherService.GenerateVoucher`.
- **Files create/modify**:
  - `@/Users/.../internal/services/voucher_service.go` — tambah `GenerateFromQuickPrint(ctx, routerID, packageName, qty, gencode) (*VoucherGenerateResult, error)`.
  - `@/Users/.../internal/api/handlers/quick_print_handler.go` — handler baru `GenerateFromPackage`.
  - `@/Users/.../internal/api/router.go` — route `POST /routers/:id/quick-print/:name/generate`.
- **DB**: tidak.
- **MikroTik**: `Bridge.GetQuickPrintPackage` (dari `/system/script`) → `Bridge.GenerateAndCreateVouchers`.
- **Redis**: cache voucher session (sudah dilakukan `GenerateVoucher`).
- **Complexity**: Low.

### Task 5 — Default Print Templates Seed + Migration
**Scope**: Seed 3 template (default, small, thermal) × 3 part (header, row, footer) saat router pertama kali dibuat (atau via endpoint manual seed).
- **Files create/modify**:
  - `@/Users/.../migrations/006_default_print_templates.up.sql` — TIDAK; templates per-router, jadi seeding lewat code lebih natural.
  - `@/Users/.../internal/services/template_service.go` — tambah `SeedDefaultsForRouter(ctx, routerID) error` yang insert 9 baris jika belum ada.
  - `@/Users/.../internal/services/router_service.go` — panggil `templateSvc.SeedDefaultsForRouter` di `CreateRouter` setelah `repo.Create` sukses.
  - Konstanta template default di `@/Users/.../internal/services/template_defaults.go` (file baru) — port dari `mikhmon/voucher/default.php`, `default-small.php`, `default-thermal.php`, dengan substitusi variabel ke Go template (`{{.Username}}`, `{{.Password}}`, dll.).
  - Endpoint manual seed: `POST /routers/:id/templates/seed-defaults` (idempotent).
- **DB**: tabel `print_templates` sudah ada.
- **MikroTik**: tidak.
- **Redis**: tidak.
- **Complexity**: Medium (port template HTML PHP → Go html/template).

### Task 6 — Voucher Print Full HTML Endpoint
**Scope**: Endpoint yang me-render full document siap cetak: `<html><head><style>...</style><script>QRious...</script></head><body onload="window.print()">{{vouchers}}</body></html>`.
- **Files create/modify**:
  - `@/Users/.../internal/services/template_service.go` — tambah `RenderFullPrintHTML(ctx, routerID, templateType, vouchers, params) ([]byte, error)`. Wrap output `Render` dengan boilerplate HTML, embed CSS print media query, embed `<script src="/js/qrious.min.js">` (atau inline data-uri PNG QR jika pakai `go-qrcode`).
  - `@/Users/.../internal/api/handlers/template_handler.go` + `voucher_handler.go` — endpoint baru:
    - `GET /routers/:id/vouchers/print?gencode={code}&template={default|small|thermal}` → `text/html` siap cetak. Ambil voucher dari Cache B (`appcache.VoucherSessionKey`), enrich dari profile RouterOS, render.
  - `@/Users/.../website/js/qrious.min.js` — copy dari `@/Users/.../mikhmon/js/qrious.min.js`.
- **DB**: read templates.
- **MikroTik**: read profile (`Bridge.GetHotspotProfile`) untuk dapat validity, price, dll. (Cache A snapshot OK).
- **Redis**: read voucher session cache.
- **Complexity**: High (HTML wrapping, QR strategy decision, cross-template testing).

### Task 7 — Logo Upload per Router
**Scope**: Multipart upload + serve.
- **Files create/modify**:
  - `@/Users/.../migrations/007_router_logo_path.up.sql` — `ALTER TABLE routers ADD COLUMN logo_path VARCHAR(255)`.
  - `@/Users/.../internal/models/router.go` — tambah `LogoPath`.
  - `@/Users/.../internal/services/router_service.go` — tambah `UploadLogo(ctx, routerID, file io.Reader, filename string) error` (validasi mime, max 1 MB, simpan ke `./uploads/logos/{routerID}{ext}`).
  - `@/Users/.../internal/api/handlers/router_handler.go` — `UploadLogo`, `GetLogo`.
  - `@/Users/.../internal/api/router.go`:
    - `POST /routers/:id/logo` (multipart `file`)
    - `GET /routers/:id/logo`
  - Inject `LogoPath` ke `RenderParams` saat task 6 render.
- **DB**: 1 kolom baru.
- **MikroTik**: tidak.
- **Redis**: invalidate dashboard cache.
- **Complexity**: Medium.

### Task 8 — System Scheduler CRUD Generic
**Scope**: Add/Set/Remove + Enable/Disable scheduler bebas (di luar Mikhmon-Expire-Monitor).
- **Files create/modify**:
  - `@/Users/.../internal/roskit/adapter/service/system.go` — tambah `AddScheduler`, `SetScheduler`, `RemoveScheduler` (standar CRUD).
  - `@/Users/.../internal/services/system_service.go` — wrap.
  - `@/Users/.../internal/api/handlers/system_handler.go` + router:
    - `POST /routers/:id/system/schedulers`
    - `PUT /routers/:id/system/schedulers/:schId`
    - `DELETE /routers/:id/system/schedulers/:schId`
    - `POST /routers/:id/system/schedulers/:schId/enable|disable`
- **DB**: tidak.
- **MikroTik**: `system/scheduler/add|set|remove`.
- **Redis**: tidak.
- **Complexity**: Low.

### Task 9 — Walled Garden + Ping Test (Low Priority)
**Scope**:
- `GET /routers/:id/hotspot/walled-garden` (Bridge `ListWalledGarden` sudah ada).
- `POST /routers/:id/system/ping` (body `{"target":"8.8.8.8","count":4}`) — tambah `Bridge.Ping(ctx, routerID, target, count)` memanggil `/ping`.
- **Complexity**: Low.

### Task 10 — User Log Report (Low Priority)
**Scope**: Endpoint `GET /routers/:id/reports/userlog?from=...&to=...&user=...` parsing `/log/print` topik `hotspot,info` mencari pesan `logged in`/`logged out`.
- **Files**: service baru `userlog_service.go`, handler.
- **MikroTik**: `Bridge.GetSystemLog`.
- **Redis**: cache hasil parse 30 detik.
- **Complexity**: Medium (parsing log format RouterOS yang verbose).

### Task 11 — Voucher Preview Endpoint (Optional / Low)
**Scope**: `GET /routers/:id/vouchers/preview?template=default&qty=1` — render contoh dummy untuk preview di editor template.
- **Complexity**: Low.

---

## 5. Assumptions & Open Questions

1. **Frontend (`@/Users/.../website/`)**: Saat ini hanya placeholder `index.html` + `app.html`. Apakah rebuild UI (replikasi mikhmon dashboard, manajemen user, voucher print, dll.) termasuk dalam scope plan ini, atau dikerjakan sebagai track terpisah setelah backend lengkap?
2. **QR Code strategy**: Untuk voucher print, lebih disukai (a) library JS lokal (`qrious.min.js` di `website/js/`, paritas mikhmon) atau (b) generate PNG di server pakai `github.com/skip2/go-qrcode` lalu embed sebagai `<img src="data:image/png;base64,...">`? Opsi (b) menghilangkan dependensi JS dan client-side processing.
3. **Logo storage**: Filesystem lokal (`./uploads/logos/`) atau S3-compatible (MinIO/R2)? Project saat ini tidak punya storage abstraction; jika lokal saja, bagaimana strategi backup/persistence di Docker? Volume mount `./uploads/`?
4. **Print template format**: Tetap pakai DB schema saat ini (`header|row|footer` per `type`), atau ubah ke 1 row per template dengan content tunggal yang berisi placeholder `{{range .Vouchers}}...{{end}}` (lebih sederhana, lebih dekat ke mikhmon yang punya 1 file PHP per template)? Keputusan ini mempengaruhi Task 5 & 6.
5. **DB-side sales delete**: Saat hapus DB sales records, apakah juga hapus `/system/script` sale records di RouterOS? Mikhmon hapus di RouterOS karena RouterOS adalah penyimpanan utama. Di roskit DB adalah utama. Rekomendasi: DELETE hanya menghapus DB, dan RouterOS-side cleanup opsional via flag `?delete_router_records=true`.
6. **Multi-language & theme**: Defer ke frontend track? (tidak masuk plan ini)
7. **Audit log integration**: Apakah aksi seperti "Generate 100 vouchers" / "Delete sales by month" wajib auto-write ke `audit_logs`? Saat ini repo `audit_logs` belum dipakai konsisten.
8. **Quick User single-shot** (`mikhmon/hotspot/quickuser.php`): generate 1 user + langsung print. Apakah cukup composite frontend-side (call `/vouchers/generate` qty=1 → `/vouchers/print`), atau butuh endpoint backend dedicated?
9. **Bluetooth print** (`printbt.php`): Format payload Android Quick Printer app (escape command thermal). Skip atau Low priority?
10. **PHP legacy encryption migration**: Apakah ada router lama yang masih perlu diimport, atau migration `config.php` sudah selesai? (`pkg/encrypt` sudah punya `DecodeBlah` dan `DecryptLegacyPHPConfig`.)

---

> **Tindakan selanjutnya**: review dan beri keputusan untuk Open Questions di atas (terutama #1, #2, #4). Setelah disetujui, implementasi dilakukan task demi task sesuai urutan, masing-masing dengan PR/commit terpisah dan verifikasi `go build ./...` + smoke test.
