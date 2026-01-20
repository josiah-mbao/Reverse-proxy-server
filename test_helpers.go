package main

import (
	"bytes"
	"log"
	"os"
)

// TestHelper provides utilities for testing
type TestHelper struct {
	originalEnv map[string]string
	logBuffer   *bytes.Buffer
}

// SetupTestEnv captures current environment and sets up test environment
func SetupTestEnv() *TestHelper {
	helper := &TestHelper{
		originalEnv: make(map[string]string),
		logBuffer:   &bytes.Buffer{},
	}

	// Capture original environment
	envVars := []string{"PROXY_PORT", "PROXY_BACKEND", "PROXY_LOG_LEVEL", "PROXY_CONFIG_FILE"}
	for _, envVar := range envVars {
		helper.originalEnv[envVar] = os.Getenv(envVar)
		os.Unsetenv(envVar)
	}

	// Redirect log output
	log.SetOutput(helper.logBuffer)

	return helper
}

// RestoreEnv restores the original environment
func (h *TestHelper) RestoreEnv() {
	for key, value := range h.originalEnv {
		if value == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, value)
		}
	}
}

// SetEnv sets an environment variable for testing
func (h *TestHelper) SetEnv(key, value string) {
	os.Setenv(key, value)
}

// GetLogs returns the captured log output
func (h *TestHelper) GetLogs() string {
	return h.logBuffer.String()
}

// ClearLogs clears the log buffer
func (h *TestHelper) ClearLogs() {
	h.logBuffer.Reset()
}
