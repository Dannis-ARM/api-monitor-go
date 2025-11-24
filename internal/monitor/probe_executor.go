package monitor

import (
	"context"
	"crypto/tls" // Added for InvalidURLProbe
	"net/http"
	"time"
)

// ProbeExecutor defines the interface for executing API probes.
type ProbeExecutor interface {
	Execute(ctx context.Context) (ProbeResult, error)
}

// HTTPGetProbe is an implementation of ProbeExecutor for executing HTTP GET requests.
type HTTPGetProbe struct {
	Client *http.Client
	URL    string
	Name   string
}

// NewHTTPGetProbe creates and returns a new HTTPGetProbe instance.
func NewHTTPGetProbe(url, name string) *HTTPGetProbe {
	return &HTTPGetProbe{
		Client: &http.Client{
			Transport: &http.Transport{}, // No TLS config needed for HTTP
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		URL:    url,
		Name:   name,
	}
}

// Execute implements the ProbeExecutor interface, performing an HTTP GET request.
func (p *HTTPGetProbe) Execute(ctx context.Context) (ProbeResult, error) {
	start := time.Now() // Start latency measurement here
	req, err := http.NewRequestWithContext(ctx, "GET", p.URL, nil)
	if err != nil {
		return NewProbeResult(p.Name, 0, 0, 0, err), err
	}

	resp, err := p.Client.Do(req)
	latency := time.Since(start).Seconds() // Calculate latency here
	if err != nil {
		return NewProbeResult(p.Name, 0, latency, 0, err), err
	}
	defer resp.Body.Close()

	return NewProbeResult(p.Name, 1, latency, resp.StatusCode, nil), nil
}

// HTTPSProbe is an implementation of ProbeExecutor for executing HTTPS GET requests.
type HTTPSProbe struct {
	Client *http.Client
	URL    string
	Name   string
}

// NewHTTPSProbe creates and returns a new HTTPSProbe instance.
func NewHTTPSProbe(url, name string) *HTTPSProbe {
	return &HTTPSProbe{
		Client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		URL:    url,
		Name:   name,
	}
}

// Execute implements the ProbeExecutor interface, performing an HTTPS GET request.
func (p *HTTPSProbe) Execute(ctx context.Context) (ProbeResult, error) {
	start := time.Now() // Start latency measurement here
	req, err := http.NewRequestWithContext(ctx, "GET", p.URL, nil)
	if err != nil {
		return NewProbeResult(p.Name, 0, 0, 0, err), err
	}

	resp, err := p.Client.Do(req)
	latency := time.Since(start).Seconds() // Calculate latency here
	if err != nil {
		return NewProbeResult(p.Name, 0, latency, 0, err), err
	}
	defer resp.Body.Close()

	return NewProbeResult(p.Name, 1, latency, resp.StatusCode, nil), nil
}

// HTTPProbe is an implementation of ProbeExecutor for executing HTTP GET requests.
type HTTPProbe struct {
	Client *http.Client
	URL    string
	Name   string
}

// NewHTTPProbe creates and returns a new HTTPProbe instance.
func NewHTTPProbe(url, name string) *HTTPProbe {
	return &HTTPProbe{
		Client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		URL:    url,
		Name:   name,
	}
}

// Execute implements the ProbeExecutor interface, performing an HTTP GET request.
func (p *HTTPProbe) Execute(ctx context.Context) (ProbeResult, error) {
	start := time.Now() // Start latency measurement here
	req, err := http.NewRequestWithContext(ctx, "GET", p.URL, nil)
	if err != nil {
		return NewProbeResult(p.Name, 0, 0, 0, err), err
	}

	resp, err := p.Client.Do(req)
	latency := time.Since(start).Seconds() // Calculate latency here
	if err != nil {
		return NewProbeResult(p.Name, 0, latency, 0, err), err
	}
	defer resp.Body.Close()

	return NewProbeResult(p.Name, 1, latency, resp.StatusCode, nil), nil
}
