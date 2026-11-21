# PROJECT GAP ANALYSIS — sisa jujur (bukan janji)

P0 (fungsional, sebelum klaim produksi):
- Uji restore backup di VPS target + dokumen hasilnya.
- Matriks izin otomatis 9 peran × rute kunci (kini manual).
- Uji beban: Approve konkuren + OCR queue + search FTS data besar.
- Klik-penuh browser: INTERACTION_TESTS 12 skenario (E2E DMS sebagian jalan).

P1 (produk):
- Rich-text editor akta (kini textarea + sisip klausul — jujur, bukan klaim editor).
- Pratinjau DOCX/XLSX (kini: tolak jujur + sarankan unduh).
- Penggabungan metadata duplikat (kini: tautan relasi manual).
- Notifikasi SLA/eskalasi workflow (kini: daftar Perlu Tindakan pasif).
- Unduh akta langsung dari portal klien (kini: via detail dengan izin staf).

P2 (skala):
- Engine search khusus (kini: MySQL FULLTEXT + LIKE fallback — cukup untuk skala kantor).
- S3 backend (interface siap; kini: local only).
- Malware scanner hook (kini: MIME+magic-byte sniff; interface adapter belum).

BLOCKED eksternal:
- Tinjauan notaris atas alur hukum (B-06) — wajib sebelum data sungguhan.
- TLS publik + COOKIE_SECURE=true di produksi.
- AV host memblokir *.test.exe (B-04) — workaround rename; CI Linux disarankan.
