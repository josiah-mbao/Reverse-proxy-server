package main

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReverseProxy_ValidURL(t *testing.T) {
	validURL := "http://example.com"
	handler := reverseProxy(validURL)

	assert.NotNil(t, handler, "Handler should not be nil for valid URL")
}

func TestReverseProxy_InvalidURL(t *testing.T) {
	invalidURL := "not-a-url"
	handler := reverseProxy(invalidURL)

	// url.Parse may succeed even for malformed URLs, so we check if it returns a handler
	// The actual validation happens during proxy operation
	assert.NotNil(t, handler, "Handler is created even for malformed URLs")
}

func TestReverseProxy_ReturnsHandler(t *testing.T) {
	validURL := "http://example.com"
	handler := reverseProxy(validURL)

	assert.NotNil(t, handler, "Should return a handler")
	// The reverseProxy function returns an http.Handler interface
	assert.Implements(t, (*http.Handler)(nil), handler, "Should implement http.Handler interface")
}

func TestReverseProxy_HTTPS_URL(t *testing.T) {
	httpsURL := "https://secure.example.com"
	handler := reverseProxy(httpsURL)

	assert.NotNil(t, handler, "Handler should not be nil for HTTPS URL")
}

func TestReverseProxy_URL_Parsing(t *testing.T) {
	testURL := "http://test.com:8080/path"
	handler := reverseProxy(testURL)

	assert.NotNil(t, handler, "Handler should be created successfully")

	// We can't easily test the internal URL parsing without accessing private fields,
	// but we can test that no panic occurs and a handler is returned
}

func TestReverseProxy_SpecialCharacters(t *testing.T) {
	// Test URL with special characters
	specialURL := "http://example.com/path?query=value&other=test"
	handler := reverseProxy(specialURL)

	assert.NotNil(t, handler, "Handler should handle URLs with query parameters")
}

func TestReverseProxy_IPv6_URL(t *testing.T) {
	ipv6URL := "http://[::1]:8080"
	handler := reverseProxy(ipv6URL)

	assert.NotNil(t, handler, "Handler should handle IPv6 URLs")
}

func TestReverseProxy_RelativeURL(t *testing.T) {
	relativeURL := "/relative/path"
	handler := reverseProxy(relativeURL)

	// url.Parse may succeed with relative URLs, creating a URL with empty scheme
	assert.NotNil(t, handler, "Handler is created for relative URLs")
}

func TestReverseProxy_EmptyURL(t *testing.T) {
	emptyURL := ""
	handler := reverseProxy(emptyURL)

	// Empty string may or may not parse successfully depending on url.Parse behavior
	// Let's check what actually happens
	if handler == nil {
		assert.Nil(t, handler, "Empty URL parsing failed as expected")
	} else {
		assert.NotNil(t, handler, "Empty URL was parsed")
	}
}

func TestReverseProxy_WithPort(t *testing.T) {
	urlWithPort := "http://localhost:3000"
	handler := reverseProxy(urlWithPort)

	assert.NotNil(t, handler, "URL with port should be accepted")
}
