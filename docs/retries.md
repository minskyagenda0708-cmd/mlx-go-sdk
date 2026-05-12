# Retry and error classification

The SDK includes opt-in retry/backoff helpers for transient transport and MLX API failures.

## What is retried

Automatic retries apply only when a client is configured with `WithRetry(...)`, the request uses an idempotent HTTP method, and the failure is classified as retryable:

- transient transport errors
- timeouts
- `429` rate-limit responses
- transient `5xx` MLX or launcher responses

Non-idempotent requests such as `POST` are not retried automatically, which avoids replaying create/import/attach-style mutations after transient failures.

Non-retryable request errors such as `400`, `401`, `403`, and most logical conflicts are returned immediately.

## Error helpers

The SDK exposes typed helpers for callers that need production-grade worker or CLI behavior:

- `ClassifyError(err)`
- `IsRetryableError(err)`
- `IsTemporaryError(err)`
- `IsRateLimitedError(err)`
- `RetryAfter(err)`

Typed classes include:

- `timeout`
- `network`
- `rate_limited`
- `unauthorized`
- `forbidden`
- `not_found`
- `conflict`
- `invalid_request`
- `server`

## Example

```go
client, err := mlx.NewFromEnv(
    mlx.WithRetry(mlx.RetryOptions{
        MaxAttempts:     4,
        InitialInterval: 500 * time.Millisecond,
        MaxInterval:     5 * time.Second,
        Multiplier:      2,
        Jitter:          0.2,
    }),
)
if err != nil {
    return err
}

folders, _, err := client.Folders.List(ctx)
if err != nil {
    if mlx.IsRateLimitedError(err) {
        fmt.Println("retry after:", mlx.RetryAfter(err))
    }
    return err
}

fmt.Println(len(folders.Data.Folders))
```

## Notes

- retries are request-scoped and respect context cancellation
- retries are disabled by default; callers must opt in with `WithRetry(...)`
- `Retry-After` is honored when present on MLX responses
- for this repository, run tests from the root module so nested `GithubExamples` fixtures are not picked up accidentally
