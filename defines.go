package pave

import (
	"net/http"
	"reflect"
	"time"

	"github.com/google/uuid"
)

// constants for subtag prefixes in parse subtag
const (
	ParseTagPrefix                          string = "parse"
	DefaultValueSubTagPrefix                string = "default"
	DefaultValueSubTagPrefixWithKVDelimiter string = "default:"
	bDefaultSubTagScopeDelimiter            byte   = byte('\'')
	sDefaultSubTagScopeDelimiter            string = "'"
	DefaultKeyValueTagDelimiter             string = ":"
	CommaDelimeter                          string = ","
)

// constants for builtin source bindings in parse subtag
const (
	JsonTagBinding     string = "json"
	CookieTagBinding   string = "cookie"
	HeaderTagBinding   string = "header"
	QueryTagBinding    string = "query"
	MapValueTagBinding string = "mapvalue"
)

// constants for builtin source binding modifiers
const (
	OmitEmptyBindingModifier string = "omitempty"
	OmitNilBindingModifier   string = "omitnil"
	OmitErrorBindingModifier string = "omiterror"
)

// Parser Name constants for built in parsers.
const (
	HTTPRequestParserName   string = "http-request-parser"
	JSONByteSliceParserName string = "json-[]byte-parser"
	JSONStringParserName    string = "json-string-parser"
	StringMapParserName     string = "stringmap-parser"
	StringAnyMapParserName  string = "map-parser"
)

// Mime Type constants for content types and encodings.
const (
	ContentEncodingUTF8        string = "UTF-8"
	ContentTypeApplicationJSON string = "application/json"
	ContentTypeDelimiter       string = ";"
)

// reflect.TypeOf constants for type checks
var (
	HTTPRequestType   reflect.Type
	JSONByteSliceType reflect.Type
	StringType        reflect.Type
	StringMapType     reflect.Type
	StringMapAnyType  reflect.Type
)

// reflect.TypeOf constants for special struct types
var (
	TimeType reflect.Type
	UUIDType reflect.Type
)

func init() {
	initTypes()
}

func initTypes() {
	initBuiltinSourceTypes()
	initSpecialStructTypes()
}

func initBuiltinSourceTypes() {
	HTTPRequestType = reflect.TypeOf(http.Request{})
	JSONByteSliceType = reflect.TypeOf([]byte{})
	StringType = reflect.TypeOf("")
	StringMapType = reflect.TypeOf(map[string]string{})
	StringMapAnyType = reflect.TypeOf(map[string]any{})
}

func initSpecialStructTypes() {
	// Initialize special struct types that should not be parsed recursively
	TimeType = reflect.TypeOf(time.Time{})
	UUIDType = reflect.TypeOf(uuid.UUID{})
}
