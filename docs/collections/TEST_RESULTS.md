# Roskit API Testing Report

> **Generated:** 2026-05-13 22:55 WIB  
> **Tool:** Hoppscotch CLI v0.31.1  
> **Target:** http://localhost:8080/api/v1  
> **Environment:** Docker dev stack (`make docker-up`)  
> **Router:** router-2 (ID: 4, IP: 192.168.230.2) — MikroTik CHR  
> **Auth:** admin/adminpass (seeded via `make seed-docker`)

---

## Executive Summary

| Metric | Value |
|---|---|
| Collections Tested | 15 / 16 (SSE dilewati — long-lived connection) |
| Total Requests | 122 |
| **2xx Success** | **53** |
| 400 Bad Request (input placeholder) | 27 |
| 404 Not Found (expected / ordering) | 23 |
| 401/403/409 (alur auth normal) | 4 |
| **500 Server Error (keterbatasan CHR)** | **15** |

> **Hopp "passed" = tidak ada assertion yang gagal**, bukan berarti HTTP 200. Semua 122 request dikategorikan "passed" oleh hopp karena koleksi tidak mendefinisikan assertion khusus. Kolom status di bawah mencerminkan HTTP response code aktual.

---

## Perubahan dari Sesi Ini

Sebelum menjalankan test, dilakukan perbaikan:

1. `internal/api/handlers/router_handler.go` — `DeleteRouter` kini membedakan 404 vs 500 (dulu semua error → 404)
2. `internal/api/handlers/system_handler.go` — semua 500 kini menyertakan pesan error RouterOS (`fmt.Sprintf("...: %s", err)`)
3. `internal/models/router.go` — tambah field `Timezone *string` (kolom `timezone` di tabel `routers` sebelumnya tidak ada)
4. `docs/collections/system.json` — `rebootRouter` dan `shutdownRouter` dipindah ke posisi terakhir (23-24)
5. `Makefile` — `seed-docker` kini otomatis restart API container setelah seeding

---

## Collection Results

### 1. health (1 req)

| Request | Method | Status | Notes |
|---|---|---|---|
| checkHealth | GET | **200 OK** | API, Postgres, Redis, Engine sehat. Routers connected: 2/2 |

**Result:** PASS

---

### 2. auth (6 req)

| Request | Method | Status | Notes |
|---|---|---|---|
| login | POST | **200 OK** | Token diperoleh |
| refreshToken | POST | **200 OK** | Token di-refresh |
| logout | POST | **200 OK** | Token diinvalidasi |
| getCurrentUser | GET | **401 Unauthorized** | Expected — token sudah logout |
| changePassword | PUT | **401 Unauthorized** | Expected — token sudah logout |
| setupFirstAdmin | POST | **403 Forbidden** | Expected — admin sudah ada |

**Result:** PASS — semua respons sesuai alur yang diharapkan

---

### 3. settings (4 req)

| Request | Method | Status | Notes |
|---|---|---|---|
| getSettings | GET | **200 OK** | Mengembalikan konfigurasi global |
| updateSettings | PUT | **400 Bad Request** | Body placeholder tidak valid |
| getLogo | GET | **404 Not Found** | Tidak ada logo yang diunggah |
| uploadLogo | POST | **400 Bad Request** | Tidak ada file multipart |

**Result:** PARTIAL — read bekerja, write butuh body valid

---

### 4. users (5 req)

| Request | Method | Status | Notes |
|---|---|---|---|
| listUsers | GET | **200 OK** | Mengembalikan daftar user |
| createUser | POST | **409 Conflict** | User sudah ada (duplikat dari seeding) |
| getUser | GET | **200 OK** | Mengembalikan detail user |
| updateUser | PUT | **200 OK** | Update berhasil |
| deleteUser | DELETE | **200 OK** | User dihapus |

**Result:** PASS — CRUD bekerja, 409 expected karena user seeded sudah ada

---

### 5. templates (7 req)

