package monitor

import (
	"time"
)

// ProbeResult contains the results of an API probe.
// All fields in ProbeResult are intended to be immutable after creation.
type ProbeResult struct {
	APIName    string
	Status     int     // 0 for FAILED, 1 for SUCCESS
	Latency    float64 // Latency in seconds
	StatusCode int     // HTTP status code
	Error      error   // Error information if the probe failed
	Timestamp  time.Time
}

// NewProbeResult creates and returns a new ProbeResult instance.
// The fields (APIName, Status, Latency, StatusCode, Error, Timestamp) are set
// during creation and should be considered immutable.
func NewProbeResult(apiName string, status int, latency float64, statusCode int, err error) ProbeResult {
	return ProbeResult{
		APIName:    apiName,
		Status:     status,
		Latency:    latency,
		StatusCode: statusCode,
		Error:      err,
		Timestamp:  time.Now(),
	}
}
