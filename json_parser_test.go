package pave

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test JSON parsers
func TestJSONByteSliceSourceParser(t *testing.T) {
	parser := NewJsonByteSliceSourceParser()

	t.Run("NewJsonByteSliceSourceParser", func(t *testing.T) {
		assert.NotNil(t, parser)
	})

	t.Run("SourceType", func(t *testing.T) {
		sourceType := parser.SourceType()
		assert.Equal(t, JSONByteSliceType, sourceType)
	})

	t.Run("Name", func(t *testing.T) {
		name := parser.Name()
		assert.Equal(t, JSONByteSliceParserName, name)
	})

	t.Run("Parse_Success", func(t *testing.T) {
		jsonData := []byte(`{"name": "John", "age": 30}`)

		var result struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		err := parser.Parse(jsonData, &result)
		require.NoError(t, err)
		assert.Equal(t, "John", result.Name)
		assert.Equal(t, 30, result.Age)
	})

	t.Run("Parse_NestedJson", func(t *testing.T) {
		jsonData := []byte(`{"person": {"name": "John", "age": 30}}`)
		var result struct {
			Person struct {
				Name string `json:"name"`
				Age  int    `json:"age"`
			} `json:"person"`
		}
		err := parser.Parse(jsonData, &result)
		require.NoError(t, err)
		assert.Equal(t, "John", result.Person.Name)
		assert.Equal(t, 30, result.Person.Age)
	})

	t.Run("Parse_InvalidJSON", func(t *testing.T) {
		jsonData := []byte(`{"name": "John", "age":}`) // Invalid JSON

		var result struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		err := parser.Parse(jsonData, &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error unmarshaling JSON data")
	})
}

func TestJSONStringSourceParser(t *testing.T) {
	parser := NewJSONStringSourceParser()

	t.Run("NewJSONStringSourceParser", func(t *testing.T) {
		assert.NotNil(t, parser)
	})

	t.Run("SourceType", func(t *testing.T) {
		sourceType := parser.SourceType()
		assert.Equal(t, StringType, sourceType)
	})

	t.Run("Name", func(t *testing.T) {
		name := parser.Name()
		assert.Equal(t, JSONStringParserName, name)
	})

	t.Run("Parse_Success", func(t *testing.T) {
		jsonString := `{"name": "John", "age": 30}`

		var result struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		err := parser.Parse(&jsonString, &result)
		require.NoError(t, err)
		assert.Equal(t, "John", result.Name)
		assert.Equal(t, 30, result.Age)
	})

	t.Run("Parse_NestedJson", func(t *testing.T) {
		jsonString := `{"person": {"name": "John", "age": 30}}`
		var result struct {
			Person struct {
				Name string `json:"name"`
				Age  int    `json:"age"`
			} `json:"person"`
		}
		err := parser.Parse(&jsonString, &result)
		require.NoError(t, err)
		assert.Equal(t, "John", result.Person.Name)
		assert.Equal(t, 30, result.Person.Age)
	})

	t.Run("Parse_InvalidJSON", func(t *testing.T) {
		jsonString := `{"name": "John", "age":}` // Invalid JSON

		var result struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		err := parser.Parse(&jsonString, &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error unmarshaling JSON data")
	})

	t.Run("Parse_EmptyString", func(t *testing.T) {
		jsonString := ""

		var result struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		err := parser.Parse(&jsonString, &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error unmarshaling JSON data")
	})
}
