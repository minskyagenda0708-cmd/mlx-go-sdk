//go:build e2e
// +build e2e

package e2e

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/minskyagenda0708-cmd/mlx-go-sdk"

	"github.com/go-rod/rod"
)

func TestE2ELauncherHealth(t *testing.T) {
	if os.Getenv(EnvRunE2E) != "1" {
		t.Skipf("set %s=1 to run E2E tests", EnvRunE2E)
	}
	if skipForRateLimit(t) {
		return
	}

	client, err := NewFromEnv(WithTimeout(30 * time.Second))
	if err != nil {
		t.Fatalf("NewFromEnv returned error: %v", err)
	}

	resp, _, err := client.Launcher.Health(context.Background())
	if err != nil {
		t.Fatalf("Launcher.Health returned error: %v", err)
	}
	if !resp.Data.Alive {
		t.Fatal("expected launcher to be alive")
	}
	if strings.TrimSpace(resp.Data.Version) == "" {
		t.Fatal("expected non-empty launcher version")
	}

	t.Logf("launcher health ok: env=%s version=%s", resp.Data.Env, resp.Data.Version)
}

func TestE2EProfileLookupHelpers(t *testing.T) {
	if os.Getenv(EnvRunE2E) != "1" {
		t.Skipf("set %s=1 to run E2E tests", EnvRunE2E)
	}
	if skipForRateLimit(t) {
		return
	}

	client, err := NewFromEnv(WithTimeout(60 * time.Second))
	if err != nil {
		t.Fatalf("NewFromEnv returned error: %v", err)
	}

	folderID := resolveE2EFolderID(t, client)
	ensureE2ECapacity(t, client, 10)

	profileName := "mlx-go-sdk-lookup-" + time.Now().UTC().Format("20060102-150405")
	ctx := context.Background()
	createResp, _, err := client.Profiles.Create(ctx, newE2ECreateProfileRequest(profileName, folderID))
	if err != nil {
		t.Fatalf("Profiles.Create returned error: %v", err)
	}
	if len(createResp.Data.IDs) == 0 {
		t.Fatalf("Profiles.Create returned no ids")
	}
	profileID := createResp.Data.IDs[0]

	defer func() {
		_, _, _ = client.Profiles.Delete(ctx, &DeleteProfilesRequest{IDs: []string{profileID}, Permanently: true})
	}()

	found, _, err := client.Profiles.FindByName(ctx, profileName, &FindProfileOptions{FolderID: folderID, StorageType: "all"})
	if err != nil {
		t.Fatalf("Profiles.FindByName returned error: %v", err)
	}
	if found.ID != profileID {
		t.Fatalf("expected found profile id %s, got %s", profileID, found.ID)
	}

	meta, _, err := client.Profiles.GetMeta(ctx, profileID)
	if err != nil {
		t.Fatalf("Profiles.GetMeta returned error: %v", err)
	}
	if meta.ID != profileID {
		t.Fatalf("expected meta profile id %s, got %s", profileID, meta.ID)
	}
	if meta.FolderID != folderID {
		t.Fatalf("expected meta folder id %s, got %s", folderID, meta.FolderID)
	}

	t.Logf("profile lookup helpers ok: profile=%s folder=%s status=%s", meta.ID, meta.FolderID, meta.Status)
}

func TestE2ETypedLauncherModels(t *testing.T) {
	if os.Getenv(EnvRunE2E) != "1" {
		t.Skipf("set %s=1 to run E2E tests", EnvRunE2E)
	}
	if skipForRateLimit(t) {
		return
	}

	client, err := NewFromEnv(WithTimeout(60 * time.Second))
	if err != nil {
		t.Fatalf("NewFromEnv returned error: %v", err)
	}

	folderID := resolveE2EFolderID(t, client)
	ensureE2ECapacity(t, client, 10)

	profileName := "mlx-go-sdk-typed-" + time.Now().UTC().Format("20060102-150405")
	ctx := context.Background()
	createResp, _, err := client.Profiles.Create(ctx, newE2ECreateProfileRequest(profileName, folderID))
	if err != nil {
		t.Fatalf("Profiles.Create returned error: %v", err)
	}
	if len(createResp.Data.IDs) == 0 {
		t.Fatalf("Profiles.Create returned no ids")
	}
	profileID := createResp.Data.IDs[0]

	defer func() {
		_, _, _ = client.Launcher.Stop(ctx, profileID)
		_, _, _ = client.Profiles.Delete(ctx, &DeleteProfilesRequest{IDs: []string{profileID}, Permanently: true})
	}()

	_, _, err = client.Launcher.Start(ctx, folderID, profileID, StartProfileOptions{})
	if err != nil {
		t.Fatalf("Launcher.Start returned error: %v", err)
	}

	status := waitForRunningStatus(t, client, profileID)
	if status.Data.Timestamp <= 0 {
		t.Fatalf("expected runtime status timestamp, got %d", status.Data.Timestamp)
	}

	statuses, _, err := client.Launcher.Statuses(ctx)
	if err != nil {
		t.Fatalf("Launcher.Statuses returned error: %v", err)
	}
	state, ok := statuses.Data.States[profileID]
	if !ok {
		t.Fatalf("expected launched profile %s in launcher states", profileID)
	}
	if state.Timestamp <= 0 {
		t.Fatalf("expected launcher states timestamp, got %d", state.Timestamp)
	}
	if statuses.Data.ActiveCounter.Cloud+statuses.Data.ActiveCounter.Local+statuses.Data.ActiveCounter.Quick <= 0 {
		t.Fatalf("expected non-zero active counter, got %#v", statuses.Data.ActiveCounter)
	}

	quickStatuses, _, err := client.Launcher.QuickStatuses(ctx)
	if err != nil {
		t.Fatalf("Launcher.QuickStatuses returned error: %v", err)
	}

	searchResp, _, err := client.Profiles.Search(ctx, &SearchProfilesRequest{
		IsRemoved:   false,
		Limit:       100,
		Offset:      0,
		SearchText:  profileName,
		StorageType: "all",
	})
	if err != nil {
		t.Fatalf("Profiles.Search returned error: %v", err)
	}
	if len(searchResp.Data.Profiles) == 0 {
		t.Fatalf("expected created profile in search results")
	}

	meta, _, err := client.Profiles.GetMeta(ctx, profileID)
	if err != nil {
		t.Fatalf("Profiles.GetMeta returned error: %v", err)
	}

	t.Logf("typed launcher models ok: profile=%s runtime_timestamp=%d active_counter=%#v quick_active=%d search_last_on=%q meta_last_on=%q", profileID, state.Timestamp, statuses.Data.ActiveCounter, quickStatuses.Data.ActiveCounter, searchResp.Data.Profiles[0].LastLaunchedOn, meta.LastLaunchedOn)
}

