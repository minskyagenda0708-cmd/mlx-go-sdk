package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	mlx "github.com/bath0ry/mlx-go-sdk"
)

func TestLoadProfileTemplatePrefersDownloadedBodyWhenUsable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "template.json")
	body := `{
		"name": "Default Template",
		"mainParams": {
			"name": "Template Profile",
			"browser_type": "mimic",
			"folder_id": "folder-1",
			"os_type": "windows",
			"notes": "from-template",
			"parameters": {
				"storage": {
					"is_local": false
				},
				"custom_start_urls": [
					"https://example.com"
				]
			}
		}
	}`

	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	doc, err := loadProfileTemplate(`{"name":"Meta Template","mainParams":{"name":"Meta Profile","browser_type":"mimic","folder_id":"folder-meta","os_type":"windows"}}`, path)
	if err != nil {
		t.Fatalf("loadProfileTemplate returned error: %v", err)
	}
	if doc == nil {
		t.Fatal("expected parsed template document")
	}
	if doc.Name != "Default Template" {
		t.Fatalf("unexpected template name: %q", doc.Name)
	}
	if doc.MainParams.Name != "Template Profile" {
		t.Fatalf("unexpected mainParams.name: %q", doc.MainParams.Name)
	}
	if doc.MainParams.BrowserType != "mimic" {
		t.Fatalf("unexpected browser type: %q", doc.MainParams.BrowserType)
	}
	if doc.MainParams.FolderID != "folder-1" {
		t.Fatalf("unexpected folder id: %q", doc.MainParams.FolderID)
	}
	if doc.MainParams.OSType != "windows" {
		t.Fatalf("unexpected os type: %q", doc.MainParams.OSType)
	}
	if doc.MainParams.Notes != "from-template" {
		t.Fatalf("unexpected notes: %q", doc.MainParams.Notes)
	}
	if doc.MainParams.Parameters == nil || doc.MainParams.Parameters.Storage == nil {
		t.Fatal("expected storage settings in parsed template")
	}
	if doc.MainParams.Parameters.Storage.IsLocal {
		t.Fatal("expected parsed template to preserve is_local=false")
	}
	if got := len(doc.MainParams.Parameters.CustomStartURLs); got != 1 {
		t.Fatalf("expected one custom start url, got %d", got)
	}
	if got := doc.MainParams.Parameters.CustomStartURLs[0]; got != "https://example.com" {
		t.Fatalf("unexpected custom start url: %q", got)
	}
}

func TestLoadProfileTemplateFallsBackToMetaInfoWhenDownloadedBodyIsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty-template.json")
	if err := os.WriteFile(path, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	doc, err := loadProfileTemplate(`{
		"name": "Meta Template",
		"mainParams": {
			"name": "Meta Profile",
			"browser_type": "mimic",
			"folder_id": "folder-meta",
			"os_type": "windows",
			"notes": "meta-info-template",
			"parameters": {
				"storage": {
					"is_local": true
				}
			}
		}
	}`, path)
	if err != nil {
		t.Fatalf("loadProfileTemplate returned error: %v", err)
	}

	if doc == nil {
		t.Fatal("expected parsed template document")
	}
	if doc.Name != "Meta Template" {
		t.Fatalf("unexpected template name: %q", doc.Name)
	}
	if doc.MainParams.Name != "Meta Profile" {
		t.Fatalf("unexpected mainParams.name: %q", doc.MainParams.Name)
	}
	if doc.MainParams.FolderID != "folder-meta" {
		t.Fatalf("unexpected folder id: %q", doc.MainParams.FolderID)
	}
	if doc.MainParams.Notes != "meta-info-template" {
		t.Fatalf("unexpected notes: %q", doc.MainParams.Notes)
	}
	if doc.MainParams.Parameters == nil || doc.MainParams.Parameters.Storage == nil {
		t.Fatal("expected storage settings from meta_info")
	}
	if !doc.MainParams.Parameters.Storage.IsLocal {
		t.Fatal("expected meta_info fallback to preserve is_local=true")
	}
}

func TestLoadProfileTemplateRejectsEmptySources(t *testing.T) {
	_, err := loadProfileTemplate("", "")
	if err == nil {
		t.Fatal("expected loadProfileTemplate to fail when both path and meta_info are empty")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "empty") && !strings.Contains(strings.ToLower(err.Error()), "path") {
		t.Fatalf("expected empty source error, got %v", err)
	}
}

func TestLoadProfileTemplateRejectsInvalidMetaInfoWhenBodyUnavailable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken-template.json")
	if err := os.WriteFile(path, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	_, err := loadProfileTemplate(`{"name":"broken","mainParams":`, path)
	if err == nil {
		t.Fatal("expected loadProfileTemplate to reject invalid meta_info")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "meta_info") {
		t.Fatalf("expected meta_info decode error, got %v", err)
	}
}

