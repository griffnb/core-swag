# Schema Builder Service

The Schema Builder Service handles building and managing OpenAPI schemas from Go types.

## Overview

This service is responsible for:
- Building OpenAPI schema definitions from Go type specifications
- Resolving schema references ($ref)
- Managing the definitions map
- Cleaning up unused definitions
- Handling type conversions and mappings

## Files

- **builder.go** (75 lines) - Schema construction and definition management
- **types.go** (160 lines) - Type system utilities and Go-to-OpenAPI type mapping
- **reference.go** (40 lines) - Reference resolution logic
- **cleanup.go** (205 lines) - Unused definition removal

Total: ~480 lines across 4 focused files

## Usage

```go
import (
    "github.com/griffnb/core-swag/internal/schema"
    "github.com/go-openapi/spec"
)

// Create a new schema builder
builder := schema.NewBuilder()

// Build a schema from a type spec
typeSpec := registry.FindTypeSpec("User", astFile)
schema, err := builder.BuildSchema(typeSpec)
if err != nil {
    // Handle error
}

// Resolve all references
err = builder.ResolveReferences()
if err != nil {
    // Handle error
}

// Clean up unused definitions
swagger := &spec.Swagger{
    Paths: paths,
    Definitions: builder.Definitions(),
}
builder.CleanupUnusedDefinitions(swagger)
```

## Key Methods

### NewBuilder

```go
func NewBuilder() *Builder
```

Creates a new schema builder with an empty definitions map.

### BuildSchema

```go
func (b *Builder) BuildSchema(typeSpec *domain.TypeSpecDef) (*spec.Schema, error)
```

Builds an OpenAPI schema from a Go type specification. Returns a schema object that may contain references to definitions.

### ResolveReferences

```go
func (b *Builder) ResolveReferences() error
```

Resolves all $ref references in the definitions map, ensuring all referenced types exist.

### CleanupUnusedDefinitions

```go
func (b *Builder) CleanupUnusedDefinitions(swagger *spec.Swagger)
```

Removes definitions that are not referenced by any path or other definition. This optimizes the final swagger specification.

### Definitions

```go
func (b *Builder) Definitions() spec.Definitions
```

Returns the definitions map containing all schemas.

## Type Mapping

The schema builder maps Go types to OpenAPI types:

### Primitive Types

| Go Type | OpenAPI Type | Format |
|---------|--------------|--------|
| string | string | - |
| bool | boolean | - |
| int | integer | int32 |
| int8 | integer | int32 |
| int16 | integer | int32 |
| int32 | integer | int32 |
| int64 | integer | int64 |
| uint | integer | int32 |
| uint8 | integer | int32 |
| uint16 | integer | int32 |
| uint32 | integer | int32 |
| uint64 | integer | int64 |
| float32 | number | float |
| float64 | number | double |
| byte | integer | int32 |
| rune | integer | int32 |

### Complex Types

| Go Type | OpenAPI Type | Notes |
|---------|--------------|-------|
| []T | array | Items schema for T |
| map[K]V | object | AdditionalProperties schema for V |
| struct | object | Properties for each field |
| *T | - | Dereferences to T |
| interface{} | object | Generic object |
| any | object | Generic object |

### Special Types

| Go Type | OpenAPI Type | Format |
|---------|--------------|--------|
| time.Time | string | date-time |
| time.Duration | integer | int64 |
| json.RawMessage | object | - |
| uuid.UUID | string | uuid |

## Reference Resolution

### Creating References

When building schemas for named types, the builder creates references:

```go
type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}
```

Generates:
```json
{
  "$ref": "#/definitions/User"
}
```

And adds the definition:
```json
{
  "User": {
    "type": "object",
    "properties": {
      "id": {"type": "integer"},
      "name": {"type": "string"}
    }
  }
}
```

### Resolving References

