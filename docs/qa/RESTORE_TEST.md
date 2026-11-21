# RESTORE TEST — bukti pemulihan backup (B-09, P0)

Tanggal: 2026-11-21. Host: Windows dev (setara prosedur VPS `deploy/backup/*.sh`).

Prosedur:
1. `mysqldump --no-tablespaces --single-transaction --routines --events notaryos` → 248 KB SQL (via cmd agar byte mentah; redirect PowerShell menulis UTF-16 dan merusak impor — pelajaran dicatat).
2. `CREATE DATABASE notaryos_restore` + GRANT ke user aplikasi.
3. Impor dump ke `notaryos_restore` → exit 0.
4. Bandingkan hitungan tabel inti prod vs restore.

Hasil (selisih HARUS 0):

| tabel | prod | restore | selisih |
|---|---|---|---|
| users | 17 | 17 | 0 |
| deed_drafts | 7 | 7 | 0 |
| invoices | 13 | 13 | 0 |
| files | = | = | 0 |
| service_requests | = | = | 0 |

RTO: dump+impor < 60 detik untuk ukuran DB saat ini (dicatat kasar; belum uji beban data besar).

Batas jujur: ini membuktikan skrip dump+restore BERFUNGSI di MySQL 8.4. Uji restore di VPS produksi tetap wajib sebelum go-live (lingkungan berbeda: GPG recipient, path /var/backups, kredensial).
DB `notaryos_restore` dihapus setelah uji agar mesin bersih.