func TestBuildCreateProfileRequestFromTemplateUsesExplicitOverrides(t *testing.T) {
	doc := &profileTemplateDocument{
		Name: "Template A",
		MainParams: mlx.CreateProfileRequest{
			Name:        "Template Profile",
			BrowserType: "mimic",
			FolderID:    "template-folder",
			OSType:      "windows",
			Notes:       "template-note",
			Parameters: &mlx.ProfileParameters{
				Storage: &mlx.Storage{
					IsLocal: false,
				},
				CustomStartURLs: []string{"https://one.example", "https://two.example"},
			},
		},
	}

	localOverride := false
	req, err := buildCreateProfileRequestFromTemplate(doc, "CLI Profile", "folder-override", &localOverride)
	if err != nil {
		t.Fatalf("buildCreateProfileRequestFromTemplate returned error: %v", err)
	}
	if req == nil {
		t.Fatal("expected create profile request")
	}
	if req.Name != "CLI Profile" {
		t.Fatalf("unexpected profile name: %q", req.Name)
	}
	if req.FolderID != "folder-override" {
		t.Fatalf("unexpected folder id: %q", req.FolderID)
	}
	if req.BrowserType != "mimic" {
		t.Fatalf("unexpected browser type: %q", req.BrowserType)
	}
	if req.OSType != "windows" {
		t.Fatalf("unexpected os type: %q", req.OSType)
	}
	if req.Notes != "template-note" {
		t.Fatalf("unexpected notes: %q", req.Notes)
	}
	if req.Parameters == nil || req.Parameters.Storage == nil {
		t.Fatal("expected parameters and storage to be present")
	}
	if req.Parameters.Storage.IsLocal {
		t.Fatal("expected local flag to remain false")
	}
	if got := len(req.Parameters.CustomStartURLs); got != 2 {
		t.Fatalf("expected custom start urls to be preserved, got %d", got)
	}
}

func TestBuildCreateProfileRequestFromTemplateFallsBackToTemplateNames(t *testing.T) {
	doc := &profileTemplateDocument{
		Name: "Template Fallback Name",
		MainParams: mlx.CreateProfileRequest{
			BrowserType: "mimic",
			FolderID:    "folder-1",
			OSType:      "windows",
		},
	}

	req, err := buildCreateProfileRequestFromTemplate(doc, "", "", nil)
	if err != nil {
		t.Fatalf("buildCreateProfileRequestFromTemplate returned error: %v", err)
	}
	if req.Name != "Template Fallback Name" {
		t.Fatalf("expected fallback to document name, got %q", req.Name)
	}
	if req.FolderID != "folder-1" {
		t.Fatalf("expected template folder id to be preserved, got %q", req.FolderID)
	}
}

func TestBuildCreateProfileRequestFromTemplateAppliesLocalOverrideAndInitializesParameters(t *testing.T) {
	doc := &profileTemplateDocument{
		Name: "Template B",
		MainParams: mlx.CreateProfileRequest{
			Name:        "Template Profile",
			BrowserType: "mimic",
			FolderID:    "folder-local",
			OSType:      "windows",
		},
	}

	localOverride := true
	req, err := buildCreateProfileRequestFromTemplate(doc, "", "", &localOverride)
	if err != nil {
		t.Fatalf("buildCreateProfileRequestFromTemplate returned error: %v", err)
	}
	if req.Parameters == nil {
		t.Fatal("expected parameters to be initialized")
	}
	if req.Parameters.Storage == nil {
		t.Fatal("expected storage to be initialized")
	}
	if !req.Parameters.Storage.IsLocal {
		t.Fatal("expected local override to set is_local=true")
	}
}

func TestBuildCreateProfileRequestFromTemplatePreservesTemplateLocalSettingWithoutOverride(t *testing.T) {
	doc := &profileTemplateDocument{
		Name: "Template Local",
		MainParams: mlx.CreateProfileRequest{
			Name:        "Template Local",
			BrowserType: "mimic",
			FolderID:    "folder-local",
			OSType:      "windows",
			Parameters: &mlx.ProfileParameters{
				Storage: &mlx.Storage{IsLocal: true},
			},
		},
	}

	req, err := buildCreateProfileRequestFromTemplate(doc, "", "", nil)
	if err != nil {
		t.Fatalf("buildCreateProfileRequestFromTemplate returned error: %v", err)
	}
	if req.Parameters == nil || req.Parameters.Storage == nil {
		t.Fatal("expected template storage settings to be preserved")
	}
	if !req.Parameters.Storage.IsLocal {
		t.Fatal("expected template is_local=true to remain unchanged without an explicit override")
	}
}

func TestBuildCreateProfileRequestFromTemplateRejectsMissingName(t *testing.T) {
	doc := &profileTemplateDocument{
		MainParams: mlx.CreateProfileRequest{
			BrowserType: "mimic",
			FolderID:    "folder-1",
			OSType:      "windows",
		},
	}

	_, err := buildCreateProfileRequestFromTemplate(doc, "", "", nil)
	if err == nil {
		t.Fatal("expected buildCreateProfileRequestFromTemplate to fail without a usable name")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "name") {
		t.Fatalf("expected name-related error, got %v", err)
	}
}

func TestBuildCreateProfileRequestFromTemplateRejectsMissingFolderID(t *testing.T) {
	doc := &profileTemplateDocument{
		Name: "Template C",
		MainParams: mlx.CreateProfileRequest{
			Name:        "Template Profile",
			BrowserType: "mimic",
			OSType:      "windows",
		},
	}

	_, err := buildCreateProfileRequestFromTemplate(doc, "", "", nil)
	if err == nil {
		t.Fatal("expected buildCreateProfileRequestFromTemplate to fail without a folder id")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "folder") {
		t.Fatalf("expected folder-related error, got %v", err)
	}
}

func TestBuildCreateProfileRequestFromTemplateRejectsNilDocument(t *testing.T) {
	_, err := buildCreateProfileRequestFromTemplate(nil, "Demo", "folder-1", nil)
	if err == nil {
		t.Fatal("expected buildCreateProfileRequestFromTemplate to reject a nil document")
	}
}
