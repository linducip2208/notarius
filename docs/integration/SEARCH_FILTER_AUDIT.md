# Search & Filter Audit
- /search: client LIKE (parameterized), permission staff.
- /dms/library & /dms/search/adv: filter MIME/tanggal/workflow/confidentiality/sort, pagination stabil, confidentiality filter server-side.
- /cms/* filter, /cari (artikel/halaman/layanan/FAQ), /arsip (bulan).
- /doc_search_index FULLTEXT OCR (MATCH…AGAINST) + LIKE fallback, RESTRICTED dikecualikan untuk peran tak berhak.
- SQL injection: tidak ada konkatenasi input ke ORDER BY (whitelist). LIKE memakai parameter.
- Reset filter: query string baru; empty state tersedia.
