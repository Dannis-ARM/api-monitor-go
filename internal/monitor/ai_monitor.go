package monitor

import (
	"context"
	"crypto/tls"
	"net/http"
	"sync"
	"time"
)

// StatusCheckProbe is an implementation of ProbeExecutor that checks HTTP status codes.
// 2xx/3xx -> success (status=1), 4xx/5xx -> failure (status=0)
type StatusCheckProbe struct {
	Client *http.Client
	URL    string
	Name   string
}

// NewStatusCheckProbe creates and returns a new StatusCheckProbe instance with TLS verification enabled.
func NewStatusCheckProbe(url, name string) *StatusCheckProbe {
	return &StatusCheckProbe{
		Client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: false}, // Verify TLS certificates
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // Don't follow redirects, record the last response
			},
		},
		URL:  url,
		Name: name,
	}
}

// Execute implements the ProbeExecutor interface, performing an HTTP GET request
// and checking the status code: 2xx/3xx -> success, 4xx/5xx -> failure.
func (p *StatusCheckProbe) Execute(ctx context.Context) (ProbeResult, error) {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, "GET", p.URL, nil)
	if err != nil {
		return NewProbeResult(p.Name, 0, 0, 0, err), err
	}

	resp, err := p.Client.Do(req)
	latency := time.Since(start).Seconds()
	if err != nil {
		return NewProbeResult(p.Name, 0, latency, 0, err), err
	}
	defer resp.Body.Close()

	// Determine success based on status code: 2xx/3xx -> 1, 4xx/5xx -> 0
	var status int
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		status = 1
	} else {
		status = 0
	}

	// Only return error for network/connection issues, not for 4xx/5xx status codes
	return NewProbeResult(p.Name, status, latency, resp.StatusCode, nil), nil
}

// AIHealthCheckProbe is an implementation of ProbeExecutor specifically for the AI health check API.
type AIHealthCheckProbe struct {
	StatusCheckProbe // Embed StatusCheckProbe
}

// NewAIHealthCheckProbe creates and returns a new AIHealthCheckProbe instance.
func NewAIHealthCheckProbe() *AIHealthCheckProbe {
	// Default URL and Name for AI health check probe
	aiURL := "https://test-ai.com"
	aiName := "AIHealthCheck"

	return &AIHealthCheckProbe{
		StatusCheckProbe: *NewStatusCheckProbe(aiURL, aiName),
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
