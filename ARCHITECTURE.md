# Swag Refactored Architecture

## Overview

The swag codebase has been refactored from a monolithic structure into a modular, service-based architecture. This refactoring addressed several key issues:

- **Large monolithic files**: parser.go (2,435 lines), operation.go (1,314 lines), packages.go (788 lines)
- **Mixed responsibilities**: Single files handled parsing, validation, and schema building
- **Scattered state**: Parser struct had 20+ fields managing different aspects
- **Difficult testing**: Tight coupling made unit testing individual components hard

The refactored architecture provides:

- **Smaller, focused files**: No file exceeds 500 lines
- **Clear separation of concerns**: Each service has a single responsibility
- **Better testability**: Services can be tested in isolation
- **Stateful services**: Each service owns its state
- **Maintainable code**: Easy to find and modify functionality

## Package Structure

```
swag/
├── cmd/                          # CLI commands (unchanged)
├── format/                       # Swagger formatter (unchanged)
│
├── internal/                     # Internal packages (refactored)
│   ├── domain/                   # Shared domain types
│   ├── loader/                   # Package discovery and loading
│   ├── registry/                 # Type and package registry
│   ├── schema/                   # Schema building and management
│   └── parser/                   # Parsing services
│       ├── base/                 # General API info parsing
│       ├── struct/               # Struct parsing
│       └── route/                # Route/operation parsing
│
└── examples/                     # Real importable Go projects for testing
    ├── basicapp/                 # Simple CRUD app example
    └── customfields/             # Custom fields example
```

## Service Descriptions

### internal/domain/

**Purpose**: Shared domain types used across all services.

**Key Types**:
- `TypeSpecDef`: Represents a Go type specification with package info
- Utility functions for type analysis

**Status**: ✅ Integrated

---

### internal/loader/

**Purpose**: Discovers and loads Go packages with their AST files.

**Key Capabilities**:
- Load Go source files from search directories
- Load package dependencies with configurable depth
- Support multiple loading strategies (filepath walk, go list, go/packages)
- Filter packages by prefix and exclude patterns

**Key Files**:
- `loader.go`: Main loading logic for search directories
- `dependency.go`: Dependency loading with depth
- `golist.go`: Integration with go list command
- `gopackages.go`: Integration with go/packages
- `package.go`: Package name resolution
- `parser.go`: Go file parsing

**Key Methods**:
```go
service := loader.NewService(options...)
result, err := service.LoadSearchDirs([]string{"./api"})
result, err := service.LoadDependencies([]string{"./api"}, depth)
result, err := service.LoadWithGoPackages([]string{"./api"}, mainFilePath)
```

**Status**: ✅ Integrated - [See README](internal/loader/README.md)

---

### internal/registry/

**Purpose**: Centralized management of type and package registries.

**Key Capabilities**:
- Register packages and their files
- Register and lookup type specifications
- Parse type definitions into schemas
- Handle enum constants
- Resolve type references across packages

**Key Files**:
- `service.go`: Main registry service and core operations
- `types.go`: Type parsing and registration
- `enums.go`: Enum and constant handling
- `lookup.go`: Type lookup and import resolution
- `constevaluator.go`: Constant expression evaluation
- `dependency.go`: External package loading

**Key Methods**:
```go
svc := registry.NewService()
err := svc.ParseFile(pkgPath, fileName, src, parseFlag)
schemas, err := svc.ParseTypes()
typeDef := svc.FindTypeSpec(typeName, astFile)
defs := svc.UniqueDefinitions()
```

**Status**: ✅ Integrated - [See README](internal/registry/README.md)

---

### internal/schema/

**Purpose**: Build and manage OpenAPI schemas from Go types.

**Key Capabilities**:
- Build OpenAPI schemas from type specifications
- Resolve schema references
- Clean up unused definitions
- Manage schema definitions map

**Key Files**:
- `builder.go`: Schema construction and management
- `types.go`: Type system utilities and Go type mapping
- `reference.go`: Reference resolution logic
- `cleanup.go`: Unused definition removal

