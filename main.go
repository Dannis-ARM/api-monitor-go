package main

import (
	"flag"
	"os"
	"time"

	"api-monitor/internal/monitor" // 引入我们创建的内部包
)

func main() {
	var configFilePath string
	var envOverride string

	// 命令行参数解析
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

	// 加载配置
	cfg, err := monitor.LoadYAMLConfig(configFilePath)
	if err != nil {
		monitor.FmtLog(monitor.LogLevelError, "Error loading configuration: %v", err)
		os.Exit(1)
	}

	// 解析配置中的持续时间
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
	// 如果命令行提供了 env，则覆盖配置文件中的 currentEnv
	if envOverride != "" {
		currentEnv = envOverride
	}

	defaultRegion := cfg.MonitorConfig.DefaultRegion
	metricsPort := cfg.MonitorConfig.MetricsPort

	// 应用默认区域到没有指定区域的 API
	for i := range cfg.MonitorAPIs {
		if cfg.MonitorAPIs[i].Region == "" {
			cfg.MonitorAPIs[i].Region = defaultRegion
		}
	}

	monitor.FmtLog(monitor.LogLevelInfo, "Monitoring environment: %s", currentEnv)
	monitor.FmtLog(monitor.LogLevelInfo, "Loaded %d APIs from %s", len(cfg.MonitorAPIs), configFilePath)

	// 启动监控服务
	monitor.StartMonitoring(cfg.MonitorAPIs, apiTimeout, apiProbeInterval, currentEnv, metricsPort)
}
