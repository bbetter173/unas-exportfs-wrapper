# Issues — unas-custom

## [2026-03-07] Session Start — No issues yet

## [2026-03-07] Tooling

- `lsp_diagnostics` initially failed because `gopls` was not installed in PATH.
- Installed `gopls` to `~/.local/bin`; diagnostics now run, but report a workspace-scoping warning (`No active builds contain ...`) rather than code errors.

## [2026-03-07] F1 Plan Compliance Audit

- Definition of Done coverage threshold appears unmet: `go test -race -cover ./...` reports <80% in `cmd/unas-custom` (62.8%) and `internal/installer` (60.0%).
- Wrapper missing-original paths return exit code 1 without a clear stderr message in `cmd/unas-custom/wrapper.go` (`runOriginalCommand`). Plan task T7 acceptance expects a clear error message for missing `.orig` binaries.
