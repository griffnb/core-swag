---
paths:
  - "internal/parser/field/**/*.go"
---

# Field Parser Package

## Overview

The Field Parser package provides shared constants for schema types and naming strategies used across the project. It also contains a collection format validation helper. This is a minimal leaf package with zero imports.

## Key Structs/Methods

### Schema Type Constants (types.go)

- [ARRAY](../../../../internal/parser/field/types.go#L6) - `"array"`
- [OBJECT](../../../../internal/parser/field/types.go#L8) - `"object"`
- [PRIMITIVE](../../../../internal/parser/field/types.go#L10) - `"primitive"`
- [BOOLEAN](../../../../internal/parser/field/types.go#L12) - `"boolean"`
- [INTEGER](../../../../internal/parser/field/types.go#L14) - `"integer"`
- [NUMBER](../../../../internal/parser/field/types.go#L16) - `"number"`
- [STRING](../../../../internal/parser/field/types.go#L18) - `"string"`

### Naming Strategy Constants (types.go)

- [CamelCase](../../../../internal/parser/field/types.go#L24) - `"camelcase"`
- [PascalCase](../../../../internal/parser/field/types.go#L26) - `"pascalcase"`
- [SnakeCase](../../../../internal/parser/field/types.go#L28) - `"snakecase"`

### Functions

- [TransToValidCollectionFormat(format)](../../../../internal/parser/field/helpers.go#L5) - Validates collection format strings (csv, multi, pipes, tsv, ssv)

## Related Packages

### Depends On
- Nothing (zero imports - leaf package)

### Used By
- [cmd/core-swag/main.go](../../../../cmd/core-swag/main.go) - Uses naming strategy constants and `TransToValidCollectionFormat`

## Docs

No dedicated README exists. Package originally contained field parsing utilities that have since been removed.

## Related Skills

No specific skills are directly related to this package.
