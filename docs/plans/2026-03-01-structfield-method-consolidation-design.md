# StructField Method Consolidation Design

**Date:** 2026-03-01
**Status:** Approved
**Files:** `internal/model/struct_field.go`, `internal/model/struct_field_lookup.go`

## Problem

`StructField` is a thin data struct. All intelligence about the field lives in standalone private helper functions that take type strings or `types.Type` as parameters. This means:

1. **Redundant analysis** — The same type information is analyzed twice: once during field extraction (`struct_field_lookup.go`) via `checkNamed`/`checkStruct`/`checkSlice`/`checkMap`, and again during schema generation (`struct_field.go`) via `buildSchemaForType` and its helpers, all re-deriving from strings.
2. **Generic detection in 3 places** — `ToSpecSchema:150` checks `strings.Contains(typeStr, "StructField[")`, `ExtractFieldsRecursive:230` checks `strings.Contains(f.Type.String(), "fields.StructField")`, and `processStructField:247` does `strings.Split(f.Type.String(), ".StructField[")`.
3. **Debugging is painful** — When schema generation produces wrong output, you have to trace through 5+ function calls, each receiving partial type info as parameters. The full field context is never in one place.
4. **Hard to extend** — Adding new type handling requires updating multiple standalone functions and passing new parameters through call chains.

## Solution

Move all type analysis functions onto `*StructField` as methods. The field already has `Type` (`types.Type`), `TypeString`, `Tag`, `Name`, and `Fields` — every helper function currently takes subsets of this information as parameters. As methods, they access everything they need through `this`.

## Method Migration Table

### Standalone functions → StructField methods

| Current function | New method | Eliminated params |
|---|---|---|
| `isPrimitiveType(typeStr string)` | `sf.IsPrimitive() bool` | `typeStr` |
| `isAnyType(typeStr string)` | `sf.IsAny() bool` | `typeStr` |
| `isFieldsWrapperType(typeStr string)` | `sf.IsFieldsWrapper() bool` | `typeStr` |
| `isGenericTypeArgStruct(t types.Type)` | `sf.IsGenericTypeArgStruct() bool` | `t` |
| `isUnderlyingStruct(t types.Type)` | `sf.IsUnderlyingStruct() bool` | `t` |
| `normalizeTypeName(typeStr string)` | `sf.NormalizedType() string` | `typeStr` |
| `extractGenericTypeParameter(typeStr string)` | `sf.GenericTypeArg() (string, error)` | `typeStr` |
| `shouldTreatAsSwaggerPrimitive(named *types.Named)` | `sf.IsSwaggerPrimitive() bool` | `named` |
| `extractConstantFieldEnumType(typeStr string)` | `sf.ConstantFieldEnumType() string` | `typeStr` |
| `primitiveTypeToSchema(typeStr string)` | `sf.PrimitiveSchema() *spec.Schema` | `typeStr` |
| `getPrimitiveSchemaForFieldType(typeStr, orig, enumLookup)` | `sf.FieldsWrapperSchema(enumLookup) (*spec.Schema, []string, error)` | `typeStr`, `originalTypeStr` |
| `buildSchemaForType(typeStr, public, forceRequired, origTypeStr, enumLookup)` | `sf.BuildSchema(public, forceRequired, enumLookup) (*spec.Schema, []string, error)` | `typeStr`, `originalTypeStr` |
| inline generic detection (3 places) | `sf.IsGeneric() bool` | scattered string checks |
| `extractTypeParameter(typeStr string)` | removed, duplicate of `GenericTypeArg()` | — |

### New convenience method

| Method | Purpose |
|---|---|
| `sf.EffectiveTypeString() string` | Returns resolved type string: uses `TypeString` if set, falls back to `Type.String()`. Single source of truth for "what type string should I work with?" |

### Stays standalone (no field context)

| Function | Why |
|---|---|
| `applyEnumsToSchema(schema, enums)` | Operates on `spec.Schema` + `[]EnumValue`, no field state needed |
| `resolveRefName(shortName, fullPath)` | Naming convention using global `DefinitionNameResolver` |
| `makeFullPathDefinitionName(fullTypeStr)` | Pure string transform matching `TypeSpecDef.TypeName()` |

## How recursive types work in BuildSchema

`buildSchemaForType` currently recurses on sub-type strings (array elements, map values). As `sf.BuildSchema()`, when it encounters `[]SomeType` or `map[string]SomeType`, it creates a child `StructField` with the element/value type info and calls `.BuildSchema()` on that child:

