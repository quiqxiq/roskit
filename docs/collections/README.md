# Roskit API — Hoppscotch Collections

Kumpulan collection Hoppscotch untuk seluruh endpoint Roskit API, siap dijalankan via CLI maupun UI.

## Prasyarat

| Requirement | Perintah |
|---|---|
| Stack berjalan | `make docker-up` |
| Schema database | `make migrate-up-docker` ← **wajib sebelum seed pertama kali** |
| Data seed | `make seed-docker` (buat admin/router default + restart API) |
| Hoppscotch CLI | `npm install -g @hoppscotch/cli` |

> **Urutan setup pertama kali:** `docker-up` → `migrate-up-docker` → `seed-docker`

## Struktur File

```
docs/collections/
├── env.json              # Environment variables (edit sesuai kebutuhan)
├── health.json           # Health check
├── auth.json             # Login, refresh, logout, password
├── settings.json         # Settings & logo
├── users.json            # CRUD user
├── hotspot.json          # Hotspot users, profiles, active, bindings, walled garden
├── ppp.json              # PPP secrets, active, profiles
├── vouchers.json         # Generate, cache, print, sales
├── templates.json        # Print templates
├── system.json           # Resource, log, clock, scheduler, script, reboot
├── sse.json              # Server-Sent Events streams (tidak cocok untuk CLI)
├── events.json           # Webhook event (on-login)
├── reports.json          # Daily, monthly, summary, export
├── quick-print.json      # Quick print packages
├── profile-mappings.json # Hotspot profile mappings
├── status.json           # User/voucher status lookup
├── routers.json          # CRUD router + connection test (jalankan TERAKHIR)
└── TEST_RESULTS.md       # Laporan hasil testing lengkap
```

## Environment (`env.json`)

File flat key-value untuk Hoppscotch CLI v0.31+. Edit sebelum menjalankan test.

| Key | Contoh | Keterangan |
|---|---|---|
| `base_url` | `http://localhost:8080/api/v1` | Base URL API |
| `username` | `admin` | Username login (dari seed) |
| `password` | `adminpass` | Password login (dari seed) |
| `access_token` | *(isi setelah login)* | JWT Bearer token — expire 15 menit |
| `refresh_token` | *(isi setelah login)* | Refresh token |
| `routerId` | `4` | ID router di database — cek via `GET /routers` |
| `routerName` | `router-2` | Nama router |
| `templateId` | `1` | ID template di database |
| `schedulerId` | `*0` | ID scheduler di RouterOS (format `*N`) |
| `scriptId` | `*1` | ID script di RouterOS (format `*N`) |
| `wid` | *(isi manual)* | Hotspot active session wid |
| `iface` | `ether1` | Nama interface jaringan |
| `profileName` | `1d-1mbps` | Nama hotspot profile |
| `id` | `1` | Generic resource ID |
| `userId` | `4` | ID user di database |
| `name` | *(isi manual)* | Generic resource name |
| `mac` | `00:11:22:33:44:55` | MAC address untuk walled garden |
| `reportDate` | `2026-05-13` | Tanggal laporan (YYYY-MM-DD) |
| `reportYear` | `2026` | Tahun laporan |
| `reportMonth` | `5` | Bulan laporan (1-12) |
| `reportFrom` | `2026-05-01` | Rentang awal laporan |
| `reportTo` | `2026-05-13` | Rentang akhir laporan |

> Seed credentials default: `admin / adminpass` dan `staff / staffpass`

### Cara update token dan routerId

```bash
# Login dan ambil token
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"adminpass"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['access_token'])")

# Cek router yang tersedia dan ID-nya
curl -s http://localhost:8080/api/v1/routers \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool

# Update env.json: isi access_token dan routerId sesuai hasil di atas
```

## Cara Menjalankan

### Satu collection

```bash
hopp test docs/collections/auth.json --env docs/collections/env.json
```

### Semua collection (urutan yang benar)

```bash
ENV="docs/collections/env.json"

for name in health auth settings users templates events status \
            hotspot ppp vouchers system quick-print reports profile-mappings \
            routers; do
  echo "=== $name ==="
  hopp test "docs/collections/${name}.json" --env "$ENV"
done
```

### Dengan delay antar request (ms)

```bash
hopp test docs/collections/system.json --env docs/collections/env.json --delay 500
```

