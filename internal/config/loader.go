// Package config provides configuration management for the Clever Better application.
package config

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Load reads and parses the configuration from file and environment variables
// It expands environment variable placeholders in the YAML file (${VAR_NAME})
func Load(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = "config/config.yaml"
	}

	// Read the configuration file
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found at %s: %w", configPath, err)
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables in the configuration (${VAR} syntax)
	expanded := os.ExpandEnv(string(data))

	// Create a new viper instance
	v := viper.New()
	v.SetConfigType("yaml")

	// Read the expanded configuration
	if err := v.ReadConfig(bytes.NewBuffer([]byte(expanded))); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set environment variable prefix
	v.SetEnvPrefix("CLEVER_BETTER")

	// Enable automatic binding of environment variables
	v.AutomaticEnv()

	// Replace dots with underscores in environment variable names
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Unmarshal configuration into Config struct
	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	return cfg, nil
}

// LoadWithDefaults loads configuration with default values for optional fields
// It expands environment variable placeholders in the YAML file (${VAR_NAME})
func LoadWithDefaults(configPath string) (*Config, error) {
	v := viper.New()

	// Set configuration file path with default
	if configPath == "" {
		configPath = "config/config.yaml"
	}

	v.SetConfigType("yaml")

	// Set environment variable prefix
	v.SetEnvPrefix("CLEVER_BETTER")

	// Enable automatic binding of environment variables
	v.AutomaticEnv()

	// Replace dots with underscores in environment variable names
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set some reasonable defaults
	v.SetDefault("app.environment", "development")
	v.SetDefault("app.log_level", "info")
	v.SetDefault("metrics.enabled", true)
	v.SetDefault("features.paper_trading_enabled", true)

	// Read and expand the configuration file if it exists
	if data, err := os.ReadFile(configPath); err == nil {
		// Expand environment variables in the configuration (${VAR} syntax)
		expanded := os.ExpandEnv(string(data))
		if err := v.ReadConfig(bytes.NewBuffer([]byte(expanded))); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	// If file doesn't exist, continue with defaults and environment variables

	// Unmarshal configuration into Config struct
	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	return cfg, nil
}

// ReloadFromEnv reloads specific configuration values from environment variables
func ReloadFromEnv(cfg *Config) error {
	v := viper.New()

	// Set environment variable prefix
	v.SetEnvPrefix("CLEVER_BETTER")

	// Enable automatic binding of environment variables
	v.AutomaticEnv()

	// Replace dots with underscores in environment variable names
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Check for specific environment variables and update the config
	if envPath := os.Getenv("CLEVER_BETTER_CONFIG_PATH"); envPath != "" {
		newCfg, err := Load(envPath)
		if err != nil {
			return err
		}
		*cfg = *newCfg
	}

	return nil
}