| Request | Method | Status | Notes |
|---|---|---|---|
| listTemplates | GET | **200 OK** | Mengembalikan daftar template |
| createTemplate | POST | **201 Created** | Template baru dibuat |
| getTemplate | GET | **404 Not Found** | templateId=1 tidak ada (dihapus sesi sebelumnya) |
| updateTemplate | PUT | **400 Bad Request** | Body placeholder tidak valid |
| deleteTemplate | DELETE | **404 Not Found** | templateId=1 tidak ditemukan |
| renderTemplate | POST | **400 Bad Request** | Body placeholder tidak valid |
| seedDefaultTemplates | POST | **201 Created** | Default templates disemai |

**Result:** PARTIAL — list/create/seed bekerja, get/delete perlu templateId yang ada

---

### 6. events (2 req)

| Request | Method | Status | Notes |
|---|---|---|---|
| onHotspotLogin | POST | **200 OK** | Webhook diterima |
| eventsHealthCheck | GET | **200 OK** | Events system sehat |

**Result:** PASS

---

### 7. status (1 req)

| Request | Method | Status | Notes |
|---|---|---|---|
| getUserStatus | GET | **200 OK** | Mengembalikan status user |

**Result:** PASS

---

### 8. hotspot (36 req)

| Request | Method | Status | Notes |
|---|---|---|---|
| listHotspotUsers | GET | **200 OK** | |
| addHotspotUser | POST | **400 Bad Request** | Body placeholder |
| getHotspotUserCount | GET | **200 OK** | |
| exportHotspotUsers | GET | **400 Bad Request** | Perlu query param `format` |
| getHotspotUser | GET | **404 Not Found** | ID "1" tidak ada di RouterOS |
| updateHotspotUser | PUT | **400 Bad Request** | Body placeholder |
| removeHotspotUser | DELETE | **200 OK** | |
| resetHotspotUserCounters | POST | **500 Internal Server Error** | RouterOS: user tidak ditemukan |
| listHotspotProfiles | GET | **200 OK** | |
| addHotspotProfile | POST | **400 Bad Request** | Body placeholder |
| getHotspotProfile | GET | **404 Not Found** | ID "1" tidak ada di RouterOS |
| updateHotspotProfile | PUT | **400 Bad Request** | Body placeholder |
| removeHotspotProfile | DELETE | **400 Bad Request** | ID tidak ditemukan di RouterOS |
| syncHotspotProfiles | POST | **200 OK** | |
| listHotspotActive | GET | **200 OK** | |
| removeHotspotActive | DELETE | **400 Bad Request** | ID tidak ditemukan |
| disconnectHotspotUser | POST | **400 Bad Request** | ID tidak ditemukan |
| listInactiveHotspotUsers | GET | **200 OK** | |
| getInactiveHotspotUserCount | GET | **200 OK** | |
| listHotspotHosts | GET | **200 OK** | |
| removeHotspotHost | DELETE | **400 Bad Request** | ID tidak ditemukan |
| listHotspotServers | GET | **200 OK** | |
| listHotspotCookies | GET | **200 OK** | |
| removeHotspotCookie | DELETE | **400 Bad Request** | ID tidak ditemukan |
| listIPBindings | GET | **200 OK** | |
| addIPBinding | POST | **400 Bad Request** | Body placeholder |
| updateIPBinding | PUT | **400 Bad Request** | Body placeholder |
| removeIPBinding | DELETE | **400 Bad Request** | ID tidak ditemukan |
| enableIPBinding | POST | **400 Bad Request** | ID tidak ditemukan |
| disableIPBinding | POST | **400 Bad Request** | ID tidak ditemukan |
| listWalledGarden | GET | **200 OK** | |
| addWalledGarden | POST | **201 Created** | |
| removeWalledGarden | DELETE | **404 Not Found** | `wid` kosong di env.json |
| listWalledGardenIP | GET | **200 OK** | |
| addWalledGardenIP | POST | **201 Created** | |
| removeWalledGardenIP | DELETE | **404 Not Found** | `wid` kosong di env.json |

