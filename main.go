package main

import (
	"flag"
	"os"
	"time"

	"api-monitor/internal/cronutils"
	"api-monitor/internal/monitor" // Import our internal package
)

func main() {
	var envOverride string
	var configPath string

	// Command-line argument parsing
	flag.StringVar(&envOverride, "env", "", "Override the current environment (e.g., 'dev', 'prod')")
	flag.StringVar(&configPath, "config", "configs/application.yaml", "Path to the YAML configuration file")
	flag.Parse()

	if envOverride != "" {
		monitor.FmtLog(monitor.LogLevelInfo, "Environment override received from command line: %s", envOverride)
	}

	// Load configuration from the specified path, defaulting to "configs/application.yaml" if not provided via --config.
	cfg, err := monitor.LoadYAMLConfig(configPath)
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

	metricsPort := cfg.MonitorConfig.MetricsPort

	monitor.FmtLog(monitor.LogLevelInfo, "Monitoring environment: %s", currentEnv)
	monitor.FmtLog(monitor.LogLevelInfo, "APIs are hardcoded in monitor.StartMonitoring")

	// Initialize and start the cron job
	cronutils.InitCronJob()

	// Start the monitoring service
	monitor.StartMonitoring(apiTimeout, apiProbeInterval, currentEnv, metricsPort)
}
