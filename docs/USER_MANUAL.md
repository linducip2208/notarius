# USER_MANUAL (Bahasa Indonesia, ringkas)

Masuk lewat `/login` (email + kata sandi). Menu menyesuaikan peran Anda.
Panduan lengkap bergambar: buka `/docs`. Detail teknis: lihat `DEPLOYMENT_GUIDE`.

## Staf — alur harian (klien → sah)

1. Daftar klien di `/clients` → Cari dulu (otomatis), bila belum ada klik Klien Baru → isi nama, telepon, alamat → Simpan.
   *Tips: bila muncul peringatan duplikat, gabungkan dulu jangan buat baru.*
2. Buat permohonan di `/requests` → pilih Klien + Layanan (bisa bundel/paket) + judul → Buat.
   *Tips: judul jelas, contoh "AJB Tanah Pondok Indah — Budi".*
3. Lengkapi syarat di halaman permohonan: buka tab Checklist (`/dms/checklists`) → centang yang sudah ada.
   *Tips: yang MISSING (kurang) harus dilengkapi sebelum maju.*
4. Untuk PT/CV/Yayasan: isi Data Korporat di halaman permohonan, atau buka Wizard `/requests/{id}/wizard` → isi pemegang saham, pengurus, pemilik manfaat (BO).
   *Tips: wizard menuntun langkah demi langkah, tidak bisa loncat.*
5. Untuk jual-beli tanah: buka Wizard AJB `/requests/{id}/wizard-ajb` → isi penjual, pembeli, objek, harga.
   *Tips: cek luas + nomor sertifikat sesuai fisik.*
6. Untuk wasiat: buka Wizard Wasiat `/requests/{id}/wizard-wasiat` → isi pewasiat + ahli waris → Simpan (hanya notaris yang membuka kembali).
   *Tips: data wasiat bersifat rahasia.*
7. Unggah berkas (KTP, KK, sertifikat) di `/documents` → klik Jalankan OCR → tunggu DONE.
   *Tips: foto miring/buram bikin OCR gagal — unggah ulang yang tegak dan jelas.*
8. Verifikasi di `/documents/{id}` (tampilan 2 kolom: kiri berkas, kanan hasil baca) → cocokkan dengan aslinya → klik Verifikasi.
   *Tips: OCR hanya membantu baca; yang mengesahkan tetap manusia. Berkas belum verifikasi tidak lolos quality gate.*
9. Gerakkan status permohonan di `/requests/{id}` via tombol (DRAFT → SUBMITTED → IN_REVIEW → APPROVED → IN_PROGRESS → COMPLETED).
   *Tips: bila Mode Ketat aktif, status tidak bisa maju selama masih ada syarat MISSING.*
10. Buat draf akta 1-klik dari halaman permohonan (tombol Buat Draf Akta) atau di `/deeds` → draf jadi otomatis dari template.
    *Tips: draf 1-klik memakai data terverifikasi, jadi verifikasi dulu sebelum klik.*
11. Buka draf di `/deeds/{id}` → Preview → periksa daftar Todo/variabel kosong (missing-vars) → klik Edit untuk perbaiki → Simpan (revisi berversi otomatis).
    *Tips: jangan kirim ke notaris selama masih ada Todo merah.*
12. Klik Kirim Review → notaris mengesahkan di `/deeds/{id}` → akta terkunci + otomatis masuk `/repertorium`.
    *Tips: setelah sah, unduh PDF/DOCX; yang belum sah ada watermark DRAFT.*
13. Catat register lain bila perlu: `/ppat` (tanah) + cetak Blanko `/ppat/{id}/blanko`, `/legalisasi`, `/protocols`, `/gov`.
    *Tips: nomor register unik per kantor, jangan isi manual ganda.*

## Notaris — setuju & tanda tangan

1. Setujui template di `/templates` → buka konsep (DRAFT) → periksa isi → klik Approve (baru bisa dipakai draf).
   *Tips: template konsep tidak bisa dipakai produksi sebelum Approved.*
2. Periksa draf di `/deeds` → buka yang status Review → Preview + Todo → setujui (Approve + nomor akta) atau kembalikan revisi.
   *Tips: cek kesiapan jual: Todo harus nol sebelum Approve.*
3. Tanda tangani di `/deeds/{id}` (tombol Sign) → akta terkunci final.
   *Tips: yang terkunci tidak bisa diubah lagi — revisi berarti versi baru.*
4. Kelola wasiat di `/wills` (serah-terima, review tahunan, buka segel) — akses khusus notaris.
   *Tips: jangan bagikan isi wasiat ke staf yang tidak berhak.*
5. Pantau antrean di `/app` (menunggu persetujuan, siap TTE, macet SLA) dan `/tasks`.
   *Tips: dahulukan yang melewati SLA (merah).*

## Keuangan — invoice, bank, kuitansi

1. Atur tarif di `/tariffs` (harga per layanan/bundel) → saat buat invoice, tarif terisi otomatis.
   *Tips: ubah tarif di sini sekali, semua invoice baru ikut.*
2. Buat invoice di `/finance` → Invoice Baru (bisa DP/termin, multi-baris) → PPN 11% otomatis → kirim ke klien.
   *Tips: untuk DP, buat invoice termin 1 dulu; pelunasan setelah akta sah.*
3. Catat pembayaran per invoice di `/finance` → kelebihan bayar ditolak sistem (anti-overpay); salah input → Void, jangan hapus.
   *Tips: void tercatat audit, hapus tidak.*
4. Rekonsiliasi di `/finance/bank` → unggah mutasi → auto-match → yang belum cocok, cocokkan manual.
   *Tips: cocokkan tiap hari agar piutang akurat.*
5. Cetak kuitansi di `/invoices/{id}/receipt` dan pantau Piutang + Penerimaan di `/finance` dan `/reports`.
   *Tips: kuitansi hanya untuk invoice yang sudah dibayar.*

## Klien — portal

1. Masuk `/login` → buka `/portal` → lihat status permohonan (timeline) + tagihan.
   *Tips: bila kosong, akun belum ditautkan — hubungi staf.*
2. Bayar sesuai invoice di `/portal/invoices` → klik Konfirmasi Bayar → unduh kuitansi di `/portal/invoices/{id}/receipt`.
   *Tips: simpan kode verifikasi (contoh INV-…) untuk cek di `/verify`.*
3. Pantau draf akta terkait dan janji (jadwal) di `/portal`; tanya via WA dari tombol di halaman permohonan.
   *Tips: klien hanya melihat datanya sendiri.*

## Aktivasi awal kantor baru (Go-Live, 10 langkah)

1. Siapkan VPS + MySQL + TLS (ikut `DEPLOYMENT_GUIDE` langkah 1–7).
2. Backup DB dulu: `deploy/backup/backup.sh`.
3. Deploy biner: `go build -o bin/notaryos ./cmd/notaryos` → `systemctl restart notaryos`.
4. Cek migrasi + `/readyz` 200.
5. Buat akun notaris + staf (`/users`, `/roles`); aktifkan MFA di `/mfa`.
6. Seed checklist: `/dms/checklists` → Pasang Standar (23 paket).
7. Seed template: `/templates` → Pasang 8 Konsep → Approve satu per satu.
8. Isi tarif (`/tariffs`) + layanan/bundel (`/layanan-kami` via `/cms/services`).
9. Uji 1 permohonan end-to-end: `/requests` → checklist → `/documents` verifikasi 2-kolom → draf 1-klik `/deeds` → invoice DP `/finance` → cek `/portal`.
10. Nyalakan Mode Ketat (`/office`) + backup otomatis + cek `/reports`; detail di `/docs`.
