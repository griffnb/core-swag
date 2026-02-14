# Struct Parser Service

The Struct Parser Service handles parsing of Go structs into OpenAPI schemas.

## Overview

This service converts Go struct definitions into OpenAPI/Swagger schema objects, handling:
- Standard Go structs
- Custom model structs (fields.StructField[T] pattern)
- Field tags (json, validate, swaggertype, swaggerignore, extensions)
- Generic types and type parameters
- Embedded fields
- Public/private field filtering
- Validation constraints
- Examples and defaults

## Files

- **service.go** (55 lines) - Main struct parsing service and orchestration
- **field.go** (500 lines) - Field parsing with comprehensive tag handling

Total: ~555 lines across 2 focused files

## Usage

```go
import (
    "github.com/griffnb/core-swag/internal/parser/struct"
    "github.com/griffnb/core-swag/internal/registry"
    "github.com/griffnb/core-swag/internal/schema"
)

// Create struct parser
registry := registry.NewService()
schemaBuilder := schema.NewBuilder()

parser := struct.NewService(
    registry,
    schemaBuilder,
    &struct.Options{
        PropNamingStrategy: "camelcase",
        RequiredByDefault: false,
    },
)

// Parse a struct
typeSpec := registry.FindTypeSpec("User", astFile)
schema, err := parser.ParseStruct(typeSpec)
if err != nil {
    // Handle error
}

// schema is now an OpenAPI schema object
```

## Supported Features

### Standard Structs

```go
type User struct {
    ID        int       `json:"id" example:"1"`
    Username  string    `json:"username" validate:"required" minLength:"3" maxLength:"50"`
    Email     string    `json:"email" format:"email"`
    CreatedAt time.Time `json:"created_at"`
}
```

Generates:
```json
{
  "type": "object",
  "required": ["username"],
  "properties": {
    "id": {
      "type": "integer",
      "example": 1
    },
    "username": {
      "type": "string",
      "minLength": 3,
      "maxLength": 50
    },
    "email": {
      "type": "string",
      "format": "email"
    },
    "created_at": {
      "type": "string",
      "format": "date-time"
    }
  }
}
```

### Custom Model Structs (fields.StructField[T])

```go
import "your.app/lib/fields"

type Account struct {
    Name    fields.StructField[string] `json:"name" public:"view,edit"`
    Balance fields.StructField[int]    `json:"balance" public:"view"`
    Secret  fields.StructField[string] `json:"secret"`
}
```

With public:"view" tag, generates a schema with only Name and Balance.
With public:"edit" tag, generates a schema with only Name.

### Field Tags

#### JSON Tags

```go
Username string `json:"username,omitempty"`
```

- Controls field name in JSON
- `omitempty` marks field as not required

#### Validate Tags

```go
Age int `validate:"required" minimum:"18" maximum:"120"`
```

- `required` marks field as required
- `minimum`, `maximum` set numeric constraints
- `minLength`, `maxLength` set string length constraints
- `minItems`, `maxItems` set array length constraints

#### Swaggertype Tag

Override the detected type:

```go
RegisterTime TimestampTime `json:"register_time" swaggertype:"primitive,integer"`
Coeffs []big.Float `json:"coeffs" swaggertype:"array,number"`
Data []byte `json:"data" swaggertype:"string" format:"base64"`
```

#### Swaggerignore Tag

Exclude a field from the schema:

```go
InternalField string `swaggerignore:"true"`
```

#### Extensions Tag

Add custom extensions:

```go
ID string `json:"id" extensions:"x-nullable,x-abc=def"`
```

Generates:
```json
{
  "type": "string",
  "x-nullable": true,
  "x-abc": "def"
}
```

#### Example and Default Tags

```go
Status string `json:"status" default:"active" example:"active"`
Count  int    `json:"count" default:"10" example:"42"`
```

#### Enums Tag

```go
Status string `json:"status" enums:"active,inactive,pending"`
```

Generates:
```json
{
  "type": "string",
  "enum": ["active", "inactive", "pending"]
}
```

### Generic Types

```go
type Response[T any] struct {
    Data T `json:"data"`
    Code int `json:"code"`
}

// Usage:
// @Success 200 {object} Response[User]
```

The parser resolves generic type parameters to concrete types.

### Embedded Fields

```go
type BaseModel struct {
    ID        int       `json:"id"`
    CreatedAt time.Time `json:"created_at"`
}

type User struct {
    BaseModel
    Username string `json:"username"`
}
```

