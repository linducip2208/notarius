# ALUR UJI REALITA — 1 Permohonan PT-DIRIAN Fiktif (statis, tanpa eksekusi DB)

> Verifikasi **statis** (baca kode saja): tanpa POST/INSERT, tanpa ubah DB.
> Objek uji fiktif: **layanan `PT-DIRIAN`**, judul **`Pendirian PT CONTOH UJI`**,
> klien fiktif (mis. "PT CONTOH UJI" / penghadap Tn. Contoh).
> Kode layanan `PT-DIRIAN` ada di seed (`migrations/005_seed.sql:4`,
> tarif dasar 5000000) dan aturan workflow-nya di `migrations/026_matter_workflow.sql:50`
> (`require_checklist=1, require_verified_identity=1, require_deed_kind='PT', min_parties=1`).

Rantai target: `POST /requests` → `POST /requests/{id}/checklist`
→ `POST /deeds` (1-klik) → `POST /deeds/{id}/review` → `POST /deeds/{id}/approve`
→ `POST /deeds/{id}/handover` → `POST /quotations` → `POST /invoices`
→ `POST /payments`.

Peta status permohonan (sumber kebenaran tunggal): `internal/requests/requests.go:11-31`

```text
DRAFT → SUBMITTED → IN_REVIEW ⇄ NEED_DOCS → APPROVED → IN_PROGRESS → COMPLETED
```

Event otomatis (`internal/app/request_status_event.go:21-22`):

| Event | Target | Legal hanya dari |
|---|---|---|
| `deed_approved` | `IN_PROGRESS` | `APPROVED` |
| `handover_done` | `COMPLETED` | `IN_PROGRESS` |

Keduanya fail-open: bila `CanMove == false`, diam tanpa error
(`internal/app/request_status_event.go:58-60`).

---

## 0. Prasyarat peran (RBAC) — penting sebelum klik

Sumber: `internal/rbac/rbac.go:59-103`.

| Langkah | Guard di kode | Peran yang lolos |
|---|---|---|
| Buat permohonan | `PermCreateClient` (`internal/app/app.go:1759`) | super_admin, office_admin, notary, staff |
| Checklist | `requireStaff` = `PermViewClient` (`internal/app/dms.go:41-47`, dipakai `dms.go:1181`) | praktis semua peran staf incl. auditor |
| Buat draf / review | `PermDraftDeed` (`app.go:1918`); review `PermDraftDeed` **atau** `PermEditDeed` (`ext.go:1135`) | office_admin, notary, substitute, staff, doc_operator |
| Approve + handover akta | `PermApproveDeed` (`ext.go:1165`, `ext.go:1331`) | **hanya** super_admin, notary, substitute (+mandat aktif). `office_admin` / `staff` / `finance` **DITOLAK** |
| Penawaran / invoice / bayar / convert / status | `PermManageFinance` (`quotations.go:37,99,120`, `app.go:2645,2700`) | **hanya** super_admin + finance. `office_admin` dan `notary` (hanya `finance.view`) **DITOLAK** |
| Ubah status permohonan | `PermEditClient` (`app.go:1855`) | super_admin, office_admin, notary, staff |

**Jebakan 0 (RBAC split):** satu akun tidak bisa menjalani alur penuh.
Skenario uji butuh minimal 2 akun: (a) `staff`/`office_admin` untuk
permohonan→checklist→draf→review + manual move status, (b) `notary` untuk
approve→handover, (c) `finance` (atau super_admin) untuk
penawaran→invoice→bayar. Jangan uji dengan 1 akun staff lalu menyimpulkan
"approve rusak" — itu 403 by design.

---

## 1. Buat permohonan — `POST /requests`

- Route: `internal/app/app.go:555`. Handler: `requestCreate`, `app.go:1758`.
- Guard: `PermCreateClient` (`app.go:1759`).
- Form: `internal/app/templates/registers.gohtml:5-40` mengirim
  `client_id`, `service_id`, `title`, `notes` + hidden `x_svccode`
  dan panel kondisional korporat (`x_coname`, `x_cocity`, `x_coaddr`,
  `x_coauth`, `x_coissued`, `x_copaid`, `p_corp`). **KECUKUPAN: cukup.**
  Handler hanya wajib `client_id/service_id/title` (`app.go:1764-1798`);
  field `x_*` opsional, dilipat ke `notes` (`[Auto ...]`) dan row
  `companies` bila `x_svccode` berprefix `PT-` (`corpEntityOf`, `app.go:1831-1852`).
