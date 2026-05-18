# GitHub Release Preparation Design

Date: 2026-05-18

## Goal

Prepare the `mlx-go-sdk` project for publishing to a private GitHub repository with clean structure, working CI, and professional presentation.

## Approach

Thematic atomic commits (Approach B) — each commit addresses one concern, producing a clean git history.

## Baseline Verification

- `go test ./...` — all pass
- `go vet ./...` — clean
- `gofmt -l .` — clean
- CLI already decomposed from 3106 to 152 lines in root.go

## Commit Plan

### Commit 1: `chore: update .gitignore and remove dev artifacts`

**Add to `.gitignore`:**
```
.windsurf/
skills-lock.json
.firecrawl/
AGENTS.md
```

**Remove:**
- `.firecrawl/` — 4 MCP leftover files (already untracked, add to gitignore)
- `skills-lock.json` — superpowers lockfile (untracked, add to gitignore)
- `.windsurf/` — superpowers skills (untracked, add to gitignore)
- `Multilogin X API.postman_collection.json` — dev tool, not SDK code
- `AGENTS.md` — local agent instructions, not for public repo

**Commit staged deletions:**
- `.planning/REQUIREMENTS.md`, `ROADMAP.md`, `STATE.md`, `config.json`

### Commit 2: `chore: remove CLAUDE.md`

- Delete `CLAUDE.md` — duplicates AGENTS.md content, contains placeholder sections

### Commit 3: `chore: fix go.mod module path for GitHub`

- `go.mod`: `module mlx-go-sdk` → `module github.com/bath0ry/mlx-go-sdk`
- Update all internal imports:
  - `cmd/mlx/main.go`: `mlx-go-sdk/internal/cli` → `github.com/bath0ry/mlx-go-sdk/internal/cli`
  - `internal/cli/*.go`: update any `mlx-go-sdk/...` imports
  - Test files: update any `mlx-go-sdk/...` imports
- Run `go mod tidy` to refresh go.sum

### Commit 4: `docs: add LICENSE, polish README, add CONTRIBUTING.md`

- Add `LICENSE` — MIT license, copyright Marvin
- Update `README.md`:
  - Add badges: Go version, tests status, license
  - Fix install command: `go get github.com/bath0ry/mlx-go-sdk`
- Add `CONTRIBUTING.md`:
  - How to run tests: `go test ./...`
  - Style guide reference: `docs/go-style.md`
  - PR process
- Audit `docs/` for broken links and stale content

### Commit 5: `ci: add golangci-lint config and GitHub Actions`

- Add `.golangci.yml`:
  - Linters: `gofmt`, `govet`, `staticcheck`, `errcheck`, `gosec`, `revive`
  - Exclude test files from some checks
- Add `.github/workflows/ci.yml`:
  - Triggers: push to master, pull_request
  - Matrix: Go 1.26
  - Steps: checkout → setup-go → cache → `go test ./...` → `go vet ./...` → `golangci-lint run`

## Decisions

1. **MIT License** — standard for Go libraries, permissive, widely understood
2. **AGENTS.md excluded** — local dev tooling, not relevant for SDK consumers
3. **Single CI workflow** — tests + lint together, simpler than separate workflows
4. **Module path `github.com/bath0ry/mlx-go-sdk`** — matches GitHub repo structure
5. **go.sum deps left as-is** — all `// indirect` is correct for a library where direct imports are only in tests

## Non-Goals

- Changing SDK public API
- Adding new features
- Refactoring code structure (already done in prior cleanup)
- Setting up release automation (can be done later)
