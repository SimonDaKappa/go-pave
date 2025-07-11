package pave

// Binding represents a complete view of a single possible value
// binding for a field. Multiple Binding's are usually defined per field.
type Binding struct {
	Name       string           // The name of the binding method with the source type
	Identifier string           // The identifier of this specific field on the binding method
	Modifiers  BindingModifiers // Additional modifiers for the binding.
}

// BindingModifiers represents all modifiers for a binding.
//
// The built-in modifiers control failure and fallback
// behavior for a single binding. Custom modifiers can be used
// however the BindingManager wishes to handle them.
type BindingModifiers struct {
	Required  bool            // If true, this is the final source to try. Error on not found.
	OmitEmpty bool            // If true, skip this source if not found
	OmitNil   bool            // If true, skip this source if the value is nil
	OmitError bool            // If true, skip this source if an error occurs
	Custom    map[string]bool // Custom modifiers for parser-specific behavior
}

type BindingOpts struct {
	AllowedBindingNames    []string
	CustomBindingModifiers []string
}

// BindingResult represents the result of a binding operation.
//
// It contains the value extracted from the source, a boolean indicating
// whether the binding was successful, and an error if any occurred during
// the binding operation.
type BindingResult struct {
	Value any
	Found bool
	Error error
}

// BindingResultNotFound creates a BindingResult indicating that
// the binding was not found in the source.
func BindingResultNotFound() BindingResult {
	return BindingResult{
		Value: nil,
		Found: false,
		Error: nil,
	}
}

// BindingResultNil creates a BindingResult indicating that the binding
// was found but the value is nil.
func BindingResultError(err error) BindingResult {
	return BindingResult{
		Value: nil,
		Found: false,
		Error: err,
	}
}

// BindingResultValue creates a BindingResult indicating that the binding
// was successful and the value was found in the source.
func BindingResultValue(value any) BindingResult {
	return BindingResult{
		Value: value,
		Found: true,
		Error: nil,
	}
}

// BindingHandlerFunc is a function type that defines how to handle a binding
// operation for a specific source type. It takes a pointer to the source type
// and a Binding, and returns the value extracted from the source, a boolean
// indicating if the binding was successful, and an error if any occurred during
// the binding operation.
type BindingHandlerFunc[S any] func(
	source *S,
	binding Binding,
) BindingResult

// BindingHandlerCachedFunc is a function type that defines how to handle a binding
// operation for a specific source type with caching. It takes a pointer to the
// source type, a CacheEntry for the cached value, and a Binding, and returns
// the value extracted from the source, a boolean indicating if the binding was
// successful, and an error if any occurred during the binding operation.
type BindingHandlerCachedFunc[S any, C any] func(
	source *S,
	entry *CacheEntry[C],
	binding Binding,
) BindingResult

// BindingManager is responsible for managing the binding process for a specific
// source type. It provides methods to create a new cache entry instance, as well
// as to handle binding operations for both cached and non-cached scenarios.
//
// All implementations of this interface should be thread-safe, as well as
// stateless.
type BindingManager[Source any, Cached any] interface {
	NewCached() Cached
	BindingHandler(source *Source, binding Binding) BindingResult
	BindingHandlerCached(source *Source, entry *CacheEntry[Cached], binding Binding) BindingResult
}
