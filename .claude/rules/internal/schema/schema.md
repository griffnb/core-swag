---
paths:
  - "internal/schema/**/*.go"
---

# Schema Package

## Overview

The Schema package handles OpenAPI schema construction, type utilities, allOf composition, reference management, and unused definition cleanup. The `BuilderService` delegates struct schema building to `CoreStructParser` from the model package.

## Key Structs/Methods

### Core Types

- [BuilderService](../../../../internal/schema/builder.go#L20) - Main schema builder managing definitions, parsed schemas, and struct parser integration
- [TypeResolver](../../../../internal/schema/builder.go#L15) - Interface for type lookup (`FindTypeSpec`)

### Builder Entry Points

- [NewBuilder()](../../../../internal/schema/builder.go#L31) - Creates new schema builder instance
- [BuilderService.SetPropNamingStrategy(strategy)](../../../../internal/schema/builder.go#L42) - Sets property naming (camelCase, snake_case)
- [BuilderService.SetTypeResolver(resolver)](../../../../internal/schema/builder.go#L50) - Sets type resolver for alias/reference resolution
- [BuilderService.SetStructParser(parser)](../../../../internal/schema/builder.go#L55) - Sets `CoreStructParser` for struct schema building
- [BuilderService.SetEnumLookup(enumLookup)](../../../../internal/schema/builder.go#L60) - Sets enum detection adapter
- [BuilderService.BuildSchema(typeSpec)](../../../../internal/schema/builder.go#L66) - Builds OpenAPI schema from TypeSpecDef
- [BuilderService.AddDefinition(name, schema)](../../../../internal/schema/builder.go#L227) - Adds a schema definition manually
- [BuilderService.GetDefinition(name)](../../../../internal/schema/builder.go#L236) - Retrieves schema definition by name
- [BuilderService.Definitions()](../../../../internal/schema/builder.go#L242) - Returns all schema definitions

### Type Utilities (types.go)

- [IsSimplePrimitiveType(typeName)](../../../../internal/schema/types.go#L29) - Checks basic primitives (string, int, bool, etc.)
- [IsPrimitiveType(typeName)](../../../../internal/schema/types.go#L38) - Checks all primitive types including aliases
- [IsComplexSchema(schema)](../../../../internal/schema/types.go#L47) - Checks if schema needs allOf wrapping
- [PrimitiveSchema(refType)](../../../../internal/schema/types.go#L68) - Creates schema for primitive type
- [BuildCustomSchema(types)](../../../../internal/schema/types.go#L73) - Builds schema from swaggertype override list
- [MergeSchema(dst, src)](../../../../internal/schema/types.go#L115) - Merges two schemas together

### Type Constants (types.go)

- `ARRAY`, `OBJECT`, `PRIMITIVE`, `BOOLEAN`, `INTEGER`, `NUMBER`, `STRING`, `FUNC` (lines 11-25)

### AllOf Composition (allof.go)

- [ParseCombinedType(refType)](../../../../internal/schema/allof.go#L11) - Parses `Type{field=Override}` syntax into base type and field map
- [BuildAllOfSchema(baseSchema, overrideProperties)](../../../../internal/schema/allof.go#L16) - Builds allOf schema with property overrides

### Reference Utilities (reference.go)

- [RefSchema(refType)](../../../../internal/schema/reference.go#L8) - Creates a `$ref` schema
- [IsRefSchema(schema)](../../../../internal/schema/reference.go#L13) - Checks if schema is a `$ref`
- [ResolveReferences(definitions)](../../../../internal/schema/reference.go#L22) - Resolves all references in definitions

### Cleanup (cleanup.go)

- [RemoveUnusedDefinitions(swagger)](../../../../internal/schema/cleanup.go#L9) - Removes definitions not referenced by any route, recursively walking all schemas

## Related Packages

### Depends On
- [internal/domain](../../../../internal/domain) - `TypeSpecDef`, `Schema` types
- [internal/model](../../../../internal/model) - `CoreStructParser`, `TypeEnumLookup` for struct schema building
- `github.com/go-openapi/spec` - OpenAPI spec types

### Used By
- [internal/gen/gen.go](../../../../internal/gen/gen.go) - `RemoveUnusedDefinitions()` after generation
- [internal/parser/route/allof.go](../../../../internal/parser/route/allof.go) - `ParseCombinedType()`, `BuildAllOfSchema()`
- [internal/orchestrator/service.go](../../../../internal/orchestrator/service.go) - Creates and configures `BuilderService`

## Docs

No dedicated README exists.

## Related Skills

No specific skills are directly related to this package.
