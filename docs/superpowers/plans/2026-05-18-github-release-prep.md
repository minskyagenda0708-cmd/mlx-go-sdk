# GitHub Release Preparation — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Clean up the `mlx-go-sdk` project and prepare it for publishing to a private GitHub repository with proper module path, CI, docs, and no dev-artifact clutter.

**Architecture:** Five atomic commits: (1) gitignore + garbage removal, (2) delete CLAUDE.md, (3) fix go.mod module path to `github.com/bath0ry/mlx-go-sdk` with all import updates, (4) add LICENSE/README polish/CONTRIBUTING.md, (5) add golangci-lint config and GitHub Actions CI workflow.

**Tech Stack:** Go 1.26, golangci-lint, GitHub Actions

---

### Task 1: Git hygiene & garbage removal

**Files:**
- Modify: `.gitignore`
- Delete: `Multilogin X API.postman_collection.json`
- Delete: `AGENTS.md`
- Delete: `.planning/REQUIREMENTS.md` (staged)
- Delete: `.planning/ROADMAP.md` (staged)
- Delete: `.planning/STATE.md` (staged)
- Delete: `.planning/config.json` (staged)

- [ ] **Step 1: Update `.gitignore`**

Read current `.gitignore`, then replace entire content:

```
# Beads / Dolt files (added by bd init)
.dolt/
*.db
.beads-credential-key
.tmp/

# Local agent/editor state
.claude/

# Superpowers dev tooling
.windsurf/
skills-lock.json

# Firecrawl MCP leftovers
.firecrawl/

# Local agent instructions (not for public repo)
AGENTS.md

# Large external reference repos kept outside local history
GithubExamples/
```

- [ ] **Step 2: Delete garbage files**

```powershell
Remove-Item -Force "c:\Users\bath0ry\mlx\mlx-go-sdk\Multilogin X API.postman_collection.json"
Remove-Item -Force "c:\Users\bath0ry\mlx\mlx-go-sdk\AGENTS.md"
```

- [ ] **Step 3: Commit staged deletions + new changes**

```bash
git add .gitignore
git rm "Multilogin X API.postman_collection.json"
git rm AGENTS.md
git commit -m "chore: update .gitignore and remove dev artifacts"
```

The `.planning/` files are already staged as deleted — they will be included in this commit automatically.

- [ ] **Step 4: Verify clean state**

```bash
git status
```

Expected: working tree clean, nothing staged.

---

### Task 2: Remove CLAUDE.md

**Files:**
- Delete: `CLAUDE.md`

- [ ] **Step 1: Delete CLAUDE.md**

```bash
git rm CLAUDE.md
```

- [ ] **Step 2: Commit**

```bash
git commit -m "chore: remove CLAUDE.md"
```

---

### Task 3: Fix go.mod module path for GitHub

**Files:**
- Modify: `go.mod:1`
- Modify: `cmd/mlx/main.go:3`
- Modify: `internal/cli/config.go:14`
- Modify: `internal/cli/root.go:11`
- Modify: `internal/cli/helpers.go:13`
- Modify: `internal/cli/commands_config_folder_template.go:14`
- Modify: `internal/cli/commands_cookies_proxy.go:12`
- Modify: `internal/cli/commands_extension.go:12`
- Modify: `internal/cli/commands_runtime_profiles_transfer.go:12`
- Modify: `internal/cli/commands_tag.go:12`
- Modify: `internal/cli/config_test.go:10`
- Modify: `internal/cli/runtime_test.go:15`
- Modify: `internal/cli/template_test.go:9`
- Modify: `internal/cli/template_commands_test.go:16`
- Modify: `integration/batch_test.go:13-14`
- Modify: `integration/workflows_test.go:17-18`
- Modify: `examples_client_test.go:9`
- Modify: `examples_proxy_test.go:6`
- Modify: `e2e/e2e_test.go:17`
- Modify: `e2e/profile_creation_limits_test.go:17`
- Modify: `e2e/profile_creation_limits_helpers_test.go` (check for import)
- Modify: Root test files that import `"mlx-go-sdk/internal/testutil"` (10 files)

- [ ] **Step 1: Change module path in go.mod**

Edit `go.mod` line 1:
```
module github.com/bath0ry/mlx-go-sdk
```

- [ ] **Step 2: Update import in cmd/mlx/main.go**

Edit `cmd/mlx/main.go` line 3:
```go
import "github.com/bath0ry/mlx-go-sdk/internal/cli"
```

