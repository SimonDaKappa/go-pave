package pave

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

var (
	// ErrNoStepBindings is returned when a parse step has no bindings
	// to execute. This can happen if the field is a struct with no bindings,
	// it will be skipped in the parse chain.
	ErrNoStepBindings             = fmt.Errorf("no bindings found for field")
	ErrFailedToParseTag           = fmt.Errorf("failed to parse tag for field")
	ErrAllBindingsFailedNoDefault = fmt.Errorf("All bindings failed with no default value for field")
	ErrFailedToBuildSubChain      = fmt.Errorf("failed to build sub-chain for field")
	ErrNilParseChain              = fmt.Errorf("parse chain is empty for type")
)

// ParseChain represents a linked list of parse steps for a struct type
//
// Uses a function-based approach for binding value retrieval, eliminating
// the need for each parser to reimplement the same linked list traversal logic.
//
// The BindingHandlerFunc provides dynamic dispatch to the appropriate
// value retrieval method for each parser type.
//
// # It takes one generic type S
//
// S is the Go Type that data will be sourced from (e.g http.Request)
type ParseChain[S any] struct {
	StructType reflect.Type          // StructType is the type of the struct being parsed
	Head       *ParseStep[S]         // Head is the first step in the chain
	Handler    BindingHandlerFunc[S] // Function to get values from sources
}

// ParseStep represents a single step in the execution chain
type ParseStep[S any] struct {
	Next          *ParseStep[S]  // Next is the next step in the current chain.
	SubChain      *ParseChain[S] // Sub-chain for recursive struct parsing. Nil if not a struct field.
	Bindings      []Binding      // Ordered list of bindings to try
	FieldName     string         // Name of the field for error reporting
	DefaultValue  string         // Default value for the field if bindings fail and not required to succeed
	IsStruct      bool           // if this field is a struct that needs recursive parsing
	ShouldRecurse bool           // Indicates whether the struct-type field gets 1-step populated by binding or not
	FieldIndex    int            // Index of the field in the struct
}

// Execute runs the entire parse chain using the provided source getter
func (chain *ParseChain[S]) Execute(
	source *S, dest any,
) error {

	if chain.Head == nil {
		return fmt.Errorf(
			"%w: %s",
			ErrNilParseChain,
			chain.StructType.Name(),
		)
	}

	// Traverse the chain and execute each step
	current := chain.Head
	for current != nil {
		// Execute current step
		err := chain.doStep(source, dest, current)
		if err != nil {
			return fmt.Errorf(
				"failed to parse field %s: %w",
				current.FieldName,
				err,
			)
		}
		current = current.Next
	}
	return nil
}

// doStep executes a single parse step
func (chain *ParseChain[S]) doStep(
	sourceData *S, dest any, step *ParseStep[S],
) error {

	// Ensure we have a valid destination value
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() == reflect.Ptr {
		destValue = destValue.Elem()
	}

	field := destValue.Field(step.FieldIndex)

	if !field.CanSet() {
		return nil
	}

	if step.IsStruct && step.ShouldRecurse {
		return chain.doStepRecursive(sourceData, field, step)
	}

	return chain.doStepRegular(sourceData, field, step)
}

var ()

// doStepRegular handles parsing of regular (non-struct) fields
func (chain *ParseChain[S]) doStepRegular(
	sourceData *S, field reflect.Value, step *ParseStep[S],
) error {

	allOmitEmpty := true
	allOmitError := true
	allOmitNil := true
	var errs error

	for _, binding := range step.Bindings {
		modifiers := binding.Modifiers

		allOmitEmpty = allOmitEmpty && modifiers.OmitEmpty
		allOmitError = allOmitError && modifiers.OmitError
		allOmitNil = allOmitNil && modifiers.OmitNil

		result := chain.Handler(sourceData, binding)

		if result.Error != nil {
			if modifiers.OmitError {
				continue
			}

			errs = fmt.Errorf("%w: %w", errs, result.Error)

			if modifiers.Required {
				return errs
			}
			continue
		}

		if result.Found {
			if result.Value != nil {
				return setFieldValue(field, fmt.Sprintf("%v", result.Value))
			}
			if modifiers.OmitNil {
				continue
			}
		}

		if modifiers.Required {
			return fmt.Errorf(
				"required field %s not found in source %s",
				binding.Identifier, binding.Name,
			)
		}
	}

	// If all sources have failed/have no data, and default value given, thats ok
	if allOmitEmpty || allOmitError || allOmitNil {
		if step.DefaultValue != "" {
			return setFieldValue(field, step.DefaultValue)
		} else {
			errs = fmt.Errorf(
				"%w: %w %s",
				errs, ErrAllBindingsFailedNoDefault, field,
			)
		}
	}

	return errs
}

