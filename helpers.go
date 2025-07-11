package pave

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/google/uuid"
)

///////////////////////////////////////////////////////////////////////////////
// Helpers
///////////////////////////////////////////////////////////////////////////////

// Set field value with type conversion
//
// Currently supports:
//   - string to string
//   - string to int
//   - string to bool
//   - string to float64
//   - string to uuid.UUID
//   - string to []byte (raw byte slice)
//   - string to array of uuid.UUID
//   - string to struct with uuid.UUID field
//   - string to struct with time.Time field
func setFieldValue(field reflect.Value, value string) error {
	switch field.Kind() {
	// If the field is a string, set it directly
	case reflect.String:
		field.SetString(value)
	// If the field is a array, check if it is an
	case reflect.Array:
		if field.Type() == UUIDType {
			uuidValue, err := uuid.Parse(value)
			if err != nil {
				return fmt.Errorf("error converting query value to UUID: %w", err)
			}
			field.Set(reflect.ValueOf(uuidValue))
		}
	// If the field is an int, convert the query value to int
	case reflect.Int:
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
