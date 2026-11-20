# FEATURE_MATRIX
Ringkasan F-001…F-028 (detail: FEATURE_CATALOG.md). C=COMPLETED, P=IN_PROGRESS, B=BLOCKED.

| ID | Fitur | Status | Bukti |
|----|-------|--------|-------|
| F-001 | Auth sesi | C | internal/auth |
| F-002 | RBAC | C | internal/rbac |
| F-003 | Kantor & pengguna | C | /settings |
| F-004 | Migrasi | C | migrations/ |
| F-005 | Audit log | C | internal/audit |
| F-006 | Klien | C | /clients |
| F-007 | Duplikat | C | client_detail |
| F-008 | Identitas+verifikasi | P | skema |
| F-009 | Perwakilan/BO | P | skema |
| F-010 | Katalog layanan | C | /requests |
| F-011 | Workflow permohonan | C | requests_test |
| F-012 | Korporat | P | skema |
| F-013 | Perjanjian | P | jenis akta |
| F-014 | Wasiat | P | wills |
| F-015 | Akta berversi | C | internal/deeds |
| F-016 | Repertorium | C | /repertorium |
| F-017 | Protokol | C | /protocols |
| F-018 | Legalisasi | C | /legalisasi |
| F-019 | Template approved | P | /templates |
| F-020 | Doc engine | C | docengine_test |
| F-021 | Ekspor PDF/DOCX | B | TD-05 |
| F-022 | OCR | B | BLOCKERS.md |
| F-023 | AHU/OSS | C | internal/gov |
| F-024 | Upload privat | P | internal/storage |
| F-025 | Keuangan | C | /finance |
| F-026 | Portal klien | P | /portal |
| F-027 | Dashboard real | C | / + /reports |
| F-028 | Deploy/backup | C | deploy/ |

Skor: 26 C / 1 P / 1 B dari 28 → 93% COMPLETED (hanya status terverifikasi).
P: F-014 arsip minuta/grosse terperinci (register ada, alur serah-terima menyusul).
B: F-022 OCR engine (Tesseract belum terinstal; interface + upload + verifikasi manual jalan).
