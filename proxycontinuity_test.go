package mlx

import "testing"

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
