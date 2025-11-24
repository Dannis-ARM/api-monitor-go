package monitor

import (
	"time"
)

// ProbeResult contains the results of an API probe.
type ProbeResult struct {
	APIName    string
	Region     string // optional
	Env        string
	Status     int     // 0 for FAILED, 1 for SUCCESS
	Latency    float64 // Latency in seconds
	StatusCode int     // HTTP status code
	Error      error   // Error information if the probe failed
	Timestamp  time.Time
}

// ProbeResultOption is a function type for configuring ProbeResult.
type ProbeResultOption func(*ProbeResult)

// WithRegion is an optional parameter to set the region for ProbeResult.
func WithRegion(region string) ProbeResultOption {
	return func(pr *ProbeResult) {
		pr.Region = region
	}
}

// NewProbeResult creates and returns a new ProbeResult instance with functional options.
func NewProbeResult(apiName, env string, status int, latency float64, statusCode int, err error, opts ...ProbeResultOption) ProbeResult {
	result := ProbeResult{
		APIName:    apiName,
		Env:        env,
		Status:     status,
		Latency:    latency,
		StatusCode: statusCode,
		Error:      err,
		Timestamp:  time.Now(),
	}

	for _, opt := range opts {
		opt(&result)
	}

	return result
}
