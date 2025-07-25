package pave

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
)

// Base Error types for tag parsing errors
var (
	ErrNoParseTagInField        = errors.New("no parse tag found in field")
	ErrUnallowedBindingName     = errors.New("binding name is not allowed")
	ErrEmptyBindingIdentifier   = errors.New("binding identifier cannot be empty")
	ErrInvalidBindingTagFormat  = errors.New("invalid binding tag format")
	ErrInvalidBindingInfoFormat = errors.New("invalid binding info format")
	ErrUnallowedBindingModifier = errors.New("binding modifier is not allowed")
	ErrEmptyTagValue            = errors.New("tag value cannot be empty for non-string types")
)

// This file contains the tag parser for the pave package. It is responsible
// for parsing the tags associated with fields in structs that are used for
// parsing and validation. The tag parser interprets the tags and generates
// the appropriate parsing and validation logic based on the specified tags.
// It supports all tags in the following grammar:
//
// Tag grammar:
//     <field> <type> <tag>
// field:
//     <Go Literal>
// type:
//     <Go Literal>
// tag:
//     <parse_tag> <validate_tag>'
//
// parse_tag:
//     (binding_tag_list)? (optional_tag_list)? // Any ordering of elements is allowed.
//
// binding_tag_list:
//    [<binding_tag>]^*
//
// binding_tag:
//     <binding_name>:"<binding_identifier>,<binding_modifier_list>"
// binding_name, binding_identifier:
//     <string>
//
// binding_modifier_list:
//     [<binding_modifier>]^* // Delimited with "," end-delim optional
// binding_modifier:
//     omitempty | omiterror | omitnil | <modifier_custom>
// modifier_custom:
//    <parser_specific>
//
// optional_tag_list:
//    [<optional_tag>]^*
// optional_tag:
//     <default_tag> | <recursive_tag> | <custom_tag>
// custom_tag:
//     <parser_specific>
//
// default_tag:
//     default:"<string>"
//
// recursive_tag:
//     recursive:"<bool>"
//
// validate_tag
//     validate:"<...>" | nil

/*
Grammar Notes:
- how to handle fields that are structs?
    1. Require explicit delineation for non-recursive parsing
    2. Recursive parsing happens by default, non-recursive parsing is indicated
	   by the tag as `recursive:"false"`
*/

type ParseTagOpts struct {
	BindingOpts
	AllowedTagOptionals []string // List of allowed optional tags
}

type ParseTag struct {
	bindingTags  []BindingTag
	defaultTag   DefaultTag
	recursiveTag RecursiveTag
	customTags   []CustomTag
}

// Corresponds to <default_tag>
// Example: default:"5"
type DefaultTag struct {
	Value string
}

// Corresponds to <recursive_tag>
type RecursiveTag struct {
	Enabled bool // If true, the field should be recursively parsed
}

type CustomTag struct {
	Name  string // The name of the optional tag
	Value string // The value of the optional tag
}

// Corresponds to <binding_tag>
// Example: form:"foo,omitnil"
type BindingTag struct {
	Name       string
	Identifier string
	Modifiers  []string
}

func DecodeParseTagV2(field reflect.StructField, opts ParseTagOpts) (ParseTag, error) {
	// Get binding list
	bindingTags, err := decodeBindingTagsV2(field, opts)
	if err != nil {
		return ParseTag{}, err
	}

	// Get optional tags
	customTags, err := decodeCustomTagsV2(field, opts)
	if err != nil {
		return ParseTag{}, err
	}

	// Get default tag
	defTag, err := decodeDefaultTagV2(field)
	if err != nil {
		return ParseTag{}, err
	}

	// Get recursive tag
	recTag, err := decodeRecursiveTagV2(field)
	if err != nil {
		return ParseTag{}, err
	}

	return ParseTag{
		customTags:   customTags,
		bindingTags:  bindingTags,
		defaultTag:   defTag,
		recursiveTag: recTag,
	}, nil
}

func decodeBindingTagsV2(field reflect.StructField, opts ParseTagOpts) ([]BindingTag, error) {
	var bindingTags []BindingTag

	for _, name := range opts.AllowedBindingNames {
		value, ok := field.Tag.Lookup(name)

		if ok {
			bindingTag, err := decodeBindingTagV2(name, value, opts.BindingOpts)
			if err != nil {
				return []BindingTag{}, fmt.Errorf("error getting binding tag %s for field %s: %w", name, field.Name, err)
			}
			bindingTags = append(bindingTags, bindingTag)
		}
	}

	return bindingTags, nil
}

