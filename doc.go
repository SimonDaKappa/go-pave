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
//   - OneShotSourceParser: Parses from a single source type, such as a byte slice
//     or a string. It is used when the source is expected to be a single value.
//     These parsers do not support multiple sources for a single field.
//   - MultipleSourceParser: Parses from sources with multiple interaction methods,
//     such as a HTTP request. These parsers build a ParseExecutionChain (see
//     [ParseChainBuilder](https://pkg.go.dev/pave#ChainExecutor) and
//     [ParseChain](https://pkg.go.dev/pave#ParseExecutionChain)) that allows
//     you to extract data from various sources like cookies, headers, query parameters,
//     and the body of the request. Execution chains allow for flexible fallbacks
//     and prioritization of sources when parsing, are cached, and can be reused
//     for any Validatable type that has the correct tags.
//
// All parsers must implement the [SourceParser](https://pkg.go.dev/pave#SourceParser) interface,
// which defines the methods required for parsing data from a source into a Validatable type.
//
// All parsers must also support the following source tag modifiers:
//   - `omitempty`: If the field is empty, it will not be parsed. If a fallback source
//     is specified, an attempt to use it instead will be made.
//   - `required`: The field must be present in the source, otherwise an
//     error will be returned. The first instance of `required` as a tag modifier
//     will cut the generation of the parse execution chain. That is, all `omitempty`
//     tags after the first `required` will be ignored, and the field will be required
//     to be present in the source. `required` is IMPLIED by default. At most one `required`
//     modifier can be present per field.
package pave

/**
PLANNING:
- Add support for default values modifiers in tags, e.g., `default:"value"`.
- Add support for recursive parsing of nested structs.
- Add support for automatic validation generation
    1. Support builtin validation library and integration. to other libraries (for instance go-playground validation)
    2. Must be able to validate for Validatable or for types that can possibly be converted to Validatable by boxing them
    3. Build Tags for enabling integrations/featureflags
- Formalize tag grammar, create generic tag parser for MultiStepSourceParser and OneShotSourceParser
    - Tag grammar shown below.
    - Multi-step tag parsers should function similarly to BaseChainExecutor. i.e., it should parse according
      to the complete grammar, but source parser specific tags should be referenced by a sourceTagFn that
      creates the FieldSource for the execution chain if the tag is recognized
- Tag aliasing.
- Global Tag String White/Blacklist registry
    - Allow per tag parse includes/excludes but also global exludes that can be set.

v0.1.0 Tag Notes:
Tag grammar:
    <field> <type> <tag>
field:
    <string>
type:
    <Type>
tag:
    '<parse_tag> <validate_tag>'

parse_tag:
    parse:"<default_tag> <source_tag_list>"

default_tag:
    default:'<default_value>'
default_value:
    <Go Literal>

source_tag_list:
    [<source_tag>]^*
source_tag:
    <source_name>:'<source_identifier>,<source_modifier_list>' // source tags are parser specific but must follow this grammar
source_name, source_identifier:
    <string>
source_modifier_list:
    [source_modifier]^* // Delimeted with "," end-delim optional
source_modifier:
    omitempty | omiterr | omitnil | ... // (any other modifiers past this point are parser specific)

validate_tag
    validate:"<...>" | nil

*/