- Efek: status awal **`DRAFT`** (`requests.go:83-94`), AutoChecklist
  (PRO+, `app.go:1807`), instance workflow generik (`app.go:1812`),
  row perusahaan korporat (`app.go:1816`).
- Klik: `/requests` → pilih klien + layanan **PT-DIRIAN** → judul
  `Pendirian PT CONTOH UJI` → isi panel korporat → `Buat →`.
- Ekspektasi: redirect `/requests/{id}?flash=Permohonan+dibuat...`
  status `DRAFT`.
- Jebakan: judul kosong → 400 (`requests.go:85`). Tanpa template
  checklist terpasang, `#checklist` masih kosong — wajar, pasang di langkah 2.

## 2. Checklist — `POST /requests/{id}/checklist`

- Route: `internal/app/app.go:747`. Handler: `dmsChecklistSet`,
  `internal/app/dms.go:1180`. Guard: `requireStaff` (longgar, `dms.go:41`).
- Handler butuh: `item_id`, `status`, `note` opsional (`dms.go:1186-1188`).
- Form utama: `pages2.gohtml:131` mengirim `item_id` + `status`
  (**tanpa input `note`** — catatan hanya bisa via API/manual; untuk happy
  path tidak masalah). **KECUKUPAN: cukup** untuk transisi,
  **kurang** untuk pesan tindak lanjut (catatan tak terkirim dari form ini;
  notifikasi `NEEDS_REVISION/REJECTED` ke pembuat di `dms.go:1199-1205`
  akan berisi note kosong).
- Store: `internal/dms/dms.go:811-858`. Status valid hanya
  `MISSING/RECEIVED/VERIFIED/REJECTED/NEEDS_REVISION`, plus **peta transisi
  ketat**: `MISSING→RECEIVED` saja; `RECEIVED→VERIFIED|REJECTED|NEEDS_REVISION`;
  `VERIFIED→REJECTED|NEEDS_REVISION`; selebihnya **400**.
- **JEBAKAN 1 (paling sering):** `<select>` di `pages2.gohtml:131`
  menawarkan semua status sekaligus, termasuk `VERIFIED` langsung.
  Item baru berstatus `MISSING` yang sekali klik `VERIFIED` → **400
  "transisi checklist tidak valid dari MISSING ke VERIFIED"**.
  Klik yang benar: tiap item **dua kali** — `MISSING→RECEIVED` dulu,
  lalu `RECEIVED→VERIFIED`. Varian wizard korporat
  (`corp.gohtml:76`) lebih sempit (hanya RECEIVED/VERIFIED/MISSING;
  tanpa REJECTED/NEEDS_REVISION) — penolakan berkas harus lewat halaman
  detail, bukan wizard.
- Klik: `/requests/{id}` → `#checklist` → (pasang paket standar PT-DIRIAN
  8 item via `/dms/checklists` bila kosong) → tiap item OK=`RECEIVED`,
  lalu OK=`VERIFIED`.
- Ekspektasi: semua item `VERIFIED`/`RECEIVED`; `autoWorkflowSyarat`
  menandai step "syarat" DONE (`dms.go:1193-1198`).
- Catatan identitas: `workflowGate` untuk PT-DIRIAN juga menuntut tiap
  pihak punya identitas `VERIFIED` (`wf_rules.go:202-212`) — verifikasi di
  `/clients/{id}` (`identities/{id}/status`), di luar form checklist.

## 3. Draf 1-klik — `POST /deeds`

- Route: `internal/app/app.go:573`. Handler: `deedCreate`, `app.go:1917`.
- Guard: `PermDraftDeed` (`app.go:1918`).
- Handler butuh: `request_id` opsional, `title`, `kind`/`template_id`
  opsional dengan **auto-resolve**: `kind` dari `serviceDeedKind`
  (`PT-DIRIAN→PT`, `app.go:1977-1985`), template approved otomatis via
  `autoDeedTemplate` (`app.go:1955-1959`). Judul fallback
  `"{NamaLayanan} — {JudulReq}"` (`app.go:1947-1954`).
- Form 1-klik: `pages2.gohtml:40` mengirim **hanya**
  `request_id` + `title` (tanpa `kind`/`template_id`).
  **KECUKUPAN: cukup** — auto-resolve menutup sisanya.
- Store: `CreateDraft` (`deeds.go:42-68`) menolak `template_id`
  yang belum approved; `template_id == nil` diizinkan (body kosong).
