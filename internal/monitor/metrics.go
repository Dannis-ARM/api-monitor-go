package monitor

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// APIStatusGauge uses a Gauge metric, 1 for success, 0 for failure.
	APIStatusGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "api_availability_status",
			Help: "API availability status (1 for up, 0 for down)",
		},
		[]string{"api_name", "env"}, // Labels to distinguish different APIs and environments
	)

	// APILatencyGauge uses a Gauge to record API response time.
	APILatencyGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "api_response_seconds",
			Help: "API response time in seconds",
		},
		[]string{"api_name", "env"}, // Labels to distinguish different APIs and environments
	)

	// CertificateTTLGauge records remaining time (in seconds) until TLS certificate expiry
	CertificateTTLGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "api_certificate_ttl_seconds",
			Help: "Remaining time in seconds until the API TLS certificate expires",
		},
		[]string{"api_name", "env"},
	)
)

// RegisterMetrics registers Prometheus metrics.
func RegisterMetrics() {
	prometheus.MustRegister(APIStatusGauge)
	prometheus.MustRegister(APILatencyGauge)
	prometheus.MustRegister(CertificateTTLGauge)
}
