package monitor

import (
	"context"
	"crypto/tls"
	"net/http"
	"strings" // 导入 strings 包
)

// ProbeExecutor 定义了执行 API 探测的接口。
type ProbeExecutor interface {
	Execute(ctx context.Context, url string) (*http.Response, error)
}

// HTTPGetProbe 是 ProbeExecutor 的一个实现，用于执行 HTTP GET 请求。
type HTTPGetProbe struct {
	Client *http.Client
}

// NewHTTPGetProbe 创建并返回一个新的 HTTPGetProbe 实例。
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

// Execute 实现 ProbeExecutor 接口，执行一个 HTTP GET 请求。
func (p *HTTPGetProbe) Execute(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	return p.Client.Do(req)
}

// HTTPPostProbe 是 ProbeExecutor 的一个实现，用于执行 HTTP POST 请求，支持自定义 Headers 和 Body。
type HTTPPostProbe struct {
	Client  *http.Client
	Headers map[string]string
	Body    string // For simplicity, body is a string. In real applications, it might be an io.Reader or []byte.
}

// NewHTTPPostProbe 创建并返回一个新的 HTTPPostProbe 实例。
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

// Execute 实现 ProbeExecutor 接口，执行一个 HTTP POST 请求。
func (p *HTTPPostProbe) Execute(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(p.Body))
	if err != nil {
		return nil, err
	}

	for key, value := range p.Headers {
		req.Header.Set(key, value)
	}

	return p.Client.Do(req)
}
