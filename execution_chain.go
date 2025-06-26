package pave

import (
	"fmt"
	"reflect"
	"sync"
)

///////////////////////////////////////////////////////////////////////////////
// Core Execution Chain Types
///////////////////////////////////////////////////////////////////////////////

// type ExecutionChain[S any] interface {
// 	Execute(source S, dest any) error
// }

type FieldBindingModifiers struct {
	Required  bool            // If true, this is the final source to try
	OmitEmpty bool            // If true, skip this source if not found
	OmitNil   bool            // If true, skip this source if the value is nil
	OmitError bool            // If true, skip this source if an error occurs
	Custom    map[string]bool // Custom modifiers for parser-specific behavior
}

// FieldBinding represents a complete view of a single possible value
// binding for a field. Multipl FieldBinding's are usually defined per field.
type FieldBinding struct {
	Name       string                // The name of the interaction method with the source type
	Identifier string                // The identifier of this specific field on the interaction method
	Modifiers  FieldBindingModifiers // Additional modifiers for the source
}

type BindingOpts struct {
	AllowedBindingNames     []string
	AllowedBindingModifiers []string
}

///////////////////////////////////////////////////////////////////////////////
// ChainExecutor
///////////////////////////////////////////////////////////////////////////////

// BindingHandler is a function that retrieves a value from source that is
// identified by the provided FieldBinding. It is used by
// The Parser -> The Parse Chain Builder -> The Parse Chain
// to retrieve values from the source type.
//
// Returns: (value, found, error)
type BindingHandler[S any] func(source S, binding FieldBinding) (any, bool, error)

// ParseChainBuilder is responsible for building and caching parse chains for a single
// source type (e.g. http.Request, map[string]any, etc.).
//
// It is en
type ParseChainBuilder[S any] struct {
	// Cache for execution Cache
	Cache map[reflect.Type]*ParseChain[S]
	// Mutex for thread-safe access to chains
	CacheMutex sync.RWMutex
	Opts       ParseChainBuilderOpts
	Handler    BindingHandler[S]
}

type ParseChainBuilderOpts struct {
	BindingOpts
}

func NewParseChainBuilder[S any](
	handler BindingHandler[S],
	opts ParseChainBuilderOpts,
) ParseChainBuilder[S] {
	return ParseChainBuilder[S]{
		Cache:      make(map[reflect.Type]*ParseChain[S]),
		CacheMutex: sync.RWMutex{},
		Opts:       opts,
		Handler:    handler,
	}
}

func (builder *ParseChainBuilder[S]) GetParseChain(t reflect.Type) (*ParseChain[S], error) {
	builder.CacheMutex.RLock()
	chain, exists := builder.Cache[t]
	builder.CacheMutex.RUnlock()

	if exists {
		return chain, nil
	}

	// If not cached, build the chain
	chain, err := builder.BuildParseChain(t)
	if err != nil {
		return nil, err
	}

	// Cache the built chain
	builder.CacheMutex.Lock()
	builder.Cache[t] = chain
	builder.CacheMutex.Unlock()

	return chain, nil
}

func (builder *ParseChainBuilder[S]) BuildParseChain(t reflect.Type) (*ParseChain[S], error) {
	var head, current *ParseStep[S]

	// Parse fields to build the execution chain
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Check if field is a struct (excluding special types like time.Time, uuid.UUID)
		isStruct := field.Type.Kind() == reflect.Struct &&
			!isSpecialStructType(field.Type)

		var subChain *ParseChain[S]
		var bindings []FieldBinding
		var defaultValue string
		var err error

		if isStruct {
			// For struct fields, build a sub-chain recursively
			subChain, err = builder.BuildParseChain(field.Type)
			if err != nil {
				return nil, fmt.Errorf("failed to build sub-chain for field %s: %w", field.Name, err)
			}
			// Struct fields don't need bindings since they use sub-chains
			bindings = []FieldBinding{}
		} else {
			// For non-struct fields, parse sources as before
			bindings, defaultValue, err = GetBindings(field, ParseTagOpts{builder.Opts.BindingOpts})
			if err != nil {
				return nil, fmt.Errorf("failed to parse tag for field %s: %w", field.Name, err)
			}
			if len(bindings) == 0 {
				continue // Skip fields with no bindings
			}
		}

		step := &ParseStep[S]{
			FieldIndex:   i,
			FieldName:    field.Name,
			Bindings:     bindings,
			DefaultValue: defaultValue,
			IsStruct:     isStruct,
			SubChain:     subChain,
		}

		if head == nil {
			head = step
			current = step
		} else {
			current.Next = step
			current = step
		}
	}

	chain := &ParseChain[S]{
		StructType: t,
		Head:       head,
		Handler:    builder.Handler,
	}

	// Cache the chain
	builder.CacheMutex.Lock()
	builder.Cache[t] = chain
	builder.CacheMutex.Unlock()

	return chain, nil
}

///////////////////////////////////////////////////////////////////////////////
// Parse Chain
///////////////////////////////////////////////////////////////////////////////