### Generate laporan JUnit (CI)

```bash
hopp test docs/collections/auth.json \
  --env docs/collections/env.json \
  --reporter-junit reports/junit-auth.xml
```

## Urutan Eksekusi yang Direkomendasikan

```
1.  health           → tidak perlu auth
2.  auth             → login → dapatkan access_token
3.  settings         → konfigurasi global
4.  users            → manajemen user
5.  templates        → print templates (independen)
6.  events           → webhook (independen)
7.  status           → status lookup (independen)
8.  hotspot          → butuh router aktif (routerId)
9.  ppp              → butuh router aktif
10. vouchers         → butuh router aktif
11. system           → butuh router aktif
12. quick-print      → butuh router aktif
13. reports          → butuh router aktif
14. profile-mappings → butuh router aktif
15. routers          → TERAKHIR — deleteRouter menghapus router dari pool
16. sse              → SKIP di CLI (long-lived SSE connection)
```

> **Mengapa `routers` harus terakhir?** Collection ini mengandung `deleteRouter` yang menghapus router dari database. Jika dijalankan lebih awal, semua collection router-dependent (hotspot, ppp, quick-print, dll) akan mengembalikan 404.

## Perilaku yang Diharapkan per Collection

| Collection | Yang Bekerja | Yang Mengembalikan Non-200 (Normal) |
|---|---|---|
| **health** | semua | — |
| **auth** | login, refresh, logout | 401 setelah logout (expected) |
| **settings** | getSettings | 400 updateSettings (body placeholder), 404 getLogo (belum upload) |
| **users** | list, get, update, delete | 409 createUser (duplikat dari seed) |
| **templates** | list, create, seedDefaults | 404 get/delete jika ID tidak ada |
| **events** | semua | — |
| **status** | semua | — |
| **hotspot** | semua list/count, add walled garden | 400/404 untuk item yang tidak ada di RouterOS |
| **ppp** | semua list, update, delete | 400 addPPPSecret (body placeholder) |
| **vouchers** | recordSale, importSales | 400 generate (body placeholder), 404 cached batch |
| **system** | semua GET/read | 500 scheduler/script write, reboot, shutdown (CHR limitation) |
| **quick-print** | semua | — (jika router ada) |
| **reports** | semua | — (jika router ada) |
| **profile-mappings** | semua | — (jika router ada) |
| **routers** | list, get | 400 create/update (koneksi timeout), delete hapus router |
| **sse** | — | TIMEOUT (expected — long-lived connection) |

## Catatan Penting

### Router Pool

Router pool bersifat **dinamis**. Router yang dibuat via API langsung masuk pool tanpa restart server. Restart API hanya diperlukan setelah `make seed-docker` karena seed menulis langsung ke database (bypass API engine).

### RouterOS CHR

MikroTik CHR (Cloud Hosted Router) trial memiliki keterbatasan:
- Reboot dan shutdown via API mengembalikan 500 — ini perilaku CHR, bukan bug
- Write operations pada scheduler/script mungkin terbatas tergantung lisensi CHR
- Fitur-fitur ini akan bekerja normal di RouterOS fisik dengan lisensi penuh

### Scheduler dan Script ID

RouterOS menggunakan ID format `*N` (contoh: `*0`, `*1`). Cek ID aktual sebelum menggunakan endpoint update/delete/enable/disable:

```bash
# Cek scheduler IDs
curl -s http://localhost:8080/api/v1/routers/$ROUTER_ID/system/schedulers \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool | grep '"id"'

# Cek script IDs
curl -s http://localhost:8080/api/v1/routers/$ROUTER_ID/system/scripts \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool | grep '"id"'
```

### SSE Streams

Endpoint `/sse/*` dan `/logs/stream/*` adalah long-lived connections. Hopp CLI akan hang. Gunakan:
```bash
curl -N -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/routers/$ROUTER_ID/sse/resources
```

### Reset Data Setelah Test

```bash
# Seed ulang dan restart API (pool diperbarui otomatis)
make seed-docker

# Jalankan ulang migrasi jika ada perubahan model
make migrate-up-docker
```

### Request dengan Body Placeholder

Request dengan body `"string"` atau nilai `0` pada collection adalah contoh schema response, bukan test case yang valid. Response `400 Bad Request` pada request tersebut adalah perilaku yang benar.
