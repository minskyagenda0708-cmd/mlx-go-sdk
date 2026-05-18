# Rod integration example

Use Rod to attach to a Multilogin X profile that was launched by the local launcher through the SDK helper path.

> Important: always call `NoDefaultDevice()` before connecting. Rod applies a default device profile unless you disable it, and that extra emulation can distort the Multilogin fingerprint.

`AutomationRod` is a semantic alias backed by launcher `playwright`. The SDK requests `playwright` from the launcher, then resolves the DevTools endpoint for Rod through the helper methods on the started profile response.

## Example

```go
package main

import (
    "context"
    "fmt"
    "log"

    mlx "mlx-go-sdk"

    "github.com/go-rod/rod"
)

func main() {
    ctx := context.Background()

    client, err := mlx.NewFromEnv()
    if err != nil {
        log.Fatalf("create client: %v", err)
    }

    folderID := "your-folder-id"
    profileID := "your-profile-id"

    started, _, err := client.Launcher.Start(ctx, folderID, profileID, mlx.StartProfileOptions{
        AutomationType: mlx.AutomationRod,
    })
    if err != nil {
        log.Fatalf("start profile: %v", err)
    }
    defer func() {
        _, _, _ = client.Launcher.Stop(ctx, profileID)
    }()

    controlURL, err := started.Data.ResolveRodControlURL(ctx)
    if err != nil {
        log.Fatalf("resolve rod control url: %v", err)
    }

    browser := rod.New().
        ControlURL(controlURL).
        NoDefaultDevice().
        MustConnect()

    page := browser.MustPage("")
    defer page.MustClose()

    page.MustNavigate("https://example.com")
    page.MustWaitLoad()

    title := page.MustEval(`() => document.title`).String()
    fmt.Println("page title:", title)
}
```

## Notes

- `started.Data.ResolveRodControlURL(ctx)` converts the launcher response into the full WebSocket debugger URL Rod needs.
- `NoDefaultDevice()` is required to avoid Rod's built-in device emulation altering the Multilogin browser fingerprint.
- `AutomationRod` requests the Rod semantic alias while the SDK normalizes the launcher call to `playwright`.
- Close the Rod page when finished, then stop the Multilogin profile through the SDK.
