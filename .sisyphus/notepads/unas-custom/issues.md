# Issues — unas-custom

## [2026-03-07] Session Start — No issues yet

## [2026-03-07] Tooling

- `lsp_diagnostics` initially failed because `gopls` was not installed in PATH.
- Installed `gopls` to `~/.local/bin`; diagnostics now run, but report a workspace-scoping warning (`No active builds contain ...`) rather than code errors.

## [2026-03-07] F1 Plan Compliance Audit

- Definition of Done coverage threshold appears unmet: `go test -race -cover ./...` reports <80% in `cmd/unas-custom` (62.8%) and `internal/installer` (60.0%).
- Wrapper missing-original paths return exit code 1 without a clear stderr message in `cmd/unas-custom/wrapper.go` (`runOriginalCommand`). Plan task T7 acceptance expects a clear error message for missing `.orig` binaries.

## [2026-03-07] F2 Code Quality Review
### CRITICAL
- None.

### MAJOR
- `internal/installer/installer.go:106` — `installWrapper` only backs up when `os.IsNotExist(err)` is true; any other `os.Stat(origPath)` error (e.g., EPERM/EIO) is ignored and install proceeds to overwrite target (`internal/installer/installer.go:111`) without a verified backup. This can break rollback/uninstall guarantees and risks destructive partial installs.
- `cmd/unas-custom/main_test.go:35` (also `cmd/unas-custom/main_test.go:42`, `cmd/unas-custom/main_test.go:49`, `cmd/unas-custom/main_test.go:56`, `cmd/unas-custom/main_test.go:63`, `cmd/unas-custom/main_test.go:70`) — multiple tests assert “0 or 1” exit codes, which allows regressions to pass silently and does not verify command behavior/output.
- `cmd/unas-custom/status_test.go:321` — `TestRunStatusImpl` only checks that exit code is between 0 and 1 (panic guard), not status semantics. This gives weak confidence for a user-facing health command.

### MINOR
- `internal/logging/logger.go:59` — logger write errors are ignored (`l.writer.Write(...)` return values discarded). On disk-full/IO errors, wrapper diagnostics are silently lost, reducing observability when fail-open paths are exercised.
- `cmd/unas-custom/status.go:47` and `cmd/unas-custom/status.go:55` — status marks wrappers “installed” based only on `.orig` backup presence; it does not verify that `/usr/sbin/exportfs` or `/usr/bin/smbcontrol` actually point to/run `unas-custom`. This can report false positives after manual drift.
- `internal/smb/generator.go:45` and `internal/smb/generator.go:60` — share names/directive values are emitted verbatim; newline-containing config values can inject unintended additional Samba lines/sections. Config is local/admin-controlled, so impact is limited, but input normalization would harden output safety.

### NITPICK
- `cmd/unas-custom/install.go:13`, `cmd/unas-custom/install.go:32`, `cmd/unas-custom/smb_apply.go:17`, `cmd/unas-custom/smb_inject.go:17` — `args` parameters are currently unused; either validate/reject unexpected args or name as `_` to communicate intentional non-use.
- `cmd/unas-custom/install_test.go:15` — local variable uses snake_case (`smb_conf`) instead of idiomatic Go camelCase (`smbConf`).

### Verdict: SHIP WITH FIXES

## [2026-03-07] F4 Scope Fidelity Check
### Gaps (planned but missing)
- systemd drop-in is not fail-open on inject failure: plan requires reload to still send SIGHUP even if inject fails (.sisyphus/plans/unas-custom.md:140), but drop-in runs inject as a required ExecReload step (internal/installer/installer.go:14-18).
- install does not generate smb-overrides.conf: plan install sequence includes generating overrides file (.sisyphus/plans/unas-custom.md:1372), but installer Install() stops after smb include injection and returns (internal/installer/installer.go:98-103).
- config loader does not support the old flat NFS config format: plan calls for backward compat load of old format (.sisyphus/plans/unas-custom.md:354), but Load() only unmarshals nested nfs/smb keys (internal/config/config.go:28-42).
- status wrapper checks are weaker than planned: plan expects checking wrapper target + backup presence (.sisyphus/plans/unas-custom.md:1534), but status only checks for .orig file existence (cmd/unas-custom/status.go:47-60).

### Extras (implemented but not in plan)
- CI uploads coverage to Codecov (.github/workflows/ci.yml:72-78).

### Verdict: MAJOR GAPS
