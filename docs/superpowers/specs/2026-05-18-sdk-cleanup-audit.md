# SDK Cleanup Audit

Date: 2026-05-18

## Verified State

- `go test ./...` passes.
- `go vet ./...` passes.
- `gofmt -l .` reports only `.tmp/` probe files that are not part of tracked
  source.

## Improvements Completed

- Added a repository-facing Go style guide in
  [go-style.md](C:/Users/bath0ry/mlx/mlx-go-sdk/docs/go-style.md).
- Added [.editorconfig](C:/Users/bath0ry/mlx/mlx-go-sdk/.editorconfig) for
  baseline whitespace and line-ending consistency.
- Reduced CLI monolith pressure by splitting:
  - help text into `internal/cli/help.go`
  - render/output helpers into `internal/cli/output.go`
  - shared CLI helpers into `internal/cli/helpers.go`
  - config/folder/template commands into
    `internal/cli/commands_config_folder_template.go`
  - launcher/profile/export/import commands into
    `internal/cli/commands_runtime_profiles_transfer.go`
- extension commands into `internal/cli/commands_extension.go`
- cookies/proxy commands into `internal/cli/commands_cookies_proxy.go`
- Reduced
  [root.go](C:/Users/bath0ry/mlx/mlx-go-sdk/internal/cli/root.go)
  from ~3106 lines to ~152 lines.

## Remaining Architectural Findings

1. Several tracked files changed only because `gofmt` normalized formatting.
   This is low risk, but it does increase diff volume for the cleanup branch.

2. The test suite still contains long inline JSON fixtures. They are valid and
   readable enough for now, but a later pass could extract a small set of shared
   fixture helpers to reduce repetition.

3. There is still no lint configuration such as `golangci-lint`. That is not a
   correctness bug, but it means style enforcement still depends mostly on
   `gofmt`, tests, and review discipline.

## Explicit Non-Issues

- A central `Tests/` directory was not introduced because it would work against
  idiomatic Go test locality.
- `.tmp/` probe files are already outside the tracked quality baseline and
  remain excluded from the Git workflow.
