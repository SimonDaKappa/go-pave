package pave

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test ParseChain functionality
func TestParseChain_SingleStepExecute(t *testing.T) {
	t.Run("NilHead", func(t *testing.T) {
		chain := &ParseChain[string]{
			StructType: reflect.TypeOf(struct{}{}),
			Head:       nil,
			Handler: func(source *string, binding Binding) BindingResult {
				return BindingResultValue("test")
			},
		}

		source := "test"
		dest := &struct{}{}

		err := chain.Execute(&source, dest)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "parse chain is empty")
	})

	t.Run("SuccessfulExecution", func(t *testing.T) {
		type TestStruct struct {
			Field1 string
		}

		// Create a simple parse step
		step := &ParseStep[string]{
			Next: nil,
			Bindings: []Binding{
				{
					Name:       "test",
					Identifier: "field1",
					Modifiers:  BindingModifiers{Required: true},
				},
			},
			FieldName:  "Field1",
			FieldIndex: 0,
			IsStruct:   false,
		}

		chain := &ParseChain[string]{
			StructType: reflect.TypeOf(TestStruct{}),
			Head:       step,
			Handler: func(source *string, binding Binding) BindingResult {
				return BindingResultValue("test_value")
			},
		}

		source := "test"
		dest := &TestStruct{}

		err := chain.Execute(&source, dest)
		require.NoError(t, err)
		assert.Equal(t, "test_value", dest.Field1)
	})

	t.Run("FailedBinding", func(t *testing.T) {
		type TestStruct struct {
			Field1 string
		}

		// Create a simple parse step
		step := &ParseStep[string]{
			Next: nil,
			Bindings: []Binding{
				{
					Name:       "test",
					Identifier: "field1",
					Modifiers:  BindingModifiers{Required: true},
				},
			},
			FieldName:  "Field1",
			FieldIndex: 0,
			IsStruct:   false,
		}

		chain := &ParseChain[string]{
			StructType: reflect.TypeOf(TestStruct{}),
			Head:       step,
			Handler: func(source *string, binding Binding) BindingResult {
				return BindingResultNotFound()
			},
		}

		source := "test"
		dest := &TestStruct{}

		err := chain.Execute(&source, dest)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse field")
	})
}

func TestParseChain_doStepRecursive(t *testing.T) {
	// This test is merged with the main execution test above
	// since doStepRecursive is tested through the Execute method
}

func TestPCManager(t *testing.T) {
	handler := func(source *string, binding Binding) BindingResult {
		return BindingResultValue("test")
	}

	t.Run("NewPCManager", func(t *testing.T) {
		opts := PCManagerOpts{
			tagOpts: ParseTagOpts{
				BindingOpts: BindingOpts{
					AllowedBindingNames: []string{"json"},
				},
			},
		}

		pcm := NewPCManager(handler, opts)
		assert.NotNil(t, pcm)
	})

	t.Run("GetParseChain", func(t *testing.T) {
		type TestStruct struct {
			Field1 string `json:"field1"`
		}

		opts := PCManagerOpts{
			tagOpts: ParseTagOpts{
				BindingOpts: BindingOpts{
					AllowedBindingNames: []string{"json"},
				},
			},
		}

		pcm := NewPCManager(handler, opts)

		chain, err := pcm.GetParseChain(reflect.TypeOf(TestStruct{}))
		require.NoError(t, err)
		assert.NotNil(t, chain)
		assert.Equal(t, reflect.TypeOf(TestStruct{}), chain.StructType)
	})

	t.Run("GetParseChain_Cached", func(t *testing.T) {
		type TestStruct struct {
			Field1 string `json:"field1"`
		}

		opts := PCManagerOpts{
			tagOpts: ParseTagOpts{
				BindingOpts: BindingOpts{
					AllowedBindingNames: []string{"json"},
				},
			},
		}

		pcm := NewPCManager(handler, opts)

		// First call should create and cache
		chain1, err := pcm.GetParseChain(reflect.TypeOf(TestStruct{}))
		require.NoError(t, err)

		// Second call should return cached version
		chain2, err := pcm.GetParseChain(reflect.TypeOf(TestStruct{}))
		require.NoError(t, err)

		// Should be the same instance
		assert.Equal(t, chain1, chain2)
	})

	t.Run("NewParseChain", func(t *testing.T) {
		type TestStruct struct {
			Field1 string `json:"field1"`
		}

		opts := PCManagerOpts{
			tagOpts: ParseTagOpts{
				BindingOpts: BindingOpts{
					AllowedBindingNames: []string{"json"},
				},
			},
		}

		pcm := NewPCManager(handler, opts)

		chain, err := pcm.NewParseChain(reflect.TypeOf(TestStruct{}))
		require.NoError(t, err)
		assert.NotNil(t, chain)
		assert.NotNil(t, chain.Head)
		assert.Equal(t, "Field1", chain.Head.FieldName)
	})

	t.Run("NewParseStep", func(t *testing.T) {
		type TestStruct struct {
			Field1 string `json:"field1"`
		}

		opts := PCManagerOpts{
			tagOpts: ParseTagOpts{
				BindingOpts: BindingOpts{
					AllowedBindingNames: []string{"json"},
				},
			},
		}

		pcm := NewPCManager(handler, opts)

		field := reflect.TypeOf(TestStruct{}).Field(0)
		step, err := pcm.NewParseStep(field, 0)
		require.NoError(t, err)
		assert.NotNil(t, step)
		assert.Equal(t, "Field1", step.FieldName)
		assert.Equal(t, 0, step.FieldIndex)
	})
}

