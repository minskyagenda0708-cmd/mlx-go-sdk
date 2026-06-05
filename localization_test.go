package mlx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/minskyagenda0708-cmd/mlx-go-sdk/internal/testutil"
)

// ── LocaleForCountry ─────────────────────────────────────────────

func TestLocaleForCountry_KnownCountries(t *testing.T) {
	cases := []struct {
		code       string
		wantLang   string
		wantLocale string
		wantAccept string
		wantZone   string
	}{
		{"US", "en-US", "en-US", "en-US,en;q=0.9", "America/New_York"},
		{"GB", "en-GB", "en-GB", "en-GB,en;q=0.9,en-US;q=0.5", "Europe/London"},
		{"DE", "de-DE", "de-DE", "de-DE,de;q=0.9,en-US;q=0.5,en;q=0.3", "Europe/Berlin"},
		{"FR", "fr-FR", "fr-FR", "fr-FR,fr;q=0.9,en-US;q=0.5,en;q=0.3", "Europe/Paris"},
		{"JP", "ja-JP", "ja-JP", "ja-JP,ja;q=0.9,en-US;q=0.5,en;q=0.3", "Asia/Tokyo"},
		{"BR", "pt-BR", "pt-BR", "pt-BR,pt;q=0.9,en-US;q=0.5,en;q=0.3", "America/Sao_Paulo"},
		{"RU", "ru-RU", "ru-RU", "ru-RU,ru;q=0.9,en-US;q=0.5,en;q=0.3", "Europe/Moscow"},
		{"UA", "uk-UA", "uk-UA", "uk-UA,uk;q=0.9,ru;q=0.7,en-US;q=0.5,en;q=0.3", "Europe/Kyiv"},
		{"CN", "zh-CN", "zh-CN", "zh-CN,zh;q=0.9,en-US;q=0.5,en;q=0.3", "Asia/Shanghai"},
		{"IN", "hi-IN", "hi-IN", "hi-IN,hi;q=0.9,en-US;q=0.5,en;q=0.3", "Asia/Kolkata"},
	}

	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			lp := LocaleForCountry(tc.code)
			if lp == nil {
				t.Fatalf("LocaleForCountry(%q) returned nil", tc.code)
			}
			if lp.Localization.Languages != tc.wantLang {
				t.Errorf("Languages: got %q, want %q", lp.Localization.Languages, tc.wantLang)
			}
			if lp.Localization.Locale != tc.wantLocale {
				t.Errorf("Locale: got %q, want %q", lp.Localization.Locale, tc.wantLocale)
			}
			if lp.Localization.AcceptLanguages != tc.wantAccept {
				t.Errorf("AcceptLanguages: got %q, want %q", lp.Localization.AcceptLanguages, tc.wantAccept)
			}
			if lp.Timezone.Zone != tc.wantZone {
				t.Errorf("Zone: got %q, want %q", lp.Timezone.Zone, tc.wantZone)
			}
		})
	}
}

func TestLocaleForCountry_CaseInsensitive(t *testing.T) {
	for _, input := range []string{"de", "De", "dE", "DE"} {
		lp := LocaleForCountry(input)
		if lp.Localization.Languages != "de-DE" {
			t.Errorf("LocaleForCountry(%q).Languages = %q, want %q", input, lp.Localization.Languages, "de-DE")
		}
	}
}

func TestLocaleForCountry_WhitespaceTrimmed(t *testing.T) {
	lp := LocaleForCountry("  DE  ")
	if lp.Localization.Languages != "de-DE" {
		t.Errorf("LocaleForCountry(%q).Languages = %q, want %q", "  DE  ", lp.Localization.Languages, "de-DE")
	}
}

func TestLocaleForCountry_UnknownFallback(t *testing.T) {
	lp := LocaleForCountry("XX")
	if lp.Localization.Languages != "en-US" {
		t.Errorf("LocaleForCountry(%q).Languages = %q, want fallback en-US", "XX", lp.Localization.Languages)
	}
	if lp.Timezone.Zone != "America/New_York" {
		t.Errorf("LocaleForCountry(%q).Zone = %q, want fallback America/New_York", "XX", lp.Timezone.Zone)
	}
}

func TestLocaleForCountry_EmptyString(t *testing.T) {
	lp := LocaleForCountry("")
	if lp.Localization.Languages != "en-US" {
		t.Errorf("LocaleForCountry(%q).Languages = %q, want fallback en-US", "", lp.Localization.Languages)
	}
}

