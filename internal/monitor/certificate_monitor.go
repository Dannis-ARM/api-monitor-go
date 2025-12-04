package monitor

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

// GetCertificateExpiry connects to the given target (URL, host or host:port)
// and returns the certificate NotAfter time (expiry time).
// Examples of target: "https://example.com", "example.com", "example.com:443".
func GetCertificateExpiry(target string, timeout time.Duration) (time.Time, error) {
    host, port, serverName, err := normalizeHostPort(target)
    if err != nil {
        FmtLog(LogLevelError, "GetCertificateExpiry: normalizeHostPort failed: %v", err)
        return time.Time{}, err
    }

    address := net.JoinHostPort(host, port)

    dialer := &net.Dialer{Timeout: timeout}
    conn, err := tls.DialWithDialer(dialer, "tcp", address, &tls.Config{
        InsecureSkipVerify: true,
        ServerName:         serverName,
    })
    if err != nil {
        FmtLog(LogLevelError, "TLS dial failed for %s: %v", address, err)
        return time.Time{}, fmt.Errorf("failed to connect: %w", err)
    }
    defer conn.Close()

    state := conn.ConnectionState()
    if len(state.PeerCertificates) == 0 {
        return time.Time{}, fmt.Errorf("no peer certificates found for %s", address)
    }

    cert := state.PeerCertificates[0]
    return cert.NotAfter, nil
}

// GetCertificateTTL returns the remaining duration until the certificate expires.
// A negative duration means the certificate is already expired.
func GetCertificateTTL(target string, timeout time.Duration) (time.Duration, error) {
    expiry, err := GetCertificateExpiry(target, timeout)
    if err != nil {
        return 0, err
    }
    return time.Until(expiry), nil
}

// normalizeHostPort parses target and returns host, port, and serverName for TLS SNI.
func normalizeHostPort(target string) (host string, port string, serverName string, err error) {
    // default TLS port
    port = "443"

    // Try parse as URL first
    if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
        u, perr := url.Parse(target)
        if perr != nil {
            return "", "", "", perr
        }
        host = u.Hostname()
        if p := u.Port(); p != "" {
            port = p
        } else if u.Scheme == "http" {
            port = "80"
        }
        serverName = host
        return
    }

    // If target contains colon, assume host:port
    if strings.Contains(target, ":") {
        h, p, perr := net.SplitHostPort(target)
        if perr == nil {
            host = h
            port = p
            serverName = host
            return
        }
        // If SplitHostPort failed, fallthrough and try URL parse
    }

    // Otherwise assume it's a bare hostname
    host = target
    serverName = host
    return
}
