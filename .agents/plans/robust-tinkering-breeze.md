# Fix: Route $ref lookups must use fully qualified package paths

## Context

Route annotations reference types by short names (e.g., `@Success 200 {object} Address`). The route parser qualifies these with just the controller's Go package name (e.g., `address.Address`), but never resolves the full import path. When the registry has a name collision (e.g., project's `address.Address` vs chargebee-go's `address.Address`), the short name key is deleted from the registry and replaced with full-path keys. The route's short-name lookup then fails with "Skipping unknown ref".

**Current flow (broken for collisions):**
```
annotation "Address" → route parser → "address.Address" → registry lookup → nil (collision deleted key)
```

**Target flow (always unambiguous):**
```
annotation "Address" → route parser resolves import → full pkg path stored in route → registry lookup by full path → found
```

## Approach: Resolve imports at route parse time

The route parser already receives the `ast.File` (which contains import declarations) and the registry (which has `findPackagePathFromImports()`). The fix is to:

1. Pass the `ast.File` into the `operation` struct so type qualification methods can access imports
2. Add a method to resolve a type name to its full import path using the file's imports
3. Store the full import path in a new field on `routedomain.Schema` alongside the existing `Ref`
4. Use the full import path for registry lookups in `buildSchemaForRef()`

The swagger `$ref` output string still uses the short readable name (e.g., `#/definitions/address.Address`), but internally the system uses the full path for unambiguous registry lookups.

## Files to modify

### 1. `internal/parser/route/domain/route.go` — Add `TypePath` field to Schema

Add a field to store the resolved full import path alongside `Ref`:

```go
type Schema struct {
    Type                 string
    Ref                  string            // e.g., "#/definitions/address.Address"
    TypePath             string            // e.g., "github.com/.../address.Address" (for registry lookup)
    Items                *Schema
    Properties           map[string]*Schema
    AdditionalProperties *Schema
    AllOf                []*Schema
    // ...
}
```

### 2. `internal/parser/route/service.go` — Pass `ast.File` to operation

Update `parseOperation()` to store the `ast.File` on the operation:

```go
type operation struct {
    // ... existing fields ...
    astFile      *ast.File // For import resolution
}

func (s *Service) parseOperation(funcDecl *ast.FuncDecl, packageName string, filePath string, fset *token.FileSet) *operation {
    op := &operation{
        // ... existing ...
        astFile: nil, // set below
    }
    // ...
}
```

And pass the astFile from `ParseRoutes()`:

```go
func (s *Service) ParseRoutes(astFile *ast.File, ...) {
    // ...
    operation := s.parseOperation(funcDecl, packageName, filePath, fset)
    operation.astFile = astFile
    // ...
}
```

### 3. `internal/parser/route/service.go` — Add `resolveTypePath` method

New method that uses the registry to resolve a short type name to its full import path:

```go
// resolveTypePath resolves a qualified type name (e.g., "address.Address") to its
// full import path (e.g., "github.com/.../address.Address") using the file's imports.
// Returns empty string if resolution fails (caller should fall back to short name).
func (s *Service) resolveTypePath(qualifiedType string, file *ast.File) string {
    if s.registry == nil || file == nil {
        return ""
    }
    typeDef := s.registry.FindTypeSpec(qualifiedType, file)
    if typeDef == nil {
        return ""
    }
    return typeDef.FullPath()
}
```

### 4. `internal/parser/route/response.go` — Set TypePath on schema refs

In `buildSchemaForTypeWithPublic()`, after building the `$ref` schema, resolve and store the full path:

```go
func (s *Service) buildSchemaForTypeWithPublic(dataType, packageName string, isPublic bool) *routedomain.Schema {
    // ... existing code to build qualifiedType and ref ...

    ref := "#/definitions/" + qualifiedType
    schema := &routedomain.Schema{Ref: ref}

    // Resolve full import path for unambiguous registry lookup.
    // Strip Public suffix for resolution since the registry stores base types.
    lookupType := qualifiedType
    if strings.HasSuffix(lookupType, "Public") {
        lookupType = strings.TrimSuffix(lookupType, "Public")
    }
    if typePath := s.resolveTypePath(lookupType, op.astFile); typePath != "" {
        schema.TypePath = typePath
        if isPublic && s.isStructType(lookupType) {
            schema.TypePath += "Public"
        }
    }

    return schema
}
```

**Problem:** `buildSchemaForTypeWithPublic` doesn't have access to `op`. We need to thread it through.

**Revised approach:** Add `astFile *ast.File` parameter to the build methods, or refactor the schema building to be a method on `operation`:

Update the call chain:
- `buildSchemaWithPackageAndPublic(schemaType, dataType, packageName, isPublic)` → add `file *ast.File` parameter
- `buildSchemaForTypeWithPublic(dataType, packageName, isPublic)` → add `file *ast.File` parameter
- `buildAllOfResponseSchema(dataType, packageName, isPublic)` → add `file *ast.File` parameter

