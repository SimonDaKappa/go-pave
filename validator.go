package validation

import (
	"fmt"
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
	Validate() ValidationError
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
//   - V Validatable: The destination for information extracted from P.
type Validator[P ValidationParser, V Validatable] struct {
}

// Validate populates dest based on the implementaion of source's
// parsing logic.
//
// If validation fails, it will return the validation error
// and zero all of dest's fields.
func (val Validator[P, V]) Validate(source P, dest V) error {
	err := source.Parse(dest)
	if err != nil {
		val.Invalidate(dest)
		return ValidationError{err.Error()}
	}

	return nil
}

func (val Validator[P, V]) Invalidate(v Validatable) {
	// do something
}