- [ ] **Step 3: Update all `mlx "mlx-go-sdk"` imports in internal/cli/**

Replace in all 10 files under `internal/cli/`:
```
mlx "mlx-go-sdk"
```
→
```
mlx "github.com/bath0ry/mlx-go-sdk"
```

Files: `config.go`, `root.go`, `helpers.go`, `commands_config_folder_template.go`, `commands_cookies_proxy.go`, `commands_extension.go`, `commands_runtime_profiles_transfer.go`, `commands_tag.go`, `config_test.go`, `runtime_test.go`, `template_test.go`, `template_commands_test.go`

- [ ] **Step 4: Update imports in integration/**

In `integration/batch_test.go`:
```
mlx "mlx-go-sdk"
"mlx-go-sdk/internal/testutil"
```
→
```
mlx "github.com/bath0ry/mlx-go-sdk"
"github.com/bath0ry/mlx-go-sdk/internal/testutil"
```

In `integration/workflows_test.go`:
```
mlx "mlx-go-sdk"
"mlx-go-sdk/internal/testutil"
```
→
```
mlx "github.com/bath0ry/mlx-go-sdk"
"github.com/bath0ry/mlx-go-sdk/internal/testutil"
```

- [ ] **Step 5: Update imports in examples/**

In `examples_client_test.go`:
```
mlx "mlx-go-sdk"
```
→
```
mlx "github.com/bath0ry/mlx-go-sdk"
```

In `examples_proxy_test.go`:
```
mlx "mlx-go-sdk"
```
→
```
mlx "github.com/bath0ry/mlx-go-sdk"
```

- [ ] **Step 6: Update imports in e2e/**

In `e2e/e2e_test.go`:
```
. "mlx-go-sdk"
```
→
```
. "github.com/bath0ry/mlx-go-sdk"
```

In `e2e/profile_creation_limits_test.go`:
```
. "mlx-go-sdk"
```
→
```
. "github.com/bath0ry/mlx-go-sdk"
```

- [ ] **Step 7: Update imports in root package test files**

Replace `"mlx-go-sdk/internal/testutil"` → `"github.com/bath0ry/mlx-go-sdk/internal/testutil"` in all 10 files:
- `archive_manager_test.go:11`
- `cookies_test.go:9`
- `folders_test.go:9`
- `launcher_test.go:13`
- `launcher_quick_test.go:10`
- `profiles_test.go:12`
- `proxy_test.go:11`
- `resources_test.go:11`
- `tags_test.go:9`
- `transfers_test.go:11`

- [ ] **Step 8: Run go mod tidy**

```bash
go mod tidy
```

- [ ] **Step 9: Verify tests pass**

```bash
go test ./...
```

Expected: all packages pass.

- [ ] **Step 10: Verify go vet**

```bash
go vet ./...
```

Expected: no output (clean).

- [ ] **Step 11: Commit**

```bash
git add -A
git commit -m "chore: fix go.mod module path for GitHub"
```

---

### Task 4: Docs, LICENSE, README, CONTRIBUTING.md

**Files:**
- Create: `LICENSE`
- Modify: `README.md`
- Create: `CONTRIBUTING.md`

- [ ] **Step 1: Create LICENSE (MIT)**

```text
MIT License

Copyright (c) 2026 Marvin

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

- [ ] **Step 2: Update README.md — add badges and fix install**

Add after the title line (`# mlx-go-sdk`):

```markdown
[![Go Version](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Tests](https://github.com/bath0ry/mlx-go-sdk/actions/workflows/ci.yml/badge.svg)](https://github.com/bath0ry/mlx-go-sdk/actions/workflows/ci.yml)
```

Change install command (line 8):
```
go get mlx-go-sdk
```
→
```
go get github.com/bath0ry/mlx-go-sdk
```

- [ ] **Step 3: Create CONTRIBUTING.md**

```markdown
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
```

- [ ] **Step 4: Commit**

```bash
git add LICENSE README.md CONTRIBUTING.md
git commit -m "docs: add LICENSE, polish README, add CONTRIBUTING.md"
```

---

### Task 5: CI & linting

**Files:**
- Create: `.golangci.yml`
- Create: `.github/workflows/ci.yml`

- [ ] **Step 1: Create .golangci.yml**

```yaml
linters:
  enable:
    - gofmt
    - govet
    - staticcheck
    - errcheck
    - gosec
    - revive
    - unused
    - ineffassign

linters-settings:
  revive:
    rules:
      - name: exported
        severity: warning
        disabled: false
  gosec:
    excludes:
      - G104

issues:
  exclude-use-default: false
  max-issues-per-linter: 0
  max-same-issues: 0

run:
  timeout: 5m
  tests: true
```

- [ ] **Step 2: Create .github/workflows/ci.yml**

```yaml
name: CI

on:
  push:
    branches: [master]
  pull_request:
    branches: [master]

jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go-version: ["1.26"]

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ matrix.go-version }}

      - name: Cache Go modules
        uses: actions/cache@v4
        with:
          path: |
            ~/.cache/go-build
            ~/go/pkg/mod
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
          restore-keys: |
            ${{ runner.os }}-go-

      - name: Download dependencies
        run: go mod download

      - name: Run tests
        run: go test ./...

      - name: Run go vet
        run: go vet ./...

  lint:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.26"

      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: latest
```

- [ ] **Step 3: Create .github directory structure**

```powershell
New-Item -ItemType Directory -Force -Path "c:\Users\bath0ry\mlx\mlx-go-sdk\.github\workflows"
```

- [ ] **Step 4: Commit**

```bash
git add .golangci.yml .github/
git commit -m "ci: add golangci-lint config and GitHub Actions"
```

---

### Final Verification

- [ ] **Step 1: Run full test suite**

```bash
go test ./...
```

Expected: all pass.

- [ ] **Step 2: Check git log**

```bash
git log --oneline -6
```

Expected: 5 new commits on top of current HEAD.

- [ ] **Step 3: Final git status**

```bash
git status
```

Expected: clean working tree, nothing to commit.
