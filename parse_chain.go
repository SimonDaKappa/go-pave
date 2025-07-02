package pave

import (
	"fmt"
	"reflect"
	"sync"
)

// ParseChain represents a linked list of parse steps for a struct type
//
// Uses a function-based approach for source value retrieval, eliminating
// the need for each parser to reimplement the same linked list traversal logic.
// The SourceGetter function provides dynamic dispatch to the appropriate
// value retrieval method for each parser type.
//
// # It takes one generic type S
//
// S is the Go Type that data will be sourced from (e.g http.Request)
type ParseChain[Source any] struct {
	StructType reflect.Type           // StructType is the type of the struct being parsed
	Head       *ParseStep[Source]     // Head is the first step in the chain
	Handler    BindingHandler[Source] // Function to get values from sources
}

// ParseStep represents a single step in the execution chain
type ParseStep[Source any] struct {
	Next         *ParseStep[Source]  // Next is the next step in the current chain.
	SubChain     *ParseChain[Source] // Sub-chain for recursive struct parsing. Nil if not a struct field.
	Bindings     []Binding           // Ordered list of bindings to try
	FieldName    string              // Name of the field for error reporting
	DefaultValue string              // Default value for the field if bindings fail and not required to succeed
	IsStruct     bool                // if this field is a struct that needs recursive parsing
	FieldIndex   int                 // Index of the field in the struct
}

// Execute runs the entire parse chain using the provided source getter
func (chain *ParseChain[Source]) Execute(source *Source, dest any) error {

	if chain.Head == nil {
		return fmt.Errorf(
			"parse chain is empty for type %s",
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
func (chain *ParseChain[Source]) doStep(
	sourceData *Source,
	dest any,
	step *ParseStep[Source],
) error {

	destValue := reflect.ValueOf(dest)
	if destValue.Kind() == reflect.Ptr {
		destValue = destValue.Elem()
	}

	field := destValue.Field(step.FieldIndex)

	// Skip unsettable fields
	if !field.CanSet() {
		return nil
	}

	// Handle struct fields with recursive parsing
	if step.IsStruct {
		return chain.doStepRecursive(sourceData, field, step)
	}

	// Handle regular fields with source parsing
	return chain.doStepRegular(sourceData, field, step)
}

// doStepRegular handles parsing of regular (non-struct) fields
func (chain *ParseChain[Source]) doStepRegular(
	sourceData *Source,
	field reflect.Value,
	step *ParseStep[Source],
) error {
	// Try each source in order
	allOmitEmpty := true
	allOmitError := true
	allOmitNil := true

	var errs error

	for _, binding := range step.Bindings {
		modifiers := binding.Modifiers
		allOmitEmpty = allOmitEmpty && binding.Modifiers.OmitEmpty
		allOmitError = allOmitError && binding.Modifiers.OmitError
		allOmitNil = allOmitNil && binding.Modifiers.OmitNil

		value, found, err := chain.Handler(sourceData, binding)
		if err != nil {

			if modifiers.OmitError {
				continue
			}

			errs = fmt.Errorf(
				"%w: error getting value from source %s: %w",
				errs,
				binding.Name,
				err,
			)

			if modifiers.Required {
				return errs
			}
			continue
		}

		if found {
			if value != nil {
				return setFieldValue(field, fmt.Sprintf("%v", value))
			}
			if modifiers.OmitNil {
				continue
			}
		}

		if modifiers.Required {
			return fmt.Errorf("required field %s not found in source %s", binding.Identifier, binding.Name)
		}
	}

	// If all sources have failed/have no data, and default value given, thats ok
	if allOmitEmpty || allOmitError || allOmitNil {
		if step.DefaultValue != "" {
			return setFieldValue(field, step.DefaultValue)
		} else {
			errs = fmt.Errorf("%w: all sources failed for field %v and no default value provided", errs, field)
		}
	}

	return errs
}

// doStepRecursive handles recursive parsing of struct fields
func (chain *ParseChain[Source]) doStepRecursive(
	sourceData *Source,
	field reflect.Value,
	step *ParseStep[Source],
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

type ParseChainManager[Source any] struct {
	Chains    map[reflect.Type]*ParseChain[Source] // Cache for chains. Keyed by Destination struct type.
	ChainsMtx sync.RWMutex                         // Mutex for thread-safe access to chains
	Opts      ParseChainManagerOpts
	Handler   BindingHandler[Source] // Binding Handler for this source type
}

type ParseChainManagerOpts struct {
	BindingOpts
}

func NewParseChainManager[Source any](
	handler BindingHandler[Source],
	opts ParseChainManagerOpts,
) *ParseChainManager[Source] {

	return &ParseChainManager[Source]{
		Chains:    make(map[reflect.Type]*ParseChain[Source]),
		ChainsMtx: sync.RWMutex{},
		Opts:      opts,
		Handler:   handler,
	}
}

func (pcMgr *ParseChainManager[Source]) GetParseChain(typ reflect.Type) (*ParseChain[Source], error) {
	pcMgr.ChainsMtx.RLock()
	chain, exists := pcMgr.Chains[typ]
	pcMgr.ChainsMtx.RUnlock()

	if exists {
		return chain, nil
	}

	// DNE. Build the chain (cached inside)
	chain, err := pcMgr.NewParseChain(typ)
	if err != nil {
		return nil, err
	}

	return chain, nil
}

func (pcMgr *ParseChainManager[Source]) NewParseChain(typ reflect.Type) (*ParseChain[Source], error) {
	var head, current *ParseStep[Source]

	// Parse fields to build the execution chain
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Check if field is a struct (excluding special types like time.Time, uuid.UUID)
		isStruct := field.Type.Kind() == reflect.Struct && !isSpecialStructType(field.Type)

		var (
			subChain     *ParseChain[Source]
			bindings     []Binding
			defaultValue string
			err          error
		)

		if isStruct {
			// For struct fields, build a sub-chain recursively
			subChain, err = pcMgr.NewParseChain(field.Type)
			if err != nil {
				return nil, fmt.Errorf("failed to build sub-chain for field %s: %w", field.Name, err)
			}
			// Struct fields don't need bindings since they use sub-chains
			bindings = []Binding{}
		} else {
			// For non-struct fields, parse sources as before
			bindings, defaultValue, err = GetBindings(field, ParseTagOpts{pcMgr.Opts.BindingOpts})
			if err != nil {
				return nil, fmt.Errorf("failed to parse tag for field %s: %w", field.Name, err)
			}
			if len(bindings) == 0 {
				continue // Skip fields with no bindings
			}
		}

		step := &ParseStep[Source]{
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

	chain := &ParseChain[Source]{
		StructType: typ,
		Head:       head,
		Handler:    pcMgr.Handler,
	}

	// Cache the chain
	pcMgr.ChainsMtx.Lock()
	pcMgr.Chains[typ] = chain
	pcMgr.ChainsMtx.Unlock()

	return chain, nil
}