```go
func (sf *StructField) BuildSchema(public, forceRequired bool, enumLookup TypeEnumLookup) (*spec.Schema, []string, error) {
    typeStr := sf.NormalizedType()

    if sf.IsAny() {
        return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"object"}}}, nil, nil
    }
    if sf.IsFieldsWrapper() {
        return sf.FieldsWrapperSchema(enumLookup)
    }
    if sf.IsPrimitive() {
        return sf.PrimitiveSchema(), nil, nil
    }

    // Arrays: create child StructField for element type
    if strings.HasPrefix(typeStr, "[]") {
        elemField := &StructField{TypeString: elemType, Type: /* derived if possible */}
        elemSchema, nested, err := elemField.BuildSchema(public, forceRequired, enumLookup)
        return spec.ArrayProperty(elemSchema), nested, err
    }

    // Maps: create child StructField for value type
    if strings.HasPrefix(typeStr, "map[") {
        valueField := &StructField{TypeString: valueType}
        valueSchema, nested, err := valueField.BuildSchema(public, forceRequired, enumLookup)
        return spec.MapProperty(valueSchema), nested, err
    }

    // Struct/enum ref handling...
}
```

## Changes to struct_field_lookup.go

The `CoreStructParser` methods (`checkNamed`, `checkStruct`, `checkSlice`, `checkMap`, `processStructField`, `ExtractFieldsRecursive`) keep their role — they need `packageMap` for recursive field extraction. But they stop duplicating type analysis:

- `processStructField` replaces inline `strings.Split(f.Type.String(), ".StructField[")` with `f.IsGeneric()` and `f.GenericTypeArg()`
- `checkNamed` replaces `shouldTreatAsSwaggerPrimitive(named)` with checking via StructField method
- `ExtractFieldsRecursive:230` replaces `strings.Contains(f.Type.String(), "fields.StructField")` with `f.IsGeneric()`

## ToSpecSchema simplification

Current `ToSpecSchema` is ~90 lines with inline type resolution, generic detection, and effective-public computation. After consolidation:

```go
func (sf *StructField) ToSpecSchema(public, forceRequired bool, enumLookup TypeEnumLookup) (string, *spec.Schema, bool, []string, error) {
    // 1. Tag-based filtering (public, swaggerignore, json:"-")
    if public && !sf.IsPublic() {
        return "", nil, false, nil, nil
    }
    tags := sf.GetTags()
    if swaggerIgnore, ok := tags["swaggerignore"]; ok && strings.EqualFold(swaggerIgnore, "true") {
        return "", nil, false, nil, nil
    }

    // 2. Extract property name from json/column tag
    propName, required := sf.extractPropInfo(tags, forceRequired)
    if propName == "" || propName == "-" {
        return "", nil, false, nil, nil
    }

    // 3. Build schema — all type intelligence is in BuildSchema and its method calls
    schema, nestedTypes, err := sf.BuildSchema(public, forceRequired, enumLookup)
    if err != nil {
        return "", nil, false, nil, fmt.Errorf("failed to build schema for field %s: %w", sf.Name, err)
    }

    return propName, schema, required, nestedTypes, nil
}
```

## File organization

All new methods live in `struct_field.go` alongside the StructField definition. No new files. The standalone helpers that remain (`applyEnumsToSchema`, `resolveRefName`, `makeFullPathDefinitionName`) stay in the same file.

## Test impact

- `struct_field_test.go` (~577 lines): Tests that call `buildSchemaForType()` directly will change to create a `StructField` and call `.BuildSchema()`. Same test logic, different call site.
- `struct_builder_test.go` (~1045 lines): Minimal changes — these create `StructField` instances and call `BuildSpecSchema` on `StructBuilder`, which calls `ToSpecSchema`. The interface doesn't change.
- `struct_field_lookup_test.go` (~179 lines): Minimal changes — tests `LookupStructFields` which returns `StructBuilder`.
- `testing/core_models_integration_test.go`: No changes — tests the full pipeline.

## Execution order

1. Add new methods to StructField (alongside existing functions, not replacing yet)
2. Update `ToSpecSchema` to use the new methods
3. Update `buildSchemaForType` → `BuildSchema` method, using other new methods internally
4. Update `struct_field_lookup.go` callers to use new methods
5. Remove old standalone functions
6. Update tests to use new method signatures
7. Run `make test-project-1` and `make test-project-2` to verify no regressions
