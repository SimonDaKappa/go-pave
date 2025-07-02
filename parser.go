package pave

import (
	"reflect"
)

// Parser is the base interface for all parsers that extract information
// from a source type and populate a destination struct. It defines the
// Parse method that extracts information from the source and populates the
// destination struct.
//
// Note that most parsers will use the provided Base parsers, and thus
// will require a small amount of boilerplate type erasure code to
// implement the Parse method. However, there are type erasure
// helpers available to simplify this process. See:
//   - [ParseTypeErasedPointer](./helpers.go#ParseTypeErasedPointer)
//   - [ParseTypeErasedSlice](./helpers.go#ParseTypeErasedSlice)
//   - [ParseTypeErasedMap](./helpers.go#ParseTypeErasedMap)
//
// The implementations of this interface will typically come in one of
// two flavors:
//   - A SingleBindingParser that only allows parsing a single binding
//     per field, which is useful for simple cases where you know
//     the source type and the bindings are straightforward.
//   - A MultiBindingParser that allows for multiple bindings per field,
//     which is useful for more complex cases where you want to try
//     multiple sources or methods of extracting the data. This allows
//     for more:
//     -- Flexibility in how fields are populated
//     -- Reusability of the same struct across different contexts.
//     -- Caching of parsed data to avoid unnecessary recomputation
//     and allocations.
//     -- Tolerant, configurable error handling during parsing
type Parser interface {
	// Parse extracts the information from the implementation and populates
	// dest's fields with the values from the source.
	//
	// Both arguments must be pointers:
	//   - source: A pointer to the source type that this parser works with.
	//   - dest: A pointer to the destination struct that will be populated with the parsed data.
	Parse(source any, dest any) error
	// SourceType returns the reflect.Type of the source this parser works with
	SourceType() reflect.Type
	// Name returns a unique identifier for this parser within its source type
	Name() string
}
