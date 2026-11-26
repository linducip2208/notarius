#!/usr/bin/env bash
# NOTARYOS — installer systemd service untuk Linux (Ubuntu 22.04/24.04, Debian 12).
# Cara pakai di VPS (sebagai root):
#   cd /opt/notaryos
#   sudo bash deploy/systemd/install.sh
# Opsi env:
#   APP_DIR=/opt/notaryos SERVICE_USER=notaryos ENV_FILE=/etc/notaryos/env sudo -E bash deploy/systemd/install.sh
#   SKIP_OCR=1  -> lewati instalasi Tesseract OCR (fitur OCR nonaktif, app tetap jalan)
#   OCR_LANGS="tesseract-ocr-ind tesseract-ocr-eng" -> paket bahasa tambahan
set -euo pipefail

APP_DIR="${APP_DIR:-/opt/notaryos}"
SERVICE_USER="${SERVICE_USER:-notaryos}"
ENV_FILE="${ENV_FILE:-/etc/notaryos/env}"
SKIP_OCR="${SKIP_OCR:-0}"
OCR_LANGS="${OCR_LANGS:-tesseract-ocr-ind tesseract-ocr-eng}"
OCR_BINARY_PATH="$(command -v tesseract 2>/dev/null || echo /usr/bin/tesseract)"
REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

echo "==> NOTARYOS systemd installer"
echo "    repo : $REPO_DIR"
echo "    app  : $APP_DIR"
echo "    user : $SERVICE_USER"
echo "    env  : $ENV_FILE"

if [[ $EUID -ne 0 ]]; then
  echo "ERROR: jalankan sebagai root (sudo)." >&2
  exit 1
fi

# 1. User sistem
if ! id "$SERVICE_USER" >/dev/null 2>&1; then
  echo "==> membuat user $SERVICE_USER"
  useradd -r -m -s /usr/sbin/nologin "$SERVICE_USER"
else
  echo "==> user $SERVICE_USER sudah ada"
fi

# 2. Direktori + biner
if [[ "$REPO_DIR" != "$APP_DIR" ]]; then
  echo "==> sinkron repo -> $APP_DIR (tanpa .git, .tmp, storage)"
  mkdir -p "$APP_DIR"
  if command -v rsync >/dev/null 2>&1; then
    rsync -a --delete \
      --exclude='.git/' --exclude='.tmp/' --exclude='storage/' --exclude='bin/' \
      "$REPO_DIR/" "$APP_DIR/"
  else
    cp -a "$REPO_DIR/." "$APP_DIR/"
  fi
fi

echo "==> build biner"
cd "$APP_DIR"
if ! command -v go >/dev/null 2>&1; then
  echo "ERROR: Go belum terinstal. Instal Go >= 1.24 dulu (lihat README.md)." >&2
  exit 1
fi
go build -trimpath -o bin/notaryos ./cmd/notaryos

# 2b. OCR — Tesseract lokal (KTP/KK/NPWP: -l ind / ind+eng).
#     Tanpa ini app tetap jalan, tapi /dms/ocr & tombol "Jalankan OCR" nonaktif
#     (log: "ocr disabled", provider NoopProvider). Sama seperti deploy/Dockerfile.
if [[ "$SKIP_OCR" == "1" ]]; then
  echo "==> OCR dilewati (SKIP_OCR=1)"
else
  echo "==> OCR: pastikan tesseract + data bahasa ind+eng"
  if ! command -v tesseract >/dev/null 2>&1; then
    echo "    tesseract belum ada -> instal via apt"
    export DEBIAN_FRONTEND=noninteractive
    apt-get update -y
    # shellcheck disable=SC2086
    apt-get install -y --no-install-recommends tesseract-ocr $OCR_LANGS
  else
    echo "    tesseract sudah ada: $(command -v tesseract) ($(tesseract --version 2>&1 | head -n1))"
    # Lengkapi paket bahasa bila belum ada (idempoten)
    MISSING_LANGS=""
    for pkg in $OCR_LANGS; do
      if ! dpkg -s "$pkg" >/dev/null 2>&1; then
        MISSING_LANGS="$MISSING_LANGS $pkg"
      fi
    done
    if [[ -n "$MISSING_LANGS" ]]; then
      echo "    melengkapi paket bahasa:$MISSING_LANGS"
      export DEBIAN_FRONTEND=noninteractive
      apt-get update -y
      # shellcheck disable=SC2086
      apt-get install -y --no-install-recommends $MISSING_LANGS
    fi
  fi
  OCR_BINARY_PATH="$(command -v tesseract || echo /usr/bin/tesseract)"
  echo "    verifikasi bahasa:"
  tesseract --list-langs || echo "    WARN: 'tesseract --list-langs' gagal — cek instalasi tessdata"
