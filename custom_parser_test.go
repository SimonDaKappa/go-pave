package pave

import (
	"testing"
)

func TestCustomMapParser(t *testing.T) {
	// Create a validator with our custom map parser
	mapParser := NewMapSourceParser()
	validator, err := NewValidator(ValidatorOpts{
		Parsers: []SourceParser{mapParser},
	})
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Test data
	configMap := map[string]string{
		"db_url": "postgresql://localhost:5432/mydb",
		"port":   "8080",
		"debug":  "true",
	}

	// Parse into struct
	var config ConfigFromMap
	err = validator.Validate(configMap, &config)
	if err != nil {
		t.Fatalf("Failed to validate: %v", err)
	}

	// Verify the parsing worked
	if config.DatabaseURL != "postgresql://localhost:5432/mydb" {
		t.Errorf("Expected DatabaseURL 'postgresql://localhost:5432/mydb', got '%s'", config.DatabaseURL)
	}
	if config.Port != 8080 {
		t.Errorf("Expected Port 8080, got %d", config.Port)
	}
	if config.Debug != true {
		t.Errorf("Expected Debug true, got %t", config.Debug)
	}

	t.Logf("Successfully parsed config: %+v", config)
}

func TestCustomParserWithMissingRequiredField(t *testing.T) {
	mapParser := NewMapSourceParser()
	validator, err := NewValidator(ValidatorOpts{
		Parsers: []SourceParser{mapParser},
	})
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Missing required db_url
	configMap := map[string]string{
		"port":  "8080",
		"debug": "false",
	}

	var config ConfigFromMap
	err = validator.Validate(configMap, &config)
	if err == nil {
		t.Error("Expected validation to fail due to missing required db_url")
	}

	t.Logf("Correctly failed with error: %v", err)
}

func TestCustomParserWithOptionalFields(t *testing.T) {
	mapParser := NewMapSourceParser()
	validator, err := NewValidator(ValidatorOpts{
		Parsers: []SourceParser{mapParser},
	})
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Only required field
	configMap := map[string]string{
		"db_url": "postgresql://localhost:5432/mydb",
	}

	var config ConfigFromMap
	err = validator.Validate(configMap, &config)
	if err != nil {
		t.Fatalf("Failed to validate: %v", err)
	}

	// Optional fields should have zero values
	if config.DatabaseURL != "postgresql://localhost:5432/mydb" {
		t.Errorf("Expected DatabaseURL 'postgresql://localhost:5432/mydb', got '%s'", config.DatabaseURL)
	}
	if config.Port != 0 {
		t.Errorf("Expected Port 0 (zero value), got %d", config.Port)
	}
	if config.Debug != false {
		t.Errorf("Expected Debug false (zero value), got %t", config.Debug)
	}

	t.Logf("Successfully parsed minimal config: %+v", config)
}
