# Testing layout

This repository keeps test execution simple from the root module while separating test intent by scope.

## Goals

- keep default validation easy: `go test ./...`
- make it obvious which tests are pure unit checks vs HTTP-backed integration checks vs live E2E validation
- keep consumer examples discoverable through Go example tests
- avoid broad workspace-level test runs that accidentally pick up nested example projects under `GithubExamples/`

## Scopes

### Unit tests

Unit tests should stay close to the package under test and focus on:

- pure data/model behavior
- request/response normalization
- retry and polling helpers
- archive/path utilities
- validation logic
- other behavior that does not require a live server

Typical characteristics:

- fast
- deterministic
- no real network access
- safe for routine `go test ./...`

For naming, prefer clear scope-oriented names when files are touched or renamed, for example:

- `*_unit_test.go`
- `examples_*_test.go` for example-style tests

### Integration tests

Integration tests should exercise the exported SDK surface against controlled HTTP fixtures, typically with `httptest` servers and shared helpers in `internal/testutil`.

These tests are the right place for:

- service method request/response coverage
- workflow orchestration against mocked MLX API/launcher behavior
- resource, cookie, proxy, transfer, and launcher contract validation
- regression tests for tricky API edge cases

Integration tests should still be safe for normal `go test ./...` runs.

This repository now also includes a dedicated `integration/` package for HTTP-backed workflow and batch coverage. That package is intended to verify exported SDK behavior through external-package tests rather than package-internal access.

Practical guidance:

- keep unit-oriented tests in the root package when they need package internals
- prefer `integration/` for mocked API and launcher flows that can use only exported APIs
- keep shared request/response test helpers in `integration/helpers_test.go` or `internal/testutil` when they are broadly reusable

This split makes the repository easier to navigate without changing the normal root-module test command.

### Example tests

Example tests document consumer usage and should stay easy to find next to the public package.

Use example tests for:

- client bootstrap from environment
- archive/path helper examples
- proxy conversion examples
- other small consumer-facing snippets

Prefer `package mlx_test` for examples so they read like real external consumer code.

### E2E tests

Live end-to-end validation is opt-in.

These tests verify behavior against real Multilogin X services and, where needed, a real launcher. They should never be part of the default fast feedback loop.

Current rules:

- guard live tests behind environment checks such as `MLX_RUN_E2E=1`
- keep E2E invocation explicit
- document required environment variables
- call out live verification gaps and platform quirks in test logs

The repository also includes a dedicated `e2e/` package entrypoint for tag-based live runs.

Recommended command:

```text
MLX_RUN_E2E=1 go test -tags=e2e ./e2e
```

## Execution policy

### Default local validation

Run from the root module:

```text
go test ./...
```

This is the standard repository-wide validation command.

### Targeted package validation

When working in a narrow area, prefer smaller runs first, for example:

```text
go test .
go test ./internal/cli
go test ./integration
```

### Live validation

Run live checks only when you intentionally want E2E coverage and the required environment is available:

```text
MLX_RUN_E2E=1 go test -tags=e2e ./e2e
```

For destructive or rate-limit spike tests, require an additional explicit guard.

Example:

```text
MLX_RUN_E2E=1 MLX_RUN_CREATION_LIMIT_SPIKE=1 MLX_E2E_PROFILE_CAP=50 go test -tags=e2e ./e2e -run TestE2EProfileCreationLimits -count=1 -v
```

Another example for sustained batched creation:

```text
MLX_RUN_E2E=1 MLX_RUN_CREATE_50_SPIKE=1 MLX_E2E_PROFILE_CAP=50 go test -tags=e2e ./e2e -run TestE2ECreateFiftyProfilesCadence -count=1 -v
```

## Repository conventions

### 1. Keep root-module execution simple

The repository should remain friendly to:

```text
go test ./...
```

Do not require custom scripts or multi-step setup for ordinary unit and integration coverage.

### 2. Keep live checks explicit

E2E coverage is valuable, but it must stay clearly separated from routine test runs.

Use:

- build tags where appropriate
- runtime environment gates
- dedicated documentation for required environment variables and launcher expectations

### 3. Keep examples polished

Public examples are part of the SDK experience. Keep them readable, minimal, and consumer-oriented.

### 4. Prefer shared test helpers for server-backed flows

If multiple HTTP-backed tests need the same support code, add helpers under `internal/testutil` or near the relevant scope instead of duplicating fixture logic across many files.

### 5. Avoid accidental cross-project execution

This repository contains nested fixture/example projects under `GithubExamples/`. Validation should be run from the root module so SDK tests are exercised without pulling unrelated nested test suites into the workflow.

## Recommended end state

The intended long-term layout is:

- root package: unit tests and examples
- `integration/`: mocked-server and workflow contract tests
- `e2e/`: live opt-in validation

That structure keeps the repository easy to navigate while preserving the simplest possible default developer workflow.

## When adding new tests

Use this quick checklist:

- Is the test pure logic with no server dependency?
  - put it with unit tests near the package
- Does it need `httptest` or mocked MLX responses?
  - treat it as integration coverage
- Does it talk to real MLX services or a real launcher?
  - make it E2E and opt-in
- Is it primarily documentation for consumers?
  - make it an example test

Keeping that distinction clear makes the SDK easier to maintain and easier for new contributors to understand.