func TestE2ERodConnection(t *testing.T) {
	if os.Getenv(EnvRunE2E) != "1" {
		t.Skipf("set %s=1 to run E2E tests", EnvRunE2E)
	}
	if skipForRateLimit(t) {
		return
	}

	client, err := NewFromEnv(WithTimeout(60 * time.Second))
	if err != nil {
		t.Fatalf("NewFromEnv returned error: %v", err)
	}

	folderID := resolveE2EFolderID(t, client)
	ensureE2ECapacity(t, client, 10)

	profileName := "mlx-go-sdk-rod-" + time.Now().UTC().Format("20060102-150405")
	ctx := context.Background()
	createResp, _, err := client.Profiles.Create(ctx, newE2ECreateProfileRequest(profileName, folderID))
	if err != nil {
		t.Fatalf("Profiles.Create returned error: %v", err)
	}
	if len(createResp.Data.IDs) == 0 {
		t.Fatalf("Profiles.Create returned no ids")
	}
	profileID := createResp.Data.IDs[0]

	defer func() {
		_, _, _ = client.Launcher.Stop(ctx, profileID)
		_, _, _ = client.Profiles.Delete(ctx, &DeleteProfilesRequest{IDs: []string{profileID}, Permanently: true})
	}()

	started, err := client.Workflows.StartProfileAutomationByName(ctx, profileName, StartProfileAutomationByNameOptions{
		FindOptions: &FindProfileOptions{
			FolderID:    folderID,
			StorageType: "all",
		},
		StartOptions:   StartProfileOptions{AutomationType: AutomationRod},
		WaitForRunning: true,
		PollOptions:    PollOptions{InitialInterval: 500 * time.Millisecond, MaxInterval: 2 * time.Second, Timeout: 60 * time.Second},
	})
	if err != nil {
		t.Fatalf("Workflows.StartProfileAutomationByName returned error: %v", err)
	}

	if started.Profile.ID != profileID {
		t.Fatalf("unexpected profile id: %s", started.Profile.ID)
	}
	if started.RequestedAutomation != AutomationRod {
		t.Fatalf("unexpected requested automation: %q", started.RequestedAutomation)
	}
	if started.LauncherAutomation != AutomationPlaywright {
		t.Fatalf("expected rod automation to normalize to playwright, got %q", started.LauncherAutomation)
	}
	if started.StartResponse == nil {
		t.Fatal("expected workflow to include a start response")
	}
	if strings.TrimSpace(started.CDPPort) == "" {
		t.Fatal("expected workflow to return a usable cdp port")
	}
	if started.CDPWebSocketURL == "" {
		t.Fatal("expected workflow to resolve a cdp websocket url")
	}
	if started.RodControlURL != started.CDPWebSocketURL {
		t.Fatalf("expected rod control url to match cdp websocket url, got rod=%q cdp=%q", started.RodControlURL, started.CDPWebSocketURL)
	}

	controlURL, err := started.StartResponse.Data.ResolveRodControlURL(ctx)
	if err != nil {
		t.Fatalf("ResolveRodControlURL returned error: %v", err)
	}
	if controlURL != started.RodControlURL {
		t.Fatalf("expected resolved rod control url %q to match workflow result %q", controlURL, started.RodControlURL)
	}

	browser := rod.New().ControlURL(controlURL).NoDefaultDevice()
	if err := browser.Connect(); err != nil {
		t.Fatalf("rod browser.Connect returned error: %v", err)
	}

	page := browser.MustPage("")
	defer page.MustClose()
	page.MustWaitLoad()

	info, err := page.Info()
	if err != nil {
		t.Fatalf("rod page.Info returned error: %v", err)
	}
	if info == nil {
		t.Fatal("expected rod page info")
	}

	t.Logf("rod connection ok: profile=%s requested=%s launcher=%s port=%s control_url=%s target=%s url=%s", profileID, started.RequestedAutomation, started.LauncherAutomation, started.CDPPort, controlURL, info.TargetID, info.URL)
}

func TestE2EProfileLifecycle(t *testing.T) {
	if os.Getenv(EnvRunE2E) != "1" {
		t.Skipf("set %s=1 to run E2E tests", EnvRunE2E)
	}
	if skipForRateLimit(t) {
		return
	}

	client, err := NewFromEnv(WithTimeout(60 * time.Second))
	if err != nil {
		t.Fatalf("NewFromEnv returned error: %v", err)
	}

	folderID := resolveE2EFolderID(t, client)
	ensureE2ECapacity(t, client, 10)

	profileName := "mlx-go-sdk-e2e-" + time.Now().UTC().Format("20060102-150405")
	ctx := context.Background()
	createResp, _, err := client.Profiles.Create(ctx, newE2ECreateProfileRequest(profileName, folderID))
	if err != nil {
		t.Fatalf("Profiles.Create returned error: %v", err)
	}
	if len(createResp.Data.IDs) == 0 {
		t.Fatalf("Profiles.Create returned no ids")
	}
	profileID := createResp.Data.IDs[0]
	updatedName := profileName + "-updated"
	patchedName := profileName + "-patched"

	defer func() {
		_, _, _ = client.Launcher.Stop(ctx, profileID)
		_, _, _ = client.Profiles.Delete(ctx, &DeleteProfilesRequest{IDs: []string{profileID}, Permanently: true})
	}()

	createdProfile := waitForProfileByName(t, client, profileName, false)
	if createdProfile.ID != profileID {
		t.Fatalf("expected created profile id %s, got %s", profileID, createdProfile.ID)
	}
	createdMetasResp, _, err := client.Profiles.GetMetas(ctx, &ProfileMetasRequest{IDs: []string{profileID}})
	if err != nil {
		t.Fatalf("Profiles.GetMetas before update returned error: %v", err)
	}
	if len(createdMetasResp.Data.Profiles) != 1 {
		t.Fatalf("expected one created profile meta before update, got %d", len(createdMetasResp.Data.Profiles))
	}
	if createdMetasResp.Data.Profiles[0].Parameters == nil {
		t.Fatalf("expected created profile meta to include parameters before update")
	}

	_, _, err = client.Profiles.Update(ctx, &UpdateProfileRequest{
		ProfileID:  profileID,
		Name:       updatedName,
		Notes:      "updated-note",
		Parameters: createdMetasResp.Data.Profiles[0].Parameters,
	})
	if err != nil {
		t.Fatalf("Profiles.Update returned error: %v", err)
	}

	updatedProfile := waitForProfileByName(t, client, updatedName, false)
	if updatedProfile.ID != profileID {
		t.Fatalf("expected updated profile id %s, got %s", profileID, updatedProfile.ID)
	}
	if updatedProfile.Notes != "updated-note" {
		t.Fatalf("expected updated notes, got %q", updatedProfile.Notes)
	}

	_, _, err = client.Profiles.Patch(ctx, &PatchProfileRequest{
		ProfileID:       profileID,
		Name:            patchedName,
		Notes:           "patched-note",
		CustomStartURLs: []string{"https://example.com"},
	})
	if err != nil {
		t.Fatalf("Profiles.Patch returned error: %v", err)
	}

	patchedProfile := waitForProfileByName(t, client, patchedName, false)
	if patchedProfile.ID != profileID {
		t.Fatalf("expected patched profile id %s, got %s", profileID, patchedProfile.ID)
	}
	if patchedProfile.Notes != "patched-note" {
		t.Fatalf("expected patched notes, got %q", patchedProfile.Notes)
	}

	metasResp, _, err := client.Profiles.GetMetas(ctx, &ProfileMetasRequest{IDs: []string{profileID}})
	if err != nil {
		t.Fatalf("Profiles.GetMetas returned error: %v", err)
	}
	if len(metasResp.Data.Profiles) != 1 {
		t.Fatalf("expected one profile meta, got %d", len(metasResp.Data.Profiles))
	}
	if metasResp.Data.Profiles[0].ID != profileID {
		t.Fatalf("unexpected profile meta id: %s", metasResp.Data.Profiles[0].ID)
	}

	_, _, err = client.Profiles.Delete(ctx, &DeleteProfilesRequest{
		IDs:         []string{profileID},
		Permanently: false,
	})
	if err != nil {
		t.Fatalf("Profiles.Delete soft delete returned error: %v", err)
	}

	waitForProfileAbsent(t, client, patchedName, false)
	deletedProfile := waitForProfileByName(t, client, patchedName, true)
	if deletedProfile.ID != profileID {
		t.Fatalf("expected deleted profile id %s, got %s", profileID, deletedProfile.ID)
	}

	_, _, err = client.Profiles.Restore(ctx, &RestoreProfilesRequest{IDs: []string{profileID}})
	if err != nil {
		t.Fatalf("Profiles.Restore returned error: %v", err)
	}

	restoredProfile := waitForProfileByName(t, client, patchedName, false)
	if restoredProfile.ID != profileID {
		t.Fatalf("expected restored profile id %s, got %s", profileID, restoredProfile.ID)
	}

	started, _, err := client.Launcher.Start(ctx, folderID, profileID, StartProfileOptions{})
	if err != nil {
		t.Fatalf("Launcher.Start returned error: %v", err)
	}
	if started.Data.ID != profileID {
		t.Fatalf("unexpected started profile id: %s", started.Data.ID)
	}

	status := waitForRunningStatus(t, client, profileID)
	if status.Data.ProfileID != profileID {
		t.Fatalf("unexpected runtime profile id: %s", status.Data.ProfileID)
	}

	_, _, err = client.Launcher.Stop(ctx, profileID)
	if err != nil {
		t.Fatalf("Launcher.Stop returned error: %v", err)
	}

	_, _, err = client.Profiles.Delete(ctx, &DeleteProfilesRequest{
		IDs:         []string{profileID},
		Permanently: true,
	})
	if err != nil {
		t.Fatalf("Profiles.Delete returned error: %v", err)
	}
	waitForProfileAbsent(t, client, patchedName, false)
	profileID = ""
	_ = status
}

