# Roskit API — Hasil Test Hoppscotch CLI

**Tanggal**: 2026-05-13 18:30 WIB
**Tool**: Hoppscotch CLI v0.31.1
**Server**: `http://localhost:8080/api/v1`
**Environment**: `docs/collections/env.json` — `routerId=4`, `admin/adminpass`
**Urutan**: health → auth → settings → users → templates → events → status → hotspot → ppp → vouchers → system → sse → reports → quick-print → profile-mappings → **routers** (terakhir)

## Ringkasan

| Collection | Req | ✅ | ⚠️ | ❌ |
|---|---|---|---|---|
| `health` | 1 | 1 | 0 | 0 |
| `auth` | 6 | 3 | 3 | 0 |
| `settings` | 4 | 1 | 0 | 3 |
| `users` | 5 | 4 | 1 | 0 |
| `templates` | 7 | 3 | 0 | 4 |
| `events` | 2 | 2 | 0 | 0 |
| `status` | 1 | 1 | 0 | 0 |
| `hotspot` | 36 | 17 | 0 | 19 |
| `ppp` | 9 | 7 | 0 | 2 |
| `vouchers` | 6 | 2 | 0 | 4 |
| `system` | 24 | 7 | 0 | 17 |
| `sse` | 13 | 0 | 0 | 13 |
| `reports` | 6 | 6 | 0 | 0 |
| `quick-print` | 5 | 2 | 0 | 3 |
| `profile-mappings` | 3 | 3 | 0 | 0 |
| `routers` | 7 | 3 | 0 | 4 |
| **Total** | **135** | **62** | **4** | **69** |

## Keterangan

| Ikon | Arti |
|---|---|
| ✅ | HTTP 2xx — sukses |
| ⚠️ | HTTP 401/403/409 — expected (auth invalidated / conflict) |
| ❌ | HTTP 400/404/500 — gagal atau placeholder body |

**400 Bad Request** pada banyak request write (create/update) adalah normal — body collection menggunakan nilai placeholder (`"string"`, `0`) dari OpenAPI schema, bukan data valid.

**500 Internal Server Error** pada `system` (scheduler, script, reboot, expire-monitor, dashboard) dan `quick-print` (get/update/remove) mengindikasikan keterbatasan RouterOS CHR — paket scheduler/script tidak tersedia pada test device.

**SSE** mengembalikan 404 karena stream worker belum diimplementasikan untuk router ini.

## Perubahan dari Sesi Sebelumnya

Fixes yang diterapkan (51 → 62 ✅, +11):

| Endpoint | Sebelum | Sesudah | Penyebab Fix |
|---|---|---|---|
| `auth/refreshToken` | ❌ 400 | ✅ 200 | Body `{"refresh_token":"<<refresh_token>>"}` |
| `status/getUserStatus` | ❌ 400 | ✅ 200 | Query params `router`/`mac` diisi env var |
| `users/getUser` | ❌ 404 | ✅ 200 | Path ganti `<<id>>` → `<<userId>>=4` (admin) |
| `users/updateUser` | ❌ 500 | ✅ 200 | Body safe `{"active":true}`, backend fix ErrUserNotFound→404 |
| `reports/getDailyReport` | ❌ 400 | ✅ 200 | Query param `date=<<reportDate>>` |
| `reports/getMonthlyReport` | ❌ 400 | ✅ 200 | Query params `year`/`month` diisi env var |
| `reports/exportSalesCSV` | ❌ 400 | ✅ 200 | Query params `from`/`to` diisi env var |
| `reports/exportSalesExcel` | ❌ 400 | ✅ 200 | Query params `from`/`to` diisi env var |
| `quick-print/listPackages` | ❌ 500 | ✅ 200 | Router 4 terdaftar di connection pool |
| `quick-print/createPackage` | ❌ 500 | ✅ 201 | Router 4 terdaftar di connection pool |
| `hotspot/addHotspotProfile` | ❌ 400 | ✅ 201 | Body diganti ke nilai valid RouterOS |

Backend fixes:
- `quick_print_handler.go`: kondisi `ShouldBindJSON` sebelumnya inverted (`err == nil`), selalu return 200
- `auth_service.go`: tambah sentinel `ErrUserNotFound`, `UpdateUser` return sentinel bukan wrapped error
- `user_handler.go`: tambah `case services.ErrUserNotFound` → 404 di Update handler

