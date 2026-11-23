# BUSINESS PROCESS AUDIT 4.0 — Alur Nyata per Kategori Layanan

> FASE 1D — BACA saja. Sumber: kode Go + migrasi SQL repo ini. Tanpa fabrikasi pasal.
> Legenda klasifikasi: **[UUJN]** = kewajiban UU Jabatan Notaris (klaim umum, tanpa nomor pasal — verifikasi ke teks UU);
> **[SOP]** = praktik baku kantor Indonesia; **[UMUM]** = praktik umum administrasi; **[KONFIG]** = aturan konfigurasi repo ini.
> **NEEDS_LEGAL_REVIEW** = butuh advokat/notaris pembina memvalidasi sebelum dijadikan acuan kepatuhan.

## 0. Tulang punggung (berlaku untuk SEMUA layanan)

- Permohonan: `service_requests` (migrasi `002_clients.sql`). Status: `DRAFT→SUBMITTED→IN_REVIEW⇄NEED_DOCS→APPROVED→IN_PROGRESS→COMPLETED` + `REJECTED/CANCELLED` terminal — `internal/requests/requests.go:11-31` (`transitions`, `CanMove`).
- Satu-satunya penggerak status resmi = `Store.Move` (`requests.go:97-118`, **pakai tx**: UPDATE + `request_events`).
- Gate tambahan = `workflowGate(db,office,req,"","move"|"approve"|"ppat")` (`internal/app/wf_rules.go:110`). Engine kedua (independen, tidak menyentuh status): `workflow_instances/steps/step_states` migrasi `027_workflow_engine.sql`, manual via `workflowStep` (`workflow_instance.go:387`), auto best-effort via `autoWorkflowStep` (`:260`).
- Akta: `deed_drafts` DRAFT→IN_REVIEW→APPROVED→SIGNED→ARCHIVED (`internal/app/ext.go:1134-1359`, `internal/deeds/deeds.go`).

## 1. Jawaban 4 pertanyaan fokus

### (1) Satu Matter bisa banyak pihak / dokumen / akta?
**Ya, didukung.** Pihak: klien utama (`service_requests.client_id`) + N pihak di `matter_parties(request_id,client_id,role)` unik per (request,client,role) — `026_matter_workflow.sql:7-17`; tambah/lepas via `partyAdd/partyRemove` (`ext.go:804-858`). Dokumen: N baris `doc_checklist_records` + `deed_files`/arsip per request. Akta: `deed_drafts.request_id` nullable, satu request boleh punya N draf (dibatasi hanya oleh `FindByRequest`-style lookup "draf pertama" di `workflowLinkedDeedKind`, `wf_rules.go:381-385` — draf kedua dst **tak dinilai gate deed_kind** = gap). Wasiat: `wills.deed_id` nullable, relasi longgar via client_id (`will_wizard.go:40-56`).

### (2) Tiap layanan punya workflow sendiri atau satu jalur dipaksa?
**Satu jalur status dipaksa + diferensiasi di 3 lapis.** (a) Status request tunggal `transitions` untuk semua (`requests.go:11`). (b) Diferensiasi via `service_workflow_rules(service_code→require_checklist,require_verified_identity,require_deed_kind,require_bpn_receipt,min_parties)` — `026:23-54`. (c) Template checklist per `service_code` (`seed_catalog.go:13-167`, 27 paket) + template engine `027` — **tapi seed engine hanya 3 layanan**: PT-DIRIAN(9 langkah), PPAT-AJB(7), KUASA(5) (`seed_catalog.go:187-217`). CV, AJB-nonPPAT, Prenup, Wasiat, Legalisasi/Waarmerking, AHU/OSS **tanpa template engine** → ikut `defaultWorkflowRule` (`wf_rules.go:60-68`) bila tanpa baris rule, atau rule tanpa langkah visual.

