# SALES_READINESS — SaktiNotaris v0.5.15 (2026-11-24)

Status: SIAP JUAL BETA / PILOT. Belum siap klaim "produksi penuh" sampai 4 penahan di bawah ditutup.

## 1. Yang baru dibereskan (delta v0.5.11 -> siap jual)

- `Makefile migrate`: dilengkapi 001-034 (sebelumnya berhenti di 020).
  Boot aplikasi tetap migrasi otomatis via `schema_migrations`; target make hanya untuk DB kosong/manual.
- `project/progress.json|features.json|roadmap.json`: disinkron ke `0.5.13`.
- `cmd/notaryos/main.go`: warning saat boot bila `COOKIE_SECURE=false` atau `APP_BASE_URL` kosong.
- `.env.example`: anotasi produksi untuk `COOKIE_SECURE=true`.
- Dokumen ini + checklist go-live eksplisit (pengganti klaim implisit "100%").

## 1b. Delta v0.5.12 (dashboard Seven) + v0.5.13 (hardening)

- `/app` ditulis ulang gaya referensi Seven (clean-room, tanpa aset tema berbayar):
  daftar Permohonan, kartu gelap aktif, Pendapatan+donat, Sorotan jujur,
  Agenda + strip minggu, Unggulan, banner tier, grafik aktivitas, tabel
  statistik (anggaran nyata + progres checklist), promo invoice, Gantt tugas
  (tab Hari/Minggu/Bulan fungsional). Semua angka query nyata.
- Guard tulis: `rbac.ReadOnly` + `requireWrite` menutup celah tulis auditor di
  taskCreate/taskDone/protocolStatus/retentionTask/dmsBulk/dmsMerge/
  dmsRestoreVersion; bulk-verify wajib `PermVerifyIdentity` (unit hijau).
- Upload: auto-scan ClamAV best-effort bila `CLAMAV_ADDR` diset (cap 64MB,
  fail-open; TERINFEKSI → audit + flash karantina, tanpa hapus otomatis).
- v0.5.14: badge tugas OPEN di sidebar (`PendingTasks`, fail-open).
- v0.5.15: tab aktif diingat per halaman; 7 predikat `AND office_id=?` atomik pada tulis sensitif.

## 2. Tier jualan (sudah di-enforce di kode `internal/plan`)

| Tier | Fitur kunci | Gate |
|---|---|---|
| LITE | Operasional inti: permohonan, akta, invoice, DMS dasar | default fail-closed |
| PRO | + portal bayar (`portal.pay`), share berjangka (`doc.shares`), checklist (`doc.checklist`) | 402 bila tier kurang |
| ENTERPRISE | + lintas-kantor acting (`office.acting`), prioritas | 402 bila tier kurang |

Harga: JANGAN hardcode di kode. Tarif jasa notaris bervariasi per akta — pakai `/tariffs` + `/quotations` (penawaran resmi -> invoice PPN 11%) + kalkulator `/kalkulator` (label "bukan harga final").

## 3. Empat penahan go-live (wajib sebelum data sungguhan)

### B-06 Tinjauan notaris (HIGH, hukum)
- Minta 1 notaris/PPAT review: alur IN_REVIEW->APPROVED->IN_PROGRESS->SIGNED->ARCHIVED, wasiat, retensi, PPN, nomor repertorium.
- Template berita acara: `docs/legal/NOTARY_PRACTICE_VALIDATION.md` + `docs/compliance/35_HUMAN_LEGAL_REVIEW_CHECKLIST.md`.
- Output: tanda tangan + tanggal. Tanpa ini DILARANG produksi (aturan repo sejak v0.3.0).

### B-08 TLS + COOKIE_SECURE (HIGH, keamanan)
1. `certbot --nginx` untuk domain, 2. `COOKIE_SECURE=true` di `/etc/notaryos/env`,
3. Verifikasi: `curl -sI https://DOMAIN/manifest.webmanifest` 200 + MIME `application/manifest+json`;
   Chrome DevTools Manifest/SW hijau. Boot log tidak boleh ada warning COOKIE_SECURE.

