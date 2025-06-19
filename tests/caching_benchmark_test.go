package pave

import (
	"net/http"
	"net/url"
	"testing"
)

// BenchmarkHeaderAccessWithoutCache benchmarks direct header access (simulating old behavior)
func BenchmarkHeaderAccessWithoutCache(b *testing.B) {
	req := &http.Request{
		Header: make(http.Header),
		URL:    &url.URL{},
	}
	req.Header.Set("X-Test-Header", "test-value")
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "test-agent")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate multiple header accesses like the old implementation
		_ = req.Header.Get("X-Test-Header")
		_ = req.Header.Get("Authorization")
		_ = req.Header.Get("Content-Type")
		_ = req.Header.Get("User-Agent")
	}
}

// BenchmarkHeaderAccessWithCache benchmarks cached header access
func BenchmarkHeaderAccessWithCache(b *testing.B) {
	req := &http.Request{
		Header: make(http.Header),
		URL:    &url.URL{},
	}
	req.Header.Set("X-Test-Header", "test-value")
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "test-agent")

	data := &HTTPRequestData{request: req}
	parser := NewHTTPRequestParser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Access headers through cached implementation
		parser.getHeaderValue(data, "X-Test-Header")
		parser.getHeaderValue(data, "Authorization")
		parser.getHeaderValue(data, "Content-Type")
		parser.getHeaderValue(data, "User-Agent")
	}
}

// BenchmarkCookieAccessWithoutCache benchmarks direct cookie access (simulating old behavior)
func BenchmarkCookieAccessWithoutCache(b *testing.B) {
	req := &http.Request{
		Header: make(http.Header),
		URL:    &url.URL{},
	}
	req.Header.Set("Cookie", "session_id=abc123; user_pref=dark_mode; cart_id=xyz789; theme=light")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate multiple cookie accesses like the old implementation
		req.Cookie("session_id")
		req.Cookie("user_pref")
		req.Cookie("cart_id")
		req.Cookie("theme")
	}
}

// BenchmarkCookieAccessWithCache benchmarks cached cookie access
func BenchmarkCookieAccessWithCache(b *testing.B) {
	req := &http.Request{
		Header: make(http.Header),
		URL:    &url.URL{},
	}
	req.Header.Set("Cookie", "session_id=abc123; user_pref=dark_mode; cart_id=xyz789; theme=light")

	data := &HTTPRequestData{request: req}
	parser := NewHTTPRequestParser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Access cookies through cached implementation
		parser.getCookieValue(data, "session_id")
		parser.getCookieValue(data, "user_pref")
		parser.getCookieValue(data, "cart_id")
		parser.getCookieValue(data, "theme")
	}
}