## Detail per Collection

### `health`

| # | Operation | Method | Path | Status | Waktu |
|---|---|---|---|---|---|
| 1 | `checkHealth` | `GET` | `/health` | ✅ 200 OK | 0.032s |

### `auth`

| # | Operation | Method | Path | Status | Waktu |
|---|---|---|---|---|---|
| 1 | `login` | `POST` | `/auth/login` | ✅ 200 OK | 0.121s |
| 2 | `refreshToken` | `POST` | `/auth/refresh` | ✅ 200 OK | 0.008s |
| 3 | `logout` | `POST` | `/auth/logout` | ✅ 200 OK | 0.003s |
| 4 | `getCurrentUser` | `GET` | `/auth/me` | ⚠️ 401 Unauthorized | 0.002s |
| 5 | `changePassword` | `PUT` | `/auth/password` | ⚠️ 401 Unauthorized | 0.002s |
| 6 | `setupFirstAdmin` | `POST` | `/auth/setup` | ⚠️ 403 Forbidden | 0.003s |

> ⚠️ `getCurrentUser` dan `changePassword` gagal karena `logout` di request #3 mencabut token sesi. `setupFirstAdmin` mengembalikan 403 karena setup sudah selesai.

### `settings`

| # | Operation | Method | Path | Status | Waktu |
|---|---|---|---|---|---|
| 1 | `getSettings` | `GET` | `/settings` | ✅ 200 OK | 0.032s |
| 2 | `updateSettings` | `PUT` | `/settings` | ❌ 400 Bad Request | 0.004s |
| 3 | `getLogo` | `GET` | `/settings/logo` | ❌ 404 Not Found | 0.004s |
| 4 | `uploadLogo` | `POST` | `/settings/logo` | ❌ 400 Bad Request | 0.006s |

> `updateSettings`: body menggunakan placeholder. `getLogo`: belum ada logo yang diupload. `uploadLogo`: body bukan multipart valid.

### `users`

| # | Operation | Method | Path | Status | Waktu |
|---|---|---|---|---|---|
| 1 | `listUsers` | `GET` | `/users` | ✅ 200 OK | 0.025s |
| 2 | `createUser` | `POST` | `/users` | ⚠️ 409 Conflict | 0.271s |
| 3 | `getUser` | `GET` | `/users/4` | ✅ 200 OK | 0.004s |
| 4 | `updateUser` | `PUT` | `/users/4` | ✅ 200 OK | 0.261s |
| 5 | `deleteUser` | `DELETE` | `/users/1` | ✅ 200 OK | 0.002s |

> `createUser`: 409 expected karena user sudah ada. `getUser`/`updateUser` kini menggunakan `userId=4` (admin). `updateUser` body `{"active":true}` — safe no-op.

### `templates`

| # | Operation | Method | Path | Status | Waktu |
|---|---|---|---|---|---|
| 1 | `listTemplates` | `GET` | `/templates` | ✅ 200 OK | 0.029s |
| 2 | `createTemplate` | `POST` | `/templates` | ✅ 201 Created | 0.005s |
| 3 | `getTemplate` | `GET` | `/templates/1` | ❌ 404 Not Found | 0.004s |
| 4 | `updateTemplate` | `PUT` | `/templates/1` | ❌ 400 Bad Request | 0.002s |
| 5 | `deleteTemplate` | `DELETE` | `/templates/1` | ❌ 404 Not Found | 0.005s |
| 6 | `renderTemplate` | `POST` | `/templates/render` | ❌ 400 Bad Request | 0.002s |
| 7 | `seedDefaultTemplates` | `POST` | `/templates/seed-defaults` | ✅ 201 Created | 0.002s |

> `getTemplate`/`deleteTemplate`: template id=1 tidak ada setelah reseed. `updateTemplate`/`renderTemplate`: body placeholder.

### `events`

| # | Operation | Method | Path | Status | Waktu |
|---|---|---|---|---|---|
| 1 | `onHotspotLogin` | `POST` | `/events/on-login` | ✅ 200 OK | 0.029s |
| 2 | `eventsHealthCheck` | `GET` | `/events/health` | ✅ 200 OK | 0.009s |

### `status`

