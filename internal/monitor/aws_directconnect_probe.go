package monitor

import (
	"context"
	"fmt"
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
	name       string
	statistic  string // Using string label: "Average", "Sum", etc.
}

// Execute implements ProbeExecutor interface
func (p *DirectConnectProbe) Execute(ctx context.Context) (ProbeResult, error) {
	startTime := time.Now()
	apiName := "direct_connect_" + p.connectionID

	// Set collection success to 0 by default
	DirectConnectCollectSuccessGauge.WithLabelValues(p.connectionID, p.currentEnv).Set(0)

	// Get CloudWatch metrics - use GetMetricData to batch fetch ALL metrics in ONE API call
	// This avoids hitting CloudWatch rate limits (400 TPS/account)
	metrics := []dxMetric{
		{"ConnectionBpsIngress", "Average"},
		{"ConnectionBpsEgress", "Average"},
		{"ConnectionPpsIngress", "Average"},
		{"ConnectionPpsEgress", "Average"},
		{"ConnectionPacketLossCountIngress", "Sum"},
		{"ConnectionPacketLossCountEgress", "Sum"},
		{"ConnectionErrorCountIngress", "Sum"},
		{"ConnectionErrorCountEgress", "Sum"},
		{"ConnectionCRCErrorCount", "Sum"},
	}

	endTime := time.Now()
	startTimeCW := endTime.Add(-time.Duration(p.lookbackMinutes) * time.Minute)
	period := int32(300) // 5 minutes

	// Fetch all 9 metrics in a SINGLE API call - TPS-friendly!
	metricValues, err := p.getMetricsBatch(ctx, metrics, startTimeCW, endTime, period)

	// Check for API level error
	if err != nil {
		FmtLog(LogLevelError, "Failed to batch fetch metrics for %s: %v", p.connectionID, err)
		APIStatusGauge.WithLabelValues(apiName, p.currentEnv).Set(0)
		APILatencyGauge.WithLabelValues(apiName, p.currentEnv).Set(time.Since(startTime).Seconds())
		return NewProbeResult(apiName, 0, time.Since(startTime).Seconds(), 0, err), err
	}

	// BPS metrics - critical for success
	if _, ok := metricValues["ConnectionBpsIngress"]; !ok {
		err := fmt.Errorf("ConnectionBpsIngress metric not found")
		FmtLog(LogLevelError, "Failed to get inbound BPS metric for %s: %v", p.connectionID, err)
		APIStatusGauge.WithLabelValues(apiName, p.currentEnv).Set(0)
		APILatencyGauge.WithLabelValues(apiName, p.currentEnv).Set(time.Since(startTime).Seconds())
		return NewProbeResult(apiName, 0, time.Since(startTime).Seconds(), 0, err), err
	}
	if _, ok := metricValues["ConnectionBpsEgress"]; !ok {
		err := fmt.Errorf("ConnectionBpsEgress metric not found")
		FmtLog(LogLevelError, "Failed to get outbound BPS metric for %s: %v", p.connectionID, err)
		APIStatusGauge.WithLabelValues(apiName, p.currentEnv).Set(0)
		APILatencyGauge.WithLabelValues(apiName, p.currentEnv).Set(time.Since(startTime).Seconds())
		return NewProbeResult(apiName, 0, time.Since(startTime).Seconds(), 0, err), err
	}

	// Set all gauges
	bpsIn := metricValues["ConnectionBpsIngress"]
	bpsOut := metricValues["ConnectionBpsEgress"]
	ppsIn := metricValues["ConnectionPpsIngress"]
	ppsOut := metricValues["ConnectionPpsEgress"]
	packetLossIn := metricValues["ConnectionPacketLossCountIngress"]
	packetLossOut := metricValues["ConnectionPacketLossCountEgress"]
	errorCountIn := metricValues["ConnectionErrorCountIngress"]
	errorCountOut := metricValues["ConnectionErrorCountEgress"]
	crcErrorCount := metricValues["ConnectionCRCErrorCount"]

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
	APIStatusGauge.WithLabelValues(apiName, p.currentEnv).Set(1)
	APILatencyGauge.WithLabelValues(apiName, p.currentEnv).Set(time.Since(startTime).Seconds())

	FmtLog(LogLevelInfo, "Direct Connect %s: In=%.2f bps, Out=%.2f bps, PPSIn=%.2f, PPSOut=%.2f, PacketLossIn=%.0f, PacketLossOut=%.0f, ErrorIn=%.0f, ErrorOut=%.0f, CRC=%.0f",
		p.connectionID, bpsIn, bpsOut, ppsIn, ppsOut, packetLossIn, packetLossOut, errorCountIn, errorCountOut, crcErrorCount)

	latency := time.Since(startTime).Seconds()
	return NewProbeResult(apiName, 1, latency, 0, nil), nil
}

// getMetricsBatch fetches ALL CloudWatch metrics in a SINGLE GetMetricData API call
// This is TPS-friendly: 1 API call instead of N separate GetMetricStatistics calls
func (p *DirectConnectProbe) getMetricsBatch(ctx context.Context, metrics []dxMetric, startTime, endTime time.Time, period int32) (map[string]float64, error) {
	// Build metric data queries - one for each metric
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
		StartTime:         aws.Time(startTime),
		EndTime:           aws.Time(endTime),
		ScanBy:            types.ScanByTimestampDescending,
	}

	resp, err := p.cwClient.GetMetricData(ctx, input)
	if err != nil {
		return nil, err
	}

	// Extract latest value for each metric
	results := make(map[string]float64)
	for _, result := range resp.MetricDataResults {
		// Find the original metric name - since we truncated Id, match by position
		// Get the latest value (first one since we ScanByTimestampDescending)
		if len(result.Values) > 0 {
			// Find metric name by matching position
			for i, q := range queries {
				if *q.Id == *result.Id {
					results[metrics[i].name] = result.Values[0]
					break
				}
			}
		}
	}

	return results, nil
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

// StartDirectConnectMonitoring creates Direct Connect probes and starts periodic monitoring in a dedicated goroutine
// This is the only public API needed - it fully encapsulates both probe creation and execution
func StartDirectConnectMonitoring(awsConfig AWSConfig, apiTimeout, probeInterval time.Duration, currentEnv string) {
	probes := createDxProbes(awsConfig, currentEnv)

	go func() {
		for {
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