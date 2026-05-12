# Verified high-level workflows

The SDK exposes high-level workflow helpers under `client.Workflows` for common Multilogin X operations that need explicit post-condition checks.

These helpers do more than fire one request and trust the immediate response. They validate the resulting state through follow-up reads or polling so callers can treat success as *confirmed state*, not just *request accepted*.

## Supported verified workflows

- `Workflows.CreateProfilesAndVerify(...)`
- `Workflows.FindProfileByNameVerified(...)`
- `Workflows.StartProfileByName(...)`
- `Workflows.StopProfileByName(...)`
- `Workflows.ImportProfileAndVerify(...)`
- `Workflows.EnableExtensionForProfileByName(...)`
- `Workflows.ExportProfileByNameToFolder(...)`
- `Workflows.GenerateProfileProxyByName(...)`

## What each workflow verifies

### CreateProfilesAndVerify

- creates one or more profiles
- polls `Profiles.GetMetas(...)`
- succeeds only when all returned profile IDs are readable as profile metas

### FindProfileByNameVerified

- resolves one exact-name match through `Profiles.FindByName(...)`
- reads `Profiles.GetMeta(...)`
- verifies that the returned meta still matches the resolved profile name

### StartProfileByName

- resolves and verifies the profile first
- starts it via the launcher
- optionally waits for launcher status to become running

### StopProfileByName

- resolves and verifies the profile first
- stops it via the launcher
- optionally polls launcher status until it reaches a stopped state
- can ignore the known already-stopped condition while still verifying final status

### ImportProfileAndVerify

- starts a launcher import job
- polls import status until it reaches `done`
- reads the imported profile meta by `new_profile_id`

### EnableExtensionForProfileByName

- resolves and verifies the profile first
- enables the extension for that profile
- polls object-to-profile usages until the binding is visible
- can also require a profile-centric usage read when that endpoint is available

### ExportProfileByNameToFolder

- resolves and verifies the profile first
- optionally stops it
- exports it through the archive manager flow
- returns the organized archive result on disk

### GenerateProfileProxyByName

- resolves and verifies the profile first
- generates a managed MLX proxy with typed affinity metadata
- optionally patches the verified profile with that proxy configuration

## Example

```go
ctx := context.Background()

created, err := client.Workflows.CreateProfilesAndVerify(ctx, &mlx.CreateProfileRequest{
    Name:        "Demo",
    BrowserType: "mimic",
    FolderID:    folderID,
    OSType:      "windows",
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

started, err := client.Workflows.StartProfileByName(ctx, "Demo", mlx.StartProfileByNameOptions{
    WaitForRunning: true,
    PollOptions: mlx.PollOptions{
        InitialInterval: 500 * time.Millisecond,
        MaxInterval:     2 * time.Second,
        Timeout:         30 * time.Second,
    },
})
if err != nil {
    return err
}

fmt.Println("created profiles:", len(created.Profiles))
fmt.Println("running profile id:", started.Profile.ID)
```

## Notes

- workflow lookup defaults to `storage_type=local` unless caller overrides `FindOptions`
- profile verification uses follow-up metadata reads instead of trusting the initial search result alone
- post-condition polling respects context cancellation and timeout settings
- extension verification should prefer object-centric usage checks for critical flows because profile-centric usage reads may be unreliable in some live environments
- local/cloud decisions should rely on confirmed signals such as search behavior and `parameters.storage.is_local`, not the raw top-level `is_local` metadata field alone
