#!/bin/bash
set -e

BINARY="./unas-custom"

echo "=== unas-custom Installation ==="

if [ ! -f "$BINARY" ]; then
    echo "Error: unas-custom binary not found in current directory"
    exit 1
fi

# Check for old installation and print migration notice
if [ -d "/persistent/nfs-intercept" ]; then
    echo ""
    echo "NOTE: Old installation detected at /persistent/nfs-intercept/"
    echo "The new tool uses /persistent/unas-custom/ instead."
    echo "To migrate your NFS rules, copy them to the new config format:"
    echo "  Old: /persistent/nfs-intercept/config.yaml (flat rules:)"
    echo "  New: /persistent/unas-custom/config.yaml (nested nfs: rules:)"
    echo "To clean up old installation: rm -rf /persistent/nfs-intercept/"
    echo ""
fi

"$BINARY" install

echo ""
echo "=== Installation Complete ==="
echo ""
echo "Next steps:"
echo "  1. Edit config: /persistent/unas-custom/config.yaml"
echo "  2. Apply SMB overrides: $BINARY smb apply"
echo "  3. Check status: $BINARY status"
echo ""
echo "IMPORTANT: After firmware updates, re-run this script to reinstall the wrappers."
