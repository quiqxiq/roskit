# Roskit Audit Report — Implementasi vs `MIKHMON_INTEGRATION_PLAN_OPENCODE.md`

**Tanggal:** 2026-04-29
**Scope:** Verifikasi klaim "Done/Partial/Missing" di plan, cari bug nyata, identifikasi gap baru.
**Metode:** Pemeriksaan langsung file source `event_handler.go`, `router.go`, `voucher_service.go`, `template_service.go`, `template_defaults.go`, `template_repo.go`, `cmd/api/main.go` + sampling cross-package.

---

## TL;DR

**Plan sudah outdated.** Mayoritas item yang plan klaim "❌ Missing" atau "⚠️ Partial" ternyata **sudah terimplementasi**. Namun audit menemukan **bug baru yang lebih kritis** yang plan tidak sebut, terutama di **default print templates**.

| Status di Plan | Realita Audit |
|---|---|
| 12 task missing/partial | Hanya **2** yang masih benar-benar bermasalah (Task 9 partial, beberapa LOW) |
| Task 1 CRITICAL: profile price stub | ✅ Sudah lengkap dengan DB lookup + RouterOS fallback (`event_handler.go:194-220`) |
| Task 2 HIGH: default templates missing | ⚠️ Sudah seed, **tapi seluruh template content RUSAK** (lihat Bug #1) |
| Task 3 HIGH: walled garden CRUD | ✅ Sudah ada handler & route (`router.go:141-146`) |
| Task 4 MED: reset counters endpoint | ✅ Sudah ada (`router.go:114`) |
| Task 5 MED: scheduler CRUD | ✅ Sudah lengkap (`router.go:182-187`) |
| Task 6 MED: full voucher print page | ✅ Endpoint `POST /vouchers/print` ada (`router.go:156`) |
| Task 7 MED: logo upload | ✅ Sudah ada + migrasi 006 (`router.go:100-101`) |
| Task 9 LOW: offline QR | ✅ Sudah pakai `github.com/skip2/go-qrcode` lokal (`template_service.go:230-236`) |
| Task 10 LOW: script management | ✅ Sudah lengkap (`router.go:188-192`) |
| Task 11 LOW: setup logging | ✅ Sudah dipanggil otomatis saat router connect (`main.go:122-126`) + endpoint manual |
| Task 12 LOW: user status page | ✅ Sudah ada `/api/v1/status` public (`router.go:261`) |

---

## Bug Nyata yang Ditemukan

### 🔴 CRITICAL — Bug #1: Default templates pakai placeholder PHP, renderer pakai Go template

**Lokasi:** `internal/services/template_defaults.go` seluruh konten template (baris 10–238) vs `internal/services/template_service.go:238-248`.

**Symptom:**
Template default berisi placeholder gaya Mikhmon PHP:
```html
%hotspotName%, %username%, %password%, %qrCode%, %validity%,
%limitUptime%, %limitBytesTotal%, %price%, %dnsName%, %#%, %timeStamp%, %logo%
```

Tetapi renderer di `template_service.go:239-247` pakai Go `html/template`:
```go
tmpl, err := template.New("").Parse(content)
err := tmpl.Execute(&buf, vars)  // vars: VoucherTemplateVars{Username, Password, ...}
```

Variable yang tersedia: `{{.Username}}`, `{{.Password}}`, `{{.HotspotName}}`, `{{.QRCode}}`, dll.

**Akibat:** Saat user mencetak voucher, output HTML akan menampilkan literal `%username%`, `%password%`, dll alih-alih nilai sebenarnya. Cetakan rusak total.

**Root cause:** Konten template di-port langsung dari Mikhmon tanpa konversi placeholder syntax.

**Suggested fix (pilih salah satu):**
- **(A) Konversi konten template** — replace `%username%` → `{{.Username}}`, dll. Untuk `%#%` (nomor) → `{{.Num}}`. Pertahankan struktur HTML/CSS/JS.
- **(B) Ganti renderer** — implementasi simple `strings.Replacer` yang substitusi `%key%` → value, lebih dekat ke perilaku Mikhmon. Lebih mudah, tapi kehilangan keuntungan auto-escape Go template (XSS protection).

Rekomendasi: **(A)** karena sudah pakai `html/template` yang escape XSS untuk konteks HTML.

---

### 🟠 HIGH — Bug #2: Idempotency key on-login event ≠ key manual record sale

**Lokasi:**
- `internal/api/handlers/event_handler.go:88-90` (on-login webhook)
  ```go
  idKey := fmt.Sprintf("%d|%s|%s|%s", router.ID, payload.Username, payload.Date, payload.Time)
  ```
- `internal/services/voucher_service.go:264-267` (manual recordSale)
  ```go
  input := fmt.Sprintf("%d|%s|%s", routerID, username, soldAt.Format("2006-01-02 15:04:05"))
  ```

**Symptom:** Sale yang sama bisa dicatat dua kali — sekali dari webhook on-login, sekali dari manual `POST /vouchers/sales` atau import. Kedua jalur menghasilkan **hash idempotency yang berbeda** untuk sale logis yang sama, sehingga `ExistsByIdempotencyKey` tidak mendeteksinya sebagai duplikat.

**Root cause:** Format kunci tidak diseragamkan. Webhook pakai `date|time` mentah (mis. `apr/29/2026|14:30:05`), service pakai `2006-01-02 15:04:05` formatted.

**Suggested fix:** Ekstrak helper `MakeSaleIdempotencyKey(routerID, username, soldAt time.Time) string` di `internal/services/voucher_service.go`, dipakai oleh **kedua jalur**. Webhook harus parse `payload.Date+payload.Time` ke `time.Time` lebih dulu lalu format sama.

---

### 🟠 HIGH — Bug #3: `parseMikroTikDateTime` pakai `time.Local`, bukan timezone router

**Lokasi:**
- `internal/api/handlers/event_handler.go:185`
- `internal/services/voucher_service.go:302`

```go
return time.Date(year, mon, day, hour, minute, second, 0, time.Local), nil
```

**Symptom:** Kalau server roskit & router MikroTik di timezone berbeda (misal server UTC, router Asia/Jakarta), `SoldAt` di DB akan offset sesuai timezone server, bukan timezone aktual transaksi. Report harian/bulanan jadi salah saat ada user login dekat tengah malam.

**Root cause:** Tidak ada lookup timezone router. Mikhmon RouterOS biasanya pakai `/system/clock/print` → field `time-zone-name`.

**Suggested fix:** Tambah field `Timezone string` di model `Router` (atau lookup on-demand & cache), parse pakai `time.LoadLocation(router.Timezone)`. Fallback ke `time.UTC` (bukan `time.Local`) jika kosong.

---

### 🟡 MEDIUM — Bug #4: `parseMikroTikDateTime` duplikasi di 2 file

**Lokasi:** `event_handler.go:157-186` dan `voucher_service.go:274-303` — fungsi identik.

**Symptom:** DRY violation. Plan #3.1 sudah catat ini sebagai LOW cleanup tapi tidak prioritas. Tapi ini bahaya kalau salah satu di-update tapi yang lain tidak — bug timezone (Bug #3) di atas akan butuh dua edit, gampang miss salah satu.

**Suggested fix:** Pindah ke package shared, contoh `pkg/mikrotik/timeparse.go`, expose `Parse(date, time, loc *time.Location) (time.Time, error)`. Hapus duplikat lokal.

---

### 🟡 MEDIUM — Bug #5: `templateSvc.SeedDefaults()` error di-swallow saat startup

**Lokasi:** `internal/api/router.go:49`
```go
_ = templateSvc.SeedDefaults(context.Background())
```

**Symptom:** Jika seed gagal (DB error, schema mismatch, dll.), startup tetap lanjut tanpa log. Operator tidak tahu template default tidak terinstall.

**Root cause:** Sengaja silent fail.

**Suggested fix:**
```go
if err := templateSvc.SeedDefaults(context.Background()); err != nil {
    slog.Default().Warn("template seed defaults failed", "error", err)
}
```

Lebih baik lagi: seed dipindah ke `cmd/api/main.go` setelah migration runner, sebelum `engine.Start()`, supaya error fatal kalau memang DB rusak.

---

### 🟡 MEDIUM — Bug #6: `templateSvc.SeedDefaults` dipanggil pada `NewRouter`, bukan startup

**Lokasi:** `internal/api/router.go:49`

**Symptom:** `NewRouter` di-call sekali di `cmd/api/main.go:152`, tapi secara semantik fungsi ini "construct router" — seed defaults adalah side effect tersembunyi yang sulit di-test. Kalau ada test yang construct `NewRouter` (mis. integration test), DB akan dimutasi.

**Suggested fix:** Pindahkan seed ke `main.go` setelah connect DB, sebelum membangun router. Atau buat `cmd/migrate/main.go` jalankan seed sebagai langkah migration terakhir.

---

### 🟡 MEDIUM — Bug #7: Voucher generation cache key bisa di-overwrite

**Lokasi:** `internal/services/voucher_service.go:111`
```go
key := appcache.VoucherSessionKey(routerID, params.Gencode)
```

**Symptom:** Kalau dua admin generate voucher untuk router yang sama dengan `gencode` yang sama (mis. gencode default kosong/sama), batch kedua **menimpa** batch pertama di Redis. Admin pertama akan kehilangan data print.

**Root cause:** Tidak ada uniqueness check pada gencode atau combine dengan timestamp/userID.

**Suggested fix:**
- Wajibkan gencode non-empty di handler (saat ini cek `params.NameLength`/`CharSet`/`UserType` ada default, tapi `Gencode` tidak).
- Generate gencode default dengan UUID jika kosong.
- Atau encode `userID` (admin) ke key.

---

### 🟡 MEDIUM — Bug #8: `GenerateVoucherComment` menerima string kosong di posisi date

**Lokasi:** `internal/services/voucher_service.go:89`
```go
comment := roskitservice.GenerateVoucherComment(params.UserType, params.Gencode, "", params.Comment)
```

**Symptom:** Param ke-3 (kemungkinan `dateTag` format `MM.DD.YY` di Mikhmon) di-pass kosong. Comment yang seharusnya `vc-{code}-04.29.26-{text}` jadi `vc-{code}--{text}` (atau apapun fallback-nya).

**Root cause:** Tampaknya placeholder hardcoded, mungkin diniatkan untuk diisi tapi belum diimplementasi.

**Suggested fix:** Tambah `time.Now().Format("01.02.06")` sebagai param ke-3, atau ubah signature `GenerateVoucherComment` untuk auto-generate date jika empty.

**Verifikasi tambahan dibutuhkan:** Cek implementasi `GenerateVoucherComment` di `internal/roskit/adapter/service/voucher.go` untuk konfirmasi semantik param.

---

### 🟢 LOW — Bug #9: `time.Date(...time.Local)` di `parseMikroTikDateTime` saat error parse silent

**Lokasi:** `event_handler.go:166-183`, `voucher_service.go:283-300`

```go
hour, _ := strconv.Atoi(parts[0])  // err ignored
mon := months[strings.ToLower(dateParts[0])]  // tidak ada !ok check
```

**Symptom:** Kalau bulan invalid (mis. router kirim "xxx/29/2026"), `mon` = 0 → `time.Month(0)` → `time.Date` akan normalisasi jadi Desember tahun sebelumnya. Sale tercatat di tanggal salah, bukan reject.

**Suggested fix:** Validasi:
```go
mon, ok := months[strings.ToLower(dateParts[0])]
if !ok { return time.Time{}, fmt.Errorf("unknown month: %s", dateParts[0]) }
```

Plus kembalikan error jika `strconv.Atoi` gagal di hour/min/sec/day/year.

---

### 🟢 LOW — Bug #10: Validasi part di `UpdateTemplateRequest` boleh kosong

**Lokasi:** `internal/services/template_service.go:78`
```go
Part string `json:"part" binding:"omitempty,oneof=header row footer"`
```

**Symptom:** OK secara binding, tapi di Update logic (line 127-129) hanya update kalau `req.Part != ""`. Jadi user tidak bisa **clear** part field. Bukan bug serius karena part wajib ada, tapi inkonsisten dengan create yang `binding:"required"`.

**Suggested fix:** Tidak perlu fix kalau memang part wajib.

---

## Verifikasi Klaim Plan (Update Status)

| # | Plan Status | Realita 2026-04-29 | Catatan |
|---|---|---|---|
| Task 1 — profile price stub | ❌ CRITICAL stub | ✅ Done | `event_handler.go:194-220` lengkap dengan fallback ke RouterOS |
| Task 2 — default templates | ❌ Missing | ⚠️ Seeded tapi RUSAK | **Lihat Bug #1** |
| Task 3 — walled garden CRUD | ❌ Missing | ✅ Done | `router.go:141-146` |
| Task 4 — reset counters | ⚠️ Partial | ✅ Done | `router.go:114` |
| Task 5 — scheduler CRUD | ❌ Missing | ✅ Done | `router.go:182-187` |
| Task 6 — full print page | ❌ Missing | ✅ Done | `router.go:156` `POST /vouchers/print` |
| Task 7 — logo upload | ❌ Missing | ✅ Done | `router.go:100-101` + migration `006_router_logo.up.sql` |
| Task 8 — enhanced dashboard | ⚠️ Partial | ⚠️ Belum diverifikasi | Cek `system_handler.GetDashboard()` apakah sudah include health/income/log |
| Task 9 — offline QR | ❌ Missing | ✅ Done | `template_service.go:230-236` pakai `skip2/go-qrcode` |
| Task 10 — script CRUD | ❌ Missing | ✅ Done | `router.go:188-192` |
| Task 11 — setup logging | ⚠️ Dead code | ✅ Done | Auto-call saat router connect via `OnRouterConnect` callback (`main.go:122-126`) + endpoint manual |
| Task 12 — user status page | ❌ Missing | ✅ Done | `router.go:261` `GET /api/v1/status` (public) |

**Item yang BENAR-BENAR perlu kerja sekarang (bukan dari plan, tapi dari audit):**
1. **Bug #1** — Fix default template placeholders (CRITICAL, blocker untuk fitur cetak voucher)
2. **Bug #2** — Unified idempotency key (HIGH, data integrity)
3. **Bug #3** — Timezone-aware datetime parsing (HIGH, akurasi report)
4. **Bug #7** — Voucher cache collision (MEDIUM, data loss)
5. **Bug #8** — Voucher comment date kosong (MEDIUM, parsing report import dari RouterOS bisa salah)

---

## Rekomendasi Plan Selanjutnya

### Sprint 1 — Data Integrity & Print Fix (1-2 hari)
**Goal:** Fitur cetak voucher functional, sale data akurat.

| Task | File | Estimated |
|------|------|-----------|
| Fix template placeholders (Bug #1) | `internal/services/template_defaults.go` (semua 9 const) | 2-3 jam |
| Tambah test rendering voucher → assert HTML tidak mengandung `%...%` | `internal/services/template_service_test.go` (baru) | 1 jam |
| Unify idempotency key (Bug #2) | `voucher_service.go` + `event_handler.go` | 1 jam |
| Konsolidasi `parseMikroTikDateTime` (Bug #4) ke `pkg/mikrotik/timeparse.go` | `pkg/mikrotik/timeparse.go` (baru) | 30 menit |
| Add timezone field + lookup (Bug #3) | `models/router.go`, migration `007_router_timezone.up.sql`, parsers | 2 jam |

### Sprint 2 — Robustness Fixes (4-6 jam)
| Task | Estimated |
|------|-----------|
| Unsilence seed errors (Bug #5) + pindah ke main.go (Bug #6) | 30 menit |
| Voucher cache collision guard (Bug #7) | 1 jam |
| Generate voucher date tag (Bug #8) | 30 menit |
| Strict datetime parsing (Bug #9) | 30 menit |
| Verifikasi enhanced dashboard (Task 8 plan) — cek apakah sudah include health/income/log | 1 jam |

### Sprint 3 — Test Coverage (sesuai rules `testing.md` 80%+)
- Test idempotency end-to-end (webhook + manual + import jalur sama tidak duplikat)
- Test render voucher dengan vars lengkap → snapshot HTML
- Test timezone parsing untuk router Asia/Jakarta vs UTC server
- Race detector test: 2 concurrent generate dengan gencode beda

---

## Catatan Audit

- **Saya tidak audit:** roskit core (engine, pool, behaviors, cache pipeline) karena 4 agent paralel yang seharusnya cover ini gagal karena rate limit. Audit infrastructure perlu dijalankan ulang.
- **Saya tidak verifikasi:** Bridge methods (walled garden CRUD, scheduler CRUD, script CRUD) — hanya pastikan route terdaftar. Implementation correctness butuh review terpisah.
- **Saya tidak verifikasi:** Hotspot user/profile cleanup cascade (script + scheduler dihapus saat user di-remove) — claim plan, butuh konfirmasi pemeriksaan `internal/services/hotspot_service.go`.

**File audit ini bisa dijadikan input untuk:**
1. Update `MIKHMON_INTEGRATION_PLAN_OPENCODE.md` — tandai Task 1, 3, 4, 5, 6, 7, 9, 10, 11, 12 sebagai DONE.
2. Buat issue list di GitHub/Linear berdasarkan Bug #1–#10.
3. Re-run audit infrastructure ketika rate limit reset (jam 21:30 GMT+7).
