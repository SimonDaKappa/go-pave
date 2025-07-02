package pave

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/tidwall/gjson"
)

var (
	// Default HTTPRequestParser Binding Options
	_httpBindingOpts = BindingOpts{
		AllowedBindingNames: []string{
			JsonTagBinding,
			CookieTagBinding,
			HeaderTagBinding,
			QueryTagBinding,
		},
		CustomBindingModifiers: []string{},
	}

	// Default HTTPRequestParser ParseChainManager Options
	_httpPCMOpts = ParseChainManagerOpts{
		BindingOpts: _httpBindingOpts,
	}

	// Default HTTPRequestParser Options
	_httpParserOpts = BaseMBParserOpts{
		UseCache: true,
		PCMOpts:  _httpPCMOpts,
	}
)

// HTTPRequestParser provides a parser for HTTP requests with the
// the following features:
//   - Parses JSON body, cookies, headers, and query parameters
//   - Caches repetively parsed data by http package to avoid extra
//     computation and allocs.
//   - Supports both cached and non-cached parsing
//   - Implements the MultiBindingParser interface for flexible
//     field bindings
//
// The following Field Bindings are supported:
//   - `json:<key>`: Parses a JSON key from the request body
//   - `cookie:<key>`: Parses a cookie value by key
//   - `header:<key>`: Parses a header value by key
//   - `query:<key>`: Parses a query parameter value by key
//
// Like all other MultiBindingParsers, this parser caches the
// parsing strategy (ParseChain) for each destination type, so
// that only the first parse takes the time to build the chain,
// and subsequent parses simply execute the pre-built chain.
//
// This parser expects the standard parse tag format.
//
// This parser does not support any custom modifiers, but it does
// support all standard modifiers (required, omitempty, omitnil, omiterr)
type HTTPRequestParser struct {
	*BaseMBParser[http.Request, HTTPRequestOnce]
}

func NewHTTPRequestParser() *HTTPRequestParser {
	base := NewBaseMBParser[http.Request, HTTPRequestOnce](
		NewHTTPBindingManager(),
		_httpParserOpts,
	)

	return &HTTPRequestParser{
		BaseMBParser: base,
	}
}

func (hp *HTTPRequestParser) Name() string {
	return HTTPRequestParserName
}

type HTTPBindingManager struct{}

func NewHTTPBindingManager() *HTTPBindingManager {
	return &HTTPBindingManager{}
}

func (mgr *HTTPBindingManager) BindingHandlerCached(
	source *http.Request,
	entry *CacheEntry[HTTPRequestOnce],
	binding Binding,
) (any, bool, error) {

	if entry == nil {
		return nil, false, ErrBindingCacheNilEntry
	}

	switch binding.Name {
	case JsonTagBinding:
		return mgr.JSONValue(source, entry, binding.Identifier)
	case CookieTagBinding:
		return mgr.CookieValue(source, entry, binding.Identifier)
	case HeaderTagBinding:
		return mgr.HeaderValue(source, entry, binding.Identifier)
	case QueryTagBinding:
		return mgr.QueryValue(source, entry, binding.Identifier)
	default:
		return nil, false, fmt.Errorf("unknown binding: %s", binding.Name)
	}
}

func (mgr *HTTPBindingManager) BindingHandler(
	source *http.Request,
	binding Binding,
) (any, bool, error) {

	// This should be fine. We onyl allow instances of HTTBindingManager to be
	// created by the HTTPRequestParser, which always uses the cache.
	return nil, false, fmt.Errorf("uncached handler not implemented for HTTPBindingManager")
}

func (hp *HTTPBindingManager) NewCached() HTTPRequestOnce {
	return NewHTTPRequestOnce()
}

func (hp *HTTPBindingManager) JSONValue(
	source *http.Request,
	entry *CacheEntry[HTTPRequestOnce],
	key string,
) (any, bool, error) {

	var jsonBody gjson.Result
	var err error

	entry.WriteData(func(data *HTTPRequestOnce) {
		data.bodyOnce.Do(func() {
			if source.Body == nil || source.ContentLength == 0 {
				data.jsonBody = gjson.Parse("{}")
				return
			}

			// Read body and restore it to so others can read it
			body, readErr := io.ReadAll(source.Body)
			if readErr != nil {
				source.Body.Close()
				data.bodyError = fmt.Errorf("failed to read request body: %w", readErr)
				return
			}
			source.Body.Close()
			source.Body = io.NopCloser(bytes.NewReader(body))

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

	result := jsonBody.Get(key)
	if !result.Exists() {
		return nil, false, nil
	}

	return result.Value(), true, nil
}

func (hp *HTTPBindingManager) CookieValue(
	source *http.Request,
	entry *CacheEntry[HTTPRequestOnce],
	key string,
) (any, bool, error) {

	var cookies map[string]*http.Cookie

	entry.WriteData(func(data *HTTPRequestOnce) {
		data.cookiesOnce.Do(func() {
			data.cookies = make(map[string]*http.Cookie)
			for _, cookie := range source.Cookies() {
				data.cookies[cookie.Name] = cookie
			}
		})
		cookies = data.cookies
	})

	cookie, exists := cookies[key]
	if !exists {
		return nil, false, nil
	}

	return cookie.Value, true, nil
}

func (hp *HTTPBindingManager) HeaderValue(
	source *http.Request,
	entry *CacheEntry[HTTPRequestOnce],
	key string,
) (any, bool, error) {

	var headers map[string]string

	entry.WriteData(func(data *HTTPRequestOnce) {
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

	value, exists := headers[key]
	if !exists || value == "" {
		return nil, false, nil
	}

	return value, true, nil
}

func (hp *HTTPBindingManager) QueryValue(
	source *http.Request,
	entry *CacheEntry[HTTPRequestOnce],
	key string,
) (any, bool, error) {

	var queryParams map[string][]string

	entry.WriteData(func(data *HTTPRequestOnce) {
		data.queryOnce.Do(func() {
			data.queryParams = source.URL.Query()
		})
		queryParams = data.queryParams
	})

	values, exists := queryParams[key]
	if !exists || len(values) == 0 {
		return nil, false, nil
	}
	return values[0], true, nil
}

// HTTPRequestOnce holds parsed HTTP request data to avoid re-parsing
// on subsequent accesses. It uses sync.Once to ensure that
// parsing is only done once per request instance. This is the
// `Cached` type used by the MBPTemplate for HTTPRequestParser.
type HTTPRequestOnce struct {
	jsonBody    gjson.Result            // Parsed JSON body from the request
	queryParams map[string][]string     // Parsed query parameters from the request
	headers     map[string]string       // Parsed headers from the request
	cookies     map[string]*http.Cookie // Parsed cookies from the request

	bodyOnce    sync.Once // Ensures the body is read only once
	queryOnce   sync.Once // Ensures query parameters are parsed only once
	headersOnce sync.Once // Ensures headers are parsed only once
	cookiesOnce sync.Once // Ensures cookies are parsed only once

	bodyError error // Error encountered while reading the request body
}

func NewHTTPRequestOnce() HTTPRequestOnce {
	return HTTPRequestOnce{
		queryParams: make(map[string][]string),
		headers:     make(map[string]string),
		cookies:     make(map[string]*http.Cookie),
	}
}
