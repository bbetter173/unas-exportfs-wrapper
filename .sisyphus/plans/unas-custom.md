# Transform exportfs-wrapper into unas-custom: Multi-Service UNAS Customization Tool

## TL;DR

> **Quick Summary**: Refactor the 271-line `exportfs-wrapper` Go binary into `unas-custom` — a busybox-style multi-call binary that overrides NFS export options (existing) and SMB share parameters (new) on a UniFi NAS Pro, without disrupting UniFi management. Single binary with THREE dispatch modes (exportfs wrapper, smbcontrol wrapper, CLI), zero new dependencies, TDD throughout.
> 
> **Deliverables**:
> - Renamed Go module (`github.com/bbettridge/unas-custom`) with busybox-style dispatch (3 modes)
> - Unified YAML config format supporting both NFS rules and SMB overrides
> - SMB override generator producing valid Samba INI config
> - NFS wrapper mode with fail-open safety and behavior preservation
> - SMB wrapper mode (smbcontrol) with include line injection before exec
> - `smb inject` subcommand for idempotent smb.conf include line management
> - Systemd drop-in for smbd.service to intercept reloads
> - Management CLI: `install`, `uninstall`, `status`, `smb apply`, `smb inject` subcommands
> - Unified install/uninstall scripts for all service hooks
> - Updated Makefile, CI pipeline, README, and config examples
> - Comprehensive TDD test coverage (currently zero tests)
> 
> **Estimated Effort**: Large
> **Parallel Execution**: YES — 5 waves + FINAL verification
> **Critical Path**: T1 (rename) → T8.5 (smb inject) → T7 (wrapper) → T10 (install) → T13 (scripts) → FINAL

---

## Context

### Original Request
Transform the existing `exportfs-wrapper` Go application into a generalized UniFi NAS (UNAS) customization tool that can override/extend configuration for multiple UniFi-managed services (NFS, SMB, and future services) without disrupting the UniFi management plane.

### Interview Summary
**Key Discussions**:
- **Project name**: `unas-custom` — short, clear purpose, used as binary name and persistent directory
- **Architecture**: Single binary, busybox-style — `filepath.Base(os.Args[0])` dispatch with 3 modes: `exportfs` (NFS wrapper), `smbcontrol` (SMB wrapper), `unas-custom` (CLI). Proven pattern (Talos, Ignition, Snapd, Granted all use it)
- **CLI framework**: Hand-rolled `os.Args` parsing — zero new dependencies, keeps binary at ~2.1MB. Only 6-7 subcommands, no need for Cobra/Kong
- **SMB strategy**: **REVISED** — UDC DOES rewrite `/etc/samba/smb.conf` during normal share operations, wiping any include line we add. TWO interception points needed: (1) wrap `smbcontrol` to re-inject include before exec, (2) systemd drop-in for smbd.service to inject before SIGHUP on reload/restart. Samba native `include` overlay still used for the overrides themselves — merges same-named sections, last directive wins
- **Tests**: TDD — write failing tests first, then implement. Currently zero test files.
- **Reinstall**: Manual `unas-custom install` after firmware updates. No cron/systemd automation.
- **Fail behavior**: Fail-open — config errors must NEVER prevent original services from working

**Research Findings**:
- **Samba include semantics**: MERGE behavior confirmed. Same `[ShareName]` in included file adds directives to existing section. Last-wins for conflicting parameters. Partial override (e.g., just `guest ok`) works without re-specifying all directives.
- **Samba reload risk**: Known issue where `smbcontrol reload-config` may not re-read included files. However, UDC triggers `systemd ReloadUnit("smbd.service")` which sends SIGHUP — should cause full re-read. smbd also auto-re-reads every 3 minutes. Must verify on device.
- **init() bug**: Current code does `os.Stat(file); if os.IsNotExist(err) { syscall.Exec(file) }` — attempts to exec a file just confirmed not to exist. Silent failure. Must be redesigned.
- **Binary size**: Current 2.1MB stripped ARM64. Adding Cobra/Kong would add ~0.7MB. Hand-rolled stays at ~2.1MB. Negligible difference for 88TB NAS, but clean to avoid.
- **Go module rename**: `go mod edit -module` + `sed` for imports. Single atomic commit. Module path doesn't need to match repo path.
- **Install script idempotency**: Confirmed — all scripts safe to run multiple times. Hardcoded values fully mapped for rename.

### Metis Review (Original + Revision)
**Identified Gaps (original, addressed)**:
- **Fail-open behavior**: Must change current `log.Fatalf` behavior in wrapper mode. Config errors should log warning and pass through to original exportfs. Applied as guardrail G2.
- **Status command output**: Not defined — now defined in T11 acceptance criteria.
- **Logging**: Added logging infrastructure as T5 — file-based log in /persistent/unas-custom/.
- **Migration path**: Explicit: no auto-migration. `install` prints message if old installation detected. Manual config recreation. Applied as guardrail G10.
- **exec.Command vs syscall.Exec**: Keep current `exec.Command` approach for wrapper — simpler error handling, exit code propagation already works.
- **Share name validation**: Not implemented — creating a new section via override is valid Samba behavior. User is responsible for correct share names.
- **Concurrent exportfs calls**: Low risk, acknowledged limitation. No file locking added.

**CRITICAL REVISION — UDC smb.conf behavior**:
- **Previous understanding was WRONG**: UDC DOES rewrite `/etc/samba/smb.conf` during normal share create/modify/delete operations
- **UDC reload flow** (reverse-engineered from `reloadSmbService`):
  1. UDC writes smb.conf → **WIPES our include line**
  2. `svc.Reload()` → `systemctl reload smbd` → SIGHUP → smbd re-reads (without our include)
  3. `samba.ReloadSession()` → `smbcontrol <pid> reload-config` (only if user Names specified)
- **Revised strategy**: TWO interception points:
  1. **smbcontrol wrapper**: inject include line → exec original smbcontrol.orig (catches path 3)
  2. **Systemd drop-in for smbd.service**: inject include line → SIGHUP (catches path 2 + restarts)
  3. **`smb inject` subcommand**: idempotent include line management (called by both interception points)
- **Gap analysis (revision)**: Metis identified ExecStartPost needed (covers restarts, not just reloads), atomic write for inject (write-to-temp + rename), fallback for missing share.conf include line, and 5 new guardrails (G15-G19)

---

## Work Objectives

### Core Objective
Transform the 3-file, 271-line `exportfs-wrapper` into `unas-custom` — a multi-service configuration override tool with busybox-style dispatch, comprehensive tests, and SMB support — while preserving exact NFS modification behavior and never breaking UniFi management.

### Concrete Deliverables
- `cmd/unas-custom/main.go` — busybox-style dispatch entry point (3 modes: exportfs, smbcontrol, CLI)
- `internal/config/config.go` — unified config format (NFS + SMB sections)
- `internal/nfs/parser.go` — preserved NFS parser (unchanged logic, new imports)
- `internal/smb/generator.go` — SMB override file generator
- `internal/smb/inject.go` — idempotent smb.conf include line management
- `internal/installer/installer.go` — install/uninstall logic (NFS wrapper + smbcontrol wrapper + systemd drop-in)
- `internal/logging/logger.go` — file-based logger
- `scripts/install.sh` + `scripts/uninstall.sh` — unified shell wrappers
- `Makefile` — updated build/deploy pipeline
- `.github/workflows/ci.yml` — updated CI with test enforcement
- `config.example.yaml` — unified config example
- `README.md` — comprehensive rewrite

### Definition of Done
- [ ] `go test -race -cover ./...` passes with >80% coverage on new code
- [ ] `go vet ./...` passes
- [ ] `gofmt -l .` produces no output
- [ ] `make build-arm64` produces a static ARM64 binary <3MB
- [ ] Binary dispatches correctly based on `os.Args[0]` (3 modes: exportfs → NFS wrapper, smbcontrol → SMB wrapper, default → CLI)
- [ ] NFS modification produces identical output to current code for same inputs
- [ ] SMB override generator produces valid Samba INI syntax
- [ ] smbcontrol wrapper intercepts and injects include line before exec
- [ ] `smb inject` idempotently manages smb.conf include line (no-op if present, adds if missing)
- [ ] `unas-custom install` sets up NFS wrapper + smbcontrol wrapper + systemd drop-in
- [ ] `unas-custom uninstall` cleanly reverses all modifications (wrappers + drop-in + include line)
- [ ] `unas-custom status` reports installation state (including smbcontrol + drop-in)
- [ ] `unas-custom smb apply` generates overrides file from config
- [ ] `unas-custom smb inject` manages smb.conf include line
- [ ] All acceptance criteria pass without human intervention

### Must Have
- Exact NFS modification behavior preservation (proven by tests)
- Fail-open wrapper mode for BOTH exportfs and smbcontrol (config/inject errors → log warning, pass through to original)
- Busybox-style argv[0] dispatch with 3 modes (exportfs, smbcontrol, unas-custom)
- Unified YAML config with `nfs:` and `smb:` sections
- SMB override via Samba `include` directive
- smbcontrol wrapper that re-injects include line before exec'ing original
- Systemd drop-in for smbd.service that injects include line before SIGHUP (reload) and after start (restart)
- `smb inject` subcommand: idempotent, atomic write, config.yaml-independent
- Idempotent install/uninstall (NFS wrapper + smbcontrol wrapper + systemd drop-in)
- Zero new Go dependencies (yaml.v3 only)
- Static ARM64 binary with `-ldflags="-s -w"`
- TDD test coverage for all new code

### Must NOT Have (Guardrails)
- **G1**: Must NOT add any Go dependency beyond `gopkg.in/yaml.v3` — enforce in CI
- **G2**: Must NOT `log.Fatalf` or `os.Exit(1)` in wrapper mode on config errors — fail-open always (applies to BOTH exportfs and smbcontrol wrapper modes)
- **G3**: Must NOT auto-migrate old config from `/persistent/nfs-intercept/` — print message only
- **G4**: Must NOT implement `nfs apply` subcommand — NFS modifications happen transparently in wrapper mode
- **G5**: Must NOT build share discovery/listing features
- **G6**: Must NOT integrate `testparm` or any external command validation
- **G7**: Must NOT add systemd services or cron jobs for our tool itself — the systemd drop-in for smbd.service is an interception mechanism, not a service for unas-custom
- **G8**: Must NOT use Cobra, Kong, urfave/cli, or any CLI framework
- **G9**: Must NOT break UDC daemon operation — all modifications are additive overlays
- **G10**: Must NOT auto-delete `/persistent/nfs-intercept/` — user handles manually
- **G11**: Must NOT skip testing CLI dispatch (argv[0] routing is safety-critical — now 3 dispatch modes)
- **G12**: Must NOT generate SMB overrides without basic syntax validation (valid INI structure)
- **G13**: Must NOT leave orphaned modifications after uninstall (exportfs replacement, smbcontrol replacement, systemd drop-in, SMB include line)
- **G14**: Must NOT use `syscall.Exec` in wrapper mode — keep `exec.Command` for error handling simplicity
- **G15**: smbcontrol wrapper MUST fail-open: if `smb inject` fails, exec `smbcontrol.orig` anyway — never break Samba operations due to our tool
- **G16**: `smb inject` MUST be idempotent: check before adding, never duplicate the include line — called from multiple interception points, potentially concurrently
- **G17**: systemd drop-in MUST preserve original reload behavior: if inject fails, SIGHUP must still be sent — Samba must reload even if our injection fails
- **G18**: smbcontrol wrapper MUST pass ALL original arguments unchanged to `smbcontrol.orig` — transparent proxy
- **G19**: `smb inject` MUST NOT require config.yaml — it only manages the include line in smb.conf, not override generation. Keeps inject fast and dependency-free

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed. No exceptions.
> Acceptance criteria requiring "user manually tests/confirms" are FORBIDDEN.

### Test Decision
- **Infrastructure exists**: NO (currently zero test files)
- **Automated tests**: TDD (test-first for all new code)
- **Framework**: `go test` (stdlib `testing` package, no test framework dependency)
- **TDD workflow**: Each task follows RED (write failing test) → GREEN (minimal implementation) → REFACTOR

### QA Policy
Every task MUST include agent-executed QA scenarios.
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **Go code**: Use Bash (`go test`, `go build`, `go vet`) — compile, run tests, verify output
- **Binary behavior**: Use Bash — build binary, invoke with different argv[0], verify dispatch
- **Config parsing**: Use Bash (`go test`) — parse valid/invalid YAML, verify error handling
- **SMB generation**: Use Bash (`go test`) — generate override files, verify INI syntax
- **Shell scripts**: Use interactive_bash (tmux) — run scripts in temp directories, verify file operations

---

## Execution Strategy

### Parallel Execution Waves

> Maximize throughput by grouping independent tasks into parallel waves.
> Each wave completes before the next begins.

