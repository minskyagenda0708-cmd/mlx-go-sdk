# CLI reference

The reference CLI is a consumer-facing command-line wrapper around `mlx-go-sdk`.

It is designed for the same scope as the SDK’s documented consumer workflows:

- folders
- profiles
- profile templates
- launcher control
- export/import
- extensions
- cookies
- proxies

Out of scope:

- interactive login flows
- token storage in config files
- mobile profile commands

## Authentication

The CLI uses the same environment-driven authentication model as the SDK.

Required environment variable:

- `MLX_TOKEN`

Optional endpoint overrides:

- `MLX_BASE_URL`
- `MLX_LAUNCHER_URL`
- `MLX_COOKIES_URL`
- `MLX_PROXY_URL`

CLI-specific optional environment overrides:

- `MLX_CONFIG_FILE`
- `MLX_OUTPUT`
- `MLX_TIMEOUT`
- `MLX_USER_AGENT`

Important rules:

- `MLX_TOKEN` is the only auth input
- no interactive login is supported
- the config file does not store a token

## Binary name

Examples below use the binary name `mlx`.

## Global flags

Available on all commands:

- `--config <path>` — path to the CLI config file
- `--output <format>` — override output format for the current command
- `-h`, `--help` — show help
- `--version` — show version

Supported output formats:

- `table`
- `json`
- `yaml`

## Command overview

    mlx config
    mlx folder
    mlx launcher
    mlx profile
    mlx template
    mlx export
    mlx import
    mlx extension
    mlx cookies
    mlx proxy
    mlx version
    mlx help

## Name vs ID selection

Many commands accept either an explicit object ID or a profile name.

Common selector rules:

- use `--id` or `--profile-id` when you already know the exact object ID
- use `--name` or `--profile-name` when you want verified exact-name resolution
- ID and name selectors are mutually exclusive
- when resolving by name, `--folder-id` can narrow the lookup
- exact-name lookup is preferred over fuzzy first-match behavior

## Template commands

SDK mapping:

- `client.Resources.ListProfileTemplates`
- `client.Resources.GetMeta`
- `client.Resources.Download`

### `mlx template list`

List available profile template resources.

Usage:

    mlx template list
    mlx template list --name Template
    mlx template list --limit 20 --offset 0
    mlx template list --trashbin

Flags:

- `--name <text>`
- `--limit <n>`
- `--offset <n>`
- `--trashbin`

Notes:

- this command lists profile template resource metadata, not profiles
- it is the discovery step before template-based profile creation

### `mlx template get`

Get one profile template resource.

Usage:

    mlx template get --id tpl-123
    mlx template get --id tpl-123 --output json

Notes:

- metadata is fetched through the resources API
- the command may also surface the downloaded template path and parsed template body used for creation workflows when available

## Configuration commands

### `mlx config path`

Print the resolved config path.

Usage:

    mlx config path

### `mlx config show`

Print the effective config after file loading and environment overrides.

Usage:

    mlx config show
    mlx config show --output json

### `mlx config init`

Write a default config file.

Usage:

    mlx config init
    mlx config init --path C:\Users\you\AppData\Roaming\mlx-go-sdk\config.json
    mlx config init --force

Notes:

- the reference CLI currently initializes a JSON config file
- if the target file exists, use `--force` to overwrite it

## Folder commands

SDK mapping:

- `client.Folders.List`
- `client.Folders.Create`
- `client.Folders.Update`
- `client.Folders.Delete`

### `mlx folder list`

List workspace folders.

Usage:

    mlx folder list

### `mlx folder create`

Create a folder.

Usage:

    mlx folder create --name "QA"
    mlx folder create --name "QA" --comment "Shared test folder"

### `mlx folder update`

Update an existing folder.

Usage:

    mlx folder update --id folder-123 --name "QA Updated"
    mlx folder update --id folder-123 --name "QA Updated" --comment "New comment"

### `mlx folder delete`

Delete one or more folders.

Usage:

    mlx folder delete --ids folder-123
    mlx folder delete --ids folder-123,folder-456

## Launcher commands

SDK mapping:

- `client.Launcher.Health`
- `client.Launcher.Version`
- `client.Launcher.Status`
- `client.Launcher.Statuses`
- `client.Launcher.Start`
- `client.Launcher.Stop`
- `client.Launcher.StopAll`

Workflow-backed variants:

