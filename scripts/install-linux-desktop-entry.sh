#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_PATH="${ROOT_DIR}/build/bin/inventory-desktop"
ICON_SRC="${ROOT_DIR}/build/appicon.png"
APP_ID="myduka"
DESKTOP_DIR="${HOME}/.local/share/applications"
ICON_DIR="${HOME}/.local/share/icons/hicolor/512x512/apps"
DESKTOP_FILE="${DESKTOP_DIR}/${APP_ID}.desktop"
ICON_DST="${ICON_DIR}/${APP_ID}.png"

if [[ ! -x "${BIN_PATH}" ]]; then
  echo "Binary not found or not executable: ${BIN_PATH}" >&2
  echo "Build first: GOCACHE=/tmp/go-build wails build -s -nopackage" >&2
  exit 1
fi

mkdir -p "${DESKTOP_DIR}" "${ICON_DIR}"
cp "${ICON_SRC}" "${ICON_DST}"

cat > "${DESKTOP_FILE}" <<DESKTOP
[Desktop Entry]
Version=1.0
Type=Application
Name=MyDuka
Comment=Local-first POS and inventory app
Exec=${BIN_PATH}
Icon=${APP_ID}
Terminal=false
Categories=Office;
StartupWMClass=${APP_ID}
DESKTOP

chmod +x "${DESKTOP_FILE}"

if command -v update-desktop-database >/dev/null 2>&1; then
  update-desktop-database "${DESKTOP_DIR}" >/dev/null 2>&1 || true
fi
if command -v gtk-update-icon-cache >/dev/null 2>&1; then
  gtk-update-icon-cache -f -q "${HOME}/.local/share/icons/hicolor" >/dev/null 2>&1 || true
fi

echo "Installed launcher: ${DESKTOP_FILE}"
echo "Search for 'MyDuka' in app launcher and pin it to Dock/Favorites."