```
Wave 1 (Start Immediately — foundation, 5 parallel):
├── T1:  Module rename + cmd/ restructure [quick]
├── T2:  Unified config format with TDD [quick]
├── T3:  NFS parser test suite [quick]
├── T4:  SMB override generator with TDD [quick]
└── T5:  Logging infrastructure [quick]

Wave 2 (After Wave 1 — core features, 5 parallel):
├── T6:   CLI dispatch + subcommand routing with TDD (depends: T1) [quick]
├── T7:   Wrapper modes refactor with TDD — NFS + smbcontrol (depends: T1, T2, T3, T5, T8.5) [deep]
├── T8:   `smb apply` command with TDD (depends: T1, T2, T4, T5) [unspecified-high]
├── T8.5: `smb inject` subcommand + internal/smb/inject.go with TDD (depends: T1, T5) [quick]
└── T9:   Makefile overhaul (depends: T1) [quick]

    NOTE: T7 depends on T8.5 (wrapper calls inject function). T8.5 is small (~30 min)
    and will complete before T7 (multi-hour). Both are Wave 2 but T7 blocks on T8.5.

Wave 3 (After Wave 2 — management commands, 3 parallel):
├── T10: `install` + `uninstall` commands with TDD (depends: T6, T7, T8, T8.5) [unspecified-high]
├── T11: `status` command with TDD (depends: T6) [quick]
└── T12: CI pipeline update (depends: T9) [quick]

Wave 4 (After Wave 3 — packaging + docs, 2 parallel):
├── T13: Install/uninstall shell scripts + migration (depends: T10) [quick]
└── T14: Config example + README rewrite (depends: all) [writing]

Wave FINAL (After ALL tasks — independent review, 4 parallel):
├── F1:  Plan compliance audit (oracle)
├── F2:  Code quality review (unspecified-high)
├── F3:  Real QA via SSH to UNAS (unspecified-high)
└── F4:  Scope fidelity check (deep)

Critical Path: T1 → T8.5 → T7 → T10 → T13 → F1-F4
Parallel Speedup: ~60% faster than sequential
Max Concurrent: 5 (Waves 1 & 2)
```

### Dependency Matrix

| Task | Depends On | Blocks | Wave |
|------|-----------|--------|------|
| T1   | —         | T6, T7, T8, T8.5, T9 | 1 |
| T2   | —         | T7, T8 | 1 |
| T3   | —         | T7 | 1 |
| T4   | —         | T8 | 1 |
| T5   | —         | T7, T8, T8.5 | 1 |
| T6   | T1        | T10, T11 | 2 |
| T7   | T1,T2,T3,T5,T8.5 | T10 | 2 |
| T8   | T1,T2,T4,T5 | T10 | 2 |
| T8.5 | T1,T5     | T7, T10 | 2 |
| T9   | T1        | T12 | 2 |
| T10  | T6,T7,T8,T8.5 | T13 | 3 |
| T11  | T6        | — | 3 |
| T12  | T9        | — | 3 |
| T13  | T10       | — | 4 |
| T14  | all       | — | 4 |

### Agent Dispatch Summary

- **Wave 1**: 5 tasks — T1→`quick`, T2→`quick`, T3→`quick`, T4→`quick`, T5→`quick`
- **Wave 2**: 5 tasks — T6→`quick`, T7→`deep`, T8→`unspecified-high`, T8.5→`quick`, T9→`quick`
- **Wave 3**: 3 tasks — T10→`unspecified-high`, T11→`quick`, T12→`quick`
- **Wave 4**: 2 tasks — T13→`quick`, T14→`writing`
- **FINAL**: 4 tasks — F1→`oracle`, F2→`unspecified-high`, F3→`unspecified-high`, F4→`deep`

---

## TODOs

> Implementation + Test = ONE Task (TDD). Never separate.
> EVERY task MUST have: Recommended Agent Profile + Parallelization info + QA Scenarios.

- [x] 1. Module Rename + Directory Restructure

  **What to do**:
  - Run `go mod edit -module github.com/bbettridge/unas-custom`
  - Update all import paths in `.go` files: `s/github.com\/bbettridge\/exportfs-wrapper/github.com\/bbettridge\/unas-custom/g`
  - Rename `cmd/exportfs-wrapper/` → `cmd/unas-custom/`
  - Update `main.go` log prefix from `"exportfs-wrapper: "` to `"unas-custom: "`
  - Run `go mod tidy` to verify clean dependency state
  - Run `go build ./...` to confirm compilation succeeds
  - This is a RENAME ONLY commit — no logic changes, no new code

  **Must NOT do**:
  - Do NOT change any logic, constants, or behavior
  - Do NOT add new files or packages
  - Do NOT modify the config path constants yet (those change in T7)
  - Do NOT rename the GitHub repository (that's a separate concern)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Mechanical find-and-replace operation, no design decisions
  - **Skills**: []
    - No specialized skills needed — text replacement + Go tooling

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T2, T3, T4, T5)
  - **Blocks**: T6, T7, T8, T9 (all subsequent tasks use new import paths)
  - **Blocked By**: None (can start immediately)

  **References**:

  **Pattern References**:
  - `go.mod:1` — Current module declaration: `module github.com/bbettridge/exportfs-wrapper`
  - `cmd/exportfs-wrapper/main.go:12-13` — Internal import paths that must change

  **API/Type References**:
  - `internal/nfs/parser.go:10` — Import of `config` package that must update
  - `internal/config/config.go` — No imports to change (only uses stdlib + yaml.v3)

  **External References**:
  - Go module rename: `go mod edit -module NEW_PATH` is the official method
  - Convention: `cmd/{binary-name}/main.go` — directory name = binary name

  **WHY Each Reference Matters**:
  - `go.mod` is the single source of truth for the module path
  - All internal imports (`github.com/bbettridge/exportfs-wrapper/internal/...`) must match go.mod
  - The `cmd/` directory name determines the default binary name from `go build`

  **Acceptance Criteria**:

  - [ ] `go mod edit -module` applied: `grep "module github.com/bbettridge/unas-custom" go.mod`
  - [ ] All imports updated: `grep -r "exportfs-wrapper" --include="*.go" .` returns nothing
  - [ ] Directory renamed: `test -d cmd/unas-custom && ! test -d cmd/exportfs-wrapper`
  - [ ] Build succeeds: `go build ./...` exits 0
  - [ ] `go mod tidy` produces no changes
  - [ ] `go vet ./...` passes

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Module compiles after rename
    Tool: Bash
    Preconditions: Working Go 1.21.4+ installation
    Steps:
      1. Run `go build ./cmd/unas-custom/`
      2. Verify exit code is 0
      3. Run `go vet ./...`
      4. Verify exit code is 0
    Expected Result: Clean build with no errors or warnings
    Failure Indicators: Any "cannot find module" or "import path" errors
    Evidence: .sisyphus/evidence/task-1-module-build.txt

  Scenario: No old module references remain
    Tool: Bash
    Preconditions: Rename complete
    Steps:
      1. Run `grep -r "exportfs-wrapper" --include="*.go" .`
      2. Run `grep -r "exportfs-wrapper" go.mod go.sum`
    Expected Result: Both commands return no matches (exit code 1)
    Failure Indicators: Any match of old module path in Go source files
    Evidence: .sisyphus/evidence/task-1-no-old-refs.txt
  ```

  **Commit**: YES
  - Message: `refactor: rename module to github.com/bbettridge/unas-custom`
  - Files: `go.mod`, `go.sum`, `cmd/unas-custom/main.go`, `internal/nfs/parser.go`
  - Pre-commit: `go build ./...`

- [x] 2. Unified Config Format with TDD

  **What to do**:
  - **RED**: Write tests first in `internal/config/config_test.go`:
    - Test loading unified YAML with both `nfs:` and `smb:` sections
    - Test loading NFS-only config (backward compatibility)
    - Test loading SMB-only config
    - Test empty sections (`nfs: { rules: [] }`)
    - Test missing config file (returns error)
    - Test malformed YAML (returns error)
    - Test validation: empty share name, empty path, empty options
    - Test `FindRule()` with matching and non-matching paths
    - Test `FindSMBOverride()` with matching and non-matching share names
  - **GREEN**: Redesign `internal/config/config.go`:
    - New top-level struct: `Config { NFS NFSConfig; SMB SMBConfig }`
    - `NFSConfig { Rules []NFSRule }` where `NFSRule { Path, Options string }`
    - `SMBConfig { Overrides []SMBOverride }` where `SMBOverride { Share string; Directives map[string]string }`
    - `Load(path)` returns `*Config` — handles both old flat format (backward compat) and new nested format
    - `FindRule(path)` preserved for NFS
    - `FindSMBOverride(share)` added for SMB
    - Validation: Share names non-empty, directive keys non-empty
  - **REFACTOR**: Clean up, ensure idiomatic Go

  **Must NOT do**:
  - Do NOT add new dependencies — use stdlib `testing` package only
  - Do NOT support auto-migration of old config format (just document the new format)
  - Do NOT validate Samba directive names (that's Samba's job)
  - Do NOT add `configPath` constants here (that's in the CLI layer)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Well-scoped data model change with clear test cases
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T1, T3, T4, T5)
  - **Blocks**: T7, T8 (both need the new config types)
  - **Blocked By**: None (can start immediately)

  **References**:

  **Pattern References**:
  - `internal/config/config.go:10-19` — Current Rule and Config structs (must evolve, not break)
  - `internal/config/config.go:22-44` — Current Load() function (signature changes: Config now has NFS/SMB sub-structs)
  - `internal/config/config.go:46-54` — Current FindRule() (preserved for NFS rules)

  **API/Type References**:
  - `internal/nfs/parser.go:10` — Imports `config` package; `nfs.ModifyContent` calls `cfg.FindRule()`
  - `cmd/unas-custom/main.go:29` — Calls `config.Load(configPath)` — return type changes

  **External References**:
  - User's config example from spec:
    ```yaml
    nfs:
      rules:
        - path: "/var/nfs/shared/Backups"
          options: "rw,sync,no_subtree_check,insecure,no_root_squash"
    smb:
      overrides:
        - share: "Media"
          directives:
            guest ok: "yes"
            create mask: "0664"
    ```

  **WHY Each Reference Matters**:
  - `config.go:10-19`: The Rule struct fields (`Path`, `Options`) must be preserved in NFSRule for backward compat with `nfs.ModifyContent`
  - `parser.go:10`: The `nfs` package calls `cfg.FindRule()` — this method must still work on the new Config struct
  - User's config example defines the exact YAML structure to implement

  **Acceptance Criteria**:

  - [ ] Test file created: `internal/config/config_test.go`
  - [ ] `go test -v -race ./internal/config/...` → PASS (minimum 8 test cases)
  - [ ] Unified YAML loads: both `nfs:` and `smb:` sections parsed correctly
  - [ ] NFS backward compat: `FindRule()` works identically to current behavior
  - [ ] SMB types: `FindSMBOverride(share)` returns matching override or nil
  - [ ] Validation: empty share name returns error, empty path returns error
  - [ ] Empty sections valid: `nfs: { rules: [] }` loads without error

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Unified config loads both NFS and SMB sections
    Tool: Bash (go test)
    Preconditions: config_test.go with test for unified YAML
    Steps:
      1. Run `go test -v -run TestLoadUnifiedConfig ./internal/config/`
      2. Verify test creates temp YAML with both nfs: and smb: sections
      3. Verify Load() returns Config with populated NFS.Rules and SMB.Overrides
    Expected Result: PASS — Config.NFS.Rules has entries, Config.SMB.Overrides has entries
    Failure Indicators: yaml.Unmarshal error, nil sections, wrong field values
    Evidence: .sisyphus/evidence/task-2-unified-config.txt

  Scenario: Invalid config returns descriptive error
    Tool: Bash (go test)
    Preconditions: config_test.go with validation test cases
    Steps:
      1. Run `go test -v -run TestConfigValidation ./internal/config/`
      2. Test cases: empty share name, empty NFS path, malformed YAML, missing file
    Expected Result: PASS — each invalid input returns an error containing descriptive message
    Failure Indicators: Panic, nil error for invalid input, generic "parse error"
    Evidence: .sisyphus/evidence/task-2-config-validation.txt
  ```

  **Commit**: YES
  - Message: `feat(config): unified YAML config format with NFS + SMB sections`
  - Files: `internal/config/config.go`, `internal/config/config_test.go`
  - Pre-commit: `go test -race ./internal/config/...`

