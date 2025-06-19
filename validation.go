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

const (
	ContentEncodingUTF8        string = "UTF-8"
	ContentTypeApplicationJSON string = "application/json"
	ContentTypeDelimeter              = ";"
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

// Validator is the main struct that handles validation
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
type Validator struct {
	RegisteredParsers map[reflect.Type]map[string]SourceParser // source type -> parser name -> parser
}

// ValidatorContext provides a curried validator with a specific parser selection
type ValidatorContext struct {
	validator  *Validator
	parserName string
}

var (
	_defaultSourceParsers = []SourceParser{
		NewHTTPRequestParser(),
		NewJsonSourceParser(),
		NewStringMapSourceParser(),
	}
)

type ValidatorOpts struct {
	Parsers         []SourceParser
	IncludeDefaults bool
}

func NewValidator(opts ValidatorOpts) (*Validator, error) {
	v := &Validator{
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

func (v *Validator) RegisterParser(parser SourceParser) error {
	t := parser.GetSourceType()
	name := parser.GetParserName()

	if v.RegisteredParsers[t] == nil {
		v.RegisteredParsers[t] = make(map[string]SourceParser)
	}

	if _, exists := v.RegisteredParsers[t][name]; exists {
		return ErrParserAlreadyRegistered
	}

	v.RegisteredParsers[t][name] = parser
	return nil
}

// WithParser returns a ValidatorContext that will use the specified parser
// for validation. This is useful when multiple parsers are registered for
// the same source type.
func (v *Validator) WithParser(parserName string) *ValidatorContext {
	return &ValidatorContext{
		validator:  v,
		parserName: parserName,
	}
}

// Validate populates dest based on the specified parser's logic.
// It expects the passed dest to be a pointer.
// If validation fails, it will return the validation error and zero all of dest's fields.
func (vc *ValidatorContext) Validate(data any, dest Validatable) error {
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
func (v *Validator) Validate(data any, dest Validatable) error {

	if dest == nil {
		return ValidationError{reason: "dest cannot be nil"}
	}
	if reflect.TypeOf(dest).Kind() != reflect.Ptr || reflect.ValueOf(dest).IsNil() {
		return ValidationError{reason: "dest must be a non-nil pointer to a Validatable type"}
	}

	parser, err := v.tryGetDefaultParser(data)
	if err != nil {
		return err
	}

	err = parser.Parse(data, dest)
	if err != nil {
		v.Invalidate(dest)
		return ValidationError{err.Error()}
	}

	err = dest.Validate()
	if err != nil {
		v.Invalidate(dest)
		return ValidationError{err.Error()}
	}

	return nil
}

// tryGetDefaultParser retrieves the appropriate SourceParser for the given data type.
//
// If multiple parsers are found for the same source type, it returns an error
// indicating that WithParser() should be used to specify which one.
//
// If no parser is found, it returns ErrNoParser.
func (v *Validator) tryGetDefaultParser(data any) (SourceParser, error) {
	t := reflect.TypeOf(data)

	parser, err := v.getParserByName(t, "")
	if err != nil {
		return nil, err
	}

	return parser, nil
}

// getParserByName retrieves a specific parser by name for the given data type.
//
// No name provided: If there is only one parser registered for the type,
// it returns that parser. If multiple parsers are registered, it returns an error
func (v *Validator) getParserByName(data any, parserName string) (SourceParser, error) {
	t := reflect.TypeOf(data)

	// Check registered parsers
	if parsersForType, exists := v.RegisteredParsers[t]; exists {

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
func (v *Validator) Invalidate(dest Validatable) error {
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

var _globalValidator *Validator = nil

func init() {
	var err error
	_globalValidator, err = NewValidator(ValidatorOpts{IncludeDefaults: true})
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize global validator: %v", err))
	}
}

// Package-level functions that delegate to the global validator

// RegisterParser registers a parser with the global validator.
func RegisterParser(parser SourceParser) error {
	return _globalValidator.RegisterParser(parser)
}

// Validate validates data using the global validator.
func Validate(data any, dest Validatable) error {
	return _globalValidator.Validate(data, dest)
}

// WithParser returns a ValidatorContext from the global validator.
func WithParser(parserName string) *ValidatorContext {
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
