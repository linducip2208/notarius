# Custom Domain Guide — DNS + reverse proxy + TLS (jujur, tanpa klaim otomatis)

Aplikasi tidak menerbitkan sertifikat sendiri dan tidak memverifikasi DNS otomatis.
Yang disediakan: penyimpanan konfigurasi + pemeriksaan Host jujur.

## Skenario didukung
1. Domain kantor sendiri (cth. `kantoranda.co.id`) → A record ke IP VPS.
2. Subdomain (cth. `profil.kantoranda.co.id`) → CNAME ke hostname VPS/aplikasi.
3. Domain sama dengan aplikasi (path `/` = website, `/app` = admin).
4. Di belakang reverse proxy (Nginx/Caddy): teruskan `Host` + `X-Forwarded-Proto`.

## Langkah (infrastruktur, dilakukan operator)
1. DNS: A/`AAAA` ke IP server, atau CNAME ke nama yang ditunjuk.
2. Reverse proxy: virtual host untuk domain, `proxy_pass` ke `127.0.0.1:8080`
   dengan `proxy_set_header Host $host;` dan `X-Forwarded-Proto $scheme;`.
   Contoh Nginx ada di `deploy/nginx/` (sesuaikan `server_name`).
3. TLS: sertifikat via ACME/CA (cth. certbot) di proxy; aplikasi menerima HTTP
   dari proxy tepercaya (`TrustedProxies` di `.env`).
4. Di `/cms/site`: isi Domain utama + Base URL `https://…`, centang HTTPS
   (hanya bila langkah 1–3 selesai), centang verifikasi setelah membuka situs
   dari domain tersebut.

## Verifikasi jujur
- Kolom `last_seen_host` mencatat Host terakhir yang mengakses website publik.
- Centang "terverifikasi" adalah **pernyataan manual operator**, bukan hasil
  pemeriksaan otomatis — dokumentasikan tanggalnya di catatan perubahan.
- Bila Host request ≠ domain utama dan bukan localhost/IP privat, aplikasi
  TIDAK redirect otomatis (menghindari open-redirect); tautan kanonis memakai
  base URL yang dikonfigurasi.

## Host header injection
Permintaan dengan Host aneh tidak menentukan canonical URL dan tidak meracuni
cache (cache per kantor + invalidasi eksplisit, bukan per Host).
