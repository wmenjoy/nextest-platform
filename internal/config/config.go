package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Config 服务配置
type Config struct {
	Server   ServerConfig   `toml:"server"`
	Database DatabaseConfig `toml:"database"`
	Test     TestConfig     `toml:"test"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Type string `toml:"type"` // sqlite, mysql, postgres
	DSN  string `toml:"dsn"`  // data source name
}

// TestConfig 测试配置
type TestConfig struct {
	TargetHost   string `toml:"target_host"`   // 被测试服务的地址
	RegistryPath string `toml:"registry_path"` // 测试用例注册路径（可选，用于导入）
}

// LoadConfig 加载配置文件
func LoadConfig(path string) (*Config, error) {
	var config Config

	// 读取配置文件
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 解析TOML
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// 设置默认值
	if config.Server.Host == "" {
		config.Server.Host = "0.0.0.0"
	}
	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}
	if config.Database.Type == "" {
		config.Database.Type = "sqlite"
	}
	if config.Database.DSN == "" {
		config.Database.DSN = "./data/test_management.db"
	}

	return &config, nil
}

// GetAddr 获取服务器监听地址
func (c *ServerConfig) GetAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
