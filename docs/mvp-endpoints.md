# MultiloginX SDK MVP endpoint map

## Scope

The initial SDK should focus on browser profile lifecycle management for Go developers using Rod and similar automation stacks.

### In scope
- workspace folders used to organize profiles
- profile create/search/read/update/delete/restore/clone flows
- launcher start/stop/status flows
- import/export flows
- basic profile metadata and summary retrieval

### Out of scope for MVP
- billing
- account switching and token refresh
- 2FA
- team/user management
- object storage as a separate module
- script runner
- browser core administration
- tags and profile passwords

## Proposed service map

| Service | Endpoint group | Purpose |
|---|---|---|
| `Folders` | workspace folders | manage profile containers |
| `Profiles` | profile management | CRUD-like profile lifecycle |
| `Launcher` | local launcher API | start, stop, and inspect running profiles |
| `Transfers` | profile import/export | move profiles in and out of MultiloginX |
| `Cookies` | pre-made cookies + launcher cookie IO | seed browsing history and move cookies in/out of profiles |

## MVP endpoints

### `Folders` service

| Include | Method | Path | SDK method | Why |
|---|---|---|---|---|
| yes | `GET` | `/workspace/folders` | `List` | needed to resolve `folder_id` for profile operations |
| yes | `POST` | `/workspace/folder_create` | `Create` | needed for profile organization |
| yes | `POST` | `/workspace/folder_update` | `Update` | rename/update folder metadata |
| yes | `POST` | `/workspace/folders_remove` | `Delete` | cleanup and lifecycle support |
| no | `GET` | `/workspace/folders_for_user` | - | team/admin scenario |
| no | `GET` | `/workspace/statistics` | - | informative only, not profile lifecycle |
| no | `GET` | `/workspace/automation_token` | - | token already comes from `MLX_TOKEN` |

### `Profiles` service

| Include | Method | Path | SDK method | Why |
|---|---|---|---|---|
| yes | `POST` | `/profile/create` | `Create` | core profile creation |
| yes | `POST` | `/profile/search` | `Search` | list/filter profiles |
| yes | `POST` | `/profile/update` | `Update` | full profile update |
| yes | `POST` | `/profile/partial_update` | `Patch` | small targeted updates |
| yes | `POST` | `/profile/remove` | `Delete` | soft/hard delete |
| yes | `POST` | `/profile/restore` | `Restore` | recover soft-deleted profiles |
| yes | `POST` | `/profile/clone` | `Clone` | common automation workflow |
| yes | `POST` | `/profile/move` | `Move` | move between folders |
| yes | `POST` | `/profile/metas` | `GetMetas` | retrieve profile metadata in bulk |
| yes | `GET` | `/profile/summary` | `GetSummary` | fingerprint/summary lookup |
| later | `POST` | `/profile/convert` | `Convert` | useful but not core for first iteration |
| later | `POST` | `/tag/*` | - | secondary concern |
| later | `POST` | `/profile/security/*` | - | password workflows can wait |
| later | `POST` | `/profile/login` | - | not part of core SDK value |

### `Launcher` service

| Include | Method | Path | SDK method | Why |
|---|---|---|---|---|
| yes | `GET` | `/api/v2/profile/f/:folder_id/p/:profile_id/start` | `Start` | core launcher integration |
| yes | `GET` | `/api/v1/profile/stop/p/:profile_id` | `Stop` | core launcher integration |
| yes | `GET` | `/api/v1/profile/stop_all` | `StopAll` | useful cleanup operation |
| yes | `GET` | `/api/v1/version` | `Health` | launcher liveness/readiness probe exposed by the SDK via version lookup |
| yes | `GET` | `/api/v1/profile/status/p/:profile_id` | `Status` | single profile runtime status |
| yes | `GET` | `/api/v1/profile/statuses` | `Statuses` | inspect all running profiles |
| yes | `GET` | `/api/v1/profile/quick/statuses` | `QuickStatuses` | useful when mixing quick profiles |
| yes | `GET` | `/api/v1/version` | `Version` | launcher readiness/debugging |
| later | `POST` | `/api/v3/profile/quick` | `StartQuick` | useful, but regular profiles first |
| later | `POST` | `/api/v1/profile/quick/save` | `SaveQuick` | only after quick profile support exists |
| later | `POST` | `/api/v1/proxy/validate` | `ValidateProxy` | adjacent helper, not core MVP |
| yes | `POST` | `/api/v1/cookie_import` | `Cookies.Import` | import explicit or pre-made cookies into a profile |
| yes | `POST` | `/api/v1/cookie_export` | `Cookies.Export` | export cookies from a profile |

### `Cookies` service

| Include | Method | Path | SDK method | Why |
|---|---|---|---|---|
| yes | `GET` | `https://cookies.multilogin.com/api/v1/cookies/metadata/websites` | `ListWebsites` | discover supported pre-made cookie targets such as `google` |
| yes | `POST` | `https://cookies.multilogin.com/api/v1/cookies/metadata` | `CreateMetadata` | attach target website metadata to a profile |
| yes | `PUT` | `https://cookies.multilogin.com/api/v1/cookies/metadata` | `UpdateMetadata` | change pre-made cookie targeting later |
| yes | `GET` | `https://cookies.multilogin.com/api/v1/cookies/:profile_id` | `List` | fetch generated cookie bundles for a profile |

### `Transfers` service

| Include | Method | Path | SDK method | Why |
|---|---|---|---|---|
| yes | `POST` | `/api/v1/profile/:profile_id/export` | `Export` | profile portability |
| yes | `GET` | `/api/v1/profile/exports/:export_id/status` | `ExportStatus` | async operation polling |
| yes | `GET` | `/api/v1/profile/exports/statuses` | `ExportStatuses` | inspect all export jobs |
| yes | `POST` | `/api/v1/profile/import` | `Import` | import profile archive |
| yes | `GET` | `/api/v1/profile/imports/:import_id/status` | `ImportStatus` | async operation polling |
| yes | `GET` | `/api/v1/profile/imports/statuses` | `ImportStatuses` | inspect all import jobs |