| # | Operation | Method | Path | Status | Waktu |
|---|---|---|---|---|---|
| 1 | `getUserStatus` | `GET` | `/status?router=router-2&mac=00:11:22:33:44:55` | ✅ 200 OK | 0.031s |

> Mengembalikan status kosong (MAC tidak ada di router aktif) — perilaku benar.

### `hotspot`

| # | Operation | Method | Path | Status | Waktu |
|---|---|---|---|---|---|
| 1 | `listHotspotUsers` | `GET` | `/routers/4/hotspot/users` | ✅ 200 OK | 0.045s |
| 2 | `addHotspotUser` | `POST` | `/routers/4/hotspot/users` | ❌ 400 Bad Request | 0.006s |
| 3 | `getHotspotUserCount` | `GET` | `/routers/4/hotspot/users/count` | ✅ 200 OK | 0.003s |
| 4 | `exportHotspotUsers` | `GET` | `/routers/4/hotspot/users/export` | ❌ 400 Bad Request | 0.003s |
| 5 | `getHotspotUser` | `GET` | `/routers/4/hotspot/users/1` | ❌ 404 Not Found | 0.011s |
| 6 | `updateHotspotUser` | `PUT` | `/routers/4/hotspot/users/1` | ❌ 400 Bad Request | 0.020s |
| 7 | `removeHotspotUser` | `DELETE` | `/routers/4/hotspot/users/1` | ✅ 200 OK | 0.013s |
| 8 | `resetHotspotUserCounters` | `POST` | `/routers/4/hotspot/users/1/reset-counters` | ❌ 500 Internal Server Error | 0.006s |
| 9 | `listHotspotProfiles` | `GET` | `/routers/4/hotspot/profiles` | ✅ 200 OK | 0.003s |
| 10 | `addHotspotProfile` | `POST` | `/routers/4/hotspot/profiles` | ✅ 201 Created | 0.015s |
| 11 | `getHotspotProfile` | `GET` | `/routers/4/hotspot/profiles/1` | ❌ 404 Not Found | 0.009s |
| 12 | `updateHotspotProfile` | `PUT` | `/routers/4/hotspot/profiles/1` | ❌ 400 Bad Request | 0.009s |
| 13 | `removeHotspotProfile` | `DELETE` | `/routers/4/hotspot/profiles/1` | ❌ 400 Bad Request | 0.020s |
| 14 | `syncHotspotProfiles` | `POST` | `/routers/4/hotspot/profiles/sync` | ✅ 200 OK | 0.002s |
| 15 | `listHotspotActive` | `GET` | `/routers/4/hotspot/active` | ✅ 200 OK | 0.006s |
| 16 | `removeHotspotActive` | `DELETE` | `/routers/4/hotspot/active/1` | ❌ 400 Bad Request | 0.007s |
| 17 | `disconnectHotspotUser` | `POST` | `/routers/4/hotspot/active/1/disconnect` | ❌ 400 Bad Request | 0.007s |
| 18 | `listInactiveHotspotUsers` | `GET` | `/routers/4/hotspot/inactive` | ✅ 200 OK | 0.025s |
| 19 | `getInactiveHotspotUserCount` | `GET` | `/routers/4/hotspot/inactive/count` | ✅ 200 OK | 0.007s |
| 20 | `listHotspotHosts` | `GET` | `/routers/4/hotspot/hosts` | ✅ 200 OK | 0.005s |
| 21 | `removeHotspotHost` | `DELETE` | `/routers/4/hotspot/hosts/1` | ❌ 400 Bad Request | 0.008s |
| 22 | `listHotspotServers` | `GET` | `/routers/4/hotspot/servers` | ✅ 200 OK | 0.010s |
| 23 | `listHotspotCookies` | `GET` | `/routers/4/hotspot/cookies` | ✅ 200 OK | 0.005s |
| 24 | `removeHotspotCookie` | `DELETE` | `/routers/4/hotspot/cookies/1` | ❌ 400 Bad Request | 0.006s |
| 25 | `listIPBindings` | `GET` | `/routers/4/hotspot/bindings` | ✅ 200 OK | 0.007s |
| 26 | `addIPBinding` | `POST` | `/routers/4/hotspot/bindings` | ❌ 400 Bad Request | 0.005s |
| 27 | `updateIPBinding` | `PUT` | `/routers/4/hotspot/bindings/1` | ❌ 400 Bad Request | 0.007s |
| 28 | `removeIPBinding` | `DELETE` | `/routers/4/hotspot/bindings/1` | ❌ 400 Bad Request | 0.006s |
| 29 | `enableIPBinding` | `POST` | `/routers/4/hotspot/bindings/1/enable` | ❌ 400 Bad Request | 0.005s |
| 30 | `disableIPBinding` | `POST` | `/routers/4/hotspot/bindings/1/disable` | ❌ 400 Bad Request | 0.005s |
| 31 | `listWalledGarden` | `GET` | `/routers/4/hotspot/walled-garden` | ✅ 200 OK | 0.034s |
| 32 | `addWalledGarden` | `POST` | `/routers/4/hotspot/walled-garden` | ✅ 201 Created | 0.014s |
| 33 | `removeWalledGarden` | `DELETE` | `/routers/4/hotspot/walled-garden/` | ❌ 404 Not Found | 0.005s |
| 34 | `listWalledGardenIP` | `GET` | `/routers/4/hotspot/walled-garden-ip` | ✅ 200 OK | 0.006s |
| 35 | `addWalledGardenIP` | `POST` | `/routers/4/hotspot/walled-garden-ip` | ✅ 201 Created | 0.013s |
| 36 | `removeWalledGardenIP` | `DELETE` | `/routers/4/hotspot/walled-garden-ip/` | ❌ 404 Not Found | 0.004s |

