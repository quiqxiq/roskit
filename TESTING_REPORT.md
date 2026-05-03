# Laporan Hasil Pengujian Komprehensif: Roskit Project

Laporan ini menyajikan hasil eksekusi pengujian secara keseluruhan pada proyek Roskit. Pengujian mencakup unit test dasar (Go), integration test dengan layanan eksternal (Redis, InfluxDB, MikroTik RouterOS), serta pengujian API End-to-End (E2E) menggunakan Python HTTP.

## 1. Konfigurasi dan Lingkungan (Environment)

Pengujian dilakukan dengan memanfaatkan Docker Compose (`docker/docker-compose.dev.yml`) untuk menyediakan infrastruktur lokal:
- **PostgreSQL**: Port 5432
- **Redis**: Port 6379
- **InfluxDB 3**: Port 8181
- **API Server**: Port 8080

Konfigurasi dimuat dari file environment yang telah disediakan:
- `.env`: Digunakan oleh kontainer Docker.
- `.env.test`: Digunakan khusus untuk integration test. Memuat kredensial real device MikroTik (`192.168.233.1:8728`), kredensial DB, Redis, InfluxDB, serta secret keys (JWT & AES).

## 2. Hasil Eksekusi Pengujian

Secara umum, *core logic* dan fungsionalitas dasar aplikasi berjalan dengan sangat baik. Namun, terdapat beberapa isu integrasi yang perlu diperbaiki, terutama terkait InfluxDB dan proses Autentikasi pada E2E test.

### 2.1. Go Unit Tests (Pure Logic)
**Status: ✅ PASSED (100%)**

Semua unit test yang tidak bergantung pada tag build khusus (seperti database atau router fisik) berhasil dijalankan tanpa masalah.
- **Package `pkg/encrypt`**: Seluruh 9 test lulus (termasuk kompatibilitas dekripsi PHP legacy dan base64 handling). *Catatan: Test dijalankan tanpa flag `-race` karena CGO dinonaktifkan secara default di environment Alpine/Windows PowerShell ini.*
- **Package `internal/roskit/core/...`**: Parsing reply RouterOS (model, network, ppp, system, parser) berjalan sempurna.
- **Package `internal/roskit/behavior/...`**: Logika stream sink, fan-out, dan concurrency lulus.
- **Package `internal/roskit/execution/...`**: Pembentukan command (Add, Set, Remove) dan helper koneksi lulus.
- **Package `internal/api/handlers`**: Pengujian handler API (mocking) lulus.

### 2.2. Go Integration Tests

Pengujian ini menggunakan flag build khusus (`-tags`) untuk menjalankan test yang terhubung langsung dengan layanan pendukung.

#### A. Redis Integration (`-tags redis`)
**Status: ✅ PASSED (18/18 Tests Lulus)**
- Modul cache, event processor, dan Pub/Sub berhasil terhubung dan berinteraksi dengan Redis container di `localhost:6379`.
- Operasi seperti Set/Get Snapshot, Indexing, dan pengiriman pesan Pub/Sub (Channel log dan event) tervalidasi dengan baik.

#### B. InfluxDB Integration (`-tags influxdb`)
**Status: ❌ FAILED (2 Gagal, 32 Lulus)**
- Penulisan data (Write Protocol) menggunakan Influx Writer sebagian besar berhasil.
- **Isu yang ditemukan pada Influx Reader**:
  1. `TestInfluxReader_QueryRange_EmptyResult`: Gagal dengan error `serde json error: expected value at line 1 column 1` (Kemungkinan respons HTTP 400/kosong dari InfluxDB saat data tidak ditemukan, yang gagal di-parse oleh JSON decoder).
  2. `TestInfluxReader_QueryRange_ReturnsData`: Gagal karena ketidaksesuaian tipe data pada schema InfluxDB: `invalid column type for column 'cpu-load', expected iox::column_type::field::integer, got iox::column_type::field::float`. Aplikasi mengirimkan float, tetapi schema di InfluxDB mengekspektasikan integer untuk field `cpu-load`.

#### C. MikroTik Integration (`-tags mikrotik`)
**Status: ⚠️ PARTIAL SUCCESS (Sebagian Gagal)**
Pengujian langsung ke router fisik (`192.168.233.1:8728`) menunjukkan bahwa pipeline stream dan polling bekerja dengan baik, namun ada kegagalan pada operasi mutasi dan query:
- **Stream & Poll**: Test `TestStreamWorker_ReceiveEvents`, `TestLogWorker_FilterHotspot`, dan `TestPollWorker_SystemResource` lulus dengan baik. Aplikasi berhasil mempertahankan koneksi persisten dan menerima data secara real-time.
- **Isu yang ditemukan**:
  1. `TestMutation_AddRemoveIPBinding`: Gagal dengan error `from RouterOS device: no such item`. Router mungkin menolak format ID atau item sudah tidak ada saat dicoba dihapus.
  2. `TestQuery_QueryOne` & `TestQuery_Filters`: Gagal saat mencoba mem-filter data atau saat ekspektasi hasil yang dikembalikan tidak sesuai dengan kondisi router saat itu.
  3. Terdapat kegagalan kompilasi (build failed) pada package `orchestrator_test` dan `service_test` akibat *import cycle* atau ketidaksesuaian tipe argumen saat memanggil fungsi *cleanup* (`engine.Dispatcher()` digunakan sebagai interface `Bridge`).

