package validation

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

var (
	// ErrNoFieldSourcesInTag is returned when attempting to populate a field but
	// tags that define how to populate the field are provided.
	ErrNoFieldSourcesInTag = errors.New("no fields sources defined in tag but attempted to validate field")
)

const (
	JSONSourceTag   = "json"
	CookieSourceTag = "cookie"
	HeaderSourceTag = "header"
	QuerySourceTag  = "query"
)

// HTTPValidator is a type alias for Validator for HTTP requests.
type HTTPValidator = Validator[*HTTPRequestParser, Validatable]

// HTTPRequestParser is the implementation of ValidationParser for
// http.Request data sources.
//
// It supports the following tags on fields
//   - "json": Parse the field from the request body
//   - "cookie": Parse the field from the value of the cookie with given name
//   - "header": Parse the field from the header with given name
//   - "query": Parse the field from the URL query parameter with given name
//
// Each field can upto one instance of each tag, and the following modifiers
// are available:
//   - source:"<name>,omitempty": If the value is not provided by source,
//     attempt to retrieve it from the next source.
type HTTPRequestParser struct {
	request   *http.Request
	jsonBody  []byte
	bodyOnce  sync.Once
	bodyError error
}

func NewHTTPRequestParser(r *http.Request) *HTTPRequestParser {
	return &HTTPRequestParser{request: r}
}

// Parse implements the ValidationParser implementation for
// HTTPRequestParser.
func (hp *HTTPRequestParser) Parse(v Validatable) error {
	return hp.parseStructFields(v)
}

// par
func (hp *HTTPRequestParser) parseStructFields(v Validatable) error {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		structField := typ.Field(i)

		if !field.CanSet() {
			continue
		}

		if omitall, err := hp.parseField(field, structField); !omitall && err != nil {
			return fmt.Errorf("failed to parse field %s: %w", structField.Name, err)
		}
	}

	return nil
}

// parseField parses a single field based on its tags. It will attempt to parse
// from FieldSources based on the ordering of the fields tags.
//
// If the field is not found but is defined as 'required' for the current source,
// returns an error
//
// If all of the sources have OmitEmpty = true and no value is found, returns
// true and the last error during parsing.
//
// If not all of the sources have OmitEmpty = true and no value found, returns
// false and the last error during parsing.
func (hp *HTTPRequestParser) parseField(field reflect.Value, structField reflect.StructField) (bool, error) {
	sources := hp.setupFieldSources(structField)

	if len(sources) == 0 {

	}

	var lastErr error
	omitall := true
	for _, source := range sources {
		omitall = omitall && source.OmitEmpty

		value, found, err := hp.getValueFromSource(source)
		if err != nil {
			lastErr = err
			if source.Required {
				return omitall, err
			}
			continue
		}

		valueStr := fmt.Sprintf("%v", value)

		if found {
			return omitall, hp.setFieldValue(field, valueStr)
		}

		if source.Required {
			return omitall, fmt.Errorf("required field %s not found in source %s", source.Name, source.Source)
		}
	}

	return omitall, lastErr
}

// FieldSource represents a source for field data with options
type FieldSource struct {
	Source    string
	Name      string
	OmitEmpty bool
	Required  bool
}

// Parse field sources from struct tags
func (hp *HTTPRequestParser) setupFieldSources(structField reflect.StructField) []FieldSource {
	var sources []FieldSource

	// Parse different source types
	sourceParsers := map[string]func(string) FieldSource{
		JSONSourceTag:   hp.parseSourceTag,
		CookieSourceTag: hp.parseSourceTag,
		HeaderSourceTag: hp.parseSourceTag,
		QuerySourceTag:  hp.parseSourceTag,
	}

	for tagName, parser := range sourceParsers {
		if tagValue := structField.Tag.Get(tagName); tagValue != "" && tagValue != "-" {
			source := parser(tagValue)
			source.Source = tagName
			sources = append(sources, source)
		}
	}

	return sources
}

// Parse source tag (handles "name,omitempty" format)
func (hp *HTTPRequestParser) parseSourceTag(tag string) FieldSource {
	parts := strings.Split(tag, ",")
	source := FieldSource{
		Name:     strings.TrimSpace(parts[0]),
		Required: true,
	}

	for _, part := range parts[1:] {
		switch strings.TrimSpace(part) {
		case "omitempty":
			source.OmitEmpty = true
			source.Required = false
		case "required":
			source.Required = true
		}
	}

	return source
}

// Get value from specific source
func (hp *HTTPRequestParser) getValueFromSource(source FieldSource) (any, bool, error) {
	switch source.Source {
	case JSONSourceTag:
		return hp.getJSONValue(source.Name)
	case CookieSourceTag:
		return hp.getCookieValue(source.Name)
	case HeaderSourceTag:
		return hp.getHeaderValue(source.Name)
	case QuerySourceTag:
		return hp.getQueryValue(source.Name)
	default:
		return nil, false, fmt.Errorf("unknown source: %s", source.Source)
	}
}

func (hp *HTTPRequestParser) getJSONValue(fieldName string) (any, bool, error) {
	jsonBody, err := hp.getJSONBody()
	if err != nil {
		return nil, false, err
	}

	result := gjson.GetBytes(jsonBody, fieldName)
	if !result.Exists() {
		return nil, false, nil
	}

	return result.Value(), true, nil
}

// Read JSON body once, but don't parse it - just store the raw bytes
func (hp *HTTPRequestParser) getJSONBody() ([]byte, error) {
	hp.bodyOnce.Do(func() {
		if hp.request.Body == nil || hp.request.ContentLength == 0 {
			hp.jsonBody = []byte("{}")
			return
		}

		body, err := io.ReadAll(hp.request.Body)
		if err != nil {
			hp.bodyError = fmt.Errorf("failed to read request body: %w", err)
			return
		}

		hp.jsonBody = body
		if len(body) == 0 {
			hp.jsonBody = []byte("{}")
		}
	})

	return hp.jsonBody, hp.bodyError
}

// Get value from cookie
func (hp *HTTPRequestParser) getCookieValue(name string) (any, bool, error) {
	cookie, err := hp.request.Cookie(name)
	if err != nil {
		return nil, false, nil // Cookie not found, not an error
	}
	return cookie.Value, true, nil
}

// Get value from header
func (hp *HTTPRequestParser) getHeaderValue(name string) (any, bool, error) {
	value := hp.request.Header.Get(name)
	if value == "" {
		return nil, false, nil
	}

	// Handle Authorization header specially
	if name == "Authorization" && strings.HasPrefix(value, "Bearer ") {
		value = strings.TrimPrefix(value, "Bearer ")
	}

	return value, true, nil
}

// Get value from query parameters
func (hp *HTTPRequestParser) getQueryValue(name string) (any, bool, error) {
	values := hp.request.URL.Query()[name]
	if len(values) == 0 {
		return nil, false, nil
	}
	return values[0], true, nil
}

// Set field value with type conversion
func (hp *HTTPRequestParser) setFieldValue(field reflect.Value, value string) error {
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
