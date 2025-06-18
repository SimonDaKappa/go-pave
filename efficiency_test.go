package pave

import (
	"bytes"
	"net/http"
	"testing"
)

// ValidatableEfficientParsing wraps our test struct to implement Validatable
type ValidatableEfficientParsing struct {
	HeaderField string `header:"X-Header-Field,omitempty" query:"header_field,omitempty" json:"header_field"`
	QueryField  string `query:"query_field,omitempty" json:"query_field"`
	JSONField   string `json:"json_field"`   // This will trigger JSON parsing
	AnotherJSON string `json:"another_json"` // This will reuse the already parsed JSON
}

func (vep *ValidatableEfficientParsing) Validate() error {
	return nil
}

// Demonstrates efficient JSON parsing - body is only parsed when a JSON field is first encountered
func TestJSONParsingPriority(t *testing.T) {
	validator, err := NewValidator(ValidatorOpts{})
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	jsonBody := `{"header_field": "from_json", "query_field": "from_json", "json_field": "json_value", "another_json": "another_value"}`

	// Test 1: All data available in query/header, JSON should not be parsed until json_field
	req1, _ := http.NewRequest("POST", "http://example.com?query_field=from_query&header_field=from_query", bytes.NewBufferString(jsonBody))
	req1.Header.Set("X-Header-Field", "from_header")
	req1.Header.Set("Content-Type", "application/json")

	result1 := &ValidatableEfficientParsing{}
	err = validator.Validate(req1, result1)
	if err != nil {
		t.Fatalf("Failed to validate: %v", err)
	}

	// Verify priority worked correctly
	if result1.HeaderField != "from_header" {
		t.Errorf("Expected HeaderField from header, got %s", result1.HeaderField)
	}
	if result1.QueryField != "from_query" {
		t.Errorf("Expected QueryField from query, got %s", result1.QueryField)
	}
	if result1.JSONField != "json_value" {
		t.Errorf("Expected JSONField from json, got %s", result1.JSONField)
	}
	if result1.AnotherJSON != "another_value" {
		t.Errorf("Expected AnotherJSON from json, got %s", result1.AnotherJSON)
	}

	t.Logf("Efficient parsing result: %+v", result1)

	// Test 2: Missing header/query data, should fallback to JSON
	req2, _ := http.NewRequest("POST", "http://example.com", bytes.NewBufferString(jsonBody))
	req2.Header.Set("Content-Type", "application/json")

	result2 := &ValidatableEfficientParsing{}
	err = validator.Validate(req2, result2)
	if err != nil {
		t.Fatalf("Failed to validate: %v", err)
	}

	// All should come from JSON since other sources aren't available
	if result2.HeaderField != "from_json" {
		t.Errorf("Expected HeaderField from json fallback, got %s", result2.HeaderField)
	}
	if result2.QueryField != "from_json" {
		t.Errorf("Expected QueryField from json fallback, got %s", result2.QueryField)
	}

	t.Logf("Fallback parsing result: %+v", result2)
}
