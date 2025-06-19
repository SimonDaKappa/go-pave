package pave

import (
	"testing"
)

// Test structs at package level
type TestStructForMapValidation struct {
	Name  string `mapvalue:"name"`
	Email string `mapvalue:"email"`
	Age   int    `mapvalue:"age"`
}

func (v *TestStructForMapValidation) Validate() error {
	if v.Name == "" {
		return ValidationError{reason: "name is required"}
	}
	return nil
}

type TestStructForJSONValidation struct {
	Name string `json:"name"`
}

func (v *TestStructForJSONValidation) Validate() error {
	return nil
}

type BinaryParser struct {
	*JsonSourceParser
}

func (bp *BinaryParser) GetParserName() string {
	return "binary"
}

type GlobalTestStruct struct {
	Name  string `mapvalue:"name"`
	Email string `mapvalue:"email"`
}

func (v *GlobalTestStruct) Validate() error {
	if v.Name == "" {
		return ValidationError{reason: "name is required"}
	}
	return nil
}

func TestMultipleParserRegistration(t *testing.T) {
	// Create a new validator
	validator, err := NewValidator(ValidatorOpts{})
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Register JSON parser for []byte
	jsonParser := NewJsonSourceParser()
	err = validator.RegisterParser(jsonParser)
	if err != nil {
		t.Fatalf("Failed to register JSON parser: %v", err)
	}

	// Try to register another parser with the same name - should fail
	jsonParser2 := NewJsonSourceParser()
	err = validator.RegisterParser(jsonParser2)
	if err == nil {
		t.Error("Expected error when registering parser with same name")
	}
	if err != ErrParserAlreadyRegistered {
		t.Errorf("Expected ErrParserAlreadyRegistered, got: %v", err)
	}
}

func TestSingleParserAutoSelection(t *testing.T) {
	// Create a new validator
	validator, err := NewValidator(ValidatorOpts{})
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Register string map parser
	stringMapParser := NewStringMapSourceParser()
	err = validator.RegisterParser(stringMapParser)
	if err != nil {
		t.Fatalf("Failed to register string map parser: %v", err)
	}

	// Test data
	mapData := map[string]string{
		"name":  "Test User",
		"email": "test@example.com",
		"age":   "25",
	}

	var result TestStructForMapValidation

	// Should work automatically since only one parser for map[string]string
	err = validator.Validate(mapData, &result)
	if err != nil {
		t.Fatalf("Failed to validate with single parser: %v", err)
	}

	if result.Name != "Test User" {
		t.Errorf("Expected name 'Test User', got: %s", result.Name)
	}
}

func TestMultipleParserRequiresSpecification(t *testing.T) {
	// Create a new validator
	validator, err := NewValidator(ValidatorOpts{})
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Register JSON parser for []byte
	jsonParser := NewJsonSourceParser()
	err = validator.RegisterParser(jsonParser)
	if err != nil {
		t.Fatalf("Failed to register JSON parser: %v", err)
	}

	// Create another parser for []byte with different name
	binaryParser := &BinaryParser{NewJsonSourceParser()}
	err = validator.RegisterParser(binaryParser)
	if err != nil {
		t.Fatalf("Failed to register binary parser: %v", err)
	}

	// Test data
	jsonData := []byte(`{"name": "Test"}`)

	var result TestStructForJSONValidation

	// Should fail since multiple parsers are available
	err = validator.Validate(jsonData, &result)
	if err == nil {
		t.Error("Expected error when multiple parsers available")
	}
	if err != ErrMultipleParsersAvailable {
		t.Errorf("Expected ErrMultipleParsersAvailable, got: %v", err)
	}

	// Should work with specific parser
	err = validator.WithParser("json").Validate(jsonData, &result)
	if err != nil {
		t.Fatalf("Failed to validate with specific parser: %v", err)
	}

	if result.Name != "Test" {
		t.Errorf("Expected name 'Test', got: %s", result.Name)
	}
}

func TestGlobalValidatorFunctions(t *testing.T) {
	// Test that package-level functions work

	// Register a parser globally
	stringMapParser := NewStringMapSourceParser()
	err := RegisterParser(stringMapParser)
	if err != nil {
		t.Fatalf("Failed to register parser globally: %v", err)
	}

	// Test data
	mapData := map[string]string{
		"name":  "Global Test",
		"email": "global@example.com",
	}

	var result GlobalTestStruct

	// Use global Validate function
	err = Validate(mapData, &result)
	if err != nil {
		t.Fatalf("Failed to validate using global function: %v", err)
	}

	if result.Name != "Global Test" {
		t.Errorf("Expected name 'Global Test', got: %s", result.Name)
	}
}
