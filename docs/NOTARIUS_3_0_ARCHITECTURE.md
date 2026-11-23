# NOTARIUS 3.0 — Arsitektur Aktual (hasil audit, Tahap 4-B)

> Status: deskripsi keadaan kode SEKARANG, bukan target. Angka dari audit repo.
> Koreksi jujur: `internal/` berisi **38** direktori paket (bukan 37);
> route unik terdaftar **±282 path** (±397 registrasi method, karena satu path
> bisa punya GET+POST); tabel migrasi **±90 CREATE TABLE unik**.

## 1. Bentuk sistem

Modular monolit Go (`go.mod` module `notaryos`, package `main` di `cmd/`).
HTTP: `chi` router, satu `App` struct (`internal/app/app.go:128`) wiring
DB + stores + sessions. Template server-side `*.gohtml`. RBAC per-handler
(`internal/rbac/rbac.go`: 9 role, ~40 permission). Audit trail sinkron
(`audit.Record`) di tiap mutasi penting.

## 2. Tabel peran 38 paket internal (1 baris per paket)

| # | Paket | Peran aktual |
|---|-------|--------------|
| 1 | app | Router chi + semua handler HTTP + render (app.go:441 `Router`) |
| 2 | audit | Struct Entry + Record audit trail |
| 3 | auth | Hash/verify password, sesi login |
| 4 | clients | Store clients/identities/notes/merge |
| 5 | cms | Store konten website (pages/articles/services/faq) |
| 6 | config | Load konfigurasi kantor/env |
| 7 | corp | Data korporasi: company/officer/shareholder/BO |
| 8 | cryptoxx | Util kripto internal (segel/hash bantu) |
| 9 | database | Open/migrasi DB MySQL/SQLite |
| 10 | deeds | Store deed_drafts/versions/signatures |
| 11 | deliver | Pengiriman/notifikasi keluar (terbatas) |
| 12 | diff | Diff versi dokumen (DMS compare) |
| 13 | dms | Logika DMS: versi, chain, retensi, hold |
| 14 | docengine | Render/rakit dokumen dari template |
| 15 | doctpl | Katalog template standar (seed konsep) |
| 16 | docx | Ekspor DOCX akta |
| 17 | files | Metadata + simpan file upload |
| 18 | finance | Store invoices/payments/bank/budget |
| 19 | gov | Mesin status gov_submissions (AHU/OSS asistensi) |
| 20 | malware | Scan upload dasar (klaim terbatas, bukan AV penuh) |
| 21 | mfa | TOTP setup/verify |
| 22 | middleware | RoleOf, rate-limiter, CSRF, readOnly |
| 23 | money | Format/hitung rupiah, tarif |
| 24 | notify | Notifikasi in-app per user |
| 25 | ocr | Antrean + panggil OCR (butuh binary eksternal) |
| 26 | officex | Profil kantor, lisensi, backup |
| 27 | pdf | Ekspor PDF akta/dokumen |
| 28 | plan | Paket/langganan kantor (office plan) |
| 29 | rbac | 9 role + Has(role,perm), dipakai tiap handler |
| 30 | remind | Scheduler pengingat/SLA (dipicu manual `/reminders/run`) |
| 31 | requests | Store + mesin transisi service_requests (requests.go:11) |
| 32 | safehtml | Sanitasi HTML template CMS |
| 33 | storage | Abstraksi simpan file (lokal) |
| 34 | users | Store users/roles/link klien |
| 35 | validate | Validasi form (required, NIK/NPWP dasar) |
| 36 | verify | Kode verifikasi publik `/verify`, segel QR |
| 37 | wib | Util tanggal/zona Asia/Jakarta |
| 38 | workers | Pool background (notifikasi best-effort) |

## 3. Lapisan Matter & workflow (026)

- `Matter` = baris `service_requests` (+ `request_events` sebagai jurnal).
  TIDAK ada tabel `matters` terpisah — istilah Matter adalah pola pakai.
- `MatterParty` = `matter_parties` (026, FK ke requests+clients, unik
  request+client+role). Handler `partyAdd/partyRemove` (ext.go:792,823).
- Aturan per layanan = `service_workflow_rules` (026, PK = service_code,
  21 seed) dibaca `workflowGate` (wf_rules.go:113). Tanpa baris rule → default
  sedang (checklist bila template ada + identitas VERIFIED + min_parties=1).

## 4. Route (±282 path unik)

Publik (`/`, `/layanan`, `/docs`, `/verify`, `/s/{token}`, CMS publik) +
terotentikasi grup `p` (`/app`, `/clients`, `/requests`, `/deeds`, `/wills`,
`/repertorium`, `/ppat`, `/legalisasi`, `/protocols`, `/gov`, `/finance`,
`/templates`, `/cms/*`, `/dms/*`, `/documents/*`, `/audit`, `/office`,
`/tasks`, `/portal/*`). Pendaftaran tersebar: app.go (~304) + ext.go (29) +
dms.go (18) + dms_enterprise.go (10) + cms*.go (~22) + lain-lain.

## 5. Register terpisah & portal

- Kenotariatan: `repertorium` (otomatis saat pengesahan, + deteksi nomor
  lompat `repertoriumGaps` app.go:2110, ekspor CSV).
- PPAT: `ppat_register` (status sendiri, resi BPN wajib ke LAPOR_BPN).
  Keduanya TIDAK berbagi sequence.
- `protocols`, `legalizations`, `wills` (register_no unik per kantor, 026)
  masing-masing register sendiri.
- Portal klien: read-only (timeline, dokumen, invoice, resi) + 2 pengecualian
  tulis: `POST /portal/pay` (konfirmasi bayar) dan `POST /notifications/read`.

## 6. Jujur: generik vs per-layanan

- GENERIK (berlaku semua): RBAC, audit trail, office-scope, gate checklist +
  identitas + min_parties, mesin transisi requests, portal read-only.
- PER LAYANAN (hanya bila rule ada): `require_deed_kind` (mis. PPAT-AJB wajib
  AJB), `require_bpn_receipt=1` (hanya PPAT-AJB/HIBAH/TUKAR/APHB +
  BUNDEL-AJB-BALIKNAMA), `min_parties=2` (AJB, SKMHT, APHT, KUASA, PRENUP…).
- BELUM ADA: SLA per layanan di engine, tarif otomatis per layanan, validasi
  BPN/AHU online (asistensi manual), tanda tangan elektronik tersertifikasi
  (hash internal saja).
