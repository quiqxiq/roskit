Gap Analysis: PROMPT.MD vs mikhmon-api
Tabel Ringkasan
Kategori	Total di PHP (PROMPT.MD)	Sudah di Go	Gap
RouterOS API Commands	47	38	9
RouterOS Scripts (On-Login, Expire Monitor, dll)	5	3	2
Format/Data Parsing	9	6	3
Encryption Schemes	3 (2 wajib)	0	2
Tool Methods (CRUD)	~15	~11	4
---
1. RouterOS API Commands yang BELUM ada di Go
Command	Lokasi PHP	Status Go
/ip/hotspot/user/remove	hotspot/removehotspotuser.php, process/remove*.php	MISSING — hanya ada RemoveActiveSession
/ip/hotspot/cookie/print	hotspot/cookies.php	MISSING
/ip/hotspot/cookie/remove	process/removecookie.php, process/removeuseractive.php	MISSING
/ip/hotspot/ip-binding/print	hotspot/ipbinding.php	MISSING
/ip/hotspot/ip-binding/set	process/pipbinding.php	MISSING
/ip/hotspot/ip-binding/remove	process/pipbinding.php	MISSING
/ip/arp/print	process/pipbinding.php	MISSING
/ip/arp/remove	process/pipbinding.php	MISSING
/ip/dhcp-server/lease/remove	process/pipbinding.php	MISSING — ada stream spec tapi tidak ada remove
/system/shutdown	process/shutdown.php	MISSING
/system/reboot	process/reboot.php	MISSING
/ip/hotspot/user/profile/remove	process/removeuserprofile.php	MISSING — bisa lewat Executor.Remove tapi tidak ada domain wrapper
Catatan: /ip/firewall/nat/print ada di Go tapi TIDAK ada di PHP — ini fitur baru, bukan gap.
---
2. Encryption Schemes — BELUM SAMA SEKALI
Scheme	Algoritma	Status Go	Kritis
enc_rypt/dec_rypt	XOR key "128" + base64	MISSING	YA — migrasi password router
blah()/unblah()	XOR 10 + double base64	MISSING	YA — credential transport
jsEncode	XOR 25	Sengaja skip	Tidak perlu (gunakan HTTPS)
Lokasi di PHP: lib/routeros_api.class.php:440-459
---
3. Tool Methods yang BELUM ada
Method	RouterOS Command	PHP Equivalent
RemoveHotspotUser	/ip/hotspot/user/remove	hotspot/removehotspotuser.php
RemoveHotspotProfile	/ip/hotspot/user/profile/remove	process/removeuserprofile.php
DisableHotspotUser	/ip/hotspot/user/set disabled=yes	process/disablehotspotuser.php
EnableHotspotUser	/ip/hotspot/user/set disabled=no	process/enablehotspotuser.php
RemoveCookies	/ip/hotspot/cookie/remove	process/removecookie.php
IPBindingCRUD	/ip/hotspot/ip-binding/*	hotspot/ipbinding.php, process/pipbinding.php
ShutdownRouter	/system/shutdown	process/shutdown.php
RebootRouter	/system/reboot	process/reboot.php
Note: Executor.Enable/Disable/Remove/Add/Set sudah generik — tinggal buat domain wrapper.
---
4. Domain/Spec yang BELUM lengkap
Item	Keterangan
HotspotServer	Tidak punya ToTags()/ToFields()/ToCacheData() dan tidak ada StreamSpec
SystemClock, SystemIdentity, SystemRouterboard, SystemHealth	Tidak punya ToTags()/ToFields()/ToCacheData() — hanya data structs
Hotspot Cookie	Tidak ada domain/hotspot_cookie.go dan tidak ada spec
Hotspot IP Binding	Tidak ada domain/ip_binding.go dan tidak ada spec
Quick Print script storage	Tidak ada — PHP simpan config paket di /system/script dengan comment=QuickPrintMikhmon
---
5. Format/Data Parsing yang BELUM ada
Format	Keterangan	Status
Router Config Delimiter (!, `@	@, #	#`, dll)
Voucher Temp File (|~ + ! delimiter)	PHP simpan state generasi terakhir	MISSING — perlu equivalent (Redis?)
Quick Print Source Format (# delimiter)	#name#server#up#vc#...#price_sprice#lock	MISSING — belum ada sama sekali
---
6. Script/Logic yang SUDAH lengkap
Item	File Go	Catatan
On-Login script generation (5 mode)	tool/script.go	Lengkap, termasuk MAC lock + server lock
Expire Monitor script	tool/scheduler.go	Lengkap, single scheduler per router
:put metadata parsing	tool/script.go	Lengkap, 8 field
User comment parsing (expiry/voucher/plain)	tool/script.go	Lengkap, posisi-aware
Sales record format (`-	-` delimiter)	domain/sales_record.go
Voucher generation (10 charset modes)	tool/voucher.go	Lengkap + batch dedup
Hotspot user count off-by-one	spec/hotspot_user.go	count - 1
Logging setup (prefix="->")	tool/system.go	Lengkap