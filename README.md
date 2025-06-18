# Go Validation Library

A flexible validation library for Go that supports parsing and validating data from multiple sources with execution chains for optimal performance.

## Features

- **Multiple Source Support**: Parse data from HTTP headers, cookies, query parameters, and JSON body
- **Priority-based Parsing**: Sources are tried in priority order with fallback support
- **Execution Chains**: Pre-built, cached execution chains for optimal performance
- **Lazy JSON Parsing**: JSON body is only parsed when needed and cached for reuse
- **Extensible Design**: Easy to add custom source parsers
- **Thread-safe**: Concurrent access to cached execution chains

## Basic Usage

### 1. Define a struct with validation tags

```go
type User struct {
    // Try header first, then query, then JSON as fallback
    ID       uuid.UUID `header:"X-User-ID,omitempty" query:"user_id,omitempty" json:"id,omitempty"`
    
    // Try query first, then JSON
    Name     string    `query:"name" json:"name"`
    
    // Only from JSON
    Email    string    `json:"email,omitempty"`
    
    // Only from header, with automatic Bearer prefix removal
    Token    string    `header:"Authorization,omitempty"`
    
    // Try cookie first, then query as fallback
    SessionID string   `cookie:"session_id,omitempty" query:"session"`
    
    // Optional field from JSON
    Age      int       `json:"age,omitempty"`
}

// Implement the Validatable interface
func (u *User) Validate() error {
    if u.Name == "" {
        return ValidationError{reason: "name is required"}
    }
    return nil
}
```

### 2. Create a validator and parse the request

```go
// Create validator
validator, err := NewValidator(ValidatorOpts{})
if err != nil {
    log.Fatal(err)
}

// Parse HTTP request
var user User
err = validator.Validate(httpRequest, &user)
if err != nil {
    log.Printf("Validation failed: %v", err)
    return
}

// Use the parsed and validated user
fmt.Printf("User: %+v\n", user)
```

## Supported Tags and Modifiers

### Source Tags

- `json:"fieldname"` - Parse from JSON request body
- `header:"Header-Name"` - Parse from HTTP headers
- `cookie:"cookie_name"` - Parse from HTTP cookies  
- `query:"param_name"` - Parse from URL query parameters

### Modifiers

- `omitempty` - Continue to next source if not found (e.g., `header:"X-User-ID,omitempty"`)
- `required` - Stop trying other sources, this must succeed (default behavior)

### Priority Order

Sources are tried in the order they appear in the struct tags:

```go
// This will try header first, then cookie, then query, then JSON
Field string `header:"X-Field,omitempty" cookie:"field_cookie,omitempty" query:"field,omitempty" json:"field"`
```

## Advanced Features

### Execution Chain Caching

The library builds execution chains once per struct type and caches them for performance:

```go
// First call builds and caches the chain
validator.Validate(req1, &user1) 

// Subsequent calls reuse the cached chain
validator.Validate(req2, &user2) // Faster!
```

### Lazy JSON Parsing

JSON body is only parsed when needed:

```go
type UserPartial struct {
    Name  string `query:"name"`     // No JSON parsing needed
    Email string `json:"email"`     // Triggers JSON parsing once
    Age   int    `json:"age"`       // Reuses already parsed JSON
}
```

### Custom Source Parsers

The function-based execution chain design makes it extremely easy to create custom parsers. You only need to implement one source-specific function - all the linked list traversal, field setting, and error handling is handled by the `BaseExecutionChain`.

Here's a complete example of a custom parser for `map[string]string` sources:

```go
type MapSourceParser struct {
    chains     map[reflect.Type]*BaseExecutionChain
    chainMutex sync.RWMutex
}

func (msp *MapSourceParser) getValueFromMap(sourceData any, source FieldSource) (any, bool, error) {
    mapData, ok := sourceData.(map[string]string)
    if !ok {
        return nil, false, fmt.Errorf("expected map[string]string, got %T", sourceData)
    }

    value, exists := mapData[source.Key]
    if !exists {
        return nil, false, nil
    }

    return value, true, nil
}

func (msp *MapSourceParser) BuildParseChain(t reflect.Type) (*BaseExecutionChain, error) {
    // ... build ParseSteps for your source type ...
    
    // The key insight: just provide your getter function!
    execChain := &BaseExecutionChain{
        StructType:   t,
        Head:         head,
        SourceGetter: msp.getValueFromMap, // <-- Only this is source-specific!
    }
    
    return execChain, nil
}
```

Usage:
```go
type Config struct {
    DatabaseURL string `mapvalue:"db_url"`
    Port        int    `mapvalue:"port,omitempty"`
}

// Register the custom parser
validator, err := NewValidator(ValidatorOpts{
    Parsers: []SourceParser{NewMapSourceParser()},
})

// Use it
configMap := map[string]string{"db_url": "postgresql://...", "port": "8080"}
var config Config
err = validator.Validate(configMap, &config)
```

#### Benefits of the Function-Based Design

1. **No Duplication**: All parsers share the same linked list traversal logic
2. **Single Responsibility**: Each parser only implements value extraction
3. **Type Safety**: The `SourceGetter` function provides compile-time guarantees
4. **Consistent Behavior**: Error handling, field setting, and validation flow is identical across all parsers
5. **Easy Testing**: You can test your `getValueFromSource` function in isolation
```

## Performance

The execution chain system provides excellent performance:

- **Chain Building**: Done once per struct type, cached thereafter
- **No Reflection in Hot Path**: Field access uses pre-computed indices
- **Lazy Parsing**: Only parse what's needed when it's needed
- **Memory Efficient**: Minimal allocations during parsing

Benchmark results:
```
BenchmarkParseChainExecution-24    	  724672	      1692 ns/op	    2130 B/op	      22 allocs/op
```

## Error Handling

The library provides detailed error information:

```go
err := validator.Validate(request, &user)
if err != nil {
    switch e := err.(type) {
    case ValidationError:
        // Handle validation-specific errors
        log.Printf("Validation failed: %s", e.Error())
    default:
        // Handle other errors
        log.Printf("Parse error: %v", err)
    }
}
```

## Examples

### Multi-source HTTP Request

```go
// HTTP Request with multiple data sources
POST /users?name=QueryName&session=sess123 HTTP/1.1
X-User-ID: 456e7890-e89b-12d3-a456-426614174001
Authorization: Bearer secret-token
Cookie: session_id=cookie-session-123
Content-Type: application/json

{"id": "123e4567-e89b-12d3-a456-426614174000", "email": "john@example.com", "age": 30}

// Results in:
User{
    ID: "456e7890-e89b-12d3-a456-426614174001", // from header (first priority)
    Name: "QueryName",                            // from query (first priority)  
    Email: "john@example.com",                    // from JSON (only source)
    Token: "secret-token",                        // from header (Bearer removed)
    SessionID: "cookie-session-123",              // from cookie (first priority)
    Age: 30,                                      // from JSON
}
```

### Fallback Behavior

```go
// Request missing header and cookie data
GET /users?session=query-session&name=TestUser HTTP/1.1

// Results in:
User{
    SessionID: "query-session",  // fell back to query parameter
    Name: "TestUser",            // from query
    // Other fields are zero values (omitempty allows this)
}
```

## Thread Safety

The validator is thread-safe and can be used concurrently:

```go
// Safe to use the same validator instance across goroutines
go func() { validator.Validate(req1, &user1) }()
go func() { validator.Validate(req2, &user2) }()
```
