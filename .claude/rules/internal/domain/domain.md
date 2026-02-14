---
paths:
  - "internal/domain/**/*.go"
---

# Domain Package

## Overview

The Domain package contains core domain types shared across the entire swag application. These types represent fundamental domain objects like type specifications, AST file information, package definitions, schema metadata, and constant evaluation. This package has no external dependencies (except standard library and required AST packages) and serves as the shared type foundation.

## Key Structs/Methods

### Core Domain Types

- [TypeSpecDef](../../../../internal/domain/types.go#L27) - Complete type specification with AST, enums, package path, and schema metadata
- [AstFileInfo](../../../../internal/domain/types.go#L126) - AST file metadata including FileSet, path, package path, and parse flags
- [PackageDefinitions](../../../../internal/domain/types.go#L144) - Package-level definitions containing files, types, constants, and package reference
- [ConstVariable](../../../../internal/domain/types.go#L303) - Constant variable with name, type, value, comment, and file reference
- [Schema](../../../../internal/domain/types.go#L20) - OpenAPI schema wrapper with package path and definition name
- [EnumValue](../../../../internal/domain/types.go#L348) - Enum constant value with key, value, and comment

### TypeSpecDef Methods

- [Name()](../../../../internal/domain/types.go#L46) - Returns simple type name
- [TypeName()](../../../../internal/domain/types.go#L55) - Returns fully qualified type name for definitions
- [SimpleTypeName()](../../../../internal/domain/types.go#L80) - Returns simplified type name (package.Type format)
- [FullPath()](../../../../internal/domain/types.go#L98) - Returns full import path with type name
- [Alias()](../../../../internal/domain/types.go#L102) - Returns name override from comment annotation
- [SetSchemaName()](../../../../internal/domain/types.go#L106) - Sets schema name using full naming
- [SetSchemaNameSimple()](../../../../internal/domain/types.go#L116) - Sets schema name using simplified naming

### PackageDefinitions Methods

- [NewPackageDefinitions(name, pkgPath)](../../../../internal/domain/types.go#L174) - Creates new package definitions instance
- [AddFile(pkgPath, file)](../../../../internal/domain/types.go#L185) - Adds AST file to package
- [AddTypeSpec(name, typeSpec)](../../../../internal/domain/types.go#L191) - Adds type specification
- [AddConst(astFile, valueSpec)](../../../../internal/domain/types.go#L198) - Adds const variable from AST
- [EvaluateConstValue(file, iota, expr, evaluator, recursiveStack)](../../../../internal/domain/types.go#L217) - Evaluates constant expression value

### ConstVariable Methods

- [VariableName()](../../../../internal/domain/types.go#L313) - Gets variable name with comment override support

### Interfaces

- [ConstVariableGlobalEvaluator](../../../../internal/domain/types.go#L167) - Interface for cross-package enum evaluation
  - `EvaluateConstValue(pkg, cv, recursiveStack)`
  - `EvaluateConstValueByName(file, pkgPath, constVariableName, recursiveStack)`
  - `FindTypeSpec(typeName, file)`

### Parse Flags

- [ParseFlag](../../../../internal/domain/types.go#L355) - Alias for loader.ParseFlag
- [ParseNone, ParseModels, ParseOperations, ParseAll](../../../../internal/domain/types.go#L358) - Parse mode constants

### Utility Functions

- [EvaluateEscapedString(text)](../../../../internal/domain/types.go#L398) - Parses escaped characters in strings
- [EvaluateEscapedChar(text)](../../../../internal/domain/types.go#L378) - Parses escaped character
- [EvaluateDataConversion(x, typeName)](../../../../internal/domain/types.go#L424) - Evaluates explicit type conversions
- [EvaluateUnary(x, operator, xtype)](../../../../internal/domain/types.go#L708) - Evaluates unary expressions (-, ^)
- [EvaluateBinary(x, y, operator, xtype, ytype)](../../../../internal/domain/types.go#L751) - Evaluates binary expressions (+, -, *, /, %, &, |, ^, <<, >>)

### Utility Functions (from utils.go)

- [IsGolangPrimitiveType(typeName)](../../../../internal/domain/utils.go) - Checks if type is Go primitive
- [fullTypeName(parts...)](../../../../internal/domain/utils.go) - Builds full type name from parts
- [ignoreNameOverride(name)](../../../../internal/domain/utils.go) - Checks for name override prefix
- [nameOverride(comment)](../../../../internal/domain/utils.go) - Extracts name override from comment

## Related Packages

### Depends On
- `go/ast` - AST node types
- `go/token` - Token and position info
- `reflect` - Type reflection for evaluation
- `github.com/go-openapi/spec` - OpenAPI schema spec (for Schema type only)
- `golang.org/x/tools/go/packages` - Package loading (for Package reference only)
- [internal/loader](../../../../internal/loader) - ParseFlag type only

### Used By
- [internal/registry](../../../../internal/registry) - Uses TypeSpecDef, PackageDefinitions, AstFileInfo
- [internal/schema](../../../../internal/schema) - Uses TypeSpecDef, Schema
- [internal/parser/struct](../../../../internal/parser/struct) - Uses TypeSpecDef
- [internal/parser/route](../../../../internal/parser/route) - Uses TypeSpecDef, AstFileInfo
- [parser.go](../../../../parser.go) - Uses all domain types

## Docs

No dedicated README. Package is documented via godoc comments at top of types.go.

## Related Skills

No specific skills are directly related to this foundational package.

## Usage Example

```go
import (
    "github.com/griffnb/core-swag/internal/domain"
    "go/ast"
    "go/token"
)

// Create package definitions
pkg := domain.NewPackageDefinitions("user", "github.com/myapp/user")

// Add file to package
pkg.AddFile("user.go", astFile)

// Add type specification
typeDef := &domain.TypeSpecDef{
    File:     astFile,
    TypeSpec: typeSpec,
    PkgPath:  "github.com/myapp/user",
}
typeDef.SetSchemaName()
pkg.AddTypeSpec(typeDef.Name(), typeDef)

// Add constant
pkg.AddConst(astFile, constValueSpec)

// Evaluate constant expression
value, _ := pkg.EvaluateConstValue(
    astFile,
    0,  // iota value
    expr,
    evaluator,
    make(map[string]struct{}),  // recursive stack
)

// Access type information
fmt.Printf("Type: %s\n", typeDef.TypeName())
fmt.Printf("Schema: %s\n", typeDef.SchemaName)
fmt.Printf("Path: %s\n", typeDef.FullPath())

// Check for name override
if alias := typeDef.Alias(); alias != "" {
    fmt.Printf("Alias: %s\n", alias)
}
```

## Design Principles

1. **Pure Domain Types**: No business logic, only data structures and basic methods
2. **Shared Foundation**: Common types used across all parser components
3. **AST Integration**: Tightly coupled with Go AST for type inspection
4. **Const Evaluation**: Full support for evaluating const expressions including iota
5. **Escape Handling**: Proper handling of escaped strings and characters
6. **Type Conversion**: Complete Go primitive type conversion support
7. **Operator Support**: Full evaluation of unary and binary operators
8. **Name Overrides**: Support for custom naming via comment annotations
9. **Schema Metadata**: Tracks both simple and fully-qualified schema names

## Const Expression Evaluation

The domain package provides comprehensive const expression evaluation:

### Supported Features
- **Literals**: int, string, char (including escaped)
- **Identifiers**: References to other consts, iota
- **Selectors**: Cross-package const references (pkg.Const)
- **Unary Expressions**: Negation (-), bitwise NOT (^)
- **Binary Expressions**: +, -, *, /, %, &, |, ^, <<, >>
- **Parentheses**: Grouping with proper precedence
- **Type Conversions**: int(x), uint32(x), string(x), etc.
- **Built-ins**: len() function

### Evaluation Context
- Tracks iota value for enum sequences
- Prevents infinite recursion with recursive stack
- Resolves cross-package const references
- Handles implicit and explicit type conversions

## Name Override Support

Types and consts can override their names via comment annotations:

```go
// @name CustomUser
type User struct {
    ID string
}

const (
    // @enum active
    StatusActive = 1
)
```

The `Alias()` and `VariableName()` methods extract these overrides.

## Common Patterns

- Create PackageDefinitions early and accumulate types/consts as discovered
- Use TypeSpecDef.SetSchemaName() for standard full naming
- Use TypeSpecDef.SetSchemaNameSimple() for shorter package.Type format
- Check TypeSpecDef.NotUnique flag to determine if full path needed
- Pass ConstVariableGlobalEvaluator for cross-package const evaluation
- Maintain recursive stack during evaluation to prevent infinite loops
- Use EvaluateConstValue for enum value extraction
- Check for name overrides with Alias()/VariableName() methods
- Store parse flags in AstFileInfo to control what gets processed