- [x] 3. NFS Parser Test Suite (Behavior Preservation)

  **What to do**:
  - **RED→GREEN**: Write comprehensive tests in `internal/nfs/parser_test.go` that prove the existing parser behavior:
    - `TestParseExport`: valid export lines with single and multiple clients
    - `TestParseExport_Comments`: lines starting with `#` return nil
    - `TestParseExport_Empty`: empty/whitespace lines return nil
    - `TestParseExport_Invalid`: malformed lines return error
    - `TestExportString`: round-trip: parse → String() reproduces original format
    - `TestModifyContent_MatchingRule`: applies rule to matching export path
    - `TestModifyContent_NoMatch`: non-matching paths pass through unchanged
    - `TestModifyContent_MultipleClients`: rule applies to ALL clients on matching line
    - `TestModifyContent_MixedContent`: file with comments, blanks, and exports
    - `TestModifyContent_EmptyConfig`: no rules = content passes through unchanged
  - Use concrete test data derived from the actual UNAS export format:
    - `/var/nfs/shared/Backups 192.168.1.0/24(rw,sync,no_subtree_check)`
    - `/var/nfs/shared/Media *(rw,async,no_root_squash) 10.0.0.0/8(ro,sync)`
  - This task writes TESTS ONLY — the parser code is not modified (it already works)

  **Must NOT do**:
  - Do NOT modify `internal/nfs/parser.go` — these tests prove EXISTING behavior
  - Do NOT change the import path (that's T1's job — if T1 runs first, use new path)
  - Do NOT add edge cases that don't reflect real UNAS export format

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Writing tests for existing code, no implementation changes
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T1, T2, T4, T5)
  - **Blocks**: T7 (wrapper refactor needs these tests to prove behavior preservation)
  - **Blocked By**: None (tests existing code)

  **References**:

  **Pattern References**:
  - `internal/nfs/parser.go:34-65` — `ParseExport()` function to test: regex parsing, client extraction
  - `internal/nfs/parser.go:68-81` — `Export.String()` serialization to verify round-trip fidelity
  - `internal/nfs/parser.go:84-125` — `ModifyContent()` main business logic: line-by-line processing, rule application

  **API/Type References**:
  - `internal/nfs/parser.go:14-23` — Export and Client structs defining the data model
  - `internal/nfs/parser.go:27-31` — Regex patterns used for parsing (exportLineRegex, clientRegex)
  - `internal/config/config.go:10-19` — Rule and Config types used by ModifyContent

  **External References**:
  - Real UNAS export line format: `/path/to/export client(options)` — e.g., `/var/nfs/shared/Backups 192.168.1.0/24(rw,sync,no_subtree_check)`
  - Current live config: `rules: [{ path: "/var/nfs/shared/Backups", options: "rw,sync,no_subtree_check,insecure,no_root_squash" }]`

  **WHY Each Reference Matters**:
  - `parser.go:34-65`: ParseExport is the entry point for line parsing — tests must cover all regex match paths
  - `parser.go:84-125`: ModifyContent is the core business logic — tests prove exact input→output behavior that T7 must preserve
  - Real UNAS format ensures tests use representative data, not synthetic examples

  **Acceptance Criteria**:

  - [ ] Test file created: `internal/nfs/parser_test.go`
  - [ ] `go test -v -race ./internal/nfs/...` → PASS (minimum 10 test cases)
  - [ ] Coverage on parser.go: `go test -cover ./internal/nfs/...` → >90%
  - [ ] Round-trip fidelity: `ParseExport(line).String()` ≈ original line format
  - [ ] ModifyContent with matching rule: options replaced for ALL clients
  - [ ] ModifyContent with no matching rule: content unchanged byte-for-byte

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Parser tests pass with high coverage
    Tool: Bash (go test)
    Preconditions: parser_test.go written
    Steps:
      1. Run `go test -v -race -cover ./internal/nfs/`
      2. Capture output
      3. Verify all tests PASS
      4. Verify coverage >90%
    Expected Result: PASS — all tests green, coverage >90% on parser.go
    Failure Indicators: Any test failure, coverage <90%
    Evidence: .sisyphus/evidence/task-3-nfs-parser-tests.txt

  Scenario: ModifyContent produces identical output for matching rules
    Tool: Bash (go test)
    Preconditions: Test case with known input/output pair
    Steps:
      1. Run `go test -v -run TestModifyContent_MatchingRule ./internal/nfs/`
      2. Test provides export content with path "/var/nfs/shared/Backups"
      3. Config has rule for that path with new options
      4. Verify output has new options for ALL clients on that line
    Expected Result: PASS — options replaced exactly as specified in rule
    Failure Indicators: Partial replacement, wrong client, original options remain
    Evidence: .sisyphus/evidence/task-3-modify-content.txt
  ```

  **Commit**: YES
  - Message: `test(nfs): comprehensive test suite proving parser behavior preservation`
  - Files: `internal/nfs/parser_test.go`
  - Pre-commit: `go test -race ./internal/nfs/...`

- [x] 4. SMB Override Generator with TDD

  **What to do**:
  - Create new package `internal/smb/`
  - **RED**: Write tests first in `internal/smb/generator_test.go`:
    - `TestGenerateOverrides_SingleShare`: one share, one directive → valid INI section
    - `TestGenerateOverrides_MultipleShares`: multiple shares → multiple INI sections
    - `TestGenerateOverrides_MultipleDirectives`: one share, many directives → all listed
    - `TestGenerateOverrides_EmptyOverrides`: no overrides → empty string (or header comment only)
    - `TestGenerateOverrides_SpecialCharacters`: directives with spaces, colons (e.g., `fruit:time machine = yes`)
    - `TestGenerateOverrides_ValidSyntax`: output is valid Samba INI (section headers in brackets, key = value format)
    - `TestValidateDirectives`: basic syntax validation (non-empty keys, non-empty values)
  - **GREEN**: Implement `internal/smb/generator.go`:
    - `GenerateOverrides(overrides []config.SMBOverride) (string, error)` — produces Samba INI text
    - Output format per override: `[ShareName]\n  key = value\n  key = value\n\n`
    - Include a header comment: `# Generated by unas-custom — do not edit manually`
    - Basic validation: reject empty share names, empty directive keys
  - **REFACTOR**: Ensure consistent indentation (Samba convention: 2-space indent for directives)

  **Must NOT do**:
  - Do NOT validate Samba directive NAMES (e.g., don't check if "guest ok" is a real param)
  - Do NOT shell out to `testparm`
  - Do NOT write files — this package only generates string content
  - Do NOT add dependencies

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Small, well-scoped package with clear input→output contract
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T1, T2, T3, T5)
  - **Blocks**: T8 (`smb apply` command uses this generator)
  - **Blocked By**: None

  **References**:

  **Pattern References**:
  - `internal/nfs/parser.go:68-81` — Export.String() pattern: building string output from struct (follow same StringBuilder pattern)

  **API/Type References**:
  - Config types from T2: `SMBOverride { Share string; Directives map[string]string }` — this is the input type

  **External References**:
  - Live UNAS share.conf example from user spec — shows exact Samba INI format:
    ```ini
    [Media]
      write list = media,uishd-...,admin,unifi-drive-backup
      read list =
      valid users = media,uishd-...,admin,unifi-drive-backup
      path = /volume/.../.srv/.unifi-drive/Media/.data
      writeable = yes
      browseable = yes
      guest ok = no
      force group = unifi-drive
      fruit:time machine = yes
    ```
  - Samba INI format: section `[Name]`, directives `  key = value` (2-space indent, spaces around `=`)

  **WHY Each Reference Matters**:
  - Live share.conf shows the exact formatting smbd expects — the generator must produce matching style
  - parser.go String() shows the pattern for struct→string serialization in this codebase
  - SMBOverride type defines the data contract between config and generator

  **Acceptance Criteria**:

  - [ ] Package created: `internal/smb/generator.go` + `internal/smb/generator_test.go`
  - [ ] `go test -v -race ./internal/smb/...` → PASS (minimum 7 test cases)
  - [ ] Output format: `[ShareName]\n  key = value\n` with 2-space indent
  - [ ] Header comment present in output
  - [ ] Empty overrides → empty/minimal output (not an error)
  - [ ] Special characters preserved: `fruit:time machine = yes`

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Generator produces valid Samba INI for Media share override
    Tool: Bash (go test)
    Preconditions: generator_test.go with concrete test data
    Steps:
      1. Run `go test -v -run TestGenerateOverrides_SingleShare ./internal/smb/`
      2. Test provides: SMBOverride{Share: "Media", Directives: {"guest ok": "yes"}}
      3. Verify output contains `[Media]` section header
      4. Verify output contains `  guest ok = yes` with 2-space indent
    Expected Result: PASS — valid INI section with correct formatting
    Failure Indicators: Missing brackets, wrong indentation, missing directive
    Evidence: .sisyphus/evidence/task-4-smb-generator.txt

  Scenario: Generator rejects invalid input
    Tool: Bash (go test)
    Preconditions: generator_test.go with validation test cases
    Steps:
      1. Run `go test -v -run TestValidateDirectives ./internal/smb/`
      2. Test cases: empty share name → error, empty directive key → error
    Expected Result: PASS — errors returned for invalid input
    Failure Indicators: Nil error for invalid input, panic
    Evidence: .sisyphus/evidence/task-4-smb-validation.txt
  ```

  **Commit**: YES
  - Message: `feat(smb): SMB override generator producing valid Samba INI`
  - Files: `internal/smb/generator.go`, `internal/smb/generator_test.go`
  - Pre-commit: `go test -race ./internal/smb/...`

- [x] 5. Logging Infrastructure

  **What to do**:
  - Create `internal/logging/logger.go`:
    - `Logger` struct wrapping `log.Logger` with file output
    - `New(logPath string) (*Logger, error)` — opens/creates log file for append
    - `Info(msg string, args ...interface{})`, `Warn(...)`, `Error(...)`
    - Log format: `2006-01-02 15:04:05 [LEVEL] message`
    - Default log path: `/persistent/unas-custom/unas-custom.log`
    - `Close()` to flush and close file handle
    - `NewNoop() *Logger` — returns a logger that discards all output (for testing/fallback)
  - **RED**: Write tests in `internal/logging/logger_test.go`:
    - Test writing to temp file, verify format
    - Test log levels appear in output
    - Test Noop logger produces no output
    - Test missing directory returns error
  - **GREEN**: Implement logger
  - **REFACTOR**: Ensure log rotation is NOT added (keep simple — file grows, user rotates manually)

  **Must NOT do**:
  - Do NOT add log rotation (out of scope)
  - Do NOT use syslog or external logging libraries
  - Do NOT make logging mandatory — if log file can't be opened, fall back to Noop logger
  - Do NOT log sensitive data (NFS options, SMB passwords)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Small utility package, ~50 lines
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T1, T2, T3, T4)
  - **Blocks**: T7, T8 (wrapper and smb apply use logging)
  - **Blocked By**: None

  **References**:

  **Pattern References**:
  - `cmd/exportfs-wrapper/main.go:85-87` — Current logging setup: `log.SetFlags(0)`, `log.SetPrefix("exportfs-wrapper: ")` — this is what we're replacing
  - `internal/config/config.go:24-26` — Error return pattern to follow

  **External References**:
  - Go stdlib `log` package: `log.New(writer, prefix, flags)` — foundation for the logger
  - `/persistent/unas-custom/` — target directory for log file on UNAS

  **WHY Each Reference Matters**:
  - Current logging goes to stderr which is invisible when UDC calls the wrapper — file logging fixes this
  - The error return pattern in config.go shows idiomatic error wrapping with `fmt.Errorf`

  **Acceptance Criteria**:

  - [ ] Package created: `internal/logging/logger.go` + `internal/logging/logger_test.go`
  - [ ] `go test -v -race ./internal/logging/...` → PASS
  - [ ] Log format: `2006-01-02 15:04:05 [INFO] message` (timestamp + level + message)
  - [ ] Noop logger: produces no output, no errors
  - [ ] File logger: appends to existing file (not truncate)
  - [ ] Missing directory: returns error (not panic)

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Logger writes to file with correct format
    Tool: Bash (go test)
    Preconditions: logger_test.go with temp file test
    Steps:
      1. Run `go test -v -run TestLoggerWritesToFile ./internal/logging/`
      2. Test creates temp dir, initializes Logger, writes Info message
      3. Reads file content, verifies format matches pattern
    Expected Result: PASS — log file contains timestamped, leveled message
    Failure Indicators: Empty file, wrong format, file not created
    Evidence: .sisyphus/evidence/task-5-logger.txt

  Scenario: Noop logger discards all output
    Tool: Bash (go test)
    Preconditions: logger_test.go with noop test
    Steps:
      1. Run `go test -v -run TestNoopLogger ./internal/logging/`
      2. Create Noop logger, call Info/Warn/Error
      3. Verify no output, no errors, no panics
    Expected Result: PASS — all methods succeed silently
    Failure Indicators: Panic, error return, output produced
    Evidence: .sisyphus/evidence/task-5-noop-logger.txt
  ```

  **Commit**: YES
  - Message: `feat(logging): file-based logger for persistent diagnostics`
  - Files: `internal/logging/logger.go`, `internal/logging/logger_test.go`
  - Pre-commit: `go test -race ./internal/logging/...`

