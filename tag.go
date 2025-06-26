package pave

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
)

// This file contains the tag parser for the pave pacckage. It is responsible for parsing the tags
// associated with fields in structs that are used for parsing and validation. The tag parser
// interprets the tags and generates the appropriate parsing and validation logic based on the
// specified tags. It supports all tags in the following grammar:
//
// Tag grammar:
//     <field> <type> <tag>
// field:
//     <string>
// type:
//     <Type>
// tag:
//     '<tag_parse> <tag_validate>'
//
// tag_parse:
//     parse:"<tag_default> <tag_binding_list>"
//
// tag_default:
//     default:'<default_value>'
// default_value:
//     <Go Literal>
//
// tag_binding_list:
//     [<tag_binding>]^*
// tag_binding:
//     <binding_name>:'<binding_identifier>,<binding_modifier_list>' // binding tags are parser specific but must follow this grammar
//
// binding_name, binding_identifier:
//     <string>
// binding_modifier_list:
//     [binding_modifier]^* // Delimeted with "," end-delim optional
// binding_modifier:
//     omitempty | omiterr | omitnil | ... // (any other modifiers past this point are parser specific)
//
// tag_validate
//     validate:"<...>" | nil

/*
Grammar Notes:
- source_tags are parser-extension specific, they must be parseable by the base grammar parser
  something like:
    type sourcetagfunc func(tags ...string) ([]FieldSource, error)
- how to handle fields that are structs?
    1. Require explicit delineation between recursive parsing and non-recursive parsing
    2. Use a tag to indicate that the field should be parsed recursively, e.g
       `parse:"recursive"` or `parse:"nonrecursive"`
    3. Recursive parsing happens by default, non-recursive parsing is indicated by the tag
*/

// TagParser is a struct that holds the logic for parsing tags
// across the pave package. It enforces a strict grammar for the tags
// and provides the necessary abstraction to easily implement
// custom tag values within the grammar.

type TagParse struct {
	DefaultTag  TagDefault
	BindingTags []TagBinding
}

type TagDefault struct {
	Value string
}

type TagBinding struct {
	Name       string
	Identifier string
	Modifiers  []string
}

type ParseTagOpts struct {
	BindingOpts
}

// GetBindings parses the tag string and returns a structured representation of the tag.
func GetBindings(field reflect.StructField, opts ParseTagOpts) ([]FieldBinding, string, error) {
	parseSubTag, ok := field.Tag.Lookup("parse")
	if !ok {
		return nil, "", fmt.Errorf("field %s does not have a parse tag", field.Name)
	}

	decoded, err := decodeParseTag(parseSubTag, opts)
	if err != nil {
		return nil, "", fmt.Errorf("error parsing parse tag for field %s: %w",
			field.Name, err)
	}

	bindings, err := makeBindings(decoded, opts)
	if err != nil {
		return nil, "", fmt.Errorf("error making field sources for field %s: %w",
			field.Name, err)
	}

	return bindings, decoded.DefaultTag.Value, nil
}

func decodeParseTag(tag string, opts ParseTagOpts) (TagParse, error) {

	defaultTag, err := decodeTagDefault(tag)
	if err != nil {
		return TagParse{}, fmt.Errorf("unable to parse 'default' subtag: %v", err)
	}

	bindingStrs := strings.Fields(tag)
	bindingTags := make([]TagBinding, 0, len(bindingStrs))

	for _, bindingStr := range bindingStrs {
		if strings.HasPrefix(bindingStr, DefaultValueSubTagPrefixWithKVDelimiter) {
			continue
		}
		bindingTag, err := decodeTagBindings(bindingStr, opts)
		if err != nil {
			return TagParse{}, err
		}
		bindingTags = append(bindingTags, bindingTag)
	}

	return TagParse{
		BindingTags: bindingTags,
		DefaultTag:  defaultTag,
	}, nil
}

func decodeTagDefault(ptag string) (TagDefault, error) {
	value, err := SubTag(ptag, DefaultValueSubTagPrefix)
	return TagDefault{value}, err
}

