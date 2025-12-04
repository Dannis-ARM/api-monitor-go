package monitor

import (
	"net"
	"testing"
	"time"
)

func TestGetCertificateTTL_Integration(t *testing.T) {
    // Try to resolve a well-known host first; if DNS/network not available, skip the test.
    if _, err := net.LookupHost("www.baidu.com"); err != nil {
        t.Skipf("network unavailable or DNS lookup failed: %v", err)
    }

    timeout := 5 * time.Second
    ttl, err := GetCertificateTTL("https://www.baidu.com", timeout)
    if err != nil {
        t.Skipf("skipping integration test because TLS dial failed: %v", err)
    }
    if ttl <= 0 {
        t.Fatalf("expected positive TTL for www.google.com, got %v", ttl)
    }
}
