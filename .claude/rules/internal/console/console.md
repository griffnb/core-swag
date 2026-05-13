---
paths:
  - "internal/console/**/*.go"
---

# Console Package

## Overview

The Console package is the **centralized logger** for the codebase. All
debug/error output flows through `console.Logger` — services do not own their
own debug interfaces. It also provides colored terminal output via three APIs:
simple color functions, a fluent builder, and a template syntax (`$Bold{$Red{text}}`).

## Key Structs/Methods

### Core Types

- [ColorBuilder](../../../../internal/console/console.go#L42) - Fluent builder for chaining color/style codes
- [Logger](../../../../internal/console/debug.go#L11) - Exported package-level `*logger` variable, gated by `DebugLevel` (default 0 = silent)

### Entry Points

- [Format(format, messages...)](../../../../internal/console/console.go#L49) - Creates a `ColorBuilder` with template-formatted text
- [Sprintf(format, args...)](../../../../internal/console/console.go#L226) - Formats string with `$Bold{$Red{text}}` template syntax
- [Logger.Debug(format, args...)](../../../../internal/console/debug.go#L16) - Verbose log to stdout, gated by `DebugLevel`. Default is silent; the CLI flips `DebugLevel = 1` when `--debug` is passed.
- [Logger.Error(format, args...)](../../../../internal/console/debug.go#L23) - Unconditional error log to **stderr**. Used for unrecoverable / user-facing failures (unknown $ref, schema build failure, unsupported output type).

### Simple Color Functions

- [Red(text)](../../../../internal/console/console.go#L235), [Green(text)](../../../../internal/console/console.go#L240), [Yellow(text)](../../../../internal/console/console.go#L245), [Blue(text)](../../../../internal/console/console.go#L250), [Magenta(text)](../../../../internal/console/console.go#L255), [Cyan(text)](../../../../internal/console/console.go#L260), [White(text)](../../../../internal/console/console.go#L265), [Bold(text)](../../../../internal/console/console.go#L270), [Underline(text)](../../../../internal/console/console.go#L275)

### Emoji Constants

- `Check`, `Fire`, `X`, `Info`, `Warning`, `Star` (lines 31-36)

## Logging Policy

- **No package-level `Debugger` interfaces.** Every service logs through `console.Logger` directly. Do not reintroduce per-service `Debugger` fields, `SetDebugger`, or `WithDebugger` options.
- `Logger.Debug` is for trace output and is silent by default.
- `Logger.Error` is for real failure states (missing types, schema build errors, unsupported outputs) and prints on **stderr** regardless of `--debug`.

## Related Packages

### Depends On
- `fmt`, `os`, `strings` (standard library only - leaf package)

### Used By
Every service that emits debug or error output, including:
- [cmd/core-swag/main.go](../../../../cmd/core-swag/main.go) - Flips `DebugLevel` based on `--debug`
- [internal/gen/gen.go](../../../../internal/gen/gen.go)
- [internal/orchestrator/service.go](../../../../internal/orchestrator/service.go), [schema_builder.go](../../../../internal/orchestrator/schema_builder.go)
- [internal/loader/loader.go](../../../../internal/loader/loader.go)
- [internal/registry/types.go](../../../../internal/registry/types.go), [enums.go](../../../../internal/registry/enums.go)
- [internal/parser/base](../../../../internal/parser/base), [internal/parser/route](../../../../internal/parser/route)
- [internal/model/struct_field.go](../../../../internal/model/struct_field.go), [struct_field_lookup.go](../../../../internal/model/struct_field_lookup.go), [enum_lookup.go](../../../../internal/model/enum_lookup.go)

## Docs

No dedicated README exists.

## Related Skills

No specific skills are directly related to this package.
