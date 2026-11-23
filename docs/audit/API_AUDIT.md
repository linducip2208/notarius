# API AUDIT 4.0 — Fase 1B (server-render, BUKAN REST)

**Temuan arsitektur (jujur): repo ini TIDAK punya REST API publik.** Tidak ada
route `/api/*`, tidak ada respons JSON kontraktual (grep `"/api` di
`internal/app/app.go` = 0 hit). Semua interaksi lewat chi + HTML form + redirect
(`internal/app/app.go` `Router()`, ±230 route). Maka audit di bawah mencakup
sebagai pengganti: SEMUA endpoint POST/GET yang mengubah data.
Legenda status memakai kosakata fase yang sama.

## 1. Fondasi guard (IMPLEMENTED_AND_VERIFIED, dengan catatan)
- Grup privat: `RequireAuth` + `CSRF` global; aset/publik di luar grup.
- Rate-limit: login 20/menit + lockout 5x/15 mnt; setup 10/mnt sekali-pakai;
  upload 30/mnt; OCR 20/mnt; konsultasi/booking publik + limiter; shared
  download `/s/{token}` + verifyLimiter. Security headers (CSP/HSTS/nosniff).
- RBAC: `mw.RequirePerm` ADA tapi **tidak dipakai di Router**; penegakan via
  `rbac.Has(...)` inline per handler (contoh `clientCreate` cek
  `PermCreateClient`). → PARTIALLY_IMPLEMENTED: bekerja bila tiap handler ingat
  cek; satu handler lupa = lubang. Rekomendasi: bungkus grup route per domain
  dengan `RequirePerm`.
- Office-scope: `officeOf(r)` + klausa `AND office_id` (±116 titik, 804 hit
  guard kasar) — IMPLEMENTED_AND_VERIFIED untuk handler sampel; cakupan penuh
  belum dibuktikan per-handler.

## 2. Auth & sesi (IMPLEMENTED_AND_VERIFIED)
| Endpoint | Handler | Validasi | RBAC | Scope | Audit |
|---|---|---|---|---|---|
| POST /login | login | email+password, lockout | publik+limit | n/a | login/failed/locked |
| POST /mfa/verify | mfaVerify | token pending | auth | user | audit login |
| POST /setup | setupSubmit | sekali-pakai | publik+limit+CSRF | n/a | — |
| POST /logout | logout | — | auth | — | logout |
| POST /offices/act | officeAct | token acting | auth | office | — |

## 3. Klien & matter (PARTIALLY_IMPLEMENTED → per-baris di bawah)
| Endpoint | Handler | Validasi | RBAC | Scope | Audit |
|---|---|---|---|---|---|
| POST /clients, /clients/new | clientCreate/NewSubmit | consent=1 wajib; field lain mentah | PermCreateClient | officeOf | client.create |
| POST /clients/{id} | clientUpdate | UpdateFull | edit? (cek inline, belum verifikasi) | office | client.update |
| POST .../identities, .../notes, .../reps | identityAdd/noteAdd/repAdd | id_type ENUM bebas klien | inline | office | client.note |
| POST /identities/{id}/verify|status|flag | identityVerify/Status/Flag | note wajib? parsial | verify? | identity.verify.* |
| POST /clients/bulk, /merge* | clientBulk/mergeDo/Preview | butuh tinjau ulang | inline? | office | — (verifikasi) |
| POST /requests | requestCreate | service_id mentah | PermRequestService-ish | office | request.create |
| POST /requests/{id}/move | requestMove | transisi + wf gate | inline | office | request.move |
| POST .../corp|legal|shareholders|officers|bo|parties | corpSave/.../boAdd/partyAdd | parseIDR/angka mentah | inline | office | sebagian |
| POST /requests/{id}/checklist, /workflow/step | dmsChecklistSet/workflowStep | status ENUM | inline | office | — (verifikasi) |
| POST /appointments, /{id}/status | appointmentCreate/Status | waktu mentah | inline | office | — |
| GET mutasi? | — | **Tidak ditemukan GET yang memutasi**; GET hanya baca/unduh (pdf/docx/receipt/package/export) — IMPLEMENTED_AND_VERIFIED |

## 4. Akta, register & legal (NEEDS_LEGAL_REVIEW)
| Endpoint | Audit log? | Catatan |
|---|---|---|
| POST /deeds, /{id}/version|review|approve|sign|archive|handover|attach | deed.draft (+lainnya?) | rantai saksi/pembaca/renvoi di DB (020); penegakan wajib-isi di `deedApprove` BELUM diverifikasi di sini |
| POST /wills, /{id}/handover|review|open | willHandover ada | pembukaan bersaksi ganda (019) — verifikasi handler `willOpen` tertunda |
| POST /protocols, /{id}/status | — (verifikasi) | status IN_STORAGE/OUT/RETURNED |
| POST /legalisasi, /{id}/advance | legalisasi.create | GROSSE ditambah tanpa migrasi data lama — legal review |
| POST /ppat, /{id}/advance | ppat.create/advance(+office) | require_bpn_receipt di seed rules; penegakan di handler BELUM diverifikasi |
| POST /gov, /{id}/advance|check|followup | — (verifikasi) | resi manual = bukti lemah; TANPA FK ke request (NOT_CONNECTED) |
| GET /verify, /qr/inv/{id} | — | verifikasi hash internal, BUKAN TTE PSrE — jujur di 013; NEEDS_LEGAL_REVIEW bila diklaim sbg TTE |

## 5. Keuangan (IMPLEMENTED_AND_VERIFIED, audit parsial)
- POST /invoices (+void), /payments, /quotations*/status|convert, /finance/bank/*,
  /budgets: invoice.create/payment.create/void tercatat + office; idem_key anti
  double-POST — IMPLEMENTED_AND_VERIFIED. paymentCreate anti-overpay di klaim
  kode; bukti HITUNG ULANG di sini belum dibaca → PARTIALLY_IMPLEMENTED.
- GET receipt/invoices/pdf = baca — aman.

## 6. DMS (IMPLEMENTED_AND_VERIFIED, audit perlu pelengkapan)
- POST /documents/{id}/meta|verify|version|restore|workflow|approve|scan|link|
  favorite|share|archive|hold|disposition|fields + /dms/bulk|ocr/run|retention|
  holds/release|duplicates/merge|chain|categories|fields + /clients/{id}/files,
  /files/{id}/ocr, /deeds/{id}/attach: upload-limit, OCR-limit, share-token.
  Audit.Record untuk dmsMove/approve/verify BELUM terbukti di sampel →
  PARTIALLY_IMPLEMENTED (tambah audit untuk workflow/approve/disposition).
- GET preview/download/version/compare/search/chain/labels = baca; akses file
  lintas-kantor dicek `canViewDeed`/office (sampel deedPDF) — verifikasi penuh
  tertunda.

## 7. CMS publik & privat (PARTIALLY_IMPLEMENTED)
- Publik TANPA auth (by design): POST /konsultasi, /booking (+limit+pubGate),
  GET kalkulator/tim/galeri/artikel/layanan/media/arsip/sitemap. Mutasi publik
  → inquiry/booking (akses tulis anonim terkontrol) — IMPLEMENTED_AND_VERIFIED
  sebagai desain; pastikan moderasi SPAM (status SPAM ada) + CAPTCHA bila abused.
- Privat /cms/* (±40 POST: branding/site/domain/media/pages/sections/articles/
  categories/services/faq/nav/history/setup/inquiries/slots/bookings/team/
  gallery/testimonials/versions-rollback): di dalam grup auth; cek
  `website.*.manage` per handler BELUM diverifikasi satu per satu →
  PARTIALLY_IMPLEMENTED. Risiko tertinggi: /cms/setup{,/sample}, /cms/export.

## 8. Admin & operasi (PARTIALLY_IMPLEMENTED)
- POST /users, /users/{id}/link, /offices/{id}/plan, /office, /office/license,
  /office/mandates*, /office/backup*, /tasks*/done, /expenses, /tariffs,
  /templates*/approve, /clauses, /mfa, /notifications/read, /reminders/run,
  /cms/inquiries/{id}/convert: userCreate/officeSave/mandateSave/backupNow
  adalah endpoint paling sensitif — audit.Record untuk ini BELUM terbukti di
  sampel → tandai TANPA AUDIT sampai dibuktikan; prioritas tambah audit.
- GET /metrics|/audit|/reports|/settings|/search|/documents: baca; /audit/export
  + /repertorium/export + /clientExport = ekspor massal — batasi role auditor/
  admin + audit setiap unduhan (client.export sudah diaudit).

## 9. Ringkasan status endpoint
- IMPLEMENTED_AND_VERIFIED: login/MFA/logout+lockout, CSRF global, rate-limit
  titik sensitif, client/request/deed/invoice/payment create+inti audit,
  share-token DMS, konsultasi/booking publik berlimit.
- PARTIALLY_IMPLEMENTED: RBAC per-route (inline, bukan middleware), validasi
  form mentah (domain `internal/validate` belum dipetakan ke tiap handler),
  audit log (inti ada; DMS/admin/gov/checklist belum terbukti), office-scope
  (sampel ok, penuh belum).
- IMPLEMENTED_BUT_BROKEN: — (tidak ada handler rusak terbukti; temuan rusak
  ada di sisi DB, lihat DATABASE_AUDIT.md §3–4).
- NOT_CONNECTED: gov_submissions ↔ request; security_incidents tanpa handler
  (0 ref); budgets/bank_mutations minim ref (7/0-hasil-campuran, verifikasi).
- NEEDS_LEGAL_REVIEW: klaim TTE (hash internal ≠ PSrE), penomoran register,
  GROSSE, pembukaan wasiat, gate BPN PPAT.
- Prioritas: (1) pasang RequirePerm per grup + audit massal untuk
  /office/*,/users,/templates/approve,/dms/*/approve|disposition,/gov/*;
  (2) daftar-validasi per handler (wajib vs opsional); (3) uji lintas-kantor
  (satu office tak boleh baca/mutasi office lain) + uji CSRF/rate-limit.
