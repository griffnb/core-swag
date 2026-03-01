---
paths:
  - "internal/console/**/*.go"
---

# Console Package

## Overview

The Console package provides colored terminal output and debug logging. It has three APIs: simple color functions, a fluent builder, and a template syntax (`$Bold{$Red{text}}`). In practice, only the debug logger (`Logger.Debug()`) is used by the rest of the codebase.

## Key Structs/Methods

### Core Types

- [ColorBuilder](../../../../internal/console/console.go#L42) - Fluent builder for chaining color/style codes
- [Logger](../../../../internal/console/debug.go#L7) - Exported package-level `*logger` variable, gated by `DebugLevel` (default 0 = silent)

### Entry Points

- [Format(format, messages...)](../../../../internal/console/console.go#L49) - Creates a `ColorBuilder` with template-formatted text
- [Sprintf(format, args...)](../../../../internal/console/console.go#L226) - Formats string with `$Bold{$Red{text}}` template syntax
- [Logger.Debug(format, args...)](../../../../internal/console/debug.go#L11) - Debug output gated by `DebugLevel`

### Simple Color Functions

- [Red(text)](../../../../internal/console/console.go#L235), [Green(text)](../../../../internal/console/console.go#L240), [Yellow(text)](../../../../internal/console/console.go#L245), [Blue(text)](../../../../internal/console/console.go#L250), [Magenta(text)](../../../../internal/console/console.go#L255), [Cyan(text)](../../../../internal/console/console.go#L260), [White(text)](../../../../internal/console/console.go#L265), [Bold(text)](../../../../internal/console/console.go#L270), [Underline(text)](../../../../internal/console/console.go#L275)

### Emoji Constants

- `Check`, `Fire`, `X`, `Info`, `Warning`, `Star` (lines 31-36)

## Related Packages

### Depends On
- `fmt`, `strings` (standard library only - leaf package)

### Used By
- [cmd/core-swag/main.go](../../../../cmd/core-swag/main.go) - Sets `console.Logger.DebugLevel`
- [internal/model/struct_field.go](../../../../internal/model/struct_field.go) - Debug logging
- [internal/model/struct_field_lookup.go](../../../../internal/model/struct_field_lookup.go) - Heavy debug logging (30+ call sites)
- [internal/model/enum_lookup.go](../../../../internal/model/enum_lookup.go) - Debug logging
- [internal/registry/types.go](../../../../internal/registry/types.go) - Debug logging
- [internal/registry/enums.go](../../../../internal/registry/enums.go) - Debug logging
- [internal/parser/route/registration.go](../../../../internal/parser/route/registration.go) - Debug logging
- [internal/gen/gen.go](../../../../internal/gen/gen.go) - Debug logging

## Docs

No dedicated README exists.

## Related Skills

No specific skills are directly related to this package.
