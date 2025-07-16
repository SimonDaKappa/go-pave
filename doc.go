// Package pave (Parse And Validate Everything) provides a flexible and extensible
// framework for parsing and validating data structures in Go.
//
// It allows you to define custom parsers for different data sources and
// all you need to do to parse into a destination struct is to define
// the appropriate tags on the struct fields.
//
// The package provides built-in parsers for common data sources,
// such as:
//   - JSON (from byte slices or strings)
//   - HTTP requests (from cookies, headers, query parameters, and body)
//   - String maps (from map[string]string or map[string]any)
//   - Map values (from map[fmt.Stringer]any)
//
// The parsers support recursive parsing, allowing you to
// define nested structures and have them automatically populated
// from the source data.
//
// To use the package, you may use the exported methods:
// - Validate(): Parse and validate with the built-in validator
// - WithParser(): Curry the built-in validator with a custom parser
// - RegisterParser(): Register a custom parser for a specific source type
// Or you may register your own parsers on an instance of the Validator
// struct.
//
// Each struct will define its own Validate() method that will be called to
// validate the struct's fields after they have been populated by the parsers.
//
// Custom parser come in two flavors:
//   - OneShotSourceParser: Parses from a source type with a single binding,
//     such as a byte slice or a string. It is used when the source is expected
//     to be a single value.
//   - MultipleSourceParser: Parses from sources with multiple interaction methods,
//     such as a HTTP request. These parsers build a ParseExecutionChain (see
//     [parseChainBuilder](https://pkg.go.dev/pave#ChainExecutor) and
//     [ParseChain](https://pkg.go.dev/pave#ParseExecutionChain)) that allows
//     you to extract data from various sources like cookies, headers, query parameters,
//     and the body of the request. Execution chains allow for flexible fallbacks
//     and prioritization of sources when parsing, are cached, and can be reused
//     for any Validatable type that has the correct tags.
//
// All parsers must implement the [Parser](https://pkg.go.dev/pave#SourceParser) interface,
// which defines the methods required for parsing data from a source into a Validatable type.
package pave

/**
PLANNING:
- Add support for default values modifiers in tags, e.g., `default:"value"`. (DONEish, need to add support for automatic default value type conversion)
- Add support for recursive parsing of nested structs. (PARTIAL: Nested structs by default recurse, but need to add support for nonrecursive struct parsing)
- Add support for automatic validation generation
    1. Support builtin validation library and integration. to other libraries (for instance go-playground validation)
    2. Must be able to validate for Validatable or for types that can possibly be converted to Validatable by boxing them
    3. Build Tags for enabling integrations/featureflags
- Formalize tag grammar, create generic tag parser for MultiStepSourceParser and OneShotSourceParser (DONE)
    - Tag grammar shown [tag.go](tag.go)
- Tag aliasing.
- Global Tag String White/Blacklist registry
    - Allow per tag parse includes/excludes but also global excludes that can be set. (WIP)
- Split Package into PAVE-Parser and PAVE-Validator
- Implement Validation
- Add support for custom parsers that can be registered with the Registry. (DONE)
- Add support for custom validators that can be registered with the Registry.
*/