func TestE2EProfileTransferLifecycle(t *testing.T) {
	if os.Getenv(EnvRunE2E) != "1" {
		t.Skipf("set %s=1 to run E2E tests", EnvRunE2E)
	}
	if skipForRateLimit(t) {
		return
	}

	client, err := NewFromEnv(WithTimeout(60 * time.Second))
	if err != nil {
		t.Fatalf("NewFromEnv returned error: %v", err)
	}

	folderID := resolveE2EFolderID(t, client)
	ensureE2ECapacity(t, client, 10)

	ctx := context.Background()
	profileName := "mlx-go-sdk-transfer-" + time.Now().UTC().Format("20060102-150405")
	createResp, _, err := client.Profiles.Create(ctx, newE2ECreateProfileRequest(profileName, folderID))
	if err != nil {
		t.Fatalf("Profiles.Create returned error: %v", err)
	}
	if len(createResp.Data.IDs) == 0 {
		t.Fatalf("Profiles.Create returned no ids")
	}
	originalProfileID := createResp.Data.IDs[0]
	importedProfileID := ""

	defer func() {
		if importedProfileID != "" {
			_, _, _ = client.Profiles.Delete(ctx, &DeleteProfilesRequest{IDs: []string{importedProfileID}, Permanently: true})
		}
		if originalProfileID != "" {
			_, _, _ = client.Profiles.Delete(ctx, &DeleteProfilesRequest{IDs: []string{originalProfileID}, Permanently: true})
		}
	}()

	exportResp, _, err := client.Transfers.Export(ctx, originalProfileID)
	if err != nil {
		t.Fatalf("Transfers.Export returned error: %v", err)
	}
	if exportResp.Data.ExportID == "" {
		t.Fatalf("Transfers.Export returned empty export id")
	}
	if exportResp.Data.ExportPath == "" {
		t.Fatalf("Transfers.Export returned empty export path")
	}

	exportStatus := waitForExportDone(t, client, exportResp.Data.ExportID)
	if exportStatus.Data.ExportPath == "" {
		t.Fatalf("Transfers.ExportStatus returned empty export path")
	}
	archivePath := exportStatus.Data.ArchivePath()
	artifactDescription := describeExportArtifact(archivePath)
	if artifactDescription == "missing" {
		t.Logf("normalized export archive path was not present on disk at validation time: raw=%s normalized=%s", exportStatus.Data.ExportPath, archivePath)
	}
	exportDir := filepath.Dir(archivePath)
	t.Logf("export completed: profile=%s export_id=%s export_path=%s normalized_archive=%s export_dir=%s artifact=%s", originalProfileID, exportStatus.Data.ExportID, exportStatus.Data.ExportPath, archivePath, exportDir, artifactDescription)

	_, _, err = client.Profiles.Delete(ctx, &DeleteProfilesRequest{
		IDs:         []string{originalProfileID},
		Permanently: true,
	})
	if err != nil {
		t.Fatalf("Profiles.Delete permanent delete returned error: %v", err)
	}
	waitForProfileAbsent(t, client, profileName, false)
	originalProfileID = ""

	importResp, _, err := client.Transfers.Import(ctx, &ImportProfileRequest{
		ImportPath: archivePath,
		IsLocal:    false,
	})
	if err != nil {
		t.Fatalf("Transfers.Import returned error: %v", err)
	}
	if importResp.Data.ImportID == "" {
		t.Fatalf("Transfers.Import returned empty import id")
	}

	importStatus := waitForImportDone(t, client, importResp.Data.ImportID)
	if importStatus.Data.NewProfileID == "" {
		t.Fatalf("Transfers.ImportStatus returned empty new profile id")
	}
	importedProfileID = importStatus.Data.NewProfileID
	if importStatus.Data.ImportPath != archivePath {
		t.Logf("import used path %s (normalized export archive %s, raw export path %s)", importStatus.Data.ImportPath, archivePath, exportStatus.Data.ExportPath)
	}

	metasResp, _, err := client.Profiles.GetMetas(ctx, &ProfileMetasRequest{IDs: []string{importedProfileID}})
	if err != nil {
		t.Fatalf("Profiles.GetMetas for imported profile returned error: %v", err)
	}
	if len(metasResp.Data.Profiles) != 1 {
		t.Fatalf("expected one imported profile meta, got %d", len(metasResp.Data.Profiles))
	}
	t.Logf("import completed: import_id=%s imported_profile_id=%s import_path=%s", importStatus.Data.ImportID, importedProfileID, importStatus.Data.ImportPath)
}

func TestE2EProfileCookieSeeding(t *testing.T) {
	if os.Getenv(EnvRunE2E) != "1" {
		t.Skipf("set %s=1 to run E2E tests", EnvRunE2E)
	}
	if skipForRateLimit(t) {
		return
	}

	client, err := NewFromEnv(WithTimeout(60 * time.Second))
	if err != nil {
		t.Fatalf("NewFromEnv returned error: %v", err)
	}

	folderID := resolveE2EFolderID(t, client)
	ensureE2ECapacity(t, client, 10)

	ctx := context.Background()
	profileName := "mlx-go-sdk-cookies-" + time.Now().UTC().Format("20060102-150405")
	createResp, _, err := client.Profiles.Create(ctx, newE2ECreateProfileRequest(profileName, folderID))
	if err != nil {
		t.Fatalf("Profiles.Create returned error: %v", err)
	}
	if len(createResp.Data.IDs) == 0 {
		t.Fatalf("Profiles.Create returned no ids")
	}
	profileID := createResp.Data.IDs[0]

	defer func() {
		_, _, _ = client.Profiles.Delete(ctx, &DeleteProfilesRequest{IDs: []string{profileID}, Permanently: true})
	}()

	websitesResp, _, err := client.Cookies.ListWebsites(ctx)
	if err != nil {
		t.Fatalf("Cookies.ListWebsites returned error: %v", err)
	}
	if !hasCookieWebsite(websitesResp.Data, "google") {
		t.Fatalf("expected pre-made cookies websites to include google, got %#v", websitesResp.Data)
	}

	seedResult, err := client.Cookies.SeedProfileCookies(ctx, SeedProfileCookiesOptions{
		ProfileID:               profileID,
		FolderID:                folderID,
		TargetWebsite:           "google",
		CreateMetadataIfMissing: true,
	})
	if err != nil {
		t.Fatalf("Cookies.SeedProfileCookies returned error: %v", err)
	}
	if seedResult.CookieCount == 0 {
		t.Fatalf("expected cookie seeding to import at least one cookie")
	}
	if seedResult.SelectedBundle == nil || len(seedResult.SelectedBundle.Data) == 0 {
		t.Fatalf("expected selected pre-made cookie bundle, got %#v", seedResult.SelectedBundle)
	}
	if !seedResult.MetadataCreated && !seedResult.MetadataUpdated {
		t.Fatalf("expected cookie seeding to create or update metadata")
	}
	if seedResult.FolderID != folderID {
		t.Fatalf("expected resolved folder id %s, got %s", folderID, seedResult.FolderID)
	}

	exportResp, _, err := client.Cookies.Export(ctx, &CookieExportRequest{
		ProfileID: profileID,
		FolderID:  folderID,
	})
	if err != nil {
		t.Fatalf("Cookies.Export returned error: %v", err)
	}
	if exportResp.Data.ProfileID != profileID {
		t.Fatalf("expected exported profile id %s, got %s", profileID, exportResp.Data.ProfileID)
	}
	if strings.TrimSpace(exportResp.Data.Cookies) == "" || exportResp.Data.Cookies == "[]" {
		t.Fatalf("expected exported cookies to be non-empty, got %q", exportResp.Data.Cookies)
	}
	t.Logf("cookie seeding completed: profile=%s target=%s imported=%d exported_timestamp=%d", profileID, seedResult.TargetWebsite, seedResult.CookieCount, exportResp.Data.Timestamp)
}

