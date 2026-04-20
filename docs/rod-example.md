# Rod integration example

Use Rod to attach to a Multilogin X profile that was launched by the local launcher.

> Important: always call `NoDefaultDevice()` before connecting. Rod applies a default device profile unless you disable it, and that extra emulation can distort the Multilogin fingerprint.

> Live API note: in current real-world testing, `automation_type=rod` may start the profile successfully but still return an empty `port`. A practical fallback is to launch with `automation_type=playwright` and then attach Rod to the returned DevTools endpoint.

## Example

```go
package main

import (
    "context"
    "fmt"
    "log"

    mlx "mlx-go-sdk"

    "github.com/go-rod/rod"
    rodlauncher "github.com/go-rod/rod/lib/launcher"
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
    if started.Data.Port == "" {
        // Current live launcher behavior may omit the port for automation_type=rod.
        started, _, err = client.Launcher.Start(ctx, folderID, profileID, mlx.StartProfileOptions{
            AutomationType: mlx.AutomationPlaywright,
        })
        if err != nil {
            log.Fatalf("start profile with playwright fallback: %v", err)
        }
    }
    defer func() {
        _, _, _ = client.Launcher.Stop(ctx, profileID)
    }()

    controlURL, err := rodlauncher.ResolveURL(started.Data.Port)
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

- `started.Data.Port` is the local DevTools port returned by Multilogin X.
- `rodlauncher.ResolveURL(...)` converts that port into the full WebSocket debugger URL Rod needs.
- `NoDefaultDevice()` is required to avoid Rod's built-in device emulation altering the Multilogin browser fingerprint.
- In current live testing, `AutomationRod` may not return a port yet. Falling back to `AutomationPlaywright` still gives Rod a valid DevTools endpoint while preserving the `NoDefaultDevice()` safeguard on the Rod side.
- Close the Rod page when finished, then stop the Multilogin profile through the SDK.
