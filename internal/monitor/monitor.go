package monitor

import (
	"context"
	"log" // log is kept only for http.ListenAndServe's Fatal
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// probeAPI performs the probing for a single API.
func probeAPI(executor ProbeExecutor, apiTimeout time.Duration, currentEnv string) {
	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
	defer cancel()

	probeResult, err := executor.Execute(ctx) // URL is now hardcoded within the probe executor

	if err != nil {
		FmtLog(LogLevelError, "  -> FAILED, error: %v", err)
		APIStatusGauge.With(
			prometheus.Labels{"api_name": probeResult.APIName, "env": currentEnv}).
			Set(0)
		APILatencyGauge.With(
			prometheus.Labels{"api_name": probeResult.APIName, "env": currentEnv}).
			Set(apiTimeout.Seconds()) // Record timeout duration on failure
		return
	}

	FmtLog(LogLevelInfo, "  -> SUCCESS (TLS connected), response time: %.2fs, status code: %d", probeResult.Latency, probeResult.StatusCode)
	APIStatusGauge.With(
		prometheus.Labels{"api_name": probeResult.APIName, "env": currentEnv}).
		Set(1)
	APILatencyGauge.With(
		prometheus.Labels{"api_name": probeResult.APIName, "env": currentEnv}).
		Set(probeResult.Latency)
}

// probeSingleAPI is a helper function to probe a single API in a goroutine.
func probeSingleAPI(executor ProbeExecutor, apiTimeout time.Duration, currentEnv string, wg *sync.WaitGroup) {
	defer wg.Done()
	probeAPI(executor, apiTimeout, currentEnv)
}

// StartMonitoring starts the API monitoring service.
func StartMonitoring(
	apiTimeout,
	apiProbeInterval time.Duration,
	currentEnv string,
	metricsPort string) {
	// Register Prometheus metrics
	RegisterMetrics()

	// Hardcode API probes
	probes := []ProbeExecutor{
		NewHTTPSProbe("https://www.google.com", "google"),
		NewHTTPSProbe("https://www.lala.com", "lala"),
		NewHTTPProbe("http://180.101.51.73", "baiduraw"),
	}

	// Start a goroutine to periodically probe APIs
	go func() {
		for {
			var wg sync.WaitGroup
			for _, probe := range probes {
				wg.Add(1)
				go probeSingleAPI(probe, apiTimeout, currentEnv, &wg)
			}
			wg.Wait() // Wait for all probes to complete

			FmtLog(LogLevelInfo, "Waiting for %v before the next probe...", apiProbeInterval)
			time.Sleep(apiProbeInterval)
		}
	}()

	// Start an HTTP server to expose metrics
	http.Handle("/metrics", promhttp.Handler())
	FmtLog(LogLevelInfo, "Prometheus metrics server started on http://localhost%s", metricsPort)
	log.Fatal(http.ListenAndServe(metricsPort, nil))
}
