# Data Concurrency Report
- Repertorium numbering: `SELECT MAX(entry_no)+1 ... FOR UPDATE` + UNIQUE backstop + retry 1062 (020) → uji 8 goroutine sebelumnya PASS (1 menang).
- Bank share download: UPDATE ... WHERE downloads<max AND expires>now (atomik, 020).
- Payment overpay: SELECT FOR UPDATE + money int64 + reject overpay (existing).
- Audit chain: SELECT tail FOR UPDATE + HMAC chain (020).
- Deed approve: row lock + status guard + hash guard + idempotent approve.
- Lockout: count by email+IP (020) + rate limiter login 20/menit.
- Test race detector native tidak dijalankan di Windows/AV (blockerCI Linux).
