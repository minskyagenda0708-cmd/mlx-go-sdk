# Go Style Guide

This repository follows official Go conventions first:

- `gofmt`: <https://go.dev/blog/gofmt>
- `Effective Go`: <https://go.dev/doc/effective_go>
- `Code Review Comments`: <https://go.dev/wiki/CodeReviewComments>
- `Go Test Comments`: <https://go.dev/wiki/TestComments>

## Practical Rules For This Repository

- Run `gofmt` on all tracked Go files before commit.
- Keep public Go API names idiomatic and stable.
- Use `mixedCaps` for internal and local identifiers, for example
  `startProfile`.
- Prefer readable line wrapping around 80-90 characters when practical, but do
  not fight `gofmt` or introduce awkward wrapping.
- Prefer early returns over deep nesting.
- Keep tests next to the code they verify in `*_test.go`.
- Extract repeated test setup into helpers when the duplication stops helping
  readability.
- Keep CLI command handlers grouped by domain instead of growing one monolithic
  file.

## Non-Goals

- No Python-style fixed 79-column rule.
- No central `Tests/` directory.
- No custom formatting that conflicts with the standard Go toolchain.