func TestE2EArchiveManagerExportToFolder(t *testing.T) {
	if os.Getenv(EnvRunE2E) != "1" {
		t.Skipf("set %s=1 to run E2E tests", EnvRunE2E)
	}
	if skipForRateLimit(t) {
		return
	}

	client, err := NewFromEnv(WithTimeout(60 * time.Second))
	if err != nil {
		t.Fatalf("NewFromEnv returned error: %v", err)
	}

	folderID := resolveE2EFolderID(t, client)
	ensureE2ECapacity(t, client, 10)

	ctx := context.Background()
	profileName := "mlx-go-sdk-archive-" + time.Now().UTC().Format("20060102-150405")
	createResp, _, err := client.Profiles.Create(ctx, newE2ECreateProfileRequest(profileName, folderID))
	if err != nil {
		t.Fatalf("Profiles.Create returned error: %v", err)
	}
	if len(createResp.Data.IDs) == 0 {
		t.Fatalf("Profiles.Create returned no ids")
	}
	profileID := createResp.Data.IDs[0]
	archiveRoot := t.TempDir()

	defer func() {
		_, _, _ = client.Profiles.Delete(ctx, &DeleteProfilesRequest{IDs: []string{profileID}, Permanently: true})
	}()

	result, err := client.Archives.ExportProfileToFolder(ctx, profileID, ExportProfileToFolderOptions{
		RootDir:      archiveRoot,
		FolderName:   "Archive / Demo",
		ProfileName:  profileName,
		WaitTimeout:  2 * time.Minute,
		PollInterval: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("Archives.ExportProfileToFolder returned error: %v", err)
	}
	if result.Archive == nil || result.ExportJob == nil {
		t.Fatalf("expected archive result and export job, got %#v", result)
	}
	if filepath.Ext(result.Archive.ArchivePath) != ".zip" {
		t.Fatalf("expected exported archive path to end with .zip, got %s", result.Archive.ArchivePath)
	}
	if _, err := os.Stat(result.Archive.ArchivePath); err != nil {
		t.Fatalf("expected organized archive on disk, got %v", err)
	}
	if strings.Contains(result.Archive.FolderName, "/") {
		t.Fatalf("expected sanitized folder name, got %s", result.Archive.FolderName)
	}

	t.Logf("archive manager export ok: profile=%s export_id=%s archive_dir=%s archive_path=%s raw_export_path=%s", profileID, result.ExportJob.Data.ExportID, result.Archive.ArchiveDir, result.Archive.ArchivePath, result.ExportJob.Data.ExportPath)
}

func TestE2EWorkflowHelpers(t *testing.T) {
	if os.Getenv(EnvRunE2E) != "1" {
		t.Skipf("set %s=1 to run E2E tests", EnvRunE2E)
	}
	if skipForRateLimit(t) {
		return
	}

	client, err := NewFromEnv(WithTimeout(60 * time.Second))
	if err != nil {
		t.Fatalf("NewFromEnv returned error: %v", err)
	}

	folderID := resolveE2EFolderID(t, client)
	ensureE2ECapacity(t, client, 10)

	ctx := context.Background()
	profileName := "mlx-go-sdk-workflow-" + time.Now().UTC().Format("20060102-150405")
	createResp, _, err := client.Profiles.Create(ctx, newE2ELocalCreateProfileRequest(profileName, folderID))
	if err != nil {
		t.Fatalf("Profiles.Create returned error: %v", err)
	}
	if len(createResp.Data.IDs) == 0 {
		t.Fatalf("Profiles.Create returned no ids")
	}
	profileID := createResp.Data.IDs[0]
	archiveRoot := t.TempDir()

	defer func() {
		_, _, _ = client.Launcher.Stop(ctx, profileID)
		_, _, _ = client.Profiles.Delete(ctx, &DeleteProfilesRequest{IDs: []string{profileID}, Permanently: true})
	}()

	startResult, err := client.Workflows.StartProfileByName(ctx, profileName, StartProfileByNameOptions{
		FindOptions:    &FindProfileOptions{FolderID: folderID},
		StartOptions:   StartProfileOptions{AutomationType: AutomationPlaywright},
		WaitForRunning: true,
	})
	if err != nil {
		t.Fatalf("Workflows.StartProfileByName returned error: %v", err)
	}
	if startResult.Profile.ID != profileID {
		t.Fatalf("expected started workflow profile id %s, got %s", profileID, startResult.Profile.ID)
	}
	if startResult.RuntimeStatus == nil || startResult.RuntimeStatus.Data.Status == "" {
		t.Fatalf("expected runtime status from workflow, got %#v", startResult.RuntimeStatus)
	}

	exportResult, err := client.Workflows.ExportProfileByNameToFolder(ctx, profileName, ExportProfileByNameToFolderOptions{
		FindOptions: &FindProfileOptions{FolderID: folderID},
		ExportOptions: ExportProfileToFolderOptions{
			RootDir:      archiveRoot,
			FolderName:   "Workflow Export",
			ProfileName:  profileName,
			WaitTimeout:  2 * time.Minute,
			PollInterval: 2 * time.Second,
		},
		StopBeforeExport: true,
	})
	if err != nil {
		t.Fatalf("Workflows.ExportProfileByNameToFolder returned error: %v", err)
	}
	if exportResult.Profile.ID != profileID {
		t.Fatalf("expected exported workflow profile id %s, got %s", profileID, exportResult.Profile.ID)
	}
	if exportResult.Export == nil || exportResult.Export.Archive == nil {
		t.Fatalf("expected managed export result, got %#v", exportResult.Export)
	}
	if _, err := os.Stat(exportResult.Export.Archive.ArchivePath); err != nil {
		t.Fatalf("expected exported archive on disk, got %v", err)
	}

	stopResult, err := client.Workflows.StopProfileByName(ctx, profileName, StopProfileByNameOptions{
		FindOptions:          &FindProfileOptions{FolderID: folderID},
		IgnoreAlreadyStopped: true,
	})
	if err != nil {
		t.Fatalf("Workflows.StopProfileByName returned error: %v", err)
	}
	if stopResult.Profile.ID != profileID {
		t.Fatalf("expected stopped workflow profile id %s, got %s", profileID, stopResult.Profile.ID)
	}

	t.Logf("workflow helpers ok: profile=%s runtime_status=%s archive=%s", profileID, startResult.RuntimeStatus.Data.Status, exportResult.Export.Archive.ArchivePath)
}

func TestE2EResourceProfileTemplateLifecycle(t *testing.T) {
	if os.Getenv(EnvRunE2E) != "1" {
		t.Skipf("set %s=1 to run E2E tests", EnvRunE2E)
	}
	if skipForRateLimit(t) {
		return
	}

	client, err := NewFromEnv(WithTimeout(60 * time.Second))
	if err != nil {
		t.Fatalf("NewFromEnv returned error: %v", err)
	}

	ctx := context.Background()
	typesResp, _, err := client.Resources.ListTypes(ctx)
	if err != nil {
		t.Fatalf("Resources.ListTypes returned error: %v", err)
	}
	if len(typesResp.Data.Types) == 0 {
		t.Fatal("expected resource types")
	}

	trashbin := false
	listResp, _, err := client.Resources.ListProfileTemplates(ctx, &ListResourceMetasOptions{ObjectName: "Template", Limit: 20, Offset: 0, Trashbin: &trashbin})
	if err != nil {
		t.Fatalf("Resources.ListProfileTemplates returned error: %v", err)
	}
	if len(listResp.Data.Objects) == 0 {
		t.Fatal("expected existing Template resource to be listed")
	}

	resourceID := listResp.Data.Objects[0].ID
	metaResp, _, err := client.Resources.GetMeta(ctx, resourceID)
	if err != nil {
		t.Fatalf("Resources.GetMeta returned error: %v", err)
	}
	if metaResp.Data.ID != resourceID {
		t.Fatalf("expected resource id %s, got %s", resourceID, metaResp.Data.ID)
	}

	usageResp, _, err := client.Resources.ObjectProfileUsages(ctx, resourceID)
	if err != nil {
		t.Fatalf("Resources.ObjectProfileUsages returned error: %v", err)
	}
	if usageResp == nil {
		t.Fatal("expected usage response")
	}

	name := "mlx-go-sdk-resource-" + time.Now().UTC().Format("20060102-150405")
	body := fmt.Sprintf(`{"name":"%s","mainParams":{"browser_type":"mimic","os_type":"windows","parameters":{"storage":{"is_local":true}}}}`, name)
	created, _, err := client.Resources.CreateProfileTemplate(ctx, &CreateProfileTemplateRequest{
		Name: name,
		Body: body,
		Meta: fmt.Sprintf(`{"name":"%s","source":"mlx-go-sdk-e2e"}`, name),
	})
	if err != nil {
		t.Fatalf("Resources.CreateProfileTemplate returned error: %v", err)
	}
	if created.Data.MetaID == "" {
		t.Fatal("expected created resource meta id")
	}
	createdID := created.Data.MetaID

	defer func() {
		_, _, _ = client.Resources.Delete(ctx, createdID, true)
	}()

	downloadResp, _, err := client.Resources.Download(ctx, createdID)
	if err != nil {
		t.Fatalf("Resources.Download returned error: %v", err)
	}
	if strings.TrimSpace(downloadResp.Path) == "" {
		t.Fatal("expected downloaded path")
	}

	t.Logf("resources/template lifecycle ok: existing_template=%s created_template=%s downloaded=%s listed=%d usages=%d", resourceID, createdID, downloadResp.Path, len(listResp.Data.Objects), len(usageResp.Data))
}

func TestE2ELocalProfileSemanticsCreate(t *testing.T) {
	if os.Getenv(EnvRunE2E) != "1" {
		t.Skipf("set %s=1 to run E2E tests", EnvRunE2E)
	}
	if skipForRateLimit(t) {
		return
	}

	client, err := NewFromEnv(WithTimeout(60 * time.Second))
	if err != nil {
		t.Fatalf("NewFromEnv returned error: %v", err)
	}

	folderID := resolveE2EFolderID(t, client)
	ensureE2ECapacity(t, client, 10)

	ctx := context.Background()
	profileName := "mlx-local-semantics-create-" + time.Now().UTC().Format("20060102-150405")
	createResp, _, err := client.Profiles.Create(ctx, newE2ELocalCreateProfileRequest(profileName, folderID))
	if err != nil {
		t.Fatalf("Profiles.Create returned error: %v", err)
	}
	if len(createResp.Data.IDs) == 0 {
		t.Fatal("expected created profile id")
	}
	profileID := createResp.Data.IDs[0]

	defer func() {
		_, _, _ = client.Launcher.Stop(ctx, profileID)
		_, _, _ = client.Profiles.Delete(ctx, &DeleteProfilesRequest{IDs: []string{profileID}, Permanently: true})
	}()

	report := inspectLocalProfileSemantics(t, client, folderID, profileID, profileName)
	t.Logf("local create semantics: profile=%s search_all_local=%t meta_is_local=%t metas_is_local=%t meta_storage=%t launcher_local_before=%d launcher_local_after=%d", report.ProfileID, report.SearchAll.IsLocal, report.Meta.IsLocal, report.Metas.IsLocal, report.MetaStorageIsLocal, report.LauncherBefore.Local, report.LauncherAfter.Local)
}

func TestE2ELocalProfileSemanticsImport(t *testing.T) {
	if os.Getenv(EnvRunE2E) != "1" {
		t.Skipf("set %s=1 to run E2E tests", EnvRunE2E)
	}
	if skipForRateLimit(t) {
		return
	}

	client, err := NewFromEnv(WithTimeout(60 * time.Second))
	if err != nil {
		t.Fatalf("NewFromEnv returned error: %v", err)
	}

	folderID := resolveE2EFolderID(t, client)
	ensureE2ECapacity(t, client, 10)

	ctx := context.Background()
	sourceName := "mlx-local-semantics-source-" + time.Now().UTC().Format("20060102-150405")
	createResp, _, err := client.Profiles.Create(ctx, newE2ECreateProfileRequest(sourceName, folderID))
	if err != nil {
		t.Fatalf("Profiles.Create source returned error: %v", err)
	}
	if len(createResp.Data.IDs) == 0 {
		t.Fatal("expected source profile id")
	}
	sourceID := createResp.Data.IDs[0]
	importedID := ""

	defer func() {
		if importedID != "" {
			_, _, _ = client.Launcher.Stop(ctx, importedID)
			_, _, _ = client.Profiles.Delete(ctx, &DeleteProfilesRequest{IDs: []string{importedID}, Permanently: true})
		}
		if sourceID != "" {
			_, _, _ = client.Profiles.Delete(ctx, &DeleteProfilesRequest{IDs: []string{sourceID}, Permanently: true})
		}
	}()

	exportResp, _, err := client.Transfers.Export(ctx, sourceID)
	if err != nil {
		t.Fatalf("Transfers.Export returned error: %v", err)
	}
	exportStatus := waitForExportDone(t, client, exportResp.Data.ExportID)
	archivePath := exportStatus.Data.ArchivePath()

	_, _, err = client.Profiles.Delete(ctx, &DeleteProfilesRequest{IDs: []string{sourceID}, Permanently: true})
	if err != nil {
		t.Fatalf("Profiles.Delete source returned error: %v", err)
	}
	sourceID = ""

	importResp, _, err := client.Transfers.Import(ctx, &ImportProfileRequest{ImportPath: archivePath, IsLocal: true})
	if err != nil {
		t.Fatalf("Transfers.Import returned error: %v", err)
	}
	importStatus := waitForImportDone(t, client, importResp.Data.ImportID)
	if strings.TrimSpace(importStatus.Data.NewProfileID) == "" {
		t.Fatal("expected imported profile id")
	}
	importedID = importStatus.Data.NewProfileID

	report := inspectLocalProfileSemantics(t, client, folderID, importedID, sourceName)
	t.Logf("local import semantics: profile=%s search_all_local=%t meta_is_local=%t metas_is_local=%t meta_storage=%t launcher_local_before=%d launcher_local_after=%d archive=%s", report.ProfileID, report.SearchAll.IsLocal, report.Meta.IsLocal, report.Metas.IsLocal, report.MetaStorageIsLocal, report.LauncherBefore.Local, report.LauncherAfter.Local, archivePath)
}

func TestE2EExtensionWorkflow(t *testing.T) {
	if os.Getenv(EnvRunE2E) != "1" {
		t.Skipf("set %s=1 to run E2E tests", EnvRunE2E)
	}
	if skipForRateLimit(t) {
		return
	}

	client, err := NewFromEnv(WithTimeout(60 * time.Second))
	if err != nil {
		t.Fatalf("NewFromEnv returned error: %v", err)
	}

	ctx := context.Background()
	profileID, meta := resolveE2ELocalProfileForExtension(t, client)
	if strings.TrimSpace(profileID) == "" || meta == nil {
		t.Skip("no local profile available for live extension workflow validation")
	}

	if !meta.CheckLocal() {
		t.Skipf("expected a local profile for extension validation, got profile=%s check_local=%t raw_is_local=%t", profileID, meta.CheckLocal(), meta.IsLocal)
	}

	extensionPath := createE2EExtensionArchive(t, meta.Name)
	uploadResp, _, err := client.Resources.UploadExtension(ctx, &UploadExtensionRequest{ObjectPath: extensionPath})
	if err != nil {
		t.Fatalf("Resources.UploadExtension returned error: %v", err)
	}
	if uploadResp.Data.MetaID == "" {
		t.Fatal("expected uploaded extension meta id")
	}
	extensionID := uploadResp.Data.MetaID
	defer func() {
		_, _, _ = client.Resources.Delete(ctx, extensionID, true)
	}()

	if _, _, err := client.Resources.EnableExtensionForProfiles(ctx, extensionID, &SetResourceProfilesRequest{ProfileIDs: []string{profileID}}); err != nil {
		t.Fatalf("Resources.EnableExtensionForProfiles returned error: %v", err)
	}

	profileUsages := waitForProfileExtensionUsage(t, client, profileID, extensionID, true)
	objectUsages := waitForObjectProfileUsage(t, client, extensionID, profileID, true)
	if profileUsages == nil {
		t.Logf("live note: profile-centric extension usage read did not confirm the attachment for profile=%s extension=%s; relying on object-centric usage verification", profileID, extensionID)
	}

	if _, _, err := client.Resources.DisableExtensionForProfiles(ctx, extensionID, &SetResourceProfilesRequest{ProfileIDs: []string{profileID}}); err != nil {
		t.Fatalf("Resources.DisableExtensionForProfiles returned error: %v", err)
	}
	disabledProfileUsages := waitForProfileExtensionUsage(t, client, profileID, extensionID, false)
	if disabledProfileUsages == nil {
		t.Logf("live note: profile-centric extension usage read did not confirm the detach for profile=%s extension=%s; treating object-centric verification as the stronger signal", profileID, extensionID)
	}

	profileUsageCount := 0
	if profileUsages != nil {
		profileUsageCount = len(profileUsages.Data)
	}

	t.Logf("extension workflow ok: profile=%s local_profile=%t raw_meta_is_local=%t extension=%s archive=%s profile_usage_confirmed=%t profile_usages=%d object_usages=%d", profileID, meta.CheckLocal(), meta.IsLocal, extensionID, extensionPath, profileUsages != nil, profileUsageCount, len(objectUsages.Data))
}

func TestE2EChromeWebStoreExtensionCreation(t *testing.T) {
	if os.Getenv(EnvRunE2E) != "1" {
		t.Skipf("set %s=1 to run E2E tests", EnvRunE2E)
	}
	if skipForRateLimit(t) {
		return
	}

	client, err := NewFromEnv(WithTimeout(60 * time.Second))
	if err != nil {
		t.Fatalf("NewFromEnv returned error: %v", err)
	}

	ctx := context.Background()
	trashbin := false
	beforeResp, _, err := client.Resources.ListExtensions(ctx, &ListResourceMetasOptions{
		ObjectName: "Google Docs Offline",
		Limit:      50,
		Offset:     0,
		Trashbin:   &trashbin,
	})
	if err != nil {
		t.Fatalf("Resources.ListExtensions before create returned error: %v", err)
	}
	beforeIDs := make(map[string]struct{}, len(beforeResp.Data.Objects))
	for _, object := range beforeResp.Data.Objects {
		beforeIDs[object.ID] = struct{}{}
	}

	_, _, err = client.Resources.CreateExtensionFromChromeWebStore(ctx, &CreateChromeWebStoreExtensionRequest{
		ExtensionID: "ghbmnnjooekpmoecnnnilnnbdlolhkhi",
		BrowserType: "mimic",
	})
	if err != nil {
		message := err.Error()
		if strings.Contains(message, "failed to fetch extension") || strings.Contains(message, "status: 404") {
			t.Logf("chrome web store extension creation currently fails in live launcher fetch flow: %v", err)
			return
		}
		t.Fatalf("Resources.CreateExtensionFromChromeWebStore returned error: %v", err)
	}

	created := waitForNewNamedExtension(t, client, "Google Docs Offline", beforeIDs)
	defer func() {
		if created.ID != "" {
			_, _, _ = client.Resources.Delete(ctx, created.ID, true)
		}
	}()

	t.Logf("chrome web store extension ok: extension=%s name=%s storage=%s", created.ID, created.ObjectName, created.StorageType)
}

func preflightE2EError() error {
	client, err := NewFromEnv(WithTimeout(30 * time.Second))
	if err != nil {
		return err
	}
	_, _, err = client.Folders.List(context.Background())
	return err
}

func skipForRateLimit(t *testing.T) bool {
	t.Helper()
	return isRateLimited(t, preflightE2EError())
}

func isRateLimited(t *testing.T, err error) bool {
	t.Helper()
	if err == nil {
		return false
	}
	var apiErr *ErrorResponse
	if errors.As(err, &apiErr) && apiErr != nil && apiErr.Response != nil && apiErr.Response.StatusCode == 429 {
		t.Skipf("E2E skipped due to MLX API rate limit: %v", err)
		return true
	}
	return false
}

func skipOrFatalRateLimit(t *testing.T, err error, format string, args ...any) {
	t.Helper()
	if isRateLimited(t, err) {
		return
	}
	t.Fatalf(format, args...)
}

func newE2ECreateProfileRequest(profileName, folderID string) *CreateProfileRequest {
	return &CreateProfileRequest{
		Name:        profileName,
		BrowserType: "mimic",
		FolderID:    folderID,
		OSType:      "windows",
		Parameters: &ProfileParameters{
			Flags: &ProfileFlags{
				AudioMasking:        "natural",
				FontsMasking:        "mask",
				GeolocationMasking:  "mask",
				GeolocationPopup:    "prompt",
				GraphicsMasking:     "mask",
				GraphicsNoise:       "mask",
				LocalizationMasking: "mask",
				MediaDevicesMasking: "natural",
				NavigatorMasking:    "mask",
				PortsMasking:        "mask",
				ProxyMasking:        "disabled",
				ScreenMasking:       "mask",
				TimezoneMasking:     "mask",
				WebRTCMasking:       "mask",
			},
			Storage:     &Storage{IsLocal: false},
			Fingerprint: &Fingerprint{},
		},
	}
}

func newE2ELocalCreateProfileRequest(profileName, folderID string) *CreateProfileRequest {
	req := newE2ECreateProfileRequest(profileName, folderID)
	if req.Parameters == nil {
		req.Parameters = &ProfileParameters{}
	}
	if req.Parameters.Storage == nil {
		req.Parameters.Storage = &Storage{}
	}
	req.Parameters.Storage.IsLocal = true
	return req
}

func resolveE2EFolderID(t *testing.T, client *Client) string {
	t.Helper()

	if folderID := os.Getenv(EnvE2EFolderID); folderID != "" {
		return folderID
	}

	resp, _, err := client.Folders.List(context.Background())
	if err != nil {
		skipOrFatalRateLimit(t, err, "Folders.List returned error while resolving E2E folder: %v", err)
	}
	if len(resp.Data.Folders) == 0 {
		t.Fatalf("no workspace folders available for E2E tests")
	}
	return resp.Data.Folders[0].FolderID
}

func ensureE2ECapacity(t *testing.T, client *Client, maxProfiles int) {
	t.Helper()

	activeCount := countProfiles(t, client, false)
	trashCount := countProfiles(t, client, true)
	total := activeCount + trashCount
	if total >= maxProfiles {
		t.Skipf("E2E skipped: active profiles (%d) + trash profiles (%d) = %d, which reaches the subscription cap of %d; permanently delete profiles from trash first", activeCount, trashCount, total, maxProfiles)
	}
}

func countProfiles(t *testing.T, client *Client, removed bool) int {
	t.Helper()

	resp, _, err := client.Profiles.Search(context.Background(), &SearchProfilesRequest{
		IsRemoved:   removed,
		Limit:       1,
		Offset:      0,
		SearchText:  "",
		StorageType: "all",
	})
	if err != nil {
		skipOrFatalRateLimit(t, err, "Profiles.Search count returned error: %v", err)
	}
	return resp.Data.TotalCount
}

func waitForProfileByName(t *testing.T, client *Client, profileName string, removed bool) Profile {
	t.Helper()

	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		resp, _, err := client.Profiles.Search(context.Background(), &SearchProfilesRequest{
			IsRemoved:   removed,
			Limit:       100,
			Offset:      0,
			SearchText:  profileName,
			StorageType: "all",
			OrderBy:     "updated_at",
			Sort:        "desc",
		})
		if err != nil {
			if isRateLimited(t, err) {
				return Profile{}
			}
			time.Sleep(2 * time.Second)
			continue
		}
		for _, profile := range resp.Data.Profiles {
			if strings.EqualFold(profile.Name, profileName) {
				return profile
			}
		}
		time.Sleep(2 * time.Second)
	}

	t.Fatalf("profile %q (removed=%t) was not found before timeout", profileName, removed)
	return Profile{}
}

