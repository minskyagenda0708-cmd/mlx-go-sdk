# Batch helpers

The workflow layer includes batch helpers for common multi-profile operations with explicit per-item results and aggregated failure reporting.

## Supported helpers

- `Workflows.StartProfilesByName(...)`
- `Workflows.StopProfilesByName(...)`
- `Workflows.ExportProfilesByNameToFolder(...)`
- `Workflows.EnableExtensionForProfilesByName(...)`

These helpers process profile names in order and return a `BatchResult[T]`.

## Result model

Each batch call returns:

- `Summary.Total`
- `Summary.Succeeded`
- `Summary.Failed`
- ordered `Items` with one entry per requested profile name

Each item contains:

- `ProfileName`
- `Result` when the operation succeeded
- `Err` when that specific profile failed

If one or more items fail, the helper still returns the full `BatchResult`, plus a `BatchProfileOperationError` that aggregates all failed items.

## Error aggregation behavior

Batch helpers are designed for consumer tools that need both:

- a single error to signal partial failure
- full per-profile detail for logs, retries, or reporting

`BatchProfileOperationError`:

- keeps the operation label such as `start` or `export`
- stores all failed profile names and underlying errors
- unwraps underlying errors so `errors.Is(...)` and `errors.As(...)` still work

## Example

```go
result, err := client.Workflows.StartProfilesByName(ctx, []string{
    "Account A",
    "Account B",
    "Account C",
}, mlx.StartProfileByNameOptions{
    WaitForRunning: true,
})
if err != nil {
    var batchErr *mlx.BatchProfileOperationError
    if errors.As(err, &batchErr) {
        for _, failure := range batchErr.Failures {
            log.Printf("start failed for %s: %v", failure.ProfileName, failure.Err)
        }
    }
}

for _, item := range result.Items {
    if item.Err != nil {
        continue
    }
    log.Printf("started %s on port %s", item.ProfileName, item.Result.StartResponse.Data.Port)
}
```

## Notes

- empty `profileNames` input is rejected
- empty individual profile names are captured as item failures
- processing is sequential and stable, which makes CLI output and retry planning easier
- callers can retry only failed items by inspecting `result.Items` or `result.Failures()`
