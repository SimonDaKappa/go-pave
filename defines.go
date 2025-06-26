package pave

import (
	"net/http"
	"reflect"
)

// constants for subtag prefixes in parse subtag
const (
	ParseTagPrefix                          = "parse"
	DefaultValueSubTagPrefix                = "default"
	DefaultValueSubTagPrefixWithKVDelimiter = "default:"
	DefaultSubTagScopeDelimiter             = byte('\'')
	DefaultKeyValueTagDelimiter             = ":"
)

// constants for builtin source bindings in parse subtag
const (
	JsonTagBinding     = "json"
	CookieTagBinding   = "cookie"
	HeaderTagBinding   = "header"
	QueryTagBinding    = "query"
	MapValueTagBinding = "mapvalue"
)

// constants for builtin source binding modifiers
const (
	OmitEmptyBindingModifier = "omitempty"
	OmitNilBindingModifier   = "omitnil"
	OmitErrBindingModifier   = "omiterr"
	RequiredBindingModifier  = "required"
)

// Parser Name constants for built in parsers.
const (
	HTTPRequestParserName   = "http-request-parser"
	JSONByteSliceParserName = "json-[]byte-parser"
	JSONStringParserName    = "json-string-parser"
	StringMapParserName     = "stringmap-parser"
	StringAnyMapParserName  = "map-parser"
)

// Mime Type constants for content types and encodings.
const (
	ContentEncodingUTF8        string = "UTF-8"
	ContentTypeApplicationJSON string = "application/json"
	ContentTypeDelimiter              = ";"
)

// reflect.TypeOf constants for type checks
var (
	HTTPRequestType   = reflect.TypeOf((*http.Request)(nil))
	JSONByteSliceType = reflect.TypeOf([]byte{})
	StringType        = reflect.TypeOf("")
	StringMapType     = reflect.TypeOf(map[string]string{})
	StringAnyMapType  = reflect.TypeOf(map[string]any{})
)

func init() {

}
