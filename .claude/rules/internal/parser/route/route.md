---
paths:
  - "internal/parser/route/**/*.go"
---

# Route Parser Package

## Overview

The Route Parser package extracts HTTP route information from Go function documentation comments. It parses `@router`, `@param`, `@success`, `@failure`, `@header`, `@accept`, `@produce`, `@security`, `@tags`, and other annotations to produce route domain objects. It also converts routes to OpenAPI `spec.Operation` objects and registers them into the swagger spec.

## Key Structs/Methods

### Core Types

- [Service](../../../../internal/parser/route/service.go#L20) - Main route parser service
- [TypeRegistry](../../../../internal/parser/route/service.go#L15) - Interface for type lookup (`FindTypeSpec`)

### Domain Types (internal/parser/route/domain/route.go)

- [Route](../../../../internal/parser/route/domain/route.go#L5) - Complete HTTP route with method, path, parameters, responses, security, tags
- [Parameter](../../../../internal/parser/route/domain/route.go#L56) - Route parameter (path, query, header, body, formData)
- [Items](../../../../internal/parser/route/domain/route.go#L101) - Array item type info
- [Response](../../../../internal/parser/route/domain/route.go#L113) - Response with description, schema, headers
- [Header](../../../../internal/parser/route/domain/route.go#L125) - Response header definition
- [Schema](../../../../internal/parser/route/domain/route.go#L137) - Route-level schema (type, ref, items, properties, allOf)

### Service Creation & Configuration

- [NewService(typeResolver, collectionFormat)](../../../../internal/parser/route/service.go#L31) - Creates new route parser
- [Service.SetMarkdownFileDir(dir)](../../../../internal/parser/route/service.go#L42) - Sets markdown file directory
- [Service.SetRegistry(registry)](../../../../internal/parser/route/service.go#L47) - Sets type registry for type resolution

### Main Parsing

- [Service.ParseRoutes(astFile, filePath, fset)](../../../../internal/parser/route/service.go#L54) - Extracts all routes from an AST file

### Registration

- [Service.RegisterRoutes(swagger, routes, strict)](../../../../internal/parser/route/registration.go#L15) - Registers parsed routes into swagger spec paths

### Converters (route domain to spec)

- [RouteToSpecOperation(route)](../../../../internal/parser/route/converter.go#L12) - Converts Route to `spec.Operation`
- [ParameterToSpec(param)](../../../../internal/parser/route/converter.go#L73) - Converts Parameter to `spec.Parameter`
- [ResponseToSpec(resp)](../../../../internal/parser/route/converter.go#L150) - Converts Response to `spec.Response`
- [HeaderToSpec(header)](../../../../internal/parser/route/converter.go#L177) - Converts Header to `spec.Header`
- [SchemaToSpec(schema)](../../../../internal/parser/route/converter.go#L190) - Converts Schema to `spec.Schema`

### AllOf Support

- [Service.buildAllOfResponseSchema(dataType, file)](../../../../internal/parser/route/allof.go#L13) - Handles combined type syntax like `Response{data=Account}`

## Related Packages

### Depends On
- [internal/console](../../../../internal/console) - Debug logging in registration
- [internal/domain](../../../../internal/domain) - `TypeSpecDef` for type resolution
- [internal/parser/route/domain](../../../../internal/parser/route/domain) - Route domain types (pure data structs, zero imports)
- [internal/schema](../../../../internal/schema) - `ParseCombinedType`, `BuildAllOfSchema` for allOf composition
- `github.com/go-openapi/spec` - OpenAPI spec types

### Used By
- [internal/orchestrator/service.go](../../../../internal/orchestrator/service.go) - Creates and configures route parser
- [internal/orchestrator/routes_parallel.go](../../../../internal/orchestrator/routes_parallel.go) - Calls `ParseRoutes` and `RegisterRoutes` in parallel

## Docs

No dedicated README exists.

## Related Skills

No specific skills are directly related to this package.