### (3) Di mana status bisa dilewati diam-diam?
1. `requestMove` (`app.go:1913`) hanya meng-gate target `IN_REVIEW/APPROVED` — transisi lain (`DRAFT→SUBMITTED→...`, `NEED_DOCS→IN_REVIEW`, `APPROVED→IN_PROGRESS→COMPLETED`, `→REJECTED/CANCELLED`) **tanpa gate checklist/identitas**. **[KONFIG]**
2. `strict_checklist` MATI (default, `office_flags`; `wf_rules.go:168-181`): gate `move` lolos total, `ppat` hanya tahan resi BPN. Fail-open bila tabel `service_workflow_rules` hilang (`:143-151`) atau db nil/req≤0 (`:111`). **[KONFIG]**
3. `workflowStep` manual (`workflow_instance.go:387`) menandai DONE **tanpa cek checklist/evidence** — hanya cek RBAC peran. Engine `ValidateTransition` (`workflow.go:104`) yang punya aturan skippable/checklist **belum dipakai handler mana pun** (komentar `workflow.go:7`). **[KONFIG]**
4. `autoWorkflowStep` fail-open: tanpa instance/tabel 027 = diam (`:281-282`); `ensureWorkflowInstance` gagal = diam (`:87-101`). Permohonan tetap jadi tanpa workflow. **[KONFIG]**
5. `advanceOnEvent` (`request_status_event.go:55`): `deed_approved` merantai IN_REVIEW→APPROVED→IN_PROGRESS otomatis; `handover_done` COMPLETED otomatis — **melewati gate `workflowGate` manusia** (CanMove saja). **[KONFIG] NEEDS_LEGAL_REVIEW** (apakah otomasi status boleh menggantikan verifikasi notaris?).
6. `legalisasiAdvance` (`ext.go:1652`) → `AdvanceLegalization` (`deeds.go:271`): REGISTERED→COMPLETED→HANDED_OVER tanpa gate identitas/checklist. `willHandover` (`deeds/wills.go:63`): satu UPDATE ke ARCHIVED tanpa gate saksi/pembacaan ulang. **[KONFIG]**

### (4) Transaksi multi-tabel TANPA transaksi DB
| Lokasi | Multi-tulis | Tanpa tx? |
|---|---|---|
| `requestCreate` (`app.go:1817`): `reqs.Create` + `AutoChecklist` + `ensureWorkflowInstance` + `corp.UpsertCompany` | 3–4 tabel | **Ya, tanpa tx induk** (masing-masing Exec sendiri; gagal parsial = permohonan yatim tanpa checklist/perusahaan) |
| `deedSign` (`ext.go:1267`): INSERT `deed_signatures` + UPDATE `deed_drafts→SIGNED` | 2 tabel | **Ya, 2 Exec terpisah** — crash di antaranya = tanda tangan tanpa status (atau sebaliknya) |
| `deedApprove` handler (`ext.go:1164`): UPDATE saksi + `Approve()`(tx sendiri) + audit + `advanceOnEvent` | lintas store | **Ya, orkestrasi tanpa tx global** (audit/auto-step best-effort) |
| `deedHandover` (`ext.go:1369`): INSERT `deed_handovers` + auto-step + `advanceOnEvent` | 2–3 tulis | **Ya** |
| `ppatAdvance` (`app.go:2466`): UPDATE `ppat_register` + auto-step + audit | 2–3 tulis | **Ya** (single-row UPDATE saja yang atomik) |
| `legalisasiCreate` (`app.go:2353`) → `RegisterLegalization` (`deeds.go:247`): validasi deed + INSERT | 1 INSERT | OK (tunggal), tapi tanpa cek duplikat `register_no` di SQL |
| Aman (pakai tx): `requests.Move` (`requests.go:105`), `deeds.Approve/approveOnce` (`deeds.go:145`, FOR UPDATE + kunci `offices`), `deeds.SaveVersion` (`:86`), `workflowStep` (`workflow_instance.go:467`), `ensureWorkflowInstance` (`:131`) | — | — |

## 2. Alur nyata per layanan (handler : tabel : gate : gap)

