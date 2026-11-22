# Cross-Module Workflow Report
Bukti E2E (Playwright + MySQL nyata), skrip di scripts/qa_shots/:

| WF | Alur | Skrip | Hasil |
|---|---|---|---|
| 01 Client→Deed | client + identitas + request + checklist + revisi + formalitas + repertorium | ops_growth.py + fase3.py | PASS (segel + repertorium entry) |
| 02 Client→Finance | invoice → payment → bank match → report | ops_growth.py + fase1a.py | PASS |
| 03 Deed→DMS | draft → attach → version → compare → package ZIP | ops_growth.py | PASS |
| 04 Portal | client user → taut → tagihan → kuitansi | ops_growth.py | PASS |
| 05 Corporate | company → shareholder → BO warning → legal | ops_growth.py | PASS |
| 06 Task/Notification | create → recur → reminder run → dashboard | ops_growth.py + fase1b.py | PASS |
| 07 Retention | hold → disposition → retention→task | ops_growth.py + fase2.py | PASS |

Validation failure & unauthorized diuji lewat matrix.py (8 peran × 26 rute), gate formalitas (400/409), lockout, consent wajib, hash mismatch (409).