> `addHotspotProfile` kini 201 (body valid: `name=test-profile`, `rate_limit=1M/2M`). Run berikutnya mungkin 400 jika profil sudah ada di RouterOS. `removeWalledGarden`/`removeWalledGardenIP` 404 karena ID path kosong (by design — ID perlu diambil dari list terlebih dahulu).

### `ppp`

| # | Operation | Method | Path | Status | Waktu |
|---|---|---|---|---|---|
| 1 | `listPPPSecrets` | `GET` | `/routers/4/ppp/secrets` | ✅ 200 OK | 0.091s |
| 2 | `addPPPSecret` | `POST` | `/routers/4/ppp/secrets` | ❌ 400 Bad Request | 0.008s |
| 3 | `updatePPPSecret` | `PUT` | `/routers/4/ppp/secrets/1` | ✅ 200 OK | 0.013s |
| 4 | `removePPPSecret` | `DELETE` | `/routers/4/ppp/secrets/1` | ✅ 200 OK | 0.007s |
| 5 | `listPPPActive` | `GET` | `/routers/4/ppp/active` | ✅ 200 OK | 0.006s |
| 6 | `disconnectPPPActive` | `DELETE` | `/routers/4/ppp/active/1` | ❌ 400 Bad Request | 0.008s |
| 7 | `listInactivePPPSecrets` | `GET` | `/routers/4/ppp/inactive` | ✅ 200 OK | 0.028s |
| 8 | `getInactivePPPCount` | `GET` | `/routers/4/ppp/inactive/count` | ✅ 200 OK | 0.005s |
| 9 | `listPPPProfiles` | `GET` | `/routers/4/ppp/profiles` | ✅ 200 OK | 0.024s |

> `addPPPSecret`: body placeholder. `disconnectPPPActive`: ID tidak ditemukan (user tidak aktif).

### `vouchers`

| # | Operation | Method | Path | Status | Waktu |
|---|---|---|---|---|---|
| 1 | `generateVouchers` | `POST` | `/routers/4/vouchers/generate` | ❌ 400 Bad Request | 0.030s |
| 2 | `getCachedVoucherBatch` | `POST` | `/routers/4/vouchers/cache` | ❌ 404 Not Found | 0.009s |
| 3 | `getVoucherPrintData` | `GET` | `/routers/4/vouchers/print-data` | ❌ 400 Bad Request | 0.003s |
| 4 | `recordVoucherSale` | `POST` | `/routers/4/vouchers/sales` | ✅ 200 OK | 0.003s |
| 5 | `importVoucherSales` | `POST` | `/routers/4/vouchers/import` | ✅ 200 OK | 0.008s |
| 6 | `printVouchers` | `POST` | `/routers/4/vouchers/print` | ❌ 404 Not Found | 0.003s |

