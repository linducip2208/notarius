# BUSINESS_RULES
1. Akta final (SIGNED/ARCHIVED/terkunci) tidak dapat diubah — hanya revisi resmi baru.
2. Draf akta hanya dari template yang sudah approved notaris.
3. Transisi status permohonan hanya via `transitions` (`internal/requests`); mundur dari COMPLETED dilarang.
4. Invoice: total = subtotal + PPN; pembayaran kumulatif tidak boleh melebihi total; VOID tak bisa dibayar.
5. Uang memakai int64 rupiah di seluruh kode; format `Rp 1.500.000` hanya presentasi.
6. Waktu: UTC di database, Asia/Jakarta di tampilan.
7. NIK tidak pernah plaintext di log/layar penuh (4 digit terakhir untuk pencarian).
8. Hasil OCR/AI tidak pernah auto-approve dokumen kenotariatan — verifikasi manusia wajib.
9. Data klien terisolasi per kantor; peran client hanya datanya sendiri.
10. Tidak ada PPAT dalam bentuk apa pun.
