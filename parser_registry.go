package pave

import (
	"errors"
	"fmt"
	"reflect"
)

var (
	ErrNoFieldSourcesInTag            = errors.New("no fields sources defined in tag but attempted to validate field")
	ErrParserAlreadyRegistered        = errors.New("a parser with this name for this source-type is already registered")
	ErrNoParser                       = errors.New("no built-in or registered parser found for this type")
	ErrNoParserBuiltin                = errors.New("no built-in parser found for this type")
	ErrNoParserRegistered             = errors.New("no registered parser found for this type")
	ErrMultipleParsersAvailable       = errors.New("multiple parsers available for this source type, use WithParser() to specify which one")
	ErrParserNotFound                 = errors.New("specified parser not found for this source type")
	ErrNoParseExecutionChain          = errors.New("no parse execution chain found for this type")
	ErrInvalidParseExecutionChainType = errors.New("improper type passed for this parse execution chain")
)

type Validatable interface {
	// Validate checks the fields of the struct and returns an error
	// if any of the fields are invalid.
	Validate() error
}

// ParserRegistry is the main struct that handles validation
// of Validatable types using registered SourceParsers.
//
// It provides methods to register parsers, validate data,
// and invalidate partially or fully validated structs.
//
// Multiple SourceParsers can be registered for each SourceType.
// If only one parser is registered for a type, it will be used
// automatically. If multiple parsers are registered, you must
// use WithParser() to specify which one to use.
//
// Each SourceParser will build and cache an execution chain
// for each unique Validatable type it is used with.
type ParserRegistry struct {
	m map[reflect.Type]map[string]Parser // source type -> parser name -> parser
}

// ParserRegistryContext provides a curried Registry with a specific parser selection
type ParserRegistryContext struct {
	registry   *ParserRegistry
	parserName string
}

var (
	_defaultSourceParsers []Parser = nil
)

type ParserRegistryOpts struct {
	Parsers         []Parser
	ExcludeDefaults bool
}

func NewParserRegistry(opts ParserRegistryOpts) (*ParserRegistry, error) {
	reg := &ParserRegistry{
		m: make(map[reflect.Type]map[string]Parser),
	}

	if !opts.ExcludeDefaults {
		for _, parser := range _defaultSourceParsers {
			err := reg.Register(parser)
			if err != nil {
				return nil, err
			}
		}
	}

	for _, parser := range opts.Parsers {
		err := reg.Register(parser)
		if err != nil {
			return nil, err
		}
	}

	return reg, nil
}

// Now your registration method can accept the non-generic interface
func (reg *ParserRegistry) Register(parser Parser) error {
	typ := parser.SourceType()
	name := parser.Name()

	if reg.m[typ] == nil {
		reg.m[typ] = make(map[string]Parser)
	}

	reg.m[typ][name] = parser
	return nil
}

// WithParser returns a ValidatorContext that will use the specified parser
// for validation. This is useful when multiple parsers are registered for
// the same source type.
func (reg *ParserRegistry) WithParser(parserName string) *ParserRegistryContext {
	return &ParserRegistryContext{
		registry:   reg,
		parserName: parserName,
	}
}

// Parse populates dest based on the specified parser's logic.
// It expects the passed dest to be a pointer.
func (regCtx *ParserRegistryContext) Parse(source any, dest any, validate bool) error {
	parser, err := regCtx.registry.getParserByName(source, regCtx.parserName)
	if err != nil {
		return err
	}

	err = parser.Parse(source, dest)
	if err != nil {
		if dest, ok := dest.(Validatable); ok {
			regCtx.registry.Invalidate(dest)
		}
		return fmt.Errorf("failed to parse with %s: %w", parser.Name(), err)
	}

	if dest, ok := dest.(Validatable); ok && validate {
		err = dest.Validate()
		if err != nil {
			regCtx.registry.Invalidate(dest)
			return fmt.Errorf("validation failed after parsing with %s: %w", parser.Name(), err)
		}
	}

	return nil
}

