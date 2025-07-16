package pave

import (
	"encoding"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Custom type that implements TextUnmarshaler
type CustomTextType struct {
	Value string
}

func (c *CustomTextType) UnmarshalText(text []byte) error {
	if string(text) == "error" {
		return errors.New("custom error")
	}
	c.Value = "custom:" + string(text)
	return nil
}

// Custom type that implements TextUnmarshaler on pointer
type CustomPointerType struct {
	Value string
}

func (c *CustomPointerType) UnmarshalText(text []byte) error {
	if string(text) == "error" {
		return errors.New("custom pointer error")
	}
	c.Value = "pointer:" + string(text)
	return nil
}

// Helper function to create a reflect.Value from any type
func valueFromInterface(v interface{}) reflect.Value {
	return reflect.ValueOf(v).Elem()
}

// Test for setFieldValue main function
func TestSetFieldValue(t *testing.T) {
	tests := []struct {
		name    string
		field   interface{}
		value   string
		want    interface{}
		wantErr bool
	}{
		// String tests
		{"string_basic", ptr(""), "hello", "hello", false},
		{"string_empty", ptr(""), "", "", false},

		// Integer tests
		{"int_basic", ptr(int(0)), "42", int(42), false},
		{"int8_basic", ptr(int8(0)), "127", int8(127), false},
		{"int16_basic", ptr(int16(0)), "32767", int16(32767), false},
		{"int32_basic", ptr(int32(0)), "2147483647", int32(2147483647), false},
		{"int64_basic", ptr(int64(0)), "9223372036854775807", int64(9223372036854775807), false},
		{"int_overflow", ptr(int8(0)), "128", int8(0), true},
		{"int_invalid", ptr(int(0)), "abc", int(0), true},

		// Unsigned integer tests
		{"uint_basic", ptr(uint(0)), "42", uint(42), false},
		{"uint8_basic", ptr(uint8(0)), "255", uint8(255), false},
		{"uint16_basic", ptr(uint16(0)), "65535", uint16(65535), false},
		{"uint32_basic", ptr(uint32(0)), "4294967295", uint32(4294967295), false},
		{"uint64_basic", ptr(uint64(0)), "18446744073709551615", uint64(18446744073709551615), false},
		{"uint_overflow", ptr(uint8(0)), "256", uint8(0), true},
		{"uint_invalid", ptr(uint(0)), "abc", uint(0), true},

		// Float tests
		{"float32_basic", ptr(float32(0)), "3.14", float32(3.14), false},
		{"float64_basic", ptr(float64(0)), "3.14159265359", float64(3.14159265359), false},
		{"float_overflow", ptr(float32(0)), "3.4028235e+39", float32(0), true},
		{"float_invalid", ptr(float64(0)), "abc", float64(0), true},

		// Complex tests
		{"complex64_basic", ptr(complex64(0)), "1+2i", complex64(1 + 2i), false},
		{"complex128_basic", ptr(complex128(0)), "1.5+2.5i", complex128(1.5 + 2.5i), false},
		{"complex_invalid", ptr(complex64(0)), "abc", complex64(0), true},

		// Boolean tests
		{"bool_true", ptr(false), "true", true, false},
		{"bool_false", ptr(true), "false", false, false},
		{"bool_yes", ptr(false), "yes", true, false},
		{"bool_no", ptr(true), "no", false, false},
		{"bool_on", ptr(false), "on", true, false},
		{"bool_off", ptr(true), "off", false, false},
		{"bool_1", ptr(false), "1", true, false},
		{"bool_0", ptr(true), "0", false, false},
		{"bool_case_insensitive", ptr(false), "TRUE", true, false},
		{"bool_invalid", ptr(false), "maybe", false, true},

		// Slice tests
		{"slice_bytes", ptr([]byte{}), "hello", []byte("hello"), false},

		// UUID tests (struct)
		{"uuid_valid", ptr(uuid.UUID{}), "550e8400-e29b-41d4-a716-446655440000", uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), false},
		{"uuid_invalid", ptr(uuid.UUID{}), "invalid-uuid", uuid.UUID{}, true},

		// Time tests
		{"time_rfc3339", ptr(time.Time{}), "2023-01-01T00:00:00Z", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"time_invalid_rfc3339", ptr(time.Time{}), "2023-01-01", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), true},
		{"time_invalid", ptr(time.Time{}), "invalid-time", time.Time{}, true},

		// Interface tests
		{"interface_empty", ptr(interface{}(nil)), "hello", "hello", false},

		// TextUnmarshaler tests
		{"custom_text_unmarshaler", ptr(CustomTextType{}), "test", CustomTextType{Value: "custom:test"}, false},
		{"custom_text_error", ptr(CustomTextType{}), "error", CustomTextType{}, true},
		{"custom_pointer_unmarshaler", ptr(CustomPointerType{}), "test", CustomPointerType{Value: "pointer:test"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := valueFromInterface(tt.field)
			err := setFieldValue(field, tt.value)

			if (err != nil) != tt.wantErr {
				t.Errorf("setFieldValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				got := field.Interface()
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("setFieldValue() got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

// Test for handleEmptyValue function
func TestHandleEmptyValue(t *testing.T) {
	tests := []struct {
		name    string
		field   interface{}
		want    interface{}
		wantErr bool
	}{
		{"string_empty", ptr("test"), "", false},
		{"slice_empty", ptr([]byte{1, 2, 3}), []byte(nil), false},
		{"map_empty", ptr(map[string]int{"a": 1}), map[string]int(nil), false},
		{"ptr_empty", ptr(&struct{}{}), (*struct{})(nil), false},
		{"interface_empty", ptr(interface{}("test")), interface{}(nil), false},
		{"int_empty", ptr(int(42)), int(0), true}, // Should error
		{"bool_empty", ptr(true), false, true},    // Should error
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := valueFromInterface(tt.field)
			err := handleEmptyValue(field)

			if (err != nil) != tt.wantErr {
				t.Errorf("handleEmptyValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				got := field.Interface()
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("handleEmptyValue() got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

// Test for individual setter functions
func TestSetStringValue(t *testing.T) {
	var s string
	field := reflect.ValueOf(&s).Elem()

	err := setStringValue(field, "hello")
	if err != nil {
		t.Errorf("setStringValue() error = %v", err)
	}
	if s != "hello" {
		t.Errorf("setStringValue() got = %v, want %v", s, "hello")
	}
}

func TestSetIntValue(t *testing.T) {
	tests := []struct {
		name    string
		field   interface{}
		value   string
		want    interface{}
		wantErr bool
	}{
		{"int_valid", ptr(int(0)), "42", int(42), false},
		{"int8_valid", ptr(int8(0)), "127", int8(127), false},
		{"int8_overflow", ptr(int8(0)), "128", int8(0), true},
		{"int_invalid", ptr(int(0)), "abc", int(0), true},
		{"int_negative", ptr(int(0)), "-42", int(-42), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := valueFromInterface(tt.field)
			err := setIntValue(field, tt.value)

			if (err != nil) != tt.wantErr {
				t.Errorf("setIntValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				got := field.Interface()
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("setIntValue() got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestSetUintValue(t *testing.T) {
	tests := []struct {
		name    string
		field   interface{}
		value   string
		want    interface{}
		wantErr bool
	}{
		{"uint_valid", ptr(uint(0)), "42", uint(42), false},
		{"uint8_valid", ptr(uint8(0)), "255", uint8(255), false},
		{"uint8_overflow", ptr(uint8(0)), "256", uint8(0), true},
		{"uint_invalid", ptr(uint(0)), "abc", uint(0), true},
		{"uint_negative", ptr(uint(0)), "-1", uint(0), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := valueFromInterface(tt.field)
			err := setUintValue(field, tt.value)

			if (err != nil) != tt.wantErr {
				t.Errorf("setUintValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				got := field.Interface()
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("setUintValue() got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestSetFloatValue(t *testing.T) {
	tests := []struct {
		name    string
		field   interface{}
		value   string
		want    interface{}
		wantErr bool
	}{
		{"float32_valid", ptr(float32(0)), "3.14", float32(3.14), false},
		{"float64_valid", ptr(float64(0)), "3.14159265359", float64(3.14159265359), false},
		{"float32_overflow", ptr(float32(0)), "3.4028235e+39", float32(0), true},
		{"float_invalid", ptr(float64(0)), "abc", float64(0), true},
		{"float_negative", ptr(float64(0)), "-3.14", float64(-3.14), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := valueFromInterface(tt.field)
			err := setFloatValue(field, tt.value)

			if (err != nil) != tt.wantErr {
				t.Errorf("setFloatValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				got := field.Interface()
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("setFloatValue() got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestSetComplexValue(t *testing.T) {
	tests := []struct {
		name    string
		field   interface{}
		value   string
		want    interface{}
		wantErr bool
	}{
		{"complex64_valid", ptr(complex64(0)), "1+2i", complex64(1 + 2i), false},
		{"complex128_valid", ptr(complex128(0)), "1.5+2.5i", complex128(1.5 + 2.5i), false},
		{"complex_real_only", ptr(complex64(0)), "42", complex64(42), false},
		{"complex_invalid", ptr(complex64(0)), "abc", complex64(0), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := valueFromInterface(tt.field)
			err := setComplexValue(field, tt.value)

			if (err != nil) != tt.wantErr {
				t.Errorf("setComplexValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				got := field.Interface()
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("setComplexValue() got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestSetBoolValue(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    bool
		wantErr bool
	}{
		// True values
		{"true_lowercase", "true", true, false},
		{"true_uppercase", "TRUE", true, false},
		{"true_mixed", "True", true, false},
		{"one", "1", true, false},
		{"yes_lowercase", "yes", true, false},
		{"yes_uppercase", "YES", true, false},
		{"on_lowercase", "on", true, false},
		{"on_uppercase", "ON", true, false},

		// False values
		{"false_lowercase", "false", false, false},
		{"false_uppercase", "FALSE", false, false},
		{"false_mixed", "False", false, false},
		{"zero", "0", false, false},
		{"no_lowercase", "no", false, false},
		{"no_uppercase", "NO", false, false},
		{"off_lowercase", "off", false, false},
		{"off_uppercase", "OFF", false, false},

		// strconv.ParseBool fallback
		{"t", "t", true, false},
		{"f", "f", false, false},

		// Invalid values
		{"invalid", "maybe", false, true},
		{"empty", "", false, true},
		{"random", "random", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b bool
			field := reflect.ValueOf(&b).Elem()
			err := setBoolValue(field, tt.value)

			if (err != nil) != tt.wantErr {
				t.Errorf("setBoolValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && b != tt.want {
				t.Errorf("setBoolValue() got = %v, want %v", b, tt.want)
			}
		})
	}
}

func TestSetSliceValue(t *testing.T) {
	tests := []struct {
		name    string
		field   interface{}
		value   string
		want    interface{}
		wantErr bool
	}{
		{"byte_slice", ptr([]byte{}), "hello", []byte("hello"), false},
		{"string_slice", ptr([]string{}), "hello", []string{}, true}, // Should error
		{"int_slice", ptr([]int{}), "hello", []int{}, true},          // Should error
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := valueFromInterface(tt.field)
			err := setSliceValue(field, tt.value)

			if (err != nil) != tt.wantErr {
				t.Errorf("setSliceValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				got := field.Interface()
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("setSliceValue() got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestSetArrayValue(t *testing.T) {
	tests := []struct {
		name    string
		field   interface{}
		value   string
		want    interface{}
		wantErr bool
	}{
		{"uuid_valid", ptr(uuid.UUID{}), "550e8400-e29b-41d4-a716-446655440000", uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), false},
		{"uuid_invalid", ptr(uuid.UUID{}), "invalid-uuid", uuid.UUID{}, true},
		{"int_array", ptr([3]int{}), "123", [3]int{}, true}, // Should error
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := valueFromInterface(tt.field)
			err := setArrayValue(field, tt.value)

			if (err != nil) != tt.wantErr {
				t.Errorf("setArrayValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				got := field.Interface()
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("setArrayValue() got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestSetStructValue(t *testing.T) {
	tests := []struct {
		name    string
		field   interface{}
		value   string
		want    interface{}
		wantErr bool
	}{
		// UUID tests
		{"uuid_valid", ptr(uuid.UUID{}), "550e8400-e29b-41d4-a716-446655440000", uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), false},
		{"uuid_invalid", ptr(uuid.UUID{}), "invalid-uuid", uuid.UUID{}, true},

		// Time tests
		{"time_rfc3339", ptr(time.Time{}), "2023-01-01T00:00:00Z", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"time_rfc3339_nano", ptr(time.Time{}), "2023-01-01T00:00:00.123456789Z", time.Date(2023, 1, 1, 0, 0, 0, 123456789, time.UTC), false},
		{"time_no_timezone", ptr(time.Time{}), "2023-01-01T00:00:00", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"time_with_space", ptr(time.Time{}), "2023-01-01 00:00:00", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"time_date_only", ptr(time.Time{}), "2023-01-01", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"time_time_only", ptr(time.Time{}), "15:04:05", time.Date(0, 1, 1, 15, 4, 5, 0, time.UTC), false},
		{"time_invalid", ptr(time.Time{}), "invalid-time", time.Time{}, true},

		// Unsupported struct
		{"custom_struct", ptr(struct{ Name string }{}), "test", struct{ Name string }{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := valueFromInterface(tt.field)
			err := setStructValue(field, tt.value)

			if (err != nil) != tt.wantErr {
				t.Errorf("setStructValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				got := field.Interface()
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("setStructValue() got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestSetInterfaceValue(t *testing.T) {
	tests := []struct {
		name    string
		field   interface{}
		value   string
		want    interface{}
		wantErr bool
	}{
		{"empty_interface", ptr(interface{}(nil)), "hello", "hello", false},
		{"interface_with_methods", ptr(encoding.TextUnmarshaler(nil)), "hello", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := valueFromInterface(tt.field)
			err := setInterfaceValue(field, tt.value)

			if (err != nil) != tt.wantErr {
				t.Errorf("setInterfaceValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				got := field.Interface()
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("setInterfaceValue() got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

// Test for zeroStructFields function
func TestZeroStructFields(t *testing.T) {
	type NestedStruct struct {
		NestedField int
	}

	type TestStruct struct {
		StringField string
		IntField    int
		BoolField   bool
		FloatField  float64
		SliceField  []int
		MapField    map[string]int
		PtrField    *int
		UUIDField   uuid.UUID
		TimeField   time.Time
		NestedField NestedStruct
		unexported  int // Should be skipped
	}

	// Create a struct with non-zero values
	original := TestStruct{
		StringField: "hello",
		IntField:    42,
		BoolField:   true,
		FloatField:  3.14,
		SliceField:  []int{1, 2, 3},
		MapField:    map[string]int{"key": 1},
		PtrField:    &[]int{123}[0],
		UUIDField:   uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		TimeField:   time.Now(),
		NestedField: NestedStruct{NestedField: 99},
		unexported:  999,
	}

	// Zero the struct
	value := reflect.ValueOf(&original).Elem()
	zeroStructFields(value)

	// Check that all fields are zeroed
	expected := TestStruct{
		unexported: 999, // Should remain unchanged
	}

	if !reflect.DeepEqual(original, expected) {
		t.Errorf("zeroStructFields() failed to zero all fields correctly")
		t.Errorf("Got: %+v", original)
		t.Errorf("Expected: %+v", expected)
	}
}

func TestZeroStructFields_NonStruct(t *testing.T) {
	// Test with non-struct value
	var i int = 42
	value := reflect.ValueOf(&i).Elem()
	zeroStructFields(value) // Should not panic and should not change value

	if i != 42 {
		t.Errorf("zeroStructFields() should not affect non-struct values, got %d", i)
	}
}

func TestZeroStructFields_UnexportedFields(t *testing.T) {
	type TestStruct struct {
		ExportedField   int
		unexportedField int
	}

	original := TestStruct{
		ExportedField:   42,
		unexportedField: 99,
	}

	value := reflect.ValueOf(&original).Elem()
	zeroStructFields(value)

	// Exported field should be zeroed, unexported should remain
	if original.ExportedField != 0 {
		t.Errorf("zeroStructFields() should zero exported field, got %d", original.ExportedField)
	}
	if original.unexportedField != 99 {
		t.Errorf("zeroStructFields() should not affect unexported field, got %d", original.unexportedField)
	}
}

// Test for isSpecialStructType function
func TestIsSpecialStructType(t *testing.T) {
	tests := []struct {
		name string
		t    reflect.Type
		want bool
	}{
		{"time.Time", reflect.TypeOf(time.Time{}), true},
		{"uuid.UUID", reflect.TypeOf(uuid.UUID{}), true},
		{"regular_struct", reflect.TypeOf(struct{ Name string }{}), false},
		{"string", reflect.TypeOf(""), false},
		{"int", reflect.TypeOf(int(0)), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSpecialStructType(tt.t); got != tt.want {
				t.Errorf("isSpecialStructType() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test for ParseTypeErasedPointer function
func TestParseTypeErasedPointer(t *testing.T) {
	// Test successful case
	t.Run("success", func(t *testing.T) {
		type TestSource struct {
			Value string
		}

		type TestDest struct {
			Result string
		}

		source := &TestSource{Value: "test"}
		dest := &TestDest{}

		parseFunc := func(src *TestSource, dst any) error {
			d := dst.(*TestDest)
			d.Result = "parsed:" + src.Value
			return nil
		}

		err := ParseTypeErasedPointer(source, dest, parseFunc)
		if err != nil {
			t.Errorf("ParseTypeErasedPointer() error = %v", err)
		}

		if dest.Result != "parsed:test" {
			t.Errorf("ParseTypeErasedPointer() result = %v, want %v", dest.Result, "parsed:test")
		}
	})

	// Test wrong source type
	t.Run("wrong_source_type", func(t *testing.T) {
		type TestSource struct {
			Value string
		}

		type TestDest struct {
			Result string
		}

		source := "wrong type"
		dest := &TestDest{}

		parseFunc := func(src *TestSource, dst any) error {
			return nil
		}

		err := ParseTypeErasedPointer(source, dest, parseFunc)
		if err == nil {
			t.Error("ParseTypeErasedPointer() should error with wrong source type")
		}
	})

	// Test wrong destination type
	t.Run("wrong_dest_type", func(t *testing.T) {
		type TestSource struct {
			Value string
		}

		source := &TestSource{Value: "test"}
		dest := "wrong type"

		parseFunc := func(src *TestSource, dst any) error {
			return nil
		}

		err := ParseTypeErasedPointer(source, dest, parseFunc)
		if err == nil {
			t.Error("ParseTypeErasedPointer() should error with wrong destination type")
		}
	})

	// Test parse function error
	t.Run("parse_error", func(t *testing.T) {
		type TestSource struct {
			Value string
		}

		type TestDest struct {
			Result string
		}

		source := &TestSource{Value: "test"}
		dest := &TestDest{}

		parseFunc := func(src *TestSource, dst any) error {
			return errors.New("parse error")
		}

		err := ParseTypeErasedPointer(source, dest, parseFunc)
		if err == nil {
			t.Error("ParseTypeErasedPointer() should propagate parse function error")
		}
		if err.Error() != "parse error" {
			t.Errorf("ParseTypeErasedPointer() error = %v, want %v", err, "parse error")
		}
	})
}

// Test for ParseTypeErasedSlice function
func TestParseTypeErasedSlice(t *testing.T) {
	// Test successful case
	t.Run("success", func(t *testing.T) {
		type TestSource struct {
			Value string
		}

		type TestDest struct {
			Results []string
		}

		source := []TestSource{{Value: "test1"}, {Value: "test2"}}
		dest := &TestDest{}

		parseFunc := func(src []TestSource, dst any) error {
			d := dst.(*TestDest)
			for _, s := range src {
				d.Results = append(d.Results, "parsed:"+s.Value)
			}
			return nil
		}

		err := ParseTypeErasedSlice(source, dest, parseFunc)
		if err != nil {
			t.Errorf("ParseTypeErasedSlice() error = %v", err)
		}

		expected := []string{"parsed:test1", "parsed:test2"}
		if !reflect.DeepEqual(dest.Results, expected) {
			t.Errorf("ParseTypeErasedSlice() result = %v, want %v", dest.Results, expected)
		}
	})

	// Test wrong source type
	t.Run("wrong_source_type", func(t *testing.T) {
		type TestSource struct {
			Value string
		}

		type TestDest struct {
			Results []string
		}

		source := "wrong type"
		dest := &TestDest{}

		parseFunc := func(src []TestSource, dst any) error {
			return nil
		}

		err := ParseTypeErasedSlice(source, dest, parseFunc)
		if err == nil {
			t.Error("ParseTypeErasedSlice() should error with wrong source type")
		}
	})

	// Test wrong destination type
	t.Run("wrong_dest_type", func(t *testing.T) {
		type TestSource struct {
			Value string
		}

		source := []TestSource{{Value: "test"}}
		dest := "wrong type"

		parseFunc := func(src []TestSource, dst any) error {
			return nil
		}

		err := ParseTypeErasedSlice(source, dest, parseFunc)
		if err == nil {
			t.Error("ParseTypeErasedSlice() should error with wrong destination type")
		}
	})

	// Test parse function error
	t.Run("parse_error", func(t *testing.T) {
		type TestSource struct {
			Value string
		}

		type TestDest struct {
			Results []string
		}

		source := []TestSource{{Value: "test"}}
		dest := &TestDest{}

		parseFunc := func(src []TestSource, dst any) error {
			return errors.New("parse error")
		}

		err := ParseTypeErasedSlice(source, dest, parseFunc)
		if err == nil {
			t.Error("ParseTypeErasedSlice() should propagate parse function error")
		}
		if err.Error() != "parse error" {
			t.Errorf("ParseTypeErasedSlice() error = %v, want %v", err, "parse error")
		}
	})
}

// Helper function to get pointer to value
func ptr[T any](v T) *T {
	return &v
}