### A. PT-DIRIAN (plus PT-UBAH/PT-BUBAR, CV, FIRMA, YAYASAN, KOPERASI)
1. `POST /requests` `requestCreate` (`app.go:1817`) → `service_requests[DRAFT]` + `AutoChecklist` (salin template `PT-DIRIAN` sbg MISSING) + `ensureWorkflowInstance` + `corp.UpsertCompany` (`companies`, `002_clients.sql:104`). **[UMUM]**
2. Pihak: `partyAdd` → `matter_parties`; identitas wajib VERIFIED (`identities`, skema 016) — gate `move`/`approve`. **[SOP + KONFIG]**
3. Maju `requestMove`→IN_REVIEW/APPROVED (gate: checklist MISSING/REJECTED blokir; pesan legacy dipertahankan `app.go:1928`). **[KONFIG]**
4. Draf: `deedCreate` (`app.go:1976`, kind=`PT` via `serviceDeedKind`, template approved wajib — `deeds.go:42-68`) → `deedToReview` (DRAFT→IN_REVIEW, `:1134`, **tanpa gate isi**) → `deedApprove` (`:1164`, gate: nomor + isi ≥50 char + 2 saksi + tempat baca + hash + mandat pengganti + `workflowGate approve`) → tx `Approve` + `repertorium` entry (`deeds.go:144-188`). **[UUJN: repertorium/notaris pengganti bermandat/SOP: 2 saksi + pembacaan — NEEDS_LEGAL_REVIEW untuk detail formalitas]**
5. `deedSign`→SIGNED, `deedArchive`→ARCHIVED, `deedHandover`→`deed_handovers` + auto COMPLETED. **[SOP]**
6. AHU/OSS: **manual asistensi**, bukan API — `gov_submissions` via `internal/gov/store.go:15` (Create) / `Advance` (`gov.go:85`, wajib bertahap, tanpa lompat) / checklist portal. Tanpa tautan FK ke `service_requests` (kolom `reference` teks bebas). **[UMUM + KONFIG]**
- **Gap**: (i) PT-UBAH/PT-BUBAR & CV/FIRMA/YAYASAN/KOPERASI **tanpa rule di 026** (hanya PT-DIRIAN/CV-DIRIAN/FIRMA/YAYASAN-DIRIAN/KOPERASI ada — PT-UBAH/BUBAR & CV-UBAH ikut default longgar); (ii) tanpa template engine (kecuali PT-DIRIAN); (iii) wizard korporat? tidak ada (hanya AJB & wasiat punya wizard read-only); (iv) `requestCreate` tanpa tx (kolom 4).

### B. AJB-nonPPAT (JUALBELI-NONPPAT, HUTANG, SEWA, PK-UMUM)
1-3. Sama seperti A (rule `026:46-49` mewajibkan kind `Agreement`, min_parties=2). `serviceDeedKind`→`Agreement` (`app.go:2069`). **[SOP: min 2 pihak untuk perjanjian timbal-balik — NEEDS_LEGAL_REVIEW apakah 2 pihak adalah syarat UU atau SOP]**
4. Draf→review→approve→sign→archive→handover identik Jalur A (repertorium otomatis). **[UUJN/SOP]**
- **Gap**: tanpa wizard terpandu (AJB-nonPPAT tak punya wizard; hanya PPAT-AJB yang punya `ajbWizard` read-only); tanpa resi/pelaporan eksternal; objek bukan sertifikat tanah (sesuai seed checklist `JUALBELI-NONPPAT`).

### C. PPAT-AJB (plus HIBAH/APHB/APHT/SKMHT/TUKAR/INBRENG, bundel AJB-BALIKNAMA)
Dua rel parallel: permohonan + register PPAT terpisah.
1. Permohonan seperti A (rule `026:34-41`: kind sesuai + `require_bpn_receipt=1` untuk AJB/HIBAH/TUKAR/APHB/BUNDEL + min_parties). **[KONFIG]**
2. `ajbWizard` (`ppat_wizard.go:16`, read-only, 4 langkah panduan; simpan via endpoint lama). **[UMUM]**
3. Draf kind AJB → approve (gate `approve` + `deed_kind` match `workflowLinkedDeedKind`, `wf_rules.go:367`: PPAT baca `ppat_register.jenis` fallback `deed_drafts.kind`). **[KONFIG]**
4. Register PPAT: `ppatCreate` (`app.go:2429`, jenis enum AJB/Hibah/TukarMenukar/APHB/Inbreng/SKMHT/APHT; `ppat_register`, `021_ppat.sql`) → `ppatAdvance` DRAFT_AKTA→AKTA_JADI→LAPOR_BPN→SERTIFIKAT_TERBIT (`:2466`; resi `-`/kosong ditolak 400; gate `ppat` dinilai bila tertaut deed). Auto-step `akta`/`lapor` DONE. **[SOP + KONFIG; blanko BPN hanya konsep cetak (`ppatBlanko`, :2539) — bukan blanko resmi]**
5. Serah: `deedHandover` + `advanceOnEvent handover_done`→COMPLETED. **[SOP]**
- **Gap**: (i) PPAT tanpa tautan `deed_id` → gate dilewati (`:2486`); (ii) `ppat_register.status` bebas maju manual satu-per-satu tapi tanpa verifikasi BPN asli (resi hanya string, tanpa validasi format); (iii) `PPAT-INBRENG/SKMHT/APHT` tanpa `require_bpn_receipt` (0) — **NEEDS_LEGAL_REVIEW**; (iv) pemisahan peran Notaris vs PPAT hanya lewat kategori UI + RBAC `PermManageRepert`, bukan pemisahan jabatan/SK — **NEEDS_LEGAL_REVIEW**.

