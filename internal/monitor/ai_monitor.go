package monitor

import (
	"context"
	"os/exec"
	"strconv"
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
	return &StatusCheckProbe{
		URL:  targetURL,
		Name: name,
	}
}

// Execute implements the ProbeExecutor interface, performing an HTTP GET request via curl
// and checking the status code: 2xx/3xx -> success, 4xx/5xx -> failure.
func (p *StatusCheckProbe) Execute(ctx context.Context) (ProbeResult, error) {
	start := time.Now()

	// Run curl command with proxy
	cmd := exec.CommandContext(ctx, "curl",
		"-x", aiHealthCheckProxy, // Proxy
		"-o", "/dev/null", // Discard response body
		"-s", "-w", "%{http_code}", // Silent mode, output only http code
		"-L", // Follow redirects
		p.URL,
	)

	output, err := cmd.Output()
	latency := time.Since(start).Seconds()

	if err != nil {
		return NewProbeResult(p.Name, 0, latency, 0, err), err
	}

	// Parse status code from output
	statusCodeStr := strings.TrimSpace(string(output))
	statusCode, err := strconv.Atoi(statusCodeStr)
	if err != nil {
		return NewProbeResult(p.Name, 0, latency, 0, err), err
	}

	// Determine success based on status code: 2xx/3xx/400/401/403 -> 1, others -> 0
	var status int
	if (statusCode >= 200 && statusCode < 400) || statusCode == 400 || statusCode == 401 || statusCode == 403 {
		status = 1
	} else {
		status = 0
	}

	return NewProbeResult(p.Name, status, latency, statusCode, nil), nil
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
				FmtLog(LogLevelError, "  Error type: %T", err)
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
