package pave

import (
	"errors"
	"fmt"
	"reflect"
)

///////////////////////////////////////////////////////////////////////////////
// Errors
///////////////////////////////////////////////////////////////////////////////

// ValidationError is a an error that occured during validating
// from a SourceParser into a Validatable impl.
type ValidationError struct {
	reason string
}

// Error implements the error interface
func (ve ValidationError) Error() string {
	return fmt.Sprintf("Failed to validate: %s", ve.reason)
}

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

///////////////////////////////////////////////////////////////////////////////
// Validator Impl.
///////////////////////////////////////////////////////////////////////////////

// Validatable is an interface that marks a struct as expecting
// to be populated by a ValidationParser and later have its fields
// validated by calling Validate()
type Validatable interface {
	// Validate checks the fields of the struct and returns an error
	// if any of the fields are invalid.
	//
	// # It expects the implementation to be a pointer
	//
	// # It is expected to be called after the struct has been populated
	//
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
	RegisteredParsers map[reflect.Type]map[string]SourceParser // source type -> parser name -> parser
}

// ParserRegistryContext provides a curried validator with a specific parser selection
type ParserRegistryContext struct {
	validator  *ParserRegistry
	parserName string
}

var (
	_defaultSourceParsers []SourceParser = nil
)

type ValidatorOpts struct {
	Parsers         []SourceParser
	IncludeDefaults bool
}

func NewValidator(opts ValidatorOpts) (*ParserRegistry, error) {
	v := &ParserRegistry{
		RegisteredParsers: make(map[reflect.Type]map[string]SourceParser),
	}

	if opts.IncludeDefaults {
		// Register default parsers if IncludeDefaults is true
		for _, parser := range _defaultSourceParsers {
			err := v.RegisterParser(parser)
			if err != nil {
				return nil, err
			}
		}
	}

	for _, parser := range opts.Parsers {
		err := v.RegisterParser(parser)
		if err != nil {
			return nil, err
		}
	}

	return v, nil
}

// Now your registration method can accept the non-generic interface
func (v *ParserRegistry) RegisterParser(parser SourceParser) error {
	sourceType := parser.GetSourceType()
	parserName := parser.GetParserName()

	if v.RegisteredParsers[sourceType] == nil {
		v.RegisteredParsers[sourceType] = make(map[string]SourceParser)
	}

	v.RegisteredParsers[sourceType][parserName] = parser
	return nil
}

// WithParser returns a ValidatorContext that will use the specified parser
// for validation. This is useful when multiple parsers are registered for
// the same source type.
func (v *ParserRegistry) WithParser(parserName string) *ParserRegistryContext {
	return &ParserRegistryContext{
		validator:  v,
		parserName: parserName,
	}
}

// Validate populates dest based on the specified parser's logic.
// It expects the passed dest to be a pointer.
// If validation fails, it will return the validation error and zero all of dest's fields.
func (vc *ParserRegistryContext) Validate(data any, dest Validatable) error {
	parser, err := vc.validator.getParserByName(data, vc.parserName)
	if err != nil {
		return err
	}

	err = parser.Parse(data, dest)
	if err != nil {
		vc.validator.Invalidate(dest)
		return ValidationError{err.Error()}
	}

	err = dest.Validate()
	if err != nil {
		vc.validator.Invalidate(dest)
		return ValidationError{err.Error()}
	}

	return nil
}

// Validate populates dest based on the implementation of source's
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
func (v *ParserRegistry) Validate(data any, dest any) error {

	if dest == nil {
		return ValidationError{reason: "dest cannot be nil"}
	}
	if reflect.TypeOf(dest).Kind() != reflect.Ptr ||
		reflect.ValueOf(dest).IsNil() ||
		reflect.TypeOf(dest).Elem().Kind() != reflect.Struct {
		return ValidationError{reason: "dest must be a non-nil pointer to a struct type"}
	}

	parser, err := v.tryGetDefaultParser(data)
	if err != nil {
		return err
	}

	err = parser.Parse(data, dest)
	if err != nil {
		if dest, ok := dest.(Validatable); ok {
			v.Invalidate(dest)
		}
		return ValidationError{err.Error()}
	}

	if dest, ok := dest.(Validatable); ok {
		err = dest.Validate()
		if err != nil {
			v.Invalidate(dest)
			return ValidationError{err.Error()}
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
func (v *ParserRegistry) tryGetDefaultParser(data any) (SourceParser, error) {
	typ := reflect.TypeOf(data)

	parser, err := v.getParserByName(typ, "")
	if err != nil {
		return nil, err
	}

	return parser, nil
}

// getParserByName retrieves a specific parser by name for the given data type.
//
// No name provided: If there is only one parser registered for the type,
// it returns that parser. If multiple parsers are registered, it returns an error
func (v *ParserRegistry) getParserByName(data any, parserName string) (SourceParser, error) {
	t := reflect.TypeOf(data)

	// Check registered parsers
	if parsersForType, exists := v.RegisteredParsers[t]; exists {

		// If no parser name is specified, handle the case of multiple parsers
		// registered for the same type.
		if parserName == "" {
			l := len(parsersForType)
			switch l {
			case 0:
				return nil, ErrNoParserRegistered
			case 1:
				// If only one parser is registered, return it
				for _, parser := range parsersForType {
					return parser, nil
				}
			default:
				// If multiple parsers are registered, return an error
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
func (v *ParserRegistry) Invalidate(dest Validatable) error {
	value := reflect.ValueOf(dest)
	if value.Kind() != reflect.Ptr || value.IsNil() {
		return ValidationError{reason: "Cannot invalidate a non ptr or nil value"}
	}

	elem := value.Elem()
	zeroStructFields(elem)

	return nil
}

///////////////////////////////////////////////////////////////////////////////
// Global Singleton and Package Functions
///////////////////////////////////////////////////////////////////////////////

var _globalValidator *ParserRegistry = nil

func init() {
	_defaultSourceParsers = []SourceParser{
		NewJsonByteSliceSourceParser(),
		NewJSONStringSourceParser(),
		// NewHTTPRequestParser(),
		// NewStringMapSourceParser(),
		// NewStringAnyMapSourceParser(),
	}

	var err error
	_globalValidator, err = NewValidator(ValidatorOpts{IncludeDefaults: true})
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize global validator: %v", err))
	}
}

// Package-level functions that delegate to the global validator

// RegisterParser registers a parser with the global validator.
// Accepts any SourceParser[T] and converts it to work with the registry.
func RegisterParser(parser SourceParser) error {
	return _globalValidator.RegisterParser(parser)
}

// Validate validates data using the global validator.
func Validate(data any, dest Validatable) error {
	return _globalValidator.Validate(data, dest)
}

// WithParser returns a ValidatorContext from the global validator.
func WithParser(parserName string) *ParserRegistryContext {
	return _globalValidator.WithParser(parserName)
}

// Invalidate invalidates a struct using the global validator.
func Invalidate(dest Validatable) error {
	return _globalValidator.Invalidate(dest)
}

// GetParser gets a parser from the global validator.
func GetParser(data any) (SourceParser, error) {
	return _globalValidator.tryGetDefaultParser(data)
}

// GetParserByName gets a specific parser by name from the global validator.
func GetParserByName(data any, parserName string) (SourceParser, error) {
	return _globalValidator.getParserByName(data, parserName)
}
