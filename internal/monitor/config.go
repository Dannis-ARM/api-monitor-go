package monitor

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath" // Import filepath for path manipulation

	"github.com/goccy/go-yaml"
)

// API 定义了要监控的 API 的结构
type API struct {
	Name        string `yaml:"name,omitempty"`        // 用于 Prometheus 标签，如果未提供将自动生成或从 Description 派生
	URL         string `yaml:"url"`                   // API 的 URL 的路径部分（不包含协议和主机）
	Protocol    string `yaml:"protocol,omitempty"`    // API 的协议 (http/https), 默认为 https
	Description string `yaml:"description,omitempty"` // 对应 sample-config.yaml 中的 description
	Region      string `yaml:"region,omitempty"`      // 可选的区域标签，如果 YAML 未指定则为空
}

// MonitorConfig 定义了监控服务的通用配置
type MonitorConfig struct {
	APITimeout       string `yaml:"api_timeout"`
	APIProbeInterval string `yaml:"api_probe_interval"`
	CurrentEnv       string `yaml:"current_env"`
	DefaultRegion    string `yaml:"default_region"`
	MetricsPort      string `yaml:"metrics_port"`
}

// YAMLConfig 定义了 YAML 配置文件的结构
type YAMLConfig struct {
	MonitorConfig MonitorConfig `yaml:"monitor_config"`
	MonitorAPIs   []API         `yaml:"monitor_apis"`
}

// LoadYAMLConfig 从 YAML 文件加载 API 配置。
func LoadYAMLConfig(filePath string) (*YAMLConfig, error) {
	// 解析文件路径：如果是相对路径，则在当前工作目录中查找
	if !filepath.IsAbs(filePath) { // 使用 filepath.IsAbs 检查绝对路径
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current working directory: %w", err)
		}
		filePath = filepath.Join(cwd, filePath) // 使用 filepath.Join 安全地连接路径
	}

	// 确保文件路径有 .yaml 后缀
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

	// 如果 Name 为空，尝试从 Description 派生或生成默认名称
	for i := range config.MonitorAPIs {
		if config.MonitorAPIs[i].Name == "" {
			if config.MonitorAPIs[i].Description != "" {
				config.MonitorAPIs[i].Name = config.MonitorAPIs[i].Description
			} else {
				config.MonitorAPIs[i].Name = fmt.Sprintf("api_yaml_%d", i+1)
			}
		}
		// 如果 Region 为空，保持为空，后续可由命令行参数中的 defaultRegion 填充
		if config.MonitorAPIs[i].Region == "" {
			config.MonitorAPIs[i].Region = ""
		}
	}

	// 处理 API URL，根据 protocol 字段构建完整的 URL
	var validAPIs []API
	for _, api := range config.MonitorAPIs {
		protocol := api.Protocol
		if protocol == "" {
			protocol = "https" // 默认协议为 https
		}

		fullURL := fmt.Sprintf("%s://%s", protocol, api.URL)
		parsedURL, err := url.Parse(fullURL)
		if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
			FmtLog(LogLevelError, "Invalid API URL format for API '%s': %s. Skipping this API.", api.Name, fullURL)
			continue // 跳过无效的 URL
		}
		api.URL = fullURL // 更新为完整的 URL
		validAPIs = append(validAPIs, api)
	}
	config.MonitorAPIs = validAPIs // 更新配置中的 API 列表

	return &config, nil
}
