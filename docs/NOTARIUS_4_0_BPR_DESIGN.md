# NOTARIUS 4.0 — BPR Design (Keputusan Final)

> Status: MEMUTUSKAN. Dokumen ini bukan audit — semua opsi ditutup di sini.
> Stack dikunci: **Go + MySQL 8.x + Tabler server-render (html/template + HTMX), modular monolith.**
> BUKAN Metronic, BUKAN SPA, BUKAN REST API publik.

## 0. Konteks 1 paragraf

Audit 3.x membuktikan fondasi cukup: Matter = `service_requests` + `matter_parties`,
Party = `clients` + `matter_parties` + `identities`, Workflow = `service_workflow_rules`
+ `internal/app/wf_rules.go` gate + fondasi generik `workflow_*`. BPR 4.0 memutuskan
menuntaskan di atas fondasi itu, bukan membangun ulang.

## 1. Prinsip: Pertahankan vs Rombak vs Tidak Dibangun

### 1.1 DIPERTAHANKAN (jangan disentuh strukturnya)

1. `service_requests` sebagai tabel Matter — karena sudah mengikat office/client/service/status/assignee + `request_events`.
2. `clients` sebagai master identitas — karena semua FK pihak, NIK-cipher, dan consent sudah menempel di sini.
3. `matter_parties(request_id, client_id, role)` sebagai relasi multi-pihak — karena UNIQUE `(request_id, client_id, role)` sudah mencegah duplikat peran.
4. `service_workflow_rules(service_code PK)` + `workflowGate()` di `internal/app/wf_rules.go` sebagai satu-satunya gate — karena fail-open/fail-closed per cek sudah melindungi DB lama.
5. Server-render Tabler + HTMX — karena keputusan D-01 sudah mengunci Tabler (MIT) dan semua handler mengembalikan HTML.
6. Modul PPAT (`ppat_register`, `deed_drafts.kind` AJB/dst) dibekukan apa adanya — karena 2-buku (Repertorium vs Register PPAT) adalah kewajiban hukum, bukan preferensi.

### 1.2 DIROMBAK (dirapikan, tanpa migrasi merusak)

1. Wizard per layanan (`ppat_wizard.go`, `will_wizard.go`) dipersempit jadi panggil `workflowGate()` — karena logika `if` ad-hoc duplikat dengan gate.
2. Checklist DMS (`doc_checklist_*`) jadi satu-satunya sumber status dokumen untuk gate — karena gate dan DMS pernah menghitung MISSING dengan query berbeda.
3. Penamaan peran `matter_parties.role` dinormalisasi ke enum tertutup — karena nilai bebas (`PIHAK` default) merusak `min_parties` dan label blokir.
4. `workflow_instances.request_id UNIQUE` dipertahankan tapi pengisian dibuat lazy (hanya saat layanan butuh step) — karena tidak semua layanan butuh mesin step.

### 1.3 TIDAK DIBANGUN (ditutup di §3)

1. Tabel `matters`, `parties` generik, mesin workflow abstrak baru, REST API publik, tema Metronic, modul PPAT baru.

## 2. Arsitektur Final (nama aktual, bukan rencana)

```
offices 1──* clients 1──* identities (status VERIFIED / verified legacy)
services 1──* service_requests (client_id = pihak utama, service_id, status)
service_requests 1──* matter_parties *──1 clients (role: PENGHADAP/PENJUAL/PEMBELI/KREDITUR/DEBITUR/PEMBERI_KUASA/PENERIMA_KUASA/SAKSI)
service_requests 1──* request_events (audit from_status → to_status)
service_requests 1──* deed_drafts 1──* ppat_register (khusus PPAT, dibekukan)
services 1──1 service_workflow_rules (require_checklist, require_verified_identity, require_deed_kind, require_bpn_receipt, min_parties)
service_requests 1──0/1 workflow_instances (request_id UNIQUE) 1──* workflow_step_states
workflow_templates (office_id, service_code, version) 1──* workflow_steps
```

