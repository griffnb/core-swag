# Centralize Custom Type Registry

**Date**: 2026-03-20
**Status**: Approved

## Problem

Custom type mappings (UUID, Decimal, Time, URN, etc.) are duplicated across 4 files, each with their own hardcoded map/switch statements. Adding a new custom type requires touching all 4 files.

**Affected files:**

| File | Duplicated Logic |
|------|-----------------|
| `internal/domain/utils.go` | `IsExtendedPrimitiveType()`, `TransToValidPrimitiveSchema()` |
| `internal/model/struct_field.go` | `IsPrimitive()` map, `primitiveTypeToSchema()`, `getPrimitiveSchemaForFieldType()` |
| `internal/parser/route/response.go` | `convertTypeToSchemaType()` |
| `internal/parser/route/schema.go` | `isModelType()` extended primitives map |

## Solution: Hybrid Registry

New package `internal/typeregistry/` with two files:

### 1. Simple Type Registry (`registry.go`)

A flat map of custom types to their OpenAPI schema type and format.

```go
type TypeEntry struct {
    SchemaType string // "string", "number", "integer", "boolean", "object"
    Format     string // "uuid", "date-time", "uri", "byte", ""
}
```

**Registered types:**

| Type Name | SchemaType | Format |
|-----------|-----------|--------|
| `time.Time` | string | date-time |
| `types.UUID` | string | uuid |
| `uuid.UUID` | string | uuid |
| `github.com/google/uuid.UUID` | string | uuid |
| `github.com/griffnb/core/lib/types.UUID` | string | uuid |
| `types.URN` | string | uri |
| `github.com/griffnb/core/lib/types.URN` | string | uri |
| `decimal.Decimal` | number | |
| `github.com/shopspring/decimal.Decimal` | number | |
| `json.RawMessage` | object | |
| `encoding/json.RawMessage` | object | |
| `[]byte` | string | byte |
| `[]uint8` | string | byte |

**Pointer handling:** `Lookup` and all public functions strip leading `*` before matching. Callers do not need to handle pointer prefixes.

**Public API:**

- `Lookup(typeName string) (TypeEntry, bool)` - returns the entry if known (strips `*` prefix)
- `IsExtendedPrimitive(typeName string) bool` - replaces scattered checks
- `ToSchema(typeName string) *spec.Schema` - builds spec.Schema with type+format

### 2. Fields Wrapper Helper (`fields.go`)

Handles `fields.*` wrapper types which require generic type extraction and enum detection.

```go
type FieldsResult struct {
    Schema             *spec.Schema
    InnerType          string // extracted generic type for StructField[T], ConstantField[T]
    IsEnum             bool   // true for IntConstantField[T], StringConstantField[T]
    FallbackSchemaType string // "integer" or "string" — used when enum lookup fails
}
```

**Supported wrappers:**

| Wrapper | Result |
|---------|--------|
| `fields.UUIDField` | schema: string/uuid |
| `fields.StringField` | schema: string |
| `fields.IntField` | schema: integer |
| `fields.BoolField` | schema: boolean |
| `fields.FloatField` | schema: number |
| `fields.DecimalField` | schema: integer |
| `fields.TimeField` | schema: string/date-time |
| `fields.StructField[T]` | InnerType: "T", caller resolves recursively |
| `fields.IntConstantField[T]` | InnerType: "T", IsEnum: true |
| `fields.StringConstantField[T]` | InnerType: "T", IsEnum: true |

**Public API:**

- `ResolveFieldsWrapper(typeName string) (*FieldsResult, bool)` - returns nil/false if not a wrapper
- `IsFieldsWrapper(typeName string) bool` - quick check
- `ExtractConstantFieldEnumType(typeStr string) string` - extracts type parameter from generic brackets (moved from `struct_field.go`)

The generic type extraction logic (`extractConstantFieldEnumType`) moves here from `struct_field.go`. Enum lookup stays with the caller since it depends on parser/registry context.

**Fallback behavior:** For unknown `fields.*[T]` generic types not in the table above, `ResolveFieldsWrapper` returns a default `string` schema with `InnerType` set — matching current behavior in `getPrimitiveSchemaForFieldType`.

**Note on `fields.StructField[T]`:** Currently handled via `IsGeneric()` + `GenericTypeArg()` in the `ToSpecSchema` code path, not via `getPrimitiveSchemaForFieldType`. `ResolveFieldsWrapper` will return `InnerType: "T"` with a nil Schema, signaling the caller to resolve recursively through its existing generic handling path.

## Consumer Refactoring

Each of the 4 files delegates to `typeregistry` but keeps its own orchestration logic (pointer stripping, enum resolution, $ref building). We centralize the **data**, not the **control flow**.

**`internal/domain/utils.go`:**
- `IsExtendedPrimitiveType()` -> delegates to `typeregistry.IsExtendedPrimitive()`
- `TransToValidPrimitiveSchema()` -> delegates to `typeregistry.ToSchema()` for custom types

**`internal/model/struct_field.go`:**
- `IsPrimitive()` map -> replaces custom type entries with `typeregistry.Lookup()`
- `primitiveTypeToSchema()` -> delegates custom type cases to `typeregistry.ToSchema()`
- `getPrimitiveSchemaForFieldType()` -> delegates to `typeregistry.ResolveFieldsWrapper()`

**`internal/parser/route/response.go`:**
- `convertTypeToSchemaType()` -> uses `typeregistry.Lookup()` for SchemaType

**`internal/parser/route/schema.go`:**
- `isModelType()` -> uses `typeregistry.IsExtendedPrimitive()` for custom types; keeps its own handling of special keywords (`any`, `interface{}`, `object`, `array`, `file`) since those are not extended primitives

## Testing

- **`internal/typeregistry/registry_test.go`** - unit tests for Lookup, IsExtendedPrimitive, ToSchema
- **`internal/typeregistry/fields_test.go`** - unit tests for ResolveFieldsWrapper, IsFieldsWrapper, generic extraction
- **Existing tests unchanged** - integration tests (`make test-project-1`, `make test-project-2`) validate no behavioral changes
- **No new integration tests** - pure internal refactor

## Notes

- **This is a strict consolidation.** The registry contains exactly the same type mappings that exist today across the 4 files. We are not adding new type coverage — only centralizing existing coverage.
- **`json.RawMessage`** is currently only handled in `struct_field.go`, not in `response.go` or `schema.go`. The registry will contain it, but the consumer refactoring will only use it where it was already used. No behavior expansion.
