# Security Audit 4.0 — RBAC, Office-Scope, PII, Upload, Session, Backup

Metode: BACA SAJA terhadap `internal/app/*.go`, `internal/middleware/*`,
`internal/rbac/*`, `internal/files/*`. Skala status FASE 1C.

## 1. RBAC 9 peran x endpoint sensitif — PARTIALLY_IMPLEMENTED

Matriks `rolePerms` (9 peran, 40+ permission) benar dan diuji
(`rbac_test.go`). Penegakan dilakukan per-handler via `rbac.Has`
(26 titik di `app.go`, 60+ di `ext.go`, helper `requireStaff/requireWeb`).
`RequirePerm` middleware ADA tapi TIDAK PERNAH dipakai di route
(0 pemakaian) — penegakan murni inline, rawan lupa.

| Endpoint | Guard terverifikasi | Status |
|---|---|---|
| `deedApprove/Sign/Archive/Handover`, `willHandover/Open` | `PermApproveDeed` + mandat pengganti | IMPLEMENTED_AND_VERIFIED |
| `invoiceVoid/paymentCreate/quotationConvert/bankMatch` | `PermManageFinance` | IMPLEMENTED_AND_VERIFIED |
| `userCreate/userLink/rolesPage` | `PermManageUsers` | IMPLEMENTED_AND_VERIFIED |
| `officeAct/officePlanSave` | `RoleSuperAdmin` only | IMPLEMENTED_AND_VERIFIED |
| `govAdvance`/`ppatAdvance`/`legalisasiAdvance` | `PermGovSubmit/Legalize/ManageRepert` | IMPLEMENTED_AND_VERIFIED |
| `dmsVerify` | `PermVerifyIdentity` + office-check | IMPLEMENTED_AND_VERIFIED |
| `backupNow/backupPurgePlaintext/remindersRun/officeSave/licenseSave` | `PermManageOffice` | IMPLEMENTED_AND_VERIFIED |
| `POST /tasks` (`taskCreate`, `ext.go:2290`) | `requireWrite` (tolak auditor+klien) sejak v0.5.12 | IMPLEMENTED_AND_VERIFIED (HIGH→tertutup) |
| `POST /tasks/{id}/done` (`taskDone`) | `requireWrite` sejak v0.5.12 | IMPLEMENTED_AND_VERIFIED (MEDIUM→tertutup) |
| `protocolStatus` (tulis status protokol) | `PermViewProtocol` + tolak `ReadOnly` (auditor) sejak v0.5.12 | IMPLEMENTED_AND_VERIFIED (MEDIUM→tertutup) |
| `retentionTask` (tulis task retensi) | `PermViewClient` + tolak `ReadOnly` sejak v0.5.12 | IMPLEMENTED_AND_VERIFIED (MEDIUM→tertutup) |
| `dmsBulk verify/approve`, `dmsMerge`, `dmsRestoreVersion` | `requireWrite`; jalur `verify` wajib `PermVerifyIdentity` sejak v0.5.12 | IMPLEMENTED_AND_VERIFIED (MEDIUM→tertutup) |
| `cmsVersionRollback` | verifikasi ulang: `requireWebAny` hanya pintu masuk; di dalam wajib `PermWebPagesUpdate`/`PermWebArticles` — sudah ketat, bukan temuan | IMPLEMENTED_AND_VERIFIED |

## 2. Office-scope query — IMPLEMENTED_AND_VERIFIED (klaim "24 query" valid)

Pola baku `SELECT office_id ... WHERE id=?` → bandingkan `officeOf(r)` →
`http.NotFound` dipakai konsisten (`deedApprove/Sign/Archive`,
`ppatAdvance`, `dmsVerify/SaveMeta/Detail`, `govAdvance`, dst.).
Sampel 150+ query sensitif: tidak ada yang bocor lintas-kantor.
Catatan defense-in-depth (LOW): pola check-then-act — `UPDATE ... WHERE id=?`
tanpa predikat `office_id` (mis. `ext.go:1156`, `dms.go:445`). Aman hari ini
karena `office_id` baris immutable, tapi tambahkan `AND office_id=?` pada
tulis agar fail-closed atomik.

## 3. PII masking — PARTIALLY_IMPLEMENTED

