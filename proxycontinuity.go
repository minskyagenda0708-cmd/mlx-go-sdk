package mlx

import (
	"context"
	"fmt"
	"strings"
)

// EnsureHealthyProxyOptions tunes proxy-continuity behavior.
type EnsureHealthyProxyOptions struct {
	ThresholdMs        int          // preferred max latency; default 2000
	HardCapMs          int          // escalation cap; default 3000
	CandidatesPerRound int          // proxies generated per geo round; default 5
	Checker            ProxyChecker // default NewHTTPProxyChecker(HTTPProxyCheckerConfig{})
}

func (o *EnsureHealthyProxyOptions) defaults() {
	if o.ThresholdMs <= 0 {
		o.ThresholdMs = 2000
	}
	if o.HardCapMs <= 0 {
		o.HardCapMs = 3000
	}
	if o.CandidatesPerRound <= 0 {
		o.CandidatesPerRound = 5
	}
	if o.Checker == nil {
		o.Checker = NewHTTPProxyChecker(HTTPProxyCheckerConfig{})
	}
}

type proxyCandidate struct {
	proxy  *Proxy
	result ProxyCheckResult
}

// selectBestProxy returns the index of the best candidate under the two-tier
// latency rule: fastest alive <= thresholdMs; if none, fastest alive <= hardCapMs;
// if none, -1.
func selectBestProxy(cands []proxyCandidate, thresholdMs, hardCapMs int) int {
	best, bestLat := -1, 0
	// Tier 1: under threshold.
	for i, c := range cands {
		if c.result.Alive && c.result.LatencyMs <= thresholdMs {
			if best == -1 || c.result.LatencyMs < bestLat {
				best, bestLat = i, c.result.LatencyMs
			}
		}
	}
	if best != -1 {
		return best
	}
	// Tier 2: under hard cap.
	for i, c := range cands {
		if c.result.Alive && c.result.LatencyMs <= hardCapMs {
			if best == -1 || c.result.LatencyMs < bestLat {
				best, bestLat = i, c.result.LatencyMs
			}
		}
	}
	return best
}

type proxyGenerator interface {
	generate(ctx context.Context, country, region, city string, count int) ([]*Proxy, error)
}

// ensureHealthyProxy verifies the current proxy and, if needed, finds a
// geography-preserving replacement. It never returns a dead/too-slow proxy:
// if nothing healthy is found it returns an error (fail-closed).
func ensureHealthyProxy(ctx context.Context, current *Proxy, gen proxyGenerator, opts EnsureHealthyProxyOptions) (*Proxy, bool, error) {
	opts.defaults()

	var cands []proxyCandidate

	// 1) Current proxy first — keep it if healthy and under threshold.
	if current != nil && strings.TrimSpace(current.Host) != "" {
		res := opts.Checker.Check(ctx, current)
		if res.Alive && res.LatencyMs <= opts.ThresholdMs {
			return current, false, nil
		}
		cands = append(cands, proxyCandidate{proxy: current, result: res})
	}

	country, region, city := deriveProxyGeo(current)

	// 2) Round A — same city (only if we have a city).
	if city != "" {
		cityProxies, err := gen.generate(ctx, country, region, city, opts.CandidatesPerRound)
		if err != nil {
			return nil, false, err
		}
		for _, p := range cityProxies {
			cands = append(cands, proxyCandidate{proxy: p, result: opts.Checker.Check(ctx, p)})
		}
	}

	// 3) Round B — same country (if city round produced no live under-threshold pick).
	if selectBestProxy(cands, opts.ThresholdMs, opts.HardCapMs) == -1 {
		countryProxies, err := gen.generate(ctx, country, region, "", opts.CandidatesPerRound)
		if err != nil {
			return nil, false, err
		}
		if len(countryProxies) == 0 && len(cands) == 0 {
			return nil, false, fmt.Errorf("no proxies available for country %q", country)
		}
		for _, p := range countryProxies {
			cands = append(cands, proxyCandidate{proxy: p, result: opts.Checker.Check(ctx, p)})
		}
	}

	// 4) Select under two-tier latency rule, rejecting geo-mismatched nodes.
	// Defense-in-depth: even if the gate returns an out-of-country node despite
	// a correct request, never hand it to a profile whose fingerprint expects a
	// different country. The current proxy is exempt (it defines the baseline).
	if country != "" {
		filtered := cands[:0:0]
		for _, c := range cands {
			if c.proxy == current || proxyCountryMatches(c.proxy, country) {
				filtered = append(filtered, c)
			}
		}
		cands = filtered
	}
	idx := selectBestProxy(cands, opts.ThresholdMs, opts.HardCapMs)
	if idx == -1 {
		return nil, false, fmt.Errorf("no healthy proxy found within %dms hard cap for country %q", opts.HardCapMs, country)
	}
	chosen := cands[idx].proxy
	changed := chosen != current
	return chosen, changed, nil
}

