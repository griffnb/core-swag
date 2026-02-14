---
paths:
  - "internal/schema/**/*.go"
---

# Schema Builder Service

## Overview

The Schema Builder Service handles OpenAPI schema construction and definition management. It builds schemas from TypeSpecDef objects, manages schema definitions, and tracks already-parsed schemas to avoid duplication.

## Key Structs/Methods

### Core Types

- [BuilderService](../../../../internal/schema/builder.go#L10) - Main schema builder managing definitions and parsed schemas
- [Schema](../../../../internal/domain/types.go#L20) - OpenAPI schema with package path and name metadata

### Service Creation

- [NewBuilder()](../../../../internal/schema/builder.go#L16) - Creates new schema builder instance

### Schema Building

- [BuildSchema(typeSpec)](../../../../internal/schema/builder.go#L25) - Builds OpenAPI schema from TypeSpecDef
- [AddDefinition(name, schema)](../../../../internal/schema/builder.go#L53) - Adds a schema definition manually
- [GetDefinition(name)](../../../../internal/schema/builder.go#L60) - Retrieves schema definition by name
- [Definitions()](../../../../internal/schema/builder.go#L66) - Returns all schema definitions

## Related Packages

### Depends On
- `github.com/go-openapi/spec` - OpenAPI specification types
- [internal/domain](../../../../internal/domain) - Domain types (TypeSpecDef, Schema)

### Used By
- [parser.go](../../../../parser.go) - Main parser uses schema builder for definitions
- [internal/parser/struct](../../../../internal/parser/struct) - Struct parser builds schemas
- [internal/registry](../../../../internal/registry) - Registry service generates schemas

## Docs

No dedicated README exists yet. The package is documented via godoc comments.

## Related Skills

No specific skills are directly related to this internal package. This is a foundational service used for schema generation.

## Usage Example

```go
// Create schema builder
builder := schema.NewBuilder()

// Build schema from type definition
schemaName, err := builder.BuildSchema(typeDef)
if err != nil {
    return err
}

// Add custom definition
customSchema := spec.Schema{
    SchemaProps: spec.SchemaProps{
        Type: []string{"object"},
        Properties: map[string]spec.Schema{
            "id": {
                SchemaProps: spec.SchemaProps{
                    Type: []string{"integer"},
                },
            },
        },
    },
}
err = builder.AddDefinition("CustomType", customSchema)

// Retrieve definition
schema, found := builder.GetDefinition("User")
if found {
    fmt.Printf("Schema type: %v\n", schema.Type)
}

// Get all definitions for swagger spec
allDefinitions := builder.Definitions()
swagger.Definitions = allDefinitions
```

## Design Principles

1. **Simple Interface**: Minimal API surface with clear purpose
2. **Deduplication**: Tracks parsed schemas to avoid building same type multiple times
3. **Stateful Management**: Maintains definitions map for entire generation process
4. **Separation**: Isolated from parsing logic - only handles schema construction
5. **Extensibility**: Allows manual addition of custom schema definitions

## Common Patterns

- Create single builder instance per swagger generation session
- Use `BuildSchema()` for TypeSpecDef conversion to OpenAPI schema
- Check return value from `BuildSchema()` - returns cached name if already built
- Use `Definitions()` to get final map for swagger.Definitions assignment
- Add overrides or custom schemas with `AddDefinition()` before generation
