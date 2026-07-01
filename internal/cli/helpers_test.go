package cli

import "testing"

func TestDefaultProfileFlagsSetsMasking(t *testing.T) {
	f := defaultProfileFlags()
	if f == nil {
		t.Fatal("expected non-nil flags")
	}
	if f.LocalizationMasking != "custom" || f.TimezoneMasking != "custom" ||
		f.ScreenMasking != "custom" || f.ProxyMasking != "custom" {
		t.Fatalf("expected custom masking, got %+v", f)
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
	if req.Parameters.Flags == nil || req.Parameters.Flags.LocalizationMasking != "custom" {
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
