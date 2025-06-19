package pave

import (
	"fmt"
	"reflect"
	"sync"
)

///////////////////////////////////////////////////////////////////////////////
// Core Execution Chain Types
///////////////////////////////////////////////////////////////////////////////

type ExecutionChain interface {
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
// ChainExecutor
///////////////////////////////////////////////////////////////////////////////

// FieldSourceParser is a function that creates the list of FieldSource's
// that a particular field can get from depending on its tags.
type FieldSourceParser func(field reflect.StructField) []FieldSource

// ValueGetter is a function that retrieves a value from a specific source
// Returns: (value, found, error)
type ValueGetter func(sourceData any, source FieldSource) (any, bool, error)

// BaseChainExecutor provides functions to create parse execution chain,
// cache them, and run them later.
//
// It takes two functions that describe how the chain is created and
// how it runs:
//
//	# fieldParser (FieldSourceParser)
//
// Used by the executor to traverse
// the fields of a struct, parse the tags for that field, and create
// the []FieldSource for that field.
//
//	# valueGetter (ValueGetter)
//
// Used by the executor to retrieve values from the available FieldSource's
// for the current step in the execution chain.
type BaseChainExecutor struct {
	// Cache for execution chains
	chains map[reflect.Type]*ParseExecutionChain
	// Mutex for thread-safe access to chains
	chainMutex sync.RWMutex

	// fieldParser is a function that parses the sources from a struct field
	// to determine where to extract data from. It is used for building the
	// execution chain.
	fieldParser FieldSourceParser

	// valueGetter extracts values from the source data
	// using the provided FieldSource. It is used by the execution chain
	// to populate the current field.
	valueGetter ValueGetter
}

func NewBaseChainExecutor(
	fieldParser FieldSourceParser,
	valueGetter ValueGetter,
) BaseChainExecutor {
	return BaseChainExecutor{
		chains:      make(map[reflect.Type]*ParseExecutionChain),
		chainMutex:  sync.RWMutex{},
		fieldParser: fieldParser,
		valueGetter: valueGetter,
	}
}

func (p *BaseChainExecutor) GetParseChain(t reflect.Type) (*ParseExecutionChain, error) {
	p.chainMutex.RLock()
	chain, exists := p.chains[t]
	p.chainMutex.RUnlock()

	if exists {
		return chain, nil
	}

	// If not cached, build the chain
	chain, err := p.BuildParseChain(t)
	if err != nil {
		return nil, err
	}

	// Cache the built chain
	p.chainMutex.Lock()
	p.chains[t] = chain
	p.chainMutex.Unlock()

	return chain, nil
}

func (p *BaseChainExecutor) BuildParseChain(t reflect.Type) (*ParseExecutionChain, error) {
	var head, current *ParseStep

	// Parse fields to build the execution chain
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		sources := p.fieldParser(field)
		if len(sources) == 0 {
			continue // Skip fields with no sources
		}

		step := &ParseStep{
			FieldIndex: i,
			FieldName:  field.Name,
			Sources:    sources,
		}

		if head == nil {
			head = step
			current = step
		} else {
			current.Next = step
			current = step
		}
	}

	chain := &ParseExecutionChain{
		StructType:   t,
		Head:         head,
		SourceGetter: p.valueGetter,
	}

	// Cache the chain
	p.chainMutex.Lock()
	p.chains[t] = chain
	p.chainMutex.Unlock()

	return chain, nil
}

///////////////////////////////////////////////////////////////////////////////
// BaseExecutionChain
///////////////////////////////////////////////////////////////////////////////

// ParseExecutionChain represents a linked list of parse steps for a struct type
//
// Uses a function-based approach for source value retrieval, eliminating
// the need for each parser to reimplement the same linked list traversal logic.
// The SourceGetter function provides dynamic dispatch to the appropriate
// value retrieval method for each parser type.
type ParseExecutionChain struct {
	StructType   reflect.Type
	Head         *ParseStep
	SourceGetter ValueGetter // Function to get values from sources
}

// Execute runs the entire parse chain using the provided source getter
func (bec *ParseExecutionChain) Execute(sourceData any, dest Validatable) error {
	current := bec.Head
	for current != nil {
		if err := bec.executeStep(sourceData, dest, current); err != nil {
			return fmt.Errorf("failed to parse field %s: %w", current.FieldName, err)
		}
		current = current.Next
	}
	return nil
}

type executeStepError struct {
	errors []error
}

func (e *executeStepError) Error() string {
	if len(e.errors) == 0 {
		return "no errors"
	}
	return fmt.Sprintf("multiple errors: %v", e.errors)
}

// executeStep executes a single parse step
func (bec *ParseExecutionChain) executeStep(sourceData any, dest Validatable, step *ParseStep) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() == reflect.Ptr {
		destValue = destValue.Elem()
	}

	field := destValue.Field(step.FieldIndex)
	if !field.CanSet() {
		return nil // Skip non-settable fields
	}

	// Try each source in order
	allOmitEmpty := true
	var errors = &executeStepError{errors: []error{}}

	for _, source := range step.Sources {
		allOmitEmpty = allOmitEmpty && source.OmitEmpty

		value, found, err := bec.SourceGetter(sourceData, source)
		if err != nil {
			errors.errors = append(errors.errors, fmt.Errorf("error getting value from source %s: %w", source.Source, err))
			if source.Required {
				return errors
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

	return errors
}
