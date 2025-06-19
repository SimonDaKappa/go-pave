package pave

import (
	"bytes"
	"net/http"
	"testing"
)

// BenchmarkOptimizedParsing tests the performance after optimizations
func BenchmarkOptimizedParsing(b *testing.B) {
	validator, err := NewValidator(ValidatorOpts{})
	if err != nil {
		b.Fatalf("Failed to create validator: %v", err)
	}

	// Create a test request with multiple query parameters to stress test the optimization
	jsonBody := `{"id": "123e4567-e89b-12d3-a456-426614174000", "name": "John Doe", "email": "john@example.com", "age": 30}`
	req, err := http.NewRequest("POST",
		"http://example.com/users?name=QueryName&session=sess123&param1=value1&param2=value2&param3=value3&param4=value4&param5=value5",
		bytes.NewBufferString(jsonBody))
	if err != nil {
		b.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret-token")
	req.Header.Set("X-Custom-Header", "custom-value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var user ExampleUser
		err := validator.Validate(req, &user)
		if err != nil {
			b.Fatalf("Failed to validate: %v", err)
		}
	}
}

// BenchmarkQueryParsingStress specifically tests query parameter parsing performance
func BenchmarkQueryParsingStress(b *testing.B) {
	validator, err := NewValidator(ValidatorOpts{})
	if err != nil {
		b.Fatalf("Failed to create validator: %v", err)
	}

	jsonBody := `{"name": "test"}`
	req, err := http.NewRequest("POST",
		"http://example.com/test?name=test&session=sess&user_id=123",
		bytes.NewBufferString(jsonBody))
	if err != nil {
		b.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result ExampleUser
		err := validator.Validate(req, &result)
		if err != nil {
			b.Fatalf("Failed to validate: %v", err)
		}
	}
}

// BenchmarkMemoryAllocations tests memory allocation patterns
func BenchmarkMemoryAllocations(b *testing.B) {
	validator, err := NewValidator(ValidatorOpts{})
	if err != nil {
		b.Fatalf("Failed to create validator: %v", err)
	}

	jsonBody := `{"id": "123e4567-e89b-12d3-a456-426614174000", "name": "John Doe", "email": "john@example.com", "age": 30}`
	req, err := http.NewRequest("POST", "http://example.com/users?name=QueryName&session=sess123", bytes.NewBufferString(jsonBody))
	if err != nil {
		b.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret-token")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var user ExampleUser
		err := validator.Validate(req, &user)
		if err != nil {
			b.Fatalf("Failed to validate: %v", err)
		}
	}
}
