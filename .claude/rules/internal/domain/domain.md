---
paths:
  - "internal/domain/**/*.go"
---

# Domain Package

## Overview

The Domain package contains core domain types shared across the entire swag application. These types represent fundamental domain objects like type specifications, AST file information, package definitions, schema metadata, and constant evaluation. This package has minimal dependencies and serves as the shared type foundation.

## Key Structs/Methods

### Core Domain Types

- [Schema](../../../../internal/domain/types.go#L20) - OpenAPI schema wrapper with package path and definition name
- [TypeSpecDef](../../../../internal/domain/types.go#L27) - Complete type specification with AST, enums, package path, and schema metadata
- [AstFileInfo](../../../../internal/domain/types.go#L126) - AST file metadata including FileSet, path, package path, and parse flags
- [PackageDefinitions](../../../../internal/domain/types.go#L144) - Package-level definitions containing files, types, constants, and package reference
- [ConstVariable](../../../../internal/domain/types.go#L303) - Constant variable with name, type, value, comment, and file reference
- [EnumValue](../../../../internal/domain/types.go#L348) - Enum constant value with key, value, and comment

### Interfaces

- [TypeSchemaResolver](../../../../internal/domain/interfaces.go#L11) - Interface for resolving types to schemas (`GetTypeSchema`, `ParseTypeExpr`)
- [ConstVariableGlobalEvaluator](../../../../internal/domain/types.go#L167) - Interface for cross-package enum evaluation

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
- [AddConst(astFile, valueSpec)](../../../../internal/domain/types.go#L197) - Adds const variable from AST
- [EvaluateConstValue(file, iota, expr, evaluator, recursiveStack)](../../../../internal/domain/types.go#L217) - Evaluates constant expression value

### Utility Functions

- [IsGolangPrimitiveType(typeName)](../../../../internal/domain/utils.go#L48) - Checks if type is Go primitive
- [IsExtendedPrimitiveType(typeName)](../../../../internal/domain/utils.go#L74) - Checks if type is extended primitive (time.Time, UUID, decimal.Decimal)
- [TransToValidPrimitiveSchema(typeName)](../../../../internal/domain/utils.go#L125) - Converts Go primitive to OpenAPI schema
- [EvaluateEscapedString(text)](../../../../internal/domain/types.go#L398) - Parses escaped characters in strings
- [EvaluateEscapedChar(text)](../../../../internal/domain/types.go#L378) - Parses escaped character
- [EvaluateDataConversion(x, typeName)](../../../../internal/domain/types.go#L424) - Evaluates explicit type conversions
- [EvaluateUnary(x, operator, xtype)](../../../../internal/domain/types.go#L708) - Evaluates unary expressions
- [EvaluateBinary(x, y, operator, xtype, ytype)](../../../../internal/domain/types.go#L751) - Evaluates binary expressions

### Constants

- Type name constants: `ARRAY`, `OBJECT`, `PRIMITIVE`, `BOOLEAN`, `INTEGER`, `NUMBER`, `STRING`, `FUNC`, `ERROR`, `INTERFACE`, `ANY`, `NIL` (utils.go lines 13-35)
- `IgnoreNameOverridePrefix` (utils.go line 40) - Prefix character `!` for ignoring name overrides
- `EnumVarNamesExtension`, `EnumCommentsExtension`, `EnumDescriptionsExtension` (types.go lines 340-344) - Extension keys
- Parse flags: `ParseNone`, `ParseModels`, `ParseOperations`, `ParseAll` (types.go lines 359-365)
- `ParseFlag` type alias for `loader.ParseFlag` (types.go line 355)

## Related Packages

### Depends On
- `go/ast`, `go/token` - AST node types
- `github.com/go-openapi/spec` - OpenAPI schema spec (for Schema type)
- `golang.org/x/tools/go/packages` - Package reference (for PackageDefinitions)
- [internal/loader](../../../../internal/loader) - ParseFlag type only

### Used By
- [internal/registry](../../../../internal/registry) - Uses TypeSpecDef, PackageDefinitions, AstFileInfo, ConstVariable
- [internal/schema/builder.go](../../../../internal/schema/builder.go) - Uses TypeSpecDef, Schema
- [internal/parser/route/service.go](../../../../internal/parser/route/service.go) - Uses TypeSpecDef
- [internal/orchestrator](../../../../internal/orchestrator) - Name resolver tests use domain types

## Docs

No dedicated README. Package is documented via godoc comments.

## Related Skills

No specific skills are directly related to this foundational package.
