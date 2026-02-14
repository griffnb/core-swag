---
paths:
  - "internal/parser/route/**/*.go"
---

# Route Parser Service

## Overview

The Route Parser Service extracts HTTP route information from Go function documentation comments. It parses annotations including @router, @param, @success, @failure, @header, @accept, @produce, @security, @tags, and other route-related annotations to generate OpenAPI operation specifications.

## Key Structs/Methods

### Core Types

- [Service](../../../../internal/parser/route/service.go#L14) - Main route parser service
- [Route](../../../../internal/parser/route/domain/types.go) - Represents a single HTTP route with full operation details
- [Parameter](../../../../internal/parser/route/domain/types.go) - Route parameter (path, query, header, body)
- [Response](../../../../internal/parser/route/domain/types.go) - Response specification with status code

### Service Creation

- [NewService(registry, structParser)](../../../../internal/parser/route/service.go#L22) - Creates new route parser instance

### Main Parsing Methods

- [ParseRoutes(astFile)](../../../../internal/parser/route/service.go#L30) - Extracts all routes from AST file
- [parseOperation(funcDecl)](../../../../internal/parser/route/service.go#L56) - Parses function declaration into operation
- [operationToRoutes(op)](../../../../internal/parser/route/service.go#L85) - Converts operation to route objects
- [parseComment(op, text)](../../../../internal/parser/route/comments.go) - Parses individual comment annotation

### Annotation Parsers

- [parseRouter(op, value)](../../../../internal/parser/route/router.go) - Parses @router annotation
- [parseParam(op, value)](../../../../internal/parser/route/params.go) - Parses @param annotation
- [parseSuccess(op, value)](../../../../internal/parser/route/responses.go) - Parses @success annotation
- [parseFailure(op, value)](../../../../internal/parser/route/responses.go) - Parses @failure annotation
- [parseHeader(op, value)](../../../../internal/parser/route/headers.go) - Parses @header annotation
- [parseSecurity(op, value)](../../../../internal/parser/route/security.go) - Parses @security annotation

## Related Packages

### Depends On
- `go/ast` - AST function declaration representation
- [internal/domain](../../../../internal/domain) - Domain types
- [internal/parser/field](../../../../internal/parser/field) - Parameter parsing utilities
- [internal/registry](../../../../internal/registry) - Type lookup for request/response models
- [internal/parser/struct](../../../../internal/parser/struct) - Schema generation for body parameters

### Used By
- [parser.go](../../../../parser.go) - Main parser uses route service to extract API operations
- [gen/gen.go](../../../../gen/gen.go) - Code generator uses parsed routes

## Docs

- [Route Package README](../../../../internal/parser/route/README.md) - Package documentation (if exists)

## Related Skills

No specific skills are directly related to this internal package.

## Usage Example

```go
// Create route parser with dependencies
routeParser := route.NewService(registryService, structParser)

// Parse all routes from file
routes, err := routeParser.ParseRoutes(astFile)
if err != nil {
    return err
}

// Process each route
for _, route := range routes {
    fmt.Printf("%s %s - %s\n",
        route.Method,
        route.Path,
        route.Summary)

    // Access parameters
    for _, param := range route.Parameters {
        fmt.Printf("  Param: %s in %s\n",
            param.Name,
            param.In)
    }

    // Access responses
    for code, response := range route.Responses {
        fmt.Printf("  Response %d: %s\n",
            code,
            response.Description)
    }
}
```

## Supported Annotations

### Route Definition
- `@router <path> [method]` - Defines HTTP route (required)
- `@summary` - Short operation summary
- `@description` - Detailed operation description
- `@id` - Unique operation ID
- `@tags` - Comma-separated tag list

### Parameters
- `@param <name> <in> <type> <required> <description>` - Parameter definition
  - `in`: path, query, header, body, formData
  - `type`: primitive types, array, object, or model reference
  - `required`: true/false

### Responses
- `@success <code> {<type>} <description>` - Success response
- `@failure <code> {<type>} <description>` - Error response
- `@header <code> {<type>} <name> <description>` - Response header

### Content Types
- `@accept` - Accepted request content types (overrides global)
- `@produce` - Produced response content types (overrides global)

### Security
- `@security <scheme> [scope1,scope2,...]` - Security requirement

### Misc
- `@deprecated` - Marks operation as deprecated
- `@x-*` - Custom extensions

## Design Principles

1. **Comment-Driven**: All route metadata extracted from function documentation
2. **Function-Based**: Each exported function with @router becomes an operation
3. **Multiple Routes**: One function can define multiple routes (different methods)
4. **Type Resolution**: Uses registry to resolve model types in parameters/responses
5. **Override Support**: Route-level accept/produce overrides global settings
6. **Security Composition**: Supports multiple security schemes per route
7. **Error Recovery**: Continues parsing if individual comment fails

## Common Patterns

- Always include @router annotation for function to be parsed
- Define @param for each path parameter, query param, and request body
- Use @success and @failure to document all possible responses
- Reference models with {model.Name} syntax in parameters and responses
- Use @tags to group related operations
- Set @id explicitly for stable operation IDs across refactors
- Combine multiple @security annotations for AND logic
- Use pipe (|) in @security for OR logic

## Parameter Formats

```
@param name       path     string  true  "User ID"
@param query      query    string  false "Search query"
@param body       body     User    true  "User object"
@param header     header   string  false "Authorization header"
@param formData   formData file    true  "Upload file"
```

## Response Formats

```
@success 200 {object} model.User         "Success response"
@success 201 {string} string             "Created"
@failure 400 {object} model.ErrorResponse "Bad request"
@failure 404 {string} string              "Not found"
```

## Route Path Syntax

- Supports path parameters: `/users/{id}`
- Supports wildcards: `/files/*filepath`
- Multiple routes: Define multiple @router annotations

## Implementation Status

**Current Status**: Service is scaffolded with basic structure. Core parsing logic is partially implemented.

**TODO**:
- Complete annotation parsers (params, responses, headers)
- Implement type resolution for model references
- Add support for array parameters
- Handle nested object parameters
- Support file upload parameters
- Implement security parsing
- Add validation for parameter definitions
- Support response schemas with examples
