package pave

import (
	"fmt"
	"log"
)

// ExampleStruct demonstrates validation with multiple parser support
type ExampleStruct struct {
	Name  string `json:"name" mapvalue:"name"`
	Email string `json:"email" mapvalue:"email"`
	Age   int    `json:"age" mapvalue:"age"`
}

func (e *ExampleStruct) Validate() error {
	if e.Name == "" {
		return fmt.Errorf("name is required")
	}
	if e.Email == "" {
		return fmt.Errorf("email is required")
	}
	if e.Age < 0 {
		return fmt.Errorf("age must be non-negative")
	}
	return nil
}

func ExampleMultipleParsers() {
	// Create parsers for the same source type ([]byte)
	jsonParser := NewJsonSourceParser()

	// Register both parsers with the global validator
	err := RegisterParser(jsonParser)
	if err != nil {
		log.Fatalf("Failed to register JSON parser: %v", err)
	}

	// Example 1: Using []byte with JSON data - need to specify parser since multiple exist
	jsonData := []byte(`{"name": "John Doe", "email": "john@example.com", "age": 30}`)

	var result1 ExampleStruct

	// This would fail because multiple parsers are registered for []byte
	err = Validate(jsonData, &result1)
	if err != nil {
		fmt.Printf("Expected error when multiple parsers available: %v\n", err)
	}

	// Use WithParser to specify which parser to use
	err = WithParser("json").Validate(jsonData, &result1)
	if err != nil {
		log.Fatalf("Failed to validate with JSON parser: %v", err)
	}

	fmt.Printf("JSON parsing result: %+v\n", result1)

	// Example 2: Using map[string]string with string map parser
	mapData := map[string]string{
		"name":  "Jane Smith",
		"email": "jane@example.com",
		"age":   "25",
	}

	// Register string map parser
	stringMapParser := NewStringMapSourceParser()
	err = RegisterParser(stringMapParser)
	if err != nil {
		log.Fatalf("Failed to register string map parser: %v", err)
	}

	var result2 ExampleStruct

	// This should work automatically since only one parser is registered for map[string]string
	err = Validate(mapData, &result2)
	if err != nil {
		log.Fatalf("Failed to validate with string map parser: %v", err)
	}

	fmt.Printf("String map parsing result: %+v\n", result2)

	// Example 3: Demonstrating the currying with different parsers
	// If we had another []byte parser (like a binary parser), we could switch between them:

	// Using specific parser names
	ctx1 := WithParser("json")
	err = ctx1.Validate(jsonData, &result1)
	if err != nil {
		log.Fatalf("Failed with curried JSON parser: %v", err)
	}

	fmt.Printf("Curried JSON parser result: %+v\n", result1)
}