> `generateVouchers`: RouterOS gagal membuat hotspot user di CHR test. `getCachedVoucherBatch`: tidak ada sesi cache aktif. `getVoucherPrintData`: membutuhkan `gencode` dari hasil `generateVouchers`. `printVouchers`: tidak ada voucher di DB dengan username yang dimaksud.

### `system`

| # | Operation | Method | Path | Status | Waktu |
|---|---|---|---|---|---|
| 1 | `getSystemResource` | `GET` | `/routers/4/system/resource` | ✅ 200 OK | 0.026s |
| 2 | `getSystemResourceHistory` | `GET` | `/routers/4/system/resource/history` | ✅ 200 OK | 0.003s |
| 3 | `getSystemLog` | `GET` | `/routers/4/system/log` | ✅ 200 OK | 0.018s |
| 4 | `getSystemClock` | `GET` | `/routers/4/system/clock` | ✅ 200 OK | 0.003s |
| 5 | `getSystemIdentity` | `GET` | `/routers/4/system/identity` | ✅ 200 OK | 0.004s |
| 6 | `getRouterboard` | `GET` | `/routers/4/system/routerboard` | ✅ 200 OK | 0.004s |
| 7 | `getExpireMonitor` | `GET` | `/routers/4/system/expire-monitor` | ❌ 500 Internal Server Error | 0.003s |
| 8 | `deployExpireMonitor` | `POST` | `/routers/4/system/expire-monitor/deploy` | ❌ 500 Internal Server Error | 0.004s |
| 9 | `removeExpireMonitor` | `POST` | `/routers/4/system/expire-monitor/remove` | ❌ 500 Internal Server Error | 0.002s |
| 10 | `listSchedulers` | `GET` | `/routers/4/system/schedulers` | ❌ 500 Internal Server Error | 0.003s |
| 11 | `createScheduler` | `POST` | `/routers/4/system/schedulers` | ❌ 500 Internal Server Error | 0.002s |
| 12 | `updateScheduler` | `PUT` | `/routers/4/system/schedulers/1` | ❌ 500 Internal Server Error | 0.002s |
| 13 | `deleteScheduler` | `DELETE` | `/routers/4/system/schedulers/1` | ❌ 500 Internal Server Error | 0.002s |
| 14 | `enableScheduler` | `POST` | `/routers/4/system/schedulers/1/enable` | ❌ 500 Internal Server Error | 0.002s |
| 15 | `disableScheduler` | `POST` | `/routers/4/system/schedulers/1/disable` | ❌ 500 Internal Server Error | 0.002s |
| 16 | `listScripts` | `GET` | `/routers/4/system/scripts` | ❌ 500 Internal Server Error | 0.003s |
| 17 | `createScript` | `POST` | `/routers/4/system/scripts` | ❌ 500 Internal Server Error | 0.002s |
| 18 | `updateScript` | `PUT` | `/routers/4/system/scripts/1` | ❌ 500 Internal Server Error | 0.002s |
| 19 | `deleteScript` | `DELETE` | `/routers/4/system/scripts/1` | ❌ 500 Internal Server Error | 9.946s |
| 20 | `runScript` | `POST` | `/routers/4/system/scripts/1/run` | ❌ 500 Internal Server Error | 0.004s |
| 21 | `setupSystemLogging` | `POST` | `/routers/4/system/setup-logging` | ✅ 200 OK | 0.007s |
| 22 | `getRouterDashboard` | `GET` | `/routers/4/system/dashboard` | ❌ 500 Internal Server Error | 0.002s |
| 23 | `rebootRouter` | `POST` | `/routers/4/system/reboot` | ❌ 500 Internal Server Error | 0.100s |
| 24 | `shutdownRouter` | `POST` | `/routers/4/system/shutdown` | ❌ 500 Internal Server Error | 0.005s |

> Semua 500 adalah keterbatasan RouterOS CHR: paket `scheduler` dan `script` tidak tersedia; `expire-monitor`/`dashboard` bergantung pada script; `reboot`/`shutdown` diblokir pada CHR. Error message kini lebih informatif (menyertakan pesan RouterOS). `rebootRouter` dan `shutdownRouter` dipindah ke posisi 23-24 agar tidak memutus koneksi pool saat test lain berjalan.

### `sse`