fi

mkdir -p "$APP_DIR/storage"
chown -R "$SERVICE_USER:$SERVICE_USER" "$APP_DIR/storage"
chown -R root:root "$APP_DIR/bin" "$APP_DIR/migrations" "$APP_DIR/web" 2>/dev/null || true
chmod +x "$APP_DIR/bin/notaryos"

# 3. File env /etc/notaryos/env (0600)
echo "==> env $ENV_FILE"
mkdir -p "$(dirname "$ENV_FILE")"
if [[ ! -f "$ENV_FILE" ]]; then
  if [[ -f "$APP_DIR/.env.example" ]]; then
    cp "$APP_DIR/.env.example" "$ENV_FILE"
  else
    cat > "$ENV_FILE" <<'EOF'
PORT=8080
ADDR=:8080
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=notaryos
DB_PASS=change-me
DB_NAME=notaryos
DB_TLS=false
COOKIE_SECURE=false
STORAGE_DIR=/opt/notaryos/storage
APP_BASE_URL=
EOF
  fi
  echo "    PENTING: edit $ENV_FILE (DB_PASS, NOTARYOS_FIELD_KEY, dll)"
else
  echo "    sudah ada, tidak ditimpa"
fi
# Pastikan OCR_BINARY terisi (jangan timpa bila admin sudah set manual)
if [[ "$SKIP_OCR" != "1" ]]; then
  if grep -qE '^OCR_BINARY=$' "$ENV_FILE"; then
    sed -i "s|^OCR_BINARY=$|OCR_BINARY=$OCR_BINARY_PATH|" "$ENV_FILE"
    echo "    OCR_BINARY=$OCR_BINARY_PATH ditulis ke $ENV_FILE"
  elif ! grep -qE '^OCR_BINARY=' "$ENV_FILE"; then
    echo "OCR_BINARY=$OCR_BINARY_PATH" >> "$ENV_FILE"
    echo "    OCR_BINARY=$OCR_BINARY_PATH ditambahkan ke $ENV_FILE"
  else
    echo "    OCR_BINARY sudah diset manual, tidak diubah ($(grep -E '^OCR_BINARY=' "$ENV_FILE"))"
  fi
fi
chown root:"$SERVICE_USER" "$ENV_FILE"
chmod 0600 "$ENV_FILE"

# 4. Pasang unit systemd
echo "==> pasang unit systemd"
cp "$REPO_DIR/deploy/systemd/notaryos.service" /etc/systemd/system/notaryos.service
if [[ -f "$REPO_DIR/deploy/systemd/notaryos-reminder.service" ]]; then
  cp "$REPO_DIR/deploy/systemd/notaryos-reminder.service" /etc/systemd/system/
fi
if [[ -f "$REPO_DIR/deploy/systemd/notaryos-reminder.timer" ]]; then
  cp "$REPO_DIR/deploy/systemd/notaryos-reminder.timer" /etc/systemd/system/
fi
chmod 644 /etc/systemd/system/notaryos*.service /etc/systemd/system/notaryos*.timer 2>/dev/null || true

systemctl daemon-reload
systemctl enable notaryos.service
# Timer pengingat opsional — aktifkan bila sudah isi /etc/notaryos/reminder.env
if [[ -f /etc/notaryos/reminder.env ]]; then
  systemctl enable --now notaryos-reminder.timer || true
fi

echo "==> restart service"
systemctl restart notaryos.service || {
  echo "ERROR: service gagal start. Cek: journalctl -u notaryos -xe" >&2
  exit 1
}

sleep 2
systemctl --no-pager --full status notaryos.service || true
echo
echo "==> verifikasi OCR:"
if [[ "$SKIP_OCR" == "1" ]]; then
  echo "    dilewati (SKIP_OCR=1). Aktifkan nanti: apt-get install -y tesseract-ocr tesseract-ocr-ind tesseract-ocr-eng"
  echo "    lalu isi OCR_BINARY=/usr/bin/tesseract di $ENV_FILE + systemctl restart notaryos"
else
  "$OCR_BINARY_PATH" --list-langs || echo "    WARN: tesseract --list-langs gagal"
  echo "    log provider: journalctl -u notaryos | grep -i 'ocr provider'"
fi
echo
echo "==> SELESAI. Perintah berguna:"
echo "    systemctl status notaryos"
echo "    journalctl -u notaryos -f"
echo "    curl -s http://127.0.0.1:8080/readyz ; echo"
echo "    curl -s http://127.0.0.1:8080/healthz ; echo"
