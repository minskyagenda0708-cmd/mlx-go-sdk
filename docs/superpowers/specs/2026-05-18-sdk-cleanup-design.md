# SDK Cleanup Design

Date: 2026-05-18

## Understanding Summary

- The repository already has working SDK and automation flows, including the
  Rod-via-Playwright/CDP fix.
- The next milestone is a cleanup pass focused on readability,
  maintainability, and low-risk structure improvements.
- The cleanup baseline should follow official Go guidance rather than mimic
  Python's PEP 8 literally.
- Internal and local identifiers should use idiomatic Go `mixedCaps`, for
  example `startProfile`.
- Readability should favor lines around 80-90 characters when practical, but
  should not fight `gofmt` or introduce awkward wrapping.
- The main architectural hotspot is `internal/cli/root.go`, which currently
  concentrates command parsing, help text, rendering, file helpers, polling,
  and all command handlers in one file.

## Official Guidance

- `gofmt`: <https://go.dev/blog/gofmt>
- `Effective Go`: <https://go.dev/doc/effective_go>
- `Code Review Comments`: <https://go.dev/wiki/CodeReviewComments>
- `Go Test Comments`: <https://go.dev/wiki/TestComments>

## Assumptions

- Public SDK API names must remain stable and idiomatic for Go consumers.
- Cleanup should prefer behavior-preserving refactors over broad redesign.
- Tests stay co-located with code in `*_test.go`; a central `Tests/` directory
  is not adopted because it is non-idiomatic for Go and weakens locality.
- Existing ignored local-state directories such as `.beads/`, `.claude/`,
  `.firecrawl/`, `.git/`, and `.tmp/` remain excluded from Git workflows.
- Untracked probe files under `.tmp/` are development artifacts and should not
  drive repository structure.

## Decisions

1. Use official Go style guidance as the project baseline.
   Alternatives considered: rigid 80-column rule, custom style unrelated to Go.
   Chosen because it matches the language ecosystem and reduces churn.

2. Treat 80-90 columns as a readability target, not a hard formatting rule.
   Alternatives considered: enforce 80 everywhere or allow arbitrary line
   length. Chosen because Go relies on `gofmt`, but some manual wrapping still
   improves readability.

3. Preserve co-located tests.
   Alternatives considered: move tests into a central `Tests/` directory.
   Chosen because Go tooling and maintenance work best when tests live near the
   code they verify.

4. Split `internal/cli/root.go` into thematic files without changing CLI
   behavior.
   Alternatives considered: leave the monolith intact or perform a larger CLI
   redesign. Chosen because it yields a large readability win at low risk.

## Approved Execution Plan

1. Normalize tracked Go files with `gofmt` and small readability fixes.
2. Remove low-value duplication in CLI and tests where extraction is obvious.
3. Decompose `internal/cli/root.go` into thematic files:
   - command dispatch and top-level help
   - config/folder/template commands
   - launcher/profile/export/import commands
   - extension/cookies/proxy commands
   - rendering and file helper utilities
4. Run verification gates:
   - `go test ./...`
   - `go vet ./...`
5. Produce a short architecture audit with remaining hotspots or deferred work.
