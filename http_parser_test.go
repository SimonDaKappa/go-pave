package pave

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Test struct for various HTTP parsing scenarios
type TestStruct struct {
	Name        string `parse:"json:'name'"`
	Age         int    `parse:"json:'age'"`
	Email       string `parse:"json:'email'"`
	OptionalVal string `parse:"json:'optional,omitempty' default:'10'"`
	Page        int    `parse:"query:'page'"`
	Limit       int    `parse:"query:'limit,omitempty' default:'10'"`
	SessionID   string `parse:"cookie:'session_id'"`
	AuthToken   string `parse:"header:'Authorization'"`
	UserAgent   string `parse:"header:'User-Agent,omitempty' default:'10'"`
}

// Test struct for benchmarking
type BenchStruct struct {
	ID            string `parse:"json:'id'"`
	Name          string `parse:"json:'name'"`
	Email         string `parse:"json:'email'"`
	AuthHeader    string `parse:"header:'Authorization'"`
	UserAgent     string `parse:"header:'User-Agent'"`
	SessionCookie string `parse:"cookie:'session'"`
	Page          int    `parse:"query:'page'"`
	Size          int    `parse:"query:'size'"`
}

// createTestRequest creates an HTTP request with all types of data for testing
func createTestRequest() *http.Request {
	// JSON body
	jsonBody := `{
		"id": "user123",
		"name": "John Doe",
		"age": 30,
		"email": "john@example.com",
		"optional": "present"
	}`

	req, _ := http.NewRequest("POST", "http://example.com/api?page=1&limit=20",
		bytes.NewBufferString(jsonBody))

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("User-Agent", "TestAgent/1.0")

	// Add cookies
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "abc123"})
	req.AddCookie(&http.Cookie{Name: "preferenc es", Value: "theme=dark"})

	return req
}

func createBenchRequest() *http.Request {
	// JSON body for benchmarking
	jsonBody := `{
		"id": "bench123",
		"name": "Benchmark User",
		"email": "bench@example.com"
	}`

	query := url.Values{}
	query.Set("page", "100")
	query.Set("size", "50")

	req, _ := http.NewRequest(
		"POST",
		"http://example.com/api?"+query.Encode(),
		bytes.NewBufferString(jsonBody),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer benchtoken")
	req.Header.Set("User-Agent", "BenchmarkAgent/1.0")
	req.AddCookie(&http.Cookie{Name: "session", Value: "sess123"})

	return req
}

// createEmptyRequest creates an HTTP request with minimal data
func createEmptyRequest() *http.Request {
	req, _ := http.NewRequest("GET", "http://example.com/", nil)
	return req
}

func TestHTTPRequestParser_JSONParsing(t *testing.T) {
	parser := NewHTTPRequestParser()
	req := createTestRequest()

	var result TestStruct
	err := parser.Parse(req, &result)

	assert.NoError(t, err)
	assert.Equal(t, "John Doe", result.Name)
	assert.Equal(t, 30, result.Age)
	assert.Equal(t, "john@example.com", result.Email)
	assert.NotNil(t, result.OptionalVal)
	assert.Equal(t, "present", result.OptionalVal)
}

func TestHTTPRequestParser_HeaderParsing(t *testing.T) {
	parser := NewHTTPRequestParser()
	req := createTestRequest()

	var result TestStruct
	err := parser.Parse(req, &result)

	assert.NoError(t, err)
	assert.Equal(t, "Bearer token123", result.AuthToken)
	assert.Equal(t, "TestAgent/1.0", result.UserAgent)
}

func TestHTTPRequestParser_CookieParsing(t *testing.T) {
	parser := NewHTTPRequestParser()
	req := createTestRequest()

	var result TestStruct
	err := parser.Parse(req, &result)

	assert.NoError(t, err)
	assert.Equal(t, "abc123", result.SessionID)
}

func TestHTTPRequestParser_QueryParsing(t *testing.T) {
	parser := NewHTTPRequestParser()
	req := createTestRequest()

	var result TestStruct
	err := parser.Parse(req, &result)

	assert.NoError(t, err)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 20, result.Limit)
}

