# Exportfs Wrapper - NFS Export Modifier

A transparent wrapper for the `exportfs` command that modifies NFS export options on-the-fly.

## How It Works

1. **Wrapper intercepts `exportfs` calls** - Moves original `/usr/sbin/exportfs` to `/usr/sbin/exportfs.orig`
2. **Modifies export files** - Before calling original exportfs, applies rules from config
3. **Transparent execution** - Passes all arguments to original exportfs
4. **Persistent storage** - Wrapper and config stored in `/persistent` partition (survives firmware updates)

## Installation

### 1. Build

```bash
make build-arm64 deploy
```

### 2. Deploy to UNAS

```bash
scp -r build/deploy/* root@<unas-ip>:/persistent/nfs-intercept/
```

### 3. Install on UNAS

```bash
ssh root@<unas-ip>
cd /persistent/nfs-intercept
chmod +x *.sh exportfs-wrapper
./install-wrapper.sh
```

### 4. Configure

Edit `/persistent/nfs-intercept/config.yaml`:

```yaml
rules:
  - path: "/var/nfs/shared/Media"
    options: "rw,sync,no_subtree_check,insecure,all_squash,anonuid=977,anongid=988"
```

### 5. Test

```bash
# Trigger NFS export reload (via UniFi UI or manually)
exportfs -ra

# Verify modifications
cat /etc/exports.d/*.exports
```

## Post-Firmware Update

After firmware updates that wipe `/usr/sbin`, re-run the install script:

```bash
ssh root@<unas-ip>
cd /persistent/nfs-intercept
./install-wrapper.sh
```

The `/persistent` partition is separate from the OS and survives firmware updates. The install script just reinstalls the wrapper to `/usr/sbin/exportfs`.

## Storage Locations

```
/persistent/nfs-intercept/          # Survives firmware updates (separate partition)
├── exportfs-wrapper                # Main binary
├── config.yaml                      # Your rules
├── install-wrapper.sh              # Run after firmware updates
└── uninstall-wrapper.sh            # Remove wrapper

/usr/sbin/                          # Wiped by firmware updates (OS partition)
├── exportfs          → wrapper     # Needs reinstall after updates
└── exportfs.orig     → backup      # Original exportfs binary
```

## Uninstalling

```bash
/persistent/nfs-intercept/uninstall-wrapper.sh
```

This:
- Restores original `exportfs` binary
- Disables systemd service
- Leaves files in `/persistent` (manual cleanup if desired)

## Configuration Reference

```yaml
rules:
  - path: "<exact-export-path>"
    options: "<comma-separated-nfs-options>"
```

**Path must match exactly** what appears in the export files.

Find your export paths:
```bash
grep -h "^/" /etc/exports.d/*.exports
```

## Troubleshooting

### Wrapper not working

```bash
# Check if wrapper is installed
ls -la /usr/sbin/exportfs*

# Reinstall if missing
/persistent/nfs-intercept/install-wrapper.sh
```

### Rules not applying

```bash
# Test wrapper manually
exportfs -ra

# Check modified content
cat /etc/exports.d/*.exports

# Verify config syntax
cat /persistent/nfs-intercept/config.yaml
```

### Restore original exportfs

```bash
# Quick restore
cp /usr/sbin/exportfs.orig /usr/sbin/exportfs

# Full uninstall
/persistent/nfs-intercept/uninstall-wrapper.sh
```

## Development

```bash
# Build
make build-arm64

# Create deployment package
make deploy

# Clean
make clean
```

## License

MIT
