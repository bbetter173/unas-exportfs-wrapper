#!/bin/bash
set -e

BINARY="./unas-custom"

echo "=== unas-custom Uninstallation ==="

if [ ! -f "$BINARY" ]; then
    echo "Error: unas-custom binary not found in current directory"
    exit 1
fi

"$BINARY" uninstall

echo ""
echo "=== Uninstallation Complete ==="
echo ""
echo "To fully clean up, remove the persistent directory manually:"
echo "  rm -rf /persistent/unas-custom/"
