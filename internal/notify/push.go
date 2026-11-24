package notify

// Langganan Web Push per pengguna+perangkat (infra penyimpanan saja).
//
// KEPUTUSAN (dibaca dulu sebelum ubah):
//   - Paket ini (notify)dipilih sebagai rumah store langganan karena sudah
//     menjadi pusat notifikasi in-app (Create/Unread/List/ReadAll) dan
//     dipakai pengingat via deliver.FromEnv(). Membuat paket baru
//     internal/push hanya untuk 3 fungsi CRUD akan menduplikasi lapisan
//     tanpa manfaat — deliver/ tetap satu-satunya tempat sender keluar
//     (SMTP/Webhook/wa.me) dan TIDAK disentuh file ini.
//   - endpoint disimpan sebagai TEXT TANPA UNIQUE (lihat migrasi 032):
//     URL endpoint push bisa > 255 karakter. Duplikat dicegah di kode:
//     SaveSubscription SELECT user_id+endpoint dulu, lalu UPDATE/INSERT.
//   - TIDAK ada pengiriman di sini. Enkripsi Web Push (RFC 8291: ECDH +
//     HKDF + AES-GCM) butuh pustaka sender yang tidak ada di go.mod dan
//     sengaja TIDAK ditambahkan (tanpa kirim otomatis ke pengguna).
//     Endpoint /push/test hanya mengembalikan payload yang AKAN dikirim
//     + menulis notifikasi in-app sebagai bukti tombol berfungsi.

import (
	"database/sql"
	"errors"
	"net/url"
	"strings"
	"unicode/utf8"
)

// PushSubscription adalah satu langganan browser milik satu pengguna.
type PushSubscription struct {
	ID        int64
	UserID    int64
	Endpoint  string
	P256DH    string
	Auth      string
	UserAgent string
	CreatedAt string
}

// Batas panjang field (anti-abuse; kunci p256dh/auth base64url pendek,
// endpoint URL push umumnya < 1KB).
const (
	maxPushEndpoint = 2048
	maxPushKey      = 512
)

// ValidatePushSubscription menolak langganan yang tidak layak simpan.
// Endpoint wajib https (http hanya diizinkan untuk localhost/127.0.0.1
// agar pengembangan lokal tetap bisa diuji).
func ValidatePushSubscription(endpoint, p256dh, auth string) error {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return errors.New("endpoint wajib diisi")
	}
	if len(endpoint) > maxPushEndpoint {
		return errors.New("endpoint terlalu panjang")
	}
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" {
		return errors.New("endpoint bukan URL valid")
	}
	host := strings.ToLower(u.Hostname())
	local := host == "localhost" || host == "127.0.0.1" || host == "::1"
	if u.Scheme == "https" {
		// ok
	} else if local && u.Scheme == "http" {
		// ok (dev lokal)
	} else {
		return errors.New("endpoint harus https")
	}
	if strings.TrimSpace(p256dh) == "" || strings.TrimSpace(auth) == "" {
		return errors.New("p256dh dan auth wajib diisi")
	}
	if len(p256dh) > maxPushKey || len(auth) > maxPushKey {
		return errors.New("kunci langganan terlalu panjang")
	}
	return nil
}

// SchemaMissing melaporkan apakah err berasal dari tabel yang belum
// dimigrasi (MySQL 1146 / SQLite "no such table"), agar handler dapat
// menjawab 503 yang jujur alih-alih 500 generik.
func SchemaMissing(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "1146") ||
		strings.Contains(msg, "doesn't exist") ||
		strings.Contains(msg, "no such table")
}

// SaveSubscription menyimpan langganan perangkat; idempoten per
// (user_id, endpoint): langganan yang sama hanya memperbarui kunci.
// Mengembalikan jumlah langganan aktif pengguna tersebut.
func SaveSubscription(db *sql.DB, userID int64, endpoint, p256dh, auth, userAgent string) (int64, error) {
	if err := ValidatePushSubscription(endpoint, p256dh, auth); err != nil {
		return 0, err
	}
	userAgent = truncateRunes(strings.TrimSpace(userAgent), 200)
	var id int64
	err := db.QueryRow(`SELECT id FROM push_subscriptions WHERE user_id=? AND endpoint=?`, userID, endpoint).Scan(&id)
	switch {
	case err == nil:
		if _, uerr := db.Exec(`UPDATE push_subscriptions SET p256dh=?, auth=?, user_agent=? WHERE id=?`,
			strings.TrimSpace(p256dh), strings.TrimSpace(auth), userAgent, id); uerr != nil {
			return 0, uerr
		}
	case errors.Is(err, sql.ErrNoRows):
		if _, ierr := db.Exec(`INSERT INTO push_subscriptions(user_id, endpoint, p256dh, auth, user_agent) VALUES(?,?,?,?,?)`,
			userID, endpoint, strings.TrimSpace(p256dh), strings.TrimSpace(auth), userAgent); ierr != nil {
			return 0, ierr
		}
	default:
		return 0, err
	}
	return CountByUser(db, userID)
}

// ListByUser mengembalikan langganan milik pengguna (terbaru dulu).
func ListByUser(db *sql.DB, userID int64) ([]PushSubscription, error) {
	rows, err := db.Query(`SELECT id, user_id, endpoint, p256dh, auth, COALESCE(user_agent,''), created_at
		FROM push_subscriptions WHERE user_id=? ORDER BY id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PushSubscription
	for rows.Next() {
		var s PushSubscription
		if err := rows.Scan(&s.ID, &s.UserID, &s.Endpoint, &s.P256DH, &s.Auth, &s.UserAgent, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// CountByUser menghitung langganan aktif pengguna.
func CountByUser(db *sql.DB, userID int64) (int64, error) {
	var n int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM push_subscriptions WHERE user_id=?`, userID).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// DeleteByEndpoint menghapus satu langganan milik pengguna.
// Mengembalikan jumlah langganan tersisa pengguna tersebut.
func DeleteByEndpoint(db *sql.DB, userID int64, endpoint string) (int64, error) {
	if strings.TrimSpace(endpoint) == "" {
		return 0, errors.New("endpoint wajib diisi")
	}
	if _, err := db.Exec(`DELETE FROM push_subscriptions WHERE user_id=? AND endpoint=?`, userID, endpoint); err != nil {
		return 0, err
	}
	return CountByUser(db, userID)
}

func truncateRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	i := 0
	for idx := range s {
		if i == max {
			return s[:idx]
		}
		i++
	}
	return s
}
