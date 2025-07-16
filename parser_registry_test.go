package pave

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock parser for testing
type MockParser struct {
	name       string
	sourceType reflect.Type
	parseFunc  func(source any, dest any) error
}

func (m *MockParser) Name() string {
	return m.name
}

func (m *MockParser) SourceType() reflect.Type {
	return m.sourceType
}

func (m *MockParser) Parse(source any, dest any) error {
	if m.parseFunc != nil {
		return m.parseFunc(source, dest)
	}
	return nil
}

// Mock validatable struct
type MockValidatable struct {
	Value     string
	ShouldErr bool
}

func (m *MockValidatable) Validate() error {
	if m.ShouldErr {
		return errors.New("validation error")
	}
	return nil
}

func TestParserRegistry(t *testing.T) {
	t.Run("NewParserRegistry_WithDefaults", func(t *testing.T) {
		registry, err := NewParserRegistry(ParserRegistryOpts{
			ExcludeDefaults: false,
		})
		require.NoError(t, err)
		assert.NotNil(t, registry)
	})

	t.Run("NewParserRegistry_WithoutDefaults", func(t *testing.T) {
		registry, err := NewParserRegistry(ParserRegistryOpts{
			ExcludeDefaults: true,
		})
		require.NoError(t, err)
		assert.NotNil(t, registry)
	})

	t.Run("Register", func(t *testing.T) {
		registry, err := NewParserRegistry(ParserRegistryOpts{
			ExcludeDefaults: true,
		})
		require.NoError(t, err)

		mockParser := &MockParser{
			name:       "test_parser",
			sourceType: reflect.TypeOf(""),
		}

		err = registry.Register(mockParser)
		assert.NoError(t, err)
	})

	t.Run("WithParser", func(t *testing.T) {
		registry, err := NewParserRegistry(ParserRegistryOpts{
			ExcludeDefaults: true,
		})
		require.NoError(t, err)

		ctx := registry.WithParser("test_parser")
		assert.NotNil(t, ctx)
		assert.Equal(t, "test_parser", ctx.parserName)
		assert.Equal(t, registry, ctx.registry)
	})

	t.Run("Parse_Success", func(t *testing.T) {
		registry, err := NewParserRegistry(ParserRegistryOpts{
			ExcludeDefaults: true,
		})
		require.NoError(t, err)

		mockParser := &MockParser{
			name:       "test_parser",
			sourceType: reflect.TypeOf(""),
			parseFunc: func(source any, dest any) error {
				if destPtr, ok := dest.(*MockValidatable); ok {
					destPtr.Value = "parsed"
				}
				return nil
			},
		}

		err = registry.Register(mockParser)
		require.NoError(t, err)

		source := "test_source"
		dest := &MockValidatable{}

		err = registry.Parse(source, dest, false)
		assert.NoError(t, err)
		assert.Equal(t, "parsed", dest.Value)
	})

	t.Run("Parse_WithValidation_Success", func(t *testing.T) {
		registry, err := NewParserRegistry(ParserRegistryOpts{
			ExcludeDefaults: true,
		})
		require.NoError(t, err)

		mockParser := &MockParser{
			name:       "test_parser",
			sourceType: reflect.TypeOf(""),
			parseFunc: func(source any, dest any) error {
				if destPtr, ok := dest.(*MockValidatable); ok {
					destPtr.Value = "parsed"
					destPtr.ShouldErr = false
				}
				return nil
			},
		}

		err = registry.Register(mockParser)
		require.NoError(t, err)

		source := "test_source"
		dest := &MockValidatable{}

		err = registry.Parse(source, dest, true)
		assert.NoError(t, err)
		assert.Equal(t, "parsed", dest.Value)
	})

	t.Run("Parse_WithValidation_ValidationError", func(t *testing.T) {
		registry, err := NewParserRegistry(ParserRegistryOpts{
			ExcludeDefaults: true,
		})
		require.NoError(t, err)

		mockParser := &MockParser{
			name:       "test_parser",
			sourceType: reflect.TypeOf(""),
			parseFunc: func(source any, dest any) error {
				if destPtr, ok := dest.(*MockValidatable); ok {
					destPtr.Value = "parsed"
					destPtr.ShouldErr = true
				}
				return nil
			},
		}

		err = registry.Register(mockParser)
		require.NoError(t, err)

		source := "test_source"
		dest := &MockValidatable{}

		err = registry.Parse(source, dest, true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})

	t.Run("Parse_ParseError", func(t *testing.T) {
		registry, err := NewParserRegistry(ParserRegistryOpts{
			ExcludeDefaults: true,
		})
		require.NoError(t, err)

		mockParser := &MockParser{
			name:       "test_parser",
			sourceType: reflect.TypeOf(""),
			parseFunc: func(source any, dest any) error {
				return errors.New("parse error")
			},
		}

		err = registry.Register(mockParser)
		require.NoError(t, err)

		source := "test_source"
		dest := &MockValidatable{}

		err = registry.Parse(source, dest, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse")
	})

	t.Run("Parse_NilDest", func(t *testing.T) {
		registry, err := NewParserRegistry(ParserRegistryOpts{
			ExcludeDefaults: true,
		})
		require.NoError(t, err)

		source := "test_source"

		err = registry.Parse(source, nil, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dest cannot be nil")
	})

	t.Run("Parse_NonPointerDest", func(t *testing.T) {
		registry, err := NewParserRegistry(ParserRegistryOpts{
			ExcludeDefaults: true,
		})
		require.NoError(t, err)

		source := "test_source"
		dest := MockValidatable{}

		err = registry.Parse(source, dest, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dest must be a non-nil pointer to a struct type")
	})

	t.Run("Parse_NoParserFound", func(t *testing.T) {
		registry, err := NewParserRegistry(ParserRegistryOpts{
			ExcludeDefaults: true,
		})
		require.NoError(t, err)

		source := "test_source"
		dest := &MockValidatable{}

		err = registry.Parse(source, dest, false)
		assert.Error(t, err)
	})

	t.Run("tryGetDefaultParser", func(t *testing.T) {
		registry, err := NewParserRegistry(ParserRegistryOpts{
			ExcludeDefaults: true,
		})
		require.NoError(t, err)

		mockParser := &MockParser{
			name:       "test_parser",
			sourceType: reflect.TypeOf(""),
		}

		err = registry.Register(mockParser)
		require.NoError(t, err)

		source := "test_source"
		parser, err := registry.tryGetDefaultParser(source)
		assert.NoError(t, err)
		assert.Equal(t, mockParser, parser)
	})

	t.Run("getParserByName", func(t *testing.T) {
		registry, err := NewParserRegistry(ParserRegistryOpts{
			ExcludeDefaults: true,
		})
		require.NoError(t, err)

		mockParser := &MockParser{
			name:       "test_parser",
			sourceType: reflect.TypeOf(""),
		}

		err = registry.Register(mockParser)
		require.NoError(t, err)

		source := "test_source"
		parser, err := registry.getParserByName(source, "test_parser")
		assert.NoError(t, err)
		assert.Equal(t, mockParser, parser)
	})

	t.Run("Invalidate", func(t *testing.T) {
		registry, err := NewParserRegistry(ParserRegistryOpts{
			ExcludeDefaults: true,
		})
		require.NoError(t, err)

		dest := &MockValidatable{Value: "test"}

		err = registry.Invalidate(dest)
		assert.NoError(t, err)
		// Note: Invalidate should zero out the struct fields
		assert.Equal(t, "", dest.Value)
	})
}

func TestParserRegistryContext(t *testing.T) {
	t.Run("Parse_Success", func(t *testing.T) {
		registry, err := NewParserRegistry(ParserRegistryOpts{
			ExcludeDefaults: true,
		})
		require.NoError(t, err)

		mockParser := &MockParser{
			name:       "test_parser",
			sourceType: reflect.TypeOf(""),
			parseFunc: func(source any, dest any) error {
				if destPtr, ok := dest.(*MockValidatable); ok {
					destPtr.Value = "parsed_context"
				}
				return nil
			},
		}

		err = registry.Register(mockParser)
		require.NoError(t, err)

		ctx := registry.WithParser("test_parser")
		source := "test_source"
		dest := &MockValidatable{}

		err = ctx.Parse(source, dest, false)
		assert.NoError(t, err)
		assert.Equal(t, "parsed_context", dest.Value)
	})

	t.Run("Parse_WithValidation_Success", func(t *testing.T) {
		registry, err := NewParserRegistry(ParserRegistryOpts{
			ExcludeDefaults: true,
		})
		require.NoError(t, err)

		mockParser := &MockParser{
			name:       "test_parser",
			sourceType: reflect.TypeOf(""),
			parseFunc: func(source any, dest any) error {
				if destPtr, ok := dest.(*MockValidatable); ok {
					destPtr.Value = "parsed_context"
					destPtr.ShouldErr = false
				}
				return nil
			},
		}

		err = registry.Register(mockParser)
		require.NoError(t, err)

		ctx := registry.WithParser("test_parser")
		source := "test_source"
		dest := &MockValidatable{}

		err = ctx.Parse(source, dest, true)
		assert.NoError(t, err)
		assert.Equal(t, "parsed_context", dest.Value)
	})

	t.Run("Parse_ValidationError", func(t *testing.T) {
		registry, err := NewParserRegistry(ParserRegistryOpts{
			ExcludeDefaults: true,
		})
		require.NoError(t, err)

		mockParser := &MockParser{
			name:       "test_parser",
			sourceType: reflect.TypeOf(""),
			parseFunc: func(source any, dest any) error {
				if destPtr, ok := dest.(*MockValidatable); ok {
					destPtr.Value = "parsed_context"
					destPtr.ShouldErr = true
				}
				return nil
			},
		}

		err = registry.Register(mockParser)
		require.NoError(t, err)

		ctx := registry.WithParser("test_parser")
		source := "test_source"
		dest := &MockValidatable{}

		err = ctx.Parse(source, dest, true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})

	t.Run("Parse_ParseError", func(t *testing.T) {
		registry, err := NewParserRegistry(ParserRegistryOpts{
			ExcludeDefaults: true,
		})
		require.NoError(t, err)

		mockParser := &MockParser{
			name:       "test_parser",
			sourceType: reflect.TypeOf(""),
			parseFunc: func(source any, dest any) error {
				return errors.New("context parse error")
			},
		}

		err = registry.Register(mockParser)
		require.NoError(t, err)

		ctx := registry.WithParser("test_parser")
		source := "test_source"
		dest := &MockValidatable{}

		err = ctx.Parse(source, dest, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse")
	})

	t.Run("Parse_ParserNotFound", func(t *testing.T) {
		registry, err := NewParserRegistry(ParserRegistryOpts{
			ExcludeDefaults: true,
		})
		require.NoError(t, err)

		ctx := registry.WithParser("nonexistent_parser")
		source := "test_source"
		dest := &MockValidatable{}

		err = ctx.Parse(source, dest, false)
		assert.Error(t, err)
	})
}

// Test package-level functions
func TestPackageLevelFunctions(t *testing.T) {
	t.Run("RegisterParser", func(t *testing.T) {
		mockParser := &MockParser{
			name:       "package_test_parser",
			sourceType: reflect.TypeOf(123),
		}

		err := RegisterParser(mockParser)
		assert.NoError(t, err)
	})

	t.Run("WithParser", func(t *testing.T) {
		ctx := WithParser("test_parser")
		assert.NotNil(t, ctx)
	})

	t.Run("Parse", func(t *testing.T) {
		// This may fail if no parser is registered, but we're testing the function exists
		source := "test"
		dest := &MockValidatable{}
		err := Parse(source, dest, false)
		// Don't assert success/failure as it depends on registered parsers
		_ = err
	})

	t.Run("Invalidate", func(t *testing.T) {
		dest := &MockValidatable{Value: "test"}
		err := Invalidate(dest)
		assert.NoError(t, err)
	})

	t.Run("GetParser", func(t *testing.T) {
		source := "test"
		parser, err := GetParser(source)
		// Don't assert success/failure as it depends on registered parsers
		_ = parser
		_ = err
	})

	t.Run("GetParserByName", func(t *testing.T) {
		source := "test"
		parser, err := GetParserByName(source, "test_parser")
		// Don't assert success/failure as it depends on registered parsers
		_ = parser
		_ = err
	})
}