// ParseChain (Parse-Execution Chain) represents a linked list of parse steps for a struct type
//
// Uses a function-based approach for source value retrieval, eliminating
// the need for each parser to reimplement the same linked list traversal logic.
// The SourceGetter function provides dynamic dispatch to the appropriate
// value retrieval method for each parser type.
//
// # It takes one generic type S
//
// S is the Go Type that data will be sourced from (e.g http.Request)
type ParseChain[S any] struct {
	StructType reflect.Type
	Head       *ParseStep[S]
	Handler    BindingHandler[S] // Function to get values from sources
}

// ParseStep represents a single step in the execution chain
type ParseStep[S any] struct {
	// Next is the next step in the current chain.
	Next *ParseStep[S]
	// if this field is a struct that needs recursive parsing
	IsStruct bool
	// Sub-chain for recursive struct parsing
	SubChain *ParseChain[S]
	// Index of the field in the struct
	FieldIndex int
	// Name of the field for error reporting
	FieldName string
	// Default value for the field if bindings fail and not required to succeed
	DefaultValue string
	// Ordered list of bindings to try
	Bindings []FieldBinding
}

// Execute runs the entire parse chain using the provided source getter
func (chain *ParseChain[S]) Execute(source S, dest any) error {
	current := chain.Head
	for current != nil {
		if err := chain.executeStep(source, dest, current); err != nil {
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
func (chain *ParseChain[S]) executeStep(sourceData S, dest any, step *ParseStep[S]) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() == reflect.Ptr {
		destValue = destValue.Elem()
	}

	field := destValue.Field(step.FieldIndex)
	if !field.CanSet() {
		return nil // Skip non-settable fields
	}

	// Handle struct fields with recursive parsing
	if step.IsStruct {
		return chain.executeRecursiveStep(sourceData, field, step)
	}

	// Handle regular fields with source parsing
	return chain.executeRegularStep(sourceData, field, step)
}

// executeRecursiveStep handles recursive parsing of struct fields
func (chain *ParseChain[S]) executeRecursiveStep(sourceData S, field reflect.Value, step *ParseStep[S]) error {
	if step.SubChain == nil {
		return fmt.Errorf("no sub-chain available for struct field %s", step.FieldName)
	}

	// Create a new instance of the struct type if the field is nil or zero
	if field.Kind() == reflect.Ptr {
		if field.IsNil() {
			// Create new instance for pointer field
			newValue := reflect.New(field.Type().Elem())
			field.Set(newValue)
		}
		// For pointer fields, we need to ensure the pointed-to value implements Validatable
		if validatable, ok := field.Interface().(Validatable); ok {
			return step.SubChain.Execute(sourceData, validatable)
		} else {
			return fmt.Errorf("struct field %s does not implement Validatable interface", step.FieldName)
		}
	} else {
		// For non-pointer struct fields, we need to get the address to implement Validatable
		if field.CanAddr() {
			fieldAddr := field.Addr()
			if validatable, ok := fieldAddr.Interface().(Validatable); ok {
				return step.SubChain.Execute(sourceData, validatable)
			} else {
				return fmt.Errorf("struct field %s does not implement Validatable interface", step.FieldName)
			}
		} else {
			return fmt.Errorf("cannot get address of struct field %s for recursive parsing", step.FieldName)
		}
	}
}

// executeRegularStep handles parsing of regular (non-struct) fields
func (chain *ParseChain[S]) executeRegularStep(sourceData S, field reflect.Value, step *ParseStep[S]) error {
	// Try each source in order
	allOmitEmpty := true
	allOmitError := true
	allOmitNil := true
	var errors = &executeStepError{errors: []error{}}

	for _, binding := range step.Bindings {
		allOmitEmpty = allOmitEmpty && binding.Modifiers.OmitEmpty
		allOmitError = allOmitError && binding.Modifiers.OmitError
		allOmitNil = allOmitNil && binding.Modifiers.OmitNil

		value, found, err := chain.Handler(sourceData, binding)
		if err != nil {

			// Handle Omit Error Modifier
			if binding.Modifiers.OmitError {
				continue
			}

			errors.errors = append(errors.errors, fmt.Errorf("error getting value from source %s: %w", binding.Name, err))
			if binding.Modifiers.Required {
				return errors
			}
			continue
		}

		if found {
			if value != nil {
				return setFieldValue(field, fmt.Sprintf("%v", value))
			}
			if binding.Modifiers.OmitNil {
				continue // Skip nil values if OmitNil is set
			}
		}

		if binding.Modifiers.Required {
			return fmt.Errorf("required field %s not found in source %s", binding.Identifier, binding.Name)
		}
	}

	// If all sources have failed/have no data, and default value given, thats ok
	if allOmitEmpty || allOmitError || allOmitNil {
		if step.DefaultValue != "" {
			return setFieldValue(field, step.DefaultValue)
		} else {
			errors.errors = append(errors.errors, fmt.Errorf("all sources failed for field %v and no default value provided", field))
		}
	}

	return errors
}
