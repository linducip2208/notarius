# UI/UX GUIDELINES — NOTARIUS (ringkas, mengikat)

1. Hierarki: judul halaman + subtitle konteks (kantor · tanggal WIB · ringkasan) di setiap halaman kerja.
2. Aksi primer satu per halaman (btn-primary/warning); sekunder ghost/outline; destruktif + confirm.
3. Spasi: kartu `row-deck`, gap-2/3; tabel `table-vcenter`; mobile: form 1 kolom, tabel scroll.
4. Tipografi: Tabler Inter bawaan; angka statistik `h1`; label kecil uppercase hanya di thead.
5. Status selalu badge + Bahasa Indonesia; tidak ada enum mentah ke pengguna.
6. Setiap halaman: breadcrumb, empty-state jujur, pesan error spesifik, tanpa tombol mati.
7. Form: grup per seksi, required jelas, CSRF, audit untuk mutasi penting.
8. Grafik: legend + tooltip (title SVG) + filter periode + empty-state; angka dari query.
9. Motion: transisi Tabler bawaan; hormati prefers-reduced-motion (tokens.css).
10. Aksesibilitas: label form, alt gambar, aria-label grafik/tabel, kontras Tabler default, fokus keyboard bawaan.
