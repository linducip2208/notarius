# USER_FLOWS
## Staf: permohonan → akta → arsip
Login → Dashboard (antrean) → Klien (buat/cari, cek duplikat) → Permohonan (buat DRAFT →
SUBMITTED → IN_REVIEW → APPROVED → IN_PROGRESS → COMPLETED; NEED_DOCS bila berkas kurang) →
Akta (draf dari template approved → revisi berversi → IN_REVIEW → APPROVED + nomor → terkunci) →
Repertorium/Protokol otomatis tercatat → Invoice → Pembayaran → Laporan.

## Keuangan
Invoice (subtotal + PPN 11% → total) → ISSUED → PARTIAL (cicilan) → PAID; overpayment ditolak;
VOID menghentikan pembayaran. Piutang & penerimaan bulanan dari query.

## Klien (portal)
Registrasi → login → permohonan baru → unggah KTP/pendukung → verifikasi hasil OCR →
lacak status → tinjau draf → revisi → terima invoice → bayar (catat) → unduh salinan aman.

## Pemerintah (asistensi)
Siapkan data (korporat) → checklist AHU/OSS → staf berwenang input manual di portal resmi →
arsipkan nomor bukti/SK → status SUBMITTED → ACCEPTED.
