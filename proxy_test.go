package mlx

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/bath0ry/mlx-go-sdk/internal/testutil"
)

func TestGenerateProxyResponseUnmarshalSingleString(t *testing.T) {
	var resp GenerateProxyResponse
	payload := []byte(`{"status":200,"data":"gate.multilogin.com:1080:user-country-us-region-new_jersey-city-east_brunswick-sid-demo:pass"}`)
	if err := resp.UnmarshalJSON(payload); err != nil {
		t.Fatalf("UnmarshalJSON returned error: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected single proxy entry, got %#v", resp.Data)
	}
}

func TestParseGeneratedProxyConnectionExtractsAffinity(t *testing.T) {
	raw := "gate.multilogin.com:1080:2235470499_bc98e4f8_7cf1_4d39_8409_7599fa5eb4e8_multilogin_com-country-us-region-new_jersey-city-east_brunswick-sid-demo-filter-medium:secret"
	conn, err := ParseGeneratedProxyConnection(raw, ProxyProtocolSOCKS5)
	if err != nil {
		t.Fatalf("ParseGeneratedProxyConnection returned error: %v", err)
	}
	if conn.Host != "gate.multilogin.com" || conn.Port != 1080 {
		t.Fatalf("unexpected endpoint: %#v", conn)
	}
	if conn.Country != "us" || conn.Region != "new_jersey" || conn.City != "east_brunswick" {
		t.Fatalf("unexpected affinity fields: %#v", conn)
	}
	if conn.SessionID != "demo" {
		t.Fatalf("unexpected session id: %s", conn.SessionID)
	}
	if conn.RetentionKey != "2235470499_bc98e4f8" {
		t.Fatalf("unexpected retention key: %s", conn.RetentionKey)
	}
}

func TestBuildProfileProxyFromGenerated(t *testing.T) {
	proxy := BuildProfileProxyFromGenerated(&GeneratedProxyConnection{
		Raw:             "gate.multilogin.com:1080:user-country-us:secret",
		Protocol:        ProxyProtocolSOCKS5,
		Host:            "gate.multilogin.com",
		Port:            1080,
		Username:        "user-country-us",
		Password:        "secret",
		Country:         "us",
		RetentionKey:    "acct_key",
		RetentionSecret: "secret",
	})
	if proxy.Type != "socks5" || proxy.Country != "us" || proxy.RetentionKey != "acct_key" {
		t.Fatalf("unexpected profile proxy: %#v", proxy)
	}
}

func TestProxiesGenerateUsesStrictHeaderAndParsesLiveArray(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/proxy/connection_url" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("X-Strict-Mode"); got != "true" {
			t.Fatalf("expected strict header, got %q", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		text := string(body)
		checks := []string{
			`"country":"us"`,
			`"protocol":"socks5"`,
			`"sessionType":"sticky"`,
			`"region":"new_jersey"`,
			`"city":"east_brunswick"`,
		}
		for _, check := range checks {
			if !strings.Contains(text, check) {
				t.Fatalf("expected request body to contain %s, got %s", check, text)
			}
		}
		fmt.Fprint(w, `{"status":200,"data":["gate.multilogin.com:1080:2235470499_bc98e4f8_multilogin_com-country-us-region-new_jersey-city-east_brunswick-sid-demo-filter-medium:secret"]}`)
	})

	client, err := New(WithToken("test-token"), WithHTTPClient(httpClient), WithProxyURL(server.URL))
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, _, err := client.Proxies.Generate(context.Background(), &GenerateProxyRequest{
		Country:     "us",
		Protocol:    ProxyProtocolSOCKS5,
		SessionType: ProxySessionSticky,
		Region:      "new_jersey",
		City:        "east_brunswick",
		StrictMode:  true,
	})
	if err != nil {
		t.Fatalf("Proxies.Generate returned error: %v", err)
	}
	if len(resp.Parsed) != 1 || resp.Parsed[0].Country != "us" {
		t.Fatalf("unexpected parsed response: %#v", resp.Parsed)
	}
}

func TestProxiesGetUsage(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/user" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		fmt.Fprint(w, `{"traffic":1501700871,"billingId":"2235470499"}`)
	})

	client, err := New(WithToken("test-token"), WithHTTPClient(httpClient), WithProxyURL(server.URL))
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, _, err := client.Proxies.GetUsage(context.Background())
	if err != nil {
		t.Fatalf("Proxies.GetUsage returned error: %v", err)
	}
	if resp.BillingID != "2235470499" || resp.Traffic != 1501700871 {
		t.Fatalf("unexpected usage response: %#v", resp)
	}
}

func TestProxiesGenerateProfileProxyPrefersSOCKS5(t *testing.T) {
	step := 0
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch step {
		case 0:
			if r.Method != http.MethodGet || r.URL.Path != "/v1/user" {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			fmt.Fprint(w, `{"traffic":10,"billingId":"2235470499"}`)
		case 1:
			if r.Method != http.MethodPost || r.URL.Path != "/v1/proxy/connection_url" {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"protocol":"socks5"`) {
				t.Fatalf("expected socks5 protocol, got %s", string(body))
			}
			fmt.Fprint(w, `{"status":200,"data":["gate.multilogin.com:1080:user-country-us-sid-demo:secret"]}`)
		default:
			t.Fatalf("unexpected request step %d", step)
		}
		step++
	})

	client, err := New(WithToken("test-token"), WithHTTPClient(httpClient), WithProxyURL(server.URL))
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, err := client.Proxies.GenerateProfileProxy(context.Background(), &GenerateProfileProxyRequest{
		GenerateProxyRequest: GenerateProxyRequest{Country: "us"},
		PreferSOCKS5:         true,
		SaveTraffic:          true,
	})
	if err != nil {
		t.Fatalf("Proxies.GenerateProfileProxy returned error: %v", err)
	}
	if resp.ProfileProxy.Type != "socks5" || !resp.ProfileProxy.SaveTraffic {
		t.Fatalf("unexpected profile proxy: %#v", resp.ProfileProxy)
	}
}

func TestNormalizeGenerateProxyRequestDefaults(t *testing.T) {
	req, err := normalizeGenerateProxyRequest(&GenerateProxyRequest{})
	if err != nil {
		t.Fatalf("normalizeGenerateProxyRequest returned error: %v", err)
	}
	if req.Country != "any" || req.Protocol != ProxyProtocolSOCKS5 || req.SessionType != ProxySessionSticky || req.Count != 1 {
		t.Fatalf("unexpected normalized request: %#v", req)
	}
}

func TestNormalizeGenerateProxyRequestSetsRotatingTTL(t *testing.T) {
	req, err := normalizeGenerateProxyRequest(&GenerateProxyRequest{SessionType: ProxySessionRotating, Protocol: ProxyProtocolHTTP})
	if err != nil {
		t.Fatalf("normalizeGenerateProxyRequest returned error: %v", err)
	}
	if req.IPTTL != 86400 {
		t.Fatalf("expected default IPTTL for rotating proxy, got %d", req.IPTTL)
	}
}