### B-09 Uji restore di VPS target (HIGH, operasional)
1. `deploy/backup/backup.sh` (DB .sql.gz.gpg) + `deploy/backup/backup_files.sh` (berkas storage).
2. Restore ke DB kosong: `restore.sh <file> [db_restore]` -> `gpg --decrypt | gunzip | mysql`.
3. Verifikasi: `diff` file storage src vs dest + spot-check checksum + `/readyz` 200 + hitung `schema_migrations` memuat `034`.
4. Catat RTO. Target repo sebelumnya: RTO <60 dtk di lab (belum di VPS pembeli).

### Pen-test minimal (MEDIUM, sebelum serah terima)
- Login brute-force 6x salah -> kunci 15 mnt; CSRF semua POST; XSS di RTE (`safehtml`);
- IDOR office-scope (coba akses deed/invoice kantor lain); upload malware (tombol Pindai ClamAV bila `CLAMAV_ADDR` diset);
- `read-only` mode `READ_ONLY=1` untuk darurat.

## 4. Checklist serah terima per kantor (10 menit)

1. Backup DB + BERKAS dulu.
2. Deploy biner: `go build -o bin/notaryos ./cmd/notaryos` + `systemctl restart notaryos`.
3. Cek `journalctl -u notaryos | grep -i migrat` + `/readyz` 200 + `SELECT version FROM schema_migrations` memuat `034`.
4. Login staf -> `/dms/checklists` -> Pasang Standar (seed 23 paket + 3 template workflow PT/AJB/Kuasa).
5. `/templates` -> Pasang 8 Konsep -> Approve tiap template (DRAFT->APPROVED).
6. Uji 1 permohonan end-to-end: buat -> draf 1-klik -> unggah -> verifikasi 2-kolom -> invoice DP -> portal timeline -> akta disahkan (rantai IN_REVIEW->APPROVED->IN_PROGRESS) -> serah -> SIGNED -> ARCHIVED.
7. Uji penawaran: buat quotation -> SENT -> konversi -> invoice ISSUED (PPN 11%); verifikasi tolak konversi bila EXPIRED.
8. Verifikasi PWA: `/offline` langsung 200, navigasi saat offline jatuh ke `/offline`, halaman privat tak di-cache.
9. Pasang timer pengingat 07:00 WIB (DEPLOYMENT_GUIDE langkah 10).
10. Serahkan USER_MANUAL + kredensial akun layanan reminder (`/etc/notaryos/reminder.env` 0600).

## 5. Batasan jujur yang harus tertulis di kontrak

- OCR butuh `tesseract-ocr + tesseract-ocr-ind` di host (`OCR_BINARY`), tanpa itu fallback Noop jujur (tak ada hasil palsu).
- WA H-1 hanya bila gateway dikonfigurasi (`NOTARYOS_NOTIFY_WEBHOOK`); default in-app + tombol WA manual.
- AI eksternal nonaktif by design tanpa kunci (`AI_BASE_URL` kosong).
- AHU/OSS: asistensi checklist manual, bukan API resmi.
- TTE di repo ini = segel HMAC internal + `/verify`, BUKAN TTE tersertifikasi BSrE/kominfo.
- Wasiat dikecualikan dari portal publik; akses willReader/notaris/auditor saja.
- Harga di landing/kalkulator bukan harga final; harga resmi via penawaran + invoice.

## 6. SLA yang disarankan (isi di kontrak, bukan klaim kode)

- Ketersediaan 99% bulanan (di luar maintenance terjadwal); RTO <= 1 jam, RPO <= 24 jam (dengan backup harian GPG + berkas).
- Dukungan: bug kritis (data bocor/transaksi gagal) respon 1x24 jam; bug sedang 3x24 jam.
- Maintenance: patch keamanan tanpa downtime terjadwal; migrasi mayor dengan backup + jendela malam.
- Pengecualian: TLS domain milik pembeli, hardware/VPS pembeli, data palsu yang di-seed pembeli.
