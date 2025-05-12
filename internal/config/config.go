package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Log     LogConfig     `mapstructure:"log"`
	Collect CollectConfig `mapstructure:"collect"`
	API     APIConfig     `mapstructure:"api"`
	Query   QueryConfig   `mapstructure:"query"`
	Plugins PluginsConfig `mapstructure:"plugins"`
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// CollectConfig holds configuration for log collection
type CollectConfig struct {
	Sources     []string `mapstructure:"sources"`
	Workers     int      `mapstructure:"workers"`
	Storage     string   `mapstructure:"storage"`
	StoragePath string   `mapstructure:"storage-path"`
	BatchSize   int      `mapstructure:"batch-size"`
}

// APIConfig holds configuration for the API server
type APIConfig struct {
	Host        string        `mapstructure:"host"`
	Port        int           `mapstructure:"port"`
	Timeout     time.Duration `mapstructure:"timeout"`
	Storage     string        `mapstructure:"storage"`
	StoragePath string        `mapstructure:"storage-path"`
}

// QueryConfig holds configuration for log queries
type QueryConfig struct {
	Filter      string   `mapstructure:"filter"`
	TimeFrom    string   `mapstructure:"time-from"`
	TimeTo      string   `mapstructure:"time-to"`
	Sources     []string `mapstructure:"sources"`
	Limit       int      `mapstructure:"limit"`
	Storage     string   `mapstructure:"storage"`
	StoragePath string   `mapstructure:"storage-path"`
	Output      string   `mapstructure:"output"`
}

// PluginsConfig holds configuration for plugins
type PluginsConfig struct {
	Directory string            `mapstructure:"directory"`
	Enabled   []string          `mapstructure:"enabled"`
	Config    map[string]string `mapstructure:"config"`
}

// Load loads configuration from viper
func Load() (*Config, error) {
	config := &Config{
		Log: LogConfig{
			Level:  "info",
			Format: "json",
		},
		Collect: CollectConfig{
			Sources:     []string{},
			Workers:     4,
			Storage:     "memory",
			StoragePath: "./logs",
			BatchSize:   100,
		},
		API: APIConfig{
			Host:        "0.0.0.0",
			Port:        8000,
			Timeout:     time.Second * 60,
			Storage:     "memory",
			StoragePath: "./logs",
		},
		Query: QueryConfig{
			Filter:      "",
			Sources:     []string{},
			Limit:       100,
			Storage:     "memory",
			StoragePath: "./logs",
			Output:      "json",
		},
		Plugins: PluginsConfig{
			Directory: "./plugins",
			Enabled:   []string{},
			Config:    make(map[string]string),
		},
	}

	// Parse sources from comma-separated list if not provided as a slice
	sourcesStr := viper.GetString("collect.sources")
	if sourcesStr != "" {
		config.Collect.Sources = strings.Split(sourcesStr, ",")
	} else {
		config.Collect.Sources = viper.GetStringSlice("collect.sources")
	}

	// Extract query sources from comma-separated list if not provided as a slice
	querySources := viper.GetString("query.sources")
	if querySources != "" {
		config.Query.Sources = strings.Split(querySources, ",")
	} else {
		config.Query.Sources = viper.GetStringSlice("query.sources")
	}

	// Extract plugin list from comma-separated list if not provided as a slice
	enabledPlugins := viper.GetString("plugins.enabled")
	if enabledPlugins != "" {
		config.Plugins.Enabled = strings.Split(enabledPlugins, ",")
	} else {
		config.Plugins.Enabled = viper.GetStringSlice("plugins.enabled")
	}

	// Load the rest from viper
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

// validateConfig performs validation on the loaded config
func validateConfig(config *Config) error {
	// Validate log level
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[strings.ToLower(config.Log.Level)] {
		return fmt.Errorf("invalid log level: %s", config.Log.Level)
	}

	// Validate worker count
	if config.Collect.Workers < 1 {
		return fmt.Errorf("invalid worker count: %d (must be at least 1)", config.Collect.Workers)
	}

	// Validate storage type
	validStorage := map[string]bool{
		"memory": true,
		"disk":   true,
	}
	if !validStorage[strings.ToLower(config.Collect.Storage)] {
		return fmt.Errorf("invalid storage type: %s", config.Collect.Storage)
	}

	// Validate query limit
	if config.Query.Limit < 1 {
		return fmt.Errorf("invalid query limit: %d (must be at least 1)", config.Query.Limit)
	}

	return nil
}
