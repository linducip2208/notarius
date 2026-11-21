# DMS — Arsitektur Storage

## 1. Prinsip

- File fisik **tidak pernah** disajikan sebagai file publik.
- Setiap unduhan melewati handler yang melakukan otorisasi.
- MySQL hanya menyimpan `ref`; biner ada di storage privat.
- Bisnis hanya bergantung pada interface — ganti backend tanpa ubah logika.

## 2. Interface (terverifikasi)

`internal/storage/storage.go`:

```go
type Store interface {
    Save(ref string, r io.Reader) error
    Open(ref string) (io.ReadCloser, error)
}
```

## 3. `LocalStore` (implementasi aktif)

- Menyimpan di bawah direktori basis (`NewLocal(base)` membuat dir `0750`).
- `path(ref)`: `filepath.Clean("/"+ref)`, tolak `/` dan ref > 512 char (anti traversal).
- `Save`: buat parent dir `0750`, tulis file `0640`, `O_TRUNC`, batas **100 MB** (`io.LimitReader`).
- `Open`: `os.Open` hasil join aman; error traversal dikembalikan, bukan di-sanitize diam-diam.

## 4. Format `ref`

- String ≤ 512 char, path relatif terhadap basis (bukan URL, bukan path absolut).
- Pembentuk ref ada di lapisan `internal/files` + handler upload; storage tidak menafsirkan maknanya.

## 5. Migrasi ke S3 (tanpa ubah bisnis)

1. Tulis `S3Store` yang memenuhi `Store` (`Save`→PutObject, `Open`→GetObject stream).
2. Registrasi di wiring (`internal/app`) berdasarkan config/env.
3. Migrasi data: copy file lokal → key S3 dengan `ref` yang sama (ref tidak berubah).
4. Validasi: sampling `sha256` lokal vs ETag/objek S3 sebelum switch traffic.
5. Rollback: kembalikan konstruktor ke `NewLocal`; tidak ada perubahan pemanggil.

## 6. Operasional

- Backup: snapshot direktori basis + dump MySQL bersamaan (agar `ref` konsisten).
- Permission: proses hanya butuh RW pada basis; tidak ada kebutuhan exec.
- Kapasitas: pantau ukuran basis; batas 100 MB/file hanya proteksi upload, bukan kuota.
