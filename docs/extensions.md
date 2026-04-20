# Extension object workflows

Multilogin X exposes browser extensions as resource objects with object type `6811b909-2e4b-45db-ab62-f14f515523cf`.

## Supported SDK flows

- `Resources.ListExtensions(...)` lists extension resource metadata.
- `Resources.UploadExtension(...)` uploads a local archive through `/api/v1/object_storage/upload`.
- `Resources.LocalToCloud(...)` promotes a local object through `/api/v1/object_storage/local_to_cloud`.
- `Resources.CreateExtensionFromURL(...)` asks the launcher to fetch an extension package from a URL.
- `Resources.CreateExtensionFromChromeWebStore(...)` builds a Chrome Web Store CRX download URL and forwards it to `CreateExtensionFromURL`.
- `Resources.EnableExtensionForProfiles(...)` and `Resources.DisableExtensionForProfiles(...)` manage attachment to profiles.
- `Resources.ProfileExtensionUsages(...)` reads extension associations for one profile.

## Live notes

- For a local profile, attaching an extension from a local zip worked reliably only when the extension object reference was cloud-backed. Because of that, `UploadExtension` and `CreateExtensionFromURL` default `storage_type` to `cloud`.
- Chrome Web Store IDs can be mapped to the canonical CRX update endpoint, but current live validation against the desktop launcher still returned `500 error on downloading extension: failed to fetch extension, status: 404` for a known public extension. Treat Chrome Web Store ingestion as a best-effort helper until the launcher fetch path is fixed or documented.

## Example

```go
ctx := context.Background()

uploaded, _, err := client.Resources.UploadExtension(ctx, &mlx.UploadExtensionRequest{
    ObjectPath: `C:\extensions\demo.zip`,
})
if err != nil {
    return err
}

_, _, err = client.Resources.EnableExtensionForProfiles(ctx, uploaded.Data.MetaID, &mlx.SetResourceProfilesRequest{
    ProfileIDs: []string{profileID},
})
if err != nil {
    return err
}

usages, _, err := client.Resources.ProfileExtensionUsages(ctx, profileID)
if err != nil {
    return err
}

fmt.Println("enabled extensions:", len(usages.Data))
```