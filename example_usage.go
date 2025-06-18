package pave

import (
	"bytes"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
)

// Example user struct with multiple source tags
type CreateUserRequest struct {
	ID       uuid.UUID `header:"X-User-ID,omitempty" query:"user_id,omitempty" json:"id"`
	Name     string    `query:"name,omitempty" json:"name"`
	Email    string    `json:"email"`
	Token    string    `header:"Authorization"`
	AdminKey string    `header:"X-Admin-Key,omitempty" query:"admin_key,omitempty"`
	Age      int       `json:"age,omitempty"`
	Active   bool      `query:"active,omitempty" json:"active,omitempty"`
}

func (c *CreateUserRequest) Validate() error {
	if c.Name == "" {
		return ValidationError{reason: "name is required"}
	}
	if c.Email == "" {
		return ValidationError{reason: "email is required"}
	}
	return nil
}

func ExampleUsage() {
	// Create the validator
	validator, err := NewValidator(ValidatorOpts{})
	if err != nil {
		log.Fatalf("Failed to create validator: %v", err)
	}

	// Example 1: Request with all sources
	fmt.Println("=== Example 1: Multiple Sources ===")

	jsonBody := `{
		"id": "123e4567-e89b-12d3-a456-426614174000",
		"name": "John from JSON", 
		"email": "john@example.com",
		"age": 30,
		"active": true
	}`

	req1, _ := http.NewRequest("POST", "http://example.com/users?name=John+from+Query&admin_key=secret123&active=false", bytes.NewBufferString(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("X-User-ID", "456e7890-e89b-12d3-a456-426614174001")
	req1.Header.Set("Authorization", "Bearer my-secret-token")
	req1.Header.Set("X-Admin-Key", "header-admin-key")

	var user1 CreateUserRequest
	if err := validator.Validate(req1, &user1); err != nil {
		log.Printf("Validation failed: %v", err)
	} else {
		fmt.Printf("Parsed user: %+v\n", user1)
		fmt.Printf("- ID came from header: %v\n", user1.ID)
		fmt.Printf("- Name came from query (first priority): %s\n", user1.Name)
		fmt.Printf("- Email came from JSON: %s\n", user1.Email)
		fmt.Printf("- Token came from header: %s\n", user1.Token)
		fmt.Printf("- AdminKey came from header (first priority): %s\n", user1.AdminKey)
		fmt.Printf("- Age came from JSON: %d\n", user1.Age)
		fmt.Printf("- Active came from query (first priority): %t\n", user1.Active)
	}

	// Example 2: Fallback behavior
	fmt.Println("\n=== Example 2: Fallback Sources ===")

	req2, _ := http.NewRequest("GET", "http://example.com/users?user_id=fallback-uuid&name=Fallback+User&admin_key=query-admin", nil)
	req2.Header.Set("Authorization", "Bearer fallback-token")

	var user2 CreateUserRequest
	if err := validator.Validate(req2, &user2); err != nil {
		// This should fail because email is required but not provided
		fmt.Printf("Expected validation failure: %v\n", err)
	}

	// Example 3: JSON-only request
	fmt.Println("\n=== Example 3: JSON Only ===")

	jsonOnlyBody := `{
		"id": "789e1234-e89b-12d3-a456-426614174002",
		"name": "JSON Only User",
		"email": "json@example.com",
		"age": 25
	}`

	req3, _ := http.NewRequest("POST", "http://example.com/users", bytes.NewBufferString(jsonOnlyBody))
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("Authorization", "Bearer json-token")

	var user3 CreateUserRequest
	if err := validator.Validate(req3, &user3); err != nil {
		log.Printf("Validation failed: %v", err)
	} else {
		fmt.Printf("Parsed JSON-only user: %+v\n", user3)
		fmt.Printf("- All data came from JSON except token from header\n")
	}
}
