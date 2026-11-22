# Dashboard Data Consistency
Widget /app dan /reports berasal dari query MySQL real-time:
- Total klien (clients), permohonan aktif (reqs.Counts), akta bulan ini (deed_drafts), jadwal hari ini (appointments), piutang (invoices).
- Kartu notaris: deed REVIEW/REVISION, APPROVED siap TTE, SLA macet (services.sla_days).
- Laporan: anggaran vs realisasi (budgets+expenses), pendapatan per layanan (invoices→request→services).
Tidak ada angka hardcoded pada widget. Semua tanggal operasional diformat WIB via helper `wib`; timestamp DB UTC konsisten (CONVERT_TZ di query lama).
