package pave

import (
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/tidwall/gjson"
)

// HTTPRequestParser implements MultipleSourceParser for HTTP requests
type HTTPRequestParser struct {
	*MultiBindingParserTemplate[http.Request, HTTPRequestData]
}

// HTTPRequestData holds parsed HTTP request data to avoid re-parsing
type HTTPRequestData struct {
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
	hp.MultiBindingParserTemplate = NewMultiBindingParserTemplate[http.Request, HTTPRequestData](
		hp.BindingHandler,
		MultiBindingParserTemplateOpts{
			BindingOpts: BindingOpts{
				// Default binding options for HTTP request parsing
			},
			EnableCaching: true,
		},
	)

	return hp
}

func (hp *HTTPRequestParser) GetParserName() string {
	return HTTPRequestParserName
}

func (hp *HTTPRequestParser) BindingHandler(source http.Request, binding FieldBinding) (any, bool, error) {
	// Get or create cache entry for this request instance
	cache := hp.GetBindingCache()
	if cache == nil {
		// Fallback to no caching if cache is disabled
		return hp.bindingHandlerNoCache(&source, binding)
	}

	cacheEntry := cache.GetOrCreate(&source, func() HTTPRequestData {
		return HTTPRequestData{} // Initialize empty cache data
	})

	switch binding.Name {
	case JsonTagBinding:
		return hp.getJSONValue(&source, cacheEntry, binding.Identifier)
	case CookieTagBinding:
		return hp.getCookieValue(&source, cacheEntry, binding.Identifier)
	case HeaderTagBinding:
		return hp.getHeaderValue(&source, cacheEntry, binding.Identifier)
	case QueryTagBinding:
		return hp.getQueryValue(&source, cacheEntry, binding.Identifier)
	default:
		return nil, false, fmt.Errorf("unknown source: %s", binding.Name)
	}
}

// bindingHandlerNoCache is a fallback for when caching is disabled
func (hp *HTTPRequestParser) bindingHandlerNoCache(source *http.Request, binding FieldBinding) (any, bool, error) {
	// Direct parsing without caching - less efficient but simpler
	switch binding.Name {
	case JsonTagBinding:
		return hp.getJSONValueNoCache(source, binding.Identifier)
	case CookieTagBinding:
		return hp.getCookieValueNoCache(source, binding.Identifier)
	case HeaderTagBinding:
		return hp.getHeaderValueNoCache(source, binding.Identifier)
	case QueryTagBinding:
		return hp.getQueryValueNoCache(source, binding.Identifier)
	default:
		return nil, false, fmt.Errorf("unknown source: %s", binding.Name)
	}
}

func (hp *HTTPRequestParser) getJSONValue(source *http.Request, cacheEntry *CacheEntry[HTTPRequestData], fieldName string) (any, bool, error) {
	var jsonBody gjson.Result
	var err error

	cacheEntry.WriteData(func(data *HTTPRequestData) {
		data.bodyOnce.Do(func() {
			if source.Body == nil || source.ContentLength == 0 {
				data.jsonBody = gjson.Parse("{}")
				return
			}

			body, readErr := io.ReadAll(source.Body)
			if readErr != nil {
				data.bodyError = fmt.Errorf("failed to read request body: %w", readErr)
				return
			}

			if len(body) == 0 {
				data.jsonBody = gjson.Parse("{}")
			} else {
				data.jsonBody = gjson.ParseBytes(body)
			}
		})
		jsonBody = data.jsonBody
		err = data.bodyError
	})

	if err != nil {
		return nil, false, err
	}

	result := jsonBody.Get(fieldName)
	if !result.Exists() {
		return nil, false, nil
	}

	return result.Value(), true, nil
}

func (hp *HTTPRequestParser) getCookieValue(source *http.Request, cacheEntry *CacheEntry[HTTPRequestData], name string) (any, bool, error) {
	var cookies map[string]*http.Cookie

	cacheEntry.WriteData(func(data *HTTPRequestData) {
		data.cookiesOnce.Do(func() {
			data.cookies = make(map[string]*http.Cookie)
			for _, cookie := range source.Cookies() {
				data.cookies[cookie.Name] = cookie
			}
		})
		cookies = data.cookies
	})

	cookie, exists := cookies[name]
	if !exists {
		return nil, false, nil
	}

	return cookie.Value, true, nil
}

func (hp *HTTPRequestParser) getHeaderValue(source *http.Request, cacheEntry *CacheEntry[HTTPRequestData], name string) (any, bool, error) {
	var headers map[string]string

	cacheEntry.WriteData(func(data *HTTPRequestData) {
		data.headersOnce.Do(func() {
			data.headers = make(map[string]string)
			for key, values := range source.Header {
				if len(values) > 0 {
					data.headers[key] = values[0]
				}
			}
		})
		headers = data.headers
	})

	value, exists := headers[name]
	if !exists || value == "" {
		return nil, false, nil
	}

	return value, true, nil
}

func (hp *HTTPRequestParser) getQueryValue(source *http.Request, cacheEntry *CacheEntry[HTTPRequestData], name string) (any, bool, error) {
	var queryParams map[string][]string

	cacheEntry.WriteData(func(data *HTTPRequestData) {
		data.queryOnce.Do(func() {
			data.queryParams = source.URL.Query()
		})
		queryParams = data.queryParams
	})

	values, exists := queryParams[name]
	if !exists || len(values) == 0 {
		return nil, false, nil
	}
	return values[0], true, nil
}

// No-cache fallback methods for when caching is disabled

func (hp *HTTPRequestParser) getJSONValueNoCache(source *http.Request, fieldName string) (any, bool, error) {
	if source.Body == nil || source.ContentLength == 0 {
		return nil, false, nil
	}

	body, err := io.ReadAll(source.Body)
	if err != nil {
		return nil, false, fmt.Errorf("failed to read request body: %w", err)
	}

	if len(body) == 0 {
		return nil, false, nil
	}

	jsonBody := gjson.ParseBytes(body)
	result := jsonBody.Get(fieldName)
	if !result.Exists() {
		return nil, false, nil
	}

	return result.Value(), true, nil
}

func (hp *HTTPRequestParser) getCookieValueNoCache(source *http.Request, name string) (any, bool, error) {
	for _, cookie := range source.Cookies() {
		if cookie.Name == name {
			return cookie.Value, true, nil
		}
	}

	return nil, false, nil
}

func (hp *HTTPRequestParser) getHeaderValueNoCache(source *http.Request, name string) (any, bool, error) {
	value := source.Header.Get(name)
	if value == "" {
		return nil, false, nil
	}

	return value, true, nil
}

func (hp *HTTPRequestParser) getQueryValueNoCache(source *http.Request, name string) (any, bool, error) {
	values := source.URL.Query()[name]
	if len(values) == 0 {
		return nil, false, nil
	}
	return values[0], true, nil
}
