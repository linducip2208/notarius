# ASET GAMBAR — ComfyUI + WAN 2.1 (lokal, reproducible)

Generator: `scripts/wan_gen/run.py` (hero 3 PNG) & `feat.py` (8 fitur → JPEG q82).
Model: `wan2.1_t2v_1.3B_fp16` + UMT5 fp8 + VAE WAN · 832×480 · 13 frame · 20 step euler cfg 6 ·
frame tengah (_00007_) · seed tetap.

| File | Prompt inti | Seed | Pakai |
|---|---|---|---|
| notarius-hero.png | meja notaris: timbangan, berkas, pulpen, cahaya pagi | 21 | hero landing |
| notarius-office.png | interior navy-gold, tim review + laptop | 22 | CTA + showcase |
| notarius-sign.png | tangan tanda tangan + segel lilin | 23 | workflow + akta |
| notarius-feat-klien.jpg | verifikasi dokumen identitas di meja | 31 | kartu Klien + banner /clients |
| notarius-feat-akta.jpg | stempel segel emas di akta | 32 | kartu Akta + banner /deeds |
| notarius-feat-repertorium.jpg | rak arsip binder navy-gold | 33 | kartu Repertorium + banner |
| notarius-feat-portal.jpg | laptop+ponsel dashboard malam hari | 34 | kartu Portal + banner /portal |
| notarius-feat-keuangan.jpg | invoice + kalkulator + koin | 35 | kartu Keuangan + banner /finance |
| notarius-feat-laporan.jpg | monitor laporan di kantor navy | 36 | kartu Laporan + banner /reports |
| notarius-feat-keamanan.jpg | gembok kuningan di dokumen berpita | 37 | kartu Keamanan + banner /settings |
| notarius-feat-dokumen.jpg | boks arsip berlabel di rak | 38 | kartu Dokumen + banner /documents |
| notarius-doc-ktp.jpg | kartu identitas + ponsel di meja | 45 | tutorial /docs/ktp-masuk |
| notarius-doc-ocr.jpg | pemindai + berkas + sorot cahaya | 46 | tutorial /docs/ocr-proses |
| notarius-doc-review.jpg | staf membandingkan berkas | 43 | tutorial /docs/review-verifikasi |
| notarius-doc-legal.jpg | akta + segel lilin + timbangan | 44 | tutorial /docs/dokumen-sah |

Lisensi: hasil generate lokal milik kantor. Regenerasi: samakan seed + jalankan skrip.
Catatan: frame tablet portal mengandung teks semu (khas T2V 1.3B) — aman dipakai kecil;
jangan perbesar sebagai hero.
