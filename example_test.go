package pave

import (
	"bytes"
	"net/http"
	"reflect"
	"testing"

	"github.com/google/uuid"
)

// Example struct that implements Validatable
type User struct {
	ID        uuid.UUID `header:"X-User-ID,omitempty" query:"user_id,omitempty" json:"id,omitempty"`
	Name      string    `query:"name" json:"name"`
	Email     string    `json:"email,omitempty"`
	Token     string    `header:"Authorization,omitempty"`
	SessionID string    `cookie:"session_id,omitempty" query:"session"`
	Age       int       `json:"age,omitempty"`
}

func (u *User) Validate() error {
	// Add your validation logic here
	if u.Name == "" {
		return ValidationError{reason: "name is required"}
	}
	return nil
}

func TestHTTPRequestParsing(t *testing.T) {
	// Create a validator
	validator, err := NewValidator(ValidatorOpts{})
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Create a test HTTP request with multiple sources
	jsonBody := `{"id": "123e4567-e89b-12d3-a456-426614174000", "name": "John Doe", "email": "john@example.com", "age": 30}`
	req, err := http.NewRequest("POST", "http://example.com/users?user_id=backup-id&name=QueryName&session=sess123", bytes.NewBufferString(jsonBody))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "456e7890-e89b-12d3-a456-426614174001")
	req.Header.Set("Authorization", "Bearer secret-token")

	// Set cookies
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "cookie-session-123"})

	// Parse into struct
	var user User
	err = validator.Validate(req, &user)
	if err != nil {
		t.Fatalf("Failed to validate: %v", err)
	}

	// Verify the parsing worked correctly
	// ID should come from header (first priority)
	expectedID := uuid.MustParse("456e7890-e89b-12d3-a456-426614174001")
	if user.ID != expectedID {
		t.Errorf("Expected ID %v, got %v", expectedID, user.ID)
	}

	// Name should come from query (first priority for this field)
	if user.Name != "QueryName" {
		t.Errorf("Expected Name 'QueryName', got '%s'", user.Name)
	}

	// Email should come from JSON (only source)
	if user.Email != "john@example.com" {
		t.Errorf("Expected Email 'john@example.com', got '%s'", user.Email)
	}

	// Token should come from header with Bearer prefix removed
	if user.Token != "secret-token" {
		t.Errorf("Expected Token 'secret-token', got '%s'", user.Token)
	}

	// SessionID should come from cookie (first priority, since omitempty allows fallback)
	if user.SessionID != "cookie-session-123" {
		t.Errorf("Expected SessionID 'cookie-session-123', got '%s'", user.SessionID)
	}

	// Age should come from JSON
	if user.Age != 30 {
		t.Errorf("Expected Age 30, got %d", user.Age)
	}

	t.Logf("Successfully parsed user: %+v", user)
}

func TestHTTPRequestParsingFallback(t *testing.T) {
	validator, err := NewValidator(ValidatorOpts{})
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Create request without header or cookie, should fallback to query for session
	req, err := http.NewRequest("GET", "http://example.com/users?session=query-session&name=TestUser", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	var user User
	err = validator.Validate(req, &user)
	if err != nil {
		t.Fatalf("Failed to validate: %v", err)
	}

	// SessionID should fallback to query parameter
	if user.SessionID != "query-session" {
		t.Errorf("Expected SessionID 'query-session', got '%s'", user.SessionID)
	}

	// Name should come from query
	if user.Name != "TestUser" {
		t.Errorf("Expected Name 'TestUser', got '%s'", user.Name)
	}

	t.Logf("Successfully parsed user with fallback: %+v", user)
}

func TestExecutionChainCaching(t *testing.T) {
	validator, err := NewValidator(ValidatorOpts{})
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	httpParser := validator.HTTPParser

	// Get chain for User type
	userType := reflect.TypeOf(User{})
	chain1, err := httpParser.GetParseChain(userType)
	if err != nil {
		t.Fatalf("Failed to get parse chain: %v", err)
	}

	// Get chain again - should be cached
	chain2, err := httpParser.GetParseChain(userType)
	if err != nil {
		t.Fatalf("Failed to get cached parse chain: %v", err)
	}

	// Should be the same instance (cached)
	if chain1 != chain2 {
		t.Error("Expected cached chain to be the same instance")
	}

	t.Log("Successfully verified chain caching")
}

func BenchmarkParseChainExecution(b *testing.B) {
	validator, err := NewValidator(ValidatorOpts{})
	if err != nil {
		b.Fatalf("Failed to create validator: %v", err)
	}

	// Create a test request
	jsonBody := `{"id": "123e4567-e89b-12d3-a456-426614174000", "name": "John Doe", "email": "john@example.com", "age": 30}`
	req, err := http.NewRequest("POST", "http://example.com/users?name=QueryName&session=sess123", bytes.NewBufferString(jsonBody))
	if err != nil {
		b.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret-token")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var user User
		err := validator.Validate(req, &user)
		if err != nil {
			b.Fatalf("Failed to validate: %v", err)
		}
	}
}
