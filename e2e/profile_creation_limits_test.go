//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	. "github.com/bath0ry/mlx-go-sdk"
)

const (
	envRunCreationLimitSpike = "MLX_RUN_CREATION_LIMIT_SPIKE"
	envRunCreateFiftySpike   = "MLX_RUN_CREATE_50_SPIKE"
	envE2EProfileCap         = "MLX_E2E_PROFILE_CAP"
	defaultE2EProfileCap     = 50
	creationLimitSpikePrefix = "mlx-go-sdk-limit-spike-"
	createFiftySpikePrefix   = "mlx-go-sdk-create50-spike-"
)

type batchCreateProbeResult struct {
	Times      int
	CreatedIDs int
	Observed   int
	Duration   time.Duration
	StatusCode int
	Err        string
}

type batchCreateLimitReport struct {
	ProfileCap     int
	ActiveProfiles int
	TrashProfiles  int
	FreeSlots      int
	Probes         []batchCreateProbeResult
	MaxSucceeded   int
}

type rateLimitProbeReport struct {
	Attempted    int
	Succeeded    int
	First429At   int
	RetryAfter   time.Duration
	StatusCode   int
	Elapsed      time.Duration
	LastError    string
	StartedAtUTC string
}

type createFiftyCadenceAttemptReport struct {
	Interval         time.Duration
	BatchSizes       []int
	RequestsSent     int
	CreatedIDs       int
	ObservedProfiles int
	FailureRequest   int
	Duration         time.Duration
	StatusCode       int
	Err              string
}

func TestE2ECreateFiftyProfilesCadence(t *testing.T) {
	if os.Getenv(EnvRunE2E) != "1" {
		t.Skipf("set %s=1 to run E2E tests", EnvRunE2E)
	}
	if os.Getenv(envRunCreateFiftySpike) != "1" {
		t.Skipf("set %s=1 to run the live create-50 cadence spike", envRunCreateFiftySpike)
	}
	if skipForRateLimit(t) {
		return
	}

	client, err := NewFromEnv(WithTimeout(60 * time.Second))
	if err != nil {
		t.Fatalf("NewFromEnv returned error: %v", err)
	}

	ctx := context.Background()
	folderID := resolveE2EFolderID(t, client)
	profileCap := resolveE2EProfileCap(t)
	activeCount := countProfiles(t, client, false)
	trashCount := countProfiles(t, client, true)
	freeSlots := availableProfileSlots(profileCap, activeCount, trashCount)
	if freeSlots < 50 {
		t.Skipf("create-50 spike needs at least 50 free slots; cap=%d active=%d trash=%d free=%d", profileCap, activeCount, trashCount, freeSlots)
	}

	intervals := createFiftyIntervalCandidates()
	reports := make([]createFiftyCadenceAttemptReport, 0, len(intervals))
	succeededAt := time.Duration(-1)

	for _, interval := range intervals {
		waitForFreshMinuteWindow(t, fmt.Sprintf("create-50 cadence attempt interval=%s", interval))
		purgeProfilesByPrefix(t, client, createFiftySpikePrefix)

		report := runCreateFiftyCadenceAttempt(t, ctx, client, folderID, interval)
		reports = append(reports, report)
		if report.Err == "" && report.CreatedIDs == 50 && report.ObservedProfiles == 50 {
			succeededAt = interval
			break
		}
	}

	t.Logf("create-50 cadence report: succeeded_at=%s theoretical_rpm_50_interval=%s attempts=%#v", succeededAt, time.Minute/50, reports)
	if succeededAt < 0 {
		t.Fatalf("failed to create 50 profiles with tested intervals; attempts=%#v", reports)
	}
}

func TestE2EProfileCreationLimits(t *testing.T) {
	if os.Getenv(EnvRunE2E) != "1" {
		t.Skipf("set %s=1 to run E2E tests", EnvRunE2E)
	}
	if os.Getenv(envRunCreationLimitSpike) != "1" {
		t.Skipf("set %s=1 to run the live profile-creation limit spike", envRunCreationLimitSpike)
	}
	if skipForRateLimit(t) {
		return
	}

	client, err := NewFromEnv(WithTimeout(60 * time.Second))
	if err != nil {
		t.Fatalf("NewFromEnv returned error: %v", err)
	}

	ctx := context.Background()
	folderID := resolveE2EFolderID(t, client)
	waitForFreshMinuteWindow(t, "batch create probe")
	purgeProfilesByPrefix(t, client, creationLimitSpikePrefix)

	profileCap := resolveE2EProfileCap(t)
	activeCount := countProfiles(t, client, false)
	trashCount := countProfiles(t, client, true)
	freeSlots := availableProfileSlots(profileCap, activeCount, trashCount)
	if freeSlots < 11 {
		t.Skipf("creation-limit spike needs at least 11 free slots to distinguish documented 10-at-once behavior from capacity pressure; cap=%d active=%d trash=%d free=%d", profileCap, activeCount, trashCount, freeSlots)
	}

	batchReport := probeBatchCreateLimit(t, ctx, client, folderID, profileCap, activeCount, trashCount)
	waitForFreshMinuteWindow(t, "read-only rate-limit burst")
	rateReport := probeReadOnlyRateLimit(t, client, 70)

	t.Logf("batch create report: cap=%d active=%d trash=%d free=%d max_succeeded=%d probes=%#v", batchReport.ProfileCap, batchReport.ActiveProfiles, batchReport.TrashProfiles, batchReport.FreeSlots, batchReport.MaxSucceeded, batchReport.Probes)
	t.Logf("read-only rate-limit report: attempted=%d succeeded=%d first_429_at=%d status=%d retry_after=%s started_at_utc=%s err=%q elapsed=%s", rateReport.Attempted, rateReport.Succeeded, rateReport.First429At, rateReport.StatusCode, rateReport.RetryAfter, rateReport.StartedAtUTC, rateReport.LastError, rateReport.Elapsed)
}

