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

func (jbsp *JSONByteSliceSourceParser) SourceType() reflect.Type {
	return JSONByteSliceType
}

func (jbsp *JSONByteSliceSourceParser) Name() string {
	return JSONByteSliceParserName
}

func (jbsp *JSONByteSliceSourceParser) Parse(source any, dest any) error {
	return ParseTypeErasedSlice(source, dest, jbsp.parse)
}

func (jbsp *JSONByteSliceSourceParser) parse(source []byte, dest any) error {
	if err := json.Unmarshal(source, dest); err != nil {
		return fmt.Errorf("error unmarshaling JSON data: %w", err)
	}
	return nil
}

type JSONStringSourceParser struct{}

func NewJSONStringSourceParser() *JSONStringSourceParser {
	return &JSONStringSourceParser{}
}
func (jssp *JSONStringSourceParser) SourceType() reflect.Type {
	return StringType
}

func (jssp *JSONStringSourceParser) Name() string {
	return JSONStringParserName
}

func (jssp *JSONStringSourceParser) Parse(source any, dest any) error {
	return ParseTypeErasedPointer(source, dest, jssp.parse)
}

func (jssp *JSONStringSourceParser) parse(source *string, dest any) error {
	if err := json.Unmarshal([]byte(*source), dest); err != nil {
		return fmt.Errorf("error unmarshaling JSON data: %w", err)
	}
	return nil
}
