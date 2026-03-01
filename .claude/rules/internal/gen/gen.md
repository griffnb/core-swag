---
paths:
  - "internal/gen/**/*.go"
---

# Gen Package

## Overview

The Gen package is the top-level orchestration layer for the CLI. It bridges CLI arguments to the internal orchestrator, sanitizes the generated swagger spec (removes Inf/NaN values), and writes output files (JSON/YAML) to disk.

## Key Structs/Methods

### Core Types

- [Gen](../../../../internal/gen/gen.go#L37) - Main generation struct with JSON marshaling and output writers
- [Config](../../../../internal/gen/gen.go#L71) - User-facing configuration struct mapping 1:1 with CLI flags (26 exported fields)
- [Debugger](../../../../internal/gen/gen.go#L46) - Debug logging interface
- [Version](../../../../internal/gen/gen.go#L26) - Current version constant (`v1.16.7`)
- [DefaultInstanceName](../../../../internal/gen/gen.go#L29) - Default swagger instance name (`swagger`)
- [DefaultOverridesFile](../../../../internal/gen/gen.go#L32) - Default overrides file (`.swaggo`)

### Entry Points

- [New()](../../../../internal/gen/gen.go#L51) - Creates a new Gen instance with JSON/YAML output writers
- [Gen.Build(config)](../../../../internal/gen/gen.go#L152) - Main entry point: validates dirs, parses overrides, creates orchestrator, calls `Parse()`, sanitizes spec, removes unused definitions, writes output files

## Related Packages

### Depends On
- [internal/console](../../../../internal/console) - Debug logging
- [internal/loader](../../../../internal/loader) - ParseFlag type conversion
- [internal/orchestrator](../../../../internal/orchestrator) - Creates `orchestrator.New()` and calls `orchestrator.Parse()`
- [internal/schema](../../../../internal/schema) - `schema.RemoveUnusedDefinitions()` cleanup
- `github.com/go-openapi/spec` - OpenAPI spec types
- `github.com/pkg/errors` - Error wrapping
- `sigs.k8s.io/yaml` - JSON to YAML conversion

### Used By
- [cmd/core-swag/main.go](../../../../cmd/core-swag/main.go) - CLI entry point calls `gen.New().Build(&gen.Config{...})`

## Docs

No dedicated README exists.

## Related Skills

No specific skills are directly related to this package.