func TestLocaleForCountry_TableCoverage(t *testing.T) {
	expected := []string{
		"US", "CA", "BR", "MX", "AR", "CO", "CL", "AU", "NZ",
		"GB", "IE", "DE", "AT", "CH", "FR", "NL", "BE", "ES", "PT", "IT",
		"SE", "NO", "DK", "FI",
		"PL", "CZ", "RO", "BG", "HR", "HU", "SK", "RU", "UA",
		"JP", "KR", "CN", "TW",
		"TH", "VN", "ID", "PH", "MY", "SG",
		"IN", "TR", "SA", "AE", "IL",
		"ZA", "NG", "EG",
	}
	if len(localeTable) < len(expected) {
		t.Errorf("localeTable has %d entries, expected at least %d", len(localeTable), len(expected))
	}
	for _, cc := range expected {
		if _, ok := localeTable[cc]; !ok {
			t.Errorf("localeTable missing entry for %q", cc)
		}
	}
}

// ── PatchProfileForProxy ─────────────────────────────────────────

func TestPatchProfileForProxy_NilProxy(t *testing.T) {
	client, err := New(WithToken("test-token"))
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	_, _, err = client.PatchProfileForProxy(context.Background(), "pid-1", nil)
	if err == nil {
		t.Fatal("expected error for nil proxy, got nil")
	}
	if _, ok := err.(*ArgError); !ok {
		t.Fatalf("expected ArgError, got %T: %v", err, err)
	}
}

func TestPatchProfileForProxy_EmptyProfileID(t *testing.T) {
	client, err := New(WithToken("test-token"))
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	_, _, err = client.PatchProfileForProxy(context.Background(), "", &Proxy{Host: "1.2.3.4", Port: 8080})
	if err == nil {
		t.Fatal("expected error for empty profileID, got nil")
	}
	if _, ok := err.(*ArgError); !ok {
		t.Fatalf("expected ArgError, got %T: %v", err, err)
	}
}

func TestPatchProfileForProxy_CountryFromProxy(t *testing.T) {
	var gotBody []byte

	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/profile/partial_update" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotBody, _ = io.ReadAll(r.Body)
		fmt.Fprint(w, `{"status":{"http_code":200,"message":"OK"}}`)
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	proxy := &Proxy{
		Host:     "de.proxy.example.com",
		Port:     8080,
		Type:     "http",
		Username: "user",
		Password: "pass",
		Country:  "DE",
	}

	_, _, err = client.PatchProfileForProxy(context.Background(), "profile-123", proxy)
	if err != nil {
		t.Fatalf("PatchProfileForProxy returned error: %v", err)
	}

	var req PatchProfileRequest
	if err := json.Unmarshal(gotBody, &req); err != nil {
		t.Fatalf("failed to unmarshal request body: %v", err)
	}

	// Verify profile ID
	if req.ProfileID != "profile-123" {
		t.Errorf("ProfileID: got %q, want %q", req.ProfileID, "profile-123")
	}

	// Verify proxy is forwarded
	if req.Proxy == nil || req.Proxy.Host != "de.proxy.example.com" {
		t.Errorf("Proxy.Host: got %v, want de.proxy.example.com", req.Proxy)
	}

	// Verify flags
	if req.Parameters == nil || req.Parameters.Flags == nil {
		t.Fatal("Parameters.Flags is nil")
	}
	flags := req.Parameters.Flags
	if flags.LocalizationMasking != "custom" {
		t.Errorf("LocalizationMasking: got %q, want %q", flags.LocalizationMasking, "custom")
	}
	if flags.TimezoneMasking != "custom" {
		t.Errorf("TimezoneMasking: got %q, want %q", flags.TimezoneMasking, "custom")
	}
	if flags.ProxyMasking != "custom" {
		t.Errorf("ProxyMasking: got %q, want %q", flags.ProxyMasking, "custom")
	}
	if flags.ScreenMasking != "custom" {
		t.Errorf("ScreenMasking: got %q, want %q", flags.ScreenMasking, "custom")
	}

	// Verify localization matches Germany
	loc := req.Parameters.Fingerprint.Localization
	if loc.Languages != "de-DE" {
		t.Errorf("Languages: got %q, want %q", loc.Languages, "de-DE")
	}
	if loc.Locale != "de-DE" {
		t.Errorf("Locale: got %q, want %q", loc.Locale, "de-DE")
	}
	if !strings.Contains(loc.AcceptLanguages, "de-DE") {
		t.Errorf("AcceptLanguages: got %q, want to contain de-DE", loc.AcceptLanguages)
	}

	// Verify timezone
	tz := req.Parameters.Fingerprint.Timezone
	if tz.Zone != "Europe/Berlin" {
		t.Errorf("Zone: got %q, want %q", tz.Zone, "Europe/Berlin")
	}

	// Verify screen fingerprint is within default bounds (1920×1080 exact)
	screen := req.Parameters.Fingerprint.Screen
	if screen == nil {
		t.Fatal("Screen fingerprint is nil")
	}
	if screen.Width != 1920 {
		t.Errorf("Screen.Width: got %d, want 1920", screen.Width)
	}
	if screen.Height != 1080 {
		t.Errorf("Screen.Height: got %d, want 1080", screen.Height)
	}
	if screen.PixelRatio != 1.0 {
		t.Errorf("PixelRatio: got %f, want 1.0", screen.PixelRatio)
	}

	// Verify CMDParams (--lang and --window-size)
	cmd := req.Parameters.Fingerprint.CMDParams
	if cmd == nil || len(cmd.Params) == 0 {
		t.Fatal("CMDParams is nil or empty")
	}
	hasLang := false
	hasWindowSize := false
	for _, p := range cmd.Params {
		if p.Flag == "--lang" {
			hasLang = true
			if p.Value != "de-DE" {
				t.Errorf("--lang value: got %q, want %q", p.Value, "de-DE")
			}
		}
		if p.Flag == "--window-size" {
			hasWindowSize = true
			// Window size must match the screen fingerprint
			expected := fmt.Sprintf("%d,%d", screen.Width, screen.Height)
			if p.Value != expected {
				t.Errorf("--window-size value: got %q, want %q", p.Value, expected)
			}
		}
	}
	if !hasLang {
		t.Error("missing --lang in CMDParams")
	}
	if !hasWindowSize {
		t.Error("missing --window-size in CMDParams")
	}
}