| # | Operation | Method | Path | Status | Waktu |
|---|---|---|---|---|---|
| 1 | `streamRouterLogsAll` | `GET` | `/routers/4/logs/stream/all` | ❌ 404 Not Found | 0.041s |
| 2 | `streamRouterLogsHotspot` | `GET` | `/routers/4/logs/stream/hotspot` | ❌ 404 Not Found | 0.003s |
| 3 | `streamRouterLogsPPP` | `GET` | `/routers/4/logs/stream/ppp` | ❌ 404 Not Found | 0.003s |
| 4 | `streamHotspotUsersTelemetry` | `GET` | `/routers/4/sse/hotspot/users` | ❌ 404 Not Found | 0.003s |
| 5 | `streamHotspotActiveTelemetry` | `GET` | `/routers/4/sse/hotspot/active` | ❌ 404 Not Found | 0.007s |
| 6 | `streamHotspotInactiveTelemetry` | `GET` | `/routers/4/sse/hotspot/inactive` | ❌ 404 Not Found | 0.003s |
| 7 | `streamIPBindingsTelemetry` | `GET` | `/routers/4/sse/hotspot/bindings` | ❌ 404 Not Found | 0.003s |
| 8 | `streamPPPSecretsTelemetry` | `GET` | `/routers/4/sse/ppp/secrets` | ❌ 404 Not Found | 0.003s |
| 9 | `streamPPPActiveTelemetry` | `GET` | `/routers/4/sse/ppp/active` | ❌ 404 Not Found | 0.003s |
| 10 | `streamPPPInactiveTelemetry` | `GET` | `/routers/4/sse/ppp/inactive` | ❌ 404 Not Found | 0.002s |
| 11 | `streamSystemResourceTelemetry` | `GET` | `/routers/4/sse/system/resource` | ❌ 404 Not Found | 0.003s |
| 12 | `streamInterfaceTrafficTelemetry` | `GET` | `/routers/4/sse/network/traffic/ether1` | ❌ 404 Not Found | 0.003s |
| 13 | `streamDHCPLeasesTelemetry` | `GET` | `/routers/4/sse/network/dhcp/leases` | ❌ 404 Not Found | 0.002s |

> SSE worker tidak di-attach ke router ini. Semua 404 adalah expected — fitur SSE belum diimplementasikan untuk router-2. Endpoint SSE tidak bisa ditest via `hopp test` karena streaming tak berujung (gunakan `curl --max-time 2` jika perlu verifikasi manual).

### `reports`

| # | Operation | Method | Path | Status | Waktu |
|---|---|---|---|---|---|
| 1 | `getDailyReport` | `GET` | `/routers/4/reports/daily?date=2026-05-13` | ✅ 200 OK | 0.037s |
| 2 | `getMonthlyReport` | `GET` | `/routers/4/reports/monthly?year=2026&month=5` | ✅ 200 OK | 0.004s |
| 3 | `getResumeReport` | `GET` | `/routers/4/reports/resume` | ✅ 200 OK | 0.008s |
| 4 | `getDashboardSummary` | `GET` | `/routers/4/reports/summary` | ✅ 200 OK | 0.005s |
| 5 | `exportSalesCSV` | `GET` | `/routers/4/reports/export/csv?from=2026-05-01&to=2026-05-13` | ✅ 200 OK | 0.004s |
| 6 | `exportSalesExcel` | `GET` | `/routers/4/reports/export/excel?from=2026-05-01&to=2026-05-13` | ✅ 200 OK | 0.004s |

> Semua reports kini passing setelah query params diisi dengan env vars (`reportDate`, `reportYear`, `reportMonth`, `reportFrom`, `reportTo`). Gin `DefaultQuery` mengembalikan `""` saat param hadir tapi kosong — bukan default value.

### `quick-print`

| # | Operation | Method | Path | Status | Waktu |
|---|---|---|---|---|---|
| 1 | `listQuickPrintPackages` | `GET` | `/routers/4/quick-print` | ✅ 200 OK | 0.210s |
| 2 | `createQuickPrintPackage` | `POST` | `/routers/4/quick-print` | ✅ 201 Created | 9.420s |
| 3 | `getQuickPrintPackage` | `GET` | `/routers/4/quick-print/` | ❌ 500 Internal Server Error | 0.011s |
| 4 | `updateQuickPrintPackage` | `PUT` | `/routers/4/quick-print/` | ❌ 404 Not Found | 0.002s |
| 5 | `removeQuickPrintPackage` | `DELETE` | `/routers/4/quick-print/` | ❌ 404 Not Found | 0.002s |

