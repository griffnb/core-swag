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
- [Debugger](../../../../internal/registry/interfaces.go#L4) - Debug logging interface

### Service Management

- [NewService()](../../../../internal/registry/service.go#L29) - Creates new registry service instance
- [Service.SetParseDependency(flag)](../../../../internal/registry/service.go#L38) - Configures what to parse from dependencies
- [Service.SetDebugger(debug)](../../../../internal/registry/service.go#L43) - Sets debug logger

### File & Package Operations

- [Service.ParseFile(packageDir, path, src, flag)](../../../../internal/registry/service.go#L48) - Parses a Go source file into AST
- [Service.CollectAstFile(fileSet, packageDir, path, astFile, flag)](../../../../internal/registry/service.go#L58) - Collects and stores parsed AST file
- [Service.RangeFiles(handle)](../../../../internal/registry/service.go#L102) - Iterates over files in alphabetic order
- [Service.AddPackages(pkgs)](../../../../internal/registry/service.go#L129) - Stores `packages.Package` references

### Type Operations

- [Service.ParseTypes()](../../../../internal/registry/types.go#L16) - Parses all registered types into schema definitions
- [Service.FindTypeSpec(typeName, file)](../../../../internal/registry/types.go#L196) - Finds type specification by name with import resolution
- [Service.FindTypeSpecByName(name)](../../../../internal/registry/service.go#L149) - Finds type by simple name lookup in unique definitions
- [Service.CheckTypeSpec(typeSpecDef)](../../../../internal/registry/types.go#L267) - Validates and sets schema naming (simple vs full-path)
- [Service.AddTypeSpecForTest(name, typeDef)](../../../../internal/registry/service.go#L156) - Test helper for adding types

### Enum Operations

- [Service.EvaluateConstValue(pkg, cv, recursiveStack)](../../../../internal/registry/enums.go#L45) - Evaluates constant expression values
- [Service.EvaluateConstValueByName(file, pkgName, constVariableName, recursiveStack)](../../../../internal/registry/enums.go#L100) - Evaluates constant by name with cross-package resolution

### Accessors

- [Service.UniqueDefinitions()](../../../../internal/registry/service.go#L161) - Returns unique type definitions map
- [Service.Packages()](../../../../internal/registry/service.go#L166) - Returns packages map
- [Service.Files()](../../../../internal/registry/service.go#L171) - Returns AST files map

## Related Packages

### Depends On
- [internal/domain](../../../../internal/domain) - TypeSpecDef, PackageDefinitions, AstFileInfo, ConstVariable
- [internal/console](../../../../internal/console) - Debug logging
- `go/ast`, `go/parser`, `go/token`, `go/types` - AST and type system
- `golang.org/x/tools/go/packages` - Package loading
- `golang.org/x/tools/go/loader` - External package loading

### Used By
- [internal/orchestrator/service.go](../../../../internal/orchestrator/service.go) - Creates and uses registry for type management
- [internal/orchestrator/name_resolver.go](../../../../internal/orchestrator/name_resolver.go) - Uses registry for definition name resolution

## Docs

- [Registry Package README](../../../../internal/registry/README.md)

## Related Skills

No specific skills are directly related to this package.
