package monitor

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Configuration constants
const (
	aiHealthCheckURL   = "https://test-ai.com"
	aiHealthCheckName  = "AIHealthCheck"
	aiHealthCheckProxy = "http://proxy.example.com:3128"
)

// StatusCheckProbe is an implementation of ProbeExecutor that checks HTTP status codes.
// 2xx/3xx -> success (status=1), 4xx/5xx -> failure (status=0)
type StatusCheckProbe struct {
	Client *http.Client
	URL    string
	Name   string
}

// NewStatusCheckProbe creates and returns a new StatusCheckProbe instance with TLS verification enabled.
func NewStatusCheckProbe(targetURL, name string) *StatusCheckProbe {
	// Configure proxy
	proxyURL, _ := url.Parse(aiHealthCheckProxy)
	FmtLog(LogLevelInfo, "Creating StatusCheckProbe: url=%s, name=%s, proxy=%s", targetURL, name, aiHealthCheckProxy)

	return &StatusCheckProbe{
		Client: &http.Client{
			Transport: &http.Transport{
				Proxy:           http.ProxyURL(proxyURL),
				TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		URL:  targetURL,
		Name: name,
	}
}

// Execute implements the ProbeExecutor interface, performing an HTTP GET request
// and checking the status code: 2xx/3xx/400/401/403 -> 1, others -> 0.
func (p *StatusCheckProbe) Execute(ctx context.Context) (ProbeResult, error) {
	start := time.Now()
	FmtLog(LogLevelInfo, "Starting request: url=%s, name=%s", p.URL, p.Name)

	req, err := http.NewRequestWithContext(ctx, "GET", p.URL, nil)
	if err != nil {
		latency := time.Since(start).Seconds()
		FmtLog(LogLevelError, "Failed to create request: %v", err)
		return NewProbeResult(p.Name, 0, latency, 0, err), err
	}

	req.Header.Set("User-Agent", "curl/8.0.0")
	FmtLog(LogLevelInfo, "Sending request: method=%s, url=%s", req.Method, req.URL)

	resp, err := p.Client.Do(req)
	latency := time.Since(start).Seconds()

	// Determine effective status code
	var effectiveStatusCode int

	if err != nil {
		FmtLog(LogLevelError, "Request failed: error=%v, error_type=%T", err, err)
		// Check if error is from proxy (401/403 etc.)
		if strings.Contains(err.Error(), "403") {
			effectiveStatusCode = 403
		} else if strings.Contains(err.Error(), "401") {
			effectiveStatusCode = 401
		} else if strings.Contains(err.Error(), "400") {
			effectiveStatusCode = 400
		} else {
			// Try to extract status code from error message more carefully
			effectiveStatusCode = extractStatusCodeFromError(err)
		}
		FmtLog(LogLevelInfo, "Request error: extracted status=%d", effectiveStatusCode)
	} else {
		defer resp.Body.Close()
		effectiveStatusCode = resp.StatusCode
		FmtLog(LogLevelInfo, "Got response: status=%d, proto=%s", resp.StatusCode, resp.Proto)
		FmtLog(LogLevelInfo, "Response headers:")
		for k, v := range resp.Header {
			FmtLog(LogLevelInfo, "  %s: %v", k, v)
		}
	}

	// Determine success based on status code: 2xx/3xx/400/401/403 -> 1, others -> 0
	var status int
	if (effectiveStatusCode >= 200 && effectiveStatusCode < 400) || effectiveStatusCode == 400 || effectiveStatusCode == 401 || effectiveStatusCode == 403 {
		status = 1
	} else {
		status = 0
	}

	FmtLog(LogLevelInfo, "Status determination: effective=%d, success=%d", effectiveStatusCode, status)

	// Only return error for non-403/401/400 cases
	if err != nil && status != 1 {
		FmtLog(LogLevelError, "Returning error (not success): %v", err)
		return NewProbeResult(p.Name, 0, latency, effectiveStatusCode, err), err
	}

	FmtLog(LogLevelInfo, "Request completed: status=%d, effective=%d, success=%d, latency=%.3fs",
		func() int {
			if resp != nil {
				return resp.StatusCode
			}
			return 0
		}(), effectiveStatusCode, status, latency)

	return NewProbeResult(p.Name, status, latency, effectiveStatusCode, nil), nil
}

// extractStatusCodeFromError tries to extract HTTP status code from error message
func extractStatusCodeFromError(err error) int {
	errStr := err.Error()
	// Common patterns: "403 Forbidden", "status code 403", "Received HTTP code 403"
	for _, code := range []int{403, 401, 400, 404, 500, 502, 503} {
		if strings.Contains(errStr, fmt.Sprintf("%d", code)) {
			return code
		}
	}
	return 0
}

// AIHealthCheckProbe is an implementation of ProbeExecutor specifically for the AI health check API.
type AIHealthCheckProbe struct {
	StatusCheckProbe // Embed StatusCheckProbe
}

// NewAIHealthCheckProbe creates and returns a new AIHealthCheckProbe instance.
func NewAIHealthCheckProbe() *AIHealthCheckProbe {
	return &AIHealthCheckProbe{
		StatusCheckProbe: *NewStatusCheckProbe(aiHealthCheckURL, aiHealthCheckName),
	}
}

// createAIProbes creates AI health check probes
func createAIProbes() []ProbeExecutor {
	return []ProbeExecutor{
		NewAIHealthCheckProbe(),
	}
}

// StartAIMonitoring creates AI health check probes and starts periodic monitoring in a dedicated goroutine
func StartAIMonitoring(apiTimeout, probeInterval time.Duration, currentEnv string) {
	probes := createAIProbes()

	go func() {
		for {
			var wg sync.WaitGroup
			executeAIProbes(probes, apiTimeout, currentEnv, &wg)
			wg.Wait()

			FmtLog(LogLevelInfo, "AI health check probes completed, waiting for %v before next run...", probeInterval)
			time.Sleep(probeInterval)
		}
	}()
}

// executeAIProbes executes all AI health check probes in separate goroutines
func executeAIProbes(probes []ProbeExecutor, apiTimeout time.Duration, currentEnv string, wg *sync.WaitGroup) {
	for _, probe := range probes {
		wg.Add(1)
		go func(p ProbeExecutor) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
			defer cancel()

			result, err := p.Execute(ctx)

			// Set Prometheus metrics based on probe result
			if err != nil {
				FmtLog(LogLevelError, "AI health check probe %s failed: %v (latency=%.3fs)", result.APIName, err, result.Latency)
				AIHealthStatusGauge.WithLabelValues(result.APIName, currentEnv).Set(0)
				AIHealthLatencyGauge.WithLabelValues(result.APIName, currentEnv).Set(result.Latency)
			} else {
				if result.Status == 1 {
					FmtLog(LogLevelInfo, "AI health check probe %s succeeded (status=%d), latency=%.3fs",
						result.APIName, result.StatusCode, result.Latency)
				} else {
					FmtLog(LogLevelWarn, "AI health check probe %s failed (status=%d), latency=%.3fs",
						result.APIName, result.StatusCode, result.Latency)
				}
				AIHealthStatusGauge.WithLabelValues(result.APIName, currentEnv).Set(float64(result.Status))
				AIHealthLatencyGauge.WithLabelValues(result.APIName, currentEnv).Set(result.Latency)
			}
		}(probe)
	}
}
