package pave

import (
	"errors"
	"fmt"
	"reflect"
)

///////////////////////////////////////////////////////////////////////////////
// Misc.
///////////////////////////////////////////////////////////////////////////////

var (
	ErrParserMethodNotImplemented = errors.New("method not implemented, please override in concrete parser implementation")
)

///////////////////////////////////////////////////////////////////////////////
// SourceParser Interface
///////////////////////////////////////////////////////////////////////////////

type SourceParser interface {
	// Parse extracts the information from the implementation and populates
	// v using the execution chain system.
	Parse(source any, dest any) error
	// GetSourceType returns the reflect.Type of the source this parser works with
	GetSourceType() reflect.Type
	// GetParserName returns a unique identifier for this parser within its source type
	GetParserName() string
}

///////////////////////////////////////////////////////////////////////////////
// OneshotSourceParser
///////////////////////////////////////////////////////////////////////////////

// SingleBindingParser defines the interface for extracting information
// from the implementation of this interface and filling a Validatable
// type with it. A SingleBindingParser is defined for each Type that you wish
// to parse.
//
// Use this interface when you know that the each field in your struct
// will only source from a single place in your source type.
//
// # Oneshot Source Parsers DO NOT build an execution chain.
//
// It is assumed that the Parse method will handle all parsing in a single shot.
// This means that tags for must necessarily be simple and only specify a single
// source.
//
// For example, json unmarshalling a struct via `json:"fieldname"` will always
// source from the JSON body, so you can use a SingleBindingParser for that.
//
// # The following are implemented by default:
//   - JsonSourceParser: Parses from []byte containing JSON data using the
//     `json` tag.
//   - StringMapSourceParser: Parses from a map[string]string source using the
//     `mapvalue` tag.
type SingleBindingParser[Source any] interface {
	SourceParser
}

///////////////////////////////////////////////////////////////////////////////
// MultipleSourceParser
///////////////////////////////////////////////////////////////////////////////

// MultiBindingParser is an interface that defines the following strategy
// for parsing:
//  1. Multiple Binding Parsers are used on source types that can have multiple
//     ways to bind a single field in a struct based on the source type.
//  2. They build a ParseChain that allows for multiple bindings on a single field
//  3. The ParseChain is built based on the tags defined on the struct fields
//     and the source type.
//  4. The ParseChain is executed field by field, (recursively if needed),
//     attempting bindings in the order they are defined in the tags.
//  5. Each binding may come with its own set of modifiers, allowing tolerant
//     error/failure handling if one binding fails, and falling back to the
//     next binding in the chain.
//
// Use this interface when any field in your struct could possibly source
// from multiple places in your source type.
//
// For example, HTTP Requests have useful extractable information from cookies,
// headers, query parameters, and the body. For instance, you might usually get
// a resource by the ID passed in the URL, but sometimes you might want to
// allow the ID to be passed in a header or cookie instead.
//
// This allows a single struct to be reused in multiple contexts.
//
// # The following are implemented by default:
//   - HTTPRequestParser: Parses from an *http.Request using the
//     `json`, `cookie`, `header`, and `query` tags.
type MultiBindingParser[Source any] interface {
	SourceParser

	GetParseChain(destType reflect.Type) (*ParseChain[Source], error)
	BuildParseChain(destType reflect.Type) (*ParseChain[Source], error)
}

// MultiBindingParserTemplate is a mostly implemented template for a
// MultiBindingParser implementation.
//
// This struct provides common functionality for building and
// executing parse chains, parsing field tags into
// a ParseChain that can be executed, and implementing the
// majority of the SourceParser interface.
//
// It is used to create a MultiBindingParser that can handle fields
// with multiple sources, such as HTTP requests, where fields can be sourced
// from cookies, headers, query parameters, and the body.
//
// Use this struct by embedding it in your own MultiBindingParser
// implementation, and then implement the required methods
// to satisfy the SourceParser and MultiBindingParser interfaces.
//  1. GetParserName() string
//  2. handler BindingHandler[S]
//     - This is the function that will actually retrieve the value
//     of a field from the source using the given binding.
//
// The template optionally supports caching of expensive operations per source
// instance using the provided BindingCache. This is useful for parsers
// that rely on external binding implementation that may be expensive to call
// multiple times for the same source instance.
type MultiBindingParserTemplate[Source any, Cached any] struct {
	ParseChainBuilder[Source]
	bindingCache *BindingCache[Source, Cached]
	useCache     bool
}

