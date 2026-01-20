package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
)

// Config holds all configuration for the proxy server
type Config struct {
	Port            int    `json:"port"`
	Backend         string `json:"backend"`
	LogLevel        string `json:"log_level"`
	CacheEnabled    bool   `json:"cache_enabled"`
	CacheSize       int    `json:"cache_size"`
	CacheTTL        int    `json:"cache_ttl_seconds"`
	RequestTimeout  int    `json:"request_timeout_seconds"`
	ShutdownTimeout int    `json:"shutdown_timeout_seconds"`
	RateLimitEnabled bool   `json:"rate_limit_enabled"`
	RateLimitRPM    int    `json:"rate_limit_requests_per_minute"`
	RateLimitBurst  int    `json:"rate_limit_burst_size"`
}

// LoadConfig loads configuration from environment variables, config file, and command-line flags
func LoadConfig() (*Config, error) {
	config := &Config{
		Port:             8080,
		Backend:          "http://127.0.0.1:5000",
		LogLevel:         "info",
		CacheEnabled:     false,
		CacheSize:        100,
		CacheTTL:         300, // 5 minutes
		RequestTimeout:   30,  // 30 seconds
		ShutdownTimeout:  30,  // 30 seconds
		RateLimitEnabled: false,
		RateLimitRPM:     100, // 100 requests per minute
		RateLimitBurst:   20,  // burst size
	}

	// Load from environment variables and config file
	if err := loadConfigFromEnvAndFile(config); err != nil {
		return nil, err
	}

	// Apply command-line flags (only if not already parsed)
	if !flag.Parsed() {
		portFlag := flag.Int("port", config.Port, "Port to listen on")
		backendFlag := flag.String("backend", config.Backend, "Backend server URL")
		logLevelFlag := flag.String("log-level", config.LogLevel, "Log level (debug, info, warn, error)")
		configFileFlag := flag.String("config", "", "Path to config file")

		flag.Parse()

		// Apply command-line flags
		config.Port = *portFlag
		config.Backend = *backendFlag
		config.LogLevel = *logLevelFlag

		// Load config file from flag if specified
		if *configFileFlag != "" {
			if err := loadConfigFromFile(*configFileFlag, config); err != nil {
				return nil, fmt.Errorf("failed to load config file: %v", err)
			}
		}
	}

	return config, nil
}

// loadConfigFromEnvAndFile loads configuration from environment variables and config file
func loadConfigFromEnvAndFile(config *Config) error {
	// Load from config file first (if it exists)
	if configFile := os.Getenv("PROXY_CONFIG_FILE"); configFile != "" {
		if err := loadConfigFromFile(configFile, config); err != nil {
			return fmt.Errorf("failed to load config file: %v", err)
		}
	}

	// Environment variables override config file
	if portStr := os.Getenv("PROXY_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			config.Port = port
		}
	}

	if backend := os.Getenv("PROXY_BACKEND"); backend != "" {
		config.Backend = backend
	}

	if logLevel := os.Getenv("PROXY_LOG_LEVEL"); logLevel != "" {
		config.LogLevel = logLevel
	}

	return nil
}

// LoadTestConfig loads configuration for testing (without parsing flags)
func LoadTestConfig() (*Config, error) {
	config := &Config{
		Port:     8080,
		Backend:  "http://127.0.0.1:5000",
		LogLevel: "info",
	}

	return config, loadConfigFromEnvAndFile(config)
}

// loadConfigFromFile loads configuration from a JSON file
func loadConfigFromFile(filename string, config *Config) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(config)
}
