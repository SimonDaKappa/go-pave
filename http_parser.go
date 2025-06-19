package pave

import (
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/tidwall/gjson"
)

const ()

var (
	__httpRequestType = reflect.TypeOf((*http.Request)(nil))

	__compTimeCheckImplementsMultiSourceParser MultipleSourceParser = &HTTPRequestParser{}
)

// HTTPRequestParser implements MultipleSourceParser for HTTP requests
type HTTPRequestParser struct {
	BaseMultipleSourceParser
}

// HTTPRequestData holds parsed HTTP request data to avoid re-parsing
type HTTPRequestData struct {
	request *http.Request
	// Cached JSON body to avoid repeated parsing
	jsonBody  gjson.Result
	bodyOnce  sync.Once
	bodyError error
	// Cache query parameters to avoid repeated URL.Query() calls
	queryParams map[string][]string
	queryOnce   sync.Once

	headers     map[string]string // Cached headers for quick access
	headersOnce sync.Once

	cookies     map[string]*http.Cookie // Cached cookies for quick access
	cookiesOnce sync.Once
}

func NewHTTPRequestParser() *HTTPRequestParser {
	hp := &HTTPRequestParser{}
	hp.BaseMultipleSourceParser = NewBaseMultipleSourceParser(
		hp.parseFieldSources,
		hp.getValueFromSource,
	)

	return hp
}

func (hp *HTTPRequestParser) GetSourceType() reflect.Type {
	return __httpRequestType
}

func (hp *HTTPRequestParser) GetParserName() string {
	return HTTPRequestParserName
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
	// Parse cookies once and cache them
	data.cookiesOnce.Do(func() {
		data.cookies = make(map[string]*http.Cookie)
		for _, cookie := range data.request.Cookies() {
			data.cookies[cookie.Name] = cookie
		}
	})

	cookie, exists := data.cookies[name]
	if !exists {
		return nil, false, nil
	}

	return cookie.Value, true, nil
}

func (hp *HTTPRequestParser) getHeaderValue(data *HTTPRequestData, name string) (any, bool, error) {
	// Parse headers once and cache them
	data.headersOnce.Do(func() {
		data.headers = make(map[string]string)
		for key, values := range data.request.Header {
			if len(values) > 0 {
				data.headers[key] = values[0]
			}
		}
	})

	value, exists := data.headers[name]
	if !exists || value == "" {
		return nil, false, nil
	}

	// Handle Authorization header specially
	if name == "Authorization" && strings.HasPrefix(value, "Bearer ") {
		value = strings.TrimPrefix(value, "Bearer ")
	}

	return value, true, nil
}

func (hp *HTTPRequestParser) getQueryValue(data *HTTPRequestData, name string) (any, bool, error) {
	// Parse query parameters once and cache them
	data.queryOnce.Do(func() {
		data.queryParams = data.request.URL.Query()
	})

	values, exists := data.queryParams[name]
	if !exists || len(values) == 0 {
		return nil, false, nil
	}
	return values[0], true, nil
}
