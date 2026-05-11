package monitor

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

// DirectConnectProbe implements ProbeExecutor for AWS Direct Connect monitoring
type DirectConnectProbe struct {
	awsConfig       AWSConfig
	currentEnv      string
	cwClient        *cloudwatch.Client
	connectionID    string
	lookbackMinutes int
}

// NewDirectConnectProbe creates a new DirectConnectProbe instance
func NewDirectConnectProbe(awsConfig AWSConfig, currentEnv string, connectionID string) (*DirectConnectProbe, error) {
	ctx := context.Background()

	// Load AWS configuration
	var cfg aws.Config
	var err error

	if awsConfig.AccessKey != "" && awsConfig.SecretKey != "" {
		// Use explicit credentials if provided
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(awsConfig.Region),
			config.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
				return aws.Credentials{
					AccessKeyID:     awsConfig.AccessKey,
					SecretAccessKey: awsConfig.SecretKey,
				}, nil
			})),
		)
	} else {
		// Use default credential chain
		cfg, err = config.LoadDefaultConfig(ctx, config.WithRegion(awsConfig.Region))
	}

	if err != nil {
		return nil, err
	}

	// Set default lookback time to 10 minutes if not configured
	lookbackMinutes := awsConfig.DirectConnect.MetricsLookbackMinutes
	if lookbackMinutes <= 0 {
		lookbackMinutes = 10
	}

	return &DirectConnectProbe{
		awsConfig:       awsConfig,
		currentEnv:      currentEnv,
		cwClient:        cloudwatch.NewFromConfig(cfg),
		connectionID:    connectionID,
		lookbackMinutes: lookbackMinutes,
	}, nil
}

// dxMetric represents a CloudWatch metric to fetch with its label for gauges
type dxMetric struct {
	name      string
	statistic string // Using string label: "Average", "Sum", etc.
	val       *float64
}