### 2.3. Python HTTP API Integration Tests (`tests/http/`)
**Status: ❌ FAILED (17 Gagal, 27 Lulus)**

Suite pengujian API E2E yang menggunakan `pytest` dan `requests` mengalami kegagalan massal yang bersumber dari kegagalan Autentikasi.

- **Analisis Kegagalan**:
  Hampir semua test (Login, Hotspot Users, Routers, MikroTik Realtime) mengembalikan HTTP 401 Unauthorized (`{"error":"invalid username or password"}`).
- **Penyebab Akar (Root Cause)**:
  1. **Konfigurasi Kredensial Uji**: File `config.py` mengekspektasikan default user `admin` dengan password `admin123`.
  2. **Proses Setup API**: Log dari kontainer `api-1` menunjukkan bahwa saat endpoint `/api/v1/auth/setup` dipanggil untuk pertama kali (sebelum test dijalankan), aplikasi justru membuat user dengan nama `newadmin` ke dalam tabel `system_users`.
  3. **Ketidaksesuaian Enkripsi/Hashing**: Meskipun pengujian telah dicoba ulang dengan kredensial `newadmin` / `admin1234` atau `admin` / `admin1234` (sesuai curl setup di Makefile), autentikasi tetap gagal. Hal ini kuat mengindikasikan adanya perbedaan pada environment variable (seperti `AES_ENCRYPTION_KEY` atau secret JWT) antara container Docker yang berjalan dan lingkungan lokal tempat skrip Python dijalankan, atau adanya kegagalan validasi hash bcrypt di level database akibat seed yang tidak cocok.

---

## 3. Kesimpulan dan Rekomendasi Perbaikan

Infrastruktur dan desain arsitektur proyek Roskit (`Bridge`, `Dispatcher`, `Pipeline`, `Stream/Poll`) terbukti solid berdasarkan hasil unit test dan Redis test. Namun, untuk memastikan CI/CD dan operasional yang stabil, beberapa perbaikan teknis harus segera dilakukan:

### 3.1. Perbaikan InfluxDB
1. **Schema Mismatch**: Perbaiki konversi data pada package `timeseries`. Jika InfluxDB mengharapkan `cpu-load` sebagai `integer`, pastikan parser MikroTik atau Influx Writer melakukan konversi (`int64`) sebelum mem-flush Line Protocol, jangan mengirimnya sebagai `float64`.
2. **Error Handling Reader**: Tambahkan pengecekan status code (misal 200 vs 400/404) pada `InfluxReader` sebelum mencoba melakukan unmarshal JSON, agar error "expected value" dapat ditangani dengan lebih rapi (misalnya me-return `nil, nil` jika data kosong).

### 3.2. Perbaikan MikroTik Integration Tests
1. **Fix Build Errors**: Perbaiki *import cycle* di `internal/roskit/execution` dan sesuaikan tipe argumen pada fungsi `cleanup` di file `engine_test.go` dan `bridge_hotspot_test.go` agar sesuai dengan *interface* yang diharapkan oleh fungsi *testhelpers*.
2. **State Management Router**: Pada test mutasi seperti IP Binding, pastikan item benar-benar berhasil dibuat dan ID-nya valid sebelum mencoba melakukan set/remove untuk menghindari error `no such item`.

### 3.3. Perbaikan Python HTTP Tests (Autentikasi)
1. **Sinkronisasi Database Seed / Setup**: Pastikan endpoint setup atau skrip *seed* database yang dieksekusi saat inisialisasi Docker (`make seed-docker` atau migrasi) memasukkan user standar yang konsisten (misal: `admin` / `admin123`), dan pastikan skrip `tests/http/config.py` menggunakan kredensial yang presisi.
2. **Validasi Environment Variables**: Pastikan bahwa file `.env` yang di-load oleh API Container sama persis isinya (khususnya untuk `AES_ENCRYPTION_KEY` dan `JWT_SECRET`) dengan `.env.test` yang mungkin dibaca oleh skrip.
3. **Database Cleanup**: Untuk testing yang konsisten, ada baiknya menambahkan mekanisme teardown di Pytest untuk mengosongkan tabel `system_users` dan melakukan setup ulang (`POST /api/v1/auth/setup`) di setiap awal *test session* agar terhindar dari *state* yang kotor ("setup already completed").
