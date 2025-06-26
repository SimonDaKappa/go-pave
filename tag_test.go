package pave

import (
	"reflect"
	"testing"
)

func TestSubTagSimple(t *testing.T) {
	tag := "default:'5' foo:'bar,omitnil'\""

	result, err := SubTag(tag, "default")
	if err != nil {
		t.Fatalf("Expected no error, got :%v", err)
	}
	expected := "5"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}

	t.Logf("result: %s", result)

	result, err = SubTag(tag, "foo")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	expected = "bar,omitnil"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}

	// Test SubTags
	subtags, err := SubTags(tag)
	if err != nil {
		t.Fatalf("Expected no error for SubTags, got: %v", err)
	}
	if len(subtags) != 2 {
		t.Errorf("Expected 2 subtags, got %d", len(subtags))
	}
	if subtags["default"] != "5" {
		t.Errorf("Expected default subtag to be '5', got %q", subtags["default"])
	}
	if subtags["foo"] != "bar,omitnil" {
		t.Errorf("Expected foo subtag to be 'bar,omitnil', got %q", subtags["foo"])
	}
}

func TestSubTagNested(t *testing.T) {
	// Test the example from the comment: parse:"a:'b:'c:'d'''"
	tag := "a:'b:'c:'d'''"
	result, err := SubTag(tag, "a")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	expected := "b:'c:'d''"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}

	// Test SubTags
	subtags, err := SubTags(tag)
	if err != nil {
		t.Fatalf("Expected no error for SubTags, got: %v", err)
	}
	if len(subtags) != 1 {
		t.Errorf("Expected 1 subtag, got %d", len(subtags))
	}
	if subtags["a"] != "b:'c:'d''" {
		t.Errorf("Expected a subtag to be %q, got %q", "b:'c:'d''", subtags["a"])
	}
}

func TestSubTagNestedDeep(t *testing.T) {
	// Test deeper nesting: "outer:'middle:'inner:'value'''"
	tag := "outer:'middle:'inner:'value'''"
	result, err := SubTag(tag, "outer")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	expected := "middle:'inner:'value''"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}

	// Test SubTags
	subtags, err := SubTags(tag)
	if err != nil {
		t.Fatalf("Expected no error for SubTags, got: %v", err)
	}
	if len(subtags) != 1 {
		t.Errorf("Expected 1 subtag, got %d", len(subtags))
	}
	if subtags["outer"] != "middle:'inner:'value''" {
		t.Errorf("Expected outer subtag to be %q, got %q", "middle:'inner:'value''", subtags["outer"])
	}
}

func TestSubTagNestedWithMultipleTags(t *testing.T) {
	// Test nested with multiple tags: "default:'0' nested:'inner:'value'' other:'simple'"
	tag := "default:'0' nested:'inner:'value'' other:'simple'"

	// Test the nested tag
	result, err := SubTag(tag, "nested")
	if err != nil {
		t.Fatalf("Expected no error for nested, got: %v", err)
	}
	expected := "inner:'value'"
	if result != expected {
		t.Errorf("Expected %q for nested, got %q", expected, result)
	}

	// Test the simple tag
	result, err = SubTag(tag, "other")
	if err != nil {
		t.Fatalf("Expected no error for other, got: %v", err)
	}
	expected = "simple"
	if result != expected {
		t.Errorf("Expected %q for other, got %q", expected, result)
	}

	// Test SubTags
	subtags, err := SubTags(tag)
	if err != nil {
		t.Fatalf("Expected no error for SubTags, got: %v", err)
	}
	if len(subtags) != 3 {
		t.Errorf("Expected 3 subtags, got %d", len(subtags))
	}
	if subtags["default"] != "0" {
		t.Errorf("Expected default subtag to be '0', got %q", subtags["default"])
	}
	if subtags["nested"] != "inner:'value'" {
		t.Errorf("Expected nested subtag to be %q, got %q", "inner:'value'", subtags["nested"])
	}
	if subtags["other"] != "simple" {
		t.Errorf("Expected other subtag to be 'simple', got %q", subtags["other"])
	}

	// Test SubTags with excludes
	subtags, err = SubTags(tag, "default")
	if err != nil {
		t.Fatalf("Expected no error for SubTags with excludes, got: %v", err)
	}
	if len(subtags) != 2 {
		t.Errorf("Expected 2 subtags after excluding default, got %d", len(subtags))
	}
	if _, exists := subtags["default"]; exists {
		t.Error("Expected default subtag to be excluded")
	}
	if subtags["nested"] != "inner:'value'" {
		t.Errorf("Expected nested subtag to be %q, got %q", "inner:'value'", subtags["nested"])
	}
	if subtags["other"] != "simple" {
		t.Errorf("Expected other subtag to be 'simple', got %q", subtags["other"])
	}
}