func TestPatchProfileForProxy_WithOptionsOverridesScreenBounds(t *testing.T) {
	var gotBody []byte

	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		fmt.Fprint(w, `{"status":{"http_code":200,"message":"OK"}}`)
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	proxy := &Proxy{Host: "jp.proxy.example.com", Port: 8080, Type: "http", Country: "JP"}

	// Force exact resolution: only 1536×864 fits
	opts := PatchProfileForProxyOptions{
		MinScreenWidth:  1536,
		MinScreenHeight: 864,
		MaxScreenWidth:  1536,
		MaxScreenHeight: 864,
	}

	_, _, err = client.PatchProfileForProxyWithOptions(context.Background(), "profile-jp", proxy, opts)
	if err != nil {
		t.Fatalf("PatchProfileForProxyWithOptions returned error: %v", err)
	}

	var req PatchProfileRequest
	if err := json.Unmarshal(gotBody, &req); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	screen := req.Parameters.Fingerprint.Screen
	if screen == nil {
		t.Fatal("Screen is nil")
	}
	if screen.Width != 1536 || screen.Height != 864 {
		t.Errorf("Screen: got %dx%d, want 1536×864", screen.Width, screen.Height)
	}

	// --lang should be Japanese
	for _, p := range req.Parameters.Fingerprint.CMDParams.Params {
		if p.Flag == "--lang" && p.Value != "ja-JP" {
			t.Errorf("--lang: got %q, want ja-JP", p.Value)
		}
	}
}

func TestPatchProfileForProxy_CountryViaValidateProxy(t *testing.T) {
	validateCalled := false
	profileCalled := false

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/proxy/validate", func(w http.ResponseWriter, r *http.Request) {
		validateCalled = true
		fmt.Fprint(w, `{"status":{"http_code":200,"message":"OK"},"data":{"country_code":"JP","timezone":"Asia/Tokyo","ip":"1.2.3.4","accuracy":1.0,"latitude":35.68,"longitude":139.69,"altitude":0}}`)
	})
	mux.HandleFunc("/profile/partial_update", func(w http.ResponseWriter, r *http.Request) {
		profileCalled = true
		fmt.Fprint(w, `{"status":{"http_code":200,"message":"OK"}}`)
	})

	// Single httptest server handles both launcher and API routes.
	server, httpClient := testutil.NewServer(t, mux.ServeHTTP)

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithBaseURL(server.URL),
		WithLauncherURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	proxy := &Proxy{
		Host:     "jp.proxy.example.com",
		Port:     8080,
		Type:     "http",
		Username: "user",
		Password: "pass",
		// Country intentionally empty to trigger ValidateProxy
	}

	_, _, err = client.PatchProfileForProxy(context.Background(), "profile-jp", proxy)
	if err != nil {
		t.Fatalf("PatchProfileForProxy returned error: %v", err)
	}

	if !validateCalled {
		t.Error("expected ValidateProxy to be called, but it was not")
	}
	if !profileCalled {
		t.Error("expected Patch to be called, but it was not")
	}
}

// ── resolveProxyCountry ───────────────────────────────────────────

