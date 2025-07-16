package pave

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test tag parsing functionality
func TestDecodeParseTagV2(t *testing.T) {
	t.Run("BasicSuccess", func(t *testing.T) {
		type TestStruct struct {
			Field1 string `json:"field1" default:"default_value"`
		}

		field := reflect.TypeOf(TestStruct{}).Field(0)
		opts := ParseTagOpts{
			BindingOpts: BindingOpts{
				AllowedBindingNames: []string{"json"},
			},
			AllowedTagOptionals: []string{},
		}

		tag, err := DecodeParseTagV2(field, opts)
		require.NoError(t, err)
		assert.Equal(t, "default_value", tag.defaultTag.Value)
		assert.False(t, tag.recursiveTag.Enabled)
	})

	t.Run("RecursiveTag", func(t *testing.T) {
		type NestedStruct struct {
			Value string
		}

		type TestStruct struct {
			Field1 NestedStruct `json:"field1" recursive:"true"`
		}

		field := reflect.TypeOf(TestStruct{}).Field(0)
		opts := ParseTagOpts{
			BindingOpts: BindingOpts{
				AllowedBindingNames: []string{"json"},
			},
			AllowedTagOptionals: []string{},
		}

		tag, err := DecodeParseTagV2(field, opts)
		require.NoError(t, err)
		assert.True(t, tag.recursiveTag.Enabled)
	})

	t.Run("NoTags", func(t *testing.T) {
		type TestStruct struct {
			Field1 string
		}

		field := reflect.TypeOf(TestStruct{}).Field(0)
		opts := ParseTagOpts{
			BindingOpts: BindingOpts{
				AllowedBindingNames: []string{"json"},
			},
			AllowedTagOptionals: []string{},
		}

		tag, err := DecodeParseTagV2(field, opts)
		require.NoError(t, err)
		assert.Equal(t, "", tag.defaultTag.Value)
		assert.False(t, tag.recursiveTag.Enabled)
	})
}

func TestDecodeBindingTagV2(t *testing.T) {
	t.Run("BasicBinding", func(t *testing.T) {
		opts := BindingOpts{
			AllowedBindingNames: []string{"json"},
		}

		tag, err := decodeBindingTagV2("json", "field1", opts)
		require.NoError(t, err)
		assert.Equal(t, "field1", tag.Identifier)
		assert.Equal(t, "json", tag.Name)
		assert.Empty(t, tag.Modifiers)
	})

	t.Run("BindingWithModifiers", func(t *testing.T) {
		opts := BindingOpts{
			AllowedBindingNames: []string{"json"},
		}

		tag, err := decodeBindingTagV2("json", "field1,omitempty,omitnil", opts)
		require.NoError(t, err)
		assert.Equal(t, "field1", tag.Identifier)
		assert.Equal(t, "json", tag.Name)
		assert.Contains(t, tag.Modifiers, "omitempty")
		assert.Contains(t, tag.Modifiers, "omitnil")
	})

	t.Run("EmptyValue", func(t *testing.T) {
		opts := BindingOpts{
			AllowedBindingNames: []string{"json"},
		}

		_, err := decodeBindingTagV2("json", "", opts)
		assert.Error(t, err)
	})
}

func TestDecodeDefaultTagV2(t *testing.T) {
	t.Run("WithDefaultTag", func(t *testing.T) {
		type TestStruct struct {
			Field1 string `default:"test_default"`
		}

		field := reflect.TypeOf(TestStruct{}).Field(0)
		tag, err := decodeDefaultTagV2(field)
		require.NoError(t, err)
		assert.Equal(t, "test_default", tag.Value)
	})

	t.Run("WithoutDefaultTag", func(t *testing.T) {
		type TestStruct struct {
			Field1 string `json:"field1"`
		}

		field := reflect.TypeOf(TestStruct{}).Field(0)
		tag, err := decodeDefaultTagV2(field)
		require.NoError(t, err)
		assert.Equal(t, "", tag.Value)
	})
}

func TestDecodeRecursiveTagV2(t *testing.T) {
	t.Run("StructTypeWithoutRecursiveTag", func(t *testing.T) {
		type NestedStruct struct {
			Value string
		}

		type TestStruct struct {
			Field1 NestedStruct
		}

		field := reflect.TypeOf(TestStruct{}).Field(0)
		tag, err := decodeRecursiveTagV2(field)
		require.NoError(t, err)
		// Should default to true for struct types without explicit recursive tag
		assert.True(t, tag.Enabled)
	})

	t.Run("NonStructType", func(t *testing.T) {
		type TestStruct struct {
			Field1 string
		}

		field := reflect.TypeOf(TestStruct{}).Field(0)
		tag, err := decodeRecursiveTagV2(field)
		require.NoError(t, err)
		assert.False(t, tag.Enabled)
	})
}

func TestMakeBindings(t *testing.T) {
	t.Run("BasicBinding", func(t *testing.T) {
		parseTag := ParseTag{
			bindingTags: []BindingTag{
				{
					Name:       "json",
					Identifier: "field1",
					Modifiers:  []string{"omitempty"},
				},
			},
		}

		opts := ParseTagOpts{
			BindingOpts: BindingOpts{
				CustomBindingModifiers: []string{},
			},
		}

		bindings, err := makeBindings(parseTag, opts)
		require.NoError(t, err)
		assert.Len(t, bindings, 1)
		assert.Equal(t, "json", bindings[0].Name)
		assert.Equal(t, "field1", bindings[0].Identifier)
		assert.False(t, bindings[0].Modifiers.Required) // omitempty makes it not required
	})

	t.Run("RequiredBinding", func(t *testing.T) {
		parseTag := ParseTag{
			bindingTags: []BindingTag{
				{
					Name:       "json",
					Identifier: "field1",
					Modifiers:  []string{}, // No modifiers = required
				},
			},
		}

		opts := ParseTagOpts{
			BindingOpts: BindingOpts{
				CustomBindingModifiers: []string{},
			},
		}

		bindings, err := makeBindings(parseTag, opts)
		require.NoError(t, err)
		assert.Len(t, bindings, 1)
		assert.True(t, bindings[0].Modifiers.Required)
	})
}

func TestBindingTag_toBinding(t *testing.T) {
	t.Run("OmitEmptyModifier", func(t *testing.T) {
		tag := BindingTag{
			Name:       "json",
			Identifier: "field1",
			Modifiers:  []string{OmitEmptyBindingModifier},
		}

		binding, err := tag.toBinding([]string{})
		require.NoError(t, err)
		assert.Equal(t, "json", binding.Name)
		assert.Equal(t, "field1", binding.Identifier)
		assert.False(t, binding.Modifiers.Required)
	})

	t.Run("OmitNilModifier", func(t *testing.T) {
		tag := BindingTag{
			Name:       "json",
			Identifier: "field1",
			Modifiers:  []string{OmitNilBindingModifier},
		}

		binding, err := tag.toBinding([]string{})
		require.NoError(t, err)
		assert.False(t, binding.Modifiers.Required)
	})

	t.Run("NoModifiers", func(t *testing.T) {
		tag := BindingTag{
			Name:       "json",
			Identifier: "field1",
			Modifiers:  []string{},
		}

		binding, err := tag.toBinding([]string{})
		require.NoError(t, err)
		assert.True(t, binding.Modifiers.Required)
	})
}