### D. Kuasa (KUASA → POA)
1-3. Seperti A; rule `026:44` (kind POA, min 2); template engine 5 langkah (`seed_catalog.go:209`). **[KONFIG]**
4. Draf→approve→sign→archive→handover. Tanpa register khusus (bukan legalisasi). **[SOP]**
- **Gap**: kuasa di bawah umur / kuasa khusus tanah (butuh SKMHT, bukan kuasa umum) **tidak dibedakan sistem** — NEEDS_LEGAL_REVIEW.

### E. Prenup/Postnup (PRENUP → Prenup)
1-3. Seperti A; rule `026:45` (kind Prenup, min 2); template konsep `tplPrenup`. **[KONFIG]**
4. Draf→approve→sign→handover. **Pencatatan/pengesahan ke Dukcapil tidak ada di sistem** (hanya draf internal). **[Gap — SOP generik: perjanjian kawin perlu pencatatan agar berlaku terhadap pihak ketiga; NEEDS_LEGAL_REVIEW]**

### F. Wasiat (WASIAT + WASIAT-DAFTAR)
Dua dunia: permohonan/akta AND register wasiat tertutup.
1. Permohonan (rule `026:42-43`); `willWizard` read-only (`will_wizard.go:16`, **notaris-only**); relasi wasiat↔request longgar (via client_id, `will_wizard.go:51-56`). **[KONFIG]**
2. Draf kind `Will` → approve → repertorium (seperti A). **[UUJN/SOP]**
3. Register: `willCreate` (`ext.go:1762`, **notaris-only**, `wills` + `register_no` wajib + unik per kantor `026:58`) → `willReview` (catat review tahunan, `ops_growth.go:677`) → `willOpen` (saksi wajib, sekali saja, `:692`) → `willHandover` (→ARCHIVED, `:1783`). List/baca **notaris-only** (`willReader`, `:1747`). **[UUJN: kerahasiaan & register wasiat — NEEDS_LEGAL_REVIEW untuk masa retensi & pelaporan]**
- **Gap**: (i) `wills` tanpa `request_id` (audit lintas alur putus); (ii) `handover_done` tidak dipasang di `willHandover` (komentar `request_status_event.go:107-109`) — permohonan WASIAT tak pernah auto-COMPLETED; (iii) status wasiat bebas (`REGISTERED/ARCHIVED`) tanpa transisi-kanonik di store.

### G. Legalisasi vs Waarmerking — DUA layanan beda
Katalog `005_seed.sql:20-21`: `LEGALISASI` (Legalisasi Tanda Tangan) vs `WAARMERKING` (pembukuan surat) + `SALINAN/KUTIPAN` — kind enum `legalizations` (`003_deeds.sql:68`, diperluas GROSSE di `020`). Handler **sama** (`legalisasi`/`legalisasiCreate`/`legalisasiAdvance`, `app.go:2317-2384` → `RegisterLegalization/AdvanceLegalization`, `deeds.go:247-295`) dengan pembeda hanya kolom `kind`:
- LEGALISASI **[SOP generik: penghadap tanda tangan di depan notaris + cap]**; WAARMERKING **[SOP generik: pembukuan tanggal surat di bawah tangan, TANPA pengesahan tanda tangan]** — perbedaannya prosedural (hadir vs catat), tapi **form, register, dan transisi identik di kode** (REGISTERED→COMPLETED→HANDED_OVER + fee). Tidak ada cek kehadiran/pembacaan di handler. **NEEDS_LEGAL_REVIEW** (pastikan SOP kantor membedakan pembuktian hadir vs bukukan; sistem saat ini mengandalkan operator).
- **Gap**: (i) SALINAN/KUTIPAN/GROSSE boleh tanpa `deed_id` (standalone) — `deeds.go:256` hanya validasi bila diisi; (ii) tanpa nomor register unik per kantor (bentrok no register hanya dicegah di akta, bukan legalisasi); (iii) tanpa gate identitas/checklist.