- [ ] 6. CLI Dispatch + Subcommand Routing with TDD

  **What to do**:
  - **RED**: Write tests in `cmd/unas-custom/dispatch_test.go`:
    - `TestDispatch_ExportfsMode`: `os.Args[0]` basename is "exportfs" → returns "nfs-wrapper" mode
    - `TestDispatch_SmbcontrolMode`: `os.Args[0]` basename is "smbcontrol" → returns "smb-wrapper" mode
    - `TestDispatch_UnasCustomMode`: `os.Args[0]` basename is "unas-custom" → returns "cli" mode
    - `TestDispatch_UnknownName`: unknown basename → returns "cli" mode (safe default)
    - `TestParseSubcommand_Install`: args `["install"]` → returns install command
    - `TestParseSubcommand_Uninstall`: args `["uninstall"]` → returns uninstall command
    - `TestParseSubcommand_Status`: args `["status"]` → returns status command
    - `TestParseSubcommand_SmbApply`: args `["smb", "apply"]` → returns smb-apply command
    - `TestParseSubcommand_SmbInject`: args `["smb", "inject"]` → returns smb-inject command
    - `TestParseSubcommand_Help`: args `["--help"]` or no args → prints help text
    - `TestParseSubcommand_Unknown`: args `["foobar"]` → prints error + help text
  - **GREEN**: Implement dispatch in `cmd/unas-custom/main.go`:
    - `func detectMode() string` — returns "nfs-wrapper", "smb-wrapper", or "cli" based on `filepath.Base(os.Args[0])`
    - In `main()`: switch on mode:
      - `"nfs-wrapper"` → call `runNfsWrapper()`
      - `"smb-wrapper"` → call `runSmbcontrolWrapper()`
      - default → call `runCLI()`
    - `func runCLI()` — hand-rolled subcommand parser using `os.Args[1:]`
    - `func printHelp()` — prints usage with all available subcommands (including `smb inject`)
    - Each subcommand handler is a placeholder `func` that will be filled by T7-T11
    - Remove the broken `init()` function entirely
  - **REFACTOR**: Extract dispatch logic into testable functions (not embedded in main)

  **Must NOT do**:
  - Do NOT use Cobra, Kong, urfave/cli, or any CLI framework
  - Do NOT implement subcommand LOGIC (just the routing — T7-T11 fill in the handlers)
  - Do NOT keep the broken `init()` function
  - Do NOT add flag parsing beyond basic subcommand matching
  - Do NOT implement wrapper logic for smbcontrol — just dispatch to a placeholder `runSmbcontrolWrapper()` function

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Small dispatch logic, clear test cases, well-scoped
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with T7, T8, T9)
  - **Blocks**: T10, T11 (need CLI routing to register commands)
  - **Blocked By**: T1 (needs renamed cmd/unas-custom/ directory)

  **References**:

  **Pattern References**:
  - `cmd/exportfs-wrapper/main.go:85-92` — Current broken init() to REMOVE
  - `cmd/exportfs-wrapper/main.go:22-39` — Current main()/run() flow to redesign

  **External References**:
  - siderolabs/talos `cmd/installer/main.go`: `switch filepath.Base(os.Args[0]) { case "imager": imager.Execute(); default: installer.Execute() }` — exact pattern to follow
  - coreos/ignition `internal/main.go`: Similar switch-based multi-call dispatch
  - fwdcloudsec/granted `cmd/granted/main.go`: Uses switch for granted vs assumego modes

  **WHY Each Reference Matters**:
  - Talos pattern is the canonical example: simple switch statement, clean dispatch, default case
  - Current init() must be understood to properly remove it without losing safety intent
  - The run() function shows the current "load config → modify → exec" flow that becomes the wrapper path

  **Acceptance Criteria**:

  - [ ] `go test -v -race ./cmd/unas-custom/...` → PASS (minimum 11 test cases)
  - [ ] `filepath.Base(os.Args[0]) == "exportfs"` → nfs-wrapper mode
  - [ ] `filepath.Base(os.Args[0]) == "smbcontrol"` → smb-wrapper mode
  - [ ] Default/unknown basename → CLI mode (safe fallback)
  - [ ] `init()` function removed entirely
  - [ ] Help text shows all subcommands: install, uninstall, status, smb apply, smb inject
  - [ ] Unknown subcommand → stderr error + help text + exit 1
  - [ ] `smb apply` and `smb inject` parsed as two-word subcommands

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Dispatch detects NFS wrapper mode from argv[0]="exportfs"
    Tool: Bash (go test)
    Preconditions: dispatch_test.go written
    Steps:
      1. Run `go test -v -run TestDispatch_ExportfsMode ./cmd/unas-custom/`
      2. Test sets os.Args[0] to a path ending in "exportfs"
      3. Verify detectMode() returns "nfs-wrapper"
    Expected Result: PASS — nfs-wrapper mode detected
    Failure Indicators: Returns "cli" or "smb-wrapper" mode for exportfs basename
    Evidence: .sisyphus/evidence/task-6-dispatch-nfs-wrapper.txt

  Scenario: Dispatch detects SMB wrapper mode from argv[0]="smbcontrol"
    Tool: Bash (go test)
    Preconditions: dispatch_test.go written
    Steps:
      1. Run `go test -v -run TestDispatch_SmbcontrolMode ./cmd/unas-custom/`
      2. Test sets os.Args[0] to a path ending in "smbcontrol"
      3. Verify detectMode() returns "smb-wrapper"
    Expected Result: PASS — smb-wrapper mode detected
    Failure Indicators: Returns "cli" or "nfs-wrapper" mode for smbcontrol basename
    Evidence: .sisyphus/evidence/task-6-dispatch-smb-wrapper.txt

  Scenario: Unknown subcommand shows error and help
    Tool: Bash (go test)
    Preconditions: dispatch_test.go with unknown subcommand test
    Steps:
      1. Run `go test -v -run TestParseSubcommand_Unknown ./cmd/unas-custom/`
      2. Test provides args ["foobar"]
      3. Verify stderr contains "unknown command" and help text
    Expected Result: PASS — error message + help text shown
    Failure Indicators: Panic, no error message, wrong exit behavior
    Evidence: .sisyphus/evidence/task-6-unknown-cmd.txt
  ```

  **Commit**: YES
  - Message: `feat(cli): busybox-style dispatch + hand-rolled subcommand routing`
  - Files: `cmd/unas-custom/main.go`, `cmd/unas-custom/dispatch_test.go`
  - Pre-commit: `go test -race ./cmd/unas-custom/...`

- [ ] 7. Wrapper Modes Refactor with TDD — NFS + smbcontrol

  **What to do**:
  - **RED**: Write tests in `cmd/unas-custom/wrapper_test.go`:
    - **NFS wrapper tests**:
      - `TestRunNfsWrapper_Success`: config loads, exports modified, original exec'd
      - `TestRunNfsWrapper_MissingConfig`: config file missing → log warning, exec original without modifications (FAIL-OPEN)
      - `TestRunNfsWrapper_InvalidConfig`: malformed YAML → log warning, exec original without modifications (FAIL-OPEN)
      - `TestRunNfsWrapper_MissingOriginal`: exportfs.orig doesn't exist → error message, exit non-zero
      - `TestRunNfsWrapper_NoMatchingRules`: valid config, no matching paths → exports pass through unchanged
    - **smbcontrol wrapper tests**:
      - `TestRunSmbcontrolWrapper_Success`: calls smb.Inject() → exec original smbcontrol.orig with all args
      - `TestRunSmbcontrolWrapper_InjectFails`: smb.Inject() errors → log warning, exec original anyway (FAIL-OPEN per G15)
      - `TestRunSmbcontrolWrapper_MissingOriginal`: smbcontrol.orig doesn't exist → error message, exit non-zero
      - `TestRunSmbcontrolWrapper_ArgsPassthrough`: ALL original arguments passed unchanged to smbcontrol.orig (G18)
      - `TestRunSmbcontrolWrapper_ExitCodePropagation`: exit code from smbcontrol.orig propagated correctly
  - **GREEN**: Implement in `cmd/unas-custom/wrapper.go`:
    - `func runNfsWrapper(args []string) int` — NFS wrapper mode (existing logic refactored)
      - Use unified config format from T2 (`cfg.NFS.Rules`)
      - Use logger from T5 for all diagnostic output
      - **FAIL-OPEN**: If config fails to load → log warning → skip modification → exec original
      - **FAIL-OPEN**: If export modification fails → log error → exec original with unmodified exports
      - Only fail hard if original binary (`exportfs.orig`) doesn't exist
      - Update constants:
        - `originalExportfs = "/usr/sbin/exportfs.orig"`
        - `configPath = "/persistent/unas-custom/config.yaml"`
        - `exportsDir = "/etc/exports.d"`
      - Keep `exec.Command` approach (not `syscall.Exec`) per G14
      - Pass `os.Args[1:]` to original (current behavior, preserved)
    - `func runSmbcontrolWrapper(args []string) int` — smbcontrol wrapper mode (NEW)
      - Call `smb.Inject(smbConfPath, includeLine)` from T8.5's `internal/smb/inject.go`
      - If inject fails → log warning → continue (FAIL-OPEN per G15)
      - Exec `smbcontrol.orig` with ALL original args unchanged (G18)
      - Propagate exit code from original smbcontrol.orig
      - Constants:
        - `originalSmbcontrol = "/usr/bin/smbcontrol.orig"`
        - `smbConfPath = "/etc/samba/smb.conf"`
        - `includeLine = "include = /persistent/unas-custom/smb-overrides.conf"`
  - **REFACTOR**: Extract `modifyExports()` and `execOriginal()` into shared testable functions used by both wrappers

  **Must NOT do**:
  - Do NOT change NFS parsing logic (that's in `internal/nfs/parser.go`, tested by T3)
  - Do NOT use `syscall.Exec` — keep `exec.Command` per G14
  - Do NOT `log.Fatalf` or `os.Exit(1)` on config errors — FAIL-OPEN per G2, G15
  - Do NOT change the `FindRule()` lookup behavior — it works via `cfg.NFS.Rules` now
  - Do NOT implement `smb.Inject()` itself — that's T8.5's deliverable, just import and call it
  - Do NOT filter smbcontrol invocations by argument — inject on ALL invocations (idempotent)

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Critical refactor — behavior preservation is safety-critical, fail-open logic is nuanced, two wrapper modes
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (but blocks on T8.5 within Wave 2)
  - **Parallel Group**: Wave 2 (with T6, T8, T8.5, T9)
  - **Blocks**: T10 (install command needs wrapper modes working)
  - **Blocked By**: T1 (module rename), T2 (config types), T3 (parser tests prove behavior), T5 (logging), T8.5 (smb.Inject function)

  **References**:

  **Pattern References**:
  - `cmd/exportfs-wrapper/main.go:28-39` — Current run() function: Load → modifyExports → execOriginal (this flow is preserved with fail-open wrapping)
  - `cmd/exportfs-wrapper/main.go:41-67` — Current modifyExports() and WalkDir logic (extract and test)
  - `cmd/exportfs-wrapper/main.go:69-83` — Current execOriginal() with exec.Command and exit code propagation (preserve exactly, reuse for smbcontrol wrapper)

  **API/Type References**:
  - `internal/config/config.go` (from T2) — New Config struct with `NFS.Rules` field
  - `internal/nfs/parser.go:84` — `ModifyContent(content, cfg)` — signature may need adapter since cfg type changes
  - `internal/smb/inject.go` (from T8.5) — `Inject(smbConfPath, includeLine string) error` — called by smbcontrol wrapper
  - `internal/logging/logger.go` (from T5) — Logger for diagnostic output

  **WHY Each Reference Matters**:
  - `main.go:28-39`: The exact flow (Load → modify → exec) is preserved but wrapped in fail-open error handling
  - `main.go:69-83`: The exec.Command pattern with exit code propagation is the proven, working approach — reused for BOTH wrappers
  - `smb/inject.go`: The smbcontrol wrapper calls this to inject the include line before exec'ing original smbcontrol
  - ModifyContent signature: currently takes `*config.Config`, will need to accept new config struct — the adapter pattern in FindRule() handles this

  **Acceptance Criteria**:

  - [ ] `go test -v -race ./cmd/unas-custom/...` includes ALL wrapper tests → PASS (10 test cases minimum)
  - [ ] NFS: Missing config → logs warning, executes original exportfs without modifications
  - [ ] NFS: Invalid config → logs warning, executes original exportfs without modifications
  - [ ] NFS: Missing exportfs.orig → clear error message, non-zero exit
  - [ ] NFS: Valid config + matching rules → modifies exports, then executes original
  - [ ] NFS: Exit code from original exportfs is propagated correctly
  - [ ] SMB: Calls `smb.Inject()` before exec'ing original smbcontrol
  - [ ] SMB: If inject fails → logs warning, execs `smbcontrol.orig` anyway (FAIL-OPEN per G15)
  - [ ] SMB: Missing smbcontrol.orig → clear error message, non-zero exit
  - [ ] SMB: ALL original arguments passed through unchanged to smbcontrol.orig (G18)
  - [ ] SMB: Exit code from original smbcontrol propagated correctly
  - [ ] Config path updated to `/persistent/unas-custom/config.yaml`

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: NFS wrapper fails open on missing config
    Tool: Bash (go test)
    Preconditions: wrapper_test.go with mock original binary
    Steps:
      1. Run `go test -v -run TestRunNfsWrapper_MissingConfig ./cmd/unas-custom/`
      2. Test sets configPath to non-existent file
      3. Test sets originalExportfs to a mock binary (echo script)
      4. Verify: warning logged, mock binary executed, exit code 0
    Expected Result: PASS — wrapper continues despite missing config
    Failure Indicators: os.Exit(1), log.Fatalf, panic
    Evidence: .sisyphus/evidence/task-7-nfs-failopen-missing-config.txt

  Scenario: NFS wrapper preserves modification behavior
    Tool: Bash (go test)
    Preconditions: wrapper_test.go with temp exports dir
    Steps:
      1. Run `go test -v -run TestRunNfsWrapper_Success ./cmd/unas-custom/`
      2. Create temp dir with .exports file containing known content
      3. Create temp config.yaml with matching rule
      4. Run wrapper logic (without exec'ing real binary — mock it)
      5. Read modified .exports file, verify options changed
    Expected Result: PASS — exports modified identically to current parser behavior
    Failure Indicators: Options not changed, wrong clients modified, file corrupted
    Evidence: .sisyphus/evidence/task-7-nfs-preservation.txt

  Scenario: smbcontrol wrapper injects and execs original
    Tool: Bash (go test)
    Preconditions: wrapper_test.go with mock smb.conf and mock smbcontrol.orig
    Steps:
      1. Run `go test -v -run TestRunSmbcontrolWrapper_Success ./cmd/unas-custom/`
      2. Create temp smb.conf without include line
      3. Set smbcontrol.orig to a mock binary
      4. Run smbcontrol wrapper with args ["123", "reload-config"]
      5. Verify: include line added to smb.conf, mock binary executed with same args
    Expected Result: PASS — inject ran, original exec'd with correct args
    Failure Indicators: smb.conf not modified, args mangled, mock not executed
    Evidence: .sisyphus/evidence/task-7-smbcontrol-wrapper-success.txt

  Scenario: smbcontrol wrapper fails open on inject error
    Tool: Bash (go test)
    Preconditions: wrapper_test.go with unwritable smb.conf
    Steps:
      1. Run `go test -v -run TestRunSmbcontrolWrapper_InjectFails ./cmd/unas-custom/`
      2. Set smb.conf path to unwritable location
      3. Set smbcontrol.orig to a mock binary
      4. Run smbcontrol wrapper
      5. Verify: warning logged, mock binary still executed
    Expected Result: PASS — inject failure doesn't block smbcontrol execution
    Failure Indicators: os.Exit(1), panic, smbcontrol.orig not executed
    Evidence: .sisyphus/evidence/task-7-smbcontrol-failopen.txt
  ```

  **Commit**: YES
  - Message: `refactor(wrapper): NFS + smbcontrol fail-open wrapper modes with logging`
  - Files: `cmd/unas-custom/wrapper.go`, `cmd/unas-custom/wrapper_test.go`, `cmd/unas-custom/main.go`
  - Pre-commit: `go test -race ./...`

