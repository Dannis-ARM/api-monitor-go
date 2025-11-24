package monitor

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath" // Import filepath for path manipulation

	"github.com/goccy/go-yaml"
)

// API defines the structure of an API to be monitored.
type API struct {
	Name        string `yaml:"name,omitempty"`        // Used for Prometheus labels; auto-generated or derived from Description if not provided.
	URL         string `yaml:"url"`                   // Path part of the API's URL (without protocol and host).
	Protocol    string `yaml:"protocol,omitempty"`    // API's protocol (http/https), defaults to https.
	Description string `yaml:"description,omitempty"` // Corresponds to description in sample-config.yaml.
	Region      string `yaml:"region,omitempty"`      // Optional region label, empty if not specified in YAML.
}

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
	MonitorAPIs   []API         `yaml:"monitor_apis"`
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

	// If Name is empty, try to derive it from Description or generate a default name.
	for i := range config.MonitorAPIs {
		if config.MonitorAPIs[i].Name == "" {
			if config.MonitorAPIs[i].Description != "" {
				config.MonitorAPIs[i].Name = config.MonitorAPIs[i].Description
			} else {
				config.MonitorAPIs[i].Name = fmt.Sprintf("api_yaml_%d", i+1)
			}
		}
		// If Region is empty, keep it empty; it can be populated later by defaultRegion from command line arguments.
		if config.MonitorAPIs[i].Region == "" {
			config.MonitorAPIs[i].Region = ""
		}
	}

	// Process API URLs, constructing the full URL based on the protocol field.
	var validAPIs []API
	for _, api := range config.MonitorAPIs {
		protocol := api.Protocol
		if protocol == "" {
			protocol = "https" // Default protocol is https.
		}

		fullURL := fmt.Sprintf("%s://%s", protocol, api.URL)
		parsedURL, err := url.Parse(fullURL)
		if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
			FmtLog(LogLevelError, "Invalid API URL format for API '%s': %s. Skipping this API.", api.Name, fullURL)
			continue // Skip invalid URLs.
		}
		api.URL = fullURL // Update to the full URL.
		validAPIs = append(validAPIs, api)
	}
	config.MonitorAPIs = validAPIs // Update the API list in the configuration.

	return &config, nil
}
