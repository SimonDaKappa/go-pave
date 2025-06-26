package pave

import (
	"errors"
	"fmt"
	"reflect"
)

///////////////////////////////////////////////////////////////////////////////
// Misc.
///////////////////////////////////////////////////////////////////////////////

var (
	ErrParserMethodNotImplemented = errors.New("method not implemented, please override in concrete parser implementation")
)

///////////////////////////////////////////////////////////////////////////////
// SourceParser Interface
///////////////////////////////////////////////////////////////////////////////

type SourceParser interface {
	// Parse extracts the information from the implementation and populates
	// v using the execution chain system.
	Parse(data any, v Validatable) error
	// GetSourceType returns the reflect.Type of the source this parser works with
	GetSourceType() reflect.Type
	// GetParserName returns a unique identifier for this parser within its source type
	GetParserName() string
}

///////////////////////////////////////////////////////////////////////////////
// OneshotSourceParser
///////////////////////////////////////////////////////////////////////////////

// OneShotSourceParser defines the interface for extracting information
// from the implementation of this interface and filling a Validatable
// type with it. A OneShotSourceParser is defined for each Type that you wish
// to parse.
//
// Use this interface when you know that the each field in your struct
// will only source from a single place in your source type.
//
// # Oneshot Source Parsers DO NOT build an execution chain.
//
// It is assumed that the Parse method will handle all parsing in a single shot.
// This means that tags for must necessarily be simple and only specify a single
// source.
//
// For example, json unmarshalling a struct via `json:"fieldname"` will always
// source from the JSON body, so you can use a OneShotSourceParser for that.
//
// # The following are implemented by default:
//   - JsonSourceParser: Parses from []byte containing JSON data using the
//     `json` tag.
//   - StringMapSourceParser: Parses from a map[string]string source using the
//     `mapvalue` tag.
type OneShotSourceParser interface {
	SourceParser
}

///////////////////////////////////////////////////////////////////////////////
// MultipleSourceParser
///////////////////////////////////////////////////////////////////////////////

// MultipleSourceParser defines the interface for extracting information
// from the implementation of this interface and filling a Validatable
// type with it. A MultipleSourceParser is defined for each Type that you wish
// to parse.
//
// Use this interface when any field in your struct could possibly source
// from multiple places in your source type.
//
// For example, HTTP Requests have useful extractable information from cookies,
// headers, query parameters, and the body. For instance, you might usually get
// a resource by the ID passed in the URL, but sometimes you might want to
// allow the ID to be passed in a header or cookie instead.
//
// This allows a single struct to be reused in multiple contexts.
//
// # The following are implemented by default:
//   - HTTPRequestParser: Parses from an *http.Request using the
//     `json`, `cookie`, `header`, and `query` tags.
type MultipleSourceParser interface {
	SourceParser

	GetParseChain(destType reflect.Type) (*ExecutionChain, error)
	BuildParseChain(destType reflect.Type) (*ExecutionChain, error)
}

// BaseMultipleSourceParser is a base implementation of MultipleSourceParser
// that provides common functionality for building and executing parse chains./
//
// It is used to create a MultipleSourceParser that can handle fields
// with multiple sources, such as HTTP requests, where fields can be sourced
// from cookies, headers, query parameters, and the body.
//
// Use this struct by embedding it in your own MultipleSourceParser implementation
// and providing the necessary methods for parsing field sources and retrieving
// values from those sources.
//
// In order to satisfy SourceParser, you must implement:
//   - Parse(data any, v Validatable) error
//   - GetSourceType() reflect.Type
//   - GetParserName() string
//
// Example Implementation:
//
//	type HTTPRequestParser struct {
//	  BaseMultipleSourceParser
//	}
//
//	func NewHTTPRequestParser() *HTTPRequestParser {
//	    hp := &HTTPRequestParser{}
//	    hp.BaseMultipleSourceParser = NewBaseMultipleSourceParser(
//		    hp.parseFieldSources,
//		    hp.getValueFromSource,
//	    )
//	 return hp
//	}
//
//	func (hp *HTTPRequestParser) Parse(source any, dest Validatable) error {
//	 request, ok := source.(*http.Request)
//	 if !ok {
//	     return fmt.Errorf("expected *http.Request, got %T", source)
//	 }
//
//	// Get the struct type
//	destType := reflect.TypeOf(dest)
//	if destType.Kind() == reflect.Ptr {
//	    destType = destType.Elem()
//	}
//
//	// Get or build the execution chain
//	chain, err := hp.GetParseChain(destType)
//	if err != nil {
//	    return err
//	}
//	// Create HTTP request data wrapper
//	requestData := &HTTPRequestData{request: request}
//
//	// Execute the chain with our HTTP-specific source getter
//	return chain.Execute(requestData, dest)
//	}
//
//	// Not Shown: FieldSourceParser and ValueGetter implementations
type BaseMultipleSourceParser[S any] struct {
	ParseChainBuilder[S]
}

type BaseMultipleSourceParserOpts struct {
	BindingOpts
}

func NewBaseMultipleSourceParser[S any](handler BindingHandler[S], opts BaseMultipleSourceParserOpts) BaseMultipleSourceParser[S] {
	return BaseMultipleSourceParser[S]{
		ParseChainBuilder: NewParseChainBuilder(
			handler,
			ParseChainBuilderOpts{BindingOpts: opts.BindingOpts},
		),
	}
}