- [ ] 8. `smb apply` Command with TDD

  **What to do**:
  - **RED**: Write tests in `cmd/unas-custom/smb_test.go`:
    - `TestSmbApply_GeneratesFile`: valid config with SMB overrides → writes smb-overrides.conf
    - `TestSmbApply_EmptyOverrides`: no SMB overrides in config → writes empty/minimal file (or skips)
    - `TestSmbApply_MissingConfig`: no config file → error message
    - `TestSmbApply_InvalidConfig`: malformed config → error message
    - `TestSmbApply_CreatesDirectory`: target directory doesn't exist → creates it
    - `TestSmbApply_FileContent`: verify generated file content matches expected Samba INI
  - **GREEN**: Implement `cmd/unas-custom/smb.go`:
    - `func runSmbApply(args []string) int` — returns exit code
    - Load config from `/persistent/unas-custom/config.yaml`
    - Call `smb.GenerateOverrides(cfg.SMB.Overrides)` from T4
    - Write result to `/persistent/unas-custom/smb-overrides.conf`
    - Log success/failure via logger from T5
    - Print summary to stdout: "Generated N share overrides to /persistent/unas-custom/smb-overrides.conf"
    - Return 0 on success, 1 on error
  - **REFACTOR**: Extract file paths as configurable (for testing with temp dirs)

  **Must NOT do**:
  - Do NOT shell out to `testparm` for validation
  - Do NOT modify `/etc/samba/smb.conf` (that's the install command's job)
  - Do NOT reload smbd (that's the install command's job)
  - Do NOT create Samba shares — only generate the overrides file

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Integrates config loading + SMB generation + file I/O with proper error handling
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with T6, T7, T9)
  - **Blocks**: T10 (install command calls smb apply)
  - **Blocked By**: T1 (module rename), T2 (config types), T4 (SMB generator), T5 (logging)

  **References**:

  **Pattern References**:
  - `cmd/exportfs-wrapper/main.go:28-39` — Current run() flow pattern: load config → do work → return error
  - `internal/smb/generator.go` (from T4) — `GenerateOverrides()` function to call

  **API/Type References**:
  - `internal/config/config.go` (from T2) — `Config.SMB.Overrides` field
  - `internal/smb/generator.go` (from T4) — `GenerateOverrides(overrides []SMBOverride) (string, error)`
  - `internal/logging/logger.go` (from T5) — Logger for diagnostic output

  **External References**:
  - Target file: `/persistent/unas-custom/smb-overrides.conf`
  - This file will be included by Samba via `include = /persistent/unas-custom/smb-overrides.conf` in smb.conf

  **WHY Each Reference Matters**:
  - The generator from T4 produces the content; this command handles config loading, file writing, and user feedback
  - The output path `/persistent/unas-custom/smb-overrides.conf` is what the install command will reference in smb.conf

  **Acceptance Criteria**:

  - [ ] `go test -v -race -run TestSmb ./cmd/unas-custom/...` → PASS
  - [ ] Valid config → generates smb-overrides.conf with correct INI content
  - [ ] Missing config → error message, exit 1
  - [ ] Empty SMB overrides → writes minimal file (header comment only)
  - [ ] Output path: `/persistent/unas-custom/smb-overrides.conf`
  - [ ] Success message printed to stdout

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: smb apply generates valid override file
    Tool: Bash (go test)
    Preconditions: smb_test.go with temp directory
    Steps:
      1. Run `go test -v -run TestSmbApply_GeneratesFile ./cmd/unas-custom/`
      2. Test creates temp config.yaml with SMB override for [Media] share
      3. Runs smb apply logic with temp output path
      4. Reads generated file, verifies [Media] section with correct directives
    Expected Result: PASS — file contains valid Samba INI with Media section
    Failure Indicators: File not created, wrong content, syntax errors
    Evidence: .sisyphus/evidence/task-8-smb-apply.txt

  Scenario: smb apply handles missing config gracefully
    Tool: Bash (go test)
    Preconditions: smb_test.go with no config test
    Steps:
      1. Run `go test -v -run TestSmbApply_MissingConfig ./cmd/unas-custom/`
      2. Point to non-existent config file
      3. Verify error message mentions config file
    Expected Result: PASS — clear error, exit code 1
    Failure Indicators: Panic, generic error, exit 0
    Evidence: .sisyphus/evidence/task-8-smb-missing-config.txt
  ```

  **Commit**: YES
  - Message: `feat(smb): smb apply command generating overrides from config`
  - Files: `cmd/unas-custom/smb.go`, `cmd/unas-custom/smb_test.go`
  - Pre-commit: `go test ./...`

- [ ] 8.5. `smb inject` Subcommand + internal/smb/inject.go with TDD

  **What to do**:
  - **RED**: Write tests in `internal/smb/inject_test.go`:
    - `TestInject_AddsIncludeLine`: smb.conf exists without include line → adds it after `include = /etc/samba/share.conf`
    - `TestInject_AlreadyPresent`: include line already in smb.conf → no-op, file unchanged
    - `TestInject_NoShareConfLine`: smb.conf exists but no `include = /etc/samba/share.conf` → appends include to end of file
    - `TestInject_PreservesContent`: all other smb.conf content preserved exactly (comments, sections, directives)
    - `TestInject_MissingSmbConf`: smb.conf doesn't exist → returns error
    - `TestInject_AtomicWrite`: verifies write uses temp file + rename pattern (no partial reads)
    - `TestInject_DuplicatePrevention`: even with concurrent simulated calls, never duplicates the include line
    - `TestInject_EmptyFile`: smb.conf is empty → appends include line
  - **GREEN**: Implement `internal/smb/inject.go`:
    - `func Inject(smbConfPath string, includeLine string) error`
      - Reads `smbConfPath`
      - Checks if `includeLine` is already present → return nil (no-op)
      - Finds `include = /etc/samba/share.conf` line → inserts our `includeLine` AFTER it
      - If share.conf line not found → appends `includeLine` to end of file
      - Writes to temp file (`smbConfPath + ".tmp"`) then `os.Rename` to `smbConfPath` (atomic write)
      - Returns nil on success, error on failure
    - `func Remove(smbConfPath string, includeLine string) error`
      - Reads `smbConfPath`
      - Removes all instances of `includeLine` (handles duplicates gracefully)
      - Writes atomically (temp + rename)
      - No-op if line not present
    - Constants:
      - `DefaultSmbConfPath = "/etc/samba/smb.conf"`
      - `DefaultIncludeLine = "include = /persistent/unas-custom/smb-overrides.conf"`
      - `ShareConfInclude = "include = /etc/samba/share.conf"`
  - Wire as CLI subcommand in `cmd/unas-custom/smb.go`:
    - `func runSmbInject(args []string) int` — calls `smb.Inject()`, prints result, returns exit code
  - **REFACTOR**: Ensure Inject/Remove functions are pure (no global state, all paths parameterized)

  **Must NOT do**:
  - Do NOT require config.yaml — `smb inject` only manages the include line, not override generation (G19)
  - Do NOT validate smb.conf syntax — just add/remove one line
  - Do NOT reload smbd — the caller (wrapper or systemd) handles that
  - Do NOT filter by smbcontrol arguments — inject is unconditional

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Small, well-scoped function with clear input/output and no external dependencies
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with T6, T7, T8, T9)
  - **Blocks**: T7 (smbcontrol wrapper calls Inject), T10 (install calls Inject, uninstall calls Remove)
  - **Blocked By**: T1 (module rename), T5 (logging)

  **References**:

  **Pattern References**:
  - `cmd/exportfs-wrapper/main.go:41-67` — Current modifyExports() file modification pattern: read → modify → write. Similar approach for smb.conf injection.

  **API/Type References**:
  - `internal/logging/logger.go` (from T5) — Logger for diagnostic output

  **External References**:
  - Samba smb.conf format: plain text, one directive per line, `include = <path>` syntax
  - UNAS smb.conf contains `include = /etc/samba/share.conf` — our line goes AFTER this
  - Atomic write pattern: `os.WriteFile(path+".tmp", data, perm)` → `os.Rename(path+".tmp", path)`

  **WHY Each Reference Matters**:
  - The file modification pattern from exportfs wrapper is the proven approach — read, check, modify, write
  - The include line placement (after share.conf) ensures our overrides take precedence (last-wins in Samba)
  - Atomic write prevents smbd from reading a half-written smb.conf during its 3-minute periodic re-read

  **Acceptance Criteria**:

  - [ ] `go test -v -race ./internal/smb/...` → PASS (minimum 8 test cases for inject)
  - [ ] Include line absent → Inject adds it after share.conf include
  - [ ] Include line present → Inject is no-op, returns nil
  - [ ] share.conf include not found → appends to end of file
  - [ ] All other smb.conf content preserved byte-for-byte
  - [ ] Missing smb.conf → returns error (not panic)
  - [ ] Atomic write: uses temp file + os.Rename
  - [ ] Remove function: removes include line, no-op if not present
  - [ ] CLI `smb inject` subcommand wired and returns correct exit code
  - [ ] Does NOT require config.yaml to exist (G19)

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: smb inject adds include line after share.conf
    Tool: Bash (go test)
    Preconditions: inject_test.go with mock smb.conf containing share.conf include
    Steps:
      1. Run `go test -v -run TestInject_AddsIncludeLine ./internal/smb/`
      2. Test creates temp smb.conf with:
         [global]
         workgroup = WORKGROUP
         include = /etc/samba/share.conf
      3. Calls Inject(tempPath, "include = /persistent/unas-custom/smb-overrides.conf")
      4. Reads result, verifies our include line appears on the line AFTER share.conf include
      5. Verifies all original content preserved
    Expected Result: PASS — include line inserted at correct position
    Failure Indicators: Line missing, wrong position, original content corrupted
    Evidence: .sisyphus/evidence/task-8.5-inject-adds-line.txt

  Scenario: smb inject is idempotent
    Tool: Bash (go test)
    Preconditions: inject_test.go with smb.conf already containing our include
    Steps:
      1. Run `go test -v -run TestInject_AlreadyPresent ./internal/smb/`
      2. Test creates temp smb.conf with our include line already present
      3. Records file content/checksum before
      4. Calls Inject()
      5. Verifies file unchanged (same checksum)
    Expected Result: PASS — no modification, no duplication
    Failure Indicators: Duplicate include line, file modification timestamp changed
    Evidence: .sisyphus/evidence/task-8.5-inject-idempotent.txt

  Scenario: smb inject falls back to append when share.conf not found
    Tool: Bash (go test)
    Preconditions: inject_test.go with minimal smb.conf (no share.conf include)
    Steps:
      1. Run `go test -v -run TestInject_NoShareConfLine ./internal/smb/`
      2. Test creates temp smb.conf without share.conf include line
      3. Calls Inject()
      4. Verifies our include line appended to end of file
    Expected Result: PASS — include line at end of file
    Failure Indicators: Error returned, file unchanged, line not appended
    Evidence: .sisyphus/evidence/task-8.5-inject-fallback-append.txt

  Scenario: smb inject handles missing smb.conf
    Tool: Bash (go test)
    Preconditions: inject_test.go with non-existent path
    Steps:
      1. Run `go test -v -run TestInject_MissingSmbConf ./internal/smb/`
      2. Set smbConfPath to non-existent file
      3. Call Inject()
      4. Verify error returned (not panic)
    Expected Result: PASS — returns descriptive error
    Failure Indicators: Panic, nil error, file created from nothing
    Evidence: .sisyphus/evidence/task-8.5-inject-missing-conf.txt
  ```

  **Commit**: YES
  - Message: `feat(smb): idempotent smb.conf include line injection with atomic write`
  - Files: `internal/smb/inject.go`, `internal/smb/inject_test.go`, `cmd/unas-custom/smb.go`
  - Pre-commit: `go test -race ./internal/smb/...`

- [ ] 9. Makefile Overhaul

  **What to do**:
  - Update `Makefile`:
    - `BINARY_NAME=unas-custom` (was `exportfs-wrapper`)
    - `CMD_DIR=cmd/unas-custom` (was `cmd/exportfs-wrapper`)
    - Keep `build`, `build-arm64`, `clean`, `deploy` targets
    - Add `symlink` target: creates `exportfs` AND `smbcontrol` symlinks pointing to `unas-custom` in build dir
    - Update deploy target:
      - Package includes: binary, scripts, config.example.yaml
      - Deploy path hint: `scp -r build/deploy/* root@unas:/persistent/unas-custom/`
    - Add `test` target: `go test -v -race ./...`
    - Add `lint` target: `go vet ./... && test -z "$$(gofmt -l .)"`
    - Preserve build flags: `CGO_ENABLED=0 GOOS=linux GOARCH=arm64 -ldflags="-s -w"`
  - Verify `make build-arm64` produces working binary

  **Must NOT do**:
  - Do NOT add goreleaser config
  - Do NOT change the cross-compilation approach
  - Do NOT add Docker-based builds

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Mechanical variable renaming + small target additions
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with T6, T7, T8)
  - **Blocks**: T12 (CI update references Makefile targets)
  - **Blocked By**: T1 (needs renamed cmd/ directory)

  **References**:

  **Pattern References**:
  - `Makefile:1-32` — Current Makefile: BINARY_NAME, CMD_DIR, build targets (read entirely for rename)

  **WHY Each Reference Matters**:
  - Every line with `$(BINARY_NAME)` or `$(CMD_DIR)` needs to reference the new names
  - The deploy target's SCP hint needs updated path

  **Acceptance Criteria**:

  - [ ] `make build-arm64` succeeds and produces `build/unas-custom-arm64`
  - [ ] `file build/unas-custom-arm64` shows ARM aarch64, statically linked
  - [ ] `make test` runs all tests
  - [ ] `make lint` runs vet + format check
  - [ ] Deploy target creates package with correct binary name

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: make build-arm64 produces correct binary
    Tool: Bash
    Preconditions: Makefile updated
    Steps:
      1. Run `make clean && make build-arm64`
      2. Verify `build/unas-custom-arm64` exists
      3. Run `file build/unas-custom-arm64`
      4. Verify output contains "ARM aarch64" and "statically linked"
    Expected Result: Static ARM64 binary named unas-custom-arm64
    Failure Indicators: Old binary name, dynamic linking, wrong architecture
    Evidence: .sisyphus/evidence/task-9-makefile-build.txt

  Scenario: No old binary name references in Makefile
    Tool: Bash
    Preconditions: Makefile updated
    Steps:
      1. Run `grep "exportfs-wrapper" Makefile`
    Expected Result: No matches (exit code 1)
    Failure Indicators: Any reference to old binary name
    Evidence: .sisyphus/evidence/task-9-makefile-no-old-refs.txt
  ```

  **Commit**: YES
  - Message: `build: update Makefile for unas-custom binary and symlink support`
  - Files: `Makefile`
  - Pre-commit: `make build-arm64`

- [ ] 10. `install` + `uninstall` Commands with TDD

  **What to do**:
  - Create `internal/installer/installer.go` with install/uninstall logic (testable, not in cmd/)
  - **RED**: Write tests in `internal/installer/installer_test.go`:
    - **NFS Install tests (using temp directories — NOT real system paths)**:
      - `TestInstall_NFS_BackupsExportfs`: copies binary to exportfs path, backs up original
      - `TestInstall_NFS_AlreadyInstalled`: re-running install is idempotent (doesn't re-backup)
    - **smbcontrol Install tests**:
      - `TestInstall_SMB_BackupsSmbcontrol`: backs up `/usr/bin/smbcontrol` → `/usr/bin/smbcontrol.orig`, installs our binary
      - `TestInstall_SMB_SmbcontrolAlreadyInstalled`: idempotent — doesn't re-backup
    - **systemd drop-in tests**:
      - `TestInstall_SMB_CreatesDropIn`: creates `/etc/systemd/system/smbd.service.d/unas-custom.conf` with correct content
      - `TestInstall_SMB_DropInContent`: verifies drop-in contains: `ExecReload=` (clear), `ExecReload=/persistent/unas-custom/unas-custom smb inject`, `ExecReload=/bin/kill -HUP $MAINPID`, `ExecStartPost=/persistent/unas-custom/unas-custom smb inject`
      - `TestInstall_SMB_DropInAlreadyExists`: idempotent — overwrites with correct content
      - `TestInstall_SMB_CreatesDropInDir`: creates parent dir `smbd.service.d/` if it doesn't exist
    - **Include injection + overrides tests**:
      - `TestInstall_SMB_InjectsIncludeLine`: calls `smb.Inject()` to add include line to smb.conf
      - `TestInstall_GeneratesOverrides`: calls smb apply logic to generate overrides file
    - **General tests**:
      - `TestInstall_CreatesConfigDir`: creates `/persistent/unas-custom/` if not exists
      - `TestInstall_DetectsOldInstallation`: prints message if `/persistent/nfs-intercept/` exists (G3, G10)
    - **Uninstall tests**:
      - `TestUninstall_NFS_RestoresOriginal`: copies exportfs.orig back to exportfs, removes .orig
      - `TestUninstall_NFS_NoBackup`: prints message, no error
      - `TestUninstall_SMB_RestoresSmbcontrol`: copies smbcontrol.orig back to smbcontrol, removes .orig
      - `TestUninstall_SMB_SmbcontrolNoBackup`: prints message, no error
      - `TestUninstall_SMB_RemovesDropIn`: removes systemd drop-in file
      - `TestUninstall_SMB_RemovesIncludeLine`: calls `smb.Remove()` to remove include line from smb.conf
      - `TestUninstall_SMB_NoIncludeLine`: no-op if include line not present
      - `TestUninstall_Idempotent`: running twice is safe
  - **GREEN**: Implement installer:
    - `type Installer struct` with configurable paths (for testing with temp dirs):
      - `BinaryPath`, `ExportfsPath`, `ExportfsOrigPath`
      - `SmbcontrolPath`, `SmbcontrolOrigPath`
      - `SmbConfPath`, `ConfigDir`, `OverridesPath`
      - `DropInDir`, `DropInPath`
      - `DaemonReloadFn func() error` — allows mocking `systemctl daemon-reload` in tests
    - `func NewInstaller(opts ...Option) *Installer` — defaults to real system paths, options override for tests
    - `func (i *Installer) Install(binaryPath string) error` — performs ALL install steps:
      1. NFS: copy binary → exportfs path, backup original to .orig (if not already backed up)
      2. SMB: backup smbcontrol → smbcontrol.orig, install our binary as smbcontrol
      3. SMB: create systemd drop-in dir + write drop-in file
      4. Run `systemctl daemon-reload` (via DaemonReloadFn)
      5. SMB: call `smb.Inject()` to add include line to smb.conf
      6. SMB: generate overrides file by running smb apply logic
      7. Migration: check for `/persistent/nfs-intercept/`, print notice if found
    - `func (i *Installer) Uninstall() error` — reverses ALL install steps:
      1. NFS: restore exportfs.orig → exportfs, remove .orig
      2. SMB: restore smbcontrol.orig → smbcontrol, remove .orig
      3. SMB: remove systemd drop-in file
      4. Run `systemctl daemon-reload` (via DaemonReloadFn)
      5. SMB: call `smb.Remove()` to remove include line from smb.conf
    - Systemd drop-in content (exact):
      ```ini
      # Managed by unas-custom — do not edit manually
      [Service]
      ExecStartPost=/persistent/unas-custom/unas-custom smb inject
      ExecReload=
      ExecReload=/persistent/unas-custom/unas-custom smb inject
      ExecReload=/bin/kill -HUP $MAINPID
      ```
  - **REFACTOR**: Ensure all file operations use `os.WriteFile` with proper permissions

  **Must NOT do**:
  - Do NOT auto-migrate old config from `/persistent/nfs-intercept/` — print message only (G3, G10)
  - Do NOT delete `/persistent/nfs-intercept/` — user handles manually (G10)
  - Do NOT add systemd units for unas-custom itself — the drop-in is for smbd.service (G7)
  - Do NOT hardcode paths in the implementation — use struct fields for testability
  - Do NOT implement `smb.Inject()`/`smb.Remove()` — those are from T8.5, just call them
  - Do NOT call `systemctl daemon-reload` directly — use `DaemonReloadFn` for testability

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: System-level file operations with idempotency requirements, multiple interception mechanisms, systemd integration
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with T11, T12)
  - **Blocks**: T13 (shell scripts wrap these commands)
  - **Blocked By**: T6 (CLI dispatch), T7 (wrapper mode), T8 (smb apply), T8.5 (smb inject functions)

  **References**:

  **Pattern References**:
  - `scripts/install-wrapper.sh:4-7` — Current path constants: INSTALL_DIR, EXPORTFS_PATH, EXPORTFS_ORIG, CONFIG_FILE
  - `scripts/install-wrapper.sh:40-43` — Current backup logic: check `.orig` exists before backing up — replicate for BOTH exportfs AND smbcontrol
  - `scripts/install-wrapper.sh:45-47` — Current install logic: copy wrapper to exportfs path
  - `scripts/uninstall-wrapper.sh:9-16` — Current restore logic: copy .orig back, remove .orig — replicate for BOTH

  **API/Type References**:
  - `internal/smb/generator.go` (from T4) — `GenerateOverrides()` for SMB override file generation
  - `internal/smb/inject.go` (from T8.5) — `Inject()` and `Remove()` for smb.conf include line management
  - `internal/config/config.go` (from T2) — `Config.SMB.Overrides` for SMB config data

  **External References**:
  - smbcontrol location: `/usr/bin/smbcontrol` (verified from UDC reverse engineering)
  - exportfs location: `/usr/sbin/exportfs` (verified from current code)
  - systemd drop-in dir: `/etc/systemd/system/smbd.service.d/` (standard systemd override path)
  - Drop-in format: Must clear `ExecReload=` before redefining, otherwise systemd APPENDS

  **WHY Each Reference Matters**:
  - Install script shows the exact idempotency pattern (check .orig before backup) to replicate in Go for BOTH wrappers
  - `smb.Inject()`/`smb.Remove()` from T8.5 handle the include line logic — installer just calls them
  - systemd drop-in must use exact format (clear + redefine) or reload will fire twice
  - `DaemonReloadFn` pattern enables testing without real systemctl

  **Acceptance Criteria**:

  - [ ] Package created: `internal/installer/installer.go` + `internal/installer/installer_test.go`
  - [ ] `go test -v -race ./internal/installer/...` → PASS (minimum 20 test cases)
  - [ ] Install NFS: binary copied to exportfs, original backed up, idempotent
  - [ ] Install SMB: smbcontrol backed up, our binary installed, idempotent
  - [ ] Install SMB: systemd drop-in created with correct content (ExecStartPost + ExecReload clear + inject + SIGHUP)
  - [ ] Install SMB: drop-in directory created if missing
  - [ ] Install SMB: `systemctl daemon-reload` called
  - [ ] Install SMB: include line injected into smb.conf via `smb.Inject()`
  - [ ] Install SMB: overrides file generated
  - [ ] Uninstall NFS: original exportfs restored, backup removed
  - [ ] Uninstall SMB: original smbcontrol restored, backup removed
  - [ ] Uninstall SMB: drop-in file removed, daemon-reload called
  - [ ] Uninstall SMB: include line removed from smb.conf via `smb.Remove()`
  - [ ] Old installation detected: prints message about `/persistent/nfs-intercept/`
  - [ ] Wire `install` and `uninstall` subcommands in CLI dispatch (T6)

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Install sets up NFS wrapper idempotently
    Tool: Bash (go test)
    Preconditions: installer_test.go with temp directory setup
    Steps:
      1. Run `go test -v -run TestInstall_NFS_AlreadyInstalled ./internal/installer/`
      2. Test creates temp dir with fake exportfs and exportfs.orig
      3. Runs Install() twice
      4. Verifies exportfs.orig unchanged, exportfs is our binary
    Expected Result: PASS — second install doesn't corrupt backup
    Failure Indicators: Backup overwritten with our binary, file corruption
    Evidence: .sisyphus/evidence/task-10-nfs-install-idempotent.txt

  Scenario: Install sets up smbcontrol wrapper
    Tool: Bash (go test)
    Preconditions: installer_test.go with temp directory setup
    Steps:
      1. Run `go test -v -run TestInstall_SMB_BackupsSmbcontrol ./internal/installer/`
      2. Test creates temp dir with fake smbcontrol binary
      3. Runs Install()
      4. Verifies smbcontrol.orig exists (backup), smbcontrol replaced with our binary
    Expected Result: PASS — original backed up, our binary in place
    Failure Indicators: No backup, original not replaced
    Evidence: .sisyphus/evidence/task-10-smbcontrol-install.txt

  Scenario: Install creates systemd drop-in with correct content
    Tool: Bash (go test)
    Preconditions: installer_test.go with temp systemd dir
    Steps:
      1. Run `go test -v -run TestInstall_SMB_DropInContent ./internal/installer/`
      2. Runs Install()
      3. Reads created drop-in file
      4. Verifies contains: `ExecStartPost=`, `ExecReload=` (bare clear), inject command, SIGHUP
    Expected Result: PASS — drop-in has all 4 lines in correct order
    Failure Indicators: Missing clear line, wrong command path, missing ExecStartPost
    Evidence: .sisyphus/evidence/task-10-dropin-content.txt

  Scenario: Uninstall removes all modifications
    Tool: Bash (go test)
    Preconditions: installer_test.go with fully installed state
    Steps:
      1. Run `go test -v -run TestUninstall_Idempotent ./internal/installer/`
      2. Set up installed state: exportfs.orig, smbcontrol.orig, drop-in, include line
      3. Runs Uninstall() twice
      4. Verifies: originals restored, drop-in removed, include line removed, DaemonReloadFn called
    Expected Result: PASS — clean state after uninstall, second run is safe no-op
    Failure Indicators: Orphaned files, panic on second run, DaemonReloadFn not called
    Evidence: .sisyphus/evidence/task-10-uninstall-complete.txt
  ```

  **Commit**: YES
  - Message: `feat(install): unified install/uninstall with NFS + smbcontrol wrappers + systemd drop-in`
  - Files: `internal/installer/installer.go`, `internal/installer/installer_test.go`, `cmd/unas-custom/main.go` (wire subcommands)
  - Pre-commit: `go test -race ./internal/installer/...`

