# mlx-go-sdk

Go SDK for Multilogin X with typed services for profiles, launcher control, cookies, resources, proxy generation, archive handling, retries, and verified high-level workflows.

## Install

```bash
go get mlx-go-sdk
```

## Quick start

Set environment variables in the consumer project:

- `MLX_TOKEN`
- `MLX_BASE_URL` (optional)
- `MLX_LAUNCHER_URL` (optional)
- `MLX_COOKIES_URL` (optional)
- `MLX_PROXY_URL` (optional)

Create a production-style client:

```go
client, err := mlx.NewFromEnv(
    mlx.WithTimeout(30*time.Second),
    mlx.WithRetry(mlx.RetryOptions{
        MaxAttempts:     4,
        InitialInterval: 500 * time.Millisecond,
        MaxInterval:     2 * time.Second,
        Multiplier:      2,
    }),
    mlx.WithUserAgent("acme-mlx-cli/1.0"),
)
```

## Reference CLI

This repository now includes a reference CLI scaffold at `cmd/mlx`.

Build or run it with:

```bash
go build ./cmd/mlx
go run ./cmd/mlx --help
```

Current command groups are:

- `config`
- `folder`
- `template`
- `profile`
- `launcher`
- `export`
- `import`
- `extension`
- `cookies`
- `proxy`

CLI configuration rules:

- authentication is **environment-only** via `MLX_TOKEN`
- endpoint overrides remain compatible with the SDK environment variables:
  - `MLX_BASE_URL`
  - `MLX_LAUNCHER_URL`
  - `MLX_COOKIES_URL`
  - `MLX_PROXY_URL`
- additional CLI-oriented environment overrides are supported for convenience:
  - `MLX_CONFIG_FILE`
  - `MLX_OUTPUT`
  - `MLX_TIMEOUT`
  - `MLX_USER_AGENT`
- effective settings follow: flags → environment → config file → built-in defaults
- supported output formats are `table`, `json`, and `yaml`

The CLI is intentionally scoped to config/folder/template/profile/launcher/export/import/extension/cookies/proxy workflows only. It does **not** add interactive auth flows or mobile profile commands.

## Consumer-oriented guides

- `docs/cli-reference.md` — reference CLI command groups, examples, and SDK mapping
- `docs/cli-config.md` — CLI config schema, precedence rules, and defaults
- `docs/verified-workflows.md` — verified create/find/start/stop/import/export/extension flows
- `docs/batch-helpers.md` — multi-profile workflow helpers with aggregated errors
- `docs/rod-example.md` — Rod attachment flow using SDK automation helpers
- `docs/extensions.md` — extension upload and attach workflows
- `docs/proxy-workflows.md` — managed MLX proxy generation and patching
- `docs/retries.md` — retry and error classification behavior
- `docs/consumer-guide.md` — production usage patterns, examples, and `cmd/` layout suggestions

## Core areas

- `client.Profiles` — create, search, patch, move, clone, meta reads
- `client.Launcher` — start, stop, status, version, health
- `client.Transfers` — import/export job control
- `client.Archives` — export-to-folder file organization
- `client.Cookies` — metadata, list, import/export, cookie seeding
- `client.Resources` — templates, extensions, object storage flows
- `client.Proxies` — MLX proxy generation and parsing
- `client.Workflows` — higher-level verified flows

## Test layout

The repository keeps the default test flow simple:

```bash
go test ./...
```

Practical test scope guidance:

- root package tests cover the SDK's fast package-level validation, including unit-style and mocked API/workflow coverage
- example/documentation tests live alongside the main package so examples stay close to exported APIs
- live validation is opt-in and guarded by `MLX_RUN_E2E=1` so ordinary test runs do not hit real Multilogin X services accidentally
- the repository is moving toward a clearer split between fast default tests and explicitly-invoked live E2E coverage

When running live checks, keep launcher/service requirements explicit and prefer targeted commands instead of broad workspace-wide test sweeps.

## Notes

- Treat `ProfileMeta.IsLocal` as diagnostic only; prefer `parameters.storage.is_local` and verified workflow signals.
- Prefer SOCKS5 for MLX managed proxies in real automation flows.
- For extension attachment, object-centric verification is stronger than profile-centric usage reads in some live environments.