func decodeBindingTagV2(key string, value string, opts BindingOpts) (BindingTag, error) {
	parts := strings.Split(value, CommaDelimeter)

	var (
		identifier string
		modifiers  []string
	)

	switch l := len(parts); l {
	case 0:
		return BindingTag{}, fmt.Errorf("%w: %s", ErrInvalidBindingInfoFormat, parts[1])
	default:
		identifier = parts[0]
		if len(identifier) == 0 {
			return BindingTag{}, fmt.Errorf("%w in tag: %s:\"%s\"", ErrEmptyBindingIdentifier, key, value)
		}

		if l > 1 {
			modifiers = parts[1:]
		}

	}

	for _, modifier := range modifiers {
		switch modifier {
		case OmitEmptyBindingModifier, OmitErrorBindingModifier, OmitNilBindingModifier:
			// These are standard modifiers, no action needed
			continue
		default:
			if !slices.Contains(opts.CustomBindingModifiers, modifier) {
				return BindingTag{}, fmt.Errorf("%w: %s", ErrUnallowedBindingModifier, modifier)
			}
		}
	}

	return BindingTag{
		Name:       key,
		Identifier: identifier,
		Modifiers:  modifiers,
	}, nil

}

func decodeCustomTagsV2(field reflect.StructField, opts ParseTagOpts) ([]CustomTag, error) {
	var customTags []CustomTag
	for _, name := range opts.AllowedTagOptionals {
		if value, ok := field.Tag.Lookup(name); ok {
			customTags = append(customTags, CustomTag{Name: name, Value: value})
		}
	}

	return customTags, nil
}

func decodeDefaultTagV2(field reflect.StructField) (DefaultTag, error) {
	// Check if the field has a `default` tag
	if defaultTag, ok := field.Tag.Lookup("default"); ok {
		// Parse the default tag
		value := strings.TrimSpace(defaultTag)
		if value == "" && field.Type.Kind() != reflect.String {
			return DefaultTag{}, fmt.Errorf("default %w", ErrEmptyTagValue)
		}
		return DefaultTag{Value: value}, nil
	} else {
		// If no default tag is found, return an empty DefaultTag
		return DefaultTag{}, nil
	}
}

func decodeRecursiveTagV2(field reflect.StructField) (RecursiveTag, error) {
	var enabled bool

	// Check if the field has a `recursive` tag
	if field.Type.Kind() == reflect.Struct {
		if recursiveTag, ok := field.Tag.Lookup("recursive"); ok {
			// Parse the recursive tag
			enabled = strings.TrimSpace(recursiveTag) == "true"
		} else {
			enabled = true // Default to true if not specified
		}
	}

	return RecursiveTag{Enabled: enabled}, nil
}

// // GetBindings parses the tag string and returns a structured representation of the tag.
// func GetBindings(field reflect.StructField, opts ParseTagOpts) ([]Binding, string, error) {

// 	// Check if the field has a `parse` tag
// 	tag, ok := field.Tag.Lookup("parse")
// 	if !ok {
// 		return nil, "", fmt.Errorf("%w: %s", ErrNoParseTagInField, field.Name)
// 	}

// 	// Decode the parse tag into a structured representation
// 	parseTag, err := decodeParseTag(tag, opts)
// 	if err != nil {
// 		return nil, "", fmt.Errorf("error parsing parse tag for field %s: %w", field.Name, err)
// 	}

// 	// Generate bindings from the decoded parse tag
// 	bindings, err := makeBindings(parseTag, opts)
// 	if err != nil {
// 		return nil, "", fmt.Errorf("error making field sources for field %s: %w", field.Name, err)
// 	}

// 	return bindings, parseTag.DefaultTag.Value, nil
// }

// func decodeParseTag(tag string, opts ParseTagOpts) (ParseTag, error) {

// 	// Decode the default tag first
// 	defaultTag, err := decodeDefaultTag(tag)
// 	if err != nil {
// 		return ParseTag{}, fmt.Errorf("unable to parse 'default' subtag: %w", err)
// 	}

// 	bindingMap, err := SubTags(tag, DefaultValueSubTagPrefix)
// 	if err != nil {
// 		return ParseTag{}, fmt.Errorf("unable to parse binding tags: %w", err)
// 	}
// 	bindingTags := make([]BindingTag, 0, len(bindingMap))

// 	for key, value := range bindingMap {

