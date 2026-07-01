package mlx

import (
	"context"
	"testing"
)

func TestSelectBestProxyPrefersUnderThreshold(t *testing.T) {
	cands := []proxyCandidate{
		{proxy: &Proxy{Host: "a"}, result: ProxyCheckResult{Alive: true, LatencyMs: 2500}},
		{proxy: &Proxy{Host: "b"}, result: ProxyCheckResult{Alive: true, LatencyMs: 1500}},
		{proxy: &Proxy{Host: "c"}, result: ProxyCheckResult{Alive: false}},
	}
	if got := selectBestProxy(cands, 2000, 3000); got != 1 {
		t.Fatalf("expected index 1 (b, 1500ms), got %d", got)
	}
}

func TestSelectBestProxyEscalatesToHardCap(t *testing.T) {
	cands := []proxyCandidate{
		{proxy: &Proxy{Host: "a"}, result: ProxyCheckResult{Alive: true, LatencyMs: 2800}},
		{proxy: &Proxy{Host: "b"}, result: ProxyCheckResult{Alive: true, LatencyMs: 2500}},
	}
	if got := selectBestProxy(cands, 2000, 3000); got != 1 {
		t.Fatalf("expected fastest within hard cap (index 1), got %d", got)
	}
}

func TestSelectBestProxyNoneWithinHardCap(t *testing.T) {
	cands := []proxyCandidate{
		{proxy: &Proxy{Host: "a"}, result: ProxyCheckResult{Alive: true, LatencyMs: 3500}},
		{proxy: &Proxy{Host: "b"}, result: ProxyCheckResult{Alive: false}},
	}
	if got := selectBestProxy(cands, 2000, 3000); got != -1 {
		t.Fatalf("expected -1 (none acceptable), got %d", got)
	}
}

// fakeGen returns preset proxies per (city/country) round.
type fakeGen struct {
	cityProxies    []*Proxy
	countryProxies []*Proxy
}

func (f *fakeGen) generate(_ context.Context, _, _, city string, _ int) ([]*Proxy, error) {
	if city != "" {
		return f.cityProxies, nil
	}
	return f.countryProxies, nil
}

// mapChecker returns a result per proxy Host.
type mapChecker struct{ byHost map[string]ProxyCheckResult }

func (m mapChecker) Check(_ context.Context, p *Proxy) ProxyCheckResult {
	if p == nil {
		return ProxyCheckResult{}
	}
	return m.byHost[p.Host]
}

func TestEnsureHealthyProxyKeepsHealthyCurrent(t *testing.T) {
	current := &Proxy{Host: "cur", Country: "DE", City: "Berlin"}
	chk := mapChecker{byHost: map[string]ProxyCheckResult{"cur": {Alive: true, LatencyMs: 800}}}
	got, changed, err := ensureHealthyProxy(context.Background(), current, &fakeGen{}, EnsureHealthyProxyOptions{Checker: chk})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed {
		t.Fatal("expected current proxy kept (changed=false)")
	}
	if got.Host != "cur" {
		t.Fatalf("expected cur, got %s", got.Host)
	}
}

func TestEnsureHealthyProxyReplacesFromCity(t *testing.T) {
	current := &Proxy{Host: "cur", Country: "DE", City: "Berlin"}
	gen := &fakeGen{cityProxies: []*Proxy{{Host: "c1", Country: "DE", City: "Berlin"}}}
	chk := mapChecker{byHost: map[string]ProxyCheckResult{
		"cur": {Alive: false},
		"c1":  {Alive: true, LatencyMs: 1200},
	}}
	got, changed, err := ensureHealthyProxy(context.Background(), current, gen, EnsureHealthyProxyOptions{Checker: chk})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed || got.Host != "c1" {
		t.Fatalf("expected replacement c1 (changed), got host=%s changed=%v", got.Host, changed)
	}
}

func TestEnsureHealthyProxyFallsBackToCountry(t *testing.T) {
	current := &Proxy{Host: "cur", Country: "DE", City: "Berlin"}
	gen := &fakeGen{
		cityProxies:    nil, // city round empty
		countryProxies: []*Proxy{{Host: "n1", Country: "DE"}},
	}
	chk := mapChecker{byHost: map[string]ProxyCheckResult{
		"cur": {Alive: false},
		"n1":  {Alive: true, LatencyMs: 1800},
	}}
	got, changed, err := ensureHealthyProxy(context.Background(), current, gen, EnsureHealthyProxyOptions{Checker: chk})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed || got.Host != "n1" {
		t.Fatalf("expected country replacement n1, got host=%s changed=%v", got.Host, changed)
	}
}

func TestServiceEnsureHealthyProxyKeepsHealthyCurrentNoGenerate(t *testing.T) {
	s := &ProxyServiceOp{} // client unused because current is healthy
	current := &Proxy{Host: "cur", Country: "DE", City: "Berlin"}
	chk := mapChecker{byHost: map[string]ProxyCheckResult{"cur": {Alive: true, LatencyMs: 500}}}
	got, changed, err := s.EnsureHealthyProxy(context.Background(), current, EnsureHealthyProfileProxyOptions{
		EnsureHealthyProxyOptions: EnsureHealthyProxyOptions{Checker: chk},
	})
	if err != nil || changed || got.Host != "cur" {
		t.Fatalf("expected healthy current kept, got host=%v changed=%v err=%v", got, changed, err)
	}
}

func TestEnsureHealthyProxyFailsClosedWhenNoneAlive(t *testing.T) {
	current := &Proxy{Host: "cur", Country: "DE", City: "Berlin"}
	gen := &fakeGen{
		cityProxies:    []*Proxy{{Host: "c1"}},
		countryProxies: []*Proxy{{Host: "n1"}},
	}
	chk := mapChecker{byHost: map[string]ProxyCheckResult{
		"cur": {Alive: false},
		"c1":  {Alive: false},
		"n1":  {Alive: true, LatencyMs: 5000}, // over hard cap
	}}
	_, _, err := ensureHealthyProxy(context.Background(), current, gen, EnsureHealthyProxyOptions{Checker: chk})
	if err == nil {
		t.Fatal("expected fail-closed error when no healthy proxy exists")
	}
}