Aturan main: (a) Matter = satu baris `service_requests`; pihak tambahan HANYA via
`matter_parties`. (b) Party = satu baris `clients`; peran = baris `matter_parties.role`.
(c) Workflow = `service_workflow_rules` dievaluasi `workflowGate(db, officeID, reqID, svcCode, kind)`
dengan `kind ∈ {move, approve, ppat}` di `requestMove`/`deedApprove`/`ppatAdvance`;
`workflow_*` generik (`internal/workflow/workflow.go`: `NextSteps`, `ValidateTransition`,
`IsStepBlocked`) hanya dipakai bila `workflow_instances` ada untuk request itu.

## 3. Daftar Eksplisit TIDAK Dibangun

| # | Yang ditolak | Alasan 1 baris |
|---|---|---|
| N-1 | REST API publik / versi JSON paralel | Semua handler kontraknya HTML Tabler+HTMX; API ganda = beban auth + PDP ganda tanpa pemakai. |
| N-2 | Migrasi ke Metronic | D-01 mengunci Tabler MIT; Metronic = risiko lisensi + rewrite template tanpa nilai bisnis. |
| N-3 | Modul PPAT baru / register baru | `ppat_register` + `deed_drafts.kind` sudah memenuhi 2-buku; ubah = risiko kepatuhan BPN. |
| N-4 | Tabel `matters` / `parties` generik | `service_requests` + `clients` + `matter_parties` sudah menutup relasi; tabel baru = migrasi data + FK ganda. |
| N-5 | Workflow engine abstrak baru (ganti gate) | `wf_rules.go` + `service_workflow_rules` + paket `workflow` murni sudah cukup; mesin baru = fail-open tak teruji. |
| N-6 | SPA / framework JS berat | Server-render + HTMX sudah memenuhi interaksi wizard; SPA = rewrite auth/CSRF tanpa kebutuhan. |

## 4. Peta Jalan 3 Batch Berikutnya (konkret)

### Batch A — Penegakan gate + peran (tanpa skema baru)
- `internal/app/wf_rules.go`: normalisasi `role`, kunci default `min_parties` per `service_code`.
- `internal/app/app.go` (`requestMove`, `ppatAdvance`), `internal/app/ext.go` (`deedApprove`): hapus `if` ad-hoc, sisa panggil `workflowGate()`.
- `web/templates/...` (form tambah-pihak + checklist): tampilkan pesan blokir gate verbatim.
- Kriteria selesai: `go test ./internal/app/ -run 'Workflow|RequestStatus'` hijau.

### Batch B — Checklist sebagai sumber tunggal
- `internal/dms/dms.go`, `internal/app/dms*.go`: samakan query hitung MISSING/REJECTED dengan `workflowChecklistCounts()`.
- `migrations/032_*.sql` (aditif saja): index `matter_parties(request_id)`, CHECK/seed `role` tertutup.
- `tests/workflow_integration_test.go`: tambah kasus multi-pihak + identitas belum VERIFIED.
- Kriteria selesai: gate dan halaman checklist tidak pernah beda angka.

### Batch C — Aktivasi selektif `workflow_*` generik
- `internal/app/workflow_instance.go`: buat instance lazy hanya untuk service_code yang butuh step (bundling AJB-BalikNama dulu).
- `internal/workflow/workflow.go`: pakai `ValidateTransition` di satu handler `advance` baru, tetap di belakang `workflowGate()`.
- `migrations/027_workflow_engine.sql`: SEED `workflow_templates/steps` hanya untuk 1 layanan pilot, tanpa ubah handler lain.
- Kriteria selesai: 1 layanan pilot berjalan step-by-step; layanan lain tidak berubah perilaku.

---
*Maks 150 baris. Melanggar = revisi. Rujukan: `migrations/002_clients.sql`, `021_ppat.sql`, `026_matter_workflow.sql`, `027_workflow_engine.sql`, `internal/app/wf_rules.go`, `internal/workflow/workflow.go`, `docs/DECISIONS.md` (D-01 Tabler).*
