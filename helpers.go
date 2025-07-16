package pave

import (
	"encoding"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/google/uuid"
)

///////////////////////////////////////////////////////////////////////////////
// Helpers
///////////////////////////////////////////////////////////////////////////////

// Set field value with type conversion
//
// Currently supports:
//   - string to string
//   - string to int (with overflow checking)
//   - string to bool
//   - string to float64 (with overflow checking)
//   - string to uuid.UUID
//   - string to []byte (raw byte slice)
//   - string to array of uuid.UUID
//   - string to struct with uuid.UUID field
//   - string to struct with time.Time field
//   - TextUnmarshaler support for custom types
//   - Interface{} support for any type
func setFieldValue(field reflect.Value, value string) error {
	// Handle nil/empty values
	if value == "" {
		return handleEmptyValue(field)
	}

	// Check for TextUnmarshaler interface
	if field.CanInterface() {
		if unmarshaler, ok := field.Interface().(encoding.TextUnmarshaler); ok {
			return unmarshaler.UnmarshalText([]byte(value))
		}
		// Check for pointer to TextUnmarshaler
		if field.CanAddr() {
			if unmarshaler, ok := field.Addr().Interface().(encoding.TextUnmarshaler); ok {
				return unmarshaler.UnmarshalText([]byte(value))
			}
		}
	}

	switch field.Kind() {
	case reflect.String:
		return setStringValue(field, value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return setIntValue(field, value)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return setUintValue(field, value)
	case reflect.Float32, reflect.Float64:
		return setFloatValue(field, value)
	case reflect.Complex64, reflect.Complex128:
		return setComplexValue(field, value)
	case reflect.Bool:
		return setBoolValue(field, value)
	case reflect.Slice:
		return setSliceValue(field, value)
	case reflect.Array:
		return setArrayValue(field, value)
	case reflect.Struct:
		return setStructValue(field, value)
	case reflect.Interface:
		return setInterfaceValue(field, value)
	default:
		return fmt.Errorf("unsupported field type: %s", field.Type().Name())
	}
}

// handleEmptyValue handles empty string values for different field types
func handleEmptyValue(field reflect.Value) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString("")
		return nil
	case reflect.Slice, reflect.Map, reflect.Ptr, reflect.Interface:
		field.SetZero()
		return nil
	default:
		return fmt.Errorf("cannot set empty value for field type: %s", field.Type().Name())
	}
}

// setStringValue sets string field values
func setStringValue(field reflect.Value, value string) error {
	field.SetString(value)
	return nil
}

// setIntValue sets integer field values with overflow checking
func setIntValue(field reflect.Value, value string) error {
	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fmt.Errorf("error converting value to int: %w", err)
	}

	if field.OverflowInt(intValue) {
		return fmt.Errorf("value %d overflows %s", intValue, field.Type().Name())
	}

	field.SetInt(intValue)
	return nil
}

// setUintValue sets unsigned integer field values with overflow checking
func setUintValue(field reflect.Value, value string) error {
	uintValue, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return fmt.Errorf("error converting value to uint: %w", err)
	}

	if field.OverflowUint(uintValue) {
		return fmt.Errorf("value %d overflows %s", uintValue, field.Type().Name())
	}

	field.SetUint(uintValue)
	return nil
}

// setFloatValue sets float field values with overflow checking
func setFloatValue(field reflect.Value, value string) error {
	floatValue, err := strconv.ParseFloat(value, field.Type().Bits())
	if err != nil {
		return fmt.Errorf("error converting value to float: %w", err)
	}

	if field.OverflowFloat(floatValue) {
		return fmt.Errorf("value %f overflows %s", floatValue, field.Type().Name())
	}

	field.SetFloat(floatValue)
	return nil
}

// setComplexValue sets complex field values
func setComplexValue(field reflect.Value, value string) error {
	complexValue, err := strconv.ParseComplex(value, field.Type().Bits())
	if err != nil {
		return fmt.Errorf("error converting value to complex: %w", err)
	}

	if field.OverflowComplex(complexValue) {
		return fmt.Errorf("value %v overflows %s", complexValue, field.Type().Name())
	}

	field.SetComplex(complexValue)
	return nil
}

// setBoolValue sets boolean field values with better validation
//
// Many common boolean representations are supported:
//   - "true", "1", "yes", "on" (case insensitive)
//   - "false", "0", "no", "off" (case insensitive)
//   - Standard boolean parsing using strconv.ParseBool
func setBoolValue(field reflect.Value, value string) error {
	// Handle common boolean representations
	switch value {
	case "true", "1", "yes", "on", "True", "TRUE", "YES", "ON":
		field.SetBool(true)
		return nil
	case "false", "0", "no", "off", "False", "FALSE", "NO", "OFF":
		field.SetBool(false)
		return nil
	default:
		// Fall back to standard parsing
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("error converting value to bool: %w", err)
		}
		field.SetBool(boolValue)
		return nil
	}
}