**Key Methods**:
```go
builder := schema.NewBuilder()
schema, err := builder.BuildSchema(typeSpec)
err := builder.ResolveReferences()
builder.CleanupUnusedDefinitions(swagger)
```

**Status**: ✅ Integrated - [See README](internal/schema/README.md)

---

### internal/parser/base/

**Purpose**: Parse general API information from comments.

**Key Capabilities**:
- Parse @title, @version, @description annotations
- Parse security definitions
- Parse contact and license information
- Handle external documentation references

**Key Files**:
- `service.go`: Main service and orchestration
- `info.go`: General API info parsing
- `security.go`: Security definitions parser
- `extensions.go`: Extension handling
- `helpers.go`: Utility functions

**Key Methods**:
```go
parser := base.NewService(swagger)
err := parser.ParseGeneralInfo(mainFilePath, mainPackage)
```

**Status**: ✅ Integrated - [See README](internal/parser/base/README.md)

---

### internal/parser/struct/

**Purpose**: Parse Go structs into OpenAPI schemas.

**Key Capabilities**:
- Parse standard Go structs
- Parse custom model structs (fields.StructField[T])
- Handle field tags (json, validate, swaggertype, etc.)
- Support generic types
- Handle embedded fields
- Public/private field filtering

**Key Files**:
- `service.go`: Main struct parsing service
- `field.go`: Field parsing with tag handling

**Key Methods**:
```go
parser := struct.NewService(registry, schemaBuilder, options)
schema, err := parser.ParseStruct(typeSpec)
```

**Status**: ✅ Integrated - [See README](internal/parser/struct/README.md)

---

### internal/parser/route/

**Purpose**: Parse function comments to extract route definitions.

**Key Capabilities**:
- Parse @router, @summary, @description annotations
- Extract @param definitions (query, path, header, body)
- Parse @success and @failure responses
- Handle security requirements
- Support code examples

**Key Files**:
- `service.go`: Main route parsing service
- `operation.go`: Operation parser
- `parameter.go`: Parameter extraction
- `response.go`: Response extraction
- `domain/route.go`: Route domain object

**Key Methods**:
```go
parser := route.NewService(registry, structParser, swagger, options)
err := parser.ParseRoutes(astFile)
operation, err := parser.ParseOperation(funcDecl)
```

**Status**: ✅ Integrated - [See README](internal/parser/route/README.md)

---

## Data Flow

### 1. Package Loading Phase

```
ParseAPI()
  → LoaderService.LoadSearchDirs()
    → Discovers Go files in search directories
    → Returns LoadResult with AstFileInfo[]

  → LoaderService.LoadDependencies()
    → Loads package dependencies with depth limit
    → Returns LoadResult with dependency files
```

### 2. Type Registration Phase

```
RegistryService.CollectFiles()
  → Registers all files from LoadResult
  → Associates files with package paths

RegistryService.ParseTypes()
  → For each registered file:
    → Parses type declarations (structs, interfaces, aliases)
    → Registers types in registry
    → Extracts enum constants
  → Returns map of schemas
```

### 3. General API Info Phase

```
BaseParserService.ParseGeneralInfo()
  → Parses main file comments
  → Extracts @title, @version, @description
  → Parses security definitions
  → Updates swagger.Info and swagger.SecurityDefinitions
```

### 4. Route Parsing Phase

```
RouteParserService.ParseRoutes()
  → For each file with functions:
    → RouteParserService.ParseOperation()
      → Parses function comments
      → Extracts route metadata (@router, @summary, etc.)
      → Parses parameters (@param)
      → Parses responses (@success, @failure)
      → Creates Operation object
    → Adds operation to swagger.Paths
```

### 5. Schema Finalization Phase

```
SchemaBuilderService.ResolveReferences()
  → Resolves all $ref references
  → Validates schema integrity

SchemaBuilderService.CleanupUnusedDefinitions()
  → Removes definitions not referenced by any path
  → Optimizes final swagger spec

→ Returns complete Swagger specification
```

