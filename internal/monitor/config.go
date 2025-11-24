package monitor

import (
	"fmt"
	"os"
	"path/filepath" // Import filepath for path manipulation

	"github.com/goccy/go-yaml"
)

// MonitorConfig defines the general configuration for the monitoring service.
type MonitorConfig struct {
	APITimeout       string `yaml:"api_timeout"`
	APIProbeInterval string `yaml:"api_probe_interval"`
	CurrentEnv       string `yaml:"current_env"`
	DefaultRegion    string `yaml:"default_region"`
	MetricsPort      string `yaml:"metrics_port"`
}

// YAMLConfig defines the structure of the YAML configuration file.
type YAMLConfig struct {
	MonitorConfig MonitorConfig `yaml:"monitor_config"`
}

// LoadYAMLConfig loads API configuration from a YAML file.
func LoadYAMLConfig(filePath string) (*YAMLConfig, error) {
	// Parse file path: if it's a relative path, look for it in the current working directory.
	if !filepath.IsAbs(filePath) { // Use filepath.IsAbs to check for absolute path.
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current working directory: %w", err)
		}
		filePath = filepath.Join(cwd, filePath) // Use filepath.Join to safely concatenate paths.
	}

	// Ensure the file path has a .yaml extension.
	if filepath.Ext(filePath) == "" {
		filePath += ".yaml"
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML file %s: %w", filePath, err)
	}

	var config YAMLConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML data from %s: %w", filePath, err)
	}

	return &config, nil
}