// doStepRecursive handles recursive parsing of struct fields
func (chain *ParseChain[S]) doStepRecursive(
	sourceData *S,
	field reflect.Value,
	step *ParseStep[S],
) error {

	if step.SubChain == nil {
		return fmt.Errorf(
			"no sub-chain available for struct field %s",
			step.FieldName,
		)
	}

	// Handle pointer vs non-pointer struct fields for sub-chains
	if field.Kind() == reflect.Ptr {
		if field.IsNil() {
			// Create new instance for pointer field
			newValue := reflect.New(field.Type().Elem())
			field.Set(newValue)
		}
		// Execute on pointer
		return step.SubChain.Execute(sourceData, field.Interface())
	} else {
		if field.Kind() == reflect.Struct && field.CanAddr() {
			fieldAddr := field.Addr()
			// Execute on struct
			return step.SubChain.Execute(sourceData, fieldAddr.Interface())
		} else {
			return fmt.Errorf(
				"cannot get address of struct field %s for recursive parsing",
				step.FieldName,
			)
		}
	}
}

// PCManager manages parse chains for different destination struct types.
//
// It is responsible creating, caching, retrieving, and executing parse chains
// for a single source type. The source type is defined by the generic type
// parameter Source, which is the type of data that will be parsed into
// destination structs.
//
// The PCManager is thread-safe and can be used concurrently
// across multiple goroutines.
//
// The BindingHandlerFunc is used to retrieve values from the source
// based on the bindings defined in the parse steps. This generally will be a
// function pointer to the BindingHandlerFunc of the BindingManager, or a closure
// of it that injects cached values (ex. BaseMBParser's BindingHandlerAdapter).
type PCManager[S any] struct {
	Chains  map[reflect.Type]*ParseChain[S] // Cache for chains. Keyed by Destination struct type.
	CMutex  sync.RWMutex                    // Mutex for thread-safe access to chains
	Opts    PCManagerOpts                   // Options for the parse chain manager
	Handler BindingHandlerFunc[S]           // Binding Handler for this source type
}

type PCManagerOpts struct {
	tagOpts ParseTagOpts
}

func NewPCManager[S any](
	handler BindingHandlerFunc[S],
	opts PCManagerOpts,
) *PCManager[S] {

	return &PCManager[S]{
		Chains:  make(map[reflect.Type]*ParseChain[S]),
		CMutex:  sync.RWMutex{},
		Opts:    opts,
		Handler: handler,
	}
}

// GetParseChain retrieves a parse chain for the given destination struct type.
//
// If not found, it will create a new parse chain for the type and cache it.
func (cman *PCManager[S]) GetParseChain(
	typ reflect.Type,
) (*ParseChain[S], error) {

	cman.CMutex.RLock()
	chain, exists := cman.Chains[typ]
	cman.CMutex.RUnlock()

	if exists {
		return chain, nil
	}

	// DNE. Build the chain (cached inside)
	chain, err := cman.NewParseChain(typ)
	if err != nil {
		return nil, err
	}

	return chain, nil
}

func (cman *PCManager[S]) NewParseChain(
	typ reflect.Type,
) (*ParseChain[S], error) {

	var head, current *ParseStep[S]

	// Parse fields to build the execution chain
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		step, err := cman.NewParseStep(field, i)
		if err != nil {
			// If no bindings, skip this field
			if errors.Is(err, ErrNoStepBindings) {
				continue
			}
			return nil, err
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
		StructType: typ,
		Head:       head,
		Handler:    cman.Handler,
	}

	// Cache the chain
	cman.CMutex.Lock()
	cman.Chains[typ] = chain
	cman.CMutex.Unlock()

	return chain, nil
}

var ()

func (cman *PCManager[S]) NewParseStep(
	field reflect.StructField, index int,
) (*ParseStep[S], error) {

	var (
		subChain     *ParseChain[S]
		bindings     []Binding
		defaultValue string
		err          error
		isStruct     bool = field.Type.Kind() == reflect.Struct && !isSpecialStructType(field.Type)
		opts              = cman.Opts.tagOpts
	)

	parseTag, err := DecodeParseTagV2(field, opts)
	if err != nil {
		return nil, fmt.Errorf("%w %s: %w", ErrFailedToParseTag, field.Name, err)
	}

	// Handle recursive parsing
	if parseTag.recursiveTag.Enabled {
		if isStruct {
			subChain, err = cman.NewParseChain(field.Type)
			if err != nil {
				return nil, fmt.Errorf("%w %s: %w", ErrFailedToBuildSubChain, field.Name, err)
			}
			// Struct fields don't need bindings since they use sub-chains
			bindings = []Binding{}
		}
	} else {
		// Handle nonrecursive parsing
		bindings, err = makeBindings(parseTag, opts)
		if err != nil {
			return nil, err
		}

		if len(bindings) == 0 {
			return nil, ErrNoStepBindings
		}

		defaultValue = parseTag.defaultTag.Value
	}

	return &ParseStep[S]{
		FieldIndex:    index,
		FieldName:     field.Name,
		Bindings:      bindings,
		DefaultValue:  defaultValue,
		IsStruct:      isStruct,
		SubChain:      subChain,
		ShouldRecurse: parseTag.recursiveTag.Enabled,
	}, nil
}