## Integration Status

### Fully Integrated Services

- ✅ **LoaderService**: Package discovery and loading
- ✅ **RegistryService**: Type and package registry
- ✅ **SchemaBuilderService**: Schema building
- ✅ **BaseParserService**: General API info parsing
- ✅ **StructParserService**: Struct parsing
- ✅ **RouteParserService**: Route/operation parsing

### Main Parser

The main `Parser` struct in `parser.go` has been refactored to act as an orchestrator:

```go
type Parser struct {
    loader       *loader.Service
    registry     *registry.Service
    baseParser   *base.Service
    structParser *struct.Service
    routeParser  *route.Service
    schemaBuilder *schema.Builder
    swagger      *spec.Swagger
}
```

The parser orchestrates all services and manages the overall parsing flow.

## Testing

All services follow TDD (Test-Driven Development) principles:

### Unit Tests
- Each service has comprehensive unit tests
- Tests are table-driven for multiple scenarios
- Services are tested in isolation with mocks

### Integration Tests
- `examples/basicapp/`: Simple CRUD app for basic testing
- `examples/customfields/`: Custom fields pattern testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests for specific service
go test ./internal/loader/...
go test ./internal/registry/...
go test ./internal/schema/...
go test ./internal/parser/base/...
go test ./internal/parser/struct/...
go test ./internal/parser/route/...

# Run with coverage
go test -cover ./...
```

## Design Principles

### 1. Single Responsibility
Each service has one clear purpose:
- Loader: Load packages
- Registry: Manage types
- Schema: Build schemas
- Parser: Parse annotations

### 2. Small Files
- No file exceeds 500 lines
- Related functions grouped in same file
- Easy to find and modify code

### 3. Stateful Services
- Services own their state
- No need to pass state through many parameters
- Clear ownership of data

### 4. Dependency Injection
- Services receive dependencies via constructors
- Easy to test with mocks
- Clear service relationships

### 5. Error Handling
- Explicit error returns
- Context added with errors.Wrapf
- No silent failures

## Migration from Old Code

### File Mapping

| Old File | New Location | Lines Reduced |
|----------|--------------|---------------|
| parser.go (2,435 lines) | Multiple services | ~85% reduction |
| operation.go (1,314 lines) | internal/parser/route/ | ~75% reduction |
| packages.go (788 lines) | internal/registry/ | ~70% reduction |
| types.go | internal/domain/ + internal/registry/types.go | Reorganized |

### Key Changes

1. **Parser.ParseAPI()**: Now orchestrates services instead of doing everything
2. **PackagesDefinitions**: Replaced by registry.Service
3. **Operation parsing**: Split into route.Service with separate files for parameters and responses
4. **Type parsing**: Moved to registry.Service and struct.Service

## Future Work

### Potential Enhancements

1. **Parser caching**: Cache parsed results for faster incremental builds
2. **Parallel parsing**: Parse independent files in parallel
3. **Plugin system**: Allow custom parsers for special annotations
4. **Better error messages**: Include more context and suggestions
5. **Performance profiling**: Optimize hot paths

### Extensibility

The modular architecture makes it easy to:
- Add new annotation types (extend route parser)
- Support new schema features (extend schema builder)
- Add custom type handling (extend struct parser)
- Implement new loading strategies (extend loader)

## Further Reading

- [Refactoring Status](REFACTORING_STATUS.md) - Current status of the refactoring
- [Refactoring Plan](.agents/plans/radiant-purring-axolotl.md) - Original refactoring plan
- Service README files:
  - [Loader Service](internal/loader/README.md)
  - [Registry Service](internal/registry/README.md)
  - [Schema Builder](internal/schema/README.md)
  - [Base Parser](internal/parser/base/README.md)
  - [Struct Parser](internal/parser/struct/README.md)
  - [Route Parser](internal/parser/route/README.md)