// Execute implements ProbeExecutor interface
func (p *DirectConnectProbe) Execute(ctx context.Context) (ProbeResult, error) {
	startTime := time.Now()
	apiName := "direct_connect_" + p.connectionID

	// Set collection success to 0 by default
	DirectConnectCollectSuccessGauge.WithLabelValues(p.connectionID, p.currentEnv).Set(0)

	// Initialize metric values with default error value
	var (
		bpsIn, bpsOut                   = metricErrorValue, metricErrorValue
		ppsIn, ppsOut                   = metricErrorValue, metricErrorValue
		packetLossIn, packetLossOut     = metricErrorValue, metricErrorValue
		errorCountIn, errorCountOut     = metricErrorValue, metricErrorValue
		crcErrorCount                   = metricErrorValue
	)

	// Get CloudWatch metrics - use GetMetricData to batch fetch ALL metrics in ONE API call
	// This avoids hitting CloudWatch rate limits (400 TPS/account)
	metrics := []dxMetric{
		{"ConnectionBpsIngress", "Average", &bpsIn},
		{"ConnectionBpsEgress", "Average", &bpsOut},
		{"ConnectionPpsIngress", "Average", &ppsIn},
		{"ConnectionPpsEgress", "Average", &ppsOut},
		{"ConnectionPacketLossCountIngress", "Sum", &packetLossIn},
		{"ConnectionPacketLossCountEgress", "Sum", &packetLossOut},
		{"ConnectionErrorCountIngress", "Sum", &errorCountIn},
		{"ConnectionErrorCountEgress", "Sum", &errorCountOut},
		{"ConnectionCRCErrorCount", "Sum", &crcErrorCount},
	}

	endTime := time.Now()
	startTimeCW := endTime.Add(-time.Duration(p.lookbackMinutes) * time.Minute)
	period := int32(300) // 5 minutes

	// Build metric queries directly, fetch all metrics in one API call
	queries := make([]types.MetricDataQuery, 0, len(metrics))
	for _, m := range metrics {
		metricStat := &types.MetricStat{
			Metric: &types.Metric{
				Namespace:  aws.String("AWS/DX"),
				MetricName: aws.String(m.name),
				Dimensions: []types.Dimension{
					{
						Name:  aws.String("ConnectionId"),
						Value: aws.String(p.connectionID),
					},
				},
			},
			Period: aws.Int32(period),
			Stat:   aws.String(m.statistic),
		}
		queries = append(queries, types.MetricDataQuery{
			Id:         aws.String(strings.ToLower(m.name)[:min(15, len(m.name))]),
			MetricStat: metricStat,
			ReturnData: aws.Bool(true),
		})
	}

	input := &cloudwatch.GetMetricDataInput{
		MetricDataQueries: queries,
		StartTime:         aws.Time(startTimeCW),
		EndTime:           aws.Time(endTime),
		ScanBy:            types.ScanByTimestampDescending,
	}

	resp, err := p.cwClient.GetMetricData(ctx, input)
	if err != nil {
		FmtLog(LogLevelError, "Failed to batch fetch metrics for %s: %v", p.connectionID, err)
		DXAPIStatusGauge.WithLabelValues(apiName, p.currentEnv).Set(0)
		DXAPILatencyGauge.WithLabelValues(apiName, p.currentEnv).Set(time.Since(startTime).Seconds())
		DirectConnectCollectSuccessGauge.WithLabelValues(p.connectionID, p.currentEnv).Set(0)
		return NewProbeResult(apiName, 0, time.Since(startTime).Seconds(), 0, err), err
	}

	// Process results and assign values directly, no intermediate map needed
	for _, result := range resp.MetricDataResults {
		if len(result.Values) > 0 {
			for i, q := range queries {
				if *q.Id == *result.Id {
					*metrics[i].val = result.Values[0]
					break
				}
			}
		}
	}

	// Check all metrics for missing values
	for _, m := range metrics {
		if *m.val == metricErrorValue {
			FmtLog(LogLevelWarn, "%s metric not found for %s", m.name, p.connectionID)
		}
	}

	// Connection_id labeled metrics
	DirectConnectBPSInGauge.WithLabelValues(p.connectionID, p.currentEnv).Set(bpsIn)
	DirectConnectBPSOutGauge.WithLabelValues(p.connectionID, p.currentEnv).Set(bpsOut)
	DirectConnectPPSInGauge.WithLabelValues(p.connectionID, p.currentEnv).Set(ppsIn)
	DirectConnectPPSOutGauge.WithLabelValues(p.connectionID, p.currentEnv).Set(ppsOut)
	DirectConnectPacketLossInGauge.WithLabelValues(p.connectionID, p.currentEnv).Set(packetLossIn)
	DirectConnectPacketLossOutGauge.WithLabelValues(p.connectionID, p.currentEnv).Set(packetLossOut)
	DirectConnectErrorCountInGauge.WithLabelValues(p.connectionID, p.currentEnv).Set(errorCountIn)
	DirectConnectErrorCountOutGauge.WithLabelValues(p.connectionID, p.currentEnv).Set(errorCountOut)
	DirectConnectCRCErrorCountGauge.WithLabelValues(p.connectionID, p.currentEnv).Set(crcErrorCount)

	// Api_name labeled metrics
	DirectConnectAPIBPSInGauge.WithLabelValues(apiName, p.currentEnv).Set(bpsIn)
	DirectConnectAPIBPSOutGauge.WithLabelValues(apiName, p.currentEnv).Set(bpsOut)
	DirectConnectAPIPPSInGauge.WithLabelValues(apiName, p.currentEnv).Set(ppsIn)
	DirectConnectAPIPPSOutGauge.WithLabelValues(apiName, p.currentEnv).Set(ppsOut)

	// Mark collection as successful
	DirectConnectCollectSuccessGauge.WithLabelValues(p.connectionID, p.currentEnv).Set(1)

	// Expose API-level generic metrics (consistent with probeAPI pattern)
	DXAPIStatusGauge.WithLabelValues(apiName, p.currentEnv).Set(1)
	DXAPILatencyGauge.WithLabelValues(apiName, p.currentEnv).Set(time.Since(startTime).Seconds())

	FmtLog(LogLevelInfo, "Direct Connect %s: In=%.2f bps, Out=%.2f bps, PPSIn=%.2f, PPSOut=%.2f, PacketLossIn=%.0f, PacketLossOut=%.0f, ErrorIn=%.0f, ErrorOut=%.0f, CRC=%.0f",
		p.connectionID, bpsIn, bpsOut, ppsIn, ppsOut, packetLossIn, packetLossOut, errorCountIn, errorCountOut, crcErrorCount)

	latency := time.Since(startTime).Seconds()
	return NewProbeResult(apiName, 1, latency, 0, nil), nil
}


