# unas-custom

UniFi NAS customization tool for NFS and SMB configuration overrides.

## Overview

UniFi NAS Pro runs a daemon called UDC that manages NFS exports and Samba shares. Any time you create or modify a share through the UniFi UI, UDC rewrites `/etc/exports.d/*.exports` and `/etc/samba/smb.conf` from scratch, wiping any manual changes you've made.

`unas-custom` solves this by intercepting the tools UDC calls to apply its changes. Your overrides get injected at the right moment, every time, without fighting the UniFi management layer.

Configuration and the binary live in `/persistent/unas-custom/`, a partition that survives firmware updates. After a firmware update you re-run `unas-custom install` to restore the hooks.

## How It Works

`unas-custom` is a single binary that dispatches into different modes based on `argv[0]`:

| `argv[0]` | Mode | What it does |
|---|---|---|
| `exportfs` | NFS wrapper | Intercepts exportfs calls, applies NFS option overrides, execs original |
| `smbcontrol` | SMB wrapper | Injects include line into smb.conf, execs original |
| `unas-custom` | CLI | Management subcommands |

Symlinks in `/usr/sbin/` and `/usr/bin/` point to the binary in `/persistent/unas-custom/`, so the system calls the wrapper transparently.

### NFS interception

When UDC calls `exportfs -ra` to reload exports, the wrapper runs first. It reads the export files, finds any paths that match your config rules, replaces their options, writes the files back, then execs the real `exportfs`. UDC never knows anything changed.

### SMB interception

SMB is trickier. UDC rewrites `smb.conf` during share operations, which removes any include line you've added. Two interception points work together to keep the include line in place:

**1. smbcontrol wrapper** (`/usr/bin/smbcontrol` symlink)

UDC calls `smbcontrol` to signal smbd after updating `smb.conf`. The wrapper re-injects the include line into `smb.conf` before passing the signal through. This covers the common case where UDC modifies a share.

**2. systemd drop-in** (`/etc/systemd/system/smbd.service.d/unas-custom.conf`)

The drop-in hooks into smbd start and reload events:

```ini
[Service]
ExecStartPost=/persistent/unas-custom/unas-custom smb inject
ExecReload=
ExecReload=/persistent/unas-custom/unas-custom smb inject
ExecReload=/bin/kill -HUP $MAINPID
```

After smbd starts or reloads, the include line gets injected before smbd reads its config. This covers restarts and edge cases the smbcontrol wrapper might miss.

**The include line** points to `/persistent/unas-custom/smb-overrides.conf`, a Samba INI file generated from your config. Samba merges sections with the same name, so your overrides add or replace directives in existing shares without touching the rest of the config.

## Installation

### Quick Install (recommended)

```bash
ssh root@<unas-ip>
curl -fsSL https://raw.githubusercontent.com/bbetter173/unas-exportfs-wrapper/main/scripts/get.sh | bash
```

This downloads the latest release, extracts it to `/persistent/unas-custom/`, and installs the hooks. Existing `config.yaml` is preserved on upgrades.

### Manual Install

```bash
# 1. Build
make build-arm64

# 2. Deploy to UNAS
make deploy
scp -r build/deploy/* root@<unas-ip>:/persistent/unas-custom/

# 3. Install on UNAS
ssh root@<unas-ip>
cd /persistent/unas-custom
chmod +x unas-custom
./unas-custom install

# 4. Configure
vi /persistent/unas-custom/config.yaml

# 5. Apply SMB overrides
./unas-custom smb apply
```

## Post-Firmware Update

Firmware updates wipe `/usr/sbin`, `/usr/bin`, and `/etc/systemd/system/`. Re-run install to restore the hooks:

```bash
ssh root@<unas-ip>
/persistent/unas-custom/unas-custom install
```

Your config and the binary in `/persistent/unas-custom/` are untouched.

## Configuration Reference

Copy `config.example.yaml` to `/persistent/unas-custom/config.yaml` and edit it.