// Parse populates dest based on the implementation of source's
// parsing logic.
//
// It only succeeds if there is exactly one parser registered
// for data's type. To use a specific parser, you must
// use the WithParser() method to specify which one to use.
//
// # It expects dest to be a pointer
//
// If validation fails, it will return the validation error
// and zero all of dest's fields.
func (reg *ParserRegistry) Parse(source any, dest any, validate bool) error {

	if dest == nil {
		return fmt.Errorf("dest cannot be nil")
	}
	if reflect.TypeOf(dest).Kind() != reflect.Ptr ||
		reflect.ValueOf(dest).IsNil() ||
		reflect.TypeOf(dest).Elem().Kind() != reflect.Struct {
		return fmt.Errorf("dest must be a non-nil pointer to a struct type")
	}

	parser, err := reg.tryGetDefaultParser(source)
	if err != nil {
		return err
	}

	err = parser.Parse(source, dest)
	if err != nil {
		if dest, ok := dest.(Validatable); ok {
			reg.Invalidate(dest)
		}
		return fmt.Errorf("failed to parse with %s: %w", parser.Name(), err)
	}

	if dest, ok := dest.(Validatable); ok && validate {
		err = dest.Validate()
		if err != nil {
			reg.Invalidate(dest)
			return fmt.Errorf("validation failed after parsing with %s: %w", parser.Name(), err)
		}
	}

	return nil
}

// tryGetDefaultParser retrieves the appropriate SourceParser for the given data type.
//
// If multiple parsers are found for the same source type, it returns an error
// indicating that WithParser() should be used to specify which one.
//
// If no parser is found, it returns ErrNoParser.
func (reg *ParserRegistry) tryGetDefaultParser(source any) (Parser, error) {
	parser, err := reg.getParserByName(source, "")
	if err != nil {
		return nil, err
	}

	return parser, nil
}

// getParserByName retrieves a specific parser by name for the given data type.
//
// No name provided: If there is only one parser registered for the type,
// it returns that parser. If multiple parsers are registered, it returns an error
func (reg *ParserRegistry) getParserByName(source any, parserName string) (Parser, error) {
	t := reflect.TypeOf(source)

	// Check registered parsers
	if parsersForType, exists := reg.m[t]; exists {

		// If no parser name is specified, handle the case of multiple parsers
		// registered for the same type.
		// - 0 parsers: return ErrNoParserRegistered
		// - 1 parser: return it
		// - >1 parsers: return an error
		if parserName == "" {
			switch len(parsersForType) {
			case 0:
				return nil, ErrNoParserRegistered
			case 1:
				for _, parser := range parsersForType {
					return parser, nil
				}
			default:
				return nil, ErrMultipleParsersAvailable
			}
		}

		if parser, found := parsersForType[parserName]; found {
			return parser, nil
		}
	}

	return nil, ErrParserNotFound
}

// Invalidate clears a partially or fully validated dest by
// setting each field to its default value.
//
// # It expects the passed v to be a pointer
//
// An error is returned if the argument is not reflect-able
func (reg *ParserRegistry) Invalidate(dest Validatable) error {
	value := reflect.ValueOf(dest)
	if value.Kind() != reflect.Ptr || value.IsNil() {
		return fmt.Errorf("cannot invalidate a non ptr or nil value")
	}

	elem := value.Elem()
	zeroStructFields(elem)

	return nil
}

///////////////////////////////////////////////////////////////////////////////
// Global Singleton and Package Functions
///////////////////////////////////////////////////////////////////////////////

var _gParserRegistry *ParserRegistry = nil

func init() {
	_defaultSourceParsers = []Parser{
		// NewJsonByteSliceSourceParser(),
		// NewJSONStringSourceParser(),
		NewHTTPRequestParser(),
		// NewStringMapSourceParser(),
		// NewStringAnyMapSourceParser(),
	}

	var err error
	_gParserRegistry, err = NewParserRegistry(ParserRegistryOpts{ExcludeDefaults: false})
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize global ParserRegistry: %v", err))
	}
}

// Package-level functions that delegate to the global ParserRegistry instance

func RegisterParser(parser Parser) error {
	return _gParserRegistry.Register(parser)
}

func Parse(source any, dest any, validate bool) error {
	return _gParserRegistry.Parse(source, dest, validate)
}

func WithParser(parserName string) *ParserRegistryContext {
	return _gParserRegistry.WithParser(parserName)
}

func Invalidate(dest Validatable) error {
	return _gParserRegistry.Invalidate(dest)
}

func GetParser(source any) (Parser, error) {
	return _gParserRegistry.tryGetDefaultParser(source)
}

func GetParserByName(source any, parserName string) (Parser, error) {
	return _gParserRegistry.getParserByName(source, parserName)
}