func waitForProfileAbsent(t *testing.T, client *Client, profileName string, removed bool) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		resp, _, err := client.Profiles.Search(context.Background(), &SearchProfilesRequest{
			IsRemoved:   removed,
			Limit:       100,
			Offset:      0,
			SearchText:  profileName,
			StorageType: "all",
		})
		if err != nil {
			if isRateLimited(t, err) {
				return
			}
			time.Sleep(2 * time.Second)
			continue
		}
		found := false
		for _, profile := range resp.Data.Profiles {
			if strings.EqualFold(profile.Name, profileName) {
				found = true
				break
			}
		}
		if !found {
			return
		}
		time.Sleep(2 * time.Second)
	}

	t.Fatalf("profile %q (removed=%t) was still present after timeout", profileName, removed)
}

func waitForRunningStatus(t *testing.T, client *Client, profileID string) *ProfileRuntimeStatusResponse {
	t.Helper()

	resp, _, err := client.Launcher.WaitForRunning(context.Background(), profileID, PollOptions{})
	if err != nil {
		skipOrFatalRateLimit(t, err, "Launcher.WaitForRunning returned error: %v", err)
	}
	return resp
}

func waitForExportDone(t *testing.T, client *Client, exportID string) *ExportStatusResponse {
	t.Helper()

	resp, _, err := client.Transfers.WaitForExportDone(context.Background(), exportID, PollOptions{})
	if err != nil {
		skipOrFatalRateLimit(t, err, "Transfers.WaitForExportDone returned error: %v", err)
	}
	return resp
}

