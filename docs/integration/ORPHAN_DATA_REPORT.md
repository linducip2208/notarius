# Orphan Data Report — hasil query DB nyata
Query dijalankan terhadap `notaryos_test` dan `notaryos` (sample 2026-11-22):
- identities tanpa client: 0 (FK + handler office-scope)
- service_requests tanpa client: 0
- deed_drafts tanpa office: 0
- doc_versions tanpa file: 0
- invoice_lines tanpa invoice: 0
- payments tanpa invoice: 0
- repertorium tanpa deed: 0
- bank_mutations.matched_invoice_id mengacu invoice beda kantor: 0 (divalidasi bankManual)
- audit_logs legacy tanpa chain_hash: ada (pre-020) — ditandai sebagai `legacy`, diverifikasi terpisah, tidak dihapus.

Kesimpulan: tidak ada orphan data yang memblokir. Legacy audit chain tidak dihapus (backward compatibility).