- [ ] 11. `status` Command with TDD

  **What to do**:
  - **RED**: Write tests in `cmd/unas-custom/status_test.go`:
    - `TestStatus_AllInstalled`: all hooks installed → reports all OK
    - `TestStatus_NothingInstalled`: fresh system → reports nothing installed
    - `TestStatus_PartialInstall_NFSOnly`: exportfs replaced but no SMB → reports partial
    - `TestStatus_PartialInstall_NoDropIn`: wrappers installed but drop-in missing → reports partial
    - `TestStatus_BrokenConfig`: config file exists but invalid → reports config error
    - `TestStatus_ExitCodes`: 0 for all OK, 1 for issues found
  - **GREEN**: Implement `cmd/unas-custom/status.go`:
    - `func runStatus(args []string) int` — checks and reports installation state
    - Output format:
      ```
      unas-custom status
      ──────────────────
      NFS wrapper:      installed ✓ (exportfs → unas-custom, original backed up)
      SMB wrapper:      installed ✓ (smbcontrol → unas-custom, original backed up)
      systemd drop-in:  installed ✓ (smbd.service ExecReload override)
      SMB include:      present ✓ (include line in smb.conf)
      SMB overrides:    generated ✓ (3 share overrides)
      Config:           valid ✓ (/persistent/unas-custom/config.yaml)
      ```
    - Checks:
      1. Is `/usr/sbin/exportfs` our binary? + does `exportfs.orig` exist?
      2. Is `/usr/bin/smbcontrol` our binary? + does `smbcontrol.orig` exist?
      3. Does systemd drop-in exist at `/etc/systemd/system/smbd.service.d/unas-custom.conf`?
      4. Does `/etc/samba/smb.conf` contain our include line?
      5. Does `/persistent/unas-custom/smb-overrides.conf` exist?
      6. Does `/persistent/unas-custom/config.yaml` exist and parse?
    - Uses configurable paths (struct with defaults) for testability
  - **REFACTOR**: Ensure output is clean and parseable

  **Must NOT do**:
  - Do NOT modify any files — status is read-only
  - Do NOT attempt repairs — just report state
  - Do NOT check Samba or NFS service status (out of scope)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Read-only checks, clear output format, slightly expanded scope
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with T10, T12)
  - **Blocks**: None
  - **Blocked By**: T6 (CLI dispatch)

  **References**:

  **Pattern References**:
  - `scripts/install-wrapper.sh:40` — `if [ -f "$EXPORTFS_ORIG" ]` — pattern for checking installation state

  **API/Type References**:
  - `internal/config/config.go` (from T2) — `Load()` for config validation check
  - `internal/installer/installer.go` (from T10) — Path constants to share (ExportfsOrigPath, SmbcontrolOrigPath, DropInPath, etc.)

  **WHY Each Reference Matters**:
  - Install script's file existence checks are the same checks status needs to perform
  - Config Load() provides validation — status can attempt to load and report any errors
  - Installer's path constants ensure status checks the same paths that install/uninstall use

  **Acceptance Criteria**:

  - [ ] `go test -v -race -run TestStatus ./cmd/unas-custom/...` → PASS (minimum 6 test cases)
  - [ ] All-installed state: shows all 6 checks as ✓, exit 0
  - [ ] Nothing-installed state: shows all 6 checks as ✗, exit 1
  - [ ] Partial install: mixed ✓/✗, exit 1
  - [ ] Output includes: NFS wrapper, SMB wrapper, systemd drop-in, SMB include, SMB overrides, Config status

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Status reports all-installed state correctly
    Tool: Bash (go test)
    Preconditions: status_test.go with mock installed state
    Steps:
      1. Run `go test -v -run TestStatus_AllInstalled ./cmd/unas-custom/`
      2. Test creates temp dirs mimicking fully installed state:
         - exportfs.orig exists, smbcontrol.orig exists
         - drop-in file exists, smb.conf has include line
         - overrides file exists, config.yaml is valid
      3. Captures stdout output
      4. Verifies all 6 status lines show installed/valid/present
    Expected Result: PASS — all checks shown as installed, exit 0
    Failure Indicators: Missing check, wrong status, exit 1
    Evidence: .sisyphus/evidence/task-11-status-installed.txt

  Scenario: Status reports nothing-installed state
    Tool: Bash (go test)
    Preconditions: status_test.go with clean state
    Steps:
      1. Run `go test -v -run TestStatus_NothingInstalled ./cmd/unas-custom/`
      2. Test uses empty temp dirs (no .orig files, no drop-in, no include, no config)
      3. Verifies all 6 status lines show not-installed
    Expected Result: PASS — all checks shown as not-installed, exit 1
    Failure Indicators: False positive, panic on missing files
    Evidence: .sisyphus/evidence/task-11-status-clean.txt

  Scenario: Status detects missing systemd drop-in as partial
    Tool: Bash (go test)
    Preconditions: status_test.go with partially installed state
    Steps:
      1. Run `go test -v -run TestStatus_PartialInstall_NoDropIn ./cmd/unas-custom/`
      2. Set up: exportfs.orig exists, smbcontrol.orig exists, but NO drop-in file
      3. Verify: NFS ✓, SMB wrapper ✓, drop-in ✗, exit 1
    Expected Result: PASS — partial state correctly identified
    Failure Indicators: Reports all OK, missing drop-in check
    Evidence: .sisyphus/evidence/task-11-status-partial-dropin.txt
  ```

  **Commit**: YES
  - Message: `feat(status): installation state reporting with NFS + SMB + drop-in checks`
  - Files: `cmd/unas-custom/status.go`, `cmd/unas-custom/status_test.go`
  - Pre-commit: `go test ./...`

- [ ] 12. CI Pipeline Update

  **What to do**:
  - Update `.github/workflows/ci.yml`:
    - Rename artifact from `exportfs-wrapper-arm64` to `unas-custom-arm64`
    - Rename tarball from `exportfs-wrapper-$TAG-arm64.tar.gz` to `unas-custom-$TAG-arm64.tar.gz`
    - Add explicit test step: `make test` (in addition to existing `go test`)
    - Add lint step: `make lint`
    - Add dependency count check: `test $(go list -m all | wc -l) -le 3` (enforce zero new deps)
    - Keep Go version at 1.21.4
    - Keep release flow via softprops/action-gh-release
  - Verify CI passes locally: `go vet ./... && gofmt -l . && go test -race ./...`

  **Must NOT do**:
  - Do NOT add goreleaser
  - Do NOT change Go version
  - Do NOT add Docker builds
  - Do NOT change the release trigger (v* tags)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Mechanical rename in YAML + small additions
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with T10, T11)
  - **Blocks**: None
  - **Blocked By**: T9 (Makefile targets referenced by CI)

  **References**:

  **Pattern References**:
  - `.github/workflows/ci.yml:62-63` — Artifact name: `name: exportfs-wrapper-arm64` → `unas-custom-arm64`
  - `.github/workflows/ci.yml:96` — Tarball: `exportfs-wrapper-$TAG-arm64.tar.gz` → `unas-custom-$TAG-arm64.tar.gz`
  - `.github/workflows/ci.yml:100-102` — Release files pattern

  **WHY Each Reference Matters**:
  - Artifact name appears in GitHub Actions UI and download URLs — must match new project name
  - Tarball name is what users download from releases — must match new project name

  **Acceptance Criteria**:

  - [ ] No references to `exportfs-wrapper` in `.github/workflows/ci.yml`
  - [ ] Artifact name: `unas-custom-arm64`
  - [ ] Tarball name: `unas-custom-$TAG-arm64.tar.gz`
  - [ ] Dependency count check present in CI
  - [ ] `go vet ./...` + `gofmt` + `go test -race` all in CI

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: CI config has no old project name references
    Tool: Bash
    Preconditions: ci.yml updated
    Steps:
      1. Run `grep "exportfs-wrapper" .github/workflows/ci.yml`
    Expected Result: No matches (exit code 1)
    Failure Indicators: Any reference to old project name
    Evidence: .sisyphus/evidence/task-12-ci-no-old-refs.txt

  Scenario: CI config is valid YAML
    Tool: Bash
    Preconditions: ci.yml updated
    Steps:
      1. Run `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml'))"`
      2. Or `go run github.com/mikefarah/yq/v4@latest eval '.jobs' .github/workflows/ci.yml`
    Expected Result: Valid YAML, no parse errors
    Failure Indicators: YAML syntax error
    Evidence: .sisyphus/evidence/task-12-ci-valid-yaml.txt
  ```

  **Commit**: YES
  - Message: `ci: update pipeline for unas-custom artifacts and test enforcement`
  - Files: `.github/workflows/ci.yml`
  - Pre-commit: `go vet ./...`

- [ ] 13. Install/Uninstall Shell Scripts + Migration Notice

  **What to do**:
  - Rewrite `scripts/install.sh` (was `install-wrapper.sh`):
    - Script name: `install.sh` (shorter, we're inside `/persistent/unas-custom/` already)
    - Check for `unas-custom` binary in current directory
    - Call `./unas-custom install` (delegate to Go binary's install command)
    - Print success message with next steps
    - Check for old installation: `if [ -d "/persistent/nfs-intercept" ]; then echo "Migration notice..."` 
  - Rewrite `scripts/uninstall.sh` (was `uninstall-wrapper.sh`):
    - Call `./unas-custom uninstall` (delegate to Go binary)
    - Print cleanup instructions for `/persistent/unas-custom/`
  - Both scripts:
    - `set -e` for safety
    - Minimal — the Go binary does the heavy lifting
    - Idempotent (safe to run multiple times)
  - Migration notice text:
    ```
    NOTE: Old installation detected at /persistent/nfs-intercept/
    The new tool uses /persistent/unas-custom/ instead.
    To migrate your NFS rules, copy them to the new config format:
      Old: /persistent/nfs-intercept/config.yaml (flat rules:)
      New: /persistent/unas-custom/config.yaml (nested nfs: rules:)
    To clean up old installation: rm -rf /persistent/nfs-intercept/
    ```

  **Must NOT do**:
  - Do NOT auto-migrate config (G3)
  - Do NOT delete old installation directory (G10)
  - Do NOT duplicate install logic — delegate to Go binary
  - Do NOT rename to `install-wrapper.sh` — use shorter `install.sh`

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Thin shell scripts delegating to Go binary
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4 (with T14)
  - **Blocks**: None
  - **Blocked By**: T10 (install/uninstall commands must exist)

  **References**:

  **Pattern References**:
  - `scripts/install-wrapper.sh:1-60` — Current install script (read for migration — most logic moves to Go)
  - `scripts/uninstall-wrapper.sh:1-22` — Current uninstall script (same — delegate to Go)

  **WHY Each Reference Matters**:
  - Current scripts show the user-facing messages and flow to preserve in the rewritten versions
  - The delegation pattern (shell → Go binary) is the key architectural change

  **Acceptance Criteria**:

  - [ ] `scripts/install.sh` exists, calls `./unas-custom install`
  - [ ] `scripts/uninstall.sh` exists, calls `./unas-custom uninstall`
  - [ ] Both use `set -e`
  - [ ] Migration notice printed if `/persistent/nfs-intercept/` exists
  - [ ] No references to `exportfs-wrapper` in any shell script
  - [ ] Old scripts `install-wrapper.sh` and `uninstall-wrapper.sh` removed

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Install script delegates to Go binary
    Tool: Bash
    Preconditions: scripts/install.sh written
    Steps:
      1. Read `scripts/install.sh`
      2. Verify it contains `./unas-custom install` command
      3. Verify it uses `set -e`
      4. Verify no references to `exportfs-wrapper`
    Expected Result: Script delegates to Go binary, clean of old references
    Failure Indicators: Hardcoded install logic, old binary name
    Evidence: .sisyphus/evidence/task-13-install-script.txt

  Scenario: No old script names remain
    Tool: Bash
    Preconditions: Scripts rewritten
    Steps:
      1. Run `ls scripts/`
      2. Verify `install.sh` and `uninstall.sh` exist
      3. Verify `install-wrapper.sh` and `uninstall-wrapper.sh` do NOT exist
    Expected Result: Only new script names present
    Failure Indicators: Old script files still present
    Evidence: .sisyphus/evidence/task-13-script-names.txt
  ```

  **Commit**: YES
  - Message: `build(scripts): unified install/uninstall shell scripts with migration notice`
  - Files: `scripts/install.sh`, `scripts/uninstall.sh`, remove `scripts/install-wrapper.sh`, remove `scripts/uninstall-wrapper.sh`
  - Pre-commit: —

