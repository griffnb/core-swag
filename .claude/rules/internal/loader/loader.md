---
paths:
  - "internal/loader/**/*.go"
---

# Loader Service

## Overview

The Loader Service handles discovering and loading Go packages with their AST files. It provides three loading strategies: filepath walk, go list command, and go/packages API. The service supports filtering by package prefixes, excluding specific packages, and configurable dependency depth resolution.

## Key Structs/Methods

### Core Types

- [ParseFlag](../../../../internal/loader/types.go#L11) - Controls what to parse (models, operations, or both)
- [Service](../../../../internal/loader/types.go#L25) - Main service struct that orchestrates package loading
- [Debugger](../../../../internal/loader/types.go#L38) - Debug logging interface
- [LoadResult](../../../../internal/loader/types.go#L43) - Contains parsed AST files and `packages.Package` references
- [AstFileInfo](../../../../internal/loader/types.go#L49) - Metadata about a parsed Go source file
- [Option](../../../../internal/loader/types.go#L58) - Functional option type for configuring Service

### Parse Flag Constants

- [ParseNone](../../../../internal/loader/types.go#L15) - `0x00`
- [ParseModels](../../../../internal/loader/types.go#L17) - `0x01`
- [ParseOperations](../../../../internal/loader/types.go#L19) - `0x02`
- [ParseAll](../../../../internal/loader/types.go#L21) - `ParseOperations | ParseModels`

### Main Entry Points

- [NewService(options ...Option)](../../../../internal/loader/options.go#L4) - Creates a new loader service with configuration options
- [Service.LoadSearchDirs(dirs)](../../../../internal/loader/loader.go#L13) - Loads Go files from specified directories using filepath walk
- [Service.LoadDependencies(dirs, maxDepth)](../../../../internal/loader/dependency.go#L15) - Loads package dependencies with configurable depth
- [Service.LoadWithGoPackages(searchDirs, absMainAPIFilePath)](../../../../internal/loader/gopackages.go#L13) - Loads packages using go/packages API (most robust method)

### Configuration Options

- [WithParseVendor(bool)](../../../../internal/loader/options.go#L25) - Enable/disable parsing vendor directories
- [WithParseInternal(bool)](../../../../internal/loader/options.go#L32) - Enable/disable parsing internal directories
- [WithExcludes(map[string]struct{})](../../../../internal/loader/options.go#L39) - Set package exclusion patterns
- [WithPackagePrefix([]string)](../../../../internal/loader/options.go#L46) - Set package prefix filters
- [WithParseExtension(string)](../../../../internal/loader/options.go#L53) - Set file extension to parse (default: ".go")
- [WithGoList(bool)](../../../../internal/loader/options.go#L60) - Enable go list for dependency resolution
- [WithGoPackages(bool)](../../../../internal/loader/options.go#L67) - Enable go/packages for loading
- [WithParseDependency(ParseFlag)](../../../../internal/loader/options.go#L74) - Set what to parse in dependencies
- [WithDebugger(Debugger)](../../../../internal/loader/options.go#L81) - Set debug logger

## Related Packages

### Depends On
- `go/ast`, `go/parser`, `go/token` - Go AST parsing
- `golang.org/x/tools/go/packages` - Package loading API
- `github.com/KyleBanks/depth` - Dependency depth analysis

### Used By
- [internal/domain/types.go](../../../../internal/domain/types.go) - Imports `ParseFlag` type alias
- [internal/orchestrator/service.go](../../../../internal/orchestrator/service.go) - Creates loader and calls all three loading methods
- [internal/gen/gen.go](../../../../internal/gen/gen.go) - Uses `ParseFlag` type conversion

## Docs

- [Loader Service README](../../../../internal/loader/README.md)

## Related Skills

No specific skills are directly related to this package.
