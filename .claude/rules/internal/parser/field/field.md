---
paths:
  - "internal/parser/field/**/*.go"
---

# Field Parser Package

## Overview

The Field Parser package provides utilities and interfaces for parsing struct fields into OpenAPI schemas. It was created to resolve import cycle issues by extracting field parsing logic, naming strategies, tag parsing, and helper functions into a separate package shared by both struct and route parsers.

## Key Constants & Types

### Schema Type Constants

- [ARRAY, OBJECT, PRIMITIVE](../../../../internal/parser/field/types.go#L5) - Schema type identifiers
- [BOOLEAN, INTEGER, NUMBER, STRING](../../../../internal/parser/field/types.go#L11) - Primitive type constants

### Naming Strategy Constants

- [CamelCase](../../../../internal/parser/field/types.go#L24) - camelCase naming strategy
- [PascalCase](../../../../internal/parser/field/types.go#L26) - PascalCase naming strategy
- [SnakeCase](../../../../internal/parser/field/types.go#L28) - snake_case naming strategy

### Tag Names

Field tags used for schema generation (lines 32-56 in types.go):
- `json`, `form`, `header`, `uri`, `binding`, `validate` - Standard tags
- `swaggertype`, `swaggerignore` - Swagger control tags
- `format`, `title`, `enums`, `example`, `default` - Schema property tags
- `maximum`, `minimum`, `maxLength`, `minLength`, `multipleOf` - Validation tags
- `readonly`, `extensions` - Additional property tags

### Key Functions

- [GetFieldName(field, strategy)](../../../../internal/parser/field/naming.go) - Gets field name using naming strategy
- [ParseTag(field, tagName)](../../../../internal/parser/field/tags.go) - Parses struct tag value
- [IsRequired(field, requiredByDefault)](../../../../internal/parser/field/tags.go) - Determines if field is required
- [IsIgnored(field)](../../../../internal/parser/field/tags.go) - Checks if field should be ignored
- [GetValidationTags(field)](../../../../internal/parser/field/tags.go) - Extracts validation constraints

### Helper Functions

- [SplitNotWrapped(s, sep)](../../../../internal/parser/field/helpers.go) - Splits string respecting brackets/quotes
- [TransToValidSchemeType(typeName)](../../../../internal/parser/field/helpers.go) - Converts Go type to schema type
- [CheckSchemaType(typeName)](../../../../internal/parser/field/helpers.go) - Validates schema type name

## Related Packages

### Depends On
- `go/ast` - AST field representation
- `reflect` - Struct tag parsing
- `strings` - String manipulation

### Used By
- [internal/parser/struct](../../../../internal/parser/struct) - Uses field parsing for struct schemas
- [internal/parser/route](../../../../internal/parser/route) - Uses field parsing for parameter extraction
- [parser.go](../../../../parser.go) - Main parser uses field utilities

## Docs

No dedicated README exists. Package created during refactoring Phase 7 to break import cycles.

## Related Skills

No specific skills are directly related to this internal package.

## Usage Example

```go
import (
    "github.com/griffnb/core-swag/internal/parser/field"
    "go/ast"
)

// Get field name with strategy
func processField(astField *ast.Field) {
    // Use camelCase naming
    name := field.GetFieldName(astField, field.CamelCase)

    // Check if field should be ignored
    if field.IsIgnored(astField) {
        return
    }

    // Parse JSON tag
    jsonTag := field.ParseTag(astField, field.jsonTag)

    // Check required status
    required := field.IsRequired(astField, true)

    // Get validation constraints
    validations := field.GetValidationTags(astField)

    // Convert Go type to schema type
    schemaType := field.TransToValidSchemeType("int64")
    // Returns: field.INTEGER
}

// Parse complex tag values
tagValue := "name,omitempty,required"
parts := field.SplitNotWrapped(tagValue, ',')
// Returns: ["name", "omitempty", "required"]
```

## Design Principles

1. **Import Cycle Resolution**: Extracted from struct parser to break circular dependencies
2. **Reusability**: Shared utilities used by multiple parser components
3. **Constants Over Strings**: Uses typed constants for type safety
4. **Strategy Pattern**: Naming strategies allow flexible field name generation
5. **Tag-Based Configuration**: Leverages Go struct tags for schema metadata
6. **Validation Support**: Extracts validation rules from binding/validate tags

## Common Patterns

- Use naming strategy constants (CamelCase, PascalCase, SnakeCase) instead of strings
- Check `IsIgnored()` before processing fields to skip internal fields
- Parse all relevant tags (json, form, binding, validate) for complete schema info
- Use `SplitNotWrapped()` for tag values that may contain nested delimiters
- Apply validation tags (maximum, minimum, maxLength) to schema properties
- Handle `omitempty` tag to mark fields as optional in schema

## Tag Processing Order

1. **Ignore Check**: Check `swaggerignore` tag first
2. **Name Resolution**: Parse json/form tag for field name, apply naming strategy
3. **Type Override**: Check `swaggertype` tag for custom type mapping
4. **Required Status**: Check `required`/`optional` tags and binding validation
5. **Validation Rules**: Extract validation constraints from tags
6. **Schema Properties**: Apply format, title, example, default, enums
7. **Constraints**: Apply maximum, minimum, maxLength, minLength, multipleOf
8. **Extensions**: Parse custom x-* extensions from `extensions` tag

## Naming Strategy Behavior

- **CamelCase**: firstName → firstName
- **PascalCase**: firstName → FirstName
- **SnakeCase**: firstName → first_name
- **Default**: Uses json tag if present, otherwise field name as-is
