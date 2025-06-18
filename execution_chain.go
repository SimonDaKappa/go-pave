package pave

import (
	"fmt"
	"reflect"
)

///////////////////////////////////////////////////////////////////////////////
// Core Execution Chain Types
///////////////////////////////////////////////////////////////////////////////

type ParseExecutionChain interface {
	Execute(v Validatable) error
}

// FieldSource represents a single source for a field with its configuration
type FieldSource struct {
	Source    string // The source type (json, cookie, header, query, etc.)
	Key       string // The name/key for the source
	OmitEmpty bool   // If true, continue to next source if not found
	Required  bool   // If true, this is the final source to try
}

// ParseStep represents a single step in the execution chain
type ParseStep struct {
	FieldIndex int           // Index of the field in the struct
	FieldName  string        // Name of the field for error reporting
	Sources    []FieldSource // Ordered list of sources to try
	Next       *ParseStep    // Next step in the chain
}

///////////////////////////////////////////////////////////////////////////////
// BaseExecutionChain
///////////////////////////////////////////////////////////////////////////////

// SourceGetter is a function that retrieves a value from a specific source
// Returns: (value, found, error)
type SourceGetter func(sourceData any, source FieldSource) (any, bool, error)

// BaseExecutionChain represents a linked list of parse steps for a struct type
//
// Uses a function-based approach for source value retrieval, eliminating
// the need for each parser to reimplement the same linked list traversal logic.
// The SourceGetter function provides dynamic dispatch to the appropriate
// value retrieval method for each parser type.
type BaseExecutionChain struct {
	StructType   reflect.Type
	Head         *ParseStep
	SourceGetter SourceGetter // Function to get values from sources
}

// Execute runs the entire parse chain using the provided source getter
func (bec *BaseExecutionChain) Execute(sourceData any, dest Validatable) error {
	current := bec.Head
	for current != nil {
		if err := bec.executeStep(sourceData, dest, current); err != nil {
			return fmt.Errorf("failed to parse field %s: %w", current.FieldName, err)
		}
		current = current.Next
	}
	return nil
}

// executeStep executes a single parse step
func (bec *BaseExecutionChain) executeStep(sourceData any, dest Validatable, step *ParseStep) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() == reflect.Ptr {
		destValue = destValue.Elem()
	}

	field := destValue.Field(step.FieldIndex)
	if !field.CanSet() {
		return nil // Skip non-settable fields
	}

	// Try each source in order
	var lastErr error
	allOmitEmpty := true

	for _, source := range step.Sources {
		allOmitEmpty = allOmitEmpty && source.OmitEmpty

		value, found, err := bec.SourceGetter(sourceData, source)
		if err != nil {
			lastErr = err
			if source.Required {
				return err
			}
			continue
		}

		if found {
			return setFieldValue(field, fmt.Sprintf("%v", value))
		}

		if source.Required {
			return fmt.Errorf("required field %s not found in source %s", source.Key, source.Source)
		}
	}

	// If all sources have omitempty and none succeeded, that's ok
	if allOmitEmpty {
		return nil
	}

	return lastErr
}
