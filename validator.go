package validation

import (
	"fmt"
	"reflect"
	"time"
)

// ValidationError is a an error that occured during validating
// from a ValidationParser into a Validatable impl.
type ValidationError struct {
	reason string
}

// Error implements the error interface
func (ve ValidationError) Error() string {
	return fmt.Sprintf("Failed to validate: %s", ve.reason)
}

// Validatable is an interface that marks a struct as expecting
// to be populated by a ValidationParser and later have its fields
// validated by calling Validate()
type Validatable interface {
	Validate() error
}

// ValidationParser defines the interface for extracting information
// from the implementation of this interface and filling a Validatable
// type with it.
type ValidationParser interface {
	// Parse extracts the information from the implementation and populates
	// v.
	// Will return a ValidationError if it fails.
	Parse(v Validatable) error
}

// Validator is the validation entry point for a Validatable type.
//
// It takes two generic types:
//   - P ValidationParser: The source of information that also implements
//     the methods to fill V
type Validator[P ValidationParser] struct{}

// Validate populates dest based on the implementaion of source's
// parsing logic.
//
// # It expects the passed v to be a pointer
//
// If validation fails, it will return the validation error
// and zero all of dest's fields.
func (validator Validator[P]) Validate(source P, dest Validatable) error {
	err := source.Parse(dest)
	if err != nil {
		validator.Invalidate(dest)
		return ValidationError{err.Error()}
	}

	return nil
}

// Invalidate clears a partially or fully validated v by
// setting each field to its default value.
//
// # It expects the passed v to be a pointer
//
// An error is returned if the argument is not reflect-able
func (validator Validator[P]) Invalidate(v Validatable) error {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return ValidationError{reason: "Cannot invalidate an Ptr or nil value"}
	}
	elem := val.Elem()
	validator.zeroStructFields(elem)
}

// zeroStructFields recursively sets all fields of a struct to
// their default vlaues.
func (validator Validator[P]) zeroStructFields(val reflect.Value) {
	if val.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if !field.CanSet() {
			continue
		}
		if field.Kind() == reflect.Struct && !field.Type().ConvertibleTo(reflect.TypeOf(time.Time{})) {
			validator.zeroStructFields(field)
		} else {
			field.Set(reflect.Zero(field.Type()))
		}
	}
}