func TestSubTagEscapedDelimiters(t *testing.T) {
	// Test escaped delimiters: "key:'value\'with\'escaped\'quotes'"
	tag := "key:'value\\'with\\'escaped\\'quotes'"
	result, err := SubTag(tag, "key")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	expected := "value\\'with\\'escaped\\'quotes"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}

	// Test SubTags
	subtags, err := SubTags(tag)
	if err != nil {
		t.Fatalf("Expected no error for SubTags, got: %v", err)
	}
	if len(subtags) != 1 {
		t.Errorf("Expected 1 subtag, got %d", len(subtags))
	}
	if subtags["key"] != "value\\'with\\'escaped\\'quotes" {
		t.Errorf("Expected key subtag to be %q, got %q", "value\\'with\\'escaped\\'quotes", subtags["key"])
	}
}

func TestSubTagNestedAndRegularValuePairsInList(t *testing.T) {
	tag := "key1:'key2:'key3:'valueA',valueB',valueC'"
	key1subtag, err := SubTag(tag, "key1")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	expected := "key2:'key3:'valueA',valueB',valueC"
	if key1subtag != expected {
		t.Errorf("Expected %q, got %q", expected, key1subtag)
	}
	// Test SubTags
	subtags, err := SubTags(tag)
	if err != nil {
		t.Fatalf("Expected no error for SubTags, got: %v", err)
	}
	if len(subtags) != 1 {
		t.Errorf("Expected 1 subtag, got %d", len(subtags))
	}
	if subtags["key1"] != "key2:'key3:'valueA',valueB',valueC" {
		t.Errorf("Expected key1 subtag to be %q, got %q", "key2:'key3:'valueA',valueB',valueC", subtags["key1"])
	}

	// Test nested subtag
	key2subtag, err := SubTag(key1subtag, "key2")
	if err != nil {
		t.Fatalf("Expected no error for nested subtag, got: %v", err)
	}
	expected = "key3:'valueA',valueB"
	if key2subtag != expected {
		t.Errorf("Expected %q, got %q", expected, key2subtag)
	}

	// Test nested subtag with key3
	key3subtag, err := SubTag(key2subtag, "key3")
	if err != nil {
		t.Fatalf("Expected no error for nested subtag, got: %v", err)
	}
	expected = "valueA"
	if key3subtag != expected {
		t.Errorf("Expected %q, got %q", expected, key3subtag)
	}
}

func TestSubTagNotFound(t *testing.T) {
	tag := "foo:'bar' baz:'qux'"
	_, err := SubTag(tag, "missing")
	if err == nil {
		t.Fatal("Expected error for missing subtag, got nil")
	}
}

func TestSubTagMalformed(t *testing.T) {
	// Test unterminated subtag
	tag := "key:'unterminated"
	_, err := SubTag(tag, "key")
	if err == nil {
		t.Fatal("Expected error for unterminated subtag, got nil")
	}
}

func TestTagGrammarSimple(t *testing.T) {
	// Simple grammar: single source binding with no modifiers
	type SimpleGrammarImpl struct {
		Name string `parse:"default:'John' json:'name'"`
	}

	opts := ParseTagOpts{
		BindingOpts: BindingOpts{
			AllowedBindingNames:     []string{"json", "http", "form"},
			AllowedBindingModifiers: []string{"omitempty", "omiterr", "omitnil", "required"},
		},
	}

	field := reflect.TypeOf(SimpleGrammarImpl{}).Field(0)
	bindings, defaultValue, err := GetBindings(field, opts)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if defaultValue != "John" {
		t.Errorf("Expected default value to be \"John\", got: \"%s\"", defaultValue)
	}

	if len(bindings) != 1 {
		t.Fatalf("Expected 1 binding, got: %d", len(bindings))
	}

	binding := bindings[0]
	if binding.Name != "json" {
		t.Errorf("Expected binding name to be \"json\", got: \"%s\"", binding.Name)
	}
	if binding.Identifier != "name" {
		t.Errorf("Expected binding identifier to be \"name\", got: \"%s\"", binding.Identifier)
	}
	if len(binding.Modifiers.Custom) != 0 {
		t.Errorf("Expected no custom modifiers, got: \"%v\"", binding.Modifiers.Custom)
	}
}

