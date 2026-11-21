# DMS — Panduan Pengguna (staf & notaris)

Antarmuka: Tabler + Go templates (server-side). Tidak ada SPA.

## 1. Mengunggah dokumen klien

1. Buka `Klien → detail klien`.
2. Bagian berkas → `Unggah` → pilih file (maks 100 MB/file).
3. Pastikan nama file deskriptif (mis. `ktp-nama.jpg`) — membantu saran klasifikasi.
4. Setelah unggah, status awal = `PENDING`.

## 2. Menjalankan OCR

1. Pada berkas berstatus `PENDING`, pilih `Proses OCR`.
2. Tunggu hingga `DONE` (atau `FAILED` bila engine/gambar bermasalah).
3. `FAILED` → periksa: engine Tesseract `ind` terinstal? gambar cukup jelas?
   Bila perlu, unggah ulang hasil pindaian yang lebih tajam.

## 3. Verifikasi hasil (wajib)

1. Buka hasil OCR; bandingkan tiap field dengan dokumen asli.
2. Koreksi yang salah; klasifikasi (`KTP/KK/NPWP/OTHER`) boleh diubah.
3. Tekan `Verifikasi`. Tanpa langkah ini data tidak boleh dipakai
   untuk identitas/akta.

## 4. Mengunduh & menautkan

- Unduh via tombol pada baris berkas (akses dicatat audit).
- Berkas tertaut ke pemiliknya (klien/layanan); unduhan lintas kantor ditolak.

## 5. Bantuan cepat

| Gejala | Tindakan |
|---|---|
| Upload ditolak (>100 MB) | kecilkan/pecah file, atau pindai ulang resolusi wajar |
| OCR `FAILED` | coba ulang sekali; bila tetap, isi manual + catat |
| Hasil OCR meragukan | jangan verifikasi; minta dokumen/scan ulang |
| Butuh arsip lama | minta akses `protocol.view`; unduhan diaudit |
