# Decisions — unas-custom

## [2026-03-07] Session Start

- Binary installation: COPY (not symlink) — consistent with existing exportfs pattern
- Race condition in concurrent inject: ACCEPTED as known limitation (consistent with exportfs)
- Log rotation: NOT added — file grows, user rotates manually
- share.conf include fallback: if `include = /etc/samba/share.conf` not found, APPEND to end of file

## [2026-03-07] T7 wrapper implementation

- NFS exports rewrite uses two-phase processing (`walk/read/modify` then `write`) so parse/read failures keep files unchanged before invoking original `exportfs`.
- Wrapper logging failures are non-fatal: fallback to `logging.NewNoop()` to preserve fail-open semantics.
- `main.go` keeps install/uninstall/status as stubs, but routes `runSmbApply` and `runSmbInject` to real implementations now.
