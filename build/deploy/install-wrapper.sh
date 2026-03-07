#!/bin/bash
set -e

INSTALL_DIR="/persistent/nfs-intercept"
EXPORTFS_PATH="/usr/sbin/exportfs"
EXPORTFS_ORIG="/usr/sbin/exportfs.orig"
CONFIG_FILE="$INSTALL_DIR/config.yaml"

echo "=== Exportfs Wrapper Installation ==="

if [ ! -f "exportfs-wrapper" ]; then
    echo "Error: exportfs-wrapper binary not found in current directory"
    exit 1
fi

echo "Creating installation directory: $INSTALL_DIR"
mkdir -p "$INSTALL_DIR"

echo "Copying wrapper to $INSTALL_DIR..."
cp exportfs-wrapper "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/exportfs-wrapper"

if [ ! -f "$CONFIG_FILE" ]; then
    echo "Creating example config file..."
    if [ -f "config.example.yaml" ]; then
        cp config.example.yaml "$CONFIG_FILE"
    else
        cat > "$CONFIG_FILE" <<'EOF'
rules:
  - path: "/var/nfs/shared/Media"
    options: "rw,sync,no_subtree_check,insecure,all_squash,anonuid=977,anongid=988"
EOF
    fi
    echo "Config created at: $CONFIG_FILE"
    echo "Please edit this file to add your NFS export rules!"
else
    echo "Config file already exists at: $CONFIG_FILE"
fi

if [ -f "$EXPORTFS_PATH" ] && [ ! -f "$EXPORTFS_ORIG" ]; then
    echo "Backing up original exportfs to $EXPORTFS_ORIG..."
    cp -a "$EXPORTFS_PATH" "$EXPORTFS_ORIG"
fi

echo "Installing wrapper..."
cp "$INSTALL_DIR/exportfs-wrapper" "$EXPORTFS_PATH"
chmod +x "$EXPORTFS_PATH"

echo ""
echo "=== Installation Complete ==="
echo ""
echo "Next steps:"
echo "  1. Edit config: $CONFIG_FILE"
echo "  2. Test: exportfs -ra"
echo "  3. Verify: cat /etc/exports.d/*.exports"
echo ""
echo "IMPORTANT: After firmware updates, re-run this script to reinstall the wrapper."
echo "The wrapper and config in $INSTALL_DIR will persist across updates."
echo ""
echo "To uninstall: $INSTALL_DIR/uninstall-wrapper.sh"
