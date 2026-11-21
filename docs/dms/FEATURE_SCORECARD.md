# DMS — Scorecard Fitur (skor awal konservatif)

Skala 0–5 (0 belum ada, 3 layak-pakai-dengan-caveat, 5 terbukti produksi).
Skor konservatif: klaim tertinggi 3 karena verifikasi browser/konkuren
masih terbuka (lihat `docs/audit/`).

| Fitur | Skor | Alasan |
|---|---|---|
| Upload privat + metadata (`files`) | 3 | kode + migrasi ada; uji spoof/traversal lintas-kantor belum dieksekusi |
| Unduh otorisasi per-file | 3 | handler ada; matriks 403/404 vs bocor belum diuji visual |
| Abstraksi `storage.Store` (siap-S3) | 3 | interface + `LocalStore` nyata; `S3Store` belum ada (by design) |
| OCR Tesseract `ind` + `ai_results` | 2 | kontrak + status ada; butuh Tesseract host + uji `DONE/FAILED` nyata |
| Verifikasi manusia wajib | 2 | kolom `verified_*` ada; enforcement ujung-ke-ujung belum dibuktikan |
| Klasifikasi heuristik | 2 | `Classify` hanya nama file; advis, mudah salah |
| Audit imutabel `audit_logs` | 3 | append-only di kode; uji attempt UPDATE/DELETE + review log belum |
| RBAC + isolasi kantor | 2 | role/permission seed ada; office-scope files/deeds baru, perlu uji cross-office |
| Pencarian metadata MySQL | 2 | query dasar mungkin; tanpa FTS/indeks khusus, performa tak terjamin |
| Retensi (`retention_policies`) | 2 | seed benar (25/25/10/10); penegakan siklus arsip masih manual |
| UI Tabler + templates | 3 | 100% Tabler; uji peran/browser final belum |
| Backup/restore pasangan | 1 | prosedur didokumentasikan; belum pernah dieksekusi tercatat |

## Naik skor bila

- TEST_REPORT.md terisi penuh dari eksekusi nyata (bukan estimasi).
- Temuan `docs/audit/` (UNIQUE sha, FK polimorfik, MIME sniff, XFF) diuji dan ditutup.
- Uji konkuren penomoran + cross-office lolos dengan bukti log.
