# PAGE DESIGN MATRIX — varian layout per karakteristik kerja

| Halaman | Layout | Alasan |
|---|---|---|
| Landing `/` | Publik 1 kolom, seksi bergantian | Pemasaran, bukan kerja |
| Dashboard `/app` | 8-4: grafik+agenda \| donat+aksi+feed+notif | Monitor + bertindak cepat |
| Daftar (klien, akta, repertorium) | Filter card + tabel + pagination + ekspor | Pindai massal |
| Detail klien | Header profil + tab (Overview/Identitas/Dokumen/Layanan/Aktivitas) | Informasi bertumpuk per konteks |
| Detail akta | 7-5: canvas+revisi+klausul+properti \| versi+komparasi | Fokus dokumen + audit |
| Detail permohonan | 7-5: data+form \| timeline+bukti | Alur + bukti |
| Form (klien baru) | Kartu tunggal + footer aksi kanan | Satu tugas, minim distraksi |
| Keuangan/pengaturan | Kartu ganda + tabel | Ringkasan + rincian |
| Error 403/404 | Kartu tengah | Interupsi jelas, jalan keluar tunggal |

Semua memakai shell yang sama; yang bervariasi hanya komposisi konten.
