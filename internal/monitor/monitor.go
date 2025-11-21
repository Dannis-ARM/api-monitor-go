package monitor

import (
	"context"
	"log" // 仅保留 log 用于 http.ListenAndServe 的 Fatal
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var httGetProber = NewHTTPGetProbe()

// probeAPI 执行单个 API 的探测。
func probeAPI(executor ProbeExecutor, api API, apiTimeout time.Duration, currentEnv string) {
	FmtLog(LogLevelInfo, "Probing %s at %s...", api.Name, api.URL)
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
	defer cancel()

	resp, err := executor.Execute(ctx, api.URL)
	latency := time.Since(start).Seconds()

	if err != nil {
		FmtLog(LogLevelError, "  -> FAILED, error: %v", err)
		APIStatusGauge.With(
			prometheus.Labels{"api_name": api.Name, "env": currentEnv, "region": api.Region}).
			Set(0)
		APILatencyGauge.With(
			prometheus.Labels{"api_name": api.Name, "env": currentEnv, "region": api.Region}).
			Set(apiTimeout.Seconds())
		return
	}
	defer resp.Body.Close()

	FmtLog(LogLevelInfo, "  -> SUCCESS (TLS connected), response time: %.2fs", latency)
	APIStatusGauge.With(
		prometheus.Labels{"api_name": api.Name, "env": currentEnv, "region": api.Region}).
		Set(1)
	APILatencyGauge.With(
		prometheus.Labels{"api_name": api.Name, "env": currentEnv, "region": api.Region}).
		Set(latency)
}

// probeSingleAPI 是一个辅助函数，用于在一个 goroutine 中探测单个 API。
func probeSingleAPI(api API, apiTimeout time.Duration, currentEnv string, wg *sync.WaitGroup) {
	defer wg.Done()
	// 默认使用 HTTPGetProbe。如果需要其他类型，需要在此处根据逻辑创建对应的 ProbeExecutor 实例。
	probeAPI(httGetProber, api, apiTimeout, currentEnv)
}

// StartMonitoring 启动 API 监控服务。
func StartMonitoring(
	apis []API,
	apiTimeout,
	apiProbeInterval time.Duration,
	currentEnv string,
	metricsPort string) {
	// 注册 Prometheus 指标
	RegisterMetrics()

	// 启动 goroutine 定期探测 API
	go func() {
		for {
			var wg sync.WaitGroup
			for _, api := range apis {
				wg.Add(1)
				go probeSingleAPI(api, apiTimeout, currentEnv, &wg)
			}
			wg.Wait() // 等待所有探测完成

			FmtLog(LogLevelInfo, "Waiting for %v before the next probe...", apiProbeInterval)
			time.Sleep(apiProbeInterval)
		}
	}()

	// 启动 HTTP 服务器暴露指标
	http.Handle("/metrics", promhttp.Handler())
	FmtLog(LogLevelInfo, "Prometheus metrics server started on http://localhost%s", metricsPort)
	log.Fatal(http.ListenAndServe(metricsPort, nil))
}
