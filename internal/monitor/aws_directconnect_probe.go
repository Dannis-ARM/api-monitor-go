package monitor

import (
	"context"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/directconnect"
)

// DirectConnectProbe implements ProbeExecutor for AWS Direct Connect monitoring
type DirectConnectProbe struct {
	awsConfig       AWSConfig
	currentEnv      string
	cwClient        *cloudwatch.Client
	dxClient        *directconnect.Client
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
		dxClient:        directconnect.NewFromConfig(cfg),
		connectionID:    connectionID,
		lookbackMinutes: lookbackMinutes,
	}, nil
}

// Execute implements ProbeExecutor interface
func (p *DirectConnectProbe) Execute(ctx context.Context) (ProbeResult, error) {
	startTime := time.Now()

	// Set collection success to 0 by default
	DirectConnectCollectSuccessGauge.WithLabelValues(p.connectionID, p.currentEnv).Set(0)

	// 1. Get connection state
	connectionState, err := p.getConnectionState(ctx)
	if err != nil {
		FmtLog(LogLevelError, "Failed to describe Direct Connect connection %s: %v", p.connectionID, err)
		return NewProbeResult("direct_connect_"+p.connectionID, 0, time.Since(startTime).Seconds(), 0, err), err
	}
	DirectConnectConnectionStateGauge.WithLabelValues(p.connectionID, p.currentEnv).Set(float64(connectionState))

	// 2. Get CloudWatch metrics for the configured lookback period
	endTime := time.Now()
	startTimeCW := endTime.Add(-time.Duration(p.lookbackMinutes) * time.Minute)
	period := int32(300) // 5 minutes

	// Traffic metrics (use Average)
	bpsIn, err := p.getMetric(ctx, "ConnectionBpsIngress", startTimeCW, endTime, period, types.StatisticAverage)
	if err != nil {
		FmtLog(LogLevelError, "Failed to get inbound BPS metric for %s: %v", p.connectionID, err)
		return NewProbeResult("direct_connect_"+p.connectionID, 0, time.Since(startTime).Seconds(), connectionState, err), err
	}
	DirectConnectBPSInGauge.WithLabelValues(p.connectionID, p.currentEnv).Set(bpsIn)
	DirectConnectAPIBPSInGauge.WithLabelValues("direct_connect_"+p.connectionID, p.currentEnv).Set(bpsIn)

	bpsOut, err := p.getMetric(ctx, "ConnectionBpsEgress", startTimeCW, endTime, period, types.StatisticAverage)
	if err != nil {
		FmtLog(LogLevelError, "Failed to get outbound BPS metric for %s: %v", p.connectionID, err)
		return NewProbeResult("direct_connect_"+p.connectionID, 0, time.Since(startTime).Seconds(), connectionState, err), err
	}
	DirectConnectBPSOutGauge.WithLabelValues(p.connectionID, p.currentEnv).Set(bpsOut)
	DirectConnectAPIBPSOutGauge.WithLabelValues("direct_connect_"+p.connectionID, p.currentEnv).Set(bpsOut)

	// Packet rate metrics (use Average)
	ppsIn, err := p.getMetric(ctx, "ConnectionPpsIngress", startTimeCW, endTime, period, types.StatisticAverage)
	if err != nil {
		FmtLog(LogLevelWarn, "Failed to get inbound PPS metric for %s: %v", p.connectionID, err)
	}
	DirectConnectPPSInGauge.WithLabelValues(p.connectionID, p.currentEnv).Set(ppsIn)
	DirectConnectAPIPPSInGauge.WithLabelValues("direct_connect_"+p.connectionID, p.currentEnv).Set(ppsIn)

	ppsOut, err := p.getMetric(ctx, "ConnectionPpsEgress", startTimeCW, endTime, period, types.StatisticAverage)
	if err != nil {
		FmtLog(LogLevelWarn, "Failed to get outbound PPS metric for %s: %v", p.connectionID, err)
	}
	DirectConnectPPSOutGauge.WithLabelValues(p.connectionID, p.currentEnv).Set(ppsOut)
	DirectConnectAPIPPSOutGauge.WithLabelValues("direct_connect_"+p.connectionID, p.currentEnv).Set(ppsOut)

	// Packet loss metrics (use Sum for counter metrics)
	packetLossIn, err := p.getMetric(ctx, "ConnectionPacketLossCountIngress", startTimeCW, endTime, period, types.StatisticSum)
	if err != nil {
		FmtLog(LogLevelWarn, "Failed to get inbound packet loss metric for %s: %v", p.connectionID, err)
	}
	DirectConnectPacketLossInGauge.WithLabelValues(p.connectionID, p.currentEnv).Set(packetLossIn)

	packetLossOut, err := p.getMetric(ctx, "ConnectionPacketLossCountEgress", startTimeCW, endTime, period, types.StatisticSum)
	if err != nil {
		FmtLog(LogLevelWarn, "Failed to get outbound packet loss metric for %s: %v", p.connectionID, err)
	}
	DirectConnectPacketLossOutGauge.WithLabelValues(p.connectionID, p.currentEnv).Set(packetLossOut)

	// Error count metrics (use Sum for counter metrics)
	errorCountIn, err := p.getMetric(ctx, "ConnectionErrorCountIngress", startTimeCW, endTime, period, types.StatisticSum)
	if err != nil {
		FmtLog(LogLevelWarn, "Failed to get inbound error count metric for %s: %v", p.connectionID, err)
	}
	DirectConnectErrorCountInGauge.WithLabelValues(p.connectionID, p.currentEnv).Set(errorCountIn)

	errorCountOut, err := p.getMetric(ctx, "ConnectionErrorCountEgress", startTimeCW, endTime, period, types.StatisticSum)
	if err != nil {
		FmtLog(LogLevelWarn, "Failed to get outbound error count metric for %s: %v", p.connectionID, err)
	}
	DirectConnectErrorCountOutGauge.WithLabelValues(p.connectionID, p.currentEnv).Set(errorCountOut)

	// CRC error count (use Sum for counter metrics)
	crcErrorCount, err := p.getMetric(ctx, "ConnectionCRCErrorCount", startTimeCW, endTime, period, types.StatisticSum)
	if err != nil {
		FmtLog(LogLevelWarn, "Failed to get CRC error count metric for %s: %v", p.connectionID, err)
	}
	DirectConnectCRCErrorCountGauge.WithLabelValues(p.connectionID, p.currentEnv).Set(crcErrorCount)

	// Mark collection as successful
	DirectConnectCollectSuccessGauge.WithLabelValues(p.connectionID, p.currentEnv).Set(1)

	// Expose API-level generic metrics (consistent with probeAPI pattern)
	apiName := "direct_connect_" + p.connectionID
	APIStatusGauge.WithLabelValues(apiName, p.currentEnv).Set(1)
	APILatencyGauge.WithLabelValues(apiName, p.currentEnv).Set(time.Since(startTime).Seconds())

	FmtLog(LogLevelInfo, "Direct Connect %s: In=%.2f bps, Out=%.2f bps, PPSIn=%.2f, PPSOut=%.2f, PacketLossIn=%.0f, PacketLossOut=%.0f, ErrorIn=%.0f, ErrorOut=%.0f, CRC=%.0f, State=%d",
		p.connectionID, bpsIn, bpsOut, ppsIn, ppsOut, packetLossIn, packetLossOut, errorCountIn, errorCountOut, crcErrorCount, connectionState)

	latency := time.Since(startTime).Seconds()
	return NewProbeResult(apiName, 1, latency, connectionState, nil), nil
}

