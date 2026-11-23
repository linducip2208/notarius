# DMS — API & Routes (terverifikasi)

Sumber: `internal/app/app.go` (chi). Hanya rute yang terlihat di kode yang
dicantumkan.

## 1. Kesehatan & publik

| Method | Path | Keterangan |
|---|---|---|
| GET | `/healthz` | liveness, `ok` |
| GET | `/readyz` | readiness (cek DB) |
| GET | `/` | landing |
| GET | `/login` / POST `/login` | login (rate-limit) |
| GET/POST | `/mfa/verify` | verifikasi MFA |

## 2. DMS / berkas (terverifikasi)

| Method | Path | Handler | Auth |
|---|---|---|---|
| POST | `/clients/{id}/files` | `fileUpload` | sesi + cek kantor klien |
| GET | `/files/{id}` | `fileDownload` | sesi + cek `office_id` file |
| POST | `/files/{id}/ocr` | `fileOCR` | sesi + operator |
| POST | `/identities/{id}/verify` | `identityVerify` | `identity.verify` |
| POST | `/clients/{id}/identities` | `identityAdd` | sesi + permission klien |

Semua di dalam grup terautentikasi (`r.Group` + middleware sesi).
Unduhan diotorisasi per-file; tidak ada URL file publik.

## 3. Klien (konteks DMS)

`GET /clients`, `POST /clients`, `GET /clients/new`, `GET /clients/{id}`,
`GET /clients/export`, `POST /clients/{id}/reps`.

## 4. Rute arsip (terverifikasi di `app.go`)

| Method | Path | Handler |
|---|---|---|
| POST | `/documents/{id}/archive` | `dmsArchiveMove` |
| GET | `/dms/archive` | `dmsArchive` |
| POST | `/dms/archive/locations` | `dmsArchiveLoc` |
| GET | `/dms/archive/labels` | `dmsLabels` |
- Bila menambah rute: daftarkan di `app.go`, lindungi dengan middleware
  sesi + RBAC, dan catat audit.
