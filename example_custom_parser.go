package pave

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// Example of how easy it is to create a custom parser with the new function-based approach

// MapSourceParser demonstrates parsing from a simple map[string]string source
type MapSourceParser struct {
	chains     map[reflect.Type]*BaseExecutionChain
	chainMutex sync.RWMutex
}

const (
	MapValueTag = "mapvalue" // Tag for specifying map keys
)

func NewMapSourceParser() *MapSourceParser {
	return &MapSourceParser{
		chains: make(map[reflect.Type]*BaseExecutionChain),
	}
}

func (msp *MapSourceParser) GetSourceType() reflect.Type {
	return reflect.TypeOf(map[string]string{})
}

func (msp *MapSourceParser) Parse(source any, dest Validatable) error {
	mapData, ok := source.(map[string]string)
	if !ok {
		return fmt.Errorf("expected map[string]string, got %T", source)
	}

	// Get the struct type
	destType := reflect.TypeOf(dest)
	if destType.Kind() == reflect.Ptr {
		destType = destType.Elem()
	}

	// Get or build the execution chain
	chain, err := msp.GetParseChain(destType)
	if err != nil {
		return err
	}

	// Execute the chain - notice how simple this is!
	return chain.Execute(mapData, dest)
}

func (msp *MapSourceParser) GetParseChain(t reflect.Type) (*BaseExecutionChain, error) {
	msp.chainMutex.RLock()
	if chain, exists := msp.chains[t]; exists {
		msp.chainMutex.RUnlock()
		return chain, nil
	}
	msp.chainMutex.RUnlock()

	return msp.BuildParseChain(t)
}

func (msp *MapSourceParser) BuildParseChain(t reflect.Type) (*BaseExecutionChain, error) {
	var head, current *ParseStep

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		sources := msp.parseFieldSources(field)
		if len(sources) == 0 {
			continue // Skip fields with no sources
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

	// Create the execution chain with our map-specific source getter
	// This is the ONLY function we need to implement that's specific to our source type!
	execChain := &BaseExecutionChain{
		StructType:   t,
		Head:         head,
		SourceGetter: msp.getValueFromMap, // <-- Our custom getter function
	}

	msp.chainMutex.Lock()
	msp.chains[t] = execChain
	msp.chainMutex.Unlock()

	return execChain, nil
}

func (msp *MapSourceParser) parseFieldSources(field reflect.StructField) []FieldSource {
	var sources []FieldSource

	if tagValue := field.Tag.Get(MapValueTag); tagValue != "" && tagValue != "-" {
		source := msp.parseSourceTag(tagValue)
		source.Source = MapValueTag
		sources = append(sources, source)
	}

	return sources
}

func (msp *MapSourceParser) parseSourceTag(tag string) FieldSource {
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

// This is the ONLY source-specific function we need to implement!
// All the linked list traversal, field setting, error handling, etc. is handled by BaseExecutionChain
func (msp *MapSourceParser) getValueFromMap(sourceData any, source FieldSource) (any, bool, error) {
	mapData, ok := sourceData.(map[string]string)
	if !ok {
		return nil, false, fmt.Errorf("expected map[string]string, got %T", sourceData)
	}

	value, exists := mapData[source.Key]
	if !exists {
		return nil, false, nil
	}

	return value, true, nil
}

// Example usage struct
type ConfigFromMap struct {
	DatabaseURL string `mapvalue:"db_url"`
	Port        int    `mapvalue:"port,omitempty"`
	Debug       bool   `mapvalue:"debug,omitempty"`
}

func (c *ConfigFromMap) Validate() error {
	if c.DatabaseURL == "" {
		return ValidationError{reason: "database URL is required"}
	}
	return nil
}
