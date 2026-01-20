package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig_Defaults(t *testing.T) {
	helper := SetupTestEnv()
	defer helper.RestoreEnv()

	config, err := LoadTestConfig()
	assert.NoError(t, err)
	assert.Equal(t, 8080, config.Port)
	assert.Equal(t, "http://127.0.0.1:5000", config.Backend)
	assert.Equal(t, "info", config.LogLevel)
}

func TestLoadConfig_EnvironmentVariables(t *testing.T) {
	helper := SetupTestEnv()
	defer helper.RestoreEnv()

	helper.SetEnv("PROXY_PORT", "9090")
	helper.SetEnv("PROXY_BACKEND", "http://test.com")
	helper.SetEnv("PROXY_LOG_LEVEL", "debug")

	config, err := LoadTestConfig()
	assert.NoError(t, err)
	assert.Equal(t, 9090, config.Port)
	assert.Equal(t, "http://test.com", config.Backend)
	assert.Equal(t, "debug", config.LogLevel)
}

func TestLoadConfig_JSONFile(t *testing.T) {
	helper := SetupTestEnv()
	defer helper.RestoreEnv()

	helper.SetEnv("PROXY_CONFIG_FILE", "testdata/config.json")

	config, err := LoadTestConfig()
	assert.NoError(t, err)
	assert.Equal(t, 9090, config.Port)
	assert.Equal(t, "http://test-backend.com", config.Backend)
	assert.Equal(t, "debug", config.LogLevel)
}

func TestLoadConfig_Precedence(t *testing.T) {
	helper := SetupTestEnv()
	defer helper.RestoreEnv()

	// Set environment variables
	helper.SetEnv("PROXY_PORT", "9090")
	helper.SetEnv("PROXY_BACKEND", "http://env-backend.com")
	helper.SetEnv("PROXY_LOG_LEVEL", "warn")

	// Set config file
	helper.SetEnv("PROXY_CONFIG_FILE", "testdata/config.json")

	// Test that env vars override file
	config, err := LoadTestConfig()
	assert.NoError(t, err)
	assert.Equal(t, 9090, config.Port)          // from env
	assert.Equal(t, "http://env-backend.com", config.Backend) // from env
	assert.Equal(t, "warn", config.LogLevel)    // from env
}

func TestLoadConfig_PartialConfig(t *testing.T) {
	helper := SetupTestEnv()
	defer helper.RestoreEnv()

	helper.SetEnv("PROXY_CONFIG_FILE", "testdata/partial_config.json")

	config, err := LoadTestConfig()
	assert.NoError(t, err)
	assert.Equal(t, 9999, config.Port)
	assert.Equal(t, "http://127.0.0.1:5000", config.Backend) // default
	assert.Equal(t, "info", config.LogLevel)                 // default
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	helper := SetupTestEnv()
	defer helper.RestoreEnv()

	helper.SetEnv("PROXY_CONFIG_FILE", "testdata/invalid_config.json")

	_, err := LoadTestConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load config file")
}

func TestLoadConfig_MissingFile(t *testing.T) {
	helper := SetupTestEnv()
	defer helper.RestoreEnv()

	helper.SetEnv("PROXY_CONFIG_FILE", "testdata/nonexistent.json")

	_, err := LoadTestConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load config file")
}
