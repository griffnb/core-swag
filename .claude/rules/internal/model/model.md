---
paths:
  - "internal/model/**/*.go"
---

# Model Package

## Overview

The Model package implements the core struct parsing pipeline (CoreStructParser) that extracts Go struct fields using `go/packages` type information and converts them to OpenAPI schemas. This is the primary schema generation engine for the project, replacing the deprecated StructParserService.

## Key Structs/Methods

### Core Types

- [StructField](../../../../internal/model/struct_field.go#L13) - Represents a parsed struct field with name, type info, tag, and nested sub-fields
- [CoreStructParser](../../../../internal/model/struct_field_lookup.go#L80) - Type-level struct field extractor using `go/packages` for full type resolution
- [StructBuilder](../../../../internal/model/struct_builder.go#L9) - Builds OpenAPI `spec.Schema` from a set of `StructField`s
- [ParserEnumLookup](../../../../internal/model/enum_lookup.go#L60) - Implements `TypeEnumLookup` for detecting enum values via `go/packages`
- [TypeEnumLookup](../../../../internal/model/struct_field.go#L41) - Interface for enum value resolution
- [DefinitionNameResolver](../../../../internal/model/struct_field.go#L56) - Interface for controlling `$ref` name generation
- [EnumValue](../../../../internal/model/struct_field.go#L46) - Enum constant value with key, value, and comment

### Entry Points

- [BuildAllSchemas(baseModule, pkgPath, typeName, packageNameOverride...)](../../../../internal/model/struct_field_lookup.go#L674) - Top-level function that builds all schemas (base + Public variant) for a type and its transitive dependencies
- [CoreStructParser.LookupStructFields(baseModule, importPath, typeName)](../../../../internal/model/struct_field_lookup.go#L125) - Resolves a type to a `StructBuilder` with all extracted fields
- [CoreStructParser.ExtractFieldsRecursive(pkg, typeName, packageMap, visited)](../../../../internal/model/struct_field_lookup.go#L354) - Recursively extracts fields from a struct type with cross-package support
- [StructBuilder.BuildSpecSchema(typeName, public, forceRequired, enumLookup)](../../../../internal/model/struct_builder.go#L16) - Converts extracted fields to OpenAPI schema
- [StructField.ToSpecSchema(public, forceRequired, enumLookup)](../../../../internal/model/struct_field.go#L90) - Converts a single field to OpenAPI schema property

### Global Cache Management

- [SeedGlobalPackageCache(pkgs)](../../../../internal/model/struct_field_lookup.go#L46) - Pre-populates package cache from `go/packages` load results
- [SeedEnumPackageCache(pkgs)](../../../../internal/model/enum_lookup.go#L25) - Pre-populates enum package cache
- [SetGlobalNameResolver(r)](../../../../internal/model/struct_field.go#L66) - Sets the global `DefinitionNameResolver` for `$ref` name generation
- [GlobalCacheStats()](../../../../internal/model/struct_field_lookup.go#L33) - Returns cache hit/miss stats
- [ResetGlobalCacheStats()](../../../../internal/model/struct_field_lookup.go#L38) - Resets cache counters
- [ParserEnumLookup.GetEnumsForType(typeName, file)](../../../../internal/model/enum_lookup.go#L81) - Resolves enum constants for a given type

## Related Packages

### Depends On
- `golang.org/x/tools/go/packages` - Full type information loading
- `github.com/go-openapi/spec` - OpenAPI schema types
- [internal/console](../../../../internal/console) - Debug logging via `console.Logger.Debug()`

### Used By
- [internal/schema/builder.go](../../../../internal/schema/builder.go) - Holds `*CoreStructParser` and `TypeEnumLookup`, delegates struct schema building
- [internal/orchestrator/service.go](../../../../internal/orchestrator/service.go) - Creates `CoreStructParser`, `ParserEnumLookup`, seeds caches, sets name resolver
- [internal/orchestrator/schema_builder.go](../../../../internal/orchestrator/schema_builder.go) - Calls `BuildAllSchemas()` for demand-driven schema generation
- [internal/orchestrator/name_resolver.go](../../../../internal/orchestrator/name_resolver.go) - Implements `DefinitionNameResolver` interface

## Docs

No dedicated README. See CLAUDE.md "STRUCT PARSING ARCHITECTURE" section for architectural overview.

## Related Skills

No specific skills are directly related to this package.