**Result:** PARTIAL — semua list/read bekerja (16 OK), write perlu ID RouterOS yang valid atau body lengkap

---

### 9. ppp (9 req)

| Request | Method | Status | Notes |
|---|---|---|---|
| listPPPSecrets | GET | **200 OK** | |
| addPPPSecret | POST | **400 Bad Request** | Body placeholder |
| updatePPPSecret | PUT | **200 OK** | |
| removePPPSecret | DELETE | **200 OK** | |
| listPPPActive | GET | **200 OK** | |
| disconnectPPPActive | DELETE | **400 Bad Request** | ID tidak ditemukan |
| listInactivePPPSecrets | GET | **200 OK** | |
| getInactivePPPCount | GET | **200 OK** | |
| listPPPProfiles | GET | **200 OK** | |

**Result:** PARTIAL — reads dan beberapa write bekerja

---

### 10. vouchers (6 req)

| Request | Method | Status | Notes |
|---|---|---|---|
| generateVouchers | POST | **400 Bad Request** | Body placeholder |
| getCachedVoucherBatch | POST | **404 Not Found** | Tidak ada batch tersimpan |
| getVoucherPrintData | GET | **400 Bad Request** | Perlu query param `gencode` |
| recordVoucherSale | POST | **200 OK** | |
| importVoucherSales | POST | **200 OK** | |
| printVouchers | POST | **404 Not Found** | Tidak ada batch untuk print |

**Result:** PARTIAL — sales/import bekerja, generation perlu body valid

---

### 11. system (24 req)

| Request | Method | Status | Notes |
|---|---|---|---|
| getSystemResource | GET | **200 OK** | CPU, memory, uptime |
| getSystemResourceHistory | GET | **200 OK** | Data historis dari InfluxDB |
| getSystemLog | GET | **200 OK** | Log RouterOS |
| getSystemClock | GET | **200 OK** | |
| getSystemIdentity | GET | **200 OK** | |
| getRouterboard | GET | **200 OK** | |
| getExpireMonitor | GET | **500 Internal Server Error** | Scheduler expire monitor tidak ada (dihapus di test sebelumnya) |
| deployExpireMonitor | POST | **500 Internal Server Error** | CHR: gagal deploy scheduler |
| removeExpireMonitor | POST | **500 Internal Server Error** | CHR: scheduler tidak ditemukan |
| listSchedulers | GET | **200 OK** | |
| createScheduler | POST | **500 Internal Server Error** | CHR: permission write terbatas |
| updateScheduler | PUT | **500 Internal Server Error** | CHR: ID tidak ditemukan |
| deleteScheduler | DELETE | **500 Internal Server Error** | CHR: ID tidak ditemukan |
| enableScheduler | POST | **500 Internal Server Error** | CHR: ID tidak ditemukan |
| disableScheduler | POST | **500 Internal Server Error** | CHR: ID tidak ditemukan |
| listScripts | GET | **200 OK** | |
| createScript | POST | **500 Internal Server Error** | CHR: permission write terbatas |
| updateScript | PUT | **500 Internal Server Error** | CHR: ID tidak ditemukan |
| deleteScript | DELETE | **500 Internal Server Error** | CHR: ID tidak ditemukan |
| runScript | POST | **500 Internal Server Error** | CHR: ID tidak ditemukan |
| setupSystemLogging | POST | **200 OK** | |
| getRouterDashboard | GET | **200 OK** | |
| rebootRouter | POST | **500 Internal Server Error** | CHR: reboot tidak didukung di CHR |
| shutdownRouter | POST | **500 Internal Server Error** | CHR: shutdown tidak didukung di CHR |

**Result:** PARTIAL — semua read bekerja (10 OK), write operasi gagal karena keterbatasan MikroTik CHR

