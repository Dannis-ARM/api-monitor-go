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

	// DirectConnectBPSInGauge records AWS Direct Connect inbound traffic in bits per second
	DirectConnectBPSInGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aws_direct_connect_bps_in",
			Help: "AWS Direct Connect inbound traffic in bits per second",
		},
		[]string{"connection_id", "env"},
	)

	// DirectConnectBPSOutGauge records AWS Direct Connect outbound traffic in bits per second
	DirectConnectBPSOutGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aws_direct_connect_bps_out",
			Help: "AWS Direct Connect outbound traffic in bits per second",
		},
		[]string{"connection_id", "env"},
	)

	// DirectConnectPPSInGauge records AWS Direct Connect inbound packets per second
	DirectConnectPPSInGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aws_direct_connect_pps_in",
			Help: "AWS Direct Connect inbound packets per second",
		},
		[]string{"connection_id", "env"},
	)

	// DirectConnectPPSOutGauge records AWS Direct Connect outbound packets per second
	DirectConnectPPSOutGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aws_direct_connect_pps_out",
			Help: "AWS Direct Connect outbound packets per second",
		},
		[]string{"connection_id", "env"},
	)

	// DirectConnectPacketLossInGauge records AWS Direct Connect inbound packet loss count
	DirectConnectPacketLossInGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aws_direct_connect_packet_loss_in",
			Help: "AWS Direct Connect inbound packet loss count",
		},
		[]string{"connection_id", "env"},
	)

	// DirectConnectPacketLossOutGauge records AWS Direct Connect outbound packet loss count
	DirectConnectPacketLossOutGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aws_direct_connect_packet_loss_out",
			Help: "AWS Direct Connect outbound packet loss count",
		},
		[]string{"connection_id", "env"},
	)

	// DirectConnectErrorCountInGauge records AWS Direct Connect inbound error count
	DirectConnectErrorCountInGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aws_direct_connect_error_count_in",
			Help: "AWS Direct Connect inbound error count",
		},
		[]string{"connection_id", "env"},
	)

	// DirectConnectErrorCountOutGauge records AWS Direct Connect outbound error count
	DirectConnectErrorCountOutGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aws_direct_connect_error_count_out",
			Help: "AWS Direct Connect outbound error count",
		},
		[]string{"connection_id", "env"},
	)

	// DirectConnectCRCErrorCountGauge records AWS Direct Connect CRC error count
	DirectConnectCRCErrorCountGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aws_direct_connect_crc_error_count",
			Help: "AWS Direct Connect CRC error count",
		},
		[]string{"connection_id", "env"},
	)

	// DirectConnectConnectionStateGauge records AWS Direct Connect connection state (1=available, 0=unavailable)
	DirectConnectConnectionStateGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aws_direct_connect_connection_state",
			Help: "AWS Direct Connect connection state (1=available, 0=unavailable)",
		},
		[]string{"connection_id", "env"},
	)

	// DirectConnectCollectSuccessGauge records AWS Direct Connect metrics collection success state (1=success, 0=failure)
	DirectConnectCollectSuccessGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aws_direct_connect_collect_success",
			Help: "AWS Direct Connect metrics collection success state (1=success, 0=failure)",
		},
		[]string{"connection_id", "env"},
	)

	// DirectConnectAPIBPSInGauge records AWS Direct Connect inbound traffic with api_name label
	DirectConnectAPIBPSInGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "api_direct_connect_bps_in",
			Help: "AWS Direct Connect inbound traffic in bits per second (api_name label)",
		},
		[]string{"api_name", "env"},
	)

	// DirectConnectAPIBPSOutGauge records AWS Direct Connect outbound traffic with api_name label
	DirectConnectAPIBPSOutGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "api_direct_connect_bps_out",
			Help: "AWS Direct Connect outbound traffic in bits per second (api_name label)",
		},
		[]string{"api_name", "env"},
	)

	// DirectConnectAPIPPSInGauge records AWS Direct Connect inbound packets per second with api_name label
	DirectConnectAPIPPSInGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "api_direct_connect_pps_in",
			Help: "AWS Direct Connect inbound packets per second (api_name label)",
		},
		[]string{"api_name", "env"},
	)

	// DirectConnectAPIPPSOutGauge records AWS Direct Connect outbound packets per second with api_name label
	DirectConnectAPIPPSOutGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "api_direct_connect_pps_out",
			Help: "AWS Direct Connect outbound packets per second (api_name label)",
		},
		[]string{"api_name", "env"},
	)

	// DXAPIStatusGauge records DX API availability status (1=up, 0=down)
	DXAPIStatusGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "api_direct_connect_status",
			Help: "DX API availability status (1 for up, 0 for down)",
		},
		[]string{"api_name", "env"},
	)

	// DXAPILatencyGauge records DX API response time in seconds
	DXAPILatencyGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "api_direct_connect_response_seconds",
			Help: "DX API response time in seconds",
		},
		[]string{"api_name", "env"},
	)
)

// RegisterMetrics registers Prometheus metrics.
func RegisterMetrics() {
	prometheus.MustRegister(APIStatusGauge)
	prometheus.MustRegister(APILatencyGauge)
	prometheus.MustRegister(CertificateTTLGauge)
	prometheus.MustRegister(DirectConnectBPSInGauge)
	prometheus.MustRegister(DirectConnectBPSOutGauge)
	prometheus.MustRegister(DirectConnectPPSInGauge)
	prometheus.MustRegister(DirectConnectPPSOutGauge)
	prometheus.MustRegister(DirectConnectPacketLossInGauge)
	prometheus.MustRegister(DirectConnectPacketLossOutGauge)
	prometheus.MustRegister(DirectConnectErrorCountInGauge)
	prometheus.MustRegister(DirectConnectErrorCountOutGauge)
	prometheus.MustRegister(DirectConnectCRCErrorCountGauge)
	prometheus.MustRegister(DirectConnectConnectionStateGauge)
	prometheus.MustRegister(DirectConnectCollectSuccessGauge)
	prometheus.MustRegister(DirectConnectAPIBPSInGauge)
	prometheus.MustRegister(DirectConnectAPIBPSOutGauge)
	prometheus.MustRegister(DirectConnectAPIPPSInGauge)
	prometheus.MustRegister(DirectConnectAPIPPSOutGauge)
	prometheus.MustRegister(DXAPIStatusGauge)
	prometheus.MustRegister(DXAPILatencyGauge)
}
