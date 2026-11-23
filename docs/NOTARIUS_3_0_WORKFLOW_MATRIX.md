# NOTARIUS 3.0 — Matriks Workflow 11 Mesin Status

Gate umum: `workflowGate` (wf_rules.go:113, dipakai requestMove app.go:1820,
ppatAdvance app.go:2394). Audit: `audit.Record` + `request_events` /
`doc_workflow_events`. Izin via `internal/rbac/rbac.go`.

| # | Mesin (tabel) | Status | Boleh (permisi) | Gate | Audit |
|---|---|---|---|---|---|
| 1 | Permohonan `service_requests` | DRAFT→SUBMITTED→IN_REVIEW→APPROVED→IN_PROGRESS→COMPLETED; cabang NEED_DOCS↔IN_REVIEW, REJECTED, CANCELLED (requests.go:11) | staff+ (`client.edit`): `requestMove` | workflowGate kind=move saat →IN_REVIEW/APPROVED | request_events + audit |
| 2 | Akta `deed_drafts` | DRAFT→IN_REVIEW→APPROVED→SIGNED | draf: `deed.draft/edit`; approve/sign: `deed.approve` (notaris) | nomor+isi wajib; hash duplikat ditolak | audit deed.* |
| 3 | PPAT `ppat_register` | DRAFT_AKTA→AKTA_JADI→LAPOR_BPN→SERTIFIKAT_TERBIT | `repertorium.manage` | workflowGate kind=ppat bila tertaut request; resi BPN asli wajib ke LAPOR_BPN | audit ppat.advance.* |
| 4 | Legalisasi `legalizations` | REGISTERED→COMPLETED→HANDED_OVER | `legalization.process` | biaya tercatat; serah dicatat | audit |
| 5 | Pemerintah `gov_submissions` | OPEN→SUBMITTED→NEED_DOCS→APPROVED→COMPLETED (+CANCELLED) | `government.submit` | checklist per idx (`govCheck`); guard tahap-3 | audit gov.* |
| 6 | Protokol `protocols` | ACTIVE→ARCHIVED (+penghapusan terlarang) | `protocol.view` + manage | lokasi arsip wajib; retensi 25 thn | audit |
| 7 | Janji `appointments` | SCHEDULED→DONE/CANCELLED/NOSHOW | staf | mundur dari terminal ditolak (tahap-3); tidak bisa DONE→SCHEDULED | audit implisit |
| 8 | Wasiat `wills` | DRAFT→REGISTERED→HANDED_OVER/OPENED | notaris | register_no unik/kantor (026); akses terbatas | audit will.* |
| 9 | Invoice `invoices` | DRAFT→ISSUED→PAID (+VOID) | lihat `finance.view`; kelola `finance.manage` | void butuh alasan; approve retry bedakan 1062 (tahap-3) | audit finance.* |
| 10 | Dokumen DMS `documents` | DRAFT→IN_REVIEW→APPROVED→ARCHIVED (+HOLD) | operator+ | `dmsMove/dmsApproveStep`; hold/disposisi terpisah | doc_workflow_events |
| 11 | Tugas `tasks` | OPEN→DONE | staf (assignee) | sederhana, tanpa gate layanan | audit minimal |

## Jebakan SUDAH diperbaiki (tahap 3, komit 75b89f0)

1. `deedSign` tidak mengubah status → kini APPROVED→SIGNED atomik bersyarat.
2. Resi BPN `-`/kosong lolos → kini 400 dengan pesan eksplisit.
3. Guard gov/appointment/checklist/booking longgar → dikeraskan.
4. Counter nomor balap → retry bedakan sumber 1062.
5. Link deed↔request hilang di PPAT → gate ppat kini resolve via
   `deed_drafts.request_id`.
6. Seed checklist HUTANG/PK/SEWA + template Wasiat hilang → `seed_catalog.go`.

## SISA jebakan (belum — jangan diklaim aman)

1. **REJECTED terminal tanpa koreksi**: `transitions["REJECTED"]={}`
   (requests.go:19) — permohonan ditolak TIDAK bisa kembali ke IN_REVIEW/
   NEED_DOCS; perlu revisi manual/duplikat. Sama untuk COMPLETED/CANCELLED.
2. **SIGNED tanpa ARCHIVED**: tidak ada endpoint SIGNED→ARCHIVED; arsip
   hanya via protokol — rantai gantung di laporan.
3. **PPAT tanpa tautan request lolos gate**: `deedID` tak valid → gate
   dilewati (fail-open disengaja, wf_rules.go) — hanya resi manual yang jaga.
4. **DB lama pra-026 fail-open**: tabel rule hilang → engine nil; kantor
   belum migrasi berjalan TANPA aturan (diam-diam longgar).
5. **Default rule longgar bila template kosong**: checklist required tapi
   tanpa item template = lolos sendiri (wf_rules.go butir 3).
6. **Portal pay-confirm tanpa verifikasi bank**: konfirmasi sepihak, rekonsiliasi
   manual di `/finance/bank`.
