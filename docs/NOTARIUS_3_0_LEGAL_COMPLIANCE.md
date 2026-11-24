# NOTARIUS 3.0 — Matriks Kepatuhan Hukum (ringkas, generik)

Prinsip: tulis NAMA regulasi + peruntukan umum saja. TANPA nomor pasal/ayat —
belum ada yang diverifikasi notaris. SEMUA baris berstatus BUTUH REVIEW NOTARIS.
Aplikasi membantu administrasi; keputusan hukum tetap di tangan Notaris/PPAT.
Rujukan awal yang sudah dicatat repo: `docs/LEGAL_REQUIREMENTS.md`.

## Matriks per kategori layanan

| Kategori | Dasar hukum GENERIK yang dipakai | Peruntukan umum di aplikasi | Status verifikasi |
|----------|----------------------------------|-----------------------------|-------------------|
| corporate (PT/CV/Firma/Yayasan/Koperasi + 2 bundel) | UUJN (UU 30/2004 jo. 2/2014); UU PT 40/2007; UU Yayasan + perubahannya; KUHD (persekutuan); UU PDP 27/2022 | Bentuk akta & kewenangan notaris; kaidah perseroan; kaidah yayasan/persekutuan; perlindungan data klien (konsent, minimisasi, NIK) | BUTUH REVIEW NOTARIS |
| agreement (PK/Hutang/Sewa/JualBeli-nonPPAT/Kuasa/Prenup) | UUJN; KUHPerdata (hukum perjanjian, umum saja); UU PDP 27/2022 | Syarat sah perjanjian dicatat sebagai checklist verifikasi manusia; data pihak dilindungi | BUTUH REVIEW NOTARIS |
| will (Wasiat + Pendaftaran) | UUJN; KUHPerdata (hukum waris/testamen, umum saja); UU PDP 27/2022 | Akses terbatas `/wills` (notaris/auditor); 2 saksi dicatat saat pengesahan; register wasiat unik per kantor | BUTUH REVIEW NOTARIS |
| legalization (Legalisasi/Waarmerking/Salinan/Kutipan) | UUJN (repertorium, minuta/salinan/grosse); UU PDP 27/2022 | Alur `/legalisasi` REGISTERED → COMPLETED → HANDED_OVER + biaya tercatat | BUTUH REVIEW NOTARIS |
| other (AHU-BANTU, OSS-BANTU) | UUJN; UU PT 40/2007 (umum); UU PDP 27/2022 | Asistensi MANUAL via `/gov` (checklist + bukti); staf input di portal resmi | BUTUH REVIEW NOTARIS |
| ppat (7 + 1 bundel AJB) | UUJN (pemisahan peran); PP 24/2016 (batas wilayah kerja PPAT, umum saja); aturan pendaftaran tanah & pajak peralihan (PPh/BPHTB/PBB, umum saja) | Register PPAT TERPISAH dari repertorium (`ppat_register`); resi BPN asli wajib; gate checklist+identitas+pihak | BUTUH REVIEW NOTARIS |
| Lintas kategori: arsip & retensi | UUJN (protokol/arsip 25 tahun, umum saja) | Bendel `/protocols` + lokasi arsip; retensi minuta/salinan/grosse | BUTUH REVIEW NOTARIS |
| Lintas kategori: sistem elektronik & audit | UU ITE + PP PSTE (umum saja) | Login+MFA, RBAC 9 peran, jejak audit, backup terenkripsi | BUTUH REVIEW NOTARIS |
| Lintas kategori: keuangan | Aturan perpajakan yang berlaku (PPN; tarif pasti ikut finance) | `taxBps` terpusat; anti-overpay; void tercatat audit | BUTUH REVIEW NOTARIS (konfirmasi tarif oleh finance) |

## Pemisahan Notaris vs PPAT (wajib)

- Dua buku berbeda: repertorium Notaris (`/repertorium`) vs register PPAT (`/ppat`, `ppat_register`). Jangan campur nomor/akun.
- Akta tanah (AJB/Hibah/Tukar/APHB/Inbreng/SKMHT/APHT) HANYA lewat jalur PPAT + wilayah kerja PPAT yang berwenang (PP 24/2016, umum) — BUTUH REVIEW NOTARIS untuk batas konkret per kantor.
- Template/akta wasiat akses terbatas (notaris/auditor); staf & portal diblokir (gate `kind='Will'`).

## DAFTAR LARANGAN KLAIM (jangan janjikan ke klien/publik)

1. Blanko resmi BPN — `/ppat/{id}/blanko` adalah LEMBAR KERJA konsep, BUKAN blanko resmi. Penyalinan ke formulir resmi dilakukan manual oleh PPAT.
2. Integrasi AHU/OSS otomatis — TIDAK ADA API resmi yang diotorisasi; `/gov` hanya checklist + bukti manual. Jangan hardcode endpoint / klaim sinkronisasi.
3. Verifikasi BPN otomatis — cek bersih sertifikat & resi BPN adalah pekerjaan manusia; aplikasi hanya menyimpan nomor resi + tanggal lapor ASLI (fiksi ditolak) + pengingat progresif (H-7/H-3 kuning, lewat jatuh tempo = tanggal lapor + 14 hari merah) + tugas pengingat otomatis saat H≤3.
4. Keputusan hukum otomatis — OCR/AI/template tidak pernah mengesahkan; gate (checklist VERIFIED, identitas terverifikasi, min pihak, 2 saksi wasiat) hanya MENAHAN maju, yang menyetujui tetap manusia.
5. Tarif pajak final — PPN/PPh/BPHTB di aplikasi bersifat pencatatan; angka berlaku ikut konfirmasi finance/konsultan pajak.

## Tindak lanjut sebelum klaim compliance

- [ ] Review notaris atas seluruh baris matriks (tandai tanggal + nama di `LEGAL_REQUIREMENTS.md`).
- [ ] Konfirmasi batas wilayah PPAT per kantor + prosedur pendaftaran tanah setempat.
- [ ] Konfirmasi tarif PPN/PPh/BPHTB yang berlaku oleh finance.
- [ ] Kebijakan retensi arsip & purge data pribadi (PDP) disahkan pimpinan.
