---
paths:
  - "internal/loader/**/*.go"
---

# Loader Service

## Overview

The Loader Service handles discovering and loading Go packages with their AST (Abstract Syntax Tree) files. It provides three loading strategies: filepath walk, go list command, and go/packages API. The service supports filtering by package prefixes, excluding specific packages, and configurable dependency depth resolution.

## Key Structs/Methods

### Core Types

- [Service](../../../../internal/loader/types.go#L24) - Main service struct that orchestrates package loading
- [LoadResult](../../../../internal/loader/types.go#L43) - Contains parsed AST files and package information
- [AstFileInfo](../../../../internal/loader/types.go#L49) - Metadata about a parsed Go source file
- [ParseFlag](../../../../internal/loader/types.go#L11) - Controls what to parse (models, operations, or both)

### Main Entry Points

- [NewService(options ...Option)](../../../../internal/loader/options.go#L4) - Creates a new loader service with configuration options
- [LoadSearchDirs(dirs []string)](../../../../internal/loader/loader.go#L13) - Loads Go files from specified directories using filepath walk
- [LoadDependencies(dirs []string, maxDepth int)](../../../../internal/loader/dependency.go#L15) - Loads package dependencies with configurable depth
- [LoadWithGoPackages(searchDirs []string, absMainAPIFilePath string)](../../../../internal/loader/gopackages.go#L13) - Loads packages using go/packages API (most robust method)

### Configuration Options

- [WithParseVendor(bool)](../../../../internal/loader/options.go#L21) - Enable/disable parsing vendor directories
- [WithParseInternal(bool)](../../../../internal/loader/options.go#L27) - Enable/disable parsing internal directories
- [WithExcludes(map[string]struct{})](../../../../internal/loader/options.go#L33) - Set package exclusion patterns
- [WithPackagePrefix([]string)](../../../../internal/loader/options.go#L39) - Set package prefix filters
- [WithParseExtension(string)](../../../../internal/loader/options.go#L45) - Set file extension to parse (default: ".go")
- [WithGoList(bool)](../../../../internal/loader/options.go#L51) - Enable go list for dependency resolution
- [WithGoPackages(bool)](../../../../internal/loader/options.go#L57) - Enable go/packages for loading
- [WithParseDependency(ParseFlag)](../../../../internal/loader/options.go#L63) - Set what to parse in dependencies
- [WithDebugger(Debugger)](../../../../internal/loader/options.go#L69) - Set debug logger

### Internal Methods

- [walkDirectory(packageDir, searchDir string, result *LoadResult)](../../../../internal/loader/loader.go#L40) - Recursively walks directories to find Go files
- [parseFile(packageDir, path string, src interface{}, flag ParseFlag, result *LoadResult)](../../../../internal/loader/loader.go#L74) - Parses a single Go file into AST
- [loadDependenciesWithGoList(dirs []string, result *LoadResult)](../../../../internal/loader/dependency.go#L32) - Loads dependencies using go list command
- [walkPackages(pkgs []*packages.Package, fset *token.FileSet, result *LoadResult, rootPkgs []*packages.Package)](../../../../internal/loader/gopackages.go#L59) - Walks package tree from go/packages

## Related Packages

### Depends On
- `go/ast` - Go's AST representation
- `go/token` - Token and position information
- `go/parser` - Go source code parser
- `golang.org/x/tools/go/packages` - Package loading API
- `github.com/KyleBanks/depth` - Dependency depth analysis

### Used By
- [parser.go](../../../../parser.go#L484) - Main parser uses LoadSearchDirs, LoadDependencies, and LoadWithGoPackages
- [internal/registry](../../../../internal/registry) - Registry service consumes loaded AST files

## Docs

- [Loader Service README](../../../../internal/loader/README.md) - Detailed service documentation with usage examples

## Related Skills

No specific skills are directly related to this internal package. This is a foundational service used throughout the codebase.

## Usage Example

```go
// Create loader with configuration
loader := loader.NewService(
    loader.WithParseVendor(false),
    loader.WithParseInternal(true),
    loader.WithParseDependency(loader.ParseModels),
)

// Load from search directories
result, err := loader.LoadSearchDirs([]string{"./api", "./internal"})
if err != nil {
    return err
}

// Load dependencies with depth limit
result, err = loader.LoadDependencies([]string{"./api"}, 10)
if err != nil {
    return err
}

// Or use go/packages (recommended for complex projects)
result, err = loader.LoadWithGoPackages([]string{"./api"}, "/path/to/main.go")
if err != nil {
    return err
}

// Access loaded files
for astFile, fileInfo := range result.Files {
    fmt.Printf("Loaded: %s (package: %s)\n", fileInfo.Path, fileInfo.PackagePath)
}
```

## Design Principles

1. **Multiple Loading Strategies**: Supports three methods (filepath walk, go list, go/packages) for different use cases
2. **Configurable Filtering**: Package prefix, vendor, internal, and custom exclusion patterns
3. **Parse Flags**: Control what gets parsed (models only, operations only, or both) to optimize performance
4. **Dependency Management**: Configurable depth resolution to avoid loading entire dependency tree
5. **Error Resilience**: Continues loading on individual file errors with debug logging

## Common Patterns

- Always use `LoadWithGoPackages` for production use (most robust)
- Use `LoadSearchDirs` for simple directory scanning
- Set `ParseDependency` flag based on what you need from dependencies
- Use `WithExcludes` to skip problematic packages
- Enable debug logging during development with `WithDebugger`