- **JEBAKAN 2:** bila kantor belum punya template PT **approved**,
  draf 1-klik tetap jadi (200) tapi **isinya kosong** → langkah approve
  nanti 400 "isi akta terlalu pendek" (`ext.go:1193`). Cek kesiapan dulu
  di `/templates` (skor SIAP JUAL ≥ 80) / seed "⚡ Pasang 9 Konsep" lalu
  **Approve** template PT oleh notaris (`POST /templates/{id}/approve`).
- Klik: di `/requests/{id}` → `📄 Buat Draf Akta` (tombol ini hilang bila
  status COMPLETED/REJECTED/CANCELLED — by design, `pages2.gohtml:40`).
- Ekspektasi: redirect `/deeds/{id}?flash=Draf+akta+dibuat`, status draf `DRAFT`.

## 4. Review — `POST /deeds/{id}/review`

- Route: `internal/app/app.go:577`. Handler: `deedToReview`, `ext.go:1134`.
- Guard: `PermDraftDeed` **atau** `PermEditDeed` (`ext.go:1135`).
- Field: **tidak ada** (hanya path `{id}`). Form `app2.gohtml:30` dan
  tombol list `registers.gohtml:99` tidak mengirim field.
  **KECUKUPAN: cukup.**
- Syarat: status draf harus `DRAFT` (`ext.go:1152`), else 400.
- Klik: `/deeds` → `Review`, atau di detail `Kirim Review`.
- Ekspektasi: `DRAFT→IN_REVIEW`, redirect `/deeds?flash=Draf+dikirim...`.

## 5. Approve — `POST /deeds/{id}/approve` (+ efek ke permohonan)

- Route: `internal/app/app.go:576`. Handler: `deedApprove`, `ext.go:1164`.
- Guard: `PermApproveDeed` — notaris/substitute/super_admin saja
  (`ext.go:1165`); substitute wajib mandat aktif (`ext.go:1202-1210`).
- Handler butuh (wajib, `ext.go:1189-1219`): `number`, `witness1`,
  `witness2`, `read_place` (+ opsional `witness1_id`, `witness2_id`,
  `renvoi_note`). Form detail `app2.gohtml:31` mengirim **semua**
  field tersebut. **KECUKUPAN: cukup — jebakan lama (tombol approve
  tanpa saksi) SUDAH TERTUTUP**: halaman list (`registers.gohtml:99`)
  hanya punya tombol `Review`, tidak ada approve langsung; satu-satunya
  form approve ada di detail dan lengkap.
- Syarat lain: isi ≥ 50 karakter (`ext.go:1193`), hash cocok/self-heal
  (`ext.go:1232-1242`), `workflowGate(..., "approve")` lolos
  (`ext.go:1227`) — untuk PT-DIRIAN: checklist lengkap + identitas
  VERIFIED + `deed_kind == PT`, lalu store `Approve` menuntut status
  `IN_REVIEW` (`deeds.go:157`) + nomor unik per kantor + tulis
  repertorium (`deeds.go:144-188`).
- Efek ke permohonan: `advanceOnEvent(..., "deed_approved", ...)`
  (`ext.go:1256`) → `APPROVED→IN_PROGRESS` bila legal, else diam.
- Klik: `/deeds/{id}` → isi No. akta + Saksi 1/2 + Tempat pembacaan →
  `Sahkan`.
- Ekspektasi: status akta `APPROVED`+terkunci, entri repertorium baru,
  flash `segel:...`.

## 6. Serah terima — `POST /deeds/{id}/handover` (+ efek ke permohonan)

- Route: `internal/app/app.go:579`. Handler: `deedHandover`, `ext.go:1330`.
- Guard: `PermApproveDeed` (notaris-only, disengaja — komentar
  `ext.go:1322-1329`). UI `CanHandover` memakai izin yang sama
  (`ext.go:1775`).
- Handler butuh: `handed_to` wajib (`ext.go:1347`), `note` opsional.
  Form `app2.gohtml:86` mengirim keduanya. **KECUKUPAN: cukup.**
- Efek: baris `deed_handovers` + `advanceOnEvent(..., "handover_done")`
  (`ext.go:1366-1369`) → `IN_PROGRESS→COMPLETED` bila legal, else diam.
- Klik: `/deeds/{id}` → `Serahkan` → isi nama penerima.
- Ekspektasi: flash `Serah-terima+akta+dicatat`; riwayat di
  `deedDetail` (`ext.go:1380`).

