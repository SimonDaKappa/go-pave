package pave

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// JsonSourceParser demonstrates parsing from []byte containing JSON data
// This shows how multiple parsers can work with the same source type
type JsonSourceParser struct {
	chains     map[reflect.Type]*ParseExecutionChain
	chainMutex sync.RWMutex
}

func NewJsonSourceParser() *JsonSourceParser {
	return &JsonSourceParser{
		chains: make(map[reflect.Type]*ParseExecutionChain),
	}
}

func (jsp *JsonSourceParser) GetSourceType() reflect.Type {
	return reflect.TypeOf([]byte{})
}

func (jsp *JsonSourceParser) GetParserName() string {
	return "json"
}

func (jsp *JsonSourceParser) Parse(source any, dest Validatable) error {
	data, ok := source.([]byte)
	if !ok {
		return fmt.Errorf("expected []byte, got %T", source)
	}

	// Parse JSON into a map for processing
	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Get the struct type
	destType := reflect.TypeOf(dest)
	if destType.Kind() == reflect.Ptr {
		destType = destType.Elem()
	}

	// Get or build the execution chain
	chain, err := jsp.GetParseChain(destType)
	if err != nil {
		return err
	}

	// Execute the chain with JSON-specific source getter
	return chain.Execute(jsonData, dest)
}

func (jsp *JsonSourceParser) GetParseChain(t reflect.Type) (*ParseExecutionChain, error) {
	jsp.chainMutex.RLock()
	if chain, exists := jsp.chains[t]; exists {
		jsp.chainMutex.RUnlock()
		return chain, nil
	}
	jsp.chainMutex.RUnlock()

	return jsp.BuildParseChain(t)
}

func (jsp *JsonSourceParser) BuildParseChain(t reflect.Type) (*ParseExecutionChain, error) {
	chain, err := jsp.buildChainForType(t)
	if err != nil {
		return nil, err
	}

	jsp.chainMutex.Lock()
	jsp.chains[t] = chain
	jsp.chainMutex.Unlock()

	return chain, nil
}

func (jsp *JsonSourceParser) buildChainForType(t reflect.Type) (*ParseExecutionChain, error) {
	chain := &ParseExecutionChain{
		StructType:   t,
		SourceGetter: jsp.getValueFromJSON,
	}

	var head *ParseStep
	var current *ParseStep

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		sources := jsp.parseFieldSources(field)
		if len(sources) == 0 {
			continue
		}

		step := &ParseStep{
			FieldIndex: i,
			FieldName:  field.Name,
			Sources:    sources,
		}

		if head == nil {
			head = step
			current = step
		} else {
			current.Next = step
			current = step
		}
	}

	chain.Head = head
	return chain, nil
}

func (jsp *JsonSourceParser) parseFieldSources(field reflect.StructField) []FieldSource {
	var sources []FieldSource

	// Only look for JSON tags for this parser
	if tagValue := field.Tag.Get(JSONSourceTag); tagValue != "" && tagValue != "-" {
		source := jsp.parseSourceTag(tagValue)
		source.Source = JSONSourceTag
		sources = append(sources, source)
	}

	return sources
}

func (jsp *JsonSourceParser) parseSourceTag(tag string) FieldSource {
	parts := strings.Split(tag, ",")
	source := FieldSource{
		Key:      strings.TrimSpace(parts[0]),
		Required: true, // Default to required
	}

	for _, part := range parts[1:] {
		switch strings.TrimSpace(part) {
		case "omitempty":
			source.OmitEmpty = true
			source.Required = false
		case "required":
			source.Required = true
		}
	}

	return source
}

// getValueFromJSON implements the SourceGetter function for JSON data
func (jsp *JsonSourceParser) getValueFromJSON(sourceData any, source FieldSource) (any, bool, error) {
	jsonData, ok := sourceData.(map[string]interface{})
	if !ok {
		return nil, false, fmt.Errorf("expected map[string]interface{}, got %T", sourceData)
	}

	switch source.Source {
	case JSONSourceTag:
		value, exists := jsonData[source.Key]
		if !exists {
			return nil, false, nil
		}
		return value, true, nil
	default:
		return nil, false, fmt.Errorf("unsupported source type: %s", source.Source)
	}
}
