#!/bin/bash
#
# unas-custom installer
# Usage: curl -fsSL https://raw.githubusercontent.com/bbetter173/unas-exportfs-wrapper/main/scripts/get.sh | bash
#
set -euo pipefail

REPO="bbetter173/unas-exportfs-wrapper"
INSTALL_DIR="/persistent/unas-custom"

info()  { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn()  { printf '\033[1;33mWARN:\033[0m %s\n' "$*"; }
error() { printf '\033[1;31mERROR:\033[0m %s\n' "$*" >&2; exit 1; }

# --- preflight checks --------------------------------------------------------

[ "$(id -u)" -eq 0 ] || error "Must run as root"
[ "$(uname -s)" = "Linux" ] || error "Only Linux is supported"

ARCH="$(uname -m)"
case "$ARCH" in
    aarch64|arm64) ;;
    *) error "Only arm64 is supported (detected: $ARCH)" ;;
esac

command -v curl  >/dev/null || error "curl is required"
command -v tar   >/dev/null || error "tar is required"

[ -d /persistent ] || error "/persistent not found — is this a UniFi NAS?"

# --- resolve latest release ---------------------------------------------------

info "Fetching latest release from github.com/$REPO ..."

RELEASE_JSON="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest")" \
    || error "Failed to fetch release info (check network / repo visibility)"

TAG="$(printf '%s' "$RELEASE_JSON" | grep -m1 '"tag_name"' | cut -d'"' -f4)"
[ -n "$TAG" ] || error "Could not parse tag from release"

ASSET_URL="$(printf '%s' "$RELEASE_JSON" \
    | grep '"browser_download_url"' \
    | grep -o 'https://[^"]*arm64\.tar\.gz')"
[ -n "$ASSET_URL" ] || error "No arm64 tarball found in release $TAG"

info "Latest release: $TAG"

# --- download & extract -------------------------------------------------------

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

info "Downloading $ASSET_URL ..."
curl -fsSL -o "$TMPDIR/release.tar.gz" "$ASSET_URL"

info "Extracting to $INSTALL_DIR ..."
mkdir -p "$INSTALL_DIR"

# preserve existing config
SAVED_CONFIG=""
if [ -f "$INSTALL_DIR/config.yaml" ]; then
    SAVED_CONFIG="$TMPDIR/config.yaml.bak"
    cp "$INSTALL_DIR/config.yaml" "$SAVED_CONFIG"
    info "Existing config.yaml backed up"
fi

tar -xzf "$TMPDIR/release.tar.gz" -C "$INSTALL_DIR"

if [ -n "$SAVED_CONFIG" ]; then
    mv "$SAVED_CONFIG" "$INSTALL_DIR/config.yaml"
    info "Existing config.yaml restored"
fi

chmod +x "$INSTALL_DIR/unas-custom" "$INSTALL_DIR/"*.sh

# --- install hooks ------------------------------------------------------------

info "Installing hooks ..."
"$INSTALL_DIR/unas-custom" install

echo ""
info "unas-custom $TAG installed to $INSTALL_DIR"
echo ""
echo "Next steps:"
echo "  1. Edit config:         vi $INSTALL_DIR/config.yaml"
echo "  2. Apply SMB overrides: $INSTALL_DIR/unas-custom smb apply"
echo "  3. Check status:        $INSTALL_DIR/unas-custom status"