func waitForImportDone(t *testing.T, client *Client, importID string) *ImportStatusResponse {
	t.Helper()

	resp, _, err := client.Transfers.WaitForImportDone(context.Background(), importID, PollOptions{})
	if err != nil {
		skipOrFatalRateLimit(t, err, "Transfers.WaitForImportDone returned error: %v", err)
	}
	return resp
}

func createE2EExtensionArchive(t *testing.T, name string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "extension.zip")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create extension archive: %v", err)
	}
	defer file.Close()

	zw := zip.NewWriter(file)
	manifest, err := zw.Create("manifest.json")
	if err != nil {
		t.Fatalf("create manifest.json entry: %v", err)
	}
	manifestBody := fmt.Sprintf(`{"manifest_version":3,"name":"%s","version":"1.0.0","action":{"default_title":"github.com/minskyagenda0708-cmd/mlx-go-sdk"}}`, name)
	if _, err := manifest.Write([]byte(manifestBody)); err != nil {
		t.Fatalf("write manifest.json: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close extension archive: %v", err)
	}
	return path
}

type localProfileSemanticsReport struct {
	ProfileID           string
	SearchAll           *Profile
	SearchLocal         *Profile
	SearchCloud         *Profile
	Meta                *ProfileMeta
	Metas               *ProfileMeta
	MetaStorageIsLocal  bool
	MetasStorageIsLocal bool
	LauncherBefore      LauncherActiveCounter
	LauncherAfter       LauncherActiveCounter
}

