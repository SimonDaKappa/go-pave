package pave

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test BindingResult functions
func TestBindingResult(t *testing.T) {
	t.Run("BindingResultError", func(t *testing.T) {
		testErr := errors.New("test error")
		result := BindingResultError(testErr)

		assert.Nil(t, result.Value)
		assert.False(t, result.Found)
		assert.Equal(t, testErr, result.Error)
	})

	t.Run("BindingResultNotFound", func(t *testing.T) {
		result := BindingResultNotFound()

		assert.Nil(t, result.Value)
		assert.False(t, result.Found)
		assert.Nil(t, result.Error)
	})

	t.Run("BindingResultValue", func(t *testing.T) {
		testValue := "test value"
		result := BindingResultValue(testValue)

		assert.Equal(t, testValue, result.Value)
		assert.True(t, result.Found)
		assert.Nil(t, result.Error)
	})
}
