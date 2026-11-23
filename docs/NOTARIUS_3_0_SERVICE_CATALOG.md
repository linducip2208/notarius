# NOTARIUS 3.0 — Katalog Layanan (32 aktif + 3 bundel)

Sumber syarat: `migrations/005_seed.sql`, `021_ppat.sql`, `022_notaris_lengkap.sql`, `024_bundling.sql`.
Checklist: `internal/app/seed_catalog.go` (`standardChecklistSeeds`, 27 paket).
Template: `internal/app/seed_catalog.go` (`standardTemplateSeeds`, 9 konsep DRAFT) +
`internal/app/app.go` (`deedTemplateKeyword`, `serviceDeedKind`, `autoDeedTemplate`:
keyword → deed kind → template approved pertama; hanya template APPROVED yang bisa dipakai draf).

> Seluruh klaim kepatuhan di bawah ini berstatus BUTUH REVIEW NOTARIS
> (lihat `docs/NOTARIUS_3_0_LEGAL_COMPLIANCE.md`). Jangan klaim compliance otomatis.

## corporate (10)

| Kode | Nama | Syarat (dari seed) | Checklist | Template keyword → seed | Deed kind | Review hukum |
|------|------|--------------------|-----------|-------------------------|-----------|--------------|
| PT-DIRIAN | Pendirian PT | KTP+NPWP pendiri; susunan direksi & komisaris; modal dasar/disetor | Ada (8 item) | `PT` → Standar Pendirian PT | PT | BUTUH REVIEW NOTARIS |
| PT-UBAH | Perubahan PT (AD/komposisi) | Akta terakhir; data perubahan; persetujuan pemegang saham | Ada (3) | `PT` → Standar Pendirian PT (serumpun) | PT | BUTUH REVIEW NOTARIS |
| PT-BUBAR | Pembubaran PT | Akta terakhir; keputusan RUPS; likuidator | Ada (3) | `PT` → Standar Pendirian PT (serumpun) | PT | BUTUH REVIEW NOTARIS |
| CV-DIRIAN | Pendirian CV | KTP sekutu; domisili usaha | Ada (4) | `CV` → Standar Pendirian CV | CV | BUTUH REVIEW NOTARIS |
| CV-UBAH | Perubahan CV | Akta terakhir; data perubahan | Ada (2) | `CV` → Standar Pendirian CV (serumpun) | CV | BUTUH REVIEW NOTARIS |
| FIRMA | Pendirian Firma | KTP sekutu; domisili usaha | Ada (4) | `Firma` → TIDAK ADA seed cocok → fallback approved pertama | Firma | BUTUH REVIEW NOTARIS |
| YAYASAN-DIRIAN | Pendirian Yayasan | KTP pendiri; struktur organ; maksud & tujuan | Ada (5) | `Yayasan` → TIDAK ADA → fallback | Yayasan | BUTUH REVIEW NOTARIS |
| KOPERASI | Akta Koperasi | KTP pendiri; AD/ART konsep | Ada (4) | `Koperasi` → TIDAK ADA → fallback | Koperasi | BUTUH REVIEW NOTARIS |
| ◆ BUNDEL-PT-OSS | Paket Pendirian PT + NIB + NPWP | KTP+NPWP pendiri; susunan direksi; modal; KBLI | Ada (6) | `PT` → Standar Pendirian PT | PT | BUTUH REVIEW NOTARIS |
| ◆ BUNDEL-CV-NIB | Paket CV + NIB | KTP sekutu; domisili; KBLI | Ada (4) | `CV` → Standar Pendirian CV | CV | BUTUH REVIEW NOTARIS |

## agreement (6)

