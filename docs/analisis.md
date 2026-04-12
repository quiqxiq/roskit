# Arsitektur Lanjut Sistem Kolektor Telemetri Jaringan

> Integrasi Clean Architecture Golang dengan MikroTik API, InfluxDB, dan Redis

## Daftar Isi

- [Pendahuluan](#pendahuluan)
- [Paradigma Pemantauan Jaringan Modern](#paradigma-pemantauan-jaringan-modern)
- [Analisis Perbandingan Protokol](#analisis-perbandingan-protokol)
- [Clean Architecture dalam Golang](#clean-architecture-dalam-golang)
- [Mekanisme Protokol API MikroTik](#mekanisme-protokol-api-mikrotik)
- [Persistensi Data dengan InfluxDB](#persistensi-data-dengan-influxdb)
- [Strategi Caching dengan Redis](#strategi-caching-dengan-redis)
- [Perancangan Folder Spec](#perancangan-folder-spec)
- [Arsitektur Engine Collector](#arsitektur-engine-collector)
- [Metrik Jaringan dan Kalkulasi](#metrik-jaringan-dan-kalkulasi)
- [Kesimpulan](#kesimpulan)

---

## Pendahuluan

Transformasi digital dalam pengelolaan infrastruktur jaringan telah bergeser dari model pemantauan reaktif berbasis Simple Network Management Protocol (SNMP) menuju paradigma telemetri berbasis streaming yang proaktif.

Kebutuhan akan data real-time dengan resolusi tinggi mendorong pengembangan sistem yang mampu menangani beban kerja yang sangat besar dengan latensi minimal. Dalam ekosistem perangkat keras MikroTik, pemanfaatan antarmuka pemrograman aplikasi (API) biner memberikan keunggulan dibandingkan metode tradisional karena kemampuannya dalam mendukung operasi asinkron dan aliran data kontinu.

Implementasi sistem semacam ini dalam bahasa pemrograman Go (Golang) memerlukan pendekatan arsitektural yang disiplin untuk memastikan kode tetap dapat dipelihara, diuji, dan diperluas. Penggunaan **Clean Architecture** menjadi fondasi utama dalam memisahkan logika bisnis dari detail teknis infrastruktur, seperti manajemen basis data dan protokol jaringan.

---

## Paradigma Pemantauan Jaringan Modern

### Dari Polling ke Streaming

Evolusi pemantauan jaringan telah mencapai titik di mana interval polling tradisional dalam hitungan menit tidak lagi mencukupi untuk mendeteksi anomali sesaat seperti lonjakan trafik atau kegagalan antarmuka yang singkat.

SNMP, meskipun masih digunakan secara luas karena kompatibilitasnya yang universal, memiliki keterbatasan dalam hal skalabilitas dan efisiensi karena setiap permintaan data memerlukan siklus tanya-jawab yang membebani CPU perangkat. Sebaliknya, telemetri berbasis streaming memungkinkan perangkat untuk secara aktif mengirimkan data hanya ketika terjadi perubahan atau berdasarkan interval waktu yang sangat singkat.

### Streaming dalam MikroTik RouterOS

Dalam konteks MikroTik RouterOS, fitur streaming diwujudkan melalui perintah `/listen` atau `print follow` yang tersedia di berbagai menu API. Mekanisme ini memastikan bahwa setiap perubahan status antarmuka, penambahan sewa DHCP, atau aktivitas pengguna hotspot dapat segera ditangkap oleh kolektor eksternal tanpa harus melakukan permintaan berulang-ulang.

Data yang dihasilkan oleh aliran ini bersifat dinamis dan sering kali mengandung indikator status seperti `.dead=yes` yang menandakan penghapusan suatu entitas dari sistem.

---

## Analisis Perbandingan Protokol

| Fitur | SNMP | MikroTik API (Biner) | REST API (JSON Wrapper) |
|-------|------|---------------------|------------------------|
| **Model Data** | MIB OIDs (Numerik) | Atribut Kalimat (String/Biner) | JSON Payloads |
| **Efisiensi** | Rendah (Polling) | Tinggi (Streaming & Tagging) | Sedang (Stateless HTTP) |
| **Real-time** | Terbatas (Traps) | Sangat Baik (Listen Command) | Terbatas (Polling/Webhook) |
| **Kompleksitas** | Tinggi (Kompilasi MIB) | Sedang (Protokol Kustom) | Rendah (Standard HTTP) |
| **Keamanan** | Community String/v3 | TLS/SSL Encrypted | HTTP Basic Auth + TLS |

> **Kesimpulan:** API biner tetap menjadi pilihan utama untuk pengembangan pustaka performa tinggi karena latensinya yang lebih rendah dibandingkan pembungkus REST. Koneksi TCP persisten memungkinkan multiplexing beberapa perintah melalui fitur tagging.

---

## Clean Architecture dalam Golang

Clean Architecture, yang dipopulerkan oleh Robert C. Martin, menekankan pada pemisahan kepentingan (*separation of concerns*) melalui lapisan-lapisan yang memiliki tanggung jawab spesifik.

### Stratifikasi Lapisan

Arsitektur disusun dalam lingkaran konsentris di mana **ketergantungan hanya boleh mengalir ke arah dalam**:

```
┌─────────────────────────────────┐
│      Infrastruktur (Outer)      │  ← Redis, InfluxDB, MikroTik API
├─────────────────────────────────┤
│      Use Case (Middle)          │  ← Logika routing event
├─────────────────────────────────┤
│      Entity (Inner)             │  ← Struktur data murni
└─────────────────────────────────┘
```

- **Entity (Domain):** Mendefinisikan struktur data fundamental seperti statistik antarmuka atau kesehatan sistem tanpa referensi ke pustaka eksternal.
- **Use Case:** Mengoordinasikan aliran data antara entitas dan repositori.
- **Infrastruktur (Adapter):** Berisi implementasi nyata dari akses data — klien API MikroTik, klien Redis, dan klien InfluxDB.

> Jika di masa depan Redis perlu diganti dengan Memcached, perubahan hanya terjadi pada lapisan adapter tanpa merusak logika bisnis utama.

### Organisasi Proyek

- **Folder `collector`:** Mesin utama yang mengelola siklus hidup koneksi ke perangkat. Bertanggung jawab untuk inisialisasi, reconnection, dan orkestrasi konkurensi melalui goroutines.
- **Folder `spec`:** Lokasi penyimpanan spesifikasi perintah yang bersifat atomik. Setiap file mendefinisikan kalimat perintah API dan logika pemrosesan data mentah.

> Pemisahan ke file-file kecil mencegah "God Object" dan memungkinkan penambahan sensor data baru tanpa memodifikasi logika inti.

---

## Mekanisme Protokol API MikroTik

### Struktur Kata dan Kalimat

RouterOS API menggunakan format kalimat yang terdiri dari serangkaian kata, dimana setiap kata diawali dengan panjang yang dikodekan dalam network byte order.

**Jenis kata:**

| Jenis | Prefix | Contoh |
|-------|--------|--------|
| Kata Perintah | `/` | `/ip/address/print` |
| Kata Atribut | `=` | `=disabled=yes` |
| Kata Atribut API | `.` | `.tag`, `.proplist` |
| Kata Kueri | `?` | `?address=192.168.1.0/24` |

Kalimat diakhiri dengan kata panjang nol sebagai terminator. Tanpa terminator ini, router tidak akan mengevaluasi kata-kata yang diterima.

### Konkurensi melalui Tagging

Fitur paling kuat adalah kemampuan menjalankan beberapa perintah secara bersamaan melalui satu koneksi socket menggunakan atribut `.tag`:

| Langkah | Kalimat Klien | Respons Router | Penjelasan |
|---------|--------------|----------------|------------|
| 1 | `/interface/listen .tag=eth-stream` | `!re .tag=eth-stream ...` | Memulai aliran data kontinu |
| 2 | `/system/resource/print .tag=sys-res` | `!re .tag=sys-res ... !done .tag=sys-res` | Mengambil data statis sekali jalan |
| 3 | `/cancel =tag=eth-stream .tag=cnc` | `!trap .tag=eth-stream ... !done .tag=cnc` | Membatalkan aliran data tertentu |

> Mekanisme ini sangat krusial bagi folder `collector`. Dengan satu koneksi persisten dan banyak tag, pustaka menghemat sumber daya dan meminimalkan beban pada router.

---

## Persistensi Data dengan InfluxDB

InfluxDB adalah basis data deret waktu yang dioptimalkan untuk menyimpan miliaran poin data per detik.

### Batching

Metode non-blocking dengan implicit batching sangat direkomendasikan untuk performa tinggi:
- Buffer internal mengumpulkan data hingga **batch size** (5.000 poin) atau **flush interval** (1 detik)
- Dikirim secara massal melalui permintaan tunggal
- Mengurangi overhead jaringan secara dramatis

### Line Protocol

Data menggunakan format:

```
measurement,tag_set field_set timestamp
```

Contoh:

```
interface_stats,router=core-01,name=ether1 rx_byte=1024000i,tx_byte=512000i 1712900000000
```

### Tips Optimasi

- **Kompresi Gzip:** Meningkatkan kinerja tulis hingga 5x
- **Tag sorting:** Urutkan tag secara leksikografis untuk mempercepat pengindeksan
- **Presisi waktu:** Gunakan presisi detik (`s`) jika data dikumpulkan per detik untuk menghemat penyimpanan

---

## Strategi Caching dengan Redis

Redis bertindak sebagai lapisan perantara yang menyediakan akses data latensi rendah.

### Status Cache (HSET)

Setiap data telemetri yang diterima segera diperbarui di Redis menggunakan `HSET`:

```
HSET roskit:core-01:interface_stats:ether1 rx_byte "1024000" tx_byte "512000" ...
```

Kunci hash menggunakan format `roskit:{router_id}:{measurement}:{entity_id}`, memungkinkan pengambilan seluruh atribut dalam satu `HGETALL`.

### Pub/Sub vs Streams

| Kriteria | Redis Pub/Sub | Redis Streams |
|----------|--------------|---------------|
| **Model Komunikasi** | Push (fire-and-forget) | Pull/Blocking |
| **Persistensi** | Tidak ada | Ya (durable) |
| **Delivery Semantics** | At-most-once | At-least-once (dengan ACK) |
| **Skalabilitas Konsumen** | Terbatas | Tinggi (via Consumer Groups) |
| **Use Case** | Real-time Broadcast | Event Sourcing / Task Queue |

> RosKit saat ini menggunakan **Pub/Sub** untuk notifikasi real-time. Untuk kebutuhan processing yang lebih reliable, pertimbangkan migrasi ke **Streams**.

---

## Perancangan Folder Spec

### Abstraksi Perintah

Setiap spesifikasi perintah mengimplementasikan interface `StreamSpec`:

```go
type StreamSpec interface {
    Command() []string                               // Kalimat perintah API
    Tag() string                                     // Tag unik
    Measurement() string                             // Nama measurement InfluxDB
    Parse(routerID string, s *proto.Sentence) (*TelemetryEvent, error)
}
```

### Modularitas

Setiap implementasi dipisahkan ke file sendiri:

| File | Perintah API | Data |
|------|-------------|------|
| `interfaces.go` | `/interface/print =follow =interval=1s` | Bandwidth |
| `resource.go` | `/system/resource/print =interval=5s` | CPU/Memory |
| `queue_simple.go` | `/queue/simple/print =stats =interval=1s` | Queue stats |
| `log.go` | `/log/print =follow` | Log entries |
| `dhcp_lease.go` | `/ip/dhcp-server/lease/print =follow` | DHCP leases |
| `hotspot_active.go` | `/ip/hotspot/active/print =follow` | Hotspot users |
| `ppp_active.go` | `/ppp/active/print =follow` | PPPoE sessions |
| `ppp_profile.go` | `/ppp/profile/print =follow` | PPP profiles |
| `ppp_secret.go` | `/ppp/secret/print =follow` | PPP secrets |

### Penanganan `.dead`

Ketika entity dihapus dari router, sistem mengirim `.dead=yes`. Spec harus mendeteksi ini dan menghasilkan `EventDead` yang akan menghapus cache Redis.

---

## Arsitektur Engine Collector

### Manajemen Koneksi

Sistem menggunakan `routeros.DialContext` atau `routeros.DialTLSContext` untuk koneksi awal, diikuti `Async()` untuk mengaktifkan pemrosesan asinkron. Setiap perintah dari folder spec dijalankan menggunakan `ListenArgsQueueContext`.

### Goroutines

Collector menjalankan **satu goroutine per perintah streaming** untuk memantau respons secara konkuren. Jika satu sensor data terhenti, yang lain tetap berfungsi.

### Graceful Shutdown

`context.Context` digunakan di seluruh lapisan untuk penghentian bersih:

1. Context di-cancel
2. Collector mengirim `/cancel` ke router
3. Semua stream goroutine selesai
4. Socket koneksi ditutup

> Ini mencegah kebocoran sumber daya di sisi router yang sering tetap menjalankan proses API jika koneksi ditutup paksa tanpa sinyal pembatalan.

---

## Metrik Jaringan dan Kalkulasi

### Perhitungan Throughput

Data `rx-byte` dan `tx-byte` dari `/interface/print stats` merepresentasikan total byte akumulatif. Untuk mendapatkan bandwidth (bits per second):

```
bps = ((Bytes_current - Bytes_previous) × 8) / (t_current - t_previous)
```

### Analisis Saturasi

Indikator kesehatan antarmuka yang perlu dipantau:

| Metrik | Indikasi |
|--------|----------|
| `rx-drop` / `tx-drop` | Saturasi bandwidth atau buffer overflow |
| `rx-error` / `tx-error` | Kegagalan perangkat keras (kabel, driver) |
| Packet loss rate | Masalah QoS atau congestion |

> Peringatan dapat dikonfigurasi berdasarkan laju peningkatan galat dalam jendela waktu tertentu (contoh: > 10 error dalam 5 menit).

---

## Kesimpulan

Rancangan arsitektur pustaka telemetri Golang ini menawarkan solusi komprehensif bagi tantangan pemantauan infrastruktur jaringan modern:

- **Clean Architecture** memastikan kode dapat dipelihara dan diperluas
- **API biner MikroTik** memberikan efisiensi streaming real-time
- **InfluxDB** menyediakan penyimpanan historis yang optimal
- **Redis** menyediakan cache latensi rendah dan distribusi pesan
- **Command Pattern** di folder `spec` memastikan modularitas dan skalabilitas

Pola desain ini memastikan pustaka tidak hanya fungsional, tetapi juga mudah dipelihara oleh tim pengembang dan dapat diperluas untuk mendukung sensor data baru tanpa memodifikasi logika inti.
