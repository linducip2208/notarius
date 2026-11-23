#!/usr/bin/env bash
# Pemicu pengingat harian via endpoint internal. Hanya localhost.
# Kredensial dari environment (systemd EnvironmentFile=/etc/notaryos/reminder.env):
#   REMINDER_USER, REMINDER_PASS, opsional REMINDER_BASE_URL, SESSION_NAME.
# Akun harus office_admin (satu-satunya peran operasional dengan izin
# office.manage) dan TANPA MFA (login script tak bisa melewati TOTP).
set -euo pipefail
BASE="${REMINDER_BASE_URL:-http://127.0.0.1:8080}"
SESS="${SESSION_NAME:-notaryos_sess}"
: "${REMINDER_USER:?REMINDER_USER belum diset (lihat /etc/notaryos/reminder.env)}"
: "${REMINDER_PASS:?REMINDER_PASS belum diset (lihat /etc/notaryos/reminder.env)}"
case "$BASE" in
  http://127.0.0.1*|http://localhost*|"") ;;
  *) echo "run-reminder: menolak BASE non-localhost ($BASE)" >&2; exit 1 ;;
esac
JAR=$(mktemp)
trap 'rm -f "$JAR"' EXIT INT TERM
# 1. Login (POST /login bebas CSRF) — cookie sesi tersimpan di jar.
curl -fsS -c "$JAR" -b "$JAR" \
  --data-urlencode "email=$REMINDER_USER" \
  --data-urlencode "password=$REMINDER_PASS" \
  "$BASE/login" -o /dev/null
if ! grep -q "[[:space:]]${SESS}[[:space:]]" "$JAR"; then
  echo "run-reminder: login gagal (cek user/password; akun MFA tidak didukung)" >&2
  exit 1
fi
# 2. Ambil cookie CSRF dari halaman render mana pun (HttpOnly: baca dari jar).
curl -fsS -c "$JAR" -b "$JAR" "$BASE/app" -o /dev/null
TOKEN=$(awk -v name="${SESS}_csrf" '$6==name{print $7}' "$JAR" | tail -n1)
if [ -z "${TOKEN:-}" ]; then
  echo "run-reminder: cookie CSRF tidak diterbitkan" >&2
  exit 1
fi
# 3. Picu pemindaian (butuh sesi + izin office.manage + field csrf).
curl -fsS -c "$JAR" -b "$JAR" \
  --data-urlencode "csrf=$TOKEN" \
  "$BASE/reminders/run" -o /dev/null
echo "reminder: ok ($(date '+%F %T %Z'))"