func resolveE2EProfileCap(t *testing.T) int {
	t.Helper()

	raw := strings.TrimSpace(os.Getenv(envE2EProfileCap))
	if raw == "" {
		return defaultE2EProfileCap
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		t.Fatalf("%s must be a positive integer, got %q", envE2EProfileCap, raw)
	}
	return value
}

func probeBatchCreateLimit(t *testing.T, ctx context.Context, client *Client, folderID string, profileCap, activeCount, trashCount int) batchCreateLimitReport {
	t.Helper()

	report := batchCreateLimitReport{
		ProfileCap:     profileCap,
		ActiveProfiles: activeCount,
		TrashProfiles:  trashCount,
		FreeSlots:      availableProfileSlots(profileCap, activeCount, trashCount),
	}

	candidates := initialBatchProbeCandidates(report.FreeSlots)
	if len(candidates) == 0 {
		return report
	}

	lastSuccess := 0
	firstFailure := 0
	seen := make(map[int]struct{})

	for _, candidate := range candidates {
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}

		probe := runBatchCreateProbe(t, ctx, client, folderID, candidate)
		report.Probes = append(report.Probes, probe)
		if probe.Err == "" && probe.CreatedIDs == candidate && probe.Observed == candidate {
			lastSuccess = candidate
			report.MaxSucceeded = candidate
			continue
		}
		firstFailure = candidate
		break
	}

	if firstFailure == 0 {
		return report
	}

	low := lastSuccess + 1
	high := firstFailure - 1
	for low <= high {
		mid := low + (high-low)/2
		if _, ok := seen[mid]; ok {
			break
		}
		seen[mid] = struct{}{}

		probe := runBatchCreateProbe(t, ctx, client, folderID, mid)
		report.Probes = append(report.Probes, probe)
		if probe.Err == "" && probe.CreatedIDs == mid && probe.Observed == mid {
			lastSuccess = mid
			report.MaxSucceeded = mid
			low = mid + 1
			continue
		}
		high = mid - 1
	}

	slices.SortFunc(report.Probes, func(a, b batchCreateProbeResult) int {
		return a.Times - b.Times
	})
	return report
}

func runBatchCreateProbe(t *testing.T, ctx context.Context, client *Client, folderID string, times int) batchCreateProbeResult {
	t.Helper()

	probeName := fmt.Sprintf("%s%s-t%d", creationLimitSpikePrefix, time.Now().UTC().Format("20060102-150405.000"), times)
	start := time.Now()
	req := newE2ECreateProfileRequest(probeName, folderID)
	req.Times = times

	result := batchCreateProbeResult{Times: times}

	createResp, _, err := client.Profiles.Create(ctx, req)
	result.Duration = time.Since(start)
	result.StatusCode = errorStatusCode(err)
	if err != nil {
		result.Err = err.Error()
	}
	if createResp != nil {
		result.CreatedIDs = len(createResp.Data.IDs)
	}

	observedProfiles := waitForProbeProfiles(t, client, probeName, result.CreatedIDs)
	result.Observed = len(observedProfiles)
	purgeProfilesByPrefix(t, client, probeName)
	return result
}

func waitForProbeProfiles(t *testing.T, client *Client, probeName string, expected int) []Profile {
	t.Helper()

	if expected == 0 {
		profiles, err := searchProfilesByPrefix(client, probeName, false)
		if err != nil {
			skipOrFatalRateLimit(t, err, "Profiles.Search after create probe %q returned error: %v", probeName, err)
		}
		return profiles
	}

	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		profiles, err := searchProfilesByPrefix(client, probeName, false)
		if err == nil && len(profiles) >= expected {
			return profiles
		}
		if err != nil && isRateLimited(t, err) {
			return nil
		}
		time.Sleep(2 * time.Second)
	}

	t.Fatalf("probe %q did not surface %d created profiles before timeout", probeName, expected)
	return nil
}