### H. AHU/OSS asistensi (AHU-BANTU, OSS-BANTU, bundel PT-OSS/CV-NIB)
1. Permohonan seperti A (tanpa rule 026 → default). Checklist seed ada (`AHU-BANTU/OSS-BANTU/BUNDEL-*`, `seed_catalog.go:118-145`). **[KONFIG]**
2. `gov_submissions` (`004_ops.sql:63`): DRAFT→READY→SUBMITTED→ACCEPTED (+REJECTED), wajib bertahap (`gov.go:85-103`), checklist manual per kanal (`:52-78`), resi/bukti arsip manual, follow-up tanggal (`ops_growth.go:716`). **[UMUM]**
3. Bundel (`024_bundling.sql` + seed `BUNDEL-*`): satu permohonan paket (PT+OSS / AJB+balik nama / CV+NIB) dengan `serviceDeedKind` memetakan ke kind tunggal (PT/CV/AJB) — **satu akta per bundel**. **[KONFIG]**
- **Gap**: (i) tanpa FK request↔gov_submission (rujukan teks); status permohonan vs status gov bisa divergen; (ii) tanpa API resmi (by design, komentar `gov.go:1-4`); (iii) bundel multi-akta (mis. PT-UBAH + APHT) tak dimodelkan — satu request satu kind.

## 3. Tabel ringkas gate vs gap

| Layanan | Gate menegakkan (file:line) | Yang HILANG |
|---|---|---|
| PT/CV/Firma/Yayasan/Koperasi | `workflowGate move/approve` (`wf_rules.go:110`); saksi+baca+hash+mandat (`ext.go:1186-1242`); tx repertorium (`deeds.go:145`) | PT-UBAH/BUBAR & CV-UBAH tanpa rule; `requestCreate` tanpa tx; gov tanpa FK |
| AJB-nonPPAT | kind Agreement + min 2 (`026:46-49`) | tanpa wizard; tanpa validasi objek |
| PPAT-AJB/dll | kind + resi BPN (`026:34-41` + `app.go:2509`); adjacent-only `ppatAdvance` | tanpa tautan = gate lewat; resi string bebas; INBRENG/SKMHT/APHT tanpa resi wajib |
| Kuasa | kind POA + min 2; engine 5 langkah | kuasa khusus tanah tak dibedakan |
| Prenup | kind Prenup + min 2 | tanpa pencatatan Dukcapil |
| Wasiat | notaris-only + register unik + saksi buka | tanpa request_id; tanpa auto-COMPLETED; |
| Legalisasi/Waarmerking | enum kind + transisi ketat (`deeds.go:280`) | prosedur hadir vs bukukan tak dibedakan kode; tanpa gate identitas |
| AHU/OSS | adjacent-only (`gov.go:85`) + checklist kanal | tanpa FK request; tanpa API (by design) |

## 4. Klasifikasi & NEEDS_LEGAL_REVIEW (5 butir)

1. Repertorium otomatis saat approve, protokol 25 thn, register PPAT terpisah, kerahasiaan wasiat → **[UUJN]** (prinsip; nomor pasal tidak dikutip di sini — cocokkan dengan teks UUJN/UU PPAT berlaku). Otomasi status (`advanceOnEvent`) menggantikan verifikasi manusia → **NEEDS_LEGAL_REVIEW**.
2. 2 saksi + tempat/tanggal pembacaan, cek bersih BPN, PBB/PPh/BPHTB lunas, persetujuan pasangan, SKW waris, RUPS/likuidator, verifikasi identitas VERIFIED, resi BPN asli (tolak "-") → **[SOP]** praktik kantor umum; bukan validasi sistem ke sumber resmi.
3. Urutan DRAFT→…→COMPLETED, SLA per layanan (`remind/sla.go:109`), template checklist 27 paket, min_parties, skippable steps → **[KONFIG]** repo ini, bukan norma hukum.
4. Perbedaan LEGALISASI (penghadap hadir) vs WAARMERKING (bukukan tanggal) → **[SOP]** generik; sistem tidak menegakkan → **NEEDS_LEGAL_REVIEW** sebelum klaim kepatuhan.
5. Pemisahan Notaris vs PPAT, pencatatan prenup ke Dukcapil, pelaporan wasiat terpusat, keabsahan TTE internal `deedSign` (hash internal, bukan PSrE tersertifikasi) → **NEEDS_LEGAL_REVIEW**; jangan sajikan TTE internal sebagai TTE tersertifikasi.