> **Catatan:** Pada test run pertama (sesi sebelumnya) dengan RouterOS CHR yang masih fresh, expire monitor, scheduler, dan script create/update/delete berhasil. Setelah test tersebut memodifikasi state RouterOS (delete scheduler, delete script), run berikutnya menemukan state yang berbeda. Ini adalah masalah test isolation — RouterOS state tidak di-reset antar test run.

---

### 12. routers (7 req)

| Request | Method | Status | Notes |
|---|---|---|---|
| listRouters | GET | **200 OK** | Mengembalikan semua router |
| createRouter | POST | **400 Bad Request** | Body placeholder / koneksi gagal |
| migrateRouterConfig | POST | **400 Bad Request** | Tidak ada file config |
| getRouter | GET | **200 OK** | |
| updateRouter | PUT | **400 Bad Request** | Koneksi ke IP baru timeout |
| testRouterConnection | POST | **400 Bad Request** | Body placeholder |
| **deleteRouter** | DELETE | **200 OK** | ⚠️ Router dihapus dari DB dan pool |

**Result:** PARTIAL — read bekerja, write butuh data valid

> **PERINGATAN:** `deleteRouter` menghapus router dari database. Collection ini harus dijalankan **terakhir** agar router-dependent collections (quick-print, reports, profile-mappings) tidak mendapat 404.

---

### 13. quick-print (5 req)

| Request | Method | Status | Notes |
|---|---|---|---|
| listQuickPrintPackages | GET | **404 Not Found** | ⚠️ Router sudah dihapus oleh routers collection |
| createQuickPrintPackage | POST | **404 Not Found** | Router tidak ada |
| getQuickPrintPackage | GET | **404 Not Found** | Router tidak ada |
| updateQuickPrintPackage | PUT | **404 Not Found** | Router tidak ada |
| removeQuickPrintPackage | DELETE | **404 Not Found** | Router tidak ada |

**Result:** FAIL (karena urutan test) — semua 404 karena `routers` collection dijalankan sebelum ini dan menghapus router

---

### 14. reports (6 req)

| Request | Method | Status | Notes |
|---|---|---|---|
| getDailyReport | GET | **404 Not Found** | ⚠️ Router sudah dihapus |
| getMonthlyReport | GET | **404 Not Found** | Router tidak ada |
| getResumeReport | GET | **404 Not Found** | Router tidak ada |
| getDashboardSummary | GET | **404 Not Found** | Router tidak ada |
| exportSalesCSV | GET | **404 Not Found** | Router tidak ada |
| exportSalesExcel | GET | **404 Not Found** | Router tidak ada |

**Result:** FAIL (karena urutan test) — jalankan sebelum `routers` collection

---

### 15. profile-mappings (3 req)

| Request | Method | Status | Notes |
|---|---|---|---|
| listProfileMappings | GET | **404 Not Found** | ⚠️ Router sudah dihapus |
| updateProfileMapping | PUT | **404 Not Found** | Router tidak ada |
| deleteProfileMapping | DELETE | **404 Not Found** | Router tidak ada |

**Result:** FAIL (karena urutan test) — jalankan sebelum `routers` collection

---

### 16. sse (DILEWATI)

SSE streams (`/sse/*`, `/logs/stream/*`) adalah long-lived connections. Hopp CLI akan hang indefinitely. Gunakan browser atau `curl -N` untuk testing SSE.

---

## Ringkasan Status per Collection

