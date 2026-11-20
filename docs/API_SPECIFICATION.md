# API_SPECIFICATION — REST (Go, chi)
Basis: `http://host:port`. Auth: cookie sesi `notaryos_sess` (login form). JSON dipakai
untuk integrasi; UI memakai form + HTMX.

| Method | Path | Izin | Deskripsi |
|--------|------|------|-----------|
| GET | /healthz, /readyz | publik | Liveness / readiness (cek ping DB) |
| POST | /login (form email,password) | publik | Buat sesi, set cookie HttpOnly |
| POST | /logout | auth | Revoke sesi |
| GET/POST | /clients, /clients/{id} | client.view/create | List (q search, HTMX partial), buat, detail+duplikat |
| GET/POST | /requests, POST /requests/{id}/move | auth | List (filter status), buat DRAFT, transisi legal |
| GET/POST | /deeds | deed.draft | List, buat draf (template approved bila dipakai) |
| GET | /repertorium, /protocols | repertorium.manage/protocol.view | Register baca |
| GET/POST | /legalisasi | legalization.process | Register + biaya |
| GET | /gov | government.submit | Daftar pengajuan asistensi AHU/OSS |
| GET/POST | /invoices, POST /payments | finance.view/manage | Invoice (PPN otomatis), pembayaran anti-overpay |
| GET | /templates, /reports, /settings, /portal | sesuai izin | Template, laporan real-query, pengguna, portal klien |

Error: 400 validasi, 403 izin, 404 scope kantor, 503 DB belum siap. Semua respons error
berbahasa Indonesia yang jelas.
