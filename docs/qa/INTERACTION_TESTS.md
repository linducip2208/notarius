# INTERACTION TESTS — skenario klik (browser)

| # | Skenario | Langkah | Harapan | Hasil |
|---|---|---|---|---|
| 1 | Login valid | isi email+sandi → Masuk | redirect /app + sapaan nama | |
| 2 | Login salah | sandi salah | error, tetap di /login | |
| 3 | Sidebar aktif | buka tiap menu | item menyala emas | |
| 4 | Quick create | + Buat → tiap item | form tujuan terbuka | |
| 5 | Notif dropdown | buka → tandai dibaca | badge hilang | |
| 6 | Client tab | klik 5 tab | panel berganti, form jalan | |
| 7 | Client filter+sort+page | ubah tiap kontrol | URL + hasil konsisten | |
| 8 | Ekspor CSV | klik | file terunduh, isi benar | |
| 9 | Deed: template+klausul | buat dari template, sisip klausul, simpan revisi | versi +1, hash berubah | |
| 10 | Deed komparasi | klik Bandingkan | dua kolom tampil | |
| 11 | Repertorium filter+cetak | filter + print | hasil + dialog cetak | |
| 12 | Logout | Keluar | sesi mati, /login | |

Status: BELUM DIJALANKAN (butuh sesi browser interaktif).
