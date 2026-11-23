// Package deliver — tambahan sender manual via tautan wa.me.
//
// KEPUTUSAN (dibaca dulu sebelum ubah):
//   (a) SMTP: kode deliver.SMTP + deliver.FromEnv() SUDAH ADA dan dipakai
//       remind.RunOnce via FromEnv(). Bila env NOTARYOS_SMTP_HOST (dkk) diisi,
//       pengingat in-app otomatis dihandoff ke relay SMTP — tanpa ubah kode.
//   (b) Bila env SMTP KOSONG (kondisi repo saat ini: .env/.env.example tidak
//       mendefinisikan NOTARYOS_SMTP_*), TIDAK ada pengiriman otomatis yang
//       diklaim. Sebagai gantinya file ini menyediakan helper tautan WA manual:
//       staf menekan tombol "Kirim pengingat WA" di halaman permohonan,
//       browser membuka chat wa.me dengan nomor + teks sudah terisi, staf
//       menekan Kirim di WhatsApp. Jujur: bukan auto-send, tidak butuh API
//       berbayar (Fonnte/Wablas/dsb), tidak menyimpan kredensial.
//
// CARA AKTIFKAN:
//   - WA manual: tanpa env, langsung pakai tombol di halaman permohonan
//     (butuh data clients.phone / clients.wa_number terisi).
//   - Email otomatis: isi di .env (lihat .env.example):
//       NOTARYOS_SMTP_HOST, NOTARYOS_SMTP_PORT (default 587),
//       NOTARYOS_SMTP_USER, NOTARYOS_SMTP_PASS, NOTARYOS_SMTP_FROM.
//   - Gateway WA otomatis (berbayar/opsional): isi NOTARYOS_NOTIFY_WEBHOOK
//     (+ NOTARYOS_NOTIFY_SECRET) dengan URL https publik; payload JSON
//     bertanda HMAC dikirim best-effort oleh remind.RunOnce.
package deliver

import (
	"net/url"
	"strings"
	"unicode"
)

// NormalizePhoneForWA mengubah nomor lokal menjadi digit internasional tanpa '+'.
// Aturan: buang non-digit; "08…" -> "628…"; "+62…"/"62…" tetap "62…".
// Mengembalikan "" bila tidak layak tautan (< 9 digit).
func NormalizePhoneForWA(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else if r == '+' {
			continue // '+' dibuang, wa.me memakai digit polos
		} else if unicode.IsSpace(r) || r == '-' || r == '.' || r == '(' || r == ')' || r == '/' {
			continue // pemisah umum diabaikan
		} else {
			continue // karakter lain diabaikan
		}
	}
	d := b.String()
	if strings.HasPrefix(d, "0") {
		d = "62" + d[1:]
	}
	if len(d) < 9 {
		return ""
	}
	return d
}

// PreferWA memilih nomor WA di atas telepon biasa (keduanya mentah dari DB).
func PreferWA(waNumber, phone string) string {
	if n := NormalizePhoneForWA(waNumber); n != "" {
		return n
	}
	return NormalizePhoneForWA(phone)
}

// ReminderText menyusun isi chat pengingat (judul + badan notifikasi in-app).
func ReminderText(title, body string) string {
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if title == "" {
		return body
	}
	if body == "" {
		return title
	}
	return title + "\n" + body
}

// WALink membangun tautan https://wa.me/<digit>?text=<pesan>.
// Mengembalikan "" bila nomor tidak valid — template wajib menyembunyikan tombol.
func WALink(phoneRaw, text string) string {
	digits := NormalizePhoneForWA(phoneRaw)
	if digits == "" {
		return ""
	}
	u := "https://wa.me/" + digits
	if text = strings.TrimSpace(text); text != "" {
		u += "?text=" + url.QueryEscape(text)
	}
	return u
}
