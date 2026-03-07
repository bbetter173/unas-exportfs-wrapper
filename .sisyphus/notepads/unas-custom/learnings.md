# Learnings — unas-custom

## [2026-03-07] Session Start

### Codebase Snapshot
- Module: `github.com/bbettridge/exportfs-wrapper` (needs rename to `github.com/bbettridge/unas-custom`)
- Go version: 1.21.4
- Directories: `cmd/exportfs-wrapper/`, `internal/config/`, `internal/nfs/`
- Target: `cmd/unas-custom/`, `internal/config/`, `internal/nfs/`, `internal/smb/`, `internal/logging/`, `internal/installer/`
- Worktree: `/Users/bbettridge/Development/ebpf-file-intercept/exportfs-wrapper` (main, feat/smb-custom-options)
- Single dependency: `gopkg.in/yaml.v3 v3.0.1`

### Key Patterns
- `exec.Command` for exec (NOT `syscall.Exec`) — per G14
- Fail-open in both wrapper modes — per G2, G15
- Atomic write for smb.conf modifications (write to .tmp + os.Rename)
- Configurable paths via struct (for testability without system files)
- TDD: tests FIRST, then implementation

### SMB Strategy (CRITICAL)
- UDC rewrites smb.conf during share operations → wipes include line
- TWO interception points: smbcontrol wrapper + systemd drop-in for smbd.service
- smb inject must be idempotent, atomic, config.yaml-independent (G19)
- Drop-in format: `ExecReload=` (clear) + inject command + SIGHUP + ExecStartPost for restarts

### Dispatch Modes
- basename "exportfs" → NFS wrapper (runNfsWrapper)
- basename "smbcontrol" → SMB wrapper (runSmbcontrolWrapper)  
- default → CLI (runCLI)

## [2026-03-07] T8.5: SMB Inject Implementation Complete

### Implementation Summary
- **internal/smb/inject.go**: Idempotent smb.conf include line management
  - `Inject()`: Adds include line after `include = /etc/samba/share.conf` (or appends if not found)
  - `Remove()`: Idempotently removes include line
  - `atomicWrite()`: Writes to temp file then renames (prevents partial reads during crashes)
  - Constants: `DefaultSmbConfPath`, `DefaultIncludeLine`, `ShareConfInclude`

- **internal/smb/inject_test.go**: 9 comprehensive test cases
  - `TestInject_AddsIncludeLine`: Verifies insertion after share.conf include
  - `TestInject_AlreadyPresent`: Idempotency check (no-op if already present)
  - `TestInject_NoShareConfLine`: Fallback to append if share.conf include not found
  - `TestInject_PreservesContent`: All other content preserved byte-for-byte
  - `TestInject_MissingSmbConf`: Returns error for missing file (not panic)
  - `TestInject_EmptyFile`: Appends to empty file
  - `TestRemove_RemovesIncludeLine`: Removes our include line
  - `TestRemove_NoIncludeLine`: Idempotency check (no-op if not present)
  - `TestRemove_PreservesContent`: All other content preserved

- **cmd/unas-custom/smb_inject.go**: CLI handler
  - `runSmbInjectImpl()`: Calls `smb.Inject()` with defaults
  - Returns 0 on success, 1 on error
  - Prints status message to stdout

### Key Design Decisions
1. **No config.yaml dependency** (G19): inject.go imports only `os`, `fmt`, `strings`
2. **Atomic write pattern**: Write to `.tmp` file, then `os.Rename()` to prevent partial reads
3. **Idempotency**: Both `Inject()` and `Remove()` are safe to call multiple times
4. **Insertion strategy**: Place after `include = /etc/samba/share.conf` so our overrides load AFTER share definitions
5. **Fallback**: If share.conf include not found, append to end of file

### Test Results
- All 9 new tests PASS
- All 8 existing generator tests PASS
- Race detector: PASS
- Build: SUCCESS

### Commit
- Hash: `fefcf3ba9ab2f5523ce8775fe89932ffcfabc796`
- Message: `feat(smb): idempotent smb.conf include line injection with atomic write`
- Files: 3 changed, 423 insertions

## [2026-03-07] T8: smb apply command

### Function Naming Pattern (Avoid Duplicate Compile Errors)
- main.go has stubs: `func runSmbApply(args []string) int { return 0 }`
- Implementation files use `WithPaths` suffix: `func runSmbApplyWithPaths(args []string, configPath, overridesPath string) int`
- Tests test the `WithPaths` variant directly (injected paths → no system file access)
- smb_inject.go follows same pattern: `runSmbInjectImpl` is the real impl, `runSmbInject` stub in main.go

### smb apply Implementation
- `smb_apply.go` defines `runSmbApplyWithPaths` only (no `runSmbApply` — that stub is in main.go)
- Constants named with `smbApply` prefix to avoid collisions: `smbApplyDefaultConfigPath`, `smbApplyDefaultOverridesPath`
- `config.Load` handles YAML parsing + validation; returns error for missing file, bad YAML, empty share names
- `smb.GenerateOverrides(cfg.SMB.Overrides)` returns `(string, error)` — header-only for empty slice
- `os.MkdirAll` before `os.WriteFile` → creates nested directories automatically

### Test YAML for Invalid Config
- `{unclosed: [bracket` reliably fails yaml.Unmarshal (yaml: line 1: did not find expected ',' or ']')
- Empty overrides config: `smb:\n  overrides: []` → returns 0, writes header-only file

## [2026-03-07] T7: Wrapper modes wired + tested

### Wrapper testability pattern
- Add package-private helpers with path injection for wrappers:
  - `runNfsWrapperWithPaths(args, configPath, exportsDir, originalPath)`
  - `runSmbcontrolWrapperWithPaths(args, smbConfPath, includeLine, originalPath)`
- Keep production entrypoints (`runNfsWrapper`, `runSmbcontrolWrapper`) as thin wrappers over constants.

### Fail-open behavior captured in tests
- NFS wrapper: missing/invalid config logs warning and still executes original `exportfs.orig`.
- SMB wrapper: `smb.Inject` errors log warning and still executes original `smbcontrol.orig`.
- Hard failure condition is limited to missing original binary path for each wrapper.

### Exec behavior
- Use `exec.Command` with stdio passthrough and propagate `ExitError.ExitCode()`.
- Test arg passthrough by writing `"$@"` from mock shell scripts to a temp file.

## T10 — Installer package (2026-03-07)

- `Installer` struct with all paths as fields + mockable `DaemonReloadFn`/`Stdout` — identical testability pattern to T8/T8.5
- `copyFile` helper: `os.Open`/`os.Create` + `io.Copy` + explicit `out.Close()` (no defer, to capture error) + `os.Chmod(0755)`
- `installWrapper` idempotency: check `os.Stat(origPath)` → `os.IsNotExist` before backing up; always copy binaryPath to targetPath
- `uninstallWrapper`: if no `.orig` → print notice + return nil (no error); otherwise copy back and remove `.orig`
- `os.Remove` on drop-in: ignore `os.IsNotExist` error for idempotency
- Test helper returns `(*Installer, string)` — tests needing DaemonReloadFn tracking override the field directly
- `smb.Inject`/`smb.Remove` handle idempotency themselves; installer just calls them
- LSP shows errors for cross-file same-package symbols until indexed — `go build` is ground truth

## T14 — README rewrite and unified config example (2026-03-07)

- Config struct uses `yaml:"nfs"` and `yaml:"smb"` top-level keys — old flat `rules:` format is incompatible
- Verified config.example.yaml loads via inline Go program using same struct/validation logic as `config.Load()`
- README structure that works: Overview → How It Works (NFS then SMB with architecture detail) → Installation → Post-Firmware → Config Reference → Subcommands → Storage Locations → Troubleshooting → Migration → Development
- SMB interception needs two points explained clearly: smbcontrol wrapper (covers UDC share ops) + systemd drop-in (covers smbd restarts/reloads)
- `go run - <<'EOF'` heredoc syntax doesn't work for inline Go programs — write to a temp file and `go run <file>` instead
- Migration section: old `rules:` → `nfs:\n  rules:`, old dir `/persistent/nfs-intercept/` → `/persistent/unas-custom/`

## [2026-03-07] F1 Plan Compliance Audit

- Must Have checks executed: all 15 checklist items pass, including `make build-arm64` producing a static 2.2MB ARM64 ELF and `go list -m all | wc -l` == 3.
- Guardrails checked (G1/G2/G4/G8/G14/G19): no violations found via grep/go list commands.
- DoD gap: package coverage for `cmd/unas-custom` and `internal/installer` is below the plan's >80% threshold.