| Kode | Nama | Syarat (dari seed) | Checklist | Template keyword → seed | Deed kind | Review hukum |
|------|------|--------------------|-----------|-------------------------|-----------|--------------|
| PK-UMUM | Perjanjian Kerjasama | Identitas para pihak; objek perjanjian | Ada (4) | `Perjanjian` → Standar Perjanjian Kerjasama | Agreement | BUTUH REVIEW NOTARIS |
| HUTANG | Perjanjian Utang-Piutang | Identitas para pihak; nilai & tenor | Ada (5) | `Perjanjian` → Standar Perjanjian Kerjasama | Agreement | BUTUH REVIEW NOTARIS |
| SEWA | Perjanjian Sewa-Menyewa | Identitas para pihak; objek sewa; jangka waktu | Ada (4) | `Perjanjian` → Standar Perjanjian Kerjasama (catatan: seed "Standar Sewa Menyewa" hanya kena bila dipilih manual) | Agreement | BUTUH REVIEW NOTARIS |
| JUALBELI-NONPPAT | Perjanjian Jual-Beli (non-PPAT) | Identitas para pihak; objek & harga | Ada (3) | `Perjanjian` → Standar Perjanjian Kerjasama | Agreement | BUTUH REVIEW NOTARIS |
| KUASA | Surat Kuasa Notariil | Identitas pemberi & penerima kuasa; ruang lingkup | Ada (2) | `Kuasa` → Standar Surat Kuasa | POA | BUTUH REVIEW NOTARIS |
| PRENUP | Perjanjian Kawin (Prenup/Postnup) | KTP calon/pasangan; daftar harta | Ada (2) | `Kawin` → TIDAK ADA nama cocok → fallback kind `Prenup` → Standar Prenup | Prenup | BUTUH REVIEW NOTARIS |

## will (2)

| Kode | Nama | Syarat (dari seed) | Checklist | Template keyword → seed | Deed kind | Review hukum |
|------|------|--------------------|-----------|-------------------------|-----------|--------------|
| WASIAT | Akta Wasiat (Olografis/Umum) | KTP pewasiat; daftar ahli waris; objek wasiat | Ada (2: + 2 saksi dicatat saat pengesahan) | `wasiat` → Standar Wasiat | Will | BUTUH REVIEW NOTARIS (akses terbatas /wills) |
| WASIAT-DAFTAR | Pendaftaran Wasiat | Salinan akta wasiat | TIDAK ADA | `wasiat` → Standar Wasiat (pendaftaran tak butuh draf) | Will | BUTUH REVIEW NOTARIS |

## legalization (4) — TANPA checklist seed, TANPA template seed

| Kode | Nama | Syarat (dari seed) | Checklist | Template | Deed kind | Review hukum |
|------|------|--------------------|-----------|----------|-----------|--------------|
| LEGALISASI | Legalisasi Tanda Tangan | Dokumen asli; KTP penghadap | TIDAK ADA | TIDAK ADA → fallback approved pertama | Other | BUTUH REVIEW NOTARIS |
| WAARMERKING | Waarmerking (pembukuan surat) | Dokumen asli | TIDAK ADA | TIDAK ADA → fallback | Other | BUTUH REVIEW NOTARIS |
| SALINAN | Salinan Akta | Nomor akta; keperluan | TIDAK ADA | TIDAK ADA → fallback | Other | BUTUH REVIEW NOTARIS |
| KUTIPAN | Kutipan Akta | Nomor akta; keperluan | TIDAK ADA | TIDAK ADA → fallback | Other | BUTUH REVIEW NOTARIS |

Alur keempatnya: `/legalisasi` (REGISTERED → COMPLETED → HANDED_OVER).

## other (2) — asistensi manual, BUKAN integrasi otomatis

| Kode | Nama | Syarat (dari seed) | Checklist | Template | Deed kind | Review hukum |
|------|------|--------------------|-----------|----------|-----------|--------------|
| AHU-BANTU | Asistensi Pengajuan AHU | Data perseroan lengkap | Ada (2) | TIDAK ADA → fallback | Other | BUTUH REVIEW NOTARIS |
| OSS-BANTU | Asistensi Perizinan OSS | Data perusahaan & KBLI | Ada (2) | TIDAK ADA → fallback | Other | BUTUH REVIEW NOTARIS |

