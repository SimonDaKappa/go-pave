# Overview

PAVE (Parse and Validate Everything) is a performant go library for parsing any data type into another.

It is designed with the following core concepts and motivations in mind:
- Under high loads, API's often spend the majority of their time parsing input data into concrete types (i.e json, gRPC, AMQP)
- Once you've seen how to parse something once, you should do it faster the next time.
- Take advantage of the language features available
- Provide a framework to easily implement efficient custom parsers with the minimal amount of code.
- Parsing should be flexible, allow for dynamic fallbacks, and developer controlled.

As such the following design decisions are present at almost every level of code:
- All pave operations must be concurrency safe
- [Struct tags]() define how to interact with a parser
- Cache everything that will be used and has a higher computational cost than retrieval.

# Usage


# Features

## Supported Types
As given by its name, the goal of this package is to be able to parse any input to any destination. That means, this packages end-goal is to support every
base type in Go, common interfaces and composite types, and all structural combinations of the prior.

For any destination struct field, the following primitives are currently supported:
```go
// strings
string
// ints
int, int8, int16, int32, int64
// uints
uint, uint8, uint16, uint32, uint64, uintptr
// floats
float32, float64,
// complex numbers
complex64, complex128,
// booleans
bool
// slices
[]byte
// interface
all interfaces
```
With many (many) more planned in the future!

Additionally, the following special struct types are already integrated
```go
uuid.UUID{}
time.Time{}
```

Lastly, any type that implements one of the following interfaces:
```go
encoding.TextUnmarshaller
```

## Struct Tags
Struct tags are the primary way to interact with a Parser. On each field of a destination type, you will define the [Binding(s)]() that the parser will use to populate the field

All parsers will use the tag grammar shown [here]().
In particular, parsers will either support either
- One binding type for a source, or
- Multiple bindings for a source

To determine which one to use, see the documentation for the parser you want

###  Tag Grammar

```
Tag grammar:
    <field> <type> <tag>
field:
    <Go Literal>
type:
    <Go Literal>
tag:
    <parse_tag> <validate_tag>'

parse_tag:
    // Any ordering of elements is allowed.
    (binding_tag_list)? (optional_tag_list)? 

binding_tag_list:
   [<binding_tag>]^*
binding_tag:
    <binding_name>:"<binding_identifier>,<binding_modifier_list>"
binding_name, binding_identifier:
    <string>
    
binding_modifier_list:
    // Delimited with "," end-delim optional
    [<binding_modifier>]^* 
binding_modifier:
    omitempty | omiterror | omitnil | <modifier_custom>
modifier_custom:
   <parser_specific>

optional_tag_list:
   [<optional_tag>]^*
optional_tag:
    <default_tag> | <recursive_tag> | <custom_tag>
custom_tag:
    <parser_specific>
default_tag:
    default:"<string>"
recursive_tag:
    recursive:"<bool>"

validate_tag
    validate:"<...>" | nil
```

As an example, say you have a type corresponding to a HTTP API request that needs data from the body, query URL, and cookies 
such as:
```go
type ExampleRequestWithSession struct {
	ses string `header:"Session-Token,omitempty" query:"session-token,omitempty" default:"invalid_session"`
	jwt string `cookie:"jwt"`
	
	metadata struct {
		session_active_since time.Time `json:"metadata.active_since,omitempty"`
		session_expires      time.Time `json:"metadata.expires,omitempty"`	
	}

	resourceID uuid.UUID `query:"rid"`
	origin string `header:"X-Origin"`
	page   uint8 `query:"limit,omitempty"`
	limit  int 'query:"limit,omitempty'
}
```

All of the configuration occurs in the struct definition. To parse an incoming request into `ExampleRequestWithSession`, simply provide the `HTTPRequestParser` with the `*http.Request` and struct instance.

## Caching

## External Library Integrations
WIP

