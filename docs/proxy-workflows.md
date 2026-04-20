# MLX proxy workflows

The SDK exposes Multilogin X managed proxy helpers through `client.Proxies` and a high-level assignment workflow through `client.Workflows.GenerateProfileProxyByName(...)`.

## Supported flows

- `Proxies.Generate(...)` requests one or more MLX-managed proxy connection strings from `https://profile-proxy.multilogin.com/v1/proxy/connection_url`.
- `Proxies.GetUsage(...)` reads proxy usage data from `https://profile-proxy.multilogin.com/v1/user`.
- `Proxies.ParseConnectionString(...)` and `BuildProfileProxyFromGenerated(...)` turn returned credentials into a typed profile-bound `Proxy` payload.
- `Proxies.GenerateProfileProxy(...)` combines usage lookup, SOCKS5-first generation, and payload conversion.
- `Workflows.GenerateProfileProxyByName(...)` resolves a profile by name, generates a managed proxy with affinity fields, and can patch the profile in one call.

## Affinity-oriented fields

MLX proxy usernames encode stable geography and session hints. The SDK parses and preserves these fields:

- `country`
- `region`
- `city`
- `session_id`

That makes it easier to keep a stable browser fingerprint and stable geography for repeated account sessions.

## SOCKS5 recommendation

Prefer `socks5` for real automation usage.

Live validation on 2026-04-20 confirmed that SOCKS5 generation works with affinity fields like `country=us`, `region=new_jersey`, and `city=east_brunswick`. The generated connection string was returned successfully and parsed into a profile-ready proxy payload.

HTTP proxy mode is still exposed because the API supports it, but it should be treated as best-effort. The issue scope explicitly tracks a known live HTTP proxy bug, so the workflow helper defaults to SOCKS5 whenever protocol is not set and `PreferSOCKS5` is enabled.

## Observed retention behavior

Live probing against the proxy API showed:

- repeated sticky requests with identical `country`, `region`, and `city` values reused the same account-level prefix and password secret
- the generated `sid` value changed between requests
- because of that, callers should not assume a second sticky request will reproduce the exact same full connection string

In practice, the SDK exposes both the raw connection string and parsed `session_id` / retention metadata so callers can decide whether to reuse existing credentials or request a fresh proxy.

## Example

```go
ctx := context.Background()

result, err := client.Workflows.GenerateProfileProxyByName(ctx, "Demo", mlx.GenerateProfileProxyByNameOptions{
    PatchProfile: true,
    GenerateOptions: mlx.GenerateProfileProxyRequest{
        GenerateProxyRequest: mlx.GenerateProxyRequest{
            Country:     "us",
            Region:      "new_jersey",
            City:        "east_brunswick",
            SessionType: mlx.ProxySessionSticky,
        },
        PreferSOCKS5: true,
        SaveTraffic:  true,
    },
})
if err != nil {
    return err
}

fmt.Println(result.ProfileProxy.Type)
fmt.Println(result.Connection.Country, result.Connection.Region, result.Connection.City)
```
