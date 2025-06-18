package pave

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/tidwall/gjson"
)

const (
	JSONSourceTag   = "json"
	CookieSourceTag = "cookie"
	HeaderSourceTag = "header"
	QuerySourceTag  = "query"
)

var (
	__httpRequestType = reflect.TypeOf((*http.Request)(nil))
)

// HTTPRequestParser implements SourceParser for HTTP requests
type HTTPRequestParser struct {
	chains     map[reflect.Type]*BaseExecutionChain
	chainMutex sync.RWMutex
}

// HTTPRequestData holds parsed HTTP request data to avoid re-parsing
type HTTPRequestData struct {
	request   *http.Request
	jsonBody  gjson.Result
	bodyOnce  sync.Once
	bodyError error
}

func NewHTTPRequestParser() *HTTPRequestParser {
	return &HTTPRequestParser{
		chains: make(map[reflect.Type]*BaseExecutionChain),
	}
}

func (hp *HTTPRequestParser) GetSourceType() reflect.Type {
	return __httpRequestType
}

func (hp *HTTPRequestParser) Parse(source any, dest Validatable) error {
	request, ok := source.(*http.Request)
	if !ok {
		return fmt.Errorf("expected *http.Request, got %T", source)
	}

	// Get the struct type
	destType := reflect.TypeOf(dest)
	if destType.Kind() == reflect.Ptr {
		destType = destType.Elem()
	}

	// Get or build the execution chain
	chain, err := hp.GetParseChain(destType)
	if err != nil {
		return err
	}

	// Create HTTP request data wrapper
	requestData := &HTTPRequestData{request: request}

	// Execute the chain with our HTTP-specific source getter
	return chain.Execute(requestData, dest)
}

func (hp *HTTPRequestParser) GetParseChain(t reflect.Type) (*BaseExecutionChain, error) {
	hp.chainMutex.RLock()
	if chain, exists := hp.chains[t]; exists {
		hp.chainMutex.RUnlock()
		return chain, nil
	}
	hp.chainMutex.RUnlock()

	return hp.BuildParseChain(t)
}

func (hp *HTTPRequestParser) BuildParseChain(t reflect.Type) (*BaseExecutionChain, error) {
	chain, err := hp.buildChainForType(t)
	if err != nil {
		return nil, err
	}

	// RW Lock map edits
	hp.chainMutex.Lock()
	hp.chains[t] = chain
	hp.chainMutex.Unlock()

	return chain, nil
}

func (hp *HTTPRequestParser) buildChainForType(t reflect.Type) (*BaseExecutionChain, error) {
	var head, current *ParseStep

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		sources := hp.parseFieldSources(field)
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

	// Create the execution chain with our HTTP-specific source getter
	execChain := &BaseExecutionChain{
		StructType:   t,
		Head:         head,
		SourceGetter: hp.getValueFromSource,
	}

	return execChain, nil
}

func (hp *HTTPRequestParser) parseFieldSources(field reflect.StructField) []FieldSource {
	var sources []FieldSource

	// Parse different source types in priority order
	sourceTypes := []string{HeaderSourceTag, CookieSourceTag, QuerySourceTag, JSONSourceTag}

	for _, sourceType := range sourceTypes {
		if tagValue := field.Tag.Get(sourceType); tagValue != "" && tagValue != "-" {
			source := hp.parseSourceTag(tagValue)
			source.Source = sourceType
			sources = append(sources, source)
		}
	}

	return sources
}

func (hp *HTTPRequestParser) parseSourceTag(tag string) FieldSource {
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

func (hp *HTTPRequestParser) getValueFromSource(sourceData any, source FieldSource) (any, bool, error) {
	requestData, ok := sourceData.(*HTTPRequestData)
	if !ok {
		return nil, false, fmt.Errorf("expected *HTTPRequestData, got %T", sourceData)
	}

	switch source.Source {
	case JSONSourceTag:
		return hp.getJSONValue(requestData, source.Key)
	case CookieSourceTag:
		return hp.getCookieValue(requestData, source.Key)
	case HeaderSourceTag:
		return hp.getHeaderValue(requestData, source.Key)
	case QuerySourceTag:
		return hp.getQueryValue(requestData, source.Key)
	default:
		return nil, false, fmt.Errorf("unknown source: %s", source.Source)
	}
}

func (hp *HTTPRequestParser) getJSONValue(data *HTTPRequestData, fieldName string) (any, bool, error) {
	jsonBody, err := hp.getJSONBody(data)
	if err != nil {
		return nil, false, err
	}

	result := jsonBody.Get(fieldName)
	if !result.Exists() {
		return nil, false, nil
	}

	return result.Value(), true, nil
}

func (hp *HTTPRequestParser) getJSONBody(data *HTTPRequestData) (gjson.Result, error) {
	data.bodyOnce.Do(func() {
		if data.request.Body == nil || data.request.ContentLength == 0 {
			data.jsonBody = gjson.Parse("{}")
			return
		}

		body, err := io.ReadAll(data.request.Body)
		if err != nil {
			data.bodyError = fmt.Errorf("failed to read request body: %w", err)
			return
		}

		if len(body) == 0 {
			data.jsonBody = gjson.Parse("{}")
		} else {
			data.jsonBody = gjson.ParseBytes(body)
		}
	})

	return data.jsonBody, data.bodyError
}

func (hp *HTTPRequestParser) getCookieValue(data *HTTPRequestData, name string) (any, bool, error) {
	cookie, err := data.request.Cookie(name)
	if err != nil && errors.Is(err, http.ErrNoCookie) {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}

	return cookie.Value, true, nil
}

func (hp *HTTPRequestParser) getHeaderValue(data *HTTPRequestData, name string) (any, bool, error) {
	value := data.request.Header.Get(name)
	if value == "" {
		return nil, false, nil
	}

	// Handle Authorization header specially
	if name == "Authorization" && strings.HasPrefix(value, "Bearer ") {
		value = strings.TrimPrefix(value, "Bearer ")
	}

	return value, true, nil
}

func (hp *HTTPRequestParser) getQueryValue(data *HTTPRequestData, name string) (any, bool, error) {
	values, exists := data.request.URL.Query()[name]
	if !exists || len(values) == 0 {
		return nil, false, nil
	}
	return values[0], true, nil
}
