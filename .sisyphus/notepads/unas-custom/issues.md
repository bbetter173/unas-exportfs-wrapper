# Issues — unas-custom

## [2026-03-07] Session Start — No issues yet

## [2026-03-07] Tooling

- `lsp_diagnostics` initially failed because `gopls` was not installed in PATH.
- Installed `gopls` to `~/.local/bin`; diagnostics now run, but report a workspace-scoping warning (`No active builds contain ...`) rather than code errors.
