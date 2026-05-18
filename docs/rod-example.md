# Rod integration example

Use Rod to attach to a Multilogin X profile that was launched by the local launcher through the SDK helper path.

> Important: always call `NoDefaultDevice()` before connecting. Rod applies a default device profile unless you disable it, and that extra emulation can distort the Multilogin fingerprint.

`AutomationRod` is a semantic alias backed by launcher `playwright`. The workflow helper requests `playwright` from the launcher, waits for the profile to report running, and returns the Rod-compatible control URL from the started profile response.

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

    profileName := "your-profile-name"

    started, err := client.Workflows.StartProfileAutomationByName(ctx, profileName, mlx.StartProfileAutomationByNameOptions{
        FindOptions: &mlx.FindProfileOptions{
            FolderID:    folderID,
            StorageType: "all",
        },
        StartOptions: mlx.StartProfileOptions{
            AutomationType: mlx.AutomationRod,
        },
        WaitForRunning: true,
    })
    if err != nil {
        log.Fatalf("start profile: %v", err)
    }
    defer func() {
        _, err := client.Workflows.StopProfileByName(ctx, profileName, mlx.StopProfileByNameOptions{
            WaitForStopped: true,
        })
        if err != nil {
            log.Printf("stop profile: %v", err)
        }
    }()

    browser := rod.New().
        ControlURL(started.RodControlURL).
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

- `client.Workflows.StartProfileAutomationByName(..., WaitForRunning: true)` waits for the profile to be ready before exposing the Rod control URL.
- `started.RodControlURL` is the full WebSocket debugger URL Rod needs.
- `NoDefaultDevice()` is required to avoid Rod's built-in device emulation altering the Multilogin browser fingerprint.
- `AutomationRod` requests the Rod semantic alias while the SDK normalizes the launcher call to `playwright`.
- Close the Rod page when finished, then stop the Multilogin profile through the SDK.
