# NOTARIUS 3.0 — Proses Bisnis 8 Tahap (pemetaan aktual)

Legenda: `handler (file:line)` → tabel. Semua mutasi staf dalam grup `p`
(terotentikasi + RBAC); publik hanya tahap 1.

## Tahap 1 — Intake (permohonan masuk)

- Publik: `pubKonsultasiSave POST /konsultasi`, `pubBookingSave POST /booking`,
  `a.inquiryToRequest POST /cms/inquiries/{id}/convert` (cms_growth_admin.go)
  → `website_inquiries`, `website_bookings`.
- Staf: `requestCreate POST /requests` (app.go) → `service_requests` status
  DRAFT; `clientCreate/clientBulk/mergeDo POST /clients*` → `clients`.
- Batas: konsultasi/booking publik tanpa login; konversi inquiry → request
  butuh staf.

## Tahap 2 — Matter (pembuka pekerjaan & pihak)

- `requestDetail GET /requests/{id}` (ext.go:415) baca Matter + checklist.
- `partyAdd POST /requests/{id}/parties`, `partyRemove .../delete`
  (ext.go:792,823) → `matter_parties` (026; master tetap `clients`).
- `identityAdd/identityVerify POST /clients/{id}/identities*` (ext.go:174,194)
  → `identities`; syarat gate: status VERIFIED.
- Korporasi: `corpSave/corpLegalSave/shareholderAdd/officerAdd/boAdd`
  (ext.go:574–752) → `companies, shareholders, company_officers,
  beneficial_owners`.

## Tahap 3 — Syarat (checklist dokumen)

- `dmsChecklistSet POST /requests/{id}/checklist` (dms.go:1178),
  `dmsChecklistSeed`, `dmsVerify POST /documents/{id}/verify` (dms.go:464)
  → `doc_checklist_records/items`, `files`.
- Gate: `workflowGate(db,office,reqID,"","move")` (app.go:1836, wf_rules.go:113):
  item selain VERIFIED/RECEIVED memblokir naik ke IN_REVIEW/APPROVED.

## Tahap 4 — Draf & review akta

- `deedCreate POST /deeds` (app.go; 1-klik dari request via template approved
  per kode layanan) → `deed_drafts` DRAFT.
- `deedSaveVersion POST /deeds/{id}/version` → `deed_versions`;
  `deedToReview` (ext.go:1122) DRAFT→IN_REVIEW (butuh deed.draft/edit).
- `deedApprove` (ext.go:1152) butuh `deed.approve` (Notaris): wajib nomor+isi;
  retry 1062 dibedakan sumbernya (tahap 3). `deedPDF/deedDOCX` → render.

## Tahap 5 — Signing (penandatanganan)

- `deedSign POST /deeds/{id}/sign` (ext.go:1246) butuh `deed.approve`;
  hanya dari APPROVED → SIGNED; hash SHA256 + seal dicatat
  `deed_signatures`. Tanda tangan KEDUA otomatis ditolak (gate status).
- Jujur: ini tanda internal (hash), BUKAN TTE tersertifikasi PSrE.

## Tahap 6 — Register (pencatatan resmi)

- Notaris: pengesahan otomatis tulis `repertorium` + cek lompat nomor
  (`repertoriumGaps` app.go:2110); `repertoriumExport` CSV.
- PPAT: `ppatAdvance POST /ppat/{id}/advance` (app.go:2367)
  DRAFT_AKTA→AKTA_JADI→LAPOR_BPN→SERTIFIKAT_TERBIT; resi BPN asli wajib
  (tanda `-`/kosong ditolak 400). `ppatBlanko` hanya konsep cetak, bukan
  blanko resmi BPN.
- Lainnya: `legalisasiAdvance` (REGISTERED→COMPLETED→HANDED_OVER),
  `protocolStatus`, `willHandover/willReview/willOpen` → `wills`
  (register_no unik/kantor, 026).

## Tahap 7 — Pasca (serah, arsip, pemerintah, portal)

- `govAdvance/govCheck POST /gov/{id}/advance|check` (ext.go:929,945) →
  `gov_submissions` (asistensi AHU/OSS manual, bukan API instansi).
- DMS: `dmsMove/dmsApproveStep/dmsArchiveMove/dmsHold/dmsDisposition`
  (dms.go:689,917,980,1009); `protocols` retensi 25 thn; portal klien
  read-only (`portal`, `portalInvoices`, `portalReceipt`; tulis hanya
  konfirmasi bayar + notif dibaca).

## Tahap 8 — Keuangan & serah-terima

- `invoiceCreate POST /invoices`, `invoiceVoid POST /invoices/{id}/void`
  (app.go:2649), `paymentCreate POST /payments`, bank
  (`bankUpload/bankMatch/bankManual`), `budgetSave`, `expenseCreate`,
  `tariffSave` → `invoices, invoice_lines, payments, bank_mutations,
  budgets, expenses`. `receipt/portalReceipt` cetak kwitansi.
- Serah: legalisasi HANDED_OVER, `willHandover`, DMS share/disposisi.

## Batas kewenangan Notaris vs PPAT

- HANTA NOTARIS (`deed.approve`, `repertorium.manage`): approve & sign akta,
  repertorium, wasiat/handover, blanko konsep. Role `notary` memegangnya;
  `staff/finance/doc_operator` TIDAK.
- PPAT (`ppatAdvance` butuh `repertorium.manage`): rantai akta tanah + resi
  BPN. Aplikasi TIDAK memvalidasi SK PPAT / wilayah kerja — itu di luar
  sistem (catat manual).
- Keuangan: `finance.manage` hanya `finance` (+ super_admin); notaris hanya
  `finance.view`. Auditor read-only penuh.
