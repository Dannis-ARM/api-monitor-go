package monitor

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// 定义 Prometheus 指标
	// apiStatusGauge 使用 Gauge 指标，1 表示成功，0 表示失败
	APIStatusGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "api_availability_status",
			Help: "API availability status (1 for up, 0 for down)",
		},
		[]string{"api_name", "env", "region"}, // 使用标签区分不同的 API、环境和区域
	)

	// apiLatencyGauge 使用 Gauge 记录 API 响应时间
	APILatencyGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "api_response_seconds",
			Help: "API response time in seconds",
		},
		[]string{"api_name", "env", "region"}, // 使用标签区分不同的 API、环境和区域
	)
)

// RegisterMetrics 注册 Prometheus 指标
func RegisterMetrics() {
	prometheus.MustRegister(APIStatusGauge)
	prometheus.MustRegister(APILatencyGauge)
}
