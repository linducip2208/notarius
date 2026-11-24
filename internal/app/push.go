package app

// Endpoint Web Push — infra + uji manual (TANPA kirim otomatis ke pengguna).
//
//   - POST /push/subscribe   (login)  : simpan langganan perangkat ini.
//   - POST /push/unsubscribe (login)  : hapus langganan perangkat ini.
//   - POST /push/test        (admin, PermManageUsers): kembalikan payload
//     yang AKAN dikirim + tulis notifikasi in-app sebagai bukti tombol
//     berfungsi. JUJUR: tidak ada pengiriman Web Push terenkripsi di sini
//     (butuh pustaka sender RFC 8291 yang sengaja tidak ditambahkan;
//     lihat docs/ops/WEBPUSH.md untuk uji kirim manual via CLI).
//
// CSRF: semua rute ini berada di grup RequireAuth + mw.CSRF. Middleware
// membaca r.FormValue("csrf") yang juga melihat query string, sehingga
// klien fetch JSON mengirim token lewat "?csrf=..." (cookie HttpOnly tak
// bisa dibaca JS). Lihat template settings untuk contoh fetch-nya.

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"

	"notaryos/internal/audit"
	mw "notaryos/internal/middleware"
	"notaryos/internal/notify"
	"notaryos/internal/rbac"
)

// vapidPublicKey dibaca dari env (tidak di-hardcode, tidak di-commit).
func vapidPublicKey() string {
	return strings.TrimSpace(os.Getenv("NOTARYOS_VAPID_PUBLIC"))
}

func writePushJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type pushSubBody struct {
	Endpoint string `json:"endpoint"`
	P256DH   string `json:"p256dh"`
	Auth     string `json:"auth"`
}

func decodePushBody(w http.ResponseWriter, r *http.Request) (pushSubBody, bool) {
	var in pushSubBody
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writePushJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "body JSON tidak valid"})
		return in, false
	}
	return in, true
}

// pushSubscribe menyimpan langganan Web Push perangkat pemanggil.
func (a *App) pushSubscribe(w http.ResponseWriter, r *http.Request) {
	id := a.uid(r)
	if id == nil {
		writePushJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "butuh login"})
		return
	}
	in, ok := decodePushBody(w, r)
	if !ok {
		return
	}
	total, err := notify.SaveSubscription(a.db, *id, in.Endpoint, in.P256DH, in.Auth, r.UserAgent())
	if notify.SchemaMissing(err) {
		writePushJSON(w, http.StatusServiceUnavailable, map[string]any{"ok": false, "error": "skema push_subscriptions belum ada — jalankan migrasi 032"})
		return
	}
	if err != nil {
		writePushJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	a.audit.Record(audit.Entry{ActorID: id, Action: "push.subscribe", Entity: "push_subscription",
		EntityID: strconv.FormatInt(*id, 10), OfficeID: officePtr(a.officeOf(r))})
	writePushJSON(w, http.StatusOK, map[string]any{"ok": true, "total": total})
}

// pushUnsubscribe menghapus langganan Web Push perangkat pemanggil.
func (a *App) pushUnsubscribe(w http.ResponseWriter, r *http.Request) {
	id := a.uid(r)
	if id == nil {
		writePushJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "butuh login"})
		return
	}
	in, ok := decodePushBody(w, r)
	if !ok {
		return
	}
	total, err := notify.DeleteByEndpoint(a.db, *id, in.Endpoint)
	if notify.SchemaMissing(err) {
		writePushJSON(w, http.StatusServiceUnavailable, map[string]any{"ok": false, "error": "skema push_subscriptions belum ada — jalankan migrasi 032"})
		return
	}
	if err != nil {
		writePushJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	a.audit.Record(audit.Entry{ActorID: id, Action: "push.unsubscribe", Entity: "push_subscription",
		EntityID: strconv.FormatInt(*id, 10), OfficeID: officePtr(a.officeOf(r))})
	writePushJSON(w, http.StatusOK, map[string]any{"ok": true, "total": total})
}

// pushTest mengembalikan payload uji + menulis notifikasi in-app.
// Admin saja (PermManageUsers). TIDAK mengirim Web Push terenkripsi —
// field "delivered" selalu false dengan alasan yang eksplisit.
func (a *App) pushTest(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageUsers) {
		a.forbidden(w, r)
		return
	}
	id := a.uid(r)
	if id == nil {
		writePushJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "butuh login"})
		return
	}
	subs, err := notify.ListByUser(a.db, *id)
	if notify.SchemaMissing(err) {
		writePushJSON(w, http.StatusServiceUnavailable, map[string]any{"ok": false, "error": "skema push_subscriptions belum ada — jalankan migrasi 032"})
		return
	}
	if err != nil {
		writePushJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "gagal memuat langganan"})
		return
	}
	payload := map[string]string{
		"title": "Uji Notifikasi — SaktiNotaris",
		"body":  "Bila perangkat Anda menerima ini, langganan Web Push berfungsi.",
		"url":   "/app",
	}
	// Bukti kasatmata bahwa tombol berfungsi: notifikasi in-app (bel header).
	_ = notify.Create(a.db, *id, payload["title"], payload["body"]+" (uji push: pengiriman Web Push terenkripsi belum dilakukan aplikasi)")
	a.audit.Record(audit.Entry{ActorID: id, Action: "push.test", Entity: "push_subscription",
		EntityID: strconv.FormatInt(*id, 10), OfficeID: officePtr(a.officeOf(r)),
		Detail: "langganan=" + strconv.Itoa(len(subs)) + " delivered=false (tanpa sender)"})
	writePushJSON(w, http.StatusOK, map[string]any{
		"ok":               true,
		"delivered":        false,
		"subscriptions":    len(subs),
		"payload":          payload,
		"vapid_configured": vapidPublicKey() != "",
		"reason":           "aplikasi tidak mengirim Web Push terenkripsi (tanpa pustaka sender, sesuai desain) — payload di atas adalah yang AKAN dikirim; cara kirim manual: docs/ops/WEBPUSH.md",
	})
}
