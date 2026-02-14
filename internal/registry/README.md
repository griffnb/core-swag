# Registry Package

The registry package provides centralized management of type and package registries for swagger documentation generation.

## Overview

This package extracts and manages:
- Package definitions and their relationships
- Type specifications (structs, interfaces, type aliases)
- Enum constants and their values
- File metadata and AST information

## Architecture

The package is organized into focused files:

- **service.go** (160 lines) - Main registry service and core operations
- **types.go** (260 lines) - Type parsing and registration
- **enums.go** (166 lines) - Enum and constant handling
- **lookup.go** (110 lines) - Type lookup and import resolution
- **constevaluator.go** (101 lines) - Constant expression evaluation
- **dependency.go** (38 lines) - External package loading
- **helpers.go** (32 lines) - Utility functions

## Usage

```go
// Create a new registry service
svc := registry.NewService()

// Parse a file
err := svc.ParseFile("github.com/user/pkg", "file.go", src, swag.ParseAll)

// Parse all types
schemas, err := svc.ParseTypes()

// Find a type specification
typeDef := svc.FindTypeSpec("User", astFile)

// Get all unique definitions
defs := svc.UniqueDefinitions()
```

## Key Features

### Type Registration
- Automatically registers types as files are parsed
- Handles unique and non-unique type names
- Manages function-scoped types
- Tracks type aliases to primitives

### Package Management
- Maintains package definitions and their files
- Resolves import paths
- Loads external dependencies when needed
- Tracks vendor packages

### Enum Support
- Collects const variables
- Evaluates const expressions
- Associates enums with their types
- Handles iota-based enums

### Type Lookup
- Finds types by name within a file's context
- Resolves qualified type names (pkg.Type)
- Handles import aliases
- Supports generic type parametrization

## Design Principles

1. **Separation of Concerns** - Each file handles a specific aspect of the registry
2. **Small Files** - All files under 300 lines for maintainability
3. **Clear Interfaces** - Service provides clean API for registry operations
4. **Testability** - Comprehensive test coverage with table-driven tests
5. **Error Handling** - Explicit error propagation and recovery

## Testing

The package includes comprehensive tests covering:
- Service creation and initialization
- File collection and parsing
- Type registration and lookup
- Enum handling
- Package management

Run tests:
```bash
go test ./internal/registry/...
```
