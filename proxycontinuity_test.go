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

// recordingGen records the country passed to each generation round so tests
// can assert the geo constraint actually propagates into replacement lookup.
type recordingGen struct {
	gotCountry     []string
	cityProxies    []*Proxy
	countryProxies []*Proxy
}

func (g *recordingGen) generate(_ context.Context, country, _, city string, _ int) ([]*Proxy, error) {
	g.gotCountry = append(g.gotCountry, country)
	if city != "" {
		return g.cityProxies, nil
	}
	return g.countryProxies, nil
}

// TestEnsureHealthyProxyDerivesGeoFromUsername covers the root cause of the
// live geo-mismatch: managed MLX proxies encode geography only inside the
// username (e.g. "-country-be-sid-..."), leaving the structural Country/City
// fields empty. Continuity must still preserve country by deriving it from the
// username when the structural field is blank, otherwise replacement rounds
// query with an empty country and can return an out-of-country exit node.
func TestEnsureHealthyProxyDerivesGeoFromUsername(t *testing.T) {
	current := &Proxy{
		Host:     "cur",
		Username: "2235470499_bc98e4f8_multilogin_com-country-be-sid-abc-filter-medium",
		// Country/City intentionally empty: mirrors live GetMeta payload.
	}
	gen := &recordingGen{
		countryProxies: []*Proxy{{Host: "n1", Username: "u-country-be-sid-z"}},
	}
	chk := mapChecker{byHost: map[string]ProxyCheckResult{
		"cur": {Alive: false},
		"n1":  {Alive: true, LatencyMs: 1000},
	}}
	got, changed, err := ensureHealthyProxy(context.Background(), current, gen, EnsureHealthyProxyOptions{Checker: chk})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed || got.Host != "n1" {
		t.Fatalf("expected replacement n1, got host=%s changed=%v", got.Host, changed)
	}
	sawBE := false
	for _, c := range gen.gotCountry {
		if c == "be" {
			sawBE = true
		}
	}
	if !sawBE {
		t.Fatalf("expected generation to request country \"be\" derived from username, got rounds=%v", gen.gotCountry)
	}
}

// TestEnsureHealthyProxyRejectsGeoMismatchedReplacement is the defense-in-depth
// half: even if a misbehaving gate returns an out-of-country node despite a
// correct request, continuity must NOT hand it to a profile whose fingerprint
// expects a different country. It must fail closed instead.
func TestEnsureHealthyProxyRejectsGeoMismatchedReplacement(t *testing.T) {
	current := &Proxy{Host: "cur", Username: "u-country-be-sid-a"}
	gen := &fakeGen{
		// Gate returns a DE node even though BE was requested.
		countryProxies: []*Proxy{{Host: "de1", Username: "u-country-de-sid-b"}},
	}
	chk := mapChecker{byHost: map[string]ProxyCheckResult{
		"cur": {Alive: false},
		"de1": {Alive: true, LatencyMs: 900},
	}}
	_, _, err := ensureHealthyProxy(context.Background(), current, gen, EnsureHealthyProxyOptions{Checker: chk})
	if err == nil {
		t.Fatal("expected fail-closed error when replacement country does not match required country")
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
