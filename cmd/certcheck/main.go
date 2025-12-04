package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"api-monitor/internal/monitor"
)

func main() {
    timeout := flag.Duration("timeout", 5*time.Second, "Dial timeout for TLS connections")
    flag.Parse()

    targets := flag.Args()
    if len(targets) == 0 {
        fmt.Fprintln(os.Stderr, "Usage: certcheck [--timeout duration] host1 [host2 ...]")
        os.Exit(2)
    }

    for _, t := range targets {
        ttl, err := monitor.GetCertificateTTL(t, *timeout)
        if err != nil {
            fmt.Printf("%s -> ERROR: %v\n", t, err)
            continue
        }
        expiry := time.Now().Add(ttl)
        fmt.Printf("%s -> expires in %v (at %s)\n", t, ttl, expiry.Format(time.RFC3339))
    }
}
