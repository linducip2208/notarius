# REGRESSION_REPORT — 2026-11-20

## Yang diuji (tidak regresi)
- Build/vet/unit: hijau sebelum dan sesudah perubahan (11 paket + validate-rename).
- Router lama tetap ada; tidak ada route yang dihapus; hanya tambah (deed detail/version, documents, audit, office, tasks, expenses, tariffs, search, files download, protocols create, legalisasi advance, appointment status, metrics, 404).
- Logout kini wajib CSRF (perubahan disengaja; sidebar sudah kirim token).

## Risiko regresi (wajib uji sebelum rilis)
1. Sidebar `can` gating: pastikan peran client tidak kehilangan /portal, staf tidak kehilangan menu kerja. Uji matrix peran.
2. CSRF logout: pastikan tombol Keluar tetap 302 (token tersedia di sidebar).
3. FOR UPDATE di Approve: pastikan tidak deadlock di MySQL 8 RR; uji 2 klien sahkan bersamaan.
4. MIME sniff: pastikan JPG/PNG/WebP/PDF asli lolos; pastikan EXE-as-PDF ditolak.
5. XFF trusted: pastikan login dari LAN langsung tetap rate-limit; dari nginx tepercaya pakai IP asli.
6. Template baru (app2.gohtml): pastikan ParseFS tidak gagal (terbukti build + render; perlu klik tiap halaman).

## Belum diuji (jujur)
Integrasi DB, API auth/CSRF/RBAC, dokumen panjang, upload serangan, backup/restore, UI browser/mobile. Lihat TEST_RESULTS.
