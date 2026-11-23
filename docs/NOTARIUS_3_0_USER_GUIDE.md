# NOTARIUS 3.0 — Panduan Harian per Peran (pola Matter multi-pihak)

Matter = 1 permohonan (`/requests/{id}`) boleh melibatkan BANYAK pihak
(penjual/pembeli/pasangan/kuasa/saksi). Aturan lolos-maju per layanan ada di
`service_workflow_rules` (migrasi 026): checklist, identitas terverifikasi,
jumlah pihak minimum, resi BPN (PPAT). Pelanggaran = status DITAHAN + pesan
alasan di `#checklist`. Template konsep wajib Approve dulu (tidak bisa dipakai draf).

## STAF (pendaftaran → draf siap review)

1. Cari/daftar klien di `/clients` (cek duplikat dulu) → buat permohonan di
`/requests` (POST `/requests`: pilih Klien + Layanan + judul jelas).
2. Tambah PIHAK di `/requests/{id}` → `POST /requests/{id}/parties`
(pilih klien + peran: PENJUAL/PEMBELI/pasangan/pemberi-penerima kuasa/saksi).
Hapus salah input via `POST /requests/{id}/parties/{pid}/delete`.
3. Pasang & lengkapi checklist: `/dms/checklists` → ⚡ Pasang Standar, lalu
`POST /requests/{id}/checklist` per item (RECEIVED → VERIFIED; MISSING menahan maju).
4. Wizard sesuai layanan: korporat `/requests/{id}/wizard` (saham/pengurus/BO),
tanah `/requests/{id}/wizard-ajb` (penjual→pembeli→objek&pajak→register PPAT),
wasiat `/requests/{id}/wizard-wasiat` (pewbasiat+ahli waris; rahasia).
5. Unggah berkas `/documents` → OCR → verifikasi 2-kolom, samakan NIK/nama
dengan data pihak (identitas tak terverifikasi = DITAHAN).
6. Gerakkan status `POST /requests/{id}/move` (DRAFT→SUBMITTED→IN_REVIEW…);
bila DITAHAN, baca alasan (checklist/pihak/identitas/resi) lalu lengkapi.
7. Buat DRAF 1-KLIK: `POST /deeds` cukup isi `request_id` (template approved
dipilih otomatis via keyword→kind→approved pertama). Cek di `/deeds/{id}`:
Preview + missing-vars harus NOL sebelum kirim.
8. Kirim ke notaris: `POST /deeds/{id}/review`. Pantau antrean di `/app`, `/tasks`.

## NOTARIS (setuju, sahkan, tanda tangan)

1. Approve template: `/templates` → buka konsep DRAFT → periksa → Approve
(`POST /templates/{id}/approve`; hanya peran notaris).
2. Review draf: `/deeds` (filter Review) → `/deeds/{id}` → Preview + Todo NOL →
Approve + nomor akta (`POST /deeds/{id}/approve`) → otomatis masuk `/repertorium`.
3. Tanda tangan: `POST /deeds/{id}/sign` di `/deeds/{id}` (terkunci final;
revisi = versi baru via `POST /deeds/{id}/version`).
4. PPAT: catat di `/ppat` (`POST /ppat`: jenis+nomor+para pihak+objek) → majukan
`POST /ppat/{id}/advance` (DRAFT_AKTA→AKTA_JADI→LAPOR_BPN→SERTIFIKAT_TERBIT).
Resi BPN ASLI wajib saat LAPOR_BPN (tanda `-`/kosong DITOLAK).
5. Wasiat khusus di `/wills`: serah-terima (`POST /wills/{id}/handover`),
review tahunan, buka segel bersaksi. Jangan bagikan isi ke yang tak berhak.

## KEUANGAN (tarif → invoice → lunas)

1. Tarif sekali di `/tariffs` (dipakai otomatis saat buat invoice).
2. Buat invoice: `/finance` → Invoice Baru (`POST /invoices`, multi-baris, DP/termin).
3. Catat bayar per invoice di `/finance` (kelebihan DITOLAK; salah input → Void
`POST /invoices/{id}/void`, jangan hapus). Rekonsiliasi `/finance/bank`.
4. Kuitansi: `/invoices/{id}/receipt`. Piutang & penerimaan: `/finance`, `/reports`.

## KLIEN (portal, read-only miliknya sendiri)

1. `/portal`: timeline status permohonan + draf akta terkait + jadwal.
2. `/portal/invoices` → Konfirmasi Bayar (`POST /portal/pay`) → kuitansi di
`/portal/invoices/{id}/receipt` (kode verifikasi bisa dicek di `/verify`).
3. Bila kosong: akun belum ditautkan — hubungi staf. Tanya via tombol WA di permohonan.

## SERAH & ARSIP (penutup Matter)

1. Legalisasi: `/legalisasi` → `POST /legalisasi/{id}/advance` sampai HANDED_OVER.
2. Wasiat: serah-terima tercatat di `/wills` (hanya notaris).
3. Arsip: bendel di `/protocols` + lokasi arsip (retensi 25 thn — BUTUH REVIEW NOTARIS).
4. Gov (AHU/OSS): `/gov` checklist + bukti manual; input resmi diketik staf di
portal pemerintah (TANPA klaim integrasi otomatis).