// 		combined := strings.TrimSpace(
// 			key + DefaultKeyValueTagDelimiter + sDefaultSubTagScopeDelimiter + value + sDefaultSubTagScopeDelimiter,
// 		)

// 		bindingTag, err := decodeBindingTags(combined, opts)
// 		if err != nil {
// 			return ParseTag{}, err
// 		}

// 		bindingTags = append(bindingTags, bindingTag)
// 	}

// 	return ParseTag{
// 		BindingTags: bindingTags,
// 		DefaultTag:  defaultTag,
// 	}, nil
// }

// func decodeDefaultTag(ptag string) (DefaultTag, error) {
// 	value, err := SubTag(ptag, DefaultValueSubTagPrefix)
// 	if err != nil {
// 		if errors.Is(err, ErrSubTagNotFound) {
// 			// If the default tag is not found, return an empty DefaultTag
// 			return DefaultTag{}, nil
// 		} else {
// 			return DefaultTag{}, fmt.Errorf("error parsing default tag: %w", err)
// 		}
// 	}
// 	return DefaultTag{value}, err
// }

// func decodeBindingTags(stag string, opts ParseTagOpts) (BindingTag, error) {
// 	// Split the tag into its components
// 	parts := strings.Split(stag, DefaultKeyValueTagDelimiter)
// 	if len(parts) != 2 {
// 		return BindingTag{}, fmt.Errorf("%w: %s", ErrInvalidBindingTagFormat, stag)
// 	}

// 	// Extract binding name from the first part
// 	// Example: "form:'foo,omitnil'" -> "form" as binding name
// 	// and "foo,omitnil" as binding info
// 	bindingName := parts[0]
// 	if !slices.Contains(opts.AllowedBindingNames, bindingName) {
// 		return BindingTag{}, fmt.Errorf("%w: %s", ErrUnallowedBindingName, bindingName)
// 	}

// 	// Extract identifier and modifiers from the second part
// 	// Example: "'foo,omitnil'" -> "foo" as identifier and "omitnil" as modifier
// 	// If there are no modifiers, it will just be the identifier
// 	bindingInfo := strings.Split(parts[1], ",")

// 	var (
// 		bindingIdentifier string
// 		bindingModifiers  []string
// 	)

// 	switch infoLen := len(bindingInfo); infoLen {
// 	case 0:
// 		return BindingTag{}, fmt.Errorf("%w: %s", ErrInvalidBindingInfoFormat, parts[1])
// 	default:
// 		bindingIdentifier = strings.Trim(bindingInfo[0], sDefaultSubTagScopeDelimiter)
// 		if len(bindingIdentifier) == 0 {
// 			return BindingTag{}, fmt.Errorf("%w in tag: %s", ErrEmptyBindingIdentifier, stag)
// 		}

// 		if infoLen > 1 {
// 			bindingModifiers = bindingInfo[1:]
// 			if len(bindingModifiers) > 0 {
// 				// Trim any single quotes from the modifiers
// 				for i, modifier := range bindingModifiers {
// 					bindingModifiers[i] = strings.Trim(modifier, sDefaultSubTagScopeDelimiter)
// 				}
// 			}
// 		}

// 	}

// 	for _, modifier := range bindingModifiers {
// 		switch modifier {
// 		case OmitEmptyBindingModifier, OmitErrorBindingModifier, OmitNilBindingModifier:
// 			// These are standard modifiers, no action needed
// 			continue
// 		default:
// 			if !slices.Contains(opts.CustomBindingModifiers, modifier) {
// 				return BindingTag{}, fmt.Errorf("%w: %s", ErrUnallowedBindingModifier, modifier)
// 			}
// 		}
// 	}

// 	return BindingTag{
// 		Name:       bindingName,
// 		Identifier: bindingIdentifier,
// 		Modifiers:  bindingModifiers,
// 	}, nil
// }

func makeBindings(parseTag ParseTag, opts ParseTagOpts) ([]Binding, error) {
	bindings := make([]Binding, 0, len(parseTag.bindingTags))

	for _, bindingTag := range parseTag.bindingTags {

		binding, err := bindingTag.toBinding(opts.CustomBindingModifiers)
		if err != nil {
			return nil, fmt.Errorf("error creating field binding from tag %s: %w", bindingTag.Name, err)
		}
		bindings = append(bindings, binding)
	}

	return bindings, nil
}