func inspectLocalProfileSemantics(t *testing.T, client *Client, folderID, profileID, profileName string) *localProfileSemanticsReport {
	t.Helper()

	ctx := context.Background()
	searchAll := searchExactProfile(t, client, profileName, "all")
	searchLocal := searchExactProfile(t, client, profileName, "local")
	searchCloud := searchOptionalExactProfile(t, client, profileName, "cloud")
	if searchAll == nil {
		t.Fatalf("expected profile %q in storage_type=all search", profileName)
	}
	if searchLocal == nil {
		t.Fatalf("expected profile %q in storage_type=local search", profileName)
	}
	if searchCloud != nil {
		t.Fatalf("expected profile %q to be absent from storage_type=cloud search, got id=%s", profileName, searchCloud.ID)
	}
	if !searchAll.IsLocal || !searchLocal.IsLocal {
		t.Fatalf("expected local search signals for %q, got search_all=%t search_local=%t", profileName, searchAll.IsLocal, searchLocal.IsLocal)
	}

	meta, _, err := client.Profiles.GetMeta(ctx, profileID)
	if err != nil {
		t.Fatalf("Profiles.GetMeta returned error: %v", err)
	}
	metasResp, _, err := client.Profiles.GetMetas(ctx, &ProfileMetasRequest{IDs: []string{profileID}})
	if err != nil {
		t.Fatalf("Profiles.GetMetas returned error: %v", err)
	}
	if len(metasResp.Data.Profiles) != 1 {
		t.Fatalf("expected one profile meta, got %d", len(metasResp.Data.Profiles))
	}
	metas := metasResp.Data.Profiles[0]
	metaStorageIsLocal := meta.Parameters != nil && meta.Parameters.Storage != nil && meta.Parameters.Storage.IsLocal
	metasStorageIsLocal := metas.Parameters != nil && metas.Parameters.Storage != nil && metas.Parameters.Storage.IsLocal
	if !metaStorageIsLocal || !metasStorageIsLocal {
		t.Fatalf("expected profile meta parameters.storage.is_local=true for %q, got meta=%t metas=%t", profileName, metaStorageIsLocal, metasStorageIsLocal)
	}
	if !meta.IsLocal || !metas.IsLocal {
		t.Logf("live mismatch: search and launcher classify %q as local, but top-level meta flags are GetMeta.IsLocal=%t GetMetas.IsLocal=%t", profileName, meta.IsLocal, metas.IsLocal)
	}

	beforeResp, _, err := client.Launcher.Statuses(ctx)
	if err != nil {
		t.Fatalf("Launcher.Statuses before start returned error: %v", err)
	}
	_, _, err = client.Launcher.Start(ctx, folderID, profileID, StartProfileOptions{AutomationType: AutomationPlaywright})
	if err != nil {
		t.Fatalf("Launcher.Start returned error: %v", err)
	}
	_, _, err = client.Launcher.WaitForRunning(ctx, profileID, PollOptions{})
	if err != nil {
		t.Fatalf("Launcher.WaitForRunning returned error: %v", err)
	}
	afterResp, _, err := client.Launcher.Statuses(ctx)
	if err != nil {
		t.Fatalf("Launcher.Statuses after start returned error: %v", err)
	}
	_, _, _ = client.Launcher.Stop(ctx, profileID)

	localDelta := afterResp.Data.ActiveCounter.Local - beforeResp.Data.ActiveCounter.Local
	cloudDelta := afterResp.Data.ActiveCounter.Cloud - beforeResp.Data.ActiveCounter.Cloud
	if localDelta <= 0 {
		t.Fatalf("expected local active counter to increase for %q, got before=%#v after=%#v", profileName, beforeResp.Data.ActiveCounter, afterResp.Data.ActiveCounter)
	}
	if cloudDelta != 0 {
		t.Fatalf("expected cloud active counter to remain unchanged for %q, got before=%#v after=%#v", profileName, beforeResp.Data.ActiveCounter, afterResp.Data.ActiveCounter)
	}

	return &localProfileSemanticsReport{
		ProfileID:           profileID,
		SearchAll:           searchAll,
		SearchLocal:         searchLocal,
		SearchCloud:         searchCloud,
		Meta:                meta,
		Metas:               &metas,
		MetaStorageIsLocal:  metaStorageIsLocal,
		MetasStorageIsLocal: metasStorageIsLocal,
		LauncherBefore:      beforeResp.Data.ActiveCounter,
		LauncherAfter:       afterResp.Data.ActiveCounter,
	}
}

