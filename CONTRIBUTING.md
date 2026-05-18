# Contributing

## Development

```bash
git clone https://github.com/bath0ry/mlx-go-sdk.git
cd mlx-go-sdk
```

## Running Tests

```bash
go test ./...
```

Live E2E tests require a Multilogin X account and are opt-in:

```bash
MLX_RUN_E2E=1 go test ./e2e/...
```

## Code Style

This project follows [official Go style guidance](https://go.dev/doc/effective_go). See `docs/go-style.md` for project-specific conventions.

Run formatting before committing:

```bash
gofmt -w .
```

## Linting

```bash
golangci-lint run
```

## Pull Requests

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Ensure tests pass: `go test ./...`
5. Submit a pull request