func (t BindingTag) toBinding(customModifiers []string) (Binding, error) {

	modifiers := BindingModifiers{}
	omit := false
	for _, modifier := range t.Modifiers {
		switch modifier {
		case OmitEmptyBindingModifier:
			modifiers.OmitEmpty = true
			omit = true
		case OmitErrorBindingModifier:
			modifiers.OmitError = true
			omit = true
		case OmitNilBindingModifier:
			modifiers.OmitNil = true
			omit = true
		default:
			if !slices.Contains(customModifiers, modifier) {
				return Binding{}, fmt.Errorf("%w: %s", ErrUnallowedBindingModifier, modifier)
			} else {
				modifiers.Custom[modifier] = true
			}
		}
	}
	modifiers.Required = !omit

	return Binding{
		Name:       t.Name,
		Identifier: t.Identifier,
		Modifiers:  modifiers,
	}, nil
}

// func SubTags(tag string, excludes ...string) (map[string]string, error) {
// 	return SubTagsByDelimiter(tag, bDefaultSubTagScopeDelimiter, excludes...)
// }

// func SubTagsByDelimiter(tag string, delim byte, excludes ...string) (map[string]string, error) {
// 	result := make(map[string]string)

// 	// Create a set of excluded keys for fast lookup
// 	excludeSet := make(map[string]bool)
// 	for _, exclude := range excludes {
// 		excludeSet[exclude] = true
// 	}

// 	i := 0
// 	for i < len(tag) {
// 		// Skip whitespace
// 		for i < len(tag) && (tag[i] == ' ' || tag[i] == '\t') {
// 			i++
// 		}
// 		if i >= len(tag) {
// 			break
// 		}

// 		// Find the next key:value pair
// 		colonIdx := strings.Index(tag[i:], DefaultKeyValueTagDelimiter)
// 		if colonIdx == -1 {
// 			break // No more key:value pairs
// 		}

// 		// Adjust colonIdx to be relative to the full tag string
// 		colonIdx += i

// 		// Extract the key
// 		key := strings.TrimSpace(tag[i:colonIdx])
// 		if key == "" {
// 			i = colonIdx + 1
// 			continue
// 		}

// 		// Skip this key if it's in the exclude list
// 		if excludeSet[key] {
// 			// Skip over its value to continue parsing
// 			valueStart := colonIdx + 1
// 			for valueStart < len(tag) && (tag[valueStart] == ' ' || tag[valueStart] == '\t') {
// 				valueStart++
// 			}

// 			if valueStart < len(tag) && tag[valueStart] == delim {
// 				// It's a delimited value, find the end
// 				valueStart++ // skip opening delimiter
// 				nestingLevel := 0
// 				escaped := false
// 				for j := valueStart; j < len(tag); j++ {
// 					c := tag[j]
// 					if c == '\\' && !escaped {
// 						escaped = true
// 						continue
// 					}
// 					if !escaped {
// 						if c == ':' && j+1 < len(tag) && tag[j+1] == delim {
// 							nestingLevel++
// 							j++ // Skip the delimiter that starts the nested value
// 						} else if c == delim {
// 							if nestingLevel == 0 {
// 								i = j + 1
// 								break
// 							}
// 							nestingLevel--
// 						}
// 					}
// 					escaped = false
// 				}
// 			} else {
// 				// It's a simple value, find the next space
// 				for valueStart < len(tag) && tag[valueStart] != ' ' && tag[valueStart] != '\t' {
// 					valueStart++
// 				}
// 				i = valueStart
// 			}
// 			continue
// 		}

// 		// Extract the value for this key
// 		value, err := SubTagByDelimeter(tag, key, delim)
// 		if err != nil {
// 			// If it's not a delimited value, try to extract as a simple value
// 			valueStart := colonIdx + 1

// 			if valueStart >= len(tag) {
// 				break
// 			}
// 			valueEnd := valueStart
// 			for valueEnd < len(tag) && tag[valueEnd] != ' ' && tag[valueEnd] != '\t' {
// 				valueEnd++
// 			}
// 			value = tag[valueStart:valueEnd]
// 			i = valueEnd
// 		} else {
// 			// Find where this delimited value ends to continue parsing
// 			valueStart := colonIdx + 1
// 			for valueStart < len(tag) && (tag[valueStart] == ' ' || tag[valueStart] == '\t') {
// 				valueStart++
// 			}
// 			if valueStart < len(tag) && tag[valueStart] == delim {
// 				// Find the end of the delimited value
// 				valueStart++ // skip opening delimiter
// 				nestingLevel := 0
// 				escaped := false
// 				for j := valueStart; j < len(tag); j++ {
// 					c := tag[j]
// 					if c == '\\' && !escaped {
// 						escaped = true
// 						continue
// 					}
// 					if !escaped {
// 						if c == ':' && j+1 < len(tag) && tag[j+1] == delim {
// 							nestingLevel++
// 							j++ // Skip the delimiter that starts the nested value
// 						} else if c == delim {
// 							if nestingLevel == 0 {
// 								i = j + 1
// 								break
// 							}
// 							nestingLevel--
// 						}
// 					}
// 					escaped = false
// 					if j == len(tag)-1 {
// 						// Reached end of string
// 						i = len(tag)
// 						break
// 					}
// 				}
// 			} else {
// 				// Simple value
// 				for valueStart < len(tag) && tag[valueStart] != ' ' && tag[valueStart] != '\t' {
// 					valueStart++
// 				}
// 				i = valueStart
// 			}
// 		}