func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// createDxProbes creates AWS Direct Connect probes based on configuration
func createDxProbes(awsConfig AWSConfig, currentEnv string) []ProbeExecutor {
	var dxProbes []ProbeExecutor

	if awsConfig.Region != "" && len(awsConfig.DirectConnect.ConnectionIDs) > 0 {
		FmtLog(LogLevelInfo, "Adding %d AWS Direct Connect probes for region %s",
			len(awsConfig.DirectConnect.ConnectionIDs), awsConfig.Region)

		for _, connID := range awsConfig.DirectConnect.ConnectionIDs {
			dxProbe, err := NewDirectConnectProbe(awsConfig, currentEnv, connID)
			if err != nil {
				FmtLog(LogLevelError, "Failed to create Direct Connect probe for %s: %v", connID, err)
				continue
			}
			dxProbes = append(dxProbes, dxProbe)
		}
	}

	return dxProbes
}

const probeRefreshInterval = 30 * time.Minute // Refresh probes every 30 minutes to renew credentials
const metricErrorValue = -1.0                 // Default value for failed metric fetch, used to mark anomalies

// StartDirectConnectMonitoring creates Direct Connect probes and starts periodic monitoring in a dedicated goroutine
// This is the only public API needed - it fully encapsulates both probe creation and execution
func StartDirectConnectMonitoring(awsConfig AWSConfig, apiTimeout, probeInterval time.Duration, currentEnv string) {
	probes := createDxProbes(awsConfig, currentEnv)
	lastRefreshTime := time.Now()

	go func() {
		for {
			// Recreate probes every 30 minutes to refresh AWS credentials
			if time.Since(lastRefreshTime) >= probeRefreshInterval {
				FmtLog(LogLevelInfo, "Refreshing Direct Connect probes to renew AWS credentials...")
				newProbes := createDxProbes(awsConfig, currentEnv)
				if len(newProbes) > 0 {
					probes = newProbes
					lastRefreshTime = time.Now()
					FmtLog(LogLevelInfo, "Successfully refreshed %d Direct Connect probes", len(probes))
				} else {
					FmtLog(LogLevelError, "Failed to refresh Direct Connect probes, continuing with existing probes")
				}
			}

			var wg sync.WaitGroup
			executeDxProbes(probes, apiTimeout, currentEnv, &wg)
			wg.Wait()

			FmtLog(LogLevelInfo, "Direct Connect probes completed, waiting for %v before next run...", probeInterval)
			time.Sleep(probeInterval)
		}
	}()
}

// executeDxProbes executes all Direct Connect probes in separate goroutines and exposes metrics via WithLabelValues
func executeDxProbes(probes []ProbeExecutor, apiTimeout time.Duration, currentEnv string, wg *sync.WaitGroup) {
	for _, probe := range probes {
		wg.Add(1)
		go func(p ProbeExecutor) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
			defer cancel()

			result, err := p.Execute(ctx)

			// All DX-specific metrics (BPS, PPS, PacketLoss, ErrorCount, CRC, APIStatus, APILatency)
			// are already exposed directly to Prometheus within the Execute() method.
			// No additional gauge setting is needed here.
			if dxProbe, ok := p.(*DirectConnectProbe); ok {
				connectionID := dxProbe.connectionID
				if err != nil {
					FmtLog(LogLevelError, "Direct Connect probe %s failed: %v (latency=%.3fs)", connectionID, err, result.Latency)
				} else {
					FmtLog(LogLevelInfo, "Direct Connect probe %s completed successfully, latency=%.3fs",
						connectionID, result.Latency)
				}
			}
		}(probe)
	}
}