## 7. Penawaran — `POST /quotations` (+ status/convert)

- Route: `internal/app/app.go:610`. Handler: `quotationCreate`,
  `quotations.go:36`. Guard: `PermManageFinance` (`quotations.go:37`).
- Handler butuh: `request_id` wajib (`quotations.go:42-46`) +
  baris `desc[]/qty[]/price[]`; bila baris kosong, prefill jujur dari
  `services.base_price` (PT-DIRIAN = 5000000, `quotations.go:68-78`).
  Harga diparse via `money.Parse`, negatif ditolak.
- Form: `pages2.gohtml:45-56` mengirim `request_id` + `valid_until` +
  `notes` + 2 baris `desc/qty/price` ter-prefill `Jasa {layanan}`.
  **KECUKUPAN: cukup.**
- Status: `SetQuotationStatus` (`finance/quotation.go:280-310`):
  `DRAFT→SENT|REJECTED|EXPIRED`, `SENT→REJECTED|EXPIRED`;
  **`ACCEPTED` manual DITOLAK** — hanya via convert.
  Convert `POST /quotations/{id}/convert` (`quotations.go:119`,
  `finance/quotation.go:327-442`): hanya `DRAFT/SENT` + punya
  `request_id` + belum lewat `valid_until` → invoice `ISSUED` + penawaran
  `ACCEPTED` atomik. Nomor invoice `INV-{suffix penawaran}`.
- Klik: `/requests/{id}` kartu Penawaran → `Buat Penawaran (DRAFT)` →
  `Kirim` (opsional) → `Setujui → Invoice`.
- Ekspektasi: penawaran `DRAFT` → (convert) → `ACCEPTED` + invoice `ISSUED`.

## 8. Invoice — `POST /invoices`

- Route: `internal/app/app.go:607`. Handler: `invoiceCreate`, `app.go:2644`.
  Guard: `PermManageFinance` (`app.go:2645`).
- Handler butuh: `number` + `client_id` wajib; `desc[]/qty[]/price[]`
  (qty > 0, harga ≥ 0); `due_date`, `invoice_kind` (FULL/DP/PELUNASAN,
  di-encode sebagai prefix baris — `app.go:2664-2689`); `request_id`
  opsional. Form `/finance`: `ops.gohtml:47-62` mengirim semuanya +
  tombol prefill tarif. **KECUKUPAN: cukup.**
- **JEBAKAN 3 (validasi diam-diam):** `request_id` yang klien/kantornya
  tidak cocok **diabaikan tanpa error** (`app.go:2652-2660`,
  `reqID` tetap `nil`) — invoice jadi tapi **tidak tertaut**
  permohonan. Pastikan `client_id` = klien permohonan PT CONTOH UJI.
  (Beda dengan quotation yang 400 bila `request_id` tak valid.)
- Matematika: subtotal + PPN 11% (`taxBps=1100`, `app.go:2690`,
  `finance.go:68-111`); nomor duplikat → 400.
- Dua jalur sah: (a) convert penawaran (disarankan — rantai terdokumentasi),
  (b) manual di `/finance?req={id}`. Keduanya berakhir `ISSUED`, `paid=0`.
- Klik: `/finance?req={id}` → isi Nomor + Klien + baris → `Terbitkan (PPN 11%)`.
- Ekspektasi: redirect `/finance?flash=Invoice+dibuat`, invoice `ISSUED`.

## 9. Bayar — `POST /payments`

- Route: `internal/app/app.go:613`. Handler: `paymentCreate`, `app.go:2699`.
  Guard: `PermManageFinance` (`app.go:2700`).
- Handler butuh: `invoice_id`, `amount` (> 0, `money.Parse`),
  `method`, `reference` (`app.go:2705-2711`). Form invoice
  `ops.gohtml:67` mengirim `invoice_id` + `amount` + `method`
  (**tanpa input `reference`** — tersimpan `""`, handler tidak
  mewajibkannya). **KECUKUPAN: cukup** (reference opsional de facto).
- Store `RecordPayment` (`finance.go:137-179`): tolak invoice VOID,
  tolak **overpay** (`paid+amount > total` → 400 sisa), status
  `ISSUED→PARTIAL→PAID` otomatis.
- Klik: `/finance` baris invoice → isi Nominal + metode → `Catat`.
  (Form portal `/portal/pay` memakai file bukti — jalur berbeda,
  `billing.gohtml:43`, butuh konfirmasi; tidak dipakai di skenario ini.)
