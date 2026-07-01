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

## Proxy continuity

Proxy continuity is a launch-time safeguard that verifies a profile's proxy is healthy *before* the profile is started, and transparently rotates to a geography-preserving replacement when the current proxy is dead or too slow. It backs the CLI's `mlx launcher start` and `mlx profile create --start` flows (via `Proxies.EnsureHealthyProxy(...)`).

### Fail-closed

The check is **fail-closed**: if proxy continuity is enabled and no healthy proxy can be confirmed — or any step of the check errors (meta lookup, generation, health probe) — the launch is aborted and the profile is **not** started. A profile is never launched on an unhealthy or unreachable proxy. The only way to bypass the check is the explicit `--skip-proxy-check` flag (see the DANGER note below).

### Two-tier latency thresholds

Health is judged against two latency tiers (defaults shown):

- **Threshold — `2000ms`**: the preferred ceiling. The current proxy is kept as-is if it is alive and at or under the threshold, and tier-1 selection prefers the fastest candidate at or under it.
- **Hard cap — `3000ms`**: the escalation ceiling. If no candidate meets the threshold, the fastest alive candidate at or under the hard cap is chosen. If nothing is alive within the hard cap, the launch fails.

Both are configurable and can be overridden per-launch with `--proxy-threshold-ms` / `--proxy-hard-cap-ms`.

### Selection order: keep-healthy-current, then city → country

1. **Current proxy first** — if the profile's existing proxy is alive and at or under the threshold, it is kept unchanged (no rotation, no patch).
2. **Same city** — if the current proxy fails and a `city` affinity is known, generate `candidates_per_round` replacements preserving `country`/`region`/`city`.
3. **Same country** — if the city round yields nothing acceptable (or no city is known), generate replacements preserving `country`/`region` only.

The best candidate across all rounds is chosen under the two-tier rule.

### Proxy-only patch

When a replacement proxy is selected (i.e. the current one was not kept), the CLI applies a **proxy-only patch** to the profile (`PatchProfileRequest{ProfileID, Proxy}`) before launching. No other profile fields are touched. If the current proxy was healthy, no patch is made.

### `--skip-proxy-check` (DANGER)

`--skip-proxy-check` bypasses the entire continuity check. The profile launches even if its proxy is dead, unreachable, or slower than the hard cap. This defeats the fail-closed guarantee — use it only when you knowingly want to launch without proxy validation.

### Config: `defaults.proxy_continuity`

```json
{
  "defaults": {
    "proxy": {
      "proxy_continuity": {
        "enabled": true,
        "latency_threshold_ms": 2000,
        "latency_hard_cap_ms": 3000,
        "candidates_per_round": 5,
        "check_targets": ["https://www.google.com", "https://www.facebook.com", "https://medium.com"],
        "check_timeout": "10s"
      }
    }
  }
}
```

- `enabled` — turn the launch-time check on/off. When disabled, `--start` and `launcher start` launch without any proxy validation.
- `latency_threshold_ms` / `latency_hard_cap_ms` — the two-tier ceilings described above.
- `candidates_per_round` — how many replacement proxies to generate per geo round.
- `check_targets` — the browser-common URLs probed to measure proxy health.
- `check_timeout` — per-target probe timeout.