Generates a schema with all fields from BaseModel and User.

### Public/Private Filtering

For custom model structs with `public:` tags:

```go
type User struct {
    Username fields.StructField[string] `json:"username" public:"view,edit"`
    Password fields.StructField[string] `json:"password"`
}
```

- Without filtering: All fields included
- With `public:"view"`: Only fields tagged with "view" included
- With `public:"edit"`: Only fields tagged with "edit" included

## Key Methods

### NewService

```go
func NewService(
    registry *registry.Service,
    schemaBuilder *schema.Builder,
    options *Options,
) *Service
```

Creates a new struct parser service.

### ParseStruct

```go
func (s *Service) ParseStruct(typeSpec *domain.TypeSpecDef) (*spec.Schema, error)
```

Parses a struct type specification into an OpenAPI schema.

## Options

```go
type Options struct {
    PropNamingStrategy string // "camelcase", "snakecase", "pascalcase"
    RequiredByDefault  bool   // If true, all fields are required unless omitempty
}
```

## Implementation Details

### Field Parsing Process

1. **Iterate struct fields**: Walk through all fields in the struct
2. **Check visibility**: Skip unexported fields
3. **Parse tags**: Extract json, validate, swaggertype, etc.
4. **Determine type**: Resolve Go type to OpenAPI type
5. **Apply constraints**: Add validation, examples, defaults
6. **Handle embedded**: Recursively process embedded structs
7. **Build schema**: Construct final OpenAPI schema object

### Custom Model Detection

The parser detects custom models by checking if a field's type is `fields.StructField[T]`:

1. Extract inner type `T` from generic parameter
2. Parse inner type into schema
3. Apply public/private filtering based on tags
4. Return filtered schema

### Type Resolution

The parser maps Go types to OpenAPI types:

| Go Type | OpenAPI Type | Format |
|---------|--------------|--------|
| string | string | - |
| int, int8, int16, int32 | integer | int32 |
| int64 | integer | int64 |
| float32 | number | float |
| float64 | number | double |
| bool | boolean | - |
| time.Time | string | date-time |
| []T | array | - |
| map[K]V | object | - |

## Testing

Comprehensive tests are provided in `service_test.go`:

```bash
go test ./internal/parser/struct/...
```

Tests cover:
- Standard struct parsing
- Custom model parsing
- Field tag handling
- Generic type resolution
- Public/private filtering
- Embedded fields
- Edge cases and error handling

## Design Principles

1. **Separation of Concerns**: Service orchestrates, field parser handles details
2. **Extensibility**: Easy to add new field tag types
3. **Recursion Safety**: Tracks parsed structs to prevent infinite loops
4. **Clear Error Messages**: All errors include context about what failed
5. **Testability**: Field parsing logic is isolated and testable

## Integration

The struct parser is integrated into the main parser flow:

1. Registry service discovers and registers all types
2. Registry calls struct parser for each type
3. Struct parser generates schemas
4. Schemas are added to swagger definitions
5. Route parser references schemas in request/response definitions

For more details, see [ARCHITECTURE.md](../../ARCHITECTURE.md).

## Common Patterns

### Model with Validation

```go
type CreateUserRequest struct {
    Username string `json:"username" validate:"required" minLength:"3" maxLength:"50"`
    Email    string `json:"email" validate:"required" format:"email"`
    Age      int    `json:"age" minimum:"18" maximum:"120"`
}
```

### Model with Examples

```go
type User struct {
    ID       int    `json:"id" example:"1"`
    Username string `json:"username" example:"john_doe"`
    Email    string `json:"email" example:"john@example.com"`
}
```

### Model with Custom Types

```go
type Account struct {
    ID        int64              `json:"id"`
    Balance   sql.NullInt64      `json:"balance" swaggertype:"integer"`
    Data      []byte             `json:"data" swaggertype:"string" format:"base64"`
    Timestamp TimestampTime      `json:"timestamp" swaggertype:"primitive,integer"`
}
```

### Partial Models (Custom Fields)

```go
type User struct {
    ID       fields.StructField[int]    `json:"id" public:"view"`
    Username fields.StructField[string] `json:"username" public:"view,edit"`
    Email    fields.StructField[string] `json:"email" public:"view,edit"`
    Password fields.StructField[string] `json:"password" public:"edit"`
    Role     fields.StructField[string] `json:"role" public:"view"`
}
```

This generates different schemas for different contexts:
- View (GET): ID, Username, Email, Role
- Edit (PUT/PATCH): Username, Email, Password
