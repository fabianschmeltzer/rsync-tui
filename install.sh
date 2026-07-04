#!/usr/bin/env sh
set -eu

REPOSITORY="fabianschmeltzer/rsync-tui"
VERSION="${VERSION:-v0.1.0}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
NO_SYSTEMD="${NO_SYSTEMD:-0}"

case "$(uname -s)" in
    Linux) ;;
    *)
        echo "rsync-tui currently supports Linux only." >&2
        exit 1
        ;;
esac

case "$(uname -m)" in
    x86_64|amd64) TARGET="amd64" ;;
    aarch64|arm64) TARGET="arm64" ;;
    armv7l|armv7) TARGET="armv7" ;;
    *)
        echo "Unsupported architecture: $(uname -m)" >&2
        exit 1
        ;;
esac

command -v curl >/dev/null 2>&1 || {
    echo "curl is required." >&2
    exit 1
}

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT HUP INT TERM

ARCHIVE="rsync-tui_linux_${TARGET}.tar.gz"
BASE_URL="https://github.com/${REPOSITORY}/releases/download/${VERSION}"

echo "Downloading rsync-tui ${VERSION} for linux/${TARGET}..."
curl --fail --location --proto '=https' --tlsv1.2 \
    --output "$TMP_DIR/$ARCHIVE" "$BASE_URL/$ARCHIVE"
curl --fail --location --proto '=https' --tlsv1.2 \
    --output "$TMP_DIR/SHA256SUMS" "$BASE_URL/SHA256SUMS"

EXPECTED="$(grep " $ARCHIVE\$" "$TMP_DIR/SHA256SUMS" | awk '{print $1}')"
[ -n "$EXPECTED" ] || {
    echo "Release checksum is missing." >&2
    exit 1
}

if command -v sha256sum >/dev/null 2>&1; then
    ACTUAL="$(sha256sum "$TMP_DIR/$ARCHIVE" | awk '{print $1}')"
else
    ACTUAL="$(shasum -a 256 "$TMP_DIR/$ARCHIVE" | awk '{print $1}')"
fi

[ "$EXPECTED" = "$ACTUAL" ] || {
    echo "Checksum verification failed." >&2
    exit 1
}

tar -xzf "$TMP_DIR/$ARCHIVE" -C "$TMP_DIR"
mkdir -p "$INSTALL_DIR"
install -m 0755 "$TMP_DIR/rsync-tui" "$INSTALL_DIR/rsync-tui"

case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *) echo "Add $INSTALL_DIR to PATH." ;;
esac

if [ "$NO_SYSTEMD" != "1" ] && command -v systemctl >/dev/null 2>&1; then
    UNIT_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
    mkdir -p "$UNIT_DIR"
    cat > "$UNIT_DIR/rsync-tui-update.service" <<EOF
[Unit]
Description=Automatically update rsync-tui
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=$INSTALL_DIR/rsync-tui update
EOF
    cat > "$UNIT_DIR/rsync-tui-update.timer" <<'EOF'
[Unit]
Description=Daily rsync-tui update check

[Timer]
OnCalendar=daily
Persistent=true
RandomizedDelaySec=30m

[Install]
WantedBy=timers.target
EOF
    systemctl --user daemon-reload
    systemctl --user enable --now rsync-tui-update.timer >/dev/null
fi

echo "Installed: $INSTALL_DIR/rsync-tui"
echo "Run: rsync-tui doctor"
