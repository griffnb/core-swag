---
paths:
  - "internal/registry/**/*.go"
---

# Registry Service

## Overview

The Registry Service provides centralized management of type specifications, package definitions, and AST file information. It handles type discovery, registration, lookup, enum evaluation, and dependency resolution across Go packages during swagger documentation generation.

## Key Structs/Methods

### Core Types

- [Service](../../../../internal/registry/service.go#L20) - Main registry service managing files, packages, and type definitions
- [TypeSpecDef](../../../../internal/domain/types.go#L27) - Complete type specification with metadata, enums, and package path
- [PackageDefinitions](../../../../internal/domain/types.go#L144) - Package-level definitions including files, types, and constants
- [AstFileInfo](../../../../internal/domain/types.go#L126) - AST file metadata with FileSet, path, and parse flags

### Service Management

- [NewService()](../../../../internal/registry/service.go#L29) - Creates new registry service instance
- [SetParseDependency(flag)](../../../../internal/registry/service.go#L38) - Configures what to parse from dependencies
- [SetDebugger(debug)](../../../../internal/registry/service.go#L43) - Sets debug logger

### File & Package Operations

- [ParseFile(packageDir, path, src, flag)](../../../../internal/registry/service.go#L48) - Parses a Go source file into AST
- [CollectAstFile(fileSet, packageDir, path, astFile, flag)](../../../../internal/registry/service.go#L58) - Collects and stores parsed AST file
- [RangeFiles(handle)](../../../../internal/registry/service.go#L102) - Iterates over files in alphabetic order
- [AddPackages(pkgs)](../../../../internal/registry/service.go#L129) - Stores packages.Package references

### Type Operations

- [ParseTypes()](../../../../internal/registry/types.go#L18) - Parses all registered types into schema definitions
- [FindTypeSpec(typeName, file)](../../../../internal/registry/lookup.go#L24) - Finds type specification by name with import resolution
- [registerTypes(pkg, file)](../../../../internal/registry/types.go#L94) - Registers types from a package file
- [addTypeSpec(typeName, typeSpec)](../../../../internal/registry/types.go#L179) - Adds type specification to registry

### Enum Operations

- [collectEnums()](../../../../internal/registry/enums.go#L19) - Collects enum constants from packages
- [collectConstVariables(pkg, file)](../../../../internal/registry/enums.go#L48) - Collects const declarations from file
- [EvaluateConstValue(pkg, cv, recursiveStack)](../../../../internal/registry/constevaluator.go#L20) - Evaluates constant expression values

### Accessors

- [UniqueDefinitions()](../../../../internal/registry/service.go#L148) - Returns unique type definitions map
- [Packages()](../../../../internal/registry/service.go#L153) - Returns packages map
- [Files()](../../../../internal/registry/service.go#L158) - Returns AST files map

## Related Packages

### Depends On
- `go/ast` - AST representation
- `go/parser` - Go source parsing
- `go/token` - Token position info
- `golang.org/x/tools/go/packages` - Package loading
- [internal/domain](../../../../internal/domain) - Domain types (TypeSpecDef, PackageDefinitions, AstFileInfo)
- [internal/loader](../../../../internal/loader) - ParseFlag constants

### Used By
- [parser.go](../../../../parser.go) - Main parser uses registry for type management
- [internal/parser/struct](../../../../internal/parser/struct) - Struct parser uses type lookup
- [internal/parser/route](../../../../internal/parser/route) - Route parser uses type resolution
- [internal/schema](../../../../internal/schema) - Schema builder uses type definitions

## Docs

- [Registry Package README](../../../../internal/registry/README.md) - Package architecture and usage documentation

## Related Skills

No specific skills are directly related to this internal package. This is a foundational service used throughout the parser system.

## Usage Example

```go
// Create registry service
registry := registry.NewService()
registry.SetParseDependency(domain.ParseModels)

// Parse files
err := registry.ParseFile("github.com/user/api", "models.go", nil, domain.ParseAll)
if err != nil {
    return err
}

// Parse all types into schemas
schemas, err := registry.ParseTypes()
if err != nil {
    return err
}

// Find specific type
typeDef := registry.FindTypeSpec("User", astFile)
if typeDef != nil {
    fmt.Printf("Found type: %s in package: %s\n",
        typeDef.TypeName(), typeDef.PkgPath)
}

// Access unique definitions
for name, typeDef := range registry.UniqueDefinitions() {
    fmt.Printf("Definition: %s\n", name)
}
```

## Design Principles

1. **Centralized Type Management**: Single source of truth for all type definitions across packages
2. **Separated Concerns**: Files organized by responsibility (types, enums, lookup, evaluation)
3. **Import Resolution**: Handles qualified type names and import aliases correctly
4. **Enum Support**: Full support for iota-based enums and const expressions
5. **Lazy Loading**: Can load external dependencies on demand
6. **File Size Control**: All files kept under 300 lines for maintainability
7. **Vendor Filtering**: Automatically filters vendor and GOROOT packages during iteration

## Common Patterns

- Use `ParseFile()` for individual file parsing with specific flags
- Use `CollectAstFile()` when you already have parsed AST
- Call `ParseTypes()` after collecting all files to generate schemas
- Use `FindTypeSpec()` for type resolution during struct/route parsing
- Enable debug logging during development with `SetDebugger()`
- Filter parsing with `ParseFlag` (ParseModels, ParseOperations, ParseAll)
