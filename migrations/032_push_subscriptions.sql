-- 032_push_subscriptions.sql — langganan Web Push per pengguna+perangkat (infra saja).
-- Satu pengguna dapat punya banyak langganan (satu per perangkat/browser).
-- endpoint disimpan sebagai TEXT TANPA UNIQUE: URL endpoint push bisa > 255
-- karakter sehingga UNIQUE(255) tidak aman; duplikat dicegah di kode
-- (notify.SaveSubscription: SELECT user_id+endpoint dulu, lalu UPDATE/INSERT).
-- Idempoten: CREATE TABLE IF NOT EXISTS (runner menoleransi DDL duplikat
-- 1060/1061/1050/1068/1091 untuk rerun parsial).
-- CATATAN JUJUR: tabel ini hanya menyimpan langganan. Pengiriman Web Push
-- (enkripsi RFC 8291) TIDAK dilakukan aplikasi — tidak ada dependensi sender
-- di go.mod. Lihat docs/ops/WEBPUSH.md untuk cara uji kirim manual via CLI.
CREATE TABLE IF NOT EXISTS push_subscriptions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT UNSIGNED NOT NULL,
  endpoint TEXT NOT NULL,
  p256dh TEXT NOT NULL,
  auth TEXT NOT NULL,
  user_agent VARCHAR(255) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  INDEX idx_pushsub_user (user_id)
);
