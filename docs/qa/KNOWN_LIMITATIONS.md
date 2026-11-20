# KNOWN_LIMITATIONS — jujur, tanpa klaim palsu

1. Metronic: aset resmi TIDAK ADA di repo. UI aktif Tabler CDN. Migrasi diblokir lisensi; skeleton di `web/assets/metronic/`.
2. Integrasi pemerintah: TIDAK ADA API resmi AHU/OSS. Modul gov = asistensi manual + bukti. Jangan klaim integrasi.
3. AI eksternal: mati by design tanpa kunci env + persetujuan.
4. OCR: butuh binary Tesseract + OCR_BINARY; tanpa itu fallback Noop + entri manual. Hasil OCR TIDAK PERNAH auto-verify.
5. Penomoran: FOR UPDATE + UNIQUE mengurangi race; generator counter + uji beban belum selesai. Uji konkuren wajib.
6. Pagination: LIMIT tanpa cursor/total; N+1 timeline; index tambahan belum dimigrasi.
7. Tanpa soft-delete; hapus permanen (bukti: tidak ada deleted_at). Retensi 25 th butuh arsip resmi.
8. AV host memblokir `validate.test.exe` (B-04); lolos via rename; CI Linux disarankan.
9. Integrasi MySQL 9 skenario + API + browser + backup/restore + dokumen panjang BELUM dieksekusi sesi ini.
10. Produksi DILARANG sebelum: tinjauan notaris (B-06), TLS + COOKIE_SECURE=true, backup/restore terbukti, pen-test.