func TestParseChain_doStepRegular(t *testing.T) {
	t.Run("SuccessfulBinding", func(t *testing.T) {
		type TestStruct struct {
			Field1 string
		}

		step := &ParseStep[string]{
			Bindings: []Binding{
				{
					Name:       "test",
					Identifier: "field1",
					Modifiers:  BindingModifiers{Required: true},
				},
			},
			FieldName:  "Field1",
			FieldIndex: 0,
			IsStruct:   false,
		}

		chain := &ParseChain[string]{
			StructType: reflect.TypeOf(TestStruct{}),
			Handler: func(source *string, binding Binding) BindingResult {
				return BindingResultValue("test_value")
			},
		}

		source := "test"
		dest := &TestStruct{}
		destValue := reflect.ValueOf(dest).Elem()
		field := destValue.Field(0)

		err := chain.doStepRegular(&source, field, step)
		require.NoError(t, err)
		assert.Equal(t, "test_value", dest.Field1)
	})

	t.Run("FailedBinding_WithDefault", func(t *testing.T) {
		type TestStruct struct {
			Field1 string
		}

		step := &ParseStep[string]{
			Bindings: []Binding{
				{
					Name:       "test",
					Identifier: "field1",
					Modifiers:  BindingModifiers{Required: false, OmitEmpty: true},
				},
			},
			FieldName:    "Field1",
			FieldIndex:   0,
			IsStruct:     false,
			DefaultValue: "default_value",
		}

		chain := &ParseChain[string]{
			StructType: reflect.TypeOf(TestStruct{}),
			Handler: func(source *string, binding Binding) BindingResult {
				return BindingResultNotFound()
			},
		}

		source := "test"
		dest := &TestStruct{}
		destValue := reflect.ValueOf(dest).Elem()
		field := destValue.Field(0)

		err := chain.doStepRegular(&source, field, step)
		require.NoError(t, err)
		assert.Equal(t, "default_value", dest.Field1)
	})

	t.Run("FailedBinding_NoDefault_Required", func(t *testing.T) {
		type TestStruct struct {
			Field1 string
		}

		step := &ParseStep[string]{
			Bindings: []Binding{
				{
					Name:       "test",
					Identifier: "field1",
					Modifiers:  BindingModifiers{Required: true},
				},
			},
			FieldName:  "Field1",
			FieldIndex: 0,
			IsStruct:   false,
		}

		chain := &ParseChain[string]{
			StructType: reflect.TypeOf(TestStruct{}),
			Handler: func(source *string, binding Binding) BindingResult {
				return BindingResultNotFound()
			},
		}

		source := "test"
		dest := &TestStruct{}
		destValue := reflect.ValueOf(dest).Elem()
		field := destValue.Field(0)

		err := chain.doStepRegular(&source, field, step)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required field field1 not found in source test")
	})
}