// deriveProxyGeo returns the country/region/city for a proxy, preferring the
// structural fields but falling back to values encoded inside the MLX managed
// username (e.g. "-country-be-region-x-city-y-sid-...") when those fields are
// blank. Managed MLX proxies carry geography only in the username, so relying
// on the structural fields alone silently drops the country constraint.
func deriveProxyGeo(current *Proxy) (country, region, city string) {
	if current == nil {
		return "", "", ""
	}
	country = strings.TrimSpace(current.Country)
	region = strings.TrimSpace(current.Region)
	city = strings.TrimSpace(current.City)
	if country != "" || region != "" || city != "" {
		return country, region, city
	}
	uc, ur, ucity := parseProxyUsernameGeo(current.Username)
	if country == "" {
		country = uc
	}
	if region == "" {
		region = ur
	}
	if city == "" {
		city = ucity
	}
	return country, region, city
}

// parseProxyUsernameGeo extracts country/region/city tokens from an MLX managed
// proxy username of the form "...-country-<c>-region-<r>-city-<ci>-sid-...".
func parseProxyUsernameGeo(username string) (country, region, city string) {
	parts := strings.Split(username, "-")
	for i := 0; i < len(parts)-1; i++ {
		switch parts[i] {
		case "country":
			country = parts[i+1]
		case "region":
			region = parts[i+1]
		case "city":
			city = parts[i+1]
		}
	}
	return country, region, city
}

// proxyCountryMatches reports whether the proxy's country (structural or
// username-encoded) equals want, case-insensitively.
func proxyCountryMatches(p *Proxy, want string) bool {
	if p == nil {
		return false
	}
	c, _, _ := deriveProxyGeo(p)
	return strings.EqualFold(strings.TrimSpace(c), strings.TrimSpace(want))
}

// EnsureHealthyProfileProxyOptions configures the service-level continuity check.
type EnsureHealthyProfileProxyOptions struct {
	EnsureHealthyProxyOptions
	PreferSOCKS5 bool
	SaveTraffic  bool
}

// clientProxyGenerator generates candidates via the MLX proxy API.
type clientProxyGenerator struct {
	svc          *ProxyServiceOp
	preferSOCKS5 bool
	saveTraffic  bool
}

func (g clientProxyGenerator) generate(ctx context.Context, country, region, city string, count int) ([]*Proxy, error) {
	out := make([]*Proxy, 0, count)
	for i := 0; i < count; i++ {
		res, err := g.svc.GenerateProfileProxy(ctx, &GenerateProfileProxyRequest{
			GenerateProxyRequest: GenerateProxyRequest{
				Country: country,
				Region:  region,
				City:    city,
				Count:   1,
			},
			PreferSOCKS5: g.preferSOCKS5,
			SaveTraffic:  g.saveTraffic,
		})
		if err != nil {
			// Stop generating more; return what we have (may be empty).
			if len(out) == 0 {
				return nil, err
			}
			return out, nil
		}
		if res != nil && res.ProfileProxy != nil {
			out = append(out, res.ProfileProxy)
		}
	}
	return out, nil
}

// EnsureHealthyProxy verifies current and finds a geo-preserving replacement if needed.
func (s *ProxyServiceOp) EnsureHealthyProxy(ctx context.Context, current *Proxy, opts EnsureHealthyProfileProxyOptions) (*Proxy, bool, error) {
	gen := clientProxyGenerator{svc: s, preferSOCKS5: opts.PreferSOCKS5, saveTraffic: opts.SaveTraffic}
	return ensureHealthyProxy(ctx, current, gen, opts.EnsureHealthyProxyOptions)
}
