package mlx

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCreateProfileRequestMarshalsTypedParameters(t *testing.T) {
	req := &CreateProfileRequest{
		Name:        "demo",
		BrowserType: "mimic",
		FolderID:    "folder-1",
		OSType:      "windows",
		Parameters: &ProfileParameters{
			Flags: &ProfileFlags{
				AudioMasking:    "mask",
				WebRTCMasking:   "mask",
				StartupBehavior: "recover",
			},
			Storage: &Storage{
				IsLocal:           false,
				SaveServiceWorker: true,
			},
			Fingerprint: &Fingerprint{
				Navigator: &NavigatorFingerprint{
					UserAgent: "test-agent",
					Platform:  "Win32",
				},
				Screen: &ScreenFingerprint{
					Width:  1920,
					Height: 1080,
				},
			},
		},
	}

	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	text := string(payload)
	checks := []string{
		`"audio_masking":"mask"`,
		`"webrtc_masking":"mask"`,
		`"startup_behavior":"recover"`,
		`"user_agent":"test-agent"`,
		`"width":1920`,
	}
	for _, check := range checks {
		if !strings.Contains(text, check) {
			t.Fatalf("expected payload to contain %s, got %s", check, text)
		}
	}
}

func TestSearchProfilesResponseUnmarshalsExtendedFields(t *testing.T) {
	payload := []byte(`{"status":{"http_code":200,"message":"Search profile successfully result"},"data":{"profiles":[{"id":"profile-1","name":"Demo","folder_id":"folder-1","abp_status":false,"browser_type":"mimic","os_type":"windows","core_version":137,"notes":"hello","created_by":"me@example.com","created_at":"2026-04-20T00:00:00Z","in_use_by":"","locked_by":"marvin@example.com","last_launched_at":"2026-04-20T00:01:00Z","last_launched_by":"marvin@example.com","last_launched_on":"localhost","updated_at":"2026-04-20T00:02:00Z","password_protected":false,"is_local":false}],"total_count":1}}`)

	var resp SearchProfilesResponse
	if err := json.Unmarshal(payload, &resp); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if resp.Data.Profiles[0].LastLaunchedOn != "localhost" {
		t.Fatalf("unexpected last launched on: %s", resp.Data.Profiles[0].LastLaunchedOn)
	}
	if resp.Data.Profiles[0].LockedBy != "marvin@example.com" {
		t.Fatalf("unexpected locked by: %s", resp.Data.Profiles[0].LockedBy)
	}
}

func TestProfileMetasResponseUnmarshalsExtendedFields(t *testing.T) {
	payload := []byte(`{"status":{"http_code":200,"message":""},"data":{"profiles":[{"id":"profile-1","name":"Demo","notes":"hello","browser_type":"stealthfox","core_version":140,"is_auto_update":true,"is_local":false,"os_type":"windows","folder_id":"folder-1","workspace_id":"workspace-1","created_at":"2026-04-20T00:00:00Z","created_by":"api_test@multilogin.com","in_use_by":"","last_launched_at":"2026-04-20T00:01:00Z","last_launched_by":"marvin@example.com","last_launched_on":"localhost","last_update_at":"2026-04-20T00:02:00Z","last_updated_by":"marvin@example.com","removed_at":"0001-01-01T00:00:00Z","removed_by":"","status":"","parameters":{"storage":{"is_local":false}}}]}}`)

	var resp ProfileMetasResponse
	if err := json.Unmarshal(payload, &resp); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if resp.Data.Profiles[0].LastLaunchedOn != "localhost" {
		t.Fatalf("unexpected last launched on: %s", resp.Data.Profiles[0].LastLaunchedOn)
	}
	if resp.Data.Profiles[0].RemovedAt == "" {
		t.Fatal("expected removed_at to unmarshal")
	}
}