type MultiBindingParserTemplateOpts struct {
	BindingOpts
	// EnableCaching enables per-source-instance caching of expensive operations
	EnableCaching bool
}

func NewMultiBindingParserTemplate[Source any, Cached any](
	handler BindingHandler[Source],
	opts MultiBindingParserTemplateOpts,
) *MultiBindingParserTemplate[Source, Cached] {

	if handler == nil {
		return nil
	}

	template := &MultiBindingParserTemplate[Source, Cached]{
		ParseChainBuilder: NewParseChainBuilder(handler,
			ParseChainBuilderOpts{
				BindingOpts: opts.BindingOpts,
			},
		),
		useCache: opts.EnableCaching,
	}

	if opts.EnableCaching {
		template.bindingCache = NewBindingCache[Source, Cached]()
	}

	return template
}

// NewMultiBindingParserTemplateWithoutCache creates a MultiBindingParserTemplate without caching support.
// This is a convenience function for parsers that don't need expensive operation caching.
func NewMultiBindingParserTemplateWithoutCache[Source any](
	handler BindingHandler[Source],
	opts MultiBindingParserTemplateOpts,
) *MultiBindingParserTemplate[Source, struct{}] {
	opts.EnableCaching = false
	return NewMultiBindingParserTemplate[Source, struct{}](handler, opts)
}

// GetSourceType returns the reflect.Type of the source this parser works with.
func (p *MultiBindingParserTemplate[Source, Cached]) GetSourceType() reflect.Type {
	return reflect.TypeOf(*new(Source))
}

// Parse executes the parse chain for the given source and populates the destination struct.
// It uses Type Erasure to allow any type of source to be passed in,
// as long as it matches the expected type S defined in the MultiBindingParserTemplate.
//
// This method is the entry point for parsing. It is not recommended to override this method
// in implementations, as parse chains are meant to be agnostic to the source type,
// except for the BindingHandler[Source] that is provided during construction.
func (p *MultiBindingParserTemplate[Source, Cached]) Parse(source any, dest any) error {
	return TypeErasureParseWrapper(p.parse)(source, dest)
}

// parse is the internal method that performs the actual parsing.
// It is separated from the Parse method to allow for type erasure
// so that SourceParser interface is satisfied.
func (p *MultiBindingParserTemplate[Source, Cached]) parse(source Source, dest any) error {
	destType := reflect.TypeOf(dest)
	chain, err := p.ParseChainBuilder.GetParseChain(destType)
	if err != nil {
		return err
	}

	// Execute the parse chain and clean up cache afterwards if caching is enabled
	if p.useCache && p.bindingCache != nil {
		defer p.bindingCache.Delete(&source)
	}

	return chain.Execute(source, dest)
}

// GetBindingCache returns the binding cache for use by concrete parser implementations.
// Returns nil if caching is not enabled.
func (p *MultiBindingParserTemplate[Source, Cached]) GetBindingCache() *BindingCache[Source, Cached] {
	return p.bindingCache
}

func TypeErasureParseWrapper[Source any, Dest any](f func(source Source, dest Dest) error) func(source any, dest any) error {
	return func(source any, dest any) error {
		typedSource, ok := source.(Source)
		if !ok {
			return fmt.Errorf("expected source type %T, got %T", *new(Source), source)
		}
		typedDest, ok := dest.(Dest)
		if !ok {
			return fmt.Errorf("expected dest type %T, got %T", *new(Dest), dest)
		}

		if (reflect.TypeOf(typedDest).Kind() != reflect.Ptr) ||
			(reflect.TypeOf(typedDest).Elem().Kind() != reflect.Struct) {
			return fmt.Errorf("destination must be a pointer to a struct, got %T", dest)
		}

		return f(typedSource, typedDest)
	}
}
