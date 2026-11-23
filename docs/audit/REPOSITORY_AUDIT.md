# REPOSITORY AUDIT 4.0 — FASE 1A (BACA saja)

Tanggal: 2026-11-24. Repo: `D:\project laravel\notaris`. Metode: baca struktur + sampling handler (apakah benar baca/tulis DB), tanpa eksekusi.

## 1. Stack AKTUAL (tegaskan)

- **Go 1.26.2 modular monolith (`notaryos`), MySQL (`go-sql-driver`), server-render `html/template` + Tabler (`web/assets/vendor/tabler.min.css/js`) + htmx.** BUKAN Laravel/PHP, BUKAN Metronic (tidak ada aset Metronic di `web/assets/vendor/`), BUKAN REST API (tidak ada JSON API publik; satu-satunya JSON internal: sitemap/readyz/healthz; semua alur via form POST + redirect + template `.gohtml`).
- Router: `go-chi/chi/v5` di `internal/app/app.go` (`Router()`), middleware sendiri (`internal/middleware`: session, CSRF, rate-limit, security headers). QR: `skip2/go-qrcode`. Kripto: `golang.org/x/crypto` + `internal/cryptoxx`.
- Deploy: `Makefile`, `deploy/` (systemd?), `cmd/notaryos/main.go`. PWA: `sw.js` + manifest.

## 2. Struktur folder

```
cmd/notaryos/      entrypoint biner
internal/app/      wiring + SEMUA handler HTTP + templates/*.gohtml (28 file .go, 27 .gohtml)
internal/<38 lib>/ paket domain/infra (lihat §3)
migrations/        001–031 *.sql (31 file; 29 skema efektif — 005_seed + 014_merge non-skema)
web/assets/        vendor Tabler/htmx + brand + landing + docs + notarius/ui.js,tokens.css
docs/audit/        audit-audit lama (arsip, bukan acuan fase 1A)
tests/, scripts/, storage/, project/, bin/, deploy/
```

## 3. 38 paket internal (selain `app`), 1 baris each

1. `audit` — pencatat jejak audit DB. 2. `auth` — login/session/lockout + sqlstore. 3. `clients` — CRUD klien + identitas + CRM. 4. `cms` — artikel/halaman/media + sanitasi HTML. 5. `config` — loader konfigurasi env. 6. `corp` — data korporat + legal AHU/OSS. 7. `cryptoxx` — util kripto/HMAC. 8. `database` — koneksi + runner migrasi. 9. `deeds` — minuta akta, legalisasi, wasiat, protokol. 10. `deliver` — pengiriman WA-link/logistik. 11. `diff` — diff versi dokumen. 12. `dms` — file, versi, workflow, OCR queue, share, arsip fisik, hold, checklist. 13. `docengine` — mesin render dokumen. 14. `doctpl` — template dokumen. 15. `docx` — parse/uji DOCX. 16. `files` — helper file. 17. `finance` — invoice, payment, quotation. 18. `gov` — state machine submisi layanan pemerintah (simulasi, BUKAN API AHU/OSS). 19. `malware` — pemindai upload. 20. `mfa` — TOTP MFA. 21. `middleware` — auth/session/CSRF/rate-limit/header. 22. `money` — integer sen + format rupiah. 23. `notify` — notifikasi. 24. `ocr` — KTP/KK/NPWP parse + tesseract opsional. 25. `officex` — berkas office/ekspor. 26. `pdf` — util PDF. 27. `plan` — paket langganan kantor. 28. `rbac` — 9 peran + matriks izin. 29. `remind` — SLA/reminder/digest. 30. `requests` — service_requests + janji + status event. 31. `safehtml` — sanitasi template. 32. `storage` — abstraksi S3/lokal. 33. `users` — pengguna + MFA + timezone. 34. `validate` — validasi input. 35. `verify` — kode verifikasi HMAC + QR. 36. `wib` — zona waktu WIB. 37. `workers` — pool worker generik. 38. `workflow` — engine workflow instans/aturan.

## 4. Inventaris klaim vs verifikasi aktual

| Klaim brief | Hasil cek aktual (jujur) |
|---|---|
| 277 route | `app.go` ≈336 registrasi `.Get/.Post` (+`ext.go` ~45, cms/dms subrouter); klaim 277 plausibel sbg path unik sesudah dedup GET+POST per path. Terima dengan catatan. |
| 127 template | 27 file `internal/app/templates/*.gohtml` + 127 file `.go` di `internal/` — angka 127 = file Go, BUKAN template. Template aktual: 27. |
| 29 migrasi | 31 file `001–031`; efektif 29 skema (005 seed, 014 merge data). Klaim 29 benar utk skema. |
| 38 paket | 39 direktori inkl. `app` → 38 lib + `app`. Benar. |

## 5. Temuan arsitektur kunci

1. Semua state via MySQL, `office_id` scoping manual per query — aman bila konsisten, rawan bocor bila satu handler lupa filter (perlu audit `office_id` menyeluruh, P0).
2. TTE hanya label UI ("siap TTE"); tidak ada integrasi BSrE/esign — JANGAN klaim sah elektronik.
3. `gov` = state machine lokal, bukan integrasi AHU/OSS/Agraria — JANGAN klaim pengajuan resmi.
4. Upload → `malware` scan + OCR queue in-DB (`ClaimOCR` in-memory? klaim race multi-instans) — butuh verifikasi konkurensi, P1.
5. Multi-kantor + 9 peran di satu skema; `role_permissions` (010) + matriks kode — konsistensi keduanya belum terverifikasi, P1.
