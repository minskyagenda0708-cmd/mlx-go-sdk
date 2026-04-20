package mlx

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-rod/rod"
	rodlauncher "github.com/go-rod/rod/lib/launcher"
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

	started, _, err := client.Launcher.Start(ctx, folderID, profileID, StartProfileOptions{AutomationType: AutomationRod})
	if err != nil {
		t.Fatalf("Launcher.Start returned error: %v", err)
	}
	selectedAutomation := AutomationRod
	if strings.TrimSpace(started.Data.Port) == "" {
		t.Log("launcher returned empty port for automation_type=rod; retrying with automation_type=playwright and attaching Rod to the resulting DevTools endpoint")
		_, _, _ = client.Launcher.Stop(ctx, profileID)
		started, _, err = client.Launcher.Start(ctx, folderID, profileID, StartProfileOptions{AutomationType: AutomationPlaywright})
		if err != nil {
			t.Fatalf("Launcher.Start playwright fallback returned error: %v", err)
		}
		selectedAutomation = AutomationPlaywright
	}
	if strings.TrimSpace(started.Data.Port) == "" {
		t.Fatal("expected launcher start to return a usable cdp port")
	}

	controlURL, err := rodlauncher.ResolveURL(started.Data.Port)
	if err != nil {
		t.Fatalf("rod launcher.ResolveURL returned error: %v", err)
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

	t.Logf("rod connection ok: profile=%s automation=%s port=%s control_url=%s target=%s url=%s", profileID, selectedAutomation, started.Data.Port, controlURL, info.TargetID, info.URL)
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
		t.Fatalf("Folders.List returned error while resolving E2E folder: %v", err)
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
		t.Fatalf("Profiles.Search count returned error: %v", err)
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
		if err == nil {
			for _, profile := range resp.Data.Profiles {
				if strings.EqualFold(profile.Name, profileName) {
					return profile
				}
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
		if err == nil {
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
		}
		time.Sleep(2 * time.Second)
	}

	t.Fatalf("profile %q (removed=%t) was still present after timeout", profileName, removed)
}

func waitForRunningStatus(t *testing.T, client *Client, profileID string) *ProfileRuntimeStatusResponse {
	t.Helper()

	resp, _, err := client.Launcher.WaitForRunning(context.Background(), profileID, PollOptions{})
	if err != nil {
		t.Fatalf("Launcher.WaitForRunning returned error: %v", err)
	}
	return resp
}

func waitForExportDone(t *testing.T, client *Client, exportID string) *ExportStatusResponse {
	t.Helper()

	resp, _, err := client.Transfers.WaitForExportDone(context.Background(), exportID, PollOptions{})
	if err != nil {
		t.Fatalf("Transfers.WaitForExportDone returned error: %v", err)
	}
	return resp
}

func waitForImportDone(t *testing.T, client *Client, importID string) *ImportStatusResponse {
	t.Helper()

	resp, _, err := client.Transfers.WaitForImportDone(context.Background(), importID, PollOptions{})
	if err != nil {
		t.Fatalf("Transfers.WaitForImportDone returned error: %v", err)
	}
	return resp
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
