package pave

import (
	"fmt"
	"reflect"
)

// BaseMBParser is a mostly implemented template for a MultiBindingParser
// implementation.
//
// This struct provides common functionality for building and executing parse
// chains, parsing field tags into a ParseChain that can be executed, caching
// parse chains, caching expensive binding operations, and handling type erasure
// for the source type.
//
// It is used to create a MultiBindingParser that can handle fields with
// multiple sources, such as HTTP requests, where fields can be sourced from
// cookies, headers, query parameters, and the body.
//
// It takes two type parameters:
//   - S: The type of the source that this parser works with. DO NOT use a
//     pointer. IT WILL PANIC. All methods should take a pointer to S instead.
//   - C: The type of the cached value that will be used by the parser during
//     binding operations. This is typically a struct that contains the cached
//     values for the source type, allowing the parser to avoid expensive
//     operations by reusing previously computed values.
//
// The template optionally supports caching of expensive operations per source
// instance using the provided BindingCache. This is useful for parsers that
// rely on external binding implementation that may be expensive to call
// multiple times for the same source instance.
type BaseMBParser[S any, C any] struct {
	PCMgr     *ParseChainManager[S]
	BMgr      BindingManager[S, C]
	BCache    *BindingCache[S, C]
	useBCache bool
}

type BaseMBParserOpts struct {
	PCMOpts  ParseChainManagerOpts
	UseCache bool
}

func NewBaseMBParser[S any, C any](
	bMgr BindingManager[S, C],
	opts BaseMBParserOpts,
) *BaseMBParser[S, C] {

	if bMgr == nil {
		return nil
	}

	// Panic if Source is a pointer type, as this will break the cache.
	if reflect.TypeOf(*new(S)).Kind() == reflect.Ptr {
		panic(fmt.Sprintf(
			"Generic %T cannot be a pointer (breaks cache)."+
				"Use a non-pointer for type constraint",
			*new(S),
		))
	}

	/* TODO $$$SIMON: Is there a better way to do this?
	   The adapter is useful for abstracting cache implementation
	   away from the BindingManager, but having to pass the pcm a funcptr
	   from a empty template, then add the pcm as a field to the template,
	   is clunky as an understatement.
	*/

	template := &BaseMBParser[S, C]{}

	pcMgr := NewParseChainManager(
		template.BindingHandlerAdapter,
		opts.PCMOpts,
	)

	template.BMgr = bMgr
	template.PCMgr = pcMgr
	template.useBCache = opts.UseCache

	if opts.UseCache {
		template.BCache = NewBindingCache[S, C]()
	} else {
		template.BCache = nil
	}

	return template
}

// SourceType returns the reflect.Type of the source this parser works with.
func (base *BaseMBParser[S, C]) SourceType() reflect.Type {
	return reflect.TypeOf(*new(S))
}

// Parse executes the parse chain for the given source and populates the
// destination struct. It uses Type Erasure to allow any type of source to be
// passed in, as long as it matches the generic type parameter Source.
//
// Both arguments must be pointers:
//   - source: A pointer to the source type that this parser works with.
//   - dest: A pointer to the destination struct that will be populated with the
//     parsed data
func (base *BaseMBParser[S, C]) Parse(source any, dest any) error {
	typedSource, ok := source.(*S)
	if !ok {
		return fmt.Errorf("expected source type %T, got %T", *new(S), source)
	}

	if (reflect.TypeOf(dest).Kind() != reflect.Ptr) ||
		(reflect.TypeOf(dest).Elem().Kind() != reflect.Struct) {
		return fmt.Errorf("destination must be a pointer to a struct, got %T", dest)
	}

	return base.parse(typedSource, dest)
}

// parse is the internal method that performs the actual parsing.
// It is separated from the Parse method to allow for type erasure
// so that Parser interface is satisfied.
func (base *BaseMBParser[S, C]) parse(source *S, dest any) error {
	typ := reflect.TypeOf(dest).Elem()

	// Get the parse chain for the destination type
	chain, err := base.PCMgr.GetParseChain(typ)
	if err != nil {
		return err
	}

	// Execute chain
	return chain.Execute(source, dest)
}

func (base *BaseMBParser[S, C]) BindingHandlerAdapter(
	source *S,
	binding Binding,
) (any, bool, error) {

	// Deref for interface but still keep pointer semantics
	if base.useBCache {
		if base.BCache == nil {
			base.BCache = NewBindingCache[S, C]()
		}

		entry := base.BCache.GetOrCreate(source, base.BMgr.NewCached)
		return base.BMgr.BindingHandlerCached(source, entry, binding)
	} else {
		return base.BMgr.BindingHandler(source, binding)
	}
}
