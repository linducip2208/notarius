# Web Push — infra + uji manual (tanpa kirim otomatis)

Status jujur: aplikasi menyimpan langganan browser (`push_subscriptions`,
migrasi `032`) dan menyediakan tombol uji di `/settings`, tetapi **tidak
mengirim Web Push terenkripsi**. Enkripsi RFC 8291 (ECDH + HKDF + AES-GCM)
butuh pustaka sender yang sengaja tidak ditambahkan ke `go.mod` (keputusan:
tanpa kirim otomatis ke pengguna). Endpoint `POST /push/test` (admin saja)
mengembalikan payload yang **akan** dikirim + menulis notifikasi in-app
sebagai bukti tombol berfungsi (`"delivered": false` eksplisit di respons).

## 1. Hasilkan kunci VAPID (sekali, jangan commit)

```sh
npx -y web-push generate-vapid-keys
```

Isi di `.env` (lihat `.env.example`):

```sh
NOTARYOS_VAPID_PUBLIC=<public key>
NOTARYOS_VAPID_PRIVATE=<private key — rahasia server saja>
NOTARYOS_VAPID_SUBJECT=mailto:kantor@contoh.id
```

Tanpa `NOTARYOS_VAPID_PUBLIC`, tombol "Aktifkan notifikasi" di `/settings`
nonaktif dengan penjelasan (tidak pura-pura bisa).

## 2. Alur normal

1. Pengguna login → `/settings` → "Aktifkan notifikasi" → browser minta
   izin → langganan (`endpoint`, `p256dh`, `auth`) tersimpan via
   `POST /push/subscribe` (butuh login; CSRF via `?csrf=` karena cookie
   HttpOnly tak bisa dibaca JS).
2. Admin menekan "Kirim uji" → `POST /push/test` → respons berisi jumlah
   langganan + payload preview + alasan tidak dikirim; notifikasi in-app
   (bel header) ditulis sebagai bukti kasatmata.
3. "Nonaktifkan notifikasi" → unsubscribe browser + `POST /push/unsubscribe`.

Service worker (`web/assets/sw.js`) sudah menangani event `push` dengan
payload JSON `{title, body, url}` dan membuka `/app` saat diklik.

## 3. Uji kirim manual (butuh gateway/CLI)

Ambil satu langganan milik sendiri dari DB, lalu kirim memakai CLI
`web-push` (kunci privat tidak pernah keluar dari server/CLI tepercaya):

```sh
npx -y web-push send-notification \
  --endpoint='<endpoint dari DB>' \
  --key='<p256dh dari DB>' \
  --auth='<auth dari DB>' \
  --vapid-subject="$NOTARYOS_VAPID_SUBJECT" \
  --vapid-pubkey="$NOTARYOS_VAPID_PUBLIC" \
  --vapid-pvtkey="$NOTARYOS_VAPID_PRIVATE" \
  --payload='{"title":"Uji Notifikasi — SaktiNotaris","body":"Langganan Web Push berfungsi.","url":"/app"}'
```

## 4. Bila nanti butuh kirim otomatis

Tambahkan pustaka sender Web Push Go (mis. `github.com/SherClockHolmes/webpush-go`)
+ antrean best-effort ala `deliver.FromEnv()` (tidak menggagalkan alur
utama), dengan kunci privat **hanya** dari env. Sampai saat itu, klaim
"terkirim" tidak boleh muncul di UI maupun respons API.
