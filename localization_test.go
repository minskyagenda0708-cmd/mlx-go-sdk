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
	// No host/port => can't validate, should fallback to US
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Should not be called
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