Alur: `/gov` (checklist + bukti manual; staf input di portal resmi). DILARANG klaim integrasi AHU/OSS otomatis.

## ppat (8) — register terpisah `/ppat`, resi BPN asli wajib

| Kode | Nama | Syarat (dari seed) | Checklist | Template keyword → seed | Deed kind | Review hukum |
|------|------|--------------------|-----------|-------------------------|-----------|--------------|
| PPAT-AJB | AJB (Akta Jual Beli Tanah) | Sertifikat asli; KTP/KK/NPWP penjual & pembeli + pasangan; PBB lunas; PPh & BPHTB | Ada (5) | `AJB` → Standar AJB | AJB | BUTUH REVIEW NOTARIS |
| PPAT-HIBAH | Akta Hibah Tanah | Sertifikat asli; KTP pemberi & penerima; PBB lunas; BPHTB | Ada (4) | `Hibah` → Standar Hibah | Hibah | BUTUH REVIEW NOTARIS |
| PPAT-TUKAR | Akta Tukar Menukar | Dua sertifikat asli; KTP kedua pihak; PBB lunas kedua objek | Ada (3) | `Tukar` → TIDAK ADA → fallback | TukarMenukar | BUTUH REVIEW NOTARIS |
| PPAT-APHB | APHB (Pembagian Hak Bersama) | Sertifikat waris; SKW + KTP semua ahli waris; PBB lunas | Ada (3) | `APHB` → TIDAK ADA → fallback | APHB | BUTUH REVIEW NOTARIS |
| PPAT-SKMHT | SKMHT (Kuasa Membebankan HT) | Sertifikat; KTP pemberi & penerima kuasa; data kredit bank | Ada (2) | `SKMHT` → TIDAK ADA → fallback | SKMHT | BUTUH REVIEW NOTARIS |
| PPAT-APHT | APHT (Pembebanan Hak Tanggungan) | Sertifikat; SKMHT bila dikuasakan; data kredit & peringkat | Ada (3, paket gabungan APHT/SKMHT) | `APHT` → TIDAK ADA → fallback | APHT | BUTUH REVIEW NOTARIS |
| PPAT-INBRENG | Inbreng Tanah ke PT | Sertifikat; KTP pemilik; akta PT; RUPS persetujuan inbreng | Ada (3) | `Inbreng` → TIDAK ADA → fallback | Inbreng | BUTUH REVIEW NOTARIS |
| ◆ BUNDEL-AJB-BALIKNAMA | Paket AJB + Balik Nama | Sertifikat; KTP penjual & pembeli; PBB lunas; PPh & BPHTB | Ada (5, + resi lapor balik nama BPN) | `AJB` → Standar AJB | AJB | BUTUH REVIEW NOTARIS |

Aturan workflow (`026_matter_workflow.sql`): PPAT-AJB/HIBAH/TUKAR/APHB + BUNDEL-AJB wajib resi BPN (`require_bpn_receipt=1`); resi `"-"`/kosong DITOLAK saat naik ke LAPOR_BPN (kode: `ppatAdvance`). Blanko cetak `/ppat/{id}/blanko` adalah LEMBAR KERJA, bukan blanko resmi BPN.

## Ringkasan kesiapan seed

- Checklist: 27/32 layanan ADA; 5 TANPA seed (WASIAT-DAFTAR, LEGALISASI, WAARMERKING, SALINAN, KUTIPAN) — pasang manual via `/dms/checklists` atau notaris tentukan syaratnya (BUTUH REVIEW NOTARIS).
- Template konsep: 9 DRAFT di kode (`standardTemplateSeeds`) — label tombol UI "8 Konsep" belum sinkron; sinkronisasi menunggu verifikasi, JANGAN ubah kode di tahap ini.
- Template konsep TIDAK bisa dipakai draf sebelum Approve notaris (`POST /templates/{id}/approve`).
