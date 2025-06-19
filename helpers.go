package pave

import (
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
