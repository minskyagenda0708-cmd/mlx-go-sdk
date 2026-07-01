package mlx

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
