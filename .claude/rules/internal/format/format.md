---
paths:
  - "internal/format/**/*.go"
---

# Format Package

## Overview

The Format package handles formatting of swagger annotation comments in Go source files. It aligns `@param`, `@success`, `@failure`, `@response`, and `@header` annotations into tab-aligned columns using `tabwriter`, and runs `goimports` for final cleanup. Used exclusively by the CLI `fmt` subcommand.

## Key Structs/Methods

### Core Types

- [Format](../../../../internal/format/format.go#L18) - Top-level coordinator handling file discovery, exclusion, and concurrent formatting
- [Config](../../../../internal/format/format.go#L34) - Configuration with `SearchDir`, `Excludes`, and `MainFile` fields
- [Formatter](../../../../internal/format/formatter.go#L50) - Core formatting engine using AST parsing and tabwriter alignment
- [Debugger](../../../../internal/format/formatter.go#L29) - Debug logging interface

### Entry Points

- [New()](../../../../internal/format/format.go#L26) - Creates a new Format instance
- [Format.Build(config)](../../../../internal/format/format.go#L48) - Walks directories, applies exclusions, formats all `.go` files concurrently via errgroup
- [Format.Run(src, dst)](../../../../internal/format/format.go#L141) - Pipe/stdin mode: reads from `io.Reader`, formats, writes to `io.Writer`
- [NewFormatter()](../../../../internal/format/formatter.go#L56) - Creates a Formatter with stdout debug logger
- [Formatter.Format(fileName, contents)](../../../../internal/format/formatter.go#L65) - Parses Go source, finds swag comments, aligns with tabwriter, runs `imports.Process`

## Related Packages

### Depends On
- `golang.org/x/sync/errgroup` - Concurrent file processing
- `golang.org/x/tools/imports` - Go imports processing
- `go/ast`, `go/parser`, `go/token` - AST parsing for comment detection
- `text/tabwriter` - Column alignment

### Used By
- [cmd/core-swag/main.go](../../../../cmd/core-swag/main.go) - CLI `fmt` subcommand

## Docs

No dedicated README exists.

## Related Skills

No specific skills are directly related to this package.