func decodeTagBindings(stag string, opts ParseTagOpts) (TagBinding, error) {
	// Split the tag into its components
	parts := strings.Split(stag, ":")
	if len(parts) != 2 {
		return TagBinding{}, fmt.Errorf("invalid source tag format: %s", stag)
	}

	sourceName := parts[0]
	if !slices.Contains(opts.AllowedBindingModifiers, sourceName) {
		return TagBinding{}, fmt.Errorf("source name %s is not allowed", sourceName)
	}

	sourceInfo := parts[1]

	// Further split the sourceInfo into identifier and modifiers
	sourceParts := strings.Split(sourceInfo, ",")
	if len(sourceParts) == 0 {
		return TagBinding{}, fmt.Errorf("invalid source info format: %s", sourceInfo)
	}

	sourceIdentifier := sourceParts[0]
	sourceModifiers := sourceParts[1:]
	for _, modifier := range sourceModifiers {
		if !slices.Contains(opts.AllowedBindingNames, modifier) {
			return TagBinding{}, fmt.Errorf("source modifier %s is not allowed", modifier)
		}
	}

	return TagBinding{
		Name:       sourceName,
		Identifier: sourceIdentifier,
		Modifiers:  sourceModifiers,
	}, nil
}

func makeBindings(ptag TagParse, opts ParseTagOpts) ([]FieldBinding, error) {
	bindings := make([]FieldBinding, 0, len(ptag.BindingTags))

	for _, btag := range ptag.BindingTags {
		binding, err := btag.toFieldBinding(opts.AllowedBindingModifiers)
		if err != nil {
			return nil, fmt.Errorf("error creating field binding from tag %s: %w", btag.Name, err)
		}
		bindings = append(bindings, binding)
	}

	return bindings, nil
}

func (t TagBinding) toFieldBinding(allowedModifiers []string) (FieldBinding, error) {

	modifiers := FieldBindingModifiers{}
	for _, modifier := range t.Modifiers {
		switch modifier {
		case OmitEmptyBindingModifier:
			modifiers.OmitEmpty = true
		case OmitErrBindingModifier:
			modifiers.OmitError = true
		case OmitNilBindingModifier:
			modifiers.OmitNil = true
		case RequiredBindingModifier:
			modifiers.Required = true
		default:
			if !slices.Contains(allowedModifiers, modifier) {
				return FieldBinding{}, fmt.Errorf("unrecognized/unallowed binding modifier: %s", modifier)
			} else {
				modifiers.Custom[modifier] = true
			}
		}
	}

	return FieldBinding{
		Name:       t.Name,
		Identifier: t.Identifier,
		Modifiers:  modifiers,
	}, nil
}

func SubTags(tag string, excludes ...string) (map[string]string, error) {
	return SubTagsByDelimiter(tag, DefaultSubTagScopeDelimiter, excludes...)
}

func SubTagsByDelimiter(tag string, delim byte, excludes ...string) (map[string]string, error) {
	result := make(map[string]string)

	// Create a set of excluded keys for fast lookup
	excludeSet := make(map[string]bool)
	for _, exclude := range excludes {
		excludeSet[exclude] = true
	}

	i := 0
	for i < len(tag) {
		// Skip whitespace
		for i < len(tag) && (tag[i] == ' ' || tag[i] == '\t') {
			i++
		}
		if i >= len(tag) {
			break
		}

		// Find the next key:value pair
		colonIdx := strings.Index(tag[i:], DefaultKeyValueTagDelimiter)
		if colonIdx == -1 {
			break // No more key:value pairs
		}

		// Adjust colonIdx to be relative to the full tag string
		colonIdx += i

		// Extract the key
		key := strings.TrimSpace(tag[i:colonIdx])
		if key == "" {
			i = colonIdx + 1
			continue
		}

		// Skip this key if it's in the exclude list
		if excludeSet[key] {
			// Skip over its value to continue parsing
			valueStart := colonIdx + 1
			for valueStart < len(tag) && (tag[valueStart] == ' ' || tag[valueStart] == '\t') {
				valueStart++
			}

			if valueStart < len(tag) && tag[valueStart] == delim {
				// It's a delimited value, find the end
				valueStart++ // skip opening delimiter
				nestingLevel := 0
				escaped := false
				for j := valueStart; j < len(tag); j++ {
					c := tag[j]
					if c == '\\' && !escaped {
						escaped = true
						continue
					}
					if !escaped {
						if c == ':' && j+1 < len(tag) && tag[j+1] == delim {
							nestingLevel++
							j++ // Skip the delimiter that starts the nested value
						} else if c == delim {
							if nestingLevel == 0 {
								i = j + 1
								break
							}
							nestingLevel--
						}
					}
					escaped = false
				}
			} else {
				// It's a simple value, find the next space
				for valueStart < len(tag) && tag[valueStart] != ' ' && tag[valueStart] != '\t' {
					valueStart++
				}
				i = valueStart
			}
			continue
		}

		// Extract the value for this key
		value, err := SubTagByDelimeter(tag, key, delim)
		if err != nil {
			// If it's not a delimited value, try to extract as a simple value
			valueStart := colonIdx + 1

			if valueStart >= len(tag) {
				break
			}
			valueEnd := valueStart
			for valueEnd < len(tag) && tag[valueEnd] != ' ' && tag[valueEnd] != '\t' {
				valueEnd++
			}
			value = tag[valueStart:valueEnd]
			i = valueEnd
		} else {
			// Find where this delimited value ends to continue parsing
			valueStart := colonIdx + 1
			for valueStart < len(tag) && (tag[valueStart] == ' ' || tag[valueStart] == '\t') {
				valueStart++
			}
			if valueStart < len(tag) && tag[valueStart] == delim {
				// Find the end of the delimited value
				valueStart++ // skip opening delimiter
				nestingLevel := 0
				escaped := false
				for j := valueStart; j < len(tag); j++ {
					c := tag[j]
					if c == '\\' && !escaped {
						escaped = true
						continue
					}
					if !escaped {
						if c == ':' && j+1 < len(tag) && tag[j+1] == delim {
							nestingLevel++
							j++ // Skip the delimiter that starts the nested value
						} else if c == delim {
							if nestingLevel == 0 {
								i = j + 1
								break
							}
							nestingLevel--
						}
					}
					escaped = false
					if j == len(tag)-1 {
						// Reached end of string
						i = len(tag)
						break
					}
				}
			} else {
				// Simple value
				for valueStart < len(tag) && tag[valueStart] != ' ' && tag[valueStart] != '\t' {
					valueStart++
				}
				i = valueStart
			}
		}

		result[key] = value
	}

	return result, nil
}

