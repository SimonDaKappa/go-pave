package pave

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/google/uuid"
)

///////////////////////////////////////////////////////////////////////////////
// Errors
///////////////////////////////////////////////////////////////////////////////

// ValidationError is a an error that occured during validating
// from a ValidationParser into a Validatable impl.
type ValidationError struct {
	reason string
}

// Error implements the error interface
func (ve ValidationError) Error() string {
	return fmt.Sprintf("Failed to validate: %s", ve.reason)
}

var (
	ErrNoFieldSourcesInTag            = errors.New("no fields sources defined in tag but attempted to validate field")
	ErrParserAlreadyRegistered        = errors.New("a parser for this source-type is already registered")
	ErrNoParser                       = errors.New("no built-in or registered parser found for this type")
	ErrNoParserBuiltin                = errors.New("no built-in parser found for this type")
	ErrNoParserRegistered             = errors.New("no registered parser found for this type")
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
	Validate() error
}

// SourceParser defines the interface for extracting information
// from the implementation of this interface and filling a Validatable
// type with it.
type SourceParser interface {
	// Parse extracts the information from the implementation and populates
	// v using the execution chain system.
	Parse(data any, v Validatable) error
	GetSourceType() reflect.Type
	// BuildParseChain builds and caches an execution chain for the given type
	BuildParseChain(t reflect.Type) (*BaseExecutionChain, error)
	// GetParseChain retrieves a cached execution chain or builds one if needed
	GetParseChain(t reflect.Type) (*BaseExecutionChain, error)
}

// Validator is the validation entry point for a Validatable type.
//
// It takes two generic types:
//   - P ValidationParser: The source of information that also implements
//     the methods to fill V
type Validator struct {
	HTTPParser        *HTTPRequestParser
	RegisteredParsers map[reflect.Type]SourceParser
}

type ValidatorOpts struct {
	Parsers []SourceParser
}

func NewValidator(opts ValidatorOpts) (*Validator, error) {
	v := &Validator{
		HTTPParser:        NewHTTPRequestParser(),
		RegisteredParsers: make(map[reflect.Type]SourceParser),
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

	if _, exists := v.RegisteredParsers[t]; exists {
		return ErrParserAlreadyRegistered
	}

	v.RegisteredParsers[t] = parser
	return nil
}

// Validate populates dest based on the implementaion of source's
// parsing logic.
//
// # It expects the passed v to be a pointer
//
// If validation fails, it will return the validation error
// and zero all of dest's fields.
func (v *Validator) Validate(data any, dest Validatable) error {

	parser, err := v.GetParser(data)
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

func (v *Validator) GetParser(data any) (SourceParser, error) {
	t := reflect.TypeOf(data)

	parser, err := v.getParserFromBuiltins(t)
	if err == nil {
		return parser, nil
	}

	parser, err = v.getParserFromRegistered(t)
	if err == nil {
		return parser, nil
	}

	return nil, ErrNoParser
}

func (v *Validator) getParserFromBuiltins(t reflect.Type) (SourceParser, error) {
	switch t {
	case __httpRequestType:
		return v.HTTPParser, nil
	default:
		return nil, ErrNoParserBuiltin
	}
}

func (v *Validator) getParserFromRegistered(t reflect.Type) (SourceParser, error) {
	parser, exists := v.RegisteredParsers[t]
	if !exists {
		return nil, ErrNoParserRegistered
	}

	return parser, nil
}

// Invalidate clears a partially or fully validated v by
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
// Helpers
///////////////////////////////////////////////////////////////////////////////

// Set field value with type conversion
func setFieldValue(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		// If the field is a string, set it directly
		field.SetString(value)
	case reflect.Array:
		// Handle UUID array type
		if field.Type() == reflect.TypeOf(uuid.UUID{}) {
			uuidValue, err := uuid.Parse(value)
			if err != nil {
				return fmt.Errorf("error converting query value to UUID: %w", err)
			}
			field.Set(reflect.ValueOf(uuidValue))
		}
	case reflect.Int:
		// If the field is an int, convert the query value to int
		intValue, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("error converting query value to int: %w", err)
		}
		field.SetInt(int64(intValue))
	case reflect.Bool:
		// If the field is a bool, convert the query value to bool
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("error converting query value to bool: %w", err)
		}
		field.SetBool(boolValue)
	case reflect.Float64:
		// If the field is a float64, convert the query value to float64
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("error converting query value to float64: %w", err)
		}
		field.SetFloat(floatValue)
	case reflect.Slice:
		// If the field is a byte slice, detect if value is base64 or raw string
		if field.Type().Elem().Kind() == reflect.Uint8 {
			data := []byte(value)
			field.SetBytes(data)
		} else {
			return fmt.Errorf("unsupported slice type for query: %s", field.Type().Name())
		}
	case reflect.Struct:
		// Handle uuid.UUID type
		if field.Type() == reflect.TypeOf(uuid.UUID{}) {
			uuidValue, err := uuid.Parse(value)
			if err != nil {
				return fmt.Errorf("error converting query value to UUID: %w", err)
			}
			field.Set(reflect.ValueOf(uuidValue))
		} else {
			return fmt.Errorf("unsupported struct type for query: %s", field.Type().Name())
		}
	default:
		return fmt.Errorf("unsupported field type for query: %s", field.Type().Name())
	}

	return nil
}

// zeroStructFields recursively sets all fields of a struct to
// their default vlaues.
func zeroStructFields(value reflect.Value) {
	if value.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		if !field.CanSet() {
			continue
		}
		if field.Kind() == reflect.Struct && !field.Type().ConvertibleTo(reflect.TypeOf(time.Time{})) {
			zeroStructFields(field)
		} else {
			field.Set(reflect.Zero(field.Type()))
		}
	}
}
