package pave

// Binding represents a complete view of a single possible value
// binding for a field. Multipl Binding's are usually defined per field.
type Binding struct {
	Name       string           // The name of the interaction method with the source type
	Identifier string           // The identifier of this specific field on the interaction method
	Modifiers  BindingModifiers // Additional modifiers for the source
}

// BidingMOdifiers represents additional modifiers for a binding.
//
// The built-in modifiers control failure and fallback
// behavior for a single binding. Custom modifiers can be used
// however the parser wishes to handle them.
type BindingModifiers struct {
	Required  bool            // If true, this is the final source to try
	OmitEmpty bool            // If true, skip this source if not found
	OmitNil   bool            // If true, skip this source if the value is nil
	OmitError bool            // If true, skip this source if an error occurs
	Custom    map[string]bool // Custom modifiers for parser-specific behavior
}

type BindingOpts struct {
	AllowedBindingNames    []string
	CustomBindingModifiers []string
}

type BindingHandler[S any] func(source *S, binding Binding) (any, bool, error)
type BindingHandlerCached[S any, C any] func(source *S, entry *CacheEntry[C], binding Binding) (any, bool, error)

type BindingManager[Source any, Cached any] interface {
	NewCached() Cached
	BindingHandler(source *Source, binding Binding) (any, bool, error)
	BindingHandlerCached(source *Source, entry *CacheEntry[Cached], binding Binding) (any, bool, error)
}
