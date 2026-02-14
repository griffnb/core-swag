---
paths:
  - "internal/parser/struct/**/*.go"
---

# Struct Parser Service

## Overview

The Struct Parser Service handles parsing of Go structs into OpenAPI schema definitions. It supports both standard Go structs and custom model structs with fields.StructField[T] wrappers, handling field extraction, type resolution, validation tags, and nested struct references.

## Key Structs/Methods

### Core Types

- [Service](../../../../internal/parser/struct/service.go#L11) - Main struct parser service

### Service Creation

- [NewService()](../../../../internal/parser/struct/service.go#L25) - Creates new struct parser instance

### Main Parsing Methods

- [ParseStruct(file, fields)](../../../../internal/parser/struct/service.go#L30) - Parses struct FieldList into OpenAPI schema
- [ParseField(file, field)](../../../../internal/parser/struct/service.go#L36) - Parses individual struct field into schema properties
- [ParseDefinition(typeSpec)](../../../../internal/parser/struct/service.go#L42) - Parses type definition (handles both structs and custom models)

## Related Packages

### Depends On
- `go/ast` - AST struct and field representation
- `github.com/go-openapi/spec` - OpenAPI schema types
- [internal/domain](../../../../internal/domain) - TypeSpecDef and domain types
- [internal/parser/field](../../../../internal/parser/field) - Field parsing utilities, naming strategies, tag parsing
- [internal/registry](../../../../internal/registry) - Type lookup and resolution
- [internal/schema](../../../../internal/schema) - Schema building and caching

### Used By
- [parser.go](../../../../parser.go) - Main parser uses struct service for type definitions
- [internal/registry](../../../../internal/registry) - Registry uses struct parser via ParseTypes()
- [internal/parser/route](../../../../internal/parser/route) - Route parser uses struct schemas for request/response bodies

## Docs

No dedicated README exists yet. The service is currently scaffolded with TODOs for future implementation.

## Related Skills

No specific skills are directly related to this internal package.

## Usage Example

```go
// NOTE: This is expected usage once implementation is complete

// Create struct parser with dependencies
structParser := structparser.NewService()
// TODO: Add dependency injection for registry, schema builder, etc.

// Parse a struct definition
schema, err := structParser.ParseStruct(astFile, structType.Fields)
if err != nil {
    return err
}

// Parse individual field
properties, required, err := structParser.ParseField(astFile, field)
if err != nil {
    return err
}

// Parse type definition (entry point from registry)
schema, err := structParser.ParseDefinition(typeSpec)
if err != nil {
    return err
}
```

## Design Principles

1. **Separation of Concerns**: Extracted from main parser into dedicated service
2. **Field Package Integration**: Uses field parser package for tag parsing and naming
3. **Type Resolution**: Delegates to registry service for referenced types
4. **Schema Caching**: Uses schema builder to avoid re-parsing same types
5. **Validation Support**: Extracts validation rules from struct tags
6. **Custom Model Support**: Handles both standard structs and fields.StructField[T] patterns

## Implementation Status

**Current Status**: Service is scaffolded but not fully implemented. Core methods return nil pending future phases.

**TODO**:
- Add dependency injection (registry, schema builder, field parser factory)
- Implement ParseStruct() for field list processing
- Implement ParseField() for individual field schema generation
- Implement ParseDefinition() as entry point from registry
- Add support for nested struct references
- Add support for generic type parameters
- Add support for embedded structs
- Handle validation tag constraints
- Support custom field naming strategies

## Expected Behavior

### ParseStruct
- Iterate through struct fields
- Apply naming strategy (camelCase, PascalCase, snake_case)
- Handle embedded fields (composition)
- Parse struct tags (json, binding, validate)
- Generate property schemas with validation constraints
- Determine required vs optional fields
- Handle custom types and references

### ParseField
- Extract field name using naming strategy
- Check for swaggerignore tag
- Parse field type (primitive, array, object, reference)
- Extract validation rules (required, min, max, pattern, enum)
- Handle nested structs and type references
- Apply format, example, default values
- Return property schemas and required field list

### ParseDefinition
- Entry point called by registry.ParseTypes()
- Dispatch to ParseStruct for struct types
- Handle other type definitions (aliases, interfaces)
- Manage recursive type references (stack protection)
- Add generated schema to schema builder
- Return final schema definition

## Common Patterns (Expected)

- Always use field package utilities for tag parsing
- Check swaggerignore before processing fields
- Use registry.FindTypeSpec() for nested type resolution
- Protect against circular references with parsing stack
- Apply naming strategy consistently across all fields
- Extract validation from both binding and validate tags
- Handle pointer types by marking as optional
- Preserve JSON tag names when present
- Support custom x-* extensions via struct tags