- `client.Workflows.StartProfileByName`
- `client.Workflows.StopProfileByName`

### `mlx launcher health`

Check launcher reachability/readiness.

Usage:

    mlx launcher health

### `mlx launcher version`

Print launcher version information.

Usage:

    mlx launcher version

### `mlx launcher status`

Get runtime status for a single profile.

Usage:

    mlx launcher status --profile-id profile-123
    mlx launcher status --profile-name "Demo"

### `mlx launcher statuses`

Get all known launcher runtime states.

Usage:

    mlx launcher statuses

### `mlx launcher start`

Start a profile.

Usage:

    mlx launcher start --profile-id profile-123 --folder-id folder-123
    mlx launcher start --profile-name "Demo" --wait
    mlx launcher start --profile-name "Demo" --automation-type rod
    mlx launcher start --profile-id profile-123 --folder-id folder-123 --headless --strict

Flags:

- `--profile-id <id>`
- `--profile-name <name>`
- `--folder-id <id>`
- `--automation-type <selenium|playwright|puppeteer|rod>`
- `--headless`
- `--strict`
- `--wait`
- `--skip-proxy-check` (DANGER: bypasses the fail-closed proxy continuity check; the profile launches even if its proxy is unhealthy or unreachable)
- `--proxy-threshold-ms <n>` (override the soft latency threshold in milliseconds for this launch)
- `--proxy-hard-cap-ms <n>` (override the hard latency cap in milliseconds for this launch)

Notes:

