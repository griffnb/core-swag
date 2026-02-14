# Orchestrator Service

## Overview

The Orchestrator Service is a clean, simple coordinator that orchestrates all parsing services to generate OpenAPI documentation. It provides a unified entry point that coordinates the loader, registry, schema builder, and parsers without containing any business logic itself.

## Purpose

The orchestrator exists to:
- Replace the monolithic 2,336-line parser.go with clean coordination
- Provide a clear, understandable flow of parsing operations
- Delegate all business logic to specialized services
- Keep coordination simple (< 300 lines)

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Orchestrator Service                      │
│  ┌────────────────────────────────────────────────────────┐ │
│  │                   Parse() Method                        │ │
│  │                                                          │ │
│  │  1. Load packages      → Loader Service                 │ │
│  │  2. Register types     → Registry Service               │ │
│  │  3. Parse API info     → Base Parser Service            │ │
│  │  4. Parse routes       → Route Parser Service (TODO)    │ │
│  │  5. Build schemas      → Schema Builder Service         │ │
│  │  6. Cleanup unused     → (TODO)                         │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## Key Features

- **Simple Coordination**: No business logic, just calls services in order
- **Configurable**: Accepts comprehensive configuration for all aspects of parsing
- **Delegating**: All work done by specialized services
- **Small Surface**: < 300 lines of coordination code
- **Tested**: Full test coverage for coordination logic

## Usage

### Basic Usage

```go
// Create with default configuration
orchestrator := orchestrator.New(nil)

// Parse API
swagger, err := orchestrator.Parse(
    []string{"./api", "./internal"}, // search directories
    "./main.go",                      // main API file
    10,                               // dependency depth
)
```

### Custom Configuration

```go
config := &orchestrator.Config{
    ParseVendor:             false,
    ParseInternal:           true,
    ParseDependency:         loader.ParseModels,
    PropNamingStrategy:      "camelcase",
    RequiredByDefault:       false,
    Strict:                  false,
    MarkdownFileDir:         "./docs",
    CodeExampleFilesDir:     "./examples",
    CollectionFormatInQuery: "csv",
    Excludes:                map[string]struct{}{"vendor": {}},
    PackagePrefix:           []string{"github.com/myapp"},
    ParseExtension:          ".go",
    ParseGoList:             true,
    ParseGoPackages:         true,
    HostState:               "production",
    ParseFuncBody:           true,
    UseStructName:           false,
    Overrides:               make(map[string]string),
    Tags:                    make(map[string]struct{}),
    Debug:                   &debugLogger{},
}

orchestrator := orchestrator.New(config)
swagger, err := orchestrator.Parse(searchDirs, mainFile, depth)
```

## Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `ParseVendor` | `bool` | `false` | Parse vendor directories |
| `ParseInternal` | `bool` | `true` | Parse internal packages |
| `ParseDependency` | `ParseFlag` | `ParseModels` | What to parse in dependencies |
| `PropNamingStrategy` | `string` | `"camelcase"` | Property naming (camelcase, pascalcase, snakecase) |
| `RequiredByDefault` | `bool` | `false` | Make all fields required by default |
| `Strict` | `bool` | `false` | Error on warnings |
| `MarkdownFileDir` | `string` | `""` | Directory for markdown docs |
| `CodeExampleFilesDir` | `string` | `""` | Directory for code examples |
| `CollectionFormatInQuery` | `string` | `"csv"` | Array format in query params |
| `Excludes` | `map[string]struct{}` | `{}` | Package patterns to exclude |
| `PackagePrefix` | `[]string` | `[]` | Package prefixes to include |
| `ParseExtension` | `string` | `".go"` | File extension to parse |
| `ParseGoList` | `bool` | `true` | Use go list for dependencies |
| `ParseGoPackages` | `bool` | `true` | Use go/packages API |
| `HostState` | `string` | `""` | Host state for swagger |
| `ParseFuncBody` | `bool` | `true` | Parse function bodies for annotations |
| `UseStructName` | `bool` | `false` | Use simple struct names |
| `Overrides` | `map[string]string` | `{}` | Type name overrides |
| `Tags` | `map[string]struct{}` | `{}` | Filter operations by tags |
| `Debug` | `Debugger` | `nil` | Debug logger |

## Parse Flow

The `Parse()` method coordinates services in this order:

### 1. Load Packages
- Uses loader service to discover and parse Go files
- Supports go/packages API (robust) or directory walking (simple)
- Loads dependencies up to specified depth

### 2. Register Types
- Collects AST files into registry
- Registers all type definitions
- Parses type information into schemas

### 3. Parse General API Info
- Uses base parser to extract API metadata
- Parses @title, @version, @description, etc.
- Builds security definitions and tags

### 4. Parse Routes (TODO)
- Will parse @router annotations from functions
- Extract parameters, responses, security
- Register operations with swagger spec

### 5. Build Schemas
- Generates OpenAPI schemas for all types
- Syncs schemas to swagger definitions
- Handles references and nested types

### 6. Cleanup (TODO)
- Remove unused definitions
- Optimize schema references

## Services Used

The orchestrator depends on these services:

- **LoaderService** (`internal/loader`) - Loads Go packages and files
- **RegistryService** (`internal/registry`) - Manages type registry
- **SchemaBuilderService** (`internal/schema`) - Builds OpenAPI schemas
- **BaseParserService** (`internal/parser/base`) - Parses general API info
- **StructParserService** (`internal/parser/struct`) - Parses struct definitions (future)
- **RouteParserService** (`internal/parser/route`) - Parses route annotations (future)

## Public Methods

### New(config *Config) *Service
Creates a new orchestrator with the given configuration.

### Parse(searchDirs []string, mainAPIFile string, parseDepth int) (*spec.Swagger, error)
Main entry point that coordinates all services to generate swagger spec.

### GetSwagger() *spec.Swagger
Returns the swagger specification.

### Registry() *registry.Service
Returns the registry service for external access.

### SchemaBuilder() *schema.BuilderService
Returns the schema builder service for external access.

## Design Principles

1. **KISS Above All**: Keep coordination simple and obvious
2. **No Business Logic**: All logic is in services, orchestrator just calls them
3. **Clear Flow**: Linear execution with clear steps
4. **Error Propagation**: Wrap errors with context, don't handle them
5. **Dependency Injection**: Services are created and injected, not accessed globally
6. **Small Size**: Keep under 300 lines by delegating everything

## Current Status

**Implemented**:
- ✅ Service creation with full configuration
- ✅ Package loading (both go/packages and directory walking)
- ✅ Type registration
- ✅ General API info parsing
- ✅ Schema building
- ✅ Test coverage

**TODO**:
- ⚠️ Route parsing (waiting for route parser completion)
- ⚠️ Unused definition cleanup
- ⚠️ Full integration testing

## Testing

Tests cover:
- Service creation with default/custom config
- Parsing simple APIs
- Handling empty directories
- Accessor methods for services

Run tests:
```bash
go test ./internal/orchestrator
```

## Size Compliance

Current line count: **285 lines** (including comments and whitespace)

Target: **< 300 lines**

Status: ✅ **PASS** - Well under target

## Future Work

1. **Complete Route Parser Integration**
   - Once route parser supports schema resolution
   - Add step 4 to Parse() method
   - Update tests to verify route parsing

2. **Add Cleanup Logic**
   - Implement step 6 to remove unused definitions
   - Track which schemas are actually referenced
   - Clean up definitions map

3. **Enhance Error Handling**
   - Add better error context
   - Group related errors
   - Provide actionable error messages

4. **Performance Optimization**
   - Profile parsing performance
   - Optimize schema building
   - Cache expensive operations

## Related Files

- `service.go` - Main orchestrator service
- `service_test.go` - Test suite
- `../loader/` - Package loading service
- `../registry/` - Type registry service
- `../schema/` - Schema builder service
- `../parser/base/` - Base parser service
- `../parser/route/` - Route parser service (future)

## Migration Path

The orchestrator is designed to replace the monolithic parser.go:

**Before** (Legacy):
- parser.go: 2,336 lines with all logic mixed together
- operation.go: 1,317 lines
- field_parser.go: 700 lines
- packages.go: 788 lines
- generics.go: 522 lines

**After** (Clean Architecture):
- orchestrator/service.go: 285 lines (coordination only)
- loader/: Package loading
- registry/: Type management
- schema/: Schema building
- parser/base/: API info parsing
- parser/route/: Route parsing
- parser/struct/: Struct parsing

Total legacy code: **5,663 lines**
New orchestrator: **285 lines** + specialized services

This represents a massive improvement in maintainability and clarity.
