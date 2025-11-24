package monitor

import (
	"context"
	"crypto/tls"
	"net/http"
	"strings"
)

// ProbeExecutor defines the interface for executing API probes.
type ProbeExecutor interface {
	Execute(ctx context.Context, url string) (ProbeResult, error)
}

// HTTPGetProbe is an implementation of ProbeExecutor for executing HTTP GET requests.
type HTTPGetProbe struct {
	Client *http.Client
}

// NewHTTPGetProbe creates and returns a new HTTPGetProbe instance.
func NewHTTPGetProbe() *HTTPGetProbe {
	return &HTTPGetProbe{
		Client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// Execute implements the ProbeExecutor interface, performing an HTTP GET request.
func (p *HTTPGetProbe) Execute(ctx context.Context, url string) (ProbeResult, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return NewProbeResult("", "", 0, 0, 0, err, WithRegion("")), err
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return NewProbeResult("", "", 0, 0, 0, err, WithRegion("")), err
	}
	defer resp.Body.Close()

	return NewProbeResult("", "", 1, 0, resp.StatusCode, nil, WithRegion("")), nil
}

// HTTPPostProbe is an implementation of ProbeExecutor for executing HTTP POST requests,
// supporting custom Headers and Body.
type HTTPPostProbe struct {
	Client  *http.Client
	Headers map[string]string
	Body    string // For simplicity, body is a string. In real applications, it might be an io.Reader or []byte.
}

// NewHTTPPostProbe creates and returns a new HTTPPostProbe instance.
func NewHTTPPostProbe(headers map[string]string, body string) *HTTPPostProbe {
	return &HTTPPostProbe{
		Client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		Headers: headers,
		Body:    body,
	}
}

// Execute implements the ProbeExecutor interface, performing an HTTP POST request.
func (p *HTTPPostProbe) Execute(ctx context.Context, url string) (ProbeResult, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(p.Body))
	if err != nil {
		return NewProbeResult("", "", 0, 0, 0, err, WithRegion("")), err
	}

	for key, value := range p.Headers {
		req.Header.Set(key, value)
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return NewProbeResult("", "", 0, 0, 0, err, WithRegion("")), err
	}
	defer resp.Body.Close()

	return NewProbeResult("", "", 1, 0, resp.StatusCode, nil, WithRegion("")), nil
}