func TestResolveProxyCountry_FromProxyField(t *testing.T) {
	client, err := New(WithToken("test-token"))
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	cc, err := client.resolveProxyCountry(context.Background(), &Proxy{Country: "FR"})
	if err != nil {
		t.Fatalf("resolveProxyCountry returned error: %v", err)
	}
	if cc != "FR" {
		t.Errorf("got %q, want %q", cc, "FR")
	}
}

func TestResolveProxyCountry_FallbackUS(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("unexpected request to server")
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithLauncherURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	cc, err := client.resolveProxyCountry(context.Background(), &Proxy{})
	if err != nil {
		t.Fatalf("resolveProxyCountry returned error: %v", err)
	}
	if cc != "US" {
		t.Errorf("got %q, want fallback %q", cc, "US")
	}
}

// ── pickScreenResolution ──────────────────────────────────────────

func TestPickScreenResolution_DefaultBounds(t *testing.T) {
	opts := PatchProfileForProxyOptions{}
	opts.defaults()

	// With defaults min=max=1920×1080, always get exactly that
	for i := 0; i < 100; i++ {
		s := pickScreenResolution(opts)
		if s.Width != 1920 || s.Height != 1080 {
			t.Errorf("pick %d: got %dx%d, want 1920×1080", i, s.Width, s.Height)
		}
		if s.PixelRatio != 1.0 {
			t.Errorf("pick %d: PixelRatio %f, want 1.0", i, s.PixelRatio)
		}
	}
}

func TestPickScreenResolution_ExactBounds(t *testing.T) {
	opts := PatchProfileForProxyOptions{
		MinScreenWidth:  1536,
		MinScreenHeight: 864,
		MaxScreenWidth:  1536,
		MaxScreenHeight: 864,
	}
	s := pickScreenResolution(opts)
	if s.Width != 1536 || s.Height != 864 {
		t.Errorf("got %dx%d, want 1536×864", s.Width, s.Height)
	}
}

func TestPickScreenResolution_NoMatchFallsBack(t *testing.T) {
	opts := PatchProfileForProxyOptions{
		MinScreenWidth:  9999,
		MinScreenHeight: 9999,
		MaxScreenWidth:  9999,
		MaxScreenHeight: 9999,
	}
	s := pickScreenResolution(opts)
	// Should fall back to the min bounds
	if s.Width != 9999 || s.Height != 9999 {
		t.Errorf("got %dx%d, want fallback 9999×9999", s.Width, s.Height)
	}
}

func TestPickScreenResolution_Variety(t *testing.T) {
	opts := PatchProfileForProxyOptions{
		MinScreenWidth:  1366,
		MinScreenHeight: 768,
		MaxScreenWidth:  2560,
		MaxScreenHeight: 1440,
	}
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		s := pickScreenResolution(opts)
		seen[fmt.Sprintf("%dx%d", s.Width, s.Height)] = true
	}
	// With range 1366-2560 wide and 768-1440 tall, the pool includes
	// many resolutions. Should see at least 5 different ones.
	if len(seen) < 5 {
		t.Errorf("expected variety in screen picks, only got %d distinct resolutions: %v", len(seen), seen)
	}
}

// ── PatchProfileForProxyOptions.defaults ───────────────────────────

func TestPatchProfileForProxyOptions_Defaults(t *testing.T) {
	opts := PatchProfileForProxyOptions{}
	opts.defaults()
	if opts.MinScreenWidth != 1920 {
		t.Errorf("MinScreenWidth: got %d, want 1920", opts.MinScreenWidth)
	}
	if opts.MinScreenHeight != 1080 {
		t.Errorf("MinScreenHeight: got %d, want 1080", opts.MinScreenHeight)
	}
	if opts.MaxScreenWidth != 1920 {
		t.Errorf("MaxScreenWidth: got %d, want 1920", opts.MaxScreenWidth)
	}
	if opts.MaxScreenHeight != 1080 {
		t.Errorf("MaxScreenHeight: got %d, want 1080", opts.MaxScreenHeight)
	}
}

func TestPatchProfileForProxyOptions_PartialOverride(t *testing.T) {
	opts := PatchProfileForProxyOptions{MaxScreenWidth: 2560, MaxScreenHeight: 1440}
	opts.defaults()
	if opts.MinScreenWidth != 1920 {
		t.Errorf("MinScreenWidth: got %d, want 1920 (default)", opts.MinScreenWidth)
	}
	if opts.MaxScreenWidth != 2560 {
		t.Errorf("MaxScreenWidth: got %d, want 2560 (overridden)", opts.MaxScreenWidth)
	}
	if opts.MaxScreenHeight != 1440 {
		t.Errorf("MaxScreenHeight: got %d, want 1440 (overridden)", opts.MaxScreenHeight)
	}
}
