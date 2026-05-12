# Profile creation limits

This repository includes an opt-in live spike for investigating Multilogin X profile creation limits against a real account.

The spike is intentionally destructive for its own test artifacts:

- it creates throwaway profiles with a unique `mlx-go-sdk-limit-spike-...` prefix
- it permanently deletes every spike-created profile with `Permanently: true`
- it verifies that both active profiles and trash-bin profiles return to zero for the spike prefix

This avoids the common MLX trap where soft-deleted profiles still count against the subscription cap.

## Where the spike lives

- live test: `e2e/TestE2EProfileCreationLimits`
- live test: `e2e/TestE2ECreateFiftyProfilesCadence`
- helper tests:
  - `TestAvailableProfileSlotsUsesActiveAndTrash`
  - `TestInitialBatchProbeCandidatesIncludeDocumentedBoundary`
  - `TestSplitIntoCreateBatchSizesBuildsTenSizedPlan`
  - `TestCreateFiftyIntervalCandidatesStartAtZeroAndIncrease`

The helper tests pin the two core assumptions used by the live spike:

- free slots are calculated as `profile_cap - active - trash`
- the first batch probe must check the documented `10` / `11` boundary

## How to run it

Run the spike explicitly:

```text
MLX_RUN_E2E=1 MLX_RUN_CREATION_LIMIT_SPIKE=1 MLX_E2E_PROFILE_CAP=50 go test -tags=e2e ./e2e -run TestE2EProfileCreationLimits -count=1 -v
```

Required environment:

- `MLX_TOKEN`

Additional spike-specific environment:

- `MLX_RUN_CREATION_LIMIT_SPIKE=1`
- `MLX_RUN_CREATE_50_SPIKE=1`
- `MLX_E2E_PROFILE_CAP=50`

Optional:

- `MLX_E2E_FOLDER_ID`
- `MLX_BASE_URL`

## What the spike measures

### 1. Batch profile creation limit

The spike probes `POST /profile/create` through `CreateProfileRequest.Times`.

It:

- waits for a fresh minute window
- creates profiles in controlled batches
- verifies the returned ID count and observed search results
- permanently deletes the created batch before the next probe

### 2. Read-only API burst behavior

The spike also runs a separate read-only burst against `Folders.List` to see when `429` appears for the current token/account combination.

This is intentionally separated from the create probe so profile capacity does not distort the RPM measurement.

### 3. Create-50 cadence behavior

The repository also includes a dedicated live spike for creating exactly `50` profiles as repeated `times=10` requests:

```text
MLX_RUN_E2E=1 MLX_RUN_CREATE_50_SPIKE=1 MLX_E2E_PROFILE_CAP=50 go test -tags=e2e ./e2e -run TestE2ECreateFiftyProfilesCadence -count=1 -v
```

It:

- waits for a fresh minute window before each interval candidate
- sends `5` create requests with batch sizes `[10, 10, 10, 10, 10]`
- starts from `0s` inter-request delay and only increases the delay if needed
- permanently deletes all created profiles after each attempt

## Live findings

### 2026-05-12

Environment during the run:

- account state before probe: `active=0`, `trash=0`
- configured profile cap: `50`

Observed batch-create behavior:

- `times=1` succeeded
- `times=10` succeeded
- `times=11` failed with HTTP `400`
- API error message:
  - `invalid profile times value. Should be from 1 to 10`

Observed read-only burst behavior:

- `70` consecutive `Folders.List` requests completed in about `21s`
- no `429` was returned during that burst

Observed create-50 cadence behavior:

- `50` profiles were created successfully as `5` requests of `10`
- the successful attempt used `0s` delay between requests
- the full create-and-verify attempt completed in about `2.7s`
- permanent cleanup returned the account to `active=0`, `trash=0`

Interpretation:

- the create endpoint currently enforces a hard `times` range of `1..10`
- for this token/account on 2026-05-12, no inter-request delay was needed to create `50` profiles as `5 x 10`
- the effective request-rate behavior for this token was not reproduced as `50 RPM` by a `Folders.List` burst on 2026-05-12
- if a `50 RPM` contract exists for the subscription, it may be:
  - token-dependent
  - endpoint-dependent
  - enforced on a different time window than this spike assumed

## Cleanup rule

For any future live profile-capacity spike:

- never rely on soft delete
- always delete the test profiles with `Permanently: true`
- remember that `active + trash` consumes the profile quota
