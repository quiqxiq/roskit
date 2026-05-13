# Roskit API — Hoppscotch Collections

Kumpulan collection Hoppscotch untuk seluruh endpoint Roskit API, siap dijalankan via CLI maupun UI.

## Prasyarat

| Requirement | Detail |
|---|---|
| Server | `make docker-up` atau `make run` |
| Seed data | `make seed` (buat admin/router default) |
| Hoppscotch CLI | `npm install -g @hoppscotch/cli` |

## Struktur File

```
docs/collections/
├── env.json              # Environment variables (edit sesuai kebutuhan)
├── health.json           # Health check
├── auth.json             # Login, refresh, logout, password
├── routers.json          # CRUD router + connection test
├── settings.json         # Settings & logo
├── users.json            # CRUD user
├── hotspot.json          # Hotspot users, profiles, active, bindings, walled garden
├── ppp.json              # PPP secrets, active, profiles
├── vouchers.json         # Generate, cache, print, sales
├── templates.json        # Print templates
├── system.json           # Resource, log, clock, scheduler, script, reboot
├── sse.json              # Server-Sent Events streams
├── events.json           # Webhook event (on-login)
├── reports.json          # Daily, monthly, summary, export
├── quick-print.json      # Quick print packages
├── profile-mappings.json # Hotspot profile mappings
└── status.json           # User/voucher status lookup
```

## Environment (`env.json`)

File ini menggunakan format flat key-value untuk Hoppscotch CLI v0.31+.

| Key | Default | Keterangan |
|---|---|---|
| `base_url` | `http://localhost:8080/api/v1` | Base URL API |
| `username` | `admin` | Username login (dari seed) |
| `password` | `adminpass` | Password login (dari seed) |
| `access_token` | *(isi setelah login)* | JWT Bearer token |
| `refresh_token` | *(isi setelah login)* | Refresh token |
| `routerId` | `1` | ID router default |
| `templateId` | `1` | ID template |
| `schedulerId` | `1` | ID scheduler RouterOS |
| `scriptId` | `1` | ID script RouterOS |
| `wid` | *(isi manual)* | Hotspot active user wid |
| `iface` | `ether1` | Nama interface |
| `profileName` | `1d-1mbps` | Nama hotspot profile |
| `id` | `1` | Generic resource ID |
| `name` | *(isi manual)* | Generic resource name |

> **Catatan**: `access_token` perlu diisi setelah jalankan `auth/login`.
> Seed credentials default: `admin / adminpass` dan `staff / staffpass`.

## Cara Menjalankan

### Satu collection

```bash
hopp test docs/collections/auth.json --env docs/collections/env.json
```

### Semua collection berurutan

```bash
COLLECTIONS=(health auth routers settings users hotspot ppp vouchers \
  templates system sse events reports quick-print profile-mappings status)
ENV="docs/collections/env.json"

for name in "${COLLECTIONS[@]}"; do
  hopp test "docs/collections/${name}.json" --env "$ENV"
done
```

### Dengan delay antar request (ms)

```bash
hopp test docs/collections/routers.json --env docs/collections/env.json --delay 500
```

### Generate laporan JUnit (CI)

```bash
hopp test docs/collections/auth.json \
  --env docs/collections/env.json \
  --reporter-junit reports/junit-auth.xml
```

## Urutan Eksekusi yang Direkomendasikan

Beberapa collection bergantung pada resource yang dibuat sebelumnya:

```
1. health          → tidak perlu auth
2. auth            → login → dapatkan access_token
3. routers         → CRUD router (routerId dipakai semua collection berikutnya)
4. settings        → konfigurasi global
5. users           → manajemen user
6. hotspot         → butuh router aktif
7. ppp             → butuh router aktif
8. vouchers        → butuh router aktif
9. templates       → independen
10. system         → butuh router aktif
11. sse            → butuh router aktif (streaming, tidak blocking di CLI)
12. events         → independen (webhook)
13. reports        → butuh router + data penjualan
14. quick-print    → butuh router aktif
15. profile-mappings → butuh router aktif
16. status         → independen
```

## Catatan Penting

- **Router-scoped endpoints** (`/routers/{routerId}/...`) akan mengembalikan `404` jika router tidak ada. Pastikan `routerId` di `env.json` sesuai dengan ID router yang ada di database.
- **SSE streams** (`/sse/`, `/logs/stream/`) adalah long-lived connections. Hopp CLI akan timeout setelah beberapa detik — ini normal.
- **Collection `routers.json`** menyertakan request `deleteRouter`. Jalankan collection ini hanya jika ingin cleanup, atau skip request tersebut saat testing normal.
- **Request dengan body placeholder** (misal `"string"`, `0`) memang akan mengembalikan `400 Bad Request` — ini merupakan contoh response schema, bukan test case yang diharapkan berhasil.