The `ResolveReferences` method ensures:
1. All $ref pointers have corresponding definitions
2. Circular references are handled correctly
3. Missing definitions cause errors

## Cleanup Process

The cleanup process removes unused definitions:

1. **Mark Phase**: Walk all paths and definitions, marking referenced schemas
2. **Sweep Phase**: Remove definitions that were not marked
3. **Repeat**: Repeat until no more definitions are removed (handles transitive references)

### Example

Before cleanup:
```json
{
  "definitions": {
    "User": { ... },          // Referenced by /users path
    "Post": { ... },          // Referenced by /posts path
    "Internal": { ... }       // Not referenced anywhere
  }
}
```

After cleanup:
```json
{
  "definitions": {
    "User": { ... },
    "Post": { ... }
  }
}
```

## Integration

The schema builder is used throughout the parsing process:

### During Type Registration

```go
// Registry parses types and uses builder
registry := registry.NewService()
builder := schema.NewBuilder()

// For each type found:
schema, err := builder.BuildSchema(typeSpec)
definitions[typeName] = schema
```

### During Route Parsing

```go
// Route parser uses builder for request/response schemas
routeParser := route.NewService(registry, structParser, swagger, options)

// When parsing @param body:
schema, err := builder.BuildSchema(paramTypeSpec)
operation.Parameters = append(operation.Parameters, spec.Parameter{
    Schema: schema,
})
```

### Final Cleanup

```go
// After all parsing is complete:
builder.ResolveReferences()
builder.CleanupUnusedDefinitions(swagger)
```

## Testing

Comprehensive tests are provided in the following test files:

```bash
go test ./internal/schema/...
```

Test files:
- **builder_test.go**: Schema building tests
- **types_test.go**: Type mapping tests
- **reference_test.go**: Reference resolution tests
- **cleanup_test.go**: Unused definition removal tests

## Design Principles

1. **Separation of Concerns**: Each file handles one aspect (building, types, references, cleanup)
2. **Immutability**: Schemas are built once and not modified
3. **Explicit Errors**: All errors include context about what failed
4. **Efficient Cleanup**: Mark-and-sweep algorithm for unused definitions
5. **Testability**: Each component is independently testable

## Common Use Cases

### Building a Simple Schema

```go
builder := schema.NewBuilder()

typeSpec := &domain.TypeSpecDef{
    Name: "User",
    // ... other fields
}

schema, err := builder.BuildSchema(typeSpec)
```

### Building Schemas for All Types

```go
builder := schema.NewBuilder()
registry := registry.NewService()

// Register all files
for _, file := range files {
    registry.ParseFile(pkgPath, fileName, src, swag.ParseAll)
}

// Parse all types
schemas, err := registry.ParseTypes()
```

### Optimizing Final Output

```go
// After all parsing
builder.ResolveReferences()
builder.CleanupUnusedDefinitions(swagger)

// swagger.Definitions now contains only referenced types
```

## Error Handling

Common errors:

- **Missing Definition**: A $ref points to a non-existent definition
- **Circular Reference**: Types reference each other in a cycle
- **Invalid Type**: Go type cannot be mapped to OpenAPI type

All errors include:
- Context about what was being processed
- The problematic type or reference
- Suggestions for fixing the issue

## Performance Considerations

### Caching

The builder caches:
- Built schemas to avoid rebuilding the same type
- Resolved references to avoid repeated lookups

### Cleanup Efficiency

The cleanup algorithm:
- Runs in O(n) time where n is the number of definitions
- May repeat multiple times for transitive references
- Typically completes in 2-3 passes

### Memory Usage

The builder stores:
- All definitions in memory
- Cache of built schemas
- Marks during cleanup phase

For large projects with thousands of types, memory usage is typically under 100MB.

## Further Reading

For more details about the overall architecture, see [ARCHITECTURE.md](../../ARCHITECTURE.md).

For information about how structs are parsed into schemas, see [Struct Parser](../parser/struct/README.md).