- when you use `--profile-name`, the CLI uses the workflow helper for exact-name resolution
- `--wait` waits for a running status using the configured polling policy
- when proxy continuity is enabled (`defaults.proxy_continuity`), the CLI runs a fail-closed proxy health check before launching; an unhealthy or unreachable proxy aborts the start (see [proxy-workflows.md](proxy-workflows.md#proxy-continuity))
- `--proxy-threshold-ms` / `--proxy-hard-cap-ms` override the configured two-tier latency thresholds for a single launch; `--skip-proxy-check` disables the check entirely

### `mlx launcher stop`

Stop a profile.

Usage:

    mlx launcher stop --profile-id profile-123
    mlx launcher stop --profile-name "Demo" --ignore-already-stopped
    mlx launcher stop --profile-name "Demo" --wait

Flags:

- `--profile-id <id>`
- `--profile-name <name>`
- `--folder-id <id>`
- `--ignore-already-stopped`
- `--wait`

Notes:

- when you use `--profile-name`, the CLI uses the workflow helper for exact-name resolution
- `--wait` waits until the profile is no longer reported as running

### `mlx launcher stop-all`

Stop all running profiles, optionally filtered by type.

Usage:

    mlx launcher stop-all
    mlx launcher stop-all --type local

## Profile commands

SDK mapping:

- `client.Profiles.Search`
- `client.Profiles.GetMeta`
- `client.Profiles.Create`
- `client.Profiles.Update`
- `client.Profiles.Patch`
- `client.Profiles.Clone`
- `client.Profiles.Move`
- `client.Profiles.Delete`
- `client.Profiles.Restore`
- `client.Profiles.GetSummary`

Workflow-backed variants:

- `client.Workflows.FindProfileByNameVerified`
- `client.Workflows.CreateProfilesAndVerify`

### `mlx profile list`

Search and list profiles.

Usage:

    mlx profile list
    mlx profile list --search Demo
    mlx profile list --search Demo --storage-type local
    mlx profile list --removed
    mlx profile list --limit 50 --offset 50

Flags:

- `--search <text>`
- `--removed`
- `--limit <n>`
- `--offset <n>`
- `--storage-type <all|local|cloud>`
- `--folder-id <id>`
- `--browser-type <browser>`
- `--os-type <os>`
- `--order-by <field>`
- `--sort <asc|desc>`
- `--tags <tag1,tag2,...>`

### `mlx profile get`

Get one profile meta record.

Usage:

    mlx profile get --id profile-123
    mlx profile get --name "Demo"

### `mlx profile create`

Create one or more profiles either from a JSON request payload or from a stored profile template resource.

Usage:

    mlx profile create --file create-profile.json
    mlx profile create --file create-profile.json --wait
    mlx profile create --template-id tpl-123 --name "Demo"
    mlx profile create --template-id tpl-123 --name "Demo" --folder-id folder-123 --local --wait
    mlx profile create --template-id tpl-123 --name "Demo" --folder-id folder-123 --managed-proxy --proxy-country us
    mlx profile create --template-id tpl-123 --name "Demo" --folder-id folder-123 --managed-proxy --proxy-country us --start

JSON payload mode:

- `--file <path>` must match `mlx.CreateProfileRequest`

Example payload:

    {
      "name": "Demo",
      "browser_type": "mimic",
      "folder_id": "folder-123",
      "os_type": "windows",
      "parameters": {
        "storage": {
          "is_local": true
        }
      }
    }

Template mode:

- `--template-id <id>` selects an existing profile template resource
- the CLI downloads the template body, reads its `mainParams`, and materializes a `mlx.CreateProfileRequest`
- `--name` is used as the created profile name
- `--folder-id` supplies the destination folder when the template does not already encode the target folder
- `--local` forces `parameters.storage.is_local=true`
- `--managed-proxy` generates an MLX managed proxy and applies it to the created profile request before creation

Launch after creation:

- `--start` launches the first created profile immediately after creation (non-`--wait` mode only)
- `--start` cannot be combined with `--wait`; run create without `--wait` to use it
- when proxy continuity is enabled, the same fail-closed proxy health check used by `mlx launcher start` runs before the launch, so a created profile with an unhealthy or unreachable proxy will not be launched (see [proxy-workflows.md](proxy-workflows.md#proxy-continuity))
- output is a combined object: `{"create": <create response>, "start": <start response>}`

Common proxy flags for template mode:

- `--proxy-country <code>`
- `--proxy-region <name>`
- `--proxy-city <name>`
- `--proxy-protocol <socks5|http>`
- `--proxy-session-type <sticky|rotating>`
- `--proxy-ip-ttl <seconds>`
- `--proxy-strict`

Notes:

- use either `--file` or `--template-id`
- `--wait` uses the verified workflow and waits until created metas are readable
- template-based creation is useful when operators want to reuse one shared stored profile template while still applying creation-time overrides such as name, storage mode, folder, and proxy

### `mlx profile update`

Fully update a profile from a JSON payload.

Usage:

    mlx profile update --file update-profile.json

The JSON file must match `mlx.UpdateProfileRequest`.

### `mlx profile patch`

Partially update a profile from a JSON payload.

Usage:

    mlx profile patch --file patch-profile.json

The JSON file must match `mlx.PatchProfileRequest`.

### `mlx profile clone`

Clone an existing profile.

Usage:

    mlx profile clone --id profile-123
    mlx profile clone --name "Demo" --times 3

### `mlx profile move`

Move profiles to another folder.

Usage:

    mlx profile move --ids profile-123 --dest-folder-id folder-456
    mlx profile move --ids profile-123,profile-456 --dest-folder-id folder-456

### `mlx profile delete`

Delete profiles.

Usage:

    mlx profile delete --ids profile-123
    mlx profile delete --ids profile-123,profile-456 --permanently

### `mlx profile restore`

Restore soft-deleted profiles.

Usage:

    mlx profile restore --ids profile-123
    mlx profile restore --ids profile-123,profile-456

### `mlx profile summary`

Get the fingerprint summary view for a profile.

Usage:

    mlx profile summary --id profile-123
    mlx profile summary --name "Demo"

## Export commands

SDK mapping:

- `client.Transfers.ExportStatus`
- `client.Transfers.ExportStatuses`
- `client.Archives.ExportProfileToFolder`

Workflow-backed variant:

- `client.Workflows.ExportProfileByNameToFolder`

### `mlx export run`

Export a profile into an organized archive folder.

Usage:

    mlx export run --profile-id profile-123 --root-dir C:\mlx-exports
    mlx export run --profile-name "Demo" --root-dir C:\mlx-exports
    mlx export run --profile-name "Demo" --root-dir C:\mlx-exports --folder-name "Release Exports"
    mlx export run --profile-name "Demo" --root-dir C:\mlx-exports --stop-before-export

Flags:

- `--profile-id <id>`
- `--profile-name <name>`
- `--folder-id <id>`
- `--root-dir <dir>`
- `--folder-name <name>`
- `--profile-name-override <name>`
- `--stop-before-export`
- `--ignore-stop-not-ready`

Notes:

- `--root-dir` is required
- when you export by profile name, the CLI uses the verified workflow helper
- when you export by profile ID, the CLI uses archive manager export directly
- organized exports normalize launcher archive path quirks for follow-up import use

### `mlx export status`

Get one export job status.

Usage:

    mlx export status --export-id export-123

### `mlx export statuses`

List export jobs.

Usage:

    mlx export statuses

## Import commands

SDK mapping:

- `client.Transfers.Import`
- `client.Transfers.ImportStatus`
- `client.Transfers.ImportStatuses`

Workflow-backed variant:

- `client.Workflows.ImportProfileAndVerify`

### `mlx import run`

Import a profile archive.

Usage:

    mlx import run --import-path C:\mlx-exports\demo.zip
    mlx import run --import-path C:\mlx-exports\demo.zip --is-local
    mlx import run --import-path C:\mlx-exports\demo.zip --wait

Flags:

- `--import-path <archive.zip>`
- `--is-local`
- `--wait`

Notes:

- `--wait` verifies that the imported profile meta is readable after import completion

### `mlx import status`

Get one import job status.

Usage:

    mlx import status --import-id import-123

### `mlx import statuses`

List import jobs.

Usage:

    mlx import statuses

## Extension commands

SDK mapping:

- `client.Resources.ListExtensions`
- `client.Resources.GetMeta`
- `client.Resources.UploadExtension`
- `client.Resources.CreateExtensionFromURL`
- `client.Resources.CreateExtensionFromChromeWebStore`
- `client.Resources.EnableExtensionForProfiles`
- `client.Resources.DisableExtensionForProfiles`
- `client.Resources.ObjectProfileUsages`
- `client.Resources.ProfileExtensionUsages`
- `client.Resources.Download`
- `client.Resources.Delete`
- `client.Resources.Restore`

Workflow-backed variant:

- `client.Workflows.EnableExtensionForProfileByName`

### `mlx extension list`

List extension resource objects.

Usage:

    mlx extension list
    mlx extension list --name uBlock
    mlx extension list --trashbin

### `mlx extension get`

Get extension metadata.

Usage:

    mlx extension get --id ext-123

### `mlx extension upload`

Upload a local extension archive.

Usage:

    mlx extension upload --path C:\extensions\demo.zip
    mlx extension upload --path C:\extensions\demo.zip --storage-type cloud

Notes:

- cloud-backed storage is the recommended default for extension workflows

### `mlx extension create-url`

Create an extension resource from a downloadable URL.

Usage:

    mlx extension create-url --url https://example.test/demo.zip
    mlx extension create-url --url https://example.test/demo.zip --browser-type mimic --storage-type cloud

### `mlx extension create-webstore`

Create an extension resource from a Chrome Web Store ID.

Usage:

    mlx extension create-webstore --extension-id ghbmnnjooekpmoecnnnilnnbdlolhkhi

Note:

- Chrome Web Store creation is best-effort because launcher-side fetch behavior may vary in live environments

### `mlx extension enable`

Enable an extension for a profile.

Usage:

    mlx extension enable --id ext-123 --profile-id profile-123
    mlx extension enable --id ext-123 --profile-name "Demo"
    mlx extension enable --id ext-123 --profile-name "Demo" --require-profile-usage-read

Notes:

- when you use `--profile-name`, the CLI uses the verified workflow helper
- object-centric verification is preferred in some live environments

### `mlx extension disable`

Disable an extension for a profile.

Usage:

    mlx extension disable --id ext-123 --profile-id profile-123
    mlx extension disable --id ext-123 --profile-name "Demo"

### `mlx extension usages`

Inspect extension usages.

Usage:

    mlx extension usages --id ext-123
    mlx extension usages --profile-id profile-123
    mlx extension usages --profile-name "Demo"

Behavior:

- `--id` returns object-to-profile usages
- `--profile-id` or `--profile-name` returns profile-to-extension usages

### `mlx extension download`

Download a resource through launcher-backed object storage.

Usage:

    mlx extension download --id ext-123

### `mlx extension delete`

Delete an extension resource.

Usage:

    mlx extension delete --id ext-123
    mlx extension delete --id ext-123 --permanently

### `mlx extension restore`

Restore an extension resource from trash.

Usage:

    mlx extension restore --id ext-123

## Cookies commands

SDK mapping:

- `client.Cookies.ListWebsites`
- `client.Cookies.CreateMetadata`
- `client.Cookies.UpdateMetadata`
- `client.Cookies.List`
- `client.Cookies.Import
