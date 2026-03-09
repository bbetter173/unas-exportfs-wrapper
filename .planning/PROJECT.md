# unas-custom Hardening

## What This Is

A Go wrapper tool for UniFi NAS Pro that intercepts NFS `exportfs` and SMB `smbcontrol` calls via symlinks, modifying exports and Samba configs based on YAML rules. Deployed as a single statically-linked ARM64 binary with fail-open resilience.

## Core Value

Wrapper operations must never break the underlying NFS/SMB services — fail-open design is non-negotiable.

## Requirements

### Validated

- ✓ NFS export interception and option rewriting — existing
- ✓ SMB smbcontrol wrapper with include injection — existing
- ✓ SMB override generation from YAML config — existing
- ✓ Per-share append_valid_users support — existing
- ✓ Symlink-based installation with .orig backups — existing
- ✓ Systemd drop-in for smbd reload — existing
- ✓ Status command for health checks — existing
- ✓ Fail-open wrapper design — existing
- ✓ Atomic file writes (temp + rename) for config files — existing
- ✓ Persistent logging to /persistent/unas-custom/ — existing

### Active

- [ ] Systemd drop-in must be fail-open on inject failure (still send SIGHUP)
- [ ] Install must generate smb-overrides.conf on completion (not just placeholder)
- [ ] Atomic symlink installation (temp symlink + rename to avoid dangling state)
- [ ] Config value sanitization (reject/strip newlines in SMB share names and directives)
- [ ] Installer backup verification must fail early on non-IsNotExist stat errors
- [ ] Better error messages for missing .orig binaries (show path, symlink target, suggest `status`)
- [ ] Status command tests must validate exit code semantics and assertion content
- [ ] Installer backup verification tests for non-IsNotExist stat errors

### Out of Scope

- Backward compat for old flat NFS config format — not yet released, no migration needed
- NFS export caching/performance optimization — not blocking, premature optimization
- Concurrency control (flock) on smb-overrides.conf — single-user tool, low risk
- INI-aware SMB config parsing — current string matching works, hardening not critical now
- Config validation for NFS option syntax — medium priority, defer

## Context

This is a brownfield hardening effort. The tool works but has gaps identified during codebase audit (`.planning/codebase/CONCERNS.md`). The focus is pragmatic fixes to unblock production use — not comprehensive refactoring.

Target platform: UniFi NAS Pro (Linux/ARM64). Config lives at `/persistent/unas-custom/config.yaml`. Binary deploys to `/persistent/unas-custom/unas-custom`.

## Constraints

- **Platform**: Linux/ARM64 only, UniFi NAS Pro — hardcoded paths in `/persistent/`, `/usr/sbin/`, `/usr/bin/`
- **Dependencies**: Minimal — only `gopkg.in/yaml.v3`, no new deps unless justified
- **Build**: CGO_ENABLED=0, static linking required
- **Safety**: All wrapper changes must preserve fail-open behavior

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Pragmatic fixes over production hardening | Unblock production use without over-engineering | — Pending |
| Skip backward compat for old NFS config | Tool not yet released, no users to migrate | — Pending |
| Focus on high-priority concerns only | 8 items vs 22 — ship faster, revisit later | — Pending |

---
*Last updated: 2026-03-09 after initialization*