// setSliceValue sets slice field values
func setSliceValue(field reflect.Value, value string) error {
	elemType := field.Type().Elem()

	switch elemType.Kind() {
	case reflect.Uint8:
		// []byte slice
		field.SetBytes([]byte(value))
		return nil
	default:
		return fmt.Errorf("unsupported slice type: %s", field.Type().Name())
	}
}

// setArrayValue sets array field values
func setArrayValue(field reflect.Value, value string) error {
	if field.Type() == UUIDType {
		uuidValue, err := uuid.Parse(value)
		if err != nil {
			return fmt.Errorf("error converting value to UUID: %w", err)
		}
		field.Set(reflect.ValueOf(uuidValue))
		return nil
	}

	return fmt.Errorf("unsupported array type: %s", field.Type().Name())
}

// setStructValue sets struct field values for special types
func setStructValue(field reflect.Value, value string) error {
	fieldType := field.Type()

	// Handle UUID type
	if fieldType == UUIDType {
		uuidValue, err := uuid.Parse(value)
		if err != nil {
			return fmt.Errorf("error converting value to UUID: %w", err)
		}
		field.Set(reflect.ValueOf(uuidValue))
		return nil
	}

	// Handle time.Time type
	if fieldType == TimeType {
		timeValue, err := time.Parse(time.RFC3339, value)
		if err != nil {
			// Try common time formats
			formats := []string{
				time.RFC3339,
				time.RFC3339Nano,
				"2006-01-02T15:04:05",
				"2006-01-02 15:04:05",
				"2006-01-02",
				"15:04:05",
			}

			for _, format := range formats {
				if timeValue, err = time.Parse(format, value); err == nil {
					break
				}
			}

			if err != nil {
				return fmt.Errorf("error converting value to time.Time: %w", err)
			}
		}
		field.Set(reflect.ValueOf(timeValue))
		return nil
	}

	return fmt.Errorf("unsupported struct type: %s", fieldType.Name())
}

// setInterfaceValue sets interface{} field values
func setInterfaceValue(field reflect.Value, value string) error {
	if field.NumMethod() != 0 {
		return fmt.Errorf("cannot set value for interface with methods: %s", field.Type().Name())
	}

	// For empty interface, store as string
	field.Set(reflect.ValueOf(value))
	return nil
}

// zeroStructFields recursively sets all fields of a struct to
// their default values.
func zeroStructFields(value reflect.Value) {
	if value.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		if !field.CanSet() {
			continue
		}
		if field.Kind() == reflect.Struct && !isSpecialStructType(field.Type()) {
			zeroStructFields(field)
		} else {
			field.Set(reflect.Zero(field.Type()))
		}
	}
}

// isSpecialStructType checks if a struct type should be treated as a primitive
// rather than being recursively parsed. Special types include time.Time, uuid.UUID, etc.
func isSpecialStructType(t reflect.Type) bool {
	// List of struct types that should be treated as primitives
	specialTypes := []reflect.Type{TimeType, UUIDType}

	for _, specialType := range specialTypes {
		if t == specialType {
			return true
		}
	}
	return false
}

func ParseTypeErasedPointer[S any](
	source any,
	dest any,
	parse func(source *S, dest any) error,
) error {
	return func(source any, dest any) error {
		typedSource, ok := source.(*S)
		if !ok {
			return fmt.Errorf("expected source type %T, got %T", *new(S), source)
		}

		if (reflect.TypeOf(dest).Kind() != reflect.Ptr) ||
			(reflect.TypeOf(dest).Elem().Kind() != reflect.Struct) {
			return fmt.Errorf("destination must be a pointer to a struct, got %T", dest)
		}

		return parse(typedSource, dest)
	}(source, dest)
}

func ParseTypeErasedSlice[S any](
	source any,
	dest any,
	parse func(source []S, dest any) error,
) error {
	typedSource, ok := source.([]S)
	if !ok {
		return fmt.Errorf("expected source type %T, got %T", *new(S), source)
	}
	if (reflect.TypeOf(dest).Kind() != reflect.Ptr) ||
		(reflect.TypeOf(dest).Elem().Kind() != reflect.Struct) {
		return fmt.Errorf("destination must be a pointer to a struct, got %T", dest)
	}
	return parse(typedSource, dest)
}
