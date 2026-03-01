---
paths:
  - "internal/orchestrator/**/*.go"
---

# Orchestrator Package

## Overview

The Orchestrator package coordinates the full swagger generation pipeline. It wires together all sub-services (loader, registry, schema builder, base parser, route parser) and executes a 6-step sequential pipeline: load packages, seed caches, register types, parse API info, parse routes in parallel, and build demand-driven schemas.

## Key Structs/Methods

### Core Types

- [Service](../../../../internal/orchestrator/service.go#L20) - Main orchestrator that holds all sub-services and coordinates the pipeline
- [Config](../../../../internal/orchestrator/service.go#L31) - All orchestrator configuration options (26 exported fields)
- [Debugger](../../../../internal/orchestrator/service.go#L55) - Debug logging interface

### Entry Points

- [New(config)](../../../../internal/orchestrator/service.go#L60) - Constructor that creates and wires all sub-services
- [Service.Parse(searchDirs, mainAPIFile, parseDepth)](../../../../internal/orchestrator/service.go#L176) - Main entry point: runs the full 6-step pipeline, returns `*spec.Swagger`
- [Service.GetSwagger()](../../../../internal/orchestrator/service.go#L345) - Returns the swagger spec
- [Service.Registry()](../../../../internal/orchestrator/service.go#L350) - Returns the registry service
- [Service.SchemaBuilder()](../../../../internal/orchestrator/service.go#L355) - Returns the schema builder

### Route Parsing

- [Service.parseRoutesParallel(files)](../../../../internal/orchestrator/routes_parallel.go#L28) - Concurrent route parsing bounded by NumCPU, deterministic output via file path sorting

### Reference Collection & Schema Building

- [CollectReferencedTypes(routes)](../../../../internal/orchestrator/refs.go#L15) - Walks all routes collecting unique `$ref` type names
- [Service.buildDemandDrivenSchemas(referencedTypes)](../../../../internal/orchestrator/schema_builder.go#L17) - Builds schemas only for route-referenced types (both base and Public variants)
- [Service.buildSchemaForRef(refName, source)](../../../../internal/orchestrator/schema_builder.go#L41) - Resolves a single `$ref` and builds its schema

### Name Resolution

- [newRegistryNameResolver(registry)](../../../../internal/orchestrator/name_resolver.go#L20) - Creates a `model.DefinitionNameResolver` backed by the registry
- [registryNameResolver.ResolveDefinitionName(fullTypePath)](../../../../internal/orchestrator/name_resolver.go#L32) - Returns canonical definition name (short for unique types, full-path for NotUnique)

## Related Packages

### Depends On
- [internal/loader](../../../../internal/loader) - Package/file discovery and loading
- [internal/registry](../../../../internal/registry) - Type collection, deduplication, uniqueness tracking
- [internal/schema](../../../../internal/schema) - Schema building for non-struct types
- [internal/model](../../../../internal/model) - CoreStructParser, enum lookup, cache seeding, name resolution
- [internal/parser/base](../../../../internal/parser/base) - General API info parsing (swagger metadata)
- [internal/parser/route](../../../../internal/parser/route) - Route annotation parsing
- [internal/parser/route/domain](../../../../internal/parser/route/domain) - Route domain types
- `github.com/go-openapi/spec` - OpenAPI spec types
- `golang.org/x/sync/errgroup` - Bounded concurrency

### Used By
- [internal/gen/gen.go](../../../../internal/gen/gen.go) - Primary caller: `Gen.Build()` creates orchestrator and calls `Parse()`
- [testing/core_models_integration_test.go](../../../../testing/core_models_integration_test.go) - Integration tests against real projects

## Docs

No dedicated README exists.

## Related Skills

No specific skills are directly related to this package.
