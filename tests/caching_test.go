package pave

import (
	"net/http"
	"net/url"
	"testing"
)

// TestHeaderCaching verifies that headers are cached after first access
func TestHeaderCaching(t *testing.T) {
	// Create a test request with headers
	req := &http.Request{
		Header: make(http.Header),
		URL:    &url.URL{},
	}
	req.Header.Set("X-Test-Header", "test-value")
	req.Header.Set("Authorization", "Bearer token123")

	data := &HTTPRequestData{request: req}
	parser := NewHTTPRequestParser()

	// First access should populate the cache
	value1, exists1, err1 := parser.getHeaderValue(data, "X-Test-Header")
	if err1 != nil {
		t.Fatalf("First header access failed: %v", err1)
	}
	if !exists1 || value1 != "test-value" {
		t.Fatalf("Expected 'test-value', got %v (exists: %v)", value1, exists1)
	}

	// Second access should use cache (verify by modifying the original header)
	req.Header.Set("X-Test-Header", "modified-value")
	value2, exists2, err2 := parser.getHeaderValue(data, "X-Test-Header")
	if err2 != nil {
		t.Fatalf("Second header access failed: %v", err2)
	}
	if !exists2 || value2 != "test-value" {
		t.Fatalf("Cache not working: expected 'test-value', got %v (exists: %v)", value2, exists2)
	}

	// Test Authorization header special handling
	authValue, authExists, authErr := parser.getHeaderValue(data, "Authorization")
	if authErr != nil {
		t.Fatalf("Authorization header access failed: %v", authErr)
	}
	if !authExists || authValue != "token123" {
		t.Fatalf("Authorization header handling failed: expected 'token123', got %v (exists: %v)", authValue, authExists)
	}
}

// TestCookieCaching verifies that cookies are cached after first access
func TestCookieCaching(t *testing.T) {
	// Create a test request with cookies
	req := &http.Request{
		Header: make(http.Header),
		URL:    &url.URL{},
	}
	req.Header.Set("Cookie", "session_id=abc123; user_pref=dark_mode")

	data := &HTTPRequestData{request: req}
	parser := NewHTTPRequestParser()

	// First access should populate the cache
	value1, exists1, err1 := parser.getCookieValue(data, "session_id")
	if err1 != nil {
		t.Fatalf("First cookie access failed: %v", err1)
	}
	if !exists1 || value1 != "abc123" {
		t.Fatalf("Expected 'abc123', got %v (exists: %v)", value1, exists1)
	}

	// Second access should use cache (verify by modifying the original cookie)
	req.Header.Set("Cookie", "session_id=modified; user_pref=light_mode")
	value2, exists2, err2 := parser.getCookieValue(data, "session_id")
	if err2 != nil {
		t.Fatalf("Second cookie access failed: %v", err2)
	}
	if !exists2 || value2 != "abc123" {
		t.Fatalf("Cache not working: expected 'abc123', got %v (exists: %v)", value2, exists2)
	}

	// Test accessing another cookie
	prefValue, prefExists, prefErr := parser.getCookieValue(data, "user_pref")
	if prefErr != nil {
		t.Fatalf("User preference cookie access failed: %v", prefErr)
	}
	if !prefExists || prefValue != "dark_mode" {
		t.Fatalf("Expected 'dark_mode', got %v (exists: %v)", prefValue, prefExists)
	}

	// Test non-existent cookie
	nonValue, nonExists, nonErr := parser.getCookieValue(data, "non_existent")
	if nonErr != nil {
		t.Fatalf("Non-existent cookie access failed: %v", nonErr)
	}
	if nonExists {
		t.Fatalf("Non-existent cookie should not exist, but got %v", nonValue)
	}
}

// TestQueryCaching verifies that query parameters maintain their caching behavior
func TestQueryCaching(t *testing.T) {
	// Create a test request with query parameters
	u, _ := url.Parse("http://example.com/test?param1=value1&param2=value2")
	req := &http.Request{
		URL: u,
	}

	data := &HTTPRequestData{request: req}
	parser := NewHTTPRequestParser()

	// First access should populate the cache
	value1, exists1, err1 := parser.getQueryValue(data, "param1")
	if err1 != nil {
		t.Fatalf("First query access failed: %v", err1)
	}
	if !exists1 || value1 != "value1" {
		t.Fatalf("Expected 'value1', got %v (exists: %v)", value1, exists1)
	}

	// Modify the URL to verify caching
	req.URL.RawQuery = "param1=modified&param2=modified"
	value2, exists2, err2 := parser.getQueryValue(data, "param1")
	if err2 != nil {
		t.Fatalf("Second query access failed: %v", err2)
	}
	if !exists2 || value2 != "value1" {
		t.Fatalf("Cache not working: expected 'value1', got %v (exists: %v)", value2, exists2)
	}
}
