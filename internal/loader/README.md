# Loader Service

The Loader Service handles discovering and loading Go packages with their AST files.

## Overview

This package provides functionality to:
- Load Go source files from search directories
- Load package dependencies with configurable depth
- Support multiple loading strategies (filepath walk, go list, go/packages)
- Filter packages by prefix and exclude patterns

## Files

- **types.go**: Core types (Service, LoadResult, AstFileInfo)
- **options.go**: Configuration options for the service
- **loader.go**: Main loading logic for search directories
- **dependency.go**: Dependency loading with depth and go list
- **gopackages.go**: Integration with go/packages
- **golist.go**: Integration with go list command
- **package.go**: Package name resolution
- **parser.go**: Go file parsing

## Usage

```go
// Create a loader service
service := loader.NewService(
    loader.WithParseVendor(false),
    loader.WithParseDependency(swag.ParseModels),
)

// Load from search directories
result, err := service.LoadSearchDirs([]string{"./api"})

// Load dependencies
result, err := service.LoadDependencies([]string{"./api"}, 10)

// Load with go/packages
result, err := service.LoadWithGoPackages([]string{"./api"}, "/path/to/main.go")
```

## Testing

All functionality is thoroughly tested in `service_test.go` following TDD principles.

Run tests:
```bash
go test ./internal/loader/...
```