- Ekspektasi: `PAID` (atau `PARTIAL` bila dicicil); kuitansi di
  `/invoices/{id}/receipt`.

---

## 10. Matriks transisi status permohonan vs skenario ini

`CanMove` (`requests.go:24-31`): satu-satunya penggerak adalah
`Store.Move` via `requestMove` (`app.go:1854`) + 2 hook otomatis.

| Dari | Ke | Cara dalam skenario | Catatan |
|---|---|---|---|
| DRAFT | SUBMITTED | manual `POST /requests/{id}/move` (`pages2.gohtml:36` / `registers.gohtml:70`, field `to`+`note`) | gate `workflowGate` hanya untuk target `IN_REVIEW`/`APPROVED` (`app.go:1867`) — move awal longgar |
| SUBMITTED | IN_REVIEW | manual | **digate**: checklist/identitas/parties bila strict mode |
| IN_REVIEW | APPROVED | manual | **digate** (sama). Mode ketat mati (`office_flags.strict_checklist=0`) → gate `move` lolos total (`wf_rules.go:168-173`) |
| **APPROVED** | **IN_PROGRESS** | **otomatis** via approve akta (`ext.go:1256`) | legal — satu-satunya jalan yang dipakai skenario |
| **IN_PROGRESS** | **COMPLETED** | **otomatis** via handover akta (`ext.go:1369`) | legal — satu-satunya jalan yang dipakai skenario |

**JALAN BUNTU UTAMA (wajib dibaca penguji):** tidak ada kode yang
memindahkan permohonan ke `APPROVED` secara otomatis. `deedApprove`
hanya mencoba `APPROVED→IN_PROGRESS` (`request_status_event.go:32-33`,
tes `request_status_event_test.go:10-24`). Urutan yang benar:

```text
manual: DRAFT→SUBMITTED→IN_REVIEW→APPROVED   (tombol "Ubah Status" / "OK")
otomatis: APPROVED→IN_PROGRESS               (saat akta disahkan)
otomatis: IN_PROGRESS→COMPLETED              (saat serah terima dicatat)
```

Bila operator mengesahkan akta sementara permohonan masih
`DRAFT/SUBMITTED/IN_REVIEW` (sangat mungkin — `deedCreate` dan
`deedApprove` tidak menuntut status permohonan apa pun), hook
`deed_approved` **diam** (fail-open) dan permohonan tetap di status lama
sementara akta sudah `APPROVED`. Demikian pula handover saat permohonan
bukan `IN_PROGRESS` → permohonan tidak `COMPLETED`. Ini **bukan bug
crash**, tapi yayımlan status bercabang yang membingungkan: akta sah +
repertorium terbit + invoice lunas, namun permohonan masih `IN_REVIEW`.
Uji realita harus menegaskan urutan: **APPROVE permohonan dulu, baru
sahkan akta; serahkan akta setelah status IN_PROGRESS terlihat.**

Celah pelengkap: `requestMove` manual mengizinkan `to=IN_PROGRESS` /
`to=COMPLETED` langsung dari `APPROVED`/`IN_PROGRESS` tanpa gate
workflow (gate hanya untuk `IN_REVIEW`/`APPROVED`), jadi operator bisa
mem-by-pass hook otomatis — dokumentasikan sebagai jalur manual cadangan,
bukan alur utama.

---

## 11. Skenario klik-demi-klik (PT CONTOH UJI)

> Akun: S = staff/office_admin, N = notary, F = finance.
> Semua ekspektasi = pesan flash / badge yang dirender template.

1. **(S)** `/requests` → `Permohonan Baru`: klien = klien fiktif,
   layanan = **PT-DIRIAN**, judul `Pendirian PT CONTOH UJI`, isi panel
   korporat (nama/kota/alamat/modal) → `Buat →`.
   Ekspektasi: `/requests/{id}`, `DRAFT`.
2. **(S)** Di `/requests/{id}#checklist`: pasang paket PT-DIRIAN bila
   kosong; tiap item: pilih `RECEIVED` → `OK`, lalu `VERIFIED` → `OK`
   (ingat Jebakan 1 — jangan langsung `VERIFIED`).
   Ekspektasi: `Checklist (8/8)`, tidak ada badge MISSING.
3. **(S)** Verifikasi identitas klien di `/clients/{id}` →
   `Proses` = `VERIFIED` (syarat gate PT-DIRIAN).