// getConnectionState retrieves the connection state from Direct Connect API
func (p *DirectConnectProbe) getConnectionState(ctx context.Context) (int, error) {
	dxInput := &directconnect.DescribeConnectionsInput{
		ConnectionId: aws.String(p.connectionID),
	}

	dxResp, err := p.dxClient.DescribeConnections(ctx, dxInput)
	if err != nil {
		return 0, err
	}

	if len(dxResp.Connections) > 0 && dxResp.Connections[0].ConnectionState == "available" {
		return 1, nil
	}
	return 0, nil
}

// getMetric retrieves CloudWatch metric data for a specific metric name
func (p *DirectConnectProbe) getMetric(ctx context.Context, metricName string, startTime, endTime time.Time, period int32, statistic types.Statistic) (float64, error) {
	input := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/DX"),
		MetricName: aws.String(metricName),
		Dimensions: []types.Dimension{
			{
				Name:  aws.String("ConnectionId"),
				Value: aws.String(p.connectionID),
			},
		},
		StartTime:  aws.Time(startTime),
		EndTime:    aws.Time(endTime),
		Period:     aws.Int32(period),
		Statistics: []types.Statistic{statistic},
	}

	resp, err := p.cwClient.GetMetricStatistics(ctx, input)
	if err != nil {
		return 0, err
	}

	return p.getLatestDatapointValue(resp.Datapoints, statistic), nil
}

// getLatestDatapointValue extracts the latest value from CloudWatch datapoints based on the statistic type
func (p *DirectConnectProbe) getLatestDatapointValue(datapoints []types.Datapoint, statistic types.Statistic) float64 {
	if len(datapoints) == 0 {
		return 0
	}

	// Get the latest datapoint
	latest := datapoints[0]
	for _, dp := range datapoints {
		if dp.Timestamp.After(*latest.Timestamp) {
			latest = dp
		}
	}

	// Extract value based on statistic type
	switch statistic {
	case types.StatisticAverage:
		if latest.Average != nil {
			return *latest.Average
		}
	case types.StatisticSum:
		if latest.Sum != nil {
			return *latest.Sum
		}
	case types.StatisticMaximum:
		if latest.Maximum != nil {
			return *latest.Maximum
		}
	case types.StatisticMinimum:
		if latest.Minimum != nil {
			return *latest.Minimum
		}
	}

	return 0
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
					FmtLog(LogLevelInfo, "Direct Connect probe %s completed successfully, State=%d, latency=%.3fs",
						connectionID, result.StatusCode, result.Latency)
				}
			}
		}(probe)
	}
}