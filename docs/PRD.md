# PRD — NOTARYOS v0.1 (2026-11-20)
## 1. Ringkasan
NOTARYOS adalah sistem manajemen kantor notaris Indonesia: klien, permohonan layanan,
akta & repertorium, protokol, legalisasi/waarmerking, asistensi AHU/OSS, keuangan, portal klien.
Backend 100% Go, MySQL 8.x, UI Tabler + HTMX, modular monolith.

## 2. Tujuan terukur
- Seluruh menu sidebar berfungsi nyata (tanpa tombol palsu).
- Seluruh angka dashboard/laporan berasal dari query database.
- RBAC ditegakkan server-side; 9 peran, 19 izin granular.
- Migrasi MySQL dapat dijalankan ulang secara idempoten.
- `go build`, `go vet`, `go test` hijau.

## 3. Non-tujuan (ditegakkan)
PPAT/AJB/APHT/SKMT, perkara pengadilan, advokat, ERP umum. Tidak ada menu/modul/database PPAT.

## 4. Pengguna
Notaris, notaris pengganti, staf, keuangan, operator dokumen, admin kantor, super admin, klien, auditor.

## 5. Persyaratan fungsional
Lihat `FEATURE_CATALOG.md` (F-001…F-028) — tiap fitur memuat ID, tujuan, peran, prasyarat,
alur, input, validasi, entitas DB, izin, error handling, acceptance, skenario uji, dependensi, prioritas, status.

## 6. Persyaratan non-fungsional
- Keamanan: bcrypt cost 12, sesi HttpOnly+SameSiteLax, CSRF per-form (bertahap), CSP, audit imutabel.
- Data: UTC di DB, WIB di presentasi; rupiah int64; FK + indeks + transaksi.
- Operasi: systemd, Nginx reverse proxy, backup GPG terenkripsi, health/ready endpoints.

## 7. Kriteria rilis
Quality gates §W master command; legal workflow wajib ditinjau notaris Indonesia sebelum produksi.
