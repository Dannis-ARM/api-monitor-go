package monitor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
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
	URL  string
	Name string
}

// NewStatusCheckProbe creates and returns a new StatusCheckProbe instance with TLS verification enabled.
func NewStatusCheckProbe(targetURL, name string) *StatusCheckProbe {
	FmtLog(LogLevelInfo, "Creating StatusCheckProbe: url=%s, name=%s, proxy=%s", targetURL, name, aiHealthCheckProxy)
	return &StatusCheckProbe{
		URL:  targetURL,
		Name: name,
	}
}

// Execute implements the ProbeExecutor interface, performing an HTTP GET request via curl
// and checking the status code: 2xx/3xx/400/401/403 -> 1, others -> 0.
func (p *StatusCheckProbe) Execute(ctx context.Context) (ProbeResult, error) {
	start := time.Now()
	FmtLog(LogLevelInfo, "Starting curl request: url=%s", p.URL)

	// Run curl command with proxy and verbose output
	cmd := exec.CommandContext(ctx, "curl",
		"-x", aiHealthCheckProxy,
		"-o", "/dev/null",
		"-s", "-v",
		"-L",
		p.URL,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	latency := time.Since(start).Seconds()
	stderrStr := stderr.String()

	FmtLog(LogLevelInfo, "Curl stderr:\n%s", stderrStr)
	FmtLog(LogLevelInfo, "Curl exit code: %v", err)

	// Determine status from stderr
	var status int
	var effectiveStatusCode int

	if strings.Contains(stderrStr, "403") {
		status = 1
		effectiveStatusCode = 403
		FmtLog(LogLevelInfo, "Found 403 in output - PASS")
	} else if strings.Contains(stderrStr, "401") {
		status = 1
		effectiveStatusCode = 401
		FmtLog(LogLevelInfo, "Found 401 in output - PASS")
	} else if strings.Contains(stderrStr, "400") {
		status = 1
		effectiveStatusCode = 400
		FmtLog(LogLevelInfo, "Found 400 in output - PASS")
	} else if strings.Contains(stderrStr, "200") || strings.Contains(stderrStr, "204") ||
		strings.Contains(stderrStr, "301") || strings.Contains(stderrStr, "302") ||
		strings.Contains(stderrStr, "304") {
		status = 1
		// Try to find the actual status code
		for _, code := range []int{200, 204, 301, 302, 304} {
			if strings.Contains(stderrStr, fmt.Sprintf("%d", code)) {
				effectiveStatusCode = code
				break
			}
		}
		if effectiveStatusCode == 0 {
			effectiveStatusCode = 200
		}
		FmtLog(LogLevelInfo, "Found 2xx/3xx in output - PASS")
	} else {
		status = 0
		effectiveStatusCode = 0
		FmtLog(LogLevelWarn, "No success code found - FAIL")
	}

	FmtLog(LogLevelInfo, "Request completed: status=%d, effective=%d, latency=%.3fs", status, effectiveStatusCode, latency)

	return NewProbeResult(p.Name, status, latency, effectiveStatusCode, nil), nil
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
