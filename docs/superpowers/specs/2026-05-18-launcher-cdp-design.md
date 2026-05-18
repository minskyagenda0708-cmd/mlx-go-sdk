# Launcher CDP Automation Design

## Summary

This design fixes the most critical launcher automation gap in `mlx-go-sdk`:
`AutomationRod` is currently exposed as a first-class launcher automation mode,
but live Multilogin X behavior may return an empty `port` for
`automation_type=rod`. The SDK already documents and tests a practical
fallback: start with `playwright`, then attach Rod to the returned DevTools
endpoint. That logic must move from documentation and E2E tests into the SDK's
actual automation contract.

The approved direction is to keep `AutomationRod` as a public semantic alias
for Rod attachment flows, while sending `automation_type=playwright` to the
launcher. On top of that, the SDK will add CDP-first helpers and a high-level
automation workflow so callers can consume a usable WebSocket endpoint without
hand-building URLs from the launcher port.

## Goals

- Preserve public `AutomationRod` for SDK consumers.
- Make `AutomationRod` behave reliably in live MLX environments by mapping it
  to launcher `playwright`.
- Return explicit metadata about requested automation versus actual launcher
  automation.
- Add ergonomic helpers for CDP WebSocket and Rod-compatible control URLs.
- Add a higher-level workflow that resolves a profile, starts it, optionally
  waits for running status, and returns ready-to-use automation endpoints.
- Drive the change with tests first, then implementation.
- Run a focused cleanup pass for touched automation and workflow code.
- Apply the agreed style rule to touched files: internal identifiers use
  `lowerCamel`, and wrapped lines should stay around 90 columns.

## Non-Goals

- No repository-wide formatting rewrite.
- No breaking rename or removal of public Go identifiers such as
  `AutomationRod`.
- No attempt to redesign unrelated services or CLI command families.
- No broad "automation session manager" abstraction beyond the approved helper
  and workflow scope.

## Current State

The current SDK state is internally inconsistent:

- `Launcher.Start(...)` forwards `AutomationRod` as literal launcher
  `automation_type=rod`.
- `docs/rod-example.md` already documents that live MLX may return an empty
  `port` for `rod`, and recommends retrying with `playwright`.
- `e2e/e2e_test.go` contains fallback logic that stops the profile and retries
  with `AutomationPlaywright` if the Rod start response has an empty `port`.
- Consumers still need to manually turn the returned `port` into a usable CDP
  or Rod connection endpoint.

That means the reliable production contract exists only in docs and tests, not
in the SDK surface itself.

## Alternatives Considered

### Option A: CDP-first ergonomics

- Keep public `AutomationRod`.
- Internally map `Rod -> Playwright` for launcher start requests.
- Add response metadata plus endpoint helpers and one high-level workflow.

Pros:

- Backward-compatible for callers already using `AutomationRod`.
- Moves live workaround into the SDK where it belongs.
- Gives consumers a clean CDP-first contract for both Rod and Playwright.
- Adds meaningful ergonomics without overbuilding a larger abstraction layer.

Cons:

- Slightly widens the public API.
- Requires careful naming so launcher behavior is explicit.

### Option B: Minimal fix

- Map `Rod -> Playwright`.
- Add one helper only.

Pros:

- Smallest implementation.

Cons:

- Leaves too much manual URL and workflow work to consumers.
- Misses the chance to make the SDK meaningfully easier to use.

### Option C: Wider automation session layer

- Include all of Option A.
- Add a broader "session" abstraction for automation clients.

Pros:

- Very ergonomic.

Cons:

- Wider surface area than needed for the current problem.
- Higher design and maintenance cost.

## Approved Design

Option A is approved.

The SDK will expose a CDP-first automation contract:

1. `AutomationRod` remains public and keeps its semantic meaning for callers.
2. Launcher requests treat `AutomationRod` as an alias for launcher
   `playwright`.
3. Start results expose both the caller-requested automation mode and the
   launcher mode actually used.
4. The SDK exposes helpers to derive a validated CDP WebSocket endpoint and a
   Rod-compatible control URL from launcher output.
5. A high-level workflow returns the resolved profile plus normalized
   automation endpoint data.

## API Design

### 1. Launcher automation aliasing

`Launcher.Start(...)` will normalize the requested automation before sending
the launcher request.

Behavior:

- requested `selenium` -> launcher `selenium`
- requested `playwright` -> launcher `playwright`
- requested `puppeteer` -> launcher `puppeteer`
- requested `rod` -> launcher `playwright`

The public API should still accept `AutomationRod`, but the launcher query
parameter must use `playwright`.

### 2. Start response enrichment

`StartProfileResponse` and/or its nested data model will be extended so callers
can inspect:

- the requested automation type
- the normalized launcher automation type
- the raw launcher port
- the normalized CDP port
- the normalized CDP WebSocket URL when derivable

The intent is to make the "Rod attaches through Playwright/CDP" behavior
explicit rather than implicit.

### 3. Endpoint helpers

The SDK will add helper behavior for start results. The exact method names can
follow existing file patterns, but the feature set must cover:

- return a usable CDP WebSocket URL from a valid start response
- return a Rod-compatible control URL from that same start response
- validate that the launcher returned a usable endpoint and fail with a typed
  error when it did not

These helpers should centralize all endpoint normalization and remove the need
for callers to manually interpret `Data.Port`.

### 4. High-level automation workflow

Add a workflow that:

- resolves a profile by name
- starts the profile with the requested automation mode
- optionally waits for running status
- returns the resolved profile, start response, and normalized automation
  endpoint data

This workflow should sit beside the existing verified workflows in
`workflows.go` and reuse the same lookup and polling patterns.

## Error Handling

The SDK should not leave callers with a silent empty string when launcher
automation output is unusable.

Required behavior:

- empty or blank `port` produces a typed SDK error with clear context
- invalid endpoint normalization produces a typed SDK error with the relevant
  raw values
- workflow methods propagate those typed errors directly

The error text should mention that the launcher did not return a usable CDP
endpoint, not just that "port is empty".

## Testing Strategy

This change is explicitly test-driven.

### Unit tests

Add failing unit tests first for:

- `AutomationRod` being sent to the launcher as query parameter
  `playwright`
- start results preserving requested automation metadata as `rod`
- endpoint helper success from a valid launcher port
- endpoint helper failures for empty or invalid ports
- Rod control URL derivation from the normalized CDP endpoint

### Integration tests

Add failing integration tests first for:

- high-level automation workflow by profile name
- optional wait-for-running behavior within that workflow
- returned normalized endpoint data from the workflow result

### E2E tests

Update live E2E coverage so the SDK itself owns the Rod-to-Playwright behavior.
The E2E test should verify the new contract rather than reimplementing the
fallback as primary business logic inside the test body.

## Documentation Changes

Update the touched documentation to match the new contract:

- `docs/rod-example.md`
- `docs/consumer-guide.md`
- README sections that currently describe Rod fallback behavior
- any CLI or testing docs that mention launcher automation expectations

The new wording should describe `AutomationRod` as a Rod attachment mode backed
by a Playwright/CDP launcher start, not as a distinct live launcher transport.

## Code Quality Pass

After implementation, run a focused review of touched areas for:

- duplicated endpoint normalization logic
- dead or now-obsolete fallback branches
- inconsistent naming between launcher, workflow, and docs layers
- comment drift versus actual live behavior

The cleanup scope is intentionally narrow:

- `launcher.go`
- `workflows.go`
- new automation helper code
- related tests
- touched docs

## Formatting and Naming Rules

For files changed by this work:

- wrap lines to approximately 90 columns where reasonable
- use `lowerCamel` for internal helper names and local variables
- keep public Go identifiers idiomatic and unchanged unless a new exported API
  is necessary

This preserves Go ecosystem conventions while still improving local consistency
in touched files.

## Implementation Notes

- Prefer a single normalization path for launcher automation modes.
- Prefer one source of truth for endpoint derivation.
- Keep helpers small and composable rather than creating a larger session
  framework.
- Reuse existing polling and verified-lookup patterns from `workflows.go`.

## Acceptance Criteria

- `AutomationRod` no longer depends on callers or tests manually retrying with
  `playwright`.
- `Launcher.Start(...)` uses launcher `playwright` when the caller requests
  `rod`.
- Start results expose requested versus launcher automation clearly.
- Callers can obtain a normalized CDP WebSocket endpoint from SDK helpers.
- Callers can obtain a Rod-compatible control URL from SDK helpers.
- A high-level workflow returns resolved profile and ready-to-use automation
  endpoint data.
- New and updated tests pass under `go test ./...`.
- Touched docs match the implemented contract.
- Touched files receive a focused quality cleanup.