- `maskNIKDisp`/`redactNIK` + `maskNIK` di template ADA dan diuji
  (`helpers_test.go`); `clients_detail` tampilkan `???? {{.NIKLast4}}`;
  `clientExport` mask NIK (`????+last4`); input NIK terenkripsi.
  → IMPLEMENTED_AND_VERIFIED untuk NIK.
- Helper `maskNIK` baru dipakai di `dms.gohtml` (3 titik); field sensitif
  lain (telepon/WA, alamat, NPWP penuh) tampil mentah di halaman detail.
  → NEEDS_LEGAL_REVIEW (UU PDP) untuk perluasan masking + audit log akses.

## 4. Upload validation — IMPLEMENTED_AND_VERIFIED

- `files.SaveUpload`: allowlist MIME (JPG/PNG/WebP/PDF) + sniff magic-byte
  anti-spoof + batas 100MB + sha256 + storage privat.
- `mediaUpload`: validasi `cms.ValidateMedia` + re-encode raster (strip
  metadata/skrip) + batas 10MB. `bankUpload`: CSV di-parse ketat per baris.
- Celah: `malware.Scanner` (ClamAV) hanya dipakai manual via
  `POST /documents/{id}/scan` (`dms.go:529`) — TIDAK dirangkai ke jalur
  upload → auto-scan: NOT_CONNECTED (MEDIUM).
- TERTUTUP v0.5.13: `fileUpload` menjalankan auto-scan best-effort bila
  `CLAMAV_ADDR` diset (cap 64MB, timeout 5 dtk, fail-open); TERINFEKSI →
  audit `file.upload.infected` + flash karantina manual; tak ada hapus
  otomatis. Tanpa `CLAMAV_ADDR` tak ada klaim "bersih".

## 5. Rate limit — IMPLEMENTED_AND_VERIFIED

Login 20/mnt + lockout 5x gagal/15 mnt (`login_attempts`) + notifikasi
"Login dari IP baru"; setup 10/mnt; upload 30/mnt; OCR 20/mnt; media 20/mnt;
publik konsultasi/booking 10/mnt; verify/share 30/mnt. Login 20/mnt longgar
tetapi lockout akun mengkompensasi (LOW).

## 6. Session / CSRF — IMPLEMENTED_AND_VERIFIED

- Cookie sesi: `HttpOnly`, `SameSite=Lax` (sesi) / `Strict` (CSRF+MFA),
  `Secure` ikut config TLS, expiry 12 jam, MFA cookie 5 mnt.
- CSRF double-submit per-sesi di semua grup auth; `/login` exempt
  by-design; form publik anonim tanpa CSRF (diterima, rate-limited).
- CSP `default-src 'self'` + `nosniff` + `SAMEORIGIN` + HSTS aktif (LOW:
  `script-src 'unsafe-inline'` masih diizinkan untuk Tabler/HTMX).

## 7. Backup encryption — IMPLEMENTED_AND_VERIFIED

- `backupNow`: MENOLAK berjalan tanpa `gpg` + `GPG_RECIPIENT`
  (tidak ada dump plaintext di disk; streaming `mysqldump|gzip|gpg`).
- `countLegacyPlaintextBackups` + `backupPurgePlaintext` (guarded) untuk
  membersihkan `backup-*.sql` lama. Sisa: jadwal rotasi off-site di luar
  kode (ops, LOW).

## 8. Ringkasan temuan

| # | Temuan | Severity | Status |
|---|---|---|---|
| 1 | `POST /tasks` tanpa guard peran | HIGH | TERTUTUP v0.5.12 (`requireWrite`) |
| 2 | `taskDone`/tulis retensi/protokol/rollback/CMS guard terlalu luas | MEDIUM | TERTUTUP v0.5.12 (rollback terbukti sudah ketat) |
| 3 | `dmsBulk verify` lewati `PermVerifyIdentity` | MEDIUM | TERTUTUP v0.5.12 |
| 4 | Upload tanpa auto malware-scan | MEDIUM | TERTUTUP v0.5.13 (best-effort, fail-open) |
| 5 | Masking PII belum menyeluruh (UU PDP) | MEDIUM | NEEDS_LEGAL_REVIEW |
| 6 | `RequirePerm` tak dipakai; UPDATE tanpa `AND office_id=?` | LOW | IMPLEMENTED_BUT_BROKEN |
| 7 | CSP `unsafe-inline`; login 20/mnt; stub `clients.gohtml` | LOW | IMPLEMENTED_AND_VERIFIED |