func TestTagGrammarMedium(t *testing.T) {
	// Medium grammar: multiple source bindings with modifiers
	type MediumGrammarImpl struct {
		UserID int `parse:"default:'0' json:user_id,omitempty http:X-User-ID,required"`
	}

	opts := ParseTagOpts{
		BindingOpts: BindingOpts{
			AllowedBindingNames:     []string{"json", "http", "form"},
			AllowedBindingModifiers: []string{"omitempty", "omiterr", "omitnil", "required"},
		},
	}

	field := reflect.TypeOf(MediumGrammarImpl{}).Field(0)
	bindings, defaultValue, err := GetBindings(field, opts)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if defaultValue != "0" {
		t.Errorf("Expected default value to be \"0\", got: \"%s\"", defaultValue)
	}

	if len(bindings) != 2 {
		t.Fatalf("Expected 2 bindings, got: %d", len(bindings))
	}

	// Check first binding (json)
	jsonBinding := bindings[0]
	if jsonBinding.Name != "json" {
		t.Errorf("Expected first binding name to be \"json\", got: \"%s\"", jsonBinding.Name)
	}
	if jsonBinding.Identifier != "user_id" {
		t.Errorf("Expected first binding identifier to be \"user_id\", got: \"%s\"", jsonBinding.Identifier)
	}
	if !jsonBinding.Modifiers.OmitEmpty {
		t.Error("Expected first binding to have OmitEmpty modifier")
	}

	// Check second binding (http)
	httpBinding := bindings[1]
	if httpBinding.Name != "http" {
		t.Errorf("Expected second binding name to be \"http\", got: \"%s\"", httpBinding.Name)
	}
	if httpBinding.Identifier != "X-User-ID" {
		t.Errorf("Expected second binding identifier to be \"X-User-ID\", got: \"%s\"", httpBinding.Identifier)
	}
	if !httpBinding.Modifiers.Required {
		t.Error("Expected second binding to have Required modifier")
	}
}

func TestTagGrammarComplex(t *testing.T) {
	// Complex grammar: multiple source bindings with multiple modifiers each
	type ComplexGrammarImpl struct {
		Email string `parse:"default:'user@example.com' json:email,omitempty,omiterr form:user_email,required,omitnil http:X-User-Email,omiterr"`
	}
	
	opts := ParseTagOpts{
		BindingOpts: BindingOpts{
			AllowedBindingNames:     []string{"json", "http", "form"},
			AllowedBindingModifiers: []string{"omitempty", "omiterr", "omitnil", "required"},
		},
	}

	field := reflect.TypeOf(ComplexGrammarImpl{}).Field(0)
	bindings, defaultValue, err := GetBindings(field, opts)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if defaultValue != "user@example.com" {
		t.Errorf("Expected default value to be \"user@example.com\", got: \"%s\"", defaultValue)
	}

	if len(bindings) != 3 {
		t.Fatalf("Expected 3 bindings, got: %d", len(bindings))
	}

	// Check first binding (json)
	jsonBinding := bindings[0]
	if jsonBinding.Name != "json" {
		t.Errorf("Expected first binding name to be \"json\", got: \"%s\"", jsonBinding.Name)
	}
	if jsonBinding.Identifier != "email" {
		t.Errorf("Expected first binding identifier to be \"email\", got: \"%s\"", jsonBinding.Identifier)
	}
	if !jsonBinding.Modifiers.OmitEmpty {
		t.Error("Expected first binding to have OmitEmpty modifier")
	}
	if !jsonBinding.Modifiers.OmitError {
		t.Error("Expected first binding to have OmitError modifier")
	}

	// Check second binding (form)
	formBinding := bindings[1]
	if formBinding.Name != "form" {
		t.Errorf("Expected second binding name to be \"form\", got: \"%s\"", formBinding.Name)
	}
	if formBinding.Identifier != "user_email" {
		t.Errorf("Expected second binding identifier to be \"user_email\", got: \"%s\"", formBinding.Identifier)
	}
	if !formBinding.Modifiers.Required {
		t.Error("Expected second binding to have Required modifier")
	}
	if !formBinding.Modifiers.OmitNil {
		t.Error("Expected second binding to have OmitNil modifier")
	}

	// Check third binding (http)
	httpBinding := bindings[2]
	if httpBinding.Name != "http" {
		t.Errorf("Expected third binding name to be \"http\", got: \"%s\"", httpBinding.Name)
	}
	if httpBinding.Identifier != "X-User-Email" {
		t.Errorf("Expected third binding identifier to be \"X-User-Email\", got: \"%s\"", httpBinding.Identifier)
	}
	if !httpBinding.Modifiers.OmitError {
		t.Error("Expected third binding to have OmitError modifier")
	}
}
