package main

import (
	"flag"
	"os"
	"time"

	"api-monitor/internal/monitor" // Import our internal package
)

func main() {
	var configFilePath string
	var envOverride string

	// Command-line argument parsing
	flag.StringVar(&configFilePath, "config", "", "Path to the YAML configuration file for APIs")
	flag.StringVar(&envOverride, "env", "", "Override the current environment (e.g., 'dev', 'prod')")
	flag.Parse()

	monitor.FmtLog(monitor.LogLevelInfo, "Config file path received: %s", configFilePath)
	if envOverride != "" {
		monitor.FmtLog(monitor.LogLevelInfo, "Environment override received from command line: %s", envOverride)
	}

	if configFilePath == "" {
		monitor.FmtLog(monitor.LogLevelError, "No configuration file path provided. Use -config flag.")
		flag.Usage()
		os.Exit(1)
	}

	// Load configuration
	cfg, err := monitor.LoadYAMLConfig(configFilePath)
	if err != nil {
		monitor.FmtLog(monitor.LogLevelError, "Error loading configuration: %v", err)
		os.Exit(1)
	}

	// Parse durations from configuration
	apiTimeout, err := time.ParseDuration(cfg.MonitorConfig.APITimeout)
	if err != nil {
		monitor.FmtLog(monitor.LogLevelError, "Error parsing api_timeout: %v", err)
		os.Exit(1)
	}
	apiProbeInterval, err := time.ParseDuration(cfg.MonitorConfig.APIProbeInterval)
	if err != nil {
		monitor.FmtLog(monitor.LogLevelError, "Error parsing api_probe_interval: %v", err)
		os.Exit(1)
	}

	currentEnv := cfg.MonitorConfig.CurrentEnv
	// If an environment is provided via command line, it overrides the one in the config file
	if envOverride != "" {
		currentEnv = envOverride
	}

	defaultRegion := cfg.MonitorConfig.DefaultRegion
	metricsPort := cfg.MonitorConfig.MetricsPort

	// Apply default region to APIs that do not specify one
	for i := range cfg.MonitorAPIs {
		if cfg.MonitorAPIs[i].Region == "" {
			cfg.MonitorAPIs[i].Region = defaultRegion
		}
	}

	monitor.FmtLog(monitor.LogLevelInfo, "Monitoring environment: %s", currentEnv)
	monitor.FmtLog(monitor.LogLevelInfo, "Loaded %d APIs from %s", len(cfg.MonitorAPIs), configFilePath)

	// Start the monitoring service
	monitor.StartMonitoring(cfg.MonitorAPIs, apiTimeout, apiProbeInterval, currentEnv, metricsPort)
}