// 		result[key] = value
// 	}

// 	return result, nil
// }

// func SubTag(tag string, key string) (string, error) {
// 	return SubTagByDelimeter(tag, key, bDefaultSubTagScopeDelimiter)
// }

// var (
// 	ErrSubTagNotFound = fmt.Errorf("subtag not found")
// )

// // Example: tag = `parse:"default:5 foo:'bar,omitnil'" validate:"min=1,max=10"`
// //
// // tag.Lookup("parse")  should return ptag = "default:5 foo:bar,omitnil"
// //
// // SubTagByDelimeter(ptag, "foo", '\”) should return "bar,omitnil"
// //
// // SubTagByDelimeter(tag, "validate", '\"') should return "min=1,max=10" <- Identical to field.Tag.Lookup("validate")
// //
// // Nested example: parse:"a:'b:'c:'d”'"
// //
// // ptag := tag.Lookup("parse") // "a:'b:'c:'d'"
// //
// // SubTag(ptag, "a") should return "b:'c:'d”"
// func SubTagByDelimeter(tag string, key string, delim byte) (string, error) {

// 	search := key + ":"
// 	idx := strings.Index(tag, search)
// 	if idx == -1 {
// 		return "", ErrSubTagNotFound
// 	}

// 	// Find the start of the value (after the colon and any whitespace)
// 	start := idx + len(search)
// 	for start < len(tag) && (tag[start] == ' ' || tag[start] == '\t') {
// 		start++
// 	}

// 	// Check if the value starts with a delimiter
// 	if start >= len(tag) {
// 		return "", fmt.Errorf("no value found after subtag %q", key)
// 	}

// 	// If the value doesn't start with our delimiter, it's a simple value
// 	if tag[start] != delim {
// 		// Find the end of the simple value (next space or end of string)
// 		end := start
// 		for end < len(tag) && tag[end] != ' ' && tag[end] != '\t' {
// 			end++
// 		}
// 		return tag[start:end], nil
// 	}

// 	start++ // skip opening delimiter

// 	// Find the closing delimiter, handling escaped delimiters and nested subtags
// 	var builder strings.Builder
// 	escaped := false
// 	nestingLevel := 0 // Track how deep we are in nested subtags

// 	for i := start; i < len(tag); i++ {
// 		c := tag[i]

// 		if c == '\\' && !escaped {
// 			escaped = true
// 			builder.WriteByte(c)
// 			continue
// 		}

// 		if !escaped {
// 			switch c {
// 			case ':':
// 				// Check if the next character is our delimiter, indicating a nested subtag
// 				if i+1 < len(tag) && tag[i+1] == delim {
// 					nestingLevel++
// 					// Add the colon to output and skip the next delimiter (it's part of the nesting)
// 					builder.WriteByte(c)
// 					i++ // Skip the next delimiter
// 					builder.WriteByte(tag[i])
// 					escaped = false
// 					continue
// 				}
// 			case delim:
// 				if nestingLevel == 0 {
// 					// This is our closing delimiter
// 					return builder.String(), nil
// 				} else {
// 					// This closes a nested subtag - add the delimiter to output before reducing nesting
// 					builder.WriteByte(c)
// 					nestingLevel--
// 					escaped = false
// 					continue
// 				}
// 			}
// 		}

// 		builder.WriteByte(c)
// 		escaped = false
// 	}

// 	return "", fmt.Errorf("unterminated subtag value for %q", key)
// }

// func trimDelimiter(value string, delim byte) string {
// 	if len(value) > 0 && value[0] == delim {
// 		value = value[1:]
// 	}
// 	if len(value) > 0 && value[len(value)-1] == delim {
// 		value = value[:len(value)-1]
// 	}
// 	return value
// }