func searchProfilesByPrefix(client *Client, prefix string, removed bool) ([]Profile, error) {
	resp, _, err := client.Profiles.Search(context.Background(), &SearchProfilesRequest{
		IsRemoved:   removed,
		Limit:       100,
		Offset:      0,
		SearchText:  prefix,
		StorageType: "all",
		OrderBy:     "updated_at",
		Sort:        "desc",
	})
	if err != nil {
		return nil, err
	}

	lowerPrefix := strings.ToLower(prefix)
	out := make([]Profile, 0)
	for _, profile := range resp.Data.Profiles {
		if strings.HasPrefix(strings.ToLower(profile.Name), lowerPrefix) {
			out = append(out, profile)
		}
	}
	return out, nil
}

func purgeProfilesByPrefix(t *testing.T, client *Client, prefix string) {
	t.Helper()

	var ids []string
	for _, removed := range []bool{false, true} {
		profiles, err := searchProfilesByPrefix(client, prefix, removed)
		if err != nil {
			skipOrFatalRateLimit(t, err, "Profiles.Search cleanup for %q returned error: %v", prefix, err)
		}
		for _, profile := range profiles {
			ids = append(ids, profile.ID)
		}
	}
	ids = uniqueStrings(ids)
	if len(ids) == 0 {
		return
	}

	if _, _, err := client.Profiles.Delete(context.Background(), &DeleteProfilesRequest{
		IDs:         ids,
		Permanently: true,
	}); err != nil {
		skipOrFatalRateLimit(t, err, "Profiles.Delete permanent cleanup for %q returned error: %v", prefix, err)
	}

	waitForNoProfilesByPrefix(t, client, prefix, false)
	waitForNoProfilesByPrefix(t, client, prefix, true)
}

func waitForNoProfilesByPrefix(t *testing.T, client *Client, prefix string, removed bool) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		profiles, err := searchProfilesByPrefix(client, prefix, removed)
		if err == nil && len(profiles) == 0 {
			return
		}
		if err != nil && isRateLimited(t, err) {
			return
		}
		time.Sleep(2 * time.Second)
	}

	t.Fatalf("profiles with prefix %q (removed=%t) still existed after permanent cleanup", prefix, removed)
}

func probeReadOnlyRateLimit(t *testing.T, client *Client, attempts int) rateLimitProbeReport {
	t.Helper()

	report := rateLimitProbeReport{
		Attempted:    attempts,
		StartedAtUTC: time.Now().UTC().Format(time.RFC3339),
	}
	start := time.Now()
	for i := 1; i <= attempts; i++ {
		_, _, err := client.Folders.List(context.Background())
		if err == nil {
			report.Succeeded++
			continue
		}
		report.LastError = err.Error()
		report.StatusCode = errorStatusCode(err)
		if IsRateLimitedError(err) {
			report.First429At = i
			report.RetryAfter = RetryAfter(err)
			break
		}
		t.Fatalf("read-only rate-limit probe failed with a non-429 error on request %d: %v", i, err)
	}
	report.Elapsed = time.Since(start)
	return report
}

func runCreateFiftyCadenceAttempt(t *testing.T, ctx context.Context, client *Client, folderID string, interval time.Duration) createFiftyCadenceAttemptReport {
	t.Helper()

	report := createFiftyCadenceAttemptReport{
		Interval:   interval,
		BatchSizes: splitIntoCreateBatchSizes(50, 10),
	}
	prefix := fmt.Sprintf("%s%s-i%dms", createFiftySpikePrefix, time.Now().UTC().Format("20060102-150405.000"), interval.Milliseconds())
	start := time.Now()

	defer func() {
		purgeProfilesByPrefix(t, client, prefix)
	}()

	for i, batchSize := range report.BatchSizes {
		if i > 0 && interval > 0 {
			time.Sleep(interval)
		}

		req := newE2ECreateProfileRequest(fmt.Sprintf("%s-r%d", prefix, i+1), folderID)
		req.Times = batchSize

		createResp, _, err := client.Profiles.Create(ctx, req)
		report.RequestsSent++
		report.StatusCode = errorStatusCode(err)
		if createResp != nil {
			report.CreatedIDs += len(createResp.Data.IDs)
		}
		if err != nil {
			report.FailureRequest = i + 1
			report.Err = err.Error()
			break
		}
	}

	observedProfiles := waitForProbeProfiles(t, client, prefix, report.CreatedIDs)
	report.ObservedProfiles = len(observedProfiles)
	report.Duration = time.Since(start)
	return report
}

func waitForFreshMinuteWindow(t *testing.T, reason string) {
	t.Helper()

	now := time.Now()
	next := now.Truncate(time.Minute).Add(time.Minute).Add(1500 * time.Millisecond)
	wait := time.Until(next)
	if wait <= 0 {
		return
	}
	t.Logf("waiting %s for a fresh minute window before %s", wait.Round(time.Second), reason)
	time.Sleep(wait)
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func errorStatusCode(err error) int {
	if err == nil {
		return 0
	}
	var apiErr *ErrorResponse
	if errors.As(err, &apiErr) && apiErr != nil && apiErr.Response != nil {
		return apiErr.Response.StatusCode
	}
	return 0
}
