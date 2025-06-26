package pave

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type JSONByteSliceSourceParser struct {
}

func NewJsonByteSliceSourceParser() *JSONByteSliceSourceParser {
	return &JSONByteSliceSourceParser{}
}

func (jsp *JSONByteSliceSourceParser) GetSourceType() reflect.Type {
	return JSONByteSliceType
}

func (jsp *JSONByteSliceSourceParser) GetParserName() string {
	return JSONByteSliceParserName
}

func (jsp *JSONByteSliceSourceParser) Parse(source any, dest any) error {
	return TypeErasureParseWrapper(jsp.parse)(source, dest)
}

func (jsp *JSONByteSliceSourceParser) parse(source []byte, dest any) error {
	if reflect.TypeOf(dest).Kind() != reflect.Ptr {
		return fmt.Errorf("destination must be a pointer to a struct, got %T", dest)
	}

	if err := json.Unmarshal(source, dest); err != nil {
		return fmt.Errorf("error unmarshaling JSON data: %w", err)
	}
	return nil
}

type JSONStringSourceParser struct{}

func NewJSONStringSourceParser() *JSONStringSourceParser {
	return &JSONStringSourceParser{}
}
func (jsp *JSONStringSourceParser) GetSourceType() reflect.Type {
	return StringType
}

func (jsp *JSONStringSourceParser) GetParserName() string {
	return JSONStringParserName
}

func (jsp *JSONStringSourceParser) Parse(source any, dest any) error {
	return TypeErasureParseWrapper(jsp.parse)(source, dest)
}

func (jsp *JSONStringSourceParser) parse(source string, dest any) error {
	if reflect.TypeOf(dest).Kind() != reflect.Ptr {
		return fmt.Errorf("destination must be a pointer to a struct, got %T", dest)
	}

	if err := json.Unmarshal([]byte(source), dest); err != nil {
		return fmt.Errorf("error unmarshaling JSON data: %w", err)
	}
	return nil
}
