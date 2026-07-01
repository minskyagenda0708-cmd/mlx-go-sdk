package mlx

import "context"

// ProxyCheckResult is the outcome of a single proxy health check.
type ProxyCheckResult struct {
	Alive     bool   // true if at least one target responded within the timeout
	LatencyMs int    // best (minimum) time-to-first-byte across targets, in ms
	Target    string // the target URL that produced the best measurement
	Err       error  // last error encountered, if Alive is false
}

// ProxyChecker measures whether a proxy is alive and how fast it is.
//
// The default implementation is HTTPProxyChecker. The interface exists so a
// JA3/TLS-impersonating checker can be substituted later without changing callers
// such as EnsureHealthyProxy.
type ProxyChecker interface {
	Check(ctx context.Context, p *Proxy) ProxyCheckResult
}
