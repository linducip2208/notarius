# DECISIONS
| ID | Tanggal | Keputusan | Alasan |
|----|---------|-----------|--------|
| D-01 | 2026-11-20 | UI **Tabler** via CDN, bukan Metronic (§A.5 diganti) | Instruksi eksplisit user "pakai tabler"; Tabler MIT, tanpa masalah lisensi vendor |
| D-02 | 2026-11-20 | chi + database/sql + go-sql-driver (tanpa ORM) | Kontrol SQL penuh untuk audit & FK; dependensi minimal |
| D-03 | 2026-11-20 | Rupiah int64 di kode, DECIMAL di DB | Presisi moneter; format hanya presentasi |
| D-04 | 2026-11-20 | UTC di DB, WIB di presentasi | Konsistensi lintas zona |
| D-05 | 2026-11-20 | OCR Noop jujur, bukan klaim palsu | §M: tidak boleh mengklaim OCR tanpa test nyata |
| D-06 | 2026-11-20 | AHU/OSS asistensi manual, tanpa endpoint liar | §N: tidak ada API resmi yang di-hardcode |
| D-07 | 2026-11-20 | Sesi in-memory dulu (TD-01) | Kecepatan v1; batas single-instance dicatat |
| D-08 | 2026-11-20 | DSN `loc=UTC`, bukan `time_zone` | Laragon MySQL tanpa tabel zona waktu; parsing UTC sisi driver |
| D-09 | 2026-11-20 | Rate-limit keyed by IP murni | RemoteAddr memuat port unik per koneksi — limiter lama tak pernah срабатывать |
| D-10 | 2026-11-20 | Scan DECIMAL sebagai string + `deci()` | Driver MySQL mengembalikan DECIMAL sebagai bytes; scan int64 gagal |
| D-11 | 2026-11-20 | Akta APPROVED ikut terkunci | Nomor akta yang sudah terbit tak boleh diubah isinya |
| D-12 | 2026-11-20 | PDF ekspor stdlib minimal | Tanpa dependensi baru; cukup untuk draf bertanda DRAFT |
