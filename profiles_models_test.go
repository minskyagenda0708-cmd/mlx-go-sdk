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
