# Consumer guide

This SDK is designed to be embedded into external automation tools, workers, and CLI applications rather than used only as a thin request wrapper.

The safest production pattern is:

1. create one configured client
2. keep transport, retry, and timeout policy in one place
3. build app-specific commands under `cmd/` in the consuming application
4. use verified workflows for stateful operations
5. treat live MLX edge cases as first-class design constraints

## Recommended project layout

A consuming application can stay simple:

```text
my-mlx-tool/
  cmd/
    profiles/
      main.go
    export/
      main.go
    cookies/
      main.go
  internal/
    app/
      client.go
      config.go
      profiles.go
      exports.go
      cookies.go
  go.mod
```

- `cmd/...` holds user-facing entrypoints
- `internal/app/client.go` builds one shared SDK client
- business logic stays outside of `main.go`

## Build one reusable client

```go
package app

import (
    "time"

    mlx "mlx-go-sdk"
)

func NewClient() (*mlx.Client, error) {
    return mlx.NewFromEnv(
        mlx.WithTimeout(30*time.Second),
        mlx.WithRetry(mlx.RetryOptions{
            MaxAttempts:     4,
            InitialInterval: 500 * time.Millisecond,
            MaxInterval:     3 * time.Second,
            Multiplier:      2,
            Jitter:          0.2,
        }),
        mlx.WithUserAgent("acme-mlx-cli/1.0"),
    )
}
```

This keeps auth, retry policy, and transport configuration centralized.

## Profile creation and verification

When profile creation matters to downstream work, prefer verified workflows over raw create calls.

```go
created, err := client.Workflows.CreateProfilesAndVerify(ctx, &mlx.CreateProfileRequest{
    Name:        "Demo",
    BrowserType: "mimic",
    FolderID:    folderID,
    OSType:      "windows",
    Parameters: &mlx.ProfileParameters{
        Storage: &mlx.Storage{IsLocal: true},
    },
}, mlx.CreateProfilesAndVerifyOptions{
    PollOptions: mlx.PollOptions{
        InitialInterval: 500 * time.Millisecond,
        MaxInterval:     2 * time.Second,
        Timeout:         30 * time.Second,
    },
})
if err != nil {
    return err
}

fmt.Println("verified profiles:", len(created.Profiles))
```

## Rod attachment

Use the Rod guide in `docs/rod-example.md`. The SDK now owns the Rod-to-Playwright normalization, so consumers should request `AutomationRod` and use the resolved control URL from the start result or workflow helper.

```go
result, err := client.Workflows.StartProfileAutomationByName(ctx, "Demo", mlx.StartProfileAutomationByNameOptions{
    StartOptions: mlx.StartProfileOptions{
        AutomationType: mlx.AutomationRod,
    },
    WaitForRunning: true,
})
if err != nil {
    return err
}

browser := rod.New().
    ControlURL(result.RodControlURL).
    NoDefaultDevice()
```

`result.RequestedAutomation` stays `rod`, while the launcher automation is normalized to `playwright` and the SDK resolves the Rod control URL for you.

## Local profile handling

Local/cloud detection in live MLX environments is nuanced.

Production guidance:

- prefer verified search behavior with `storage_type=local`
- prefer `parameters.storage.is_local`
- do not rely on the raw top-level `ProfileMeta.IsLocal` field alone

If your tool depends on storage semantics, keep those checks explicit and close to the workflow that uses them.

## Extension workflows

Typical production flow:

1. upload an extension object
2. attach it to a verified profile
3. verify object-to-profile usage

```go
uploaded, _, err := client.Resources.UploadExtension(ctx, &mlx.UploadExtensionRequest{
    ObjectPath: `C:\extensions\demo.zip`,
})
if err != nil {
    return err
}

_, err = client.Workflows.EnableExtensionForProfileByName(ctx, "Demo", uploaded.Data.MetaID, mlx.EnableExtensionForProfileByNameOptions{
    PollOptions: mlx.PollOptions{
        InitialInterval: 500 * time.Millisecond,
        MaxInterval:     2 * time.Second,
        Timeout:         20 * time.Second,
    },
})
if err != nil {
    return err
}
```

For Chrome Web Store flows, `CreateExtensionFromChromeWebStore(...)` is best-effort because live launcher fetches may still fail for public extensions.

## Cookies workflows

For pre-made cookie seeding, prefer the high-level helper:

```go
seeded, err := client.Cookies.SeedProfileCookies(ctx, mlx.SeedProfileCookiesOptions{
    ProfileID:               profileID,
    TargetWebsite:           "google.com",
    CreateMetadataIfMissing: true,
    ImportAdvancedCookies:   false,
})
if err != nil {
    return err
}

fmt.Println("imported cookies:", seeded.CookieCount)
```

This helper can:

- create or update metadata
- fetch generated cookie bundles
- resolve folder id when omitted
- import the selected cookie bundle into the launcher profile

## Import and export workflows

### Export

```go
exported, err := client.Workflows.ExportProfileByNameToFolder(ctx, "Demo", mlx.ExportProfileByNameToFolderOptions{
    StopBeforeExport: true,
    ExportOptions: mlx.ExportProfileToFolderOptions{
        RootDir: `C:\mlx-exports`,
    },
})
if err != nil {
    return err
}

fmt.Println(exported.Export.Archive.ArchivePath)
```

This gives a stable archive folder structure suitable for follow-up processing by other tools.

### Import

```go
imported, err := client.Workflows.ImportProfileAndVerify(ctx, &mlx.ImportProfileRequest{
    ImportPath: `C:\mlx-exports\demo.zip`,
    IsLocal:    true,
}, mlx.ImportProfileWorkflowOptions{
    PollOptions: mlx.PollOptions{
        InitialInterval: 500 * time.Millisecond,
        MaxInterval:     2 * time.Second,
        Timeout:         30 * time.Second,
    },
})
if err != nil {
    return err
}

fmt.Println(imported.ProfileMeta.ID)
```

## Safe production patterns

### 1. Prefer verified workflows for stateful operations

Use raw service methods for simple CRUD-style calls.
Use `client.Workflows` when your next step depends on confirmed state.

### 2. Push transport policy into the client, not every call site

Configure timeout, retry, and user-agent once.

### 3. Respect context cancellation everywhere

Pass request-scoped contexts from CLI commands, workers, or HTTP handlers.

### 4. Surface typed error classes to operators

`ClassifyError`, `IsRetryableError`, and `RetryAfter` help external tools decide whether to retry, fail fast, or back off.

### 5. Keep filesystem concerns separate

Use `client.Archives` or export workflows for archive organization rather than ad-hoc file moves in command handlers.

### 6. Treat live MLX quirks as normal conditions

Examples:

- Rod automation should request `AutomationRod` and use the SDK-resolved control URL instead of reimplementing launcher fallback logic
- extension profile-usage reads may be weaker than object-usage reads
- local profile semantics should not depend on the raw top-level meta flag
- proxy retention should use parsed affinity metadata rather than assuming identical regenerated credentials

## Suggested consumer entrypoints under `cmd/`

For a downstream consumer CLI, good candidate `cmd/` entrypoints are:

- `cmd/profiles`
- `cmd/exports`
- `cmd/imports`
- `cmd/extensions`
- `cmd/cookies`
- `cmd/proxy`

Within those entrypoints, expose focused subcommands such as:

- `profiles create`
- `profiles start`
- `profiles stop`
- `exports run`
- `imports run`
- `extensions enable`
- `cookies seed`
- `proxy assign`

Each command should:

- load config
- build one shared client
- call one focused application function
- return typed, operator-readable errors
