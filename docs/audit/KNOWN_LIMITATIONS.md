# KNOWN_LIMITATIONS — v0.1
1. Butuh MySQL 8.x berjalan; tanpa DB aplikasi menolak start (by design, bukan crash misterius).
2. Sesi in-memory: restart menghapus login; single-instance saja (TD-01).
3. OCR/AI eksternal nonaktif hingga engine/kunci tersedia (B-01/B-02).
4. Belum ada ekspor PDF/DOCX (F-021), rate-limit/CSRF/MFA (B-05).
5. Portal klien belum terisolasi per-client di query (F-026 parsial).
6. Migrasi rollback = restore backup (tanpa skrip down; disengaja agar riwayat protokol tak hilang diam-diam).
7. Alur hukum BELUM ditinjau notaris — jangan pakai untuk akta sungguhan sebelum §W terpenuhi.