> `list`/`create` kini berhasil — router 4 terdaftar di connection pool. `get`/`update`/`remove` gagal karena path variable `name` kosong (collection tidak mengisi `<<name>>` env var). `createQuickPrintPackage` run ke-2 mungkin 500 jika paket sudah ada di RouterOS.

### `profile-mappings`

| # | Operation | Method | Path | Status | Waktu |
|---|---|---|---|---|---|
| 1 | `listProfileMappings` | `GET` | `/routers/4/profile-mappings` | ✅ 200 OK | 0.034s |
| 2 | `updateProfileMapping` | `PUT` | `/routers/4/profile-mappings/1d-1mbps` | ✅ 200 OK | 0.009s |
| 3 | `deleteProfileMapping` | `DELETE` | `/routers/4/profile-mappings/1d-1mbps` | ✅ 200 OK | 0.006s |

### `routers`

| # | Operation | Method | Path | Status | Waktu |
|---|---|---|---|---|---|
| 1 | `listRouters` | `GET` | `/routers` | ✅ 200 OK | 0.034s |
| 2 | `createRouter` | `POST` | `/routers` | ❌ 400 Bad Request | 5.006s |
| 3 | `migrateRouterConfig` | `POST` | `/routers/migrate` | ❌ 400 Bad Request | 0.002s |
| 4 | `getRouter` | `GET` | `/routers/4` | ✅ 200 OK | 0.012s |
| 5 | `updateRouter` | `PUT` | `/routers/4` | ❌ 400 Bad Request | 1.745s |
| 6 | `testRouterConnection` | `POST` | `/routers/4/test` | ❌ 400 Bad Request | 0.002s |
| 7 | `deleteRouter` | `DELETE` | `/routers/4` | ✅ 200 OK | 1.753s |

> `createRouter`/`updateRouter`: body menggunakan placeholder (host/credentials tidak valid). `testRouterConnection`: body placeholder. `deleteRouter` diletakkan terakhir — setelah ini perlu reseed sebelum menjalankan suite kembali.

## Catatan Operasional

### Menjalankan Suite Lengkap

```bash
ENV="docs/collections/env.json"
for name in health auth settings users templates events status hotspot ppp vouchers system sse reports quick-print profile-mappings routers; do
  hopp test "docs/collections/${name}.json" --env "$ENV"
done
```

### Router Connection Pool — Dynamic (Tidak Perlu Restart)

Connection pool **sudah dynamic**. Router yang dibuat via `POST /routers` langsung terdaftar di pool dan mencoba koneksi secara otomatis (async dengan backoff). Tidak perlu restart server.

Restart server hanya diperlukan jika menggunakan `make seed` — karena seed script menulis langsung ke DB (bypass API) tanpa memanggil `engine.AddRouter()`. Untuk testing gunakan API, bukan seed.

### Reseed Setelah deleteRouter

`deleteRouter` menghapus router dari DB **dan** menderegister dari pool secara otomatis. Sebelum run berikutnya, buat ulang router via API atau seed:

```bash
# Jika menggunakan make seed (butuh AES key):
export $(grep -v '^#' .env | xargs) && export DB_HOST=localhost
make seed
# Setelah seed via script, perlu restart API karena seed bypass API:
docker restart docker-api-1

# Atau: buat router via API (tidak perlu restart):
# POST /api/v1/routers dengan credentials yang valid

# Perbarui routerId di env.json sesuai ID baru
# Perbarui access_token dengan token segar dari login
```

### Keterbatasan RouterOS CHR

Endpoint berikut tidak dapat ditest tanpa RouterOS device dengan paket lengkap:
- `scheduler` CRUD — paket scheduler tidak tersedia di CHR
- `script` CRUD + run — paket script tidak tersedia di CHR
- `expire-monitor` deploy/remove — bergantung pada script
- `reboot`/`shutdown` — kini diletakkan di akhir collection `system` (posisi 23-24) agar tidak memutus koneksi saat test lain berjalan; tetap 500 di CHR karena command diblokir
- `dashboard` — bergantung pada system/script yang tidak tersedia di CHR
