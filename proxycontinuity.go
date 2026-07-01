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

	country := ""
	city := ""
	region := ""
	if current != nil {
		country = strings.TrimSpace(current.Country)
		city = strings.TrimSpace(current.City)
		region = strings.TrimSpace(current.Region)
	}

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

	// 4) Select under two-tier latency rule.
	idx := selectBestProxy(cands, opts.ThresholdMs, opts.HardCapMs)
	if idx == -1 {
		return nil, false, fmt.Errorf("no healthy proxy found within %dms hard cap for country %q", opts.HardCapMs, country)
	}
	chosen := cands[idx].proxy
	changed := chosen != current
	return chosen, changed, nil
}

// EnsureHealthyProfileProxyOptions configures the service-level continuity check.
type EnsureHealthyProfileProxyOptions struct {
	EnsureHealthyProxyOptions
	Region       string
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
