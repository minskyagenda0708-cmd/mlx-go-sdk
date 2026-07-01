package mlx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// staticChecker is a test double proving the interface is satisfiable.
type staticChecker struct{ res ProxyCheckResult }

func (s staticChecker) Check(_ context.Context, _ *Proxy) ProxyCheckResult { return s.res }

func TestProxyCheckerInterfaceSatisfied(t *testing.T) {
	var c ProxyChecker = staticChecker{res: ProxyCheckResult{Alive: true, LatencyMs: 42, Target: "https://x"}}
	got := c.Check(context.Background(), &Proxy{Host: "h", Port: 1})
	if !got.Alive || got.LatencyMs != 42 || got.Target != "https://x" {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestProxyURLMapping(t *testing.T) {
	u, err := proxyURL(&Proxy{Type: "socks5", Host: "1.2.3.4", Port: 1080, Username: "u", Password: "p"})
	if err != nil {
		t.Fatalf("proxyURL error: %v", err)
	}
	if u.Scheme != "socks5" || u.Host != "1.2.3.4:1080" {
		t.Fatalf("unexpected url: %s", u.String())
	}
	if pw, _ := u.User.Password(); u.User.Username() != "u" || pw != "p" {
		t.Fatalf("unexpected userinfo: %s", u.User.String())
	}
}

func TestHTTPProxyCheckerMeasuresLiveTarget(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewHTTPProxyChecker(HTTPProxyCheckerConfig{
		Targets:          []string{srv.URL},
		PerTargetTimeout: 2 * time.Second,
	})
	// Direct-connection sentinel: Host "direct" tells the checker to skip the
	// proxy and dial the target directly (test-only escape used to exercise the
	// measurement path without a live proxy server).
	res := c.Check(context.Background(), &Proxy{Type: "direct", Host: "direct", Port: 1})
	if !res.Alive {
		t.Fatalf("expected alive, got err=%v", res.Err)
	}
	if res.LatencyMs < 0 {
		t.Fatalf("expected non-negative latency, got %d", res.LatencyMs)
	}
	if res.Target != srv.URL {
		t.Fatalf("expected target %s, got %s", srv.URL, res.Target)
	}
}

func TestHTTPProxyCheckerDeadTargetNotAlive(t *testing.T) {
	c := NewHTTPProxyChecker(HTTPProxyCheckerConfig{
		Targets:          []string{"http://127.0.0.1:0"},
		PerTargetTimeout: 200 * time.Millisecond,
	})
	res := c.Check(context.Background(), &Proxy{Type: "direct", Host: "direct", Port: 1})
	if res.Alive {
		t.Fatalf("expected not alive for unreachable target")
	}
	if res.Err == nil {
		t.Fatalf("expected an error for unreachable target")
	}
}