func TestHTTPRequestParser_EmptyJSONBody(t *testing.T) {
	parser := NewHTTPRequestParser()

	type EmptyStruct struct {
		Name string `parse:"json:'name,omitempty' default:'johndoe12312'"`
	}

	// Test with empty body
	req, _ := http.NewRequest("POST", "http://example.com/", nil)
	var result EmptyStruct
	err := parser.Parse(req, &result)

	assert.NoError(t, err)
	assert.Equal(t, "johndoe12312", result.Name) // Should be zero value
}

func TestHTTPRequestParser_InvalidJSON(t *testing.T) {
	parser := NewHTTPRequestParser()

	type JSONStruct struct {
		Name string `parse:"json:'name'"`
	}

	// Test with invalid JSON
	req, _ := http.NewRequest("POST", "http://example.com/",
		bytes.NewBufferString("{invalid json"))

	var result JSONStruct
	err := parser.Parse(req, &result)

	assert.Error(t, err)
	assert.Equal(t, "", result.Name)
}

func TestHTTPRequestParser_MissingRequiredField(t *testing.T) {
	parser := NewHTTPRequestParser()

	type RequiredStruct struct {
		Name string `parse:"json:'name,required'"`
	}

	// Test with missing required field
	req, _ := http.NewRequest("POST", "http://example.com/",
		bytes.NewBufferString(`{"other": "value"}`))

	var result RequiredStruct
	err := parser.Parse(req, &result)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestHTTPRequestParser_OmitEmptyModifier(t *testing.T) {
	parser := NewHTTPRequestParser()

	type OmitEmptyStruct struct {
		Name  string `parse:"json:'name,omitempty'"`
		Email string `parse:"json:'email,omitempty'"`
	}

	// Test with empty values
	req, _ := http.NewRequest("POST", "http://example.com/",
		bytes.NewBufferString(`{"name": "", "email": ""}`))

	var result OmitEmptyStruct
	err := parser.Parse(req, &result)

	assert.NoError(t, err)
	// Empty strings should remain empty (omitempty doesn't affect parsing, only serialization)
	assert.Equal(t, "", result.Name)
	assert.Equal(t, "", result.Email)
}

func TestHTTPRequestParser_OmitNilModifier(t *testing.T) {
	parser := NewHTTPRequestParser()

	type OmitNilStruct struct {
		OptionalVal string `parse:"json:'optional,omitnil' default:'default value'"`
	}

	// Test with null value in JSON
	req, _ := http.NewRequest("POST", "http://example.com/",
		bytes.NewBufferString(`{"optional": null}`))

	var result OmitNilStruct
	err := parser.Parse(req, &result)

	assert.NoError(t, err)
	assert.Equal(t, "default value", result.OptionalVal)
}

func TestHTTPRequestParser_MultipleHeaders(t *testing.T) {
	parser := NewHTTPRequestParser()

	type MultiHeaderStruct struct {
		Accept      string `parse:"header:'Accept'"`
		ContentType string `parse:"header:'Content-Type'"`
		Custom      string `parse:"header:'X-Custom-Header'"`
	}

	req, _ := http.NewRequest("GET", "http://example.com/", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Custom-Header", "custom-value")

	var result MultiHeaderStruct
	err := parser.Parse(req, &result)

	assert.NoError(t, err)
	assert.Equal(t, "application/json", result.Accept)
	assert.Equal(t, "application/json", result.ContentType)
	assert.Equal(t, "custom-value", result.Custom)
}

func TestHTTPRequestParser_MultipleCookies(t *testing.T) {
	parser := NewHTTPRequestParser()

	type MultiCookieStruct struct {
		Session     string `parse:"cookie:'session'"`
		Preferences string `parse:"cookie:'prefs'"`
		TrackingID  string `parse:"cookie:'tracking'"`
	}

	req, _ := http.NewRequest("GET", "http://example.com/", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "sess123"})
	req.AddCookie(&http.Cookie{Name: "prefs", Value: "dark-theme"})
	req.AddCookie(&http.Cookie{Name: "tracking", Value: "track456"})

	var result MultiCookieStruct
	err := parser.Parse(req, &result)

	assert.NoError(t, err)
	assert.Equal(t, "sess123", result.Session)
	assert.Equal(t, "dark-theme", result.Preferences)
	assert.Equal(t, "track456", result.TrackingID)
}

func TestHTTPRequestParser_ComplexQueryParams(t *testing.T) {
	parser := NewHTTPRequestParser()

	type QueryStruct struct {
		Page   int    `parse:"query:'page'"`
		Size   int    `parse:"query:'size'"`
		Tags   string `parse:"query:'tags'"`
		Filter string `parse:"query:'filter'"`
	}

	// Create URL with complex query parameters
	queryStr := url.Values{}
	queryStr.Set("page", "5")
	queryStr.Set("size", "50")
	queryStr.Set("tags", "golang,testing")
	queryStr.Set("filter", "active")

	req, _ := http.NewRequest("GET", "http://example.com/?"+queryStr.Encode(), nil)

	var result QueryStruct
	err := parser.Parse(req, &result)

	assert.NoError(t, err)
	assert.Equal(t, 5, result.Page)
	assert.Equal(t, 50, result.Size)
	assert.Equal(t, "golang,testing", result.Tags)
	assert.Equal(t, "active", result.Filter)
}

func TestHTTPRequestParser_NestedJSONStructure(t *testing.T) {
	parser := NewHTTPRequestParser()

	type NestedStruct struct {
		UserName    string `parse:"json:'user.name'"`
		UserEmail   string `parse:"json:'user.email'"`
		CompanyName string `parse:"json:'company.name'"`
		CompanyID   int    `parse:"json:'company.id'"`
	}

	jsonBody := `{
		"user": {
			"name": "Jane Doe",
			"email": "jane@example.com"
		},
		"company": {
			"name": "Acme Corp",
			"id": 12345
		}
	}`

	req, _ := http.NewRequest("POST", "http://example.com/",
		bytes.NewBufferString(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	var result NestedStruct
	err := parser.Parse(req, &result)

	assert.NoError(t, err)
	assert.Equal(t, "Jane Doe", result.UserName)
	assert.Equal(t, "jane@example.com", result.UserEmail)
	assert.Equal(t, "Acme Corp", result.CompanyName)
	assert.Equal(t, 12345, result.CompanyID)
}

// Test tag grammar edge cases
func TestHTTPRequestParser_InvalidTagBinding(t *testing.T) {
	parser := NewHTTPRequestParser()

	type InvalidStruct struct {
		Value string `parse:"invalidbinding:'test'"`
	}

	req := createTestRequest()
	var result InvalidStruct
	err := parser.Parse(req, &result)

	assert.Error(t, err)
	if !errors.Is(err, ErrUnallowedBindingName) {
		t.Errorf("Expected ErrUnallowedBindingName, got %v", err)
	}
}

func TestHTTPRequestParser_UnexpectedModifiers(t *testing.T) {
	parser := NewHTTPRequestParser()

	// Test with custom modifiers that shouldn't affect parsing
	type ModifierStruct struct {
		Name  string `parse:"json:'name,custommodifier'"`
		Email string `parse:"json:'email,anothercustom,omitempty'"`
	}

	req := createTestRequest()
	var result ModifierStruct
	err := parser.Parse(req, &result)

	if !errors.Is(err, ErrUnallowedBindingModifier) {
		t.Errorf("Expected ErrUnallowedBindingModifier, got %v", err)
	}
}

func TestHTTPRequestParser_EmptyIdentifiers(t *testing.T) {
	parser := NewHTTPRequestParser()

	type EmptyIdentifierStruct struct {
		Header string `parse:"header:''"`
		Cookie string `parse:"cookie:''"`
		Query  string `parse:"query:''"`
		JSON   string `parse:"json:''"`
	}

	req := createTestRequest()
	var result EmptyIdentifierStruct
	err := parser.Parse(req, &result)

	if !errors.Is(err, ErrEmptyBindingIdentifier) {
		t.Errorf("Expected ErrEmptyBindingIdentifier, got %v", err)
	}
}

func TestHTTPRequestParser_CacheConsistency(t *testing.T) {
	parser := NewHTTPRequestParser()
	req := createTestRequest()

	// Parse multiple times with same request
	var result1, result2, result3 TestStruct

	err1 := parser.Parse(req, &result1)
	err2 := parser.Parse(req, &result2)
	err3 := parser.Parse(req, &result3)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)

	// All results should be identical
	assert.Equal(t, result1, result2)
	assert.Equal(t, result2, result3)
}

// Benchmark tests
func BenchmarkHTTPRequestParser_CachedParsing(b *testing.B) {
	parser := NewHTTPRequestParser()
	req := createBenchRequest()

	// First parse to warm up cache
	var warmup BenchStruct
	_ = parser.Parse(req, &warmup)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result BenchStruct
		err := parser.Parse(req, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHTTPRequestParser_JSONParsing(b *testing.B) {
	parser := NewHTTPRequestParser()

	type JSONOnlyStruct struct {
		ID    string `parse:"json:'id'"`
		Name  string `parse:"json:'name'"`
		Email string `parse:"json:'email'"`
		Age   int    `parse:"json:'age'"`
	}

	jsonBody := `{
		"id": "user123",
		"name": "Test User",
		"email": "test@example.com",
		"age": 25
	}`

	req, _ := http.NewRequest("POST", "http://example.com/",
		bytes.NewBufferString(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {

		var result JSONOnlyStruct
		err := parser.Parse(req, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHTTPRequestParser_HeaderParsing(b *testing.B) {
	parser := NewHTTPRequestParser()

	type HeaderOnlyStruct struct {
		Auth      string `parse:"header:'Authorization'"`
		UserAgent string `parse:"header:'User-Agent'"`
		Accept    string `parse:"header:'Accept'"`
		Host      string `parse:"header:'Host'"`
	}

	req, _ := http.NewRequest("GET", "http://example.com/", nil)
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("User-Agent", "TestAgent/1.0")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Host", "example.com")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {

		var result HeaderOnlyStruct
		err := parser.Parse(req, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHTTPRequestParser_CookieParsing(b *testing.B) {
	parser := NewHTTPRequestParser()

	type CookieOnlyStruct struct {
		SessionID string `parse:"cookie:'session_id'"`
		Theme     string `parse:"cookie:'theme'"`
	}

	req, _ := http.NewRequest("GET", "http://example.com/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess123"})
	req.AddCookie(&http.Cookie{Name: "theme", Value: "dark"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result CookieOnlyStruct
		err := parser.Parse(req, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHTTPRequestParser_QueryParsing(b *testing.B) {
	parser := NewHTTPRequestParser()

	type QueryOnlyStruct struct {
		Page  int    `parse:"query:'page'"`
		Limit int    `parse:"query:'limit'"`
		Sort  string `parse:"query:'sort'"`
	}

	queryStr := url.Values{}
	queryStr.Set("page", "1")
	queryStr.Set("limit", "20")
	queryStr.Set("sort", "asc")

	req, _ := http.NewRequest("GET", "http://example.com/?"+queryStr.Encode(), nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {

		var result QueryOnlyStruct
		err := parser.Parse(req, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHTTPRequestParser_ComplexParsing(b *testing.B) {
	parser := NewHTTPRequestParser()
	req := createTestRequest()
	var result TestStruct

	b.ResetTimer()
	for i := 0; i < b.N; i++ {

		err := parser.Parse(req, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Edge case tests
func TestHTTPRequestParser_NilRequest(t *testing.T) {
	parser := NewHTTPRequestParser()

	var result TestStruct
	err := parser.Parse(nil, &result)

	assert.Error(t, err)
}

func TestHTTPRequestParser_LargeJSONBody(t *testing.T) {
	parser := NewHTTPRequestParser()

	type LargeStruct struct {
		Data string `parse:"json:'data'"`
	}

	// Create a large JSON payload
	largeData := strings.Repeat("x", 10000)
	jsonBody := fmt.Sprintf(`{"data": "%s"}`, largeData)

	req, _ := http.NewRequest("POST", "http://example.com/",
		bytes.NewBufferString(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	var result LargeStruct
	err := parser.Parse(req, &result)

	assert.NoError(t, err)
	assert.Equal(t, largeData, result.Data)
}

// TODO $$$SIMON FIX
// func TestHTTPRequestParser_SpecialCharactersInValues(t *testing.T) {
// 	parser := NewHTTPRequestParser()

// 	type SpecialCharsStruct struct {
// 		JSONField   string `parse:"json:'special'"`
// 		HeaderField string `parse:"header:'X-Special'"`
// 		CookieField string `parse:"cookie:'special'"`
// 		QueryField  string `parse:"query:'special'"`
// 	}

// 	specialValue := `{"nested": "value with spaces & symbols !@#$%^&*()"}`
// 	jsonBody := fmt.Sprintf(`{"special": %q}`, specialValue)

// 	queryStr := url.Values{}
// 	queryStr.Set("special", specialValue)

// 	req, _ := http.NewRequest("POST", "http://example.com/?"+queryStr.Encode(),
// 		bytes.NewBufferString(jsonBody))
// 	req.Header.Set("X-Special", specialValue)
// 	req.AddCookie(&http.Cookie{Name: "special", Value: specialValue})

// 	var result SpecialCharsStruct
// 	err := parser.Parse(req, &result)

// 	assert.NoError(t, err)
// 	assert.Equal(t, specialValue, result.JSONField)
// 	assert.Equal(t, specialValue, result.HeaderField)
// 	assert.Equal(t, specialValue, result.CookieField)
// 	assert.Equal(t, specialValue, result.QueryField)
// }

func TestHTTPRequestParser_BenchmarkParseChainCache(t *testing.T) {

	var iterCount int = 1000

	var timesCacheDisabled, timesCacheEnabled []time.Duration

	// Test time with cache disabled (just create a new parser each time)
	for i := 0; i < iterCount; i++ {
		parser := NewHTTPRequestParser()
		req := createBenchRequest()

		t1 := time.Now()
		var result BenchStruct
		err := parser.Parse(req, &result)
		assert.NoError(t, err)
		timesCacheDisabled = append(timesCacheDisabled, time.Since(t1))
	}

	// Test time with cache enabled
	parser := NewHTTPRequestParser()
	req := createBenchRequest()
	for i := 0; i < iterCount; i++ {
		var result BenchStruct
		t1 := time.Now()
		err := parser.Parse(req, &result)
		assert.NoError(t, err)
		timesCacheEnabled = append(timesCacheEnabled, time.Since(t1))
	}

	// Calculate average times
	var avgCacheDisabled, avgCacheEnabled time.Duration
	for _, t := range timesCacheDisabled {
		avgCacheDisabled += t
	}
	for _, t := range timesCacheEnabled {
		avgCacheEnabled += t
	}
	avgCacheDisabled /= time.Duration(iterCount)
	avgCacheEnabled /= time.Duration(iterCount)

	fmt.Printf("Average time with cache disabled: %v\n", avgCacheDisabled)
	fmt.Printf("Average time with cache enabled: %v\n", avgCacheEnabled)
	assert.Less(t, avgCacheEnabled, avgCacheDisabled, "Cache should improve performance")
}
