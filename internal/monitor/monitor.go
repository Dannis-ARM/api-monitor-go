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

var httGetProber = NewHTTPGetProbe()

// probeAPI performs the probing for a single API.
func probeAPI(executor ProbeExecutor, api API, apiTimeout time.Duration, currentEnv string) {
	FmtLog(LogLevelInfo, "Probing %s at %s...", api.Name, api.URL)
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
	defer cancel()

	probeResult, err := executor.Execute(ctx, api.URL)
	probeResult.APIName = api.Name
	probeResult.Region = api.Region
	probeResult.Env = currentEnv
	probeResult.Latency = time.Since(start).Seconds() // Record actual latency

	if err != nil {
		FmtLog(LogLevelError, "  -> FAILED, error: %v", err)
		APIStatusGauge.With(
			prometheus.Labels{"api_name": probeResult.APIName, "env": probeResult.Env, "region": probeResult.Region}).
			Set(0)
		APILatencyGauge.With(
			prometheus.Labels{"api_name": probeResult.APIName, "env": probeResult.Env, "region": probeResult.Region}).
			Set(apiTimeout.Seconds()) // Record timeout duration on failure
		return
	}

	FmtLog(LogLevelInfo, "  -> SUCCESS (TLS connected), response time: %.2fs, status code: %d", probeResult.Latency, probeResult.StatusCode)
	APIStatusGauge.With(
		prometheus.Labels{"api_name": probeResult.APIName, "env": probeResult.Env, "region": probeResult.Region}).
		Set(1)
	APILatencyGauge.With(
		prometheus.Labels{"api_name": probeResult.APIName, "env": probeResult.Env, "region": probeResult.Region}).
		Set(probeResult.Latency)
}

// probeSingleAPI is a helper function to probe a single API in a goroutine.
func probeSingleAPI(api API, apiTimeout time.Duration, currentEnv string, wg *sync.WaitGroup) {
	defer wg.Done()
	// Default to using HTTPGetProbe. If other types are needed,
	// corresponding ProbeExecutor instances should be created here based on logic.
	probeAPI(httGetProber, api, apiTimeout, currentEnv)
}

// StartMonitoring starts the API monitoring service.
func StartMonitoring(
	apis []API,
	apiTimeout,
	apiProbeInterval time.Duration,
	currentEnv string,
	metricsPort string) {
	// Register Prometheus metrics
	RegisterMetrics()

	// Start a goroutine to periodically probe APIs
	go func() {
		for {
			var wg sync.WaitGroup
			for _, api := range apis {
				wg.Add(1)
				go probeSingleAPI(api, apiTimeout, currentEnv, &wg)
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
