package pave

import (
	"reflect"
)

type StringMap[T any] map[string]T

type MapValueSourceParser struct{}

func (mvp *MapValueSourceParser) GetSourceType() reflect.Type {
	return StringAnyMapType
}

func (mvp *MapValueSourceParser) GetParserName() string {
	return StringAnyMapParserName
}
