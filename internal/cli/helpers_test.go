package cli

import "testing"

func TestDefaultProfileFlagsSetsMasking(t *testing.T) {
	f := defaultProfileFlags()
	if f == nil {
		t.Fatal("expected non-nil flags")
	}
	// The API rejects "custom" masking values unless explicit custom
	// fingerprint values are also supplied (400 "wrong audio masking flag").
	// Flag-only creation must use API-valid enum values: mask/natural/disabled.
	valid := map[string]bool{"mask": true, "natural": true, "disabled": true, "prompt": true}
	checks := map[string]string{
		"audio_masking":        f.AudioMasking,
		"fonts_masking":        f.FontsMasking,
		"geolocation_masking":  f.GeolocationMasking,
		"graphics_masking":     f.GraphicsMasking,
		"graphics_noise":       f.GraphicsNoise,
		"localization_masking": f.LocalizationMasking,
		"media_devices_masking": f.MediaDevicesMasking,
		"navigator_masking":    f.NavigatorMasking,
		"ports_masking":        f.PortsMasking,
		"proxy_masking":        f.ProxyMasking,
		"screen_masking":       f.ScreenMasking,
		"timezone_masking":     f.TimezoneMasking,
		"webrtc_masking":       f.WebRTCMasking,
	}
	for name, val := range checks {
		if !valid[val] {
			t.Fatalf("%s must be an API-valid masking value (mask/natural/disabled), got %q", name, val)
		}
	}
	// audio masking specifically must be "natural" per the live-validated e2e reference.
	if f.AudioMasking != "natural" {
		t.Fatalf("expected audio_masking=natural, got %q", f.AudioMasking)
	}
}

func TestBuildCreateProfileRequestFromFlagsGermany(t *testing.T) {
	req, err := buildCreateProfileRequestFromFlags(createFromFlagsInput{
		Name:        "shop-de",
		BrowserType: "mimic",
		OSType:      "windows",
		Country:     "de",
		FolderID:    "folder-1",
		IsLocal:     true,
		Times:       1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Name != "shop-de" || req.BrowserType != "mimic" || req.OSType != "windows" || req.FolderID != "folder-1" {
		t.Fatalf("unexpected main params: %+v", req)
	}
	if req.Parameters == nil || req.Parameters.Fingerprint == nil {
		t.Fatal("expected parameters + fingerprint")
	}
	loc := req.Parameters.Fingerprint.Localization
	if loc == nil || loc.Locale != "de-DE" {
		t.Fatalf("expected de-DE locale, got %+v", loc)
	}
	tz := req.Parameters.Fingerprint.Timezone
	if tz == nil || tz.Zone != "Europe/Berlin" {
		t.Fatalf("expected Europe/Berlin, got %+v", tz)
	}
	scr := req.Parameters.Fingerprint.Screen
	if scr == nil || scr.Width < 1920 || scr.Height < 1080 {
		t.Fatalf("expected screen >= 1920x1080, got %+v", scr)
	}
	if req.Parameters.Storage == nil || !req.Parameters.Storage.IsLocal {
		t.Fatal("expected local storage")
	}
	if req.Parameters.Flags == nil || req.Parameters.Flags.LocalizationMasking != "mask" {
		t.Fatal("expected masking flags applied")
	}
}

func TestBuildCreateProfileRequestFromFlagsDefaultsToUSWhenNoCountry(t *testing.T) {
	req, err := buildCreateProfileRequestFromFlags(createFromFlagsInput{
		Name: "x", BrowserType: "mimic", OSType: "windows", FolderID: "f",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	loc := req.Parameters.Fingerprint.Localization
	if loc == nil || loc.Locale != "en-US" {
		t.Fatalf("expected en-US fallback, got %+v", loc)
	}
}

func TestBuildCreateProfileRequestFromFlagsRequiresName(t *testing.T) {
	_, err := buildCreateProfileRequestFromFlags(createFromFlagsInput{
		BrowserType: "mimic", OSType: "windows", FolderID: "f",
	})
	if err == nil {
		t.Fatal("expected error when name is empty")
	}
}