```yaml
# NFS export option overrides
# Replaces options for matching export paths (path must match exactly)
nfs:
  rules:
    - path: "/var/nfs/shared/Media"
      options: "rw,sync,no_subtree_check,insecure,all_squash,anonuid=977,anongid=988"
    - path: "/var/nfs/shared/Backups"
      options: "rw,sync,no_subtree_check,insecure,no_root_squash"

# SMB share parameter overrides
# Adds/overrides directives in existing Samba shares (share name must match exactly)
smb:
  overrides:
    - share: "Media"
      directives:
        guest ok: "yes"
        create mask: "0664"
        directory mask: "0775"
    - share: "Backups"
      directives:
        read only: "no"
        force user: "root"
```

**NFS rules**: `path` must match exactly what appears in the export files. Find your paths:

```bash
grep -h "^/" /etc/exports.d/*.exports
```

**SMB overrides**: `share` must match the share name exactly (case-sensitive). `directives` is a map of Samba parameter names to values. These are merged into the existing share section, so you only need to list the parameters you want to change.

## Subcommands

```
unas-custom install      Install NFS wrapper, smbcontrol wrapper, and systemd drop-in
unas-custom uninstall    Remove all hooks and restore originals
unas-custom status       Show installation state (6 checks)
unas-custom smb apply    Generate smb-overrides.conf from config
unas-custom smb inject   Idempotently inject include line into smb.conf
```

## Storage Locations

```
/persistent/unas-custom/          # Survives firmware updates
├── unas-custom                   # Main binary
├── exportfs -> unas-custom       # NFS wrapper symlink
├── smbcontrol -> unas-custom     # SMB wrapper symlink
├── config.yaml                   # Your configuration
├── smb-overrides.conf            # Generated SMB overrides (Samba INI)
└── unas-custom.log               # Wrapper activity log

/usr/sbin/                        # Wiped by firmware updates
├── exportfs -> /persistent/unas-custom/exportfs
└── exportfs.orig                 # Original exportfs backup

/usr/bin/                         # Wiped by firmware updates
├── smbcontrol -> /persistent/unas-custom/smbcontrol
└── smbcontrol.orig               # Original smbcontrol backup

/etc/systemd/system/smbd.service.d/
└── unas-custom.conf              # systemd drop-in (wiped by firmware)
```

## Troubleshooting

### Wrapper not working after firmware update

Re-run install:

```bash
/persistent/unas-custom/unas-custom install
```

Check status:

```bash
unas-custom status
```

### SMB overrides not applying

Check that `smb.conf` has the include line:

```bash
grep include /etc/samba/smb.conf
```

If missing, inject it:

```bash
unas-custom smb inject
```

Check that `smb-overrides.conf` exists and has the right content:

```bash
cat /persistent/unas-custom/smb-overrides.conf
```

If missing, regenerate it:

```bash
unas-custom smb apply
```

### NFS rules not applying

Verify the path in your config matches exactly:

```bash
grep -h "^/" /etc/exports.d/*.exports
```

Check the wrapper log:

```bash
tail /persistent/unas-custom/unas-custom.log
```

### Restore original exportfs

```bash
cp /usr/sbin/exportfs.orig /usr/sbin/exportfs
```

## Migration from exportfs-wrapper

If you're upgrading from the old `exportfs-wrapper` tool:

**Config format**: The old flat `rules:` key moves under `nfs:`:

```yaml
# Old format
rules:
  - path: "/var/nfs/shared/Media"
    options: "..."

# New format
nfs:
  rules:
    - path: "/var/nfs/shared/Media"
      options: "..."
```

**Install directory**: `/persistent/nfs-intercept/` becomes `/persistent/unas-custom/`. After installing the new tool, the old directory can be removed:

```bash
rm -rf /persistent/nfs-intercept/
```

Run `unas-custom install` to set up the new hooks.

## Development

```bash
make build-arm64   # Build ARM64 binary
make test          # Run all tests
make lint          # Run vet + format check
make deploy        # Create deployment package
```

## License

MIT