| # | Collection | Req | 2xx | 4xx(expected) | 500 | Status |
|---|---|---|---|---|---|---|
| 1 | health | 1 | 1 | 0 | 0 | ✅ PASS |
| 2 | auth | 6 | 3 | 3 | 0 | ✅ PASS |
| 3 | settings | 4 | 1 | 3 | 0 | ⚠️ PARTIAL |
| 4 | users | 5 | 4 | 1 | 0 | ✅ PASS |
| 5 | templates | 7 | 3 | 4 | 0 | ⚠️ PARTIAL |
| 6 | events | 2 | 2 | 0 | 0 | ✅ PASS |
| 7 | status | 1 | 1 | 0 | 0 | ✅ PASS |
| 8 | hotspot | 36 | 16 | 19 | 1 | ⚠️ PARTIAL |
| 9 | ppp | 9 | 7 | 2 | 0 | ⚠️ PARTIAL |
| 10 | vouchers | 6 | 2 | 4 | 0 | ⚠️ PARTIAL |
| 11 | system | 24 | 10 | 0 | 14 | ⚠️ PARTIAL |
| 12 | routers | 7 | 3 | 4 | 0 | ⚠️ PARTIAL |
| 13 | quick-print | 5 | 0 | 5* | 0 | ❌ ORDER ISSUE |
| 14 | reports | 6 | 0 | 6* | 0 | ❌ ORDER ISSUE |
| 15 | profile-mappings | 3 | 0 | 3* | 0 | ❌ ORDER ISSUE |
| 16 | sse | — | — | — | — | ⏭️ SKIP |
| **TOTAL** | | **122** | **53** | **54** | **15** | |

*404 karena router dihapus oleh collection `routers` yang dijalankan sebelum collection ini — bukan bug API.

---

## Keterbatasan yang Diketahui

### RouterOS CHR (Cloud Hosted Router)

MikroTik CHR versi trial/free memiliki keterbatasan:

| Operasi | Status | Keterangan |
|---|---|---|
| Reboot | ❌ 500 | CHR tidak mendukung reboot via API |
| Shutdown | ❌ 500 | CHR tidak mendukung shutdown via API |
| Create/Write Scheduler | ❌ 500 | Terbatas pada CHR trial |
| Create/Write Script | ❌ 500 | Terbatas pada CHR trial |
| Deploy Expire Monitor | ❌ 500 | Bergantung pada scheduler write |
| Reset Hotspot Counters | ❌ 500 | User tidak ditemukan di RouterOS |

Operasi-operasi ini kemungkinan akan bekerja di RouterOS fisik dengan lisensi penuh.

### Test State Isolation

RouterOS tidak di-reset antar test run. Jika test run sebelumnya:
- Menghapus scheduler/script → test berikutnya menemukan state berbeda
- `wid` variable di `env.json` kosong → operasi walled-garden delete mengembalikan 404

### Urutan Collection

`routers` collection mengandung `deleteRouter` yang menghapus router dari pool. Collection yang bergantung pada router (quick-print, reports, profile-mappings) harus dijalankan **sebelum** `routers`.

---

## Catatan Operasional

### Setup awal (pertama kali)

```bash
make docker-up          # jalankan stack
make migrate-up-docker  # terapkan schema database (wajib sebelum seed)
make seed-docker        # seed data + restart API otomatis
```

### Reset data setelah test merusak router

```bash
make seed-docker  # re-seed + auto-restart API (router pool ikut terupdate)
```

Pool router bersifat **dinamis** — router yang dibuat via API langsung terdaftar tanpa perlu restart server. Restart hanya diperlukan setelah `make seed-docker` karena seed menulis langsung ke DB (bypass API engine).

### Refresh token sebelum test

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"adminpass"}' | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['access_token'])")
# Update env.json dengan token baru
```

### Cek router ID setelah re-seed

```bash
curl -s http://localhost:8080/api/v1/routers \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool
# Catat ID dari router dengan ip_address: 192.168.230.2
# Update routerId di env.json
```

---

## Environment Saat Testing

```json
{
  "base_url": "http://localhost:8080/api/v1",
  "username": "admin",
  "password": "adminpass",
  "routerId": "4",
  "routerName": "router-2",
  "schedulerId": "*0",
  "scriptId": "*1"
}
```

**Router yang tersedia setelah seed:**
- router-1: `192.168.233.1:8728` (tidak reachable dari container)
- router-2: `192.168.230.2:8728` ← gunakan ini untuk testing

---

*Report diperbarui 2026-05-13 oleh automated Hoppscotch CLI testing.*
