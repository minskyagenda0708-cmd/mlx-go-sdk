package mlx

import (
	"context"
	"testing"
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
