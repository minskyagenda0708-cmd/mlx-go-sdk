# Quick Profiles, Proxy Validation, Tags, and Local/Cloud Helpers Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Multilogin X SDK coverage for quick-profile launcher flows, proxy validation, tag management, and convenience local/cloud profile creation helpers.

**Architecture:** Follow existing SDK patterns: one file per service (`tags.go`, extend `launcher.go`), typed request/response models, co-located tests, and wire new services into `Client`. Keep local/cloud helpers as workflow-layer conveniences over existing `Profiles.Create`. Proxy validation lives on the launcher service because it uses the launcher base URL.

**Tech Stack:** Go 1.26, `httptest`, `bd` issue tracking, existing SDK retry/polling helpers.

---

## Files

| File | Responsibility |
|---|---|
| `tags.go` | `TagsService` interface, request/response models, and API calls |
| `tags_test.go` | Mocked unit tests for all tag endpoints |
| `launcher.go` | Add `StartQuick`, `SaveQuick`, `ConvertQuickToProfile`, `ValidateProxy` |
| `launcher_test.go` | Add quick-profile and proxy-validate mocked tests |
| `workflows.go` | Add `CreateLocalProfile` and `CreateCloudProfile` helpers |
| `integration/workflows_test.go` | Workflow-level mocked coverage for new helpers |
| `client.go` | Wire `Tags` field into `Client` |

---

## Task 1: Tags Service

**Files:**
- Create: `tags.go`
- Create: `tags_test.go`
- Modify: `client.go` (add `Tags TagsService` field + initialization)

Endpoints from Postman:
- `POST /tag/create` — create tags
- `POST /tag/update` — update tags
- `POST /tag/remove` — remove tags
- `POST /tag/assign_to_profiles` — assign tags to profiles
- `POST /tag/search` — search tags

Models:
- `Tag` {ID, Name, Color, CreatedAt, UpdatedAt, CreatedBy, InUseCount}
- `CreateTagsRequest`, `UpdateTagsRequest`, `RemoveTagsRequest`, `AssignTagsRequest`, `SearchTagsRequest`
- `TagsResponse`, `SearchTagsResponse`

- [ ] **Step 1: Write failing tag tests**
  Create `tags_test.go` with `TestTagsCreate`, `TestTagsUpdate`, `TestTagsRemove`, `TestTagsAssign`, `TestTagsSearch` against `httptest` server fixtures. Run them to verify they fail (types and service do not exist).
- [ ] **Step 2: Implement tags.go**
  Implement the interface and methods, following the `Profiles` service pattern (POST JSON, bearer token, status envelope).
- [ ] **Step 3: Wire into Client**
  Add `Tags` field and `&TagsServiceOp{client: c}` in `client.go` `New`/`NewFromEnv` path.
- [ ] **Step 4: Run tests and commit**
  `go test ./... -run TestTags` should pass.

---

## Task 2: Launcher Quick Profiles + Proxy Validate

**Files:**
- Modify: `launcher.go`
- Modify: `launcher_test.go`

Endpoints from Postman:
- `POST /api/v3/profile/quick` — `StartQuick`
- `POST /api/v1/profile/quick/save` — `SaveQuick`
- `POST /api/v1/profile/quick/convert` — `ConvertQuickToProfile` (query param `profile_id`)
- `POST /api/v1/proxy/validate` — `ValidateProxy`

Models:
- `StartQuickProfileRequest` / `StartQuickProfileResponse` (reuse `StartedProfileData` if response shape matches)
- `SaveQuickProfileRequest` / `SaveQuickProfileResponse`
- `ConvertQuickProfileRequest` / response (likely empty data)
- `ValidateProxyRequest` {Type, Host, Port, Username, Password}
- `ValidateProxyResponse` {Data: ProxyValidationData}
- `ProxyValidationData` {Accuracy, Altitude, CountryCode, IP, Latitude, Longitude, Timezone}

- [ ] **Step 1: Write failing launcher tests**
  Add `TestLauncherStartQuick`, `TestLauncherSaveQuick`, `TestLauncherConvertQuickToProfile`, `TestLauncherValidateProxy` in `launcher_test.go`. Run to confirm failures.
- [ ] **Step 2: Implement methods in launcher.go**
  Add to `LauncherService` interface and `LauncherServiceOp`.
- [ ] **Step 3: Run tests and commit**
  `go test ./... -run "TestLauncherStartQuick|TestLauncherSaveQuick|TestLauncherConvertQuick|TestLauncherValidateProxy"` should pass.

---

## Task 3: Local/Cloud Profile Helpers

**Files:**
- Modify: `workflows.go`
- Modify: `integration/workflows_test.go`

These are convenience workflows because the API endpoint is the same (`/profile/create`) with `parameters.storage.is_local` toggled.

- `Workflows.CreateLocalProfile(ctx, *CreateProfileRequest, ...)` — sets `Parameters.Storage.IsLocal = true` then calls `CreateProfilesAndVerify`.
- `Workflows.CreateCloudProfile(ctx, *CreateProfileRequest, ...)` — sets `Parameters.Storage.IsLocal = false` then calls `CreateProfilesAndVerify`.

- [ ] **Step 1: Write failing workflow tests**
  Add `TestWorkflowCreateLocalProfile` and `TestWorkflowCreateCloudProfile` that assert `Storage.IsLocal` in the outgoing create request.
- [ ] **Step 2: Implement helpers**
  Add methods in `workflows.go` that normalize `Parameters.Storage` and delegate to `CreateProfilesAndVerify`.
- [ ] **Step 3: Run tests and commit**

---

## Task 4: Folders verification

**Files:**
- Review: `folders.go`, `folders_test.go`

- [ ] **Step 1: Verify full coverage**
  Confirm `List`, `Create`, `Update`, `Delete` are present and tested. If any method lacks a direct unit test, add it. No new API surface needed.

---

## Task 5: Quality gates and final integration

**Files:**
- All modified files

- [ ] **Step 1: Format**
  `gofmt -w .`
- [ ] **Step 2: Vet**
  `go vet ./...`
- [ ] **Step 3: Test**
  `go test ./...`
- [ ] **Step 4: Create/close bd issues**
  Create beads issues before implementation (one per task) and close them as tasks complete.
- [ ] **Step 5: Commit**
  `git add -A && git commit -m "feat: add quick profiles, proxy validation, tags, local/cloud helpers"`