4. **(S)** `Ubah Status`: `SUBMITTED` → `IN_REVIEW` → `APPROVED`
   (berturut-turut; gate menolak bila langkah 2–3 belum tuntas —
   pesan `Mode ketat checklist aktif: ...` + `#checklist`).
   Ekspektasi: badge `APPROVED`.
5. **(S)** `📄 Buat Draf Akta` → buka `/deeds/{id}` → `Simpan revisi`
   hingga isi ≥ 50 karakter (template PT approved terisi otomatis bila
   Jebakan 2 sudah dibereskan). Ekspektasi: preview PT tampil.
6. **(S)** `Kirim Review`. Ekspektasi: status akta `IN_REVIEW`.
7. **(N)** Di `/deeds/{id}`: isi No. akta + Saksi 1 + Saksi 2 +
   Tempat pembacaan → `Sahkan`. Ekspektasi: `APPROVED` + segel +
   entri `/repertorium` baru + permohonan `APPROVED→IN_PROGRESS` otomatis.
   Verifikasi silang: badge permohonan kini `IN_PROGRESS`.
8. **(N)** Di `/deeds/{id}`: `Serahkan` → nama penerima (mis. direksi
   PT CONTOH UJI). Ekspektasi: flash serah terima + permohonan
   `IN_PROGRESS→COMPLETED` otomatis.
9. **(F)** Di `/requests/{id}` kartu Penawaran → `Buat Penawaran (DRAFT)`
   (prefill `Jasa Pendirian PT`) → `Kirim` (opsional) →
   `Setujui → Invoice`. Ekspektasi: penawaran `ACCEPTED`, invoice baru
   `ISSUED` (PPN 11%).
   Alternatif: `/finance?req={id}` → `Terbitkan` manual (ingat Jebakan 3:
   `client_id` harus klien yang sama).
10. **(F)** Di `/finance` baris invoice → Nominal penuh + `TRANSFER` →
    `Catat`. Ekspektasi: `PAID`; kuitansi `/invoices/{id}/receipt`.
    Cicilan parsial → `PARTIAL`, boleh lanjut hingga `PAID`; overpay
    ditolak jujur dengan sisa.
11. Verifikasi akhir: permohonan `COMPLETED`, akta `APPROVED`+repertorium,
    penawaran `ACCEPTED`, invoice `PAID`, checklist 8/8, audit
    `request.create/move`, `deed.draft/review/approve/handover`,
    `quotation.create/convert`, `invoice.create`, `payment.create`.

---

## 12. Ringkasan jebakan (cek ulang sebelum klaim "rusak")

| # | Lokasi kode | Gejala | Sebab & sikap |
|---|---|---|---|
| 0 | `rbac.go:59-103` | 403 di approve/finance padahal login sah | split peran by design; ganti akun N/F |
| 1 | `dms.go:833-850` vs `pages2.gohtml:131` | 400 transisi checklist MISSING→VERIFIED | form over-promise; wajib via RECEIVED dulu |
| 2 | `deeds.go:42-68`, `ext.go:1193` | draf kosong / approve "isi terlalu pendek" | belum ada template PT approved; pasang+approve template dulu |
| 3 | `app.go:2652-2660` | invoice jadi tapi tak tertaut permohonan | `request_id` mismatch diabaikan diam; samakan `client_id` |
| 4 | `request_status_event.go:58-60`, `requests.go:11-17` | akta APPROVED tapi permohonan tak maju | urutan salah: APPROVE permohonan manual dulu; hook fail-open diam |
| 5 | `quotations.go` + `finance/quotation.go:280-310` | tak bisa set ACCEPTED manual | by design; hanya via `convert` |
| 6 | `finance.go:159-161` | 400 "kelebihan bayar" | anti-overpay; bayar tepat sisa |
| 7 | form checklist tanpa `note` (`pages2.gohtml:131`) | notifikasi revisi tanpa pesan | keterbatasan form, bukan blokir alur |
| 8 | tombol draf hilang saat COMPLETED (`pages2.gohtml:40`) | tak bisa buat draf susulan | by design; buat draf sebelum COMPLETED |

Jebakan lama "tombol approve list tak kirim saksi" **tidak ditemukan
lagi**: satu-satunya form approve (`app2.gohtml:31`) lengkap
(`number/witness1/witness2/read_place` + opsional), dan halaman list
hanya menawarkan `Review` (`registers.gohtml:99`, `app2.gohtml:30`).
