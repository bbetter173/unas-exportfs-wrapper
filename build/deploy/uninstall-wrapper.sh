#!/bin/bash
set -e

EXPORTFS_PATH="/usr/sbin/exportfs"
EXPORTFS_ORIG="/usr/sbin/exportfs.orig"

echo "=== Exportfs Wrapper Uninstall ==="

if [ -f "$EXPORTFS_ORIG" ]; then
    echo "Restoring original exportfs..."
    cp -a "$EXPORTFS_ORIG" "$EXPORTFS_PATH"
    rm -f "$EXPORTFS_ORIG"
    echo "Original exportfs restored"
else
    echo "No backup found at $EXPORTFS_ORIG"
fi

echo ""
echo "=== Uninstall Complete ==="
echo ""
echo "Files in /persistent/nfs-intercept have NOT been deleted."
echo "To fully remove: rm -rf /persistent/nfs-intercept"