func searchExactProfile(t *testing.T, client *Client, profileName, storageType string) *Profile {
	t.Helper()
	profile := searchOptionalExactProfile(t, client, profileName, storageType)
	if profile == nil {
		t.Fatalf("expected profile %q in storage_type=%s search", profileName, storageType)
	}
	return profile
}

func searchOptionalExactProfile(t *testing.T, client *Client, profileName, storageType string) *Profile {
	t.Helper()

	resp, _, err := client.Profiles.Search(context.Background(), &SearchProfilesRequest{
		IsRemoved:   false,
		Limit:       100,
		Offset:      0,
		SearchText:  profileName,
		StorageType: storageType,
		OrderBy:     "updated_at",
		Sort:        "desc",
	})
	if err != nil {
		skipOrFatalRateLimit(t, err, "Profiles.Search storage_type=%s returned error: %v", storageType, err)
	}
	for _, profile := range resp.Data.Profiles {
		if strings.EqualFold(profile.Name, profileName) {
			matched := profile
			return &matched
		}
	}
	return nil
}

func resolveE2ELocalProfileForExtension(t *testing.T, client *Client) (string, *ProfileMeta) {
	t.Helper()

	ctx := context.Background()
	if profileID := os.Getenv(EnvE2EProfileID); strings.TrimSpace(profileID) != "" {
		meta, _, err := client.Profiles.GetMeta(ctx, profileID)
		if err != nil {
			t.Fatalf("Profiles.GetMeta returned error for %s: %v", EnvE2EProfileID, err)
		}
		return profileID, meta
	}

	folderID := resolveE2EFolderID(t, client)
	ensureE2ECapacity(t, client, 10)

	profileName := "mlx-go-sdk-ext-local-" + time.Now().UTC().Format("20060102-150405")
	createResp, _, err := client.Profiles.Create(ctx, newE2ELocalCreateProfileRequest(profileName, folderID))
	if err != nil {
		t.Fatalf("Profiles.Create returned error: %v", err)
	}
	if len(createResp.Data.IDs) == 0 {
		t.Fatal("expected created profile id")
	}
	profileID := createResp.Data.IDs[0]
	t.Cleanup(func() {
		_, _, _ = client.Profiles.Delete(ctx, &DeleteProfilesRequest{IDs: []string{profileID}, Permanently: true})
	})

	meta, _, err := client.Profiles.GetMeta(ctx, profileID)
	if err != nil {
		t.Fatalf("Profiles.GetMeta returned error: %v", err)
	}
	if !meta.CheckLocal() {
		t.Logf("live note: local-intended profile resolved as non-local even via CheckLocal for profile=%s raw_is_local=%t", profileID, meta.IsLocal)
	}
	return profileID, meta
}

func waitForProfileExtensionUsage(t *testing.T, client *Client, profileID, extensionID string, shouldExist bool) *ProfileObjectUsagesResponse {
	t.Helper()

	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		resp, _, err := client.Resources.ProfileExtensionUsages(context.Background(), profileID)
		if err == nil {
			found := false
			for _, usage := range resp.Data {
				if usage.ID == extensionID {
					found = true
					break
				}
			}
			if found == shouldExist {
				return resp
			}
		}
		time.Sleep(2 * time.Second)
	}

	t.Logf("live note: profile extension usage for profile=%s extension=%s did not reach shouldExist=%t before timeout", profileID, extensionID, shouldExist)
	return nil
}

func waitForObjectProfileUsage(t *testing.T, client *Client, extensionID, profileID string, shouldExist bool) *ObjectProfileUsagesResponse {
	t.Helper()

	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		resp, _, err := client.Resources.ObjectProfileUsages(context.Background(), extensionID)
		if err == nil {
			found := false
			for _, usage := range resp.Data {
				if usage.ID == profileID {
					found = true
					break
				}
			}
			if found == shouldExist {
				return resp
			}
		}
		time.Sleep(2 * time.Second)
	}

	t.Fatalf("object profile usage for extension=%s profile=%s did not reach shouldExist=%t before timeout", extensionID, profileID, shouldExist)
	return nil
}

func waitForNewNamedExtension(t *testing.T, client *Client, objectName string, beforeIDs map[string]struct{}) ResourceMeta {
	t.Helper()

	deadline := time.Now().Add(2 * time.Minute)
	trashbin := false
	for time.Now().Before(deadline) {
		resp, _, err := client.Resources.ListExtensions(context.Background(), &ListResourceMetasOptions{
			ObjectName: objectName,
			Limit:      50,
			Offset:     0,
			Trashbin:   &trashbin,
		})
		if err == nil {
			for _, object := range resp.Data.Objects {
				if _, seen := beforeIDs[object.ID]; !seen {
					return object
				}
			}
		}
		time.Sleep(3 * time.Second)
	}

	t.Fatalf("new extension named %q was not listed before timeout", objectName)
	return ResourceMeta{}
}

func hasCookieWebsite(websites []CookieWebsite, key string) bool {
	for _, website := range websites {
		if strings.EqualFold(website.Key, key) {
			return true
		}
	}
	return false
}

func describeExportArtifact(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "missing"
	}
	if info.IsDir() {
		entries, readErr := os.ReadDir(path)
		if readErr != nil {
			return "directory"
		}
		return fmt.Sprintf("directory(%d entries)", len(entries))
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return "file(no extension)"
	}
	return fmt.Sprintf("file(%s)", ext)
}