## SDK package plan

```text
mlx-go-sdk/
  client.go
  options.go
  request.go
  response.go
  errors.go
  types.go
  auth.go
  profiles.go
  launcher.go
  folders.go
  transfers.go
  archive_manager.go
  cookies.go
  internal/testutil/
  docs/mvp-endpoints.md
```

## First public API draft

```go
client, err := mlx.NewFromEnv()

profileIDs, _, err := client.Profiles.Create(ctx, req)
profiles, _, err := client.Profiles.Search(ctx, searchReq)
profile, _, err := client.Profiles.FindByName(ctx, "Demo", &mlx.FindProfileOptions{StorageType: "all"})
meta, _, err := client.Profiles.GetMeta(ctx, profile.ID)
_, err = client.Profiles.Delete(ctx, mlx.DeleteProfilesRequest{IDs: ids, Permanently: false})

started, _, err := client.Launcher.Start(ctx, folderID, profileID, mlx.StartProfileOptions{Automation: mlx.AutomationRod})
health, _, err := client.Launcher.Health(ctx)
status, _, err := client.Launcher.Status(ctx, profileID)
_, err = client.Launcher.Stop(ctx, profileID)

folders, _, err := client.Folders.List(ctx)
job, _, err := client.Transfers.Export(ctx, profileID)
websites, _, err := client.Cookies.ListWebsites(ctx)
_, _, err = client.Cookies.CreateMetadata(ctx, &mlx.CreateCookiesMetadataRequest{ProfileID: profileID, TargetWebsite: "google"})
```

## TDD rollout order

1. client construction and request plumbing
2. `Folders.List` and `Profiles.Search`
3. `Profiles.Create`, `Update`, `Delete`, `Restore`, `Clone`
4. `Launcher.Start`, `Stop`, `Status`
5. `Transfers.Export` and `Import` polling
6. integration tests against mocked HTTP servers
7. E2E tests gated by env vars:
   - `MLX_TOKEN`
   - optional launcher URL override
   - test folder/profile fixtures

## Operational notes verified or to verify with E2E

- MultiloginX profile capacity is effectively constrained by **active profiles + trash-bin profiles**.
- Soft-deleted profiles still consume profile slots until they are restored or permanently deleted.
- For environments with a 10-profile subscription cap, E2E tests should check combined active and removed profile counts before creating new profiles.
- Live API note: export start responses may return an `export_path` ending in `.zip`, while export status responses may return the same archive path **without** the `.zip` suffix.
- Import accepts a file path via `import_path`; current launcher API shape does **not** expose a destination directory parameter for export.
- Current API shape suggests export location is controlled by the launcher, not by the SDK request payload.
- If one-profile-per-folder archival is required, the SDK will likely need a post-export filesystem orchestration layer outside the current launcher API request surface.
- That filesystem orchestration layer should organize parent folders only and must **never rename the exported `.zip` file itself**.
- Live API note: when using export results as import input, normalize extensionless `export_path` values to the corresponding `.zip` archive path.
- Current checked-in docs/Postman data expose `/api/v1/version`, but no dedicated launcher `/health` endpoint was found; the SDK should use `Launcher.Health` as a readiness probe backed by `Version`.
- Profile lookup convenience helpers should be validated against live API responses because `search` payloads can differ from documentation in subtle ways such as required defaults, returned field coverage, and exact-match filtering behavior.
- Launcher status payloads expose additional typed fields in practice, including `last_launched_at`, `last_launched_on`, and `timestamp`, while `/api/v1/profile/statuses` returns a structured `active_counter` object and `/api/v1/profile/quick/statuses` returns a numeric quick-profile counter.
- Rod integrations must call `rod.New().ControlURL(...).NoDefaultDevice()` when attaching to a Multilogin-launched browser. Skipping `NoDefaultDevice()` lets Rod apply its own default device emulation on top of the Multilogin fingerprint, which can introduce automation artifacts.
- Extensions are resource objects under `object_type_id=6811b909-2e4b-45db-ab62-f14f515523cf`; list them through `/api/v1/resources/metas`, inspect per-profile associations through `/api/v1/resources/profile_object_usages`, and enable or disable them with `/api/v1/resources/:object_id/{enable,disable}_for_profiles`.
- Live extension note: for practical local-profile attachment flows, uploading the extension archive as a **cloud** resource object is the safe default even when the source archive is local.
- The launcher exposes `/api/v1/object_storage/upload` and `/api/v1/object_storage/local_to_cloud` for extension archives, plus `/api/v1/create_extension_from_url` for URL-based ingestion.
- Chrome Web Store IDs can be converted into the standard CRX update URL shape and passed to `/api/v1/create_extension_from_url`, but live validation currently shows the launcher may still fail the fetch with `500 error on downloading extension: failed to fetch extension, status: 404`; treat this as a known live limitation and validate against the current desktop build before depending on it.
- Local profile semantics are currently **split across signals**: live `profile/search` with `storage_type=local` and launcher active counters both classify imported or created local-intended profiles as local, and `parameters.storage.is_local=true` is preserved in `profile/metas`; however, the top-level `is_local` field returned by `GetMeta`/`GetMetas` remained `false` in live validation. SDK workflows should therefore treat `parameters.storage.is_local`, `storage_type` search behavior, and launcher local counters as stronger evidence than the top-level `is_local` metadata field until MLX clarifies or fixes the mismatch.