- [ ] 14. Config Example + README Rewrite

  **What to do**:
  - Rewrite `config.example.yaml` with unified format:
    ```yaml
    # unas-custom configuration
    # NFS export option overrides
    nfs:
      rules:
        - path: "/var/nfs/shared/Media"
          options: "rw,sync,no_subtree_check,insecure,all_squash,anonuid=977,anongid=988"
        - path: "/var/nfs/shared/Backups"
          options: "rw,sync,no_subtree_check,insecure,no_root_squash"

    # SMB share parameter overrides
    smb:
      overrides:
        - share: "Media"
          directives:
            guest ok: "yes"
            create mask: "0664"
            directory mask: "0775"
    ```
  - Rewrite `README.md` covering:
    - Project name and description (what it does, why it exists)
    - How it works: busybox dispatch (3 modes), NFS wrapper, SMB interception mechanism (smbcontrol wrapper + systemd drop-in + smb inject), Samba include overlay for overrides
    - Architecture diagram: UDC → writes smb.conf → reload triggers → our interception → inject include → original tool
    - Installation instructions (build, deploy, install)
    - Configuration reference (unified YAML format with all options)
    - Post-firmware-update instructions (re-run `unas-custom install` — restores wrappers + drop-in)
    - Subcommands reference (install, uninstall, status, smb apply, smb inject)
    - Troubleshooting guide
    - Migration from old exportfs-wrapper installation
    - Development instructions (build, test, contribute)
  - Remove any references to eBPF or old project name

  **Must NOT do**:
  - Do NOT add emojis to README
  - Do NOT reference the parent directory name (ebpf-file-intercept)
  - Do NOT over-document — keep it practical
  - Do NOT add badges beyond what's useful (build status is fine)

  **Recommended Agent Profile**:
  - **Category**: `writing`
    - Reason: Documentation-heavy task requiring clear technical writing
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4 (with T13)
  - **Blocks**: None
  - **Blocked By**: All previous tasks (docs must reflect final implementation)

  **References**:

  **Pattern References**:
  - `README.md:1-120` — Current README (read for structure and tone to preserve/improve)
  - `config.example.yaml:1-5` — Current config format (replace with unified format)

  **External References**:
  - User's spec: SMB config example with `share:` + `directives:` structure
  - User's spec: UNAS filesystem architecture, storage locations, service orchestration

  **WHY Each Reference Matters**:
  - Current README shows the documentation style and coverage level — the new version should be at least as comprehensive
  - Config example must exactly match the format that `config.Load()` (T2) expects

  **Acceptance Criteria**:

  - [ ] `config.example.yaml` uses unified format with `nfs:` and `smb:` sections
  - [ ] `README.md` covers: overview, architecture (3-mode dispatch + SMB interception), installation, config, subcommands (including smb inject), troubleshooting, migration, development
  - [ ] README explains WHY smbcontrol wrapper + systemd drop-in are needed (UDC rewrites smb.conf)
  - [ ] No references to `exportfs-wrapper`, eBPF, or old paths in README
  - [ ] Config example is valid YAML that loads successfully
  - [ ] README subcommand reference matches actual CLI output

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Config example is valid and loadable
    Tool: Bash
    Preconditions: config.example.yaml rewritten
    Steps:
      1. Run `go run ./cmd/unas-custom/ --test-config config.example.yaml` (or write a small test)
      2. Or: `go test -v -run TestLoadExampleConfig ./internal/config/` with test that loads config.example.yaml
      3. Verify config loads without error
      4. Verify NFS rules and SMB overrides are populated
    Expected Result: Config loads successfully with expected entries
    Failure Indicators: Parse error, empty sections, wrong field names
    Evidence: .sisyphus/evidence/task-14-config-example.txt

  Scenario: README has no old project references
    Tool: Bash
    Preconditions: README.md rewritten
    Steps:
      1. Run `grep -i "exportfs-wrapper\|nfs-intercept\|ebpf" README.md`
    Expected Result: No matches (exit code 1) — except in "Migration" section which references old path
    Failure Indicators: References to old project name outside migration context
    Evidence: .sisyphus/evidence/task-14-readme-no-old-refs.txt
  ```

  **Commit**: YES
  - Message: `docs: comprehensive README rewrite and unified config example`
  - Files: `config.example.yaml`, `README.md`
  - Pre-commit: —

---

## Final Verification Wave (MANDATORY — after ALL implementation tasks)

> 4 review agents run in PARALLEL. ALL must APPROVE. Rejection → fix → re-run.

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in .sisyphus/evidence/. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run `go vet ./...` + `gofmt -l .` + `go test -race -cover ./...`. Review all changed files for: `as any`/`@ts-ignore` equivalents, empty catches, fmt.Print in prod code, commented-out code, unused imports. Check AI slop: excessive comments, over-abstraction, generic names. Verify zero new dependencies: `go list -m all | wc -l` should be 3 (module + yaml.v3 + check.v1).
  Output: `Build [PASS/FAIL] | Vet [PASS/FAIL] | Tests [N pass/N fail] | Coverage [N%] | Deps [N] | VERDICT`

- [ ] F3. **Real QA via SSH to UNAS** — `unspecified-high` (+ `mcp-ssh`)
  SSH to UNAS. Deploy the new binary. Run `unas-custom install`. Verify: exportfs is replaced, smbcontrol is replaced, systemd drop-in exists (`systemctl show smbd.service -p ExecReload` shows our inject command), smb.conf has include line, config.yaml exists. Run `unas-custom status` (all ✓). Run `unas-custom smb apply`. Run `unas-custom smb inject` (verify idempotent). Trigger NFS reload via `exportfs -ra`. Verify NFS exports are modified. Verify SMB interception: manually write smb.conf without include line, then trigger smbcontrol reload-config — verify include line re-injected. Run `unas-custom uninstall`. Verify: exportfs restored, smbcontrol restored, drop-in removed, include line removed. Save evidence.
  Output: `Install [PASS/FAIL] | NFS [PASS/FAIL] | SMB [PASS/FAIL] | Uninstall [PASS/FAIL] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff (git log/diff). Verify 1:1 — everything in spec was built (no missing), nothing beyond spec was built (no creep). Check "Must NOT do" compliance. Detect cross-task contamination: Task N touching Task M's files. Flag unaccounted changes.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

| Commit | Scope | Message | Pre-commit |
|--------|-------|---------|------------|
| 1 | T1 | `refactor: rename module to github.com/bbettridge/unas-custom` | `go build ./...` |
| 2 | T2 | `feat(config): unified YAML config format with NFS + SMB sections` | `go test ./internal/config/...` |
| 3 | T3 | `test(nfs): comprehensive test suite proving parser behavior preservation` | `go test ./internal/nfs/...` |
| 4 | T4 | `feat(smb): SMB override generator producing valid Samba INI` | `go test ./internal/smb/...` |
| 5 | T5 | `feat(logging): file-based logger for persistent diagnostics` | `go test ./internal/logging/...` |
| 6 | T6 | `feat(cli): busybox-style dispatch + hand-rolled subcommand routing (3 modes)` | `go test ./cmd/unas-custom/...` |
| 7 | T8.5 | `feat(smb): idempotent smb.conf include line injection with atomic write` | `go test -race ./internal/smb/...` |
| 8 | T7 | `refactor(wrapper): NFS + smbcontrol fail-open wrapper modes with logging` | `go test -race ./...` |
| 9 | T8 | `feat(smb): smb apply command generating overrides from config` | `go test ./...` |
| 10 | T9 | `build: update Makefile for unas-custom binary and symlink support` | `make build-arm64` |
| 11 | T10 | `feat(install): unified install/uninstall with NFS + smbcontrol wrappers + systemd drop-in` | `go test ./internal/installer/...` |
| 12 | T11 | `feat(status): installation state reporting with NFS + SMB + drop-in checks` | `go test ./...` |
| 13 | T12 | `ci: update pipeline for unas-custom artifacts and test enforcement` | `go vet ./...` |
| 14 | T13 | `build(scripts): unified install/uninstall shell scripts with migration notice` | — |
| 15 | T14 | `docs: comprehensive README rewrite and unified config example` | — |

---

## Success Criteria

### Verification Commands
```bash
# Build
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o unas-custom ./cmd/unas-custom
file unas-custom                          # Expected: ELF 64-bit LSB executable, ARM aarch64, statically linked
ls -la unas-custom                        # Expected: <3MB

# Tests
go test -v -race -cover ./...             # Expected: PASS, >80% coverage on new code
go vet ./...                              # Expected: no output
test -z "$(gofmt -l .)"                   # Expected: exit 0

# Dependencies
go list -m all | wc -l                    # Expected: 3 (module, yaml.v3, check.v1)

# Dispatch — 3 modes (on ARM64 or via cross-test)
ln -sf ./unas-custom ./exportfs
ln -sf ./unas-custom ./smbcontrol
./exportfs --help 2>&1 || true            # Expected: runs NFS wrapper mode (not CLI help)
./smbcontrol --help 2>&1 || true          # Expected: runs SMB wrapper mode (not CLI help)
./unas-custom --help                      # Expected: shows CLI help with subcommands

# SMB inject (on UNAS via SSH)
unas-custom smb inject                    # Expected: adds include line to smb.conf (or no-op if present)
unas-custom smb inject                    # Expected: no-op (idempotent)
grep "include = /persistent/unas-custom/smb-overrides.conf" /etc/samba/smb.conf  # Expected: match

# Systemd drop-in (on UNAS via SSH)
systemctl show smbd.service -p ExecReload # Expected: shows our inject command
cat /etc/systemd/system/smbd.service.d/unas-custom.conf  # Expected: correct drop-in content
```

### Final Checklist
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" (G1-G19) absent
- [ ] All tests pass with race detector
- [ ] Binary <3MB, static, ARM64
- [ ] Zero new dependencies
- [ ] NFS behavior identical to current code (test-proven)
- [ ] SMB overrides generate valid Samba INI
- [ ] smbcontrol wrapper injects include before exec (fail-open)
- [ ] systemd drop-in intercepts smbd reload AND restart
- [ ] `smb inject` is idempotent with atomic write
- [ ] Install/uninstall idempotent (NFS + smbcontrol + drop-in)
- [ ] README accurate and complete (documents SMB interception mechanism)
