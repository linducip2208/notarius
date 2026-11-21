# Panduan Fitur Operasional (semua ide non-SaaS; bukti: `scripts/qa_shots/ops_growth.py` nihil gagal)

## Klien & CRM
- **Skor kelengkapan** di detail klien + daftar kekurangan (dorong staf melengkapi).
- **Timeline 360**: permohonan, tagihan, pembayaran, janji, berkas, verifikasi identitas.
- **Spesimen** (`/clients/{id}/specimens`): TTD/CAP/STEMPEL gambar + banding visual berdampingan. Tanpa skor otomatis — keputusan manusia.

## Keuangan
- **Rekonsiliasi bank** (`/finance/bank`): impor CSV → automatch nominal sama persis ±0 dalam 30 hari/7 hari; selisih eksplisit; taut manual.
- **Anggaran vs realisasi** per kategori per bulan di Laporan + form simpan.
- **Laba per layanan**: hanya invoice tertaut permohonan (jujur: "tanpa tautan" bila tidak).
- **Invoice tertaut permohonan**: buat dari halaman permohonan (`/finance?req=`), divalidasi klien sama.

## Operasi
- **Inquiry → permohonan**: pilih layanan → buat klien (HP wajib valid) + permohonan; inquiry jadi DIPROSES.
- **Korporat**: peringatan selisih saham/modal, nama tak mirip; struktur kepemilikan + cetak.
- **Akta**: gate pra-pengesahan (nomor + isi ≥50 char + belum APPROVED); paket ZIP (HTML+meta+TTD); taut berkas dua arah; kolom ke-3 template baku.
- **Legalisasi**: nomor antre REGISTERED + estimasi ±2 hari kerja.
- **Gov**: tanggal tindak lanjut resi + taut daftar biaya resmi.
- **Wasiat**: review tahunan + pembukaan bersaksi (notaris + saksi + berita acara); handover tetap terpisah.
- **Tugas**: pengulangan harian/mingguan/bulanan (bergulir saat selesai) + tautan konteks (akta/permohonan/klien/berkas).
- **Retensi → tugas** satu klik; **backup** mysqldump 1 klik (butuh biner server); **mode darurat** `READ_ONLY=1` (kecuali /login).
- **Kalender 14 hari**, **kartu notaris** (review/TTE/macet), **banner kata sandi >180 hari**, **celah repertorium + label QR**.

## Pengingat & keamanan
- Ultah 7 hari, identitas ≤30 hari, review wasiat tahunan, ringkasan pagi per staf, SLA per layanan (`services.sla_days`, default 3).
- Login catat IP/UA; notifikasi "login dari IP baru" (jalur biasa + MFA).
- Kuitansi portal per klien tertaut; notifikasi berkas baru untuk akun portal.

## Batasan jujur
- H-1 WA hanya bila gateway webhook/SMTP dikonfigurasi (`deliver.FromEnv`); tanpa itu tetap in-app.
- Pencarian OCR sudah ada — dipakai ulang, bukan fitur baru.
- Ganti kata sandi mandiri belum ada (kolom `password_changed_at` tercatat; reset via admin).
- Estimasi legalisasi ±2 hari kerja adalah asumsi operasional, bukan janji.