func SubTag(tag string, key string) (string, error) {
	return SubTagByDelimeter(tag, key, DefaultSubTagScopeDelimiter)
}

// Example: tag = `parse:"default:5 foo:'bar,omitnil'" validate:"min=1,max=10"`
// tag.Lookup("parse")  should return ptag = "default:5 foo:bar,omitnil"
// SubTagByDelimeter(ptag, "foo", '\”) should return  'bar,omitnil
// SubTagByDelimeter(ptag, "validate", '\”) should return "min=1,max=10"
//
// Nested example: parse:"a:'b:'c:'d”'"
// SubTag(tag, "a") should return "b:'c:'d”"
func SubTagByDelimeter(tag string, key string, delim byte) (string, error) {

	search := key + ":"
	idx := strings.Index(tag, search)
	if idx == -1 {
		return "", fmt.Errorf("subtag %q not found", key)
	}

	// Find the start of the value (after the colon and any whitespace)
	start := idx + len(search)
	for start < len(tag) && (tag[start] == ' ' || tag[start] == '\t') {
		start++
	}

	// Check if the value starts with a delimiter
	if start >= len(tag) {
		return "", fmt.Errorf("no value found after subtag %q", key)
	}

	// If the value doesn't start with our delimiter, it's a simple value
	if tag[start] != delim {
		// Find the end of the simple value (next space or end of string)
		end := start
		for end < len(tag) && tag[end] != ' ' && tag[end] != '\t' {
			end++
		}
		return tag[start:end], nil
	}

	start++ // skip opening delimiter

	// Find the closing delimiter, handling escaped delimiters and nested subtags
	var value strings.Builder
	escaped := false
	nestingLevel := 0 // Track how deep we are in nested subtags

	for i := start; i < len(tag); i++ {
		c := tag[i]

		if c == '\\' && !escaped {
			escaped = true
			value.WriteByte(c)
			continue
		}

		if !escaped {
			switch c {
			case ':':
				// Check if the next character is our delimiter, indicating a nested subtag
				if i+1 < len(tag) && tag[i+1] == delim {
					nestingLevel++
					// Add the colon to output and skip the next delimiter (it's part of the nesting)
					value.WriteByte(c)
					i++ // Skip the next delimiter
					value.WriteByte(tag[i])
					escaped = false
					continue
				}
			case delim:
				if nestingLevel == 0 {
					// This is our closing delimiter
					return value.String(), nil
				} else {
					// This closes a nested subtag - add the delimiter to output before reducing nesting
					value.WriteByte(c)
					nestingLevel--
					escaped = false
					continue
				}
			}
		}

		value.WriteByte(c)
		escaped = false
	}

	return "", fmt.Errorf("unterminated subtag value for %q", key)
}