All callers pass `op.astFile`.

### 5. `internal/parser/route/parameter.go` — Set TypePath on parameter schema refs

Same pattern for body parameter refs:

```go
if paramType == "body" && isModelType(dataType) {
    // ... existing qualification ...
    param.Schema = &domain.Schema{
        Ref: "#/definitions/" + qualifiedType,
    }
    // Resolve full path
    if typePath := s.resolveTypePath(qualifiedType, op.astFile); typePath != "" {
        param.Schema.TypePath = typePath
    }
}
```

**Note:** `parseParam` also doesn't have access to `op.astFile` — need to pass it through. Change `parseParam(op *operation, line string)` — it already has `op`, so we can use `op.astFile`.

### 6. `internal/orchestrator/refs.go` — Collect TypePath alongside ref names

Update `CollectReferencedTypes` to return full paths when available:

```go
// RefInfo holds both the short ref name and the resolved full import path.
type RefInfo struct {
    Source   string // human-readable source location
    TypePath string // full import path (empty if unresolved)
}

func CollectReferencedTypes(routes []*routedomain.Route) map[string]RefInfo {
    refs := make(map[string]RefInfo)
    // ... walk routes, collect refs with TypePath ...
}
```

Update `collectRefsFromSchema` to propagate `TypePath`.

### 7. `internal/orchestrator/schema_builder.go` — Use TypePath for registry lookup

Update `buildSchemaForRef` to use the full path when available:

```go
func (s *Service) buildSchemaForRef(refName string, info RefInfo, processed map[string]bool) error {
    // ... existing processed checks ...

    baseName := refName
    if strings.HasSuffix(refName, "Public") {
        baseName = strings.TrimSuffix(refName, "Public")
    }

    // Try full-path lookup first (unambiguous)
    var typeDef *domain.TypeSpecDef
    if info.TypePath != "" {
        lookupPath := info.TypePath
        if strings.HasSuffix(lookupPath, "Public") {
            lookupPath = strings.TrimSuffix(lookupPath, "Public")
        }
        typeDef = s.registry.FindTypeSpecByFullPath(lookupPath)
    }

    // Fall back to short name lookup
    if typeDef == nil {
        typeDef = s.registry.FindTypeSpecByName(baseName)
    }

    if typeDef == nil {
        // Log warning and skip
        return nil
    }
    // ... rest unchanged ...
}
```

### 8. `internal/registry/service.go` — Add `FindTypeSpecByFullPath` method

New method for unambiguous lookup by full import path:

```go
// FindTypeSpecByFullPath looks up a type by its full import path + type name
// (e.g., "github.com/CrowdShield/atlas-go/internal/models/address.Address").
// This is unambiguous and bypasses the NotUnique collision issue.
func (s *Service) FindTypeSpecByFullPath(fullPath string) *domain.TypeSpecDef {
    // Split into package path and type name
    lastDot := strings.LastIndex(fullPath, ".")
    if lastDot < 0 {
        return nil
    }
    pkgPath := fullPath[:lastDot]
    typeName := fullPath[lastDot+1:]
    return s.findTypeSpec(pkgPath, typeName)
}
```

This uses the existing `packages` map which stores types by `pkgPath → TypeDefinitions[typeName]`, completely bypassing the `uniqueDefinitions` collision issue.

### 9. Clean up: Remove `FindTypeSpecByName` fallback (optional)

After this change, the `isProjectLocal` fallback in `FindTypeSpecByName` becomes unnecessary for route refs. It can be kept as a safety net or removed to simplify the code. I recommend keeping it for now since other callers may still use short-name lookups.

## Files summary

| File | Change |
|------|--------|
| `internal/parser/route/domain/route.go` | Add `TypePath` field to `Schema` |
| `internal/parser/route/service.go` | Store `astFile` on `operation`, add `resolveTypePath()` |
| `internal/parser/route/response.go` | Thread `*ast.File` through build methods, set `TypePath` |
| `internal/parser/route/parameter.go` | Set `TypePath` on body param schema refs |
| `internal/parser/route/allof.go` | Thread `*ast.File` through, set `TypePath` on allOf refs |
| `internal/orchestrator/refs.go` | Collect `TypePath` alongside ref names |
| `internal/orchestrator/schema_builder.go` | Use `TypePath` for registry lookup when available |
| `internal/registry/service.go` | Add `FindTypeSpecByFullPath()` method |

## Verification

1. `go test ./internal/parser/route/...` — existing route parser tests still pass
2. `go test ./internal/registry/...` — existing + new registry tests pass
3. `go test ./internal/orchestrator/...` — existing orchestrator tests pass
4. `cd testing && go test -run TestRealProjectIntegration -v` — zero "Skipping unknown ref" for real struct types
5. `make test-project-2` — verify output has all expected definitions
6. `go test ./...` — full suite passes
