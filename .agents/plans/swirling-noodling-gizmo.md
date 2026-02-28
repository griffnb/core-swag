# Fix: $ref Name Mismatch for Types with Name Collisions

## Context

When multiple packages define a type with the same short name (e.g., `enum.Source` exists in 3 different chargebee sub-packages), the registry marks them as `NotUnique` and stores definitions with full-path names like `github_com_chargebee_chargebee-go_v3_enum.Source`.

However, `buildSchemaForType()` in `struct_field.go` normalizes all type names to short form (`enum.Source`) via `normalizeTypeName()` before creating `$ref` references. This creates a mismatch: the `$ref` points to `#/definitions/enum.Source` but the definition is stored as `github_com_chargebee_chargebee-go_v3_enum.Source`.

**Affected output**: `testing/real_actual_output.json` has `$ref: "#/definitions/enum.Source"` (lines 147779, 469156) but no `enum.Source` definition exists. The actual definition is at `github_com_chargebee_chargebee-go_v3_enum.Source` (line 165386).

## Root Cause

`normalizeTypeName()` at `internal/model/struct_field.go:140` strips the full module path from type strings (e.g., `github.com/chargebee/chargebee-go/v3/enum.Source` → `enum.Source`). This loses the information needed to match the definition name when `NotUnique=true`.

## Fix Strategy: Two-Part Fix

### Part 1: `buildSchemaForType()` — Use full-path ref names for module-qualified types

**File**: `internal/model/struct_field.go`

Before normalization at line 228, save the original full type string. When creating a `$ref` (both enum refs at line 324 and struct refs at line 363), if the original type has a full module path (contains `/`), compute the full-path definition-format name and use it for the `$ref`.

**Add utility function** `makeFullPathDefinitionName(fullTypeStr string) string`:
- Input: `github.com/chargebee/chargebee-go/v3/enum.Source`
- Split at last `.` → pkgPath: `github.com/chargebee/chargebee-go/v3/enum`, typeName: `Source`
- Replace `\`, `/`, `.` in pkgPath with `_` → `github_com_chargebee_chargebee-go_v3_enum`
- Return: `github_com_chargebee_chargebee-go_v3_enum.Source`
- This matches the algorithm in `TypeSpecDef.TypeName()` at `internal/domain/types.go:62-67`

Changes in `buildSchemaForType()`:
```go
// Line 228 area:
fullTypeStr := typeStr  // Save before normalization
typeStr = normalizeTypeName(typeStr)

// Line 324 area (enum $ref):
refName := typeStr
if strings.Contains(fullTypeStr, "/") {
    refName = makeFullPathDefinitionName(fullTypeStr)
}

// Line 363 area (struct $ref):
refName := typeName
if strings.Contains(fullTypeStr, "/") {
    refName = makeFullPathDefinitionName(fullTypeStr)
}
```

### Part 2: Orchestrator — Add redirect definitions for unique types

**File**: `internal/orchestrator/service.go`

After syncing schemas to swagger (after line 399), for each type in `uniqueDefinitions`:
- Compute the `fullPathDefName` using the same algorithm
- If `fullPathDefName != actualDefinitionName` (type is unique and uses short name), add a redirect definition

```go
// For each uniqueDef, ensure full-path name resolves
for _, typeDef := range uniqueDefs {
    if typeDef == nil { continue }
    actualName := typeDef.TypeName()
    fullPathName := makeFullPathDefName(typeDef.PkgPath, typeDef.Name())
    if fullPathName != actualName {
        if _, exists := s.swagger.Definitions[fullPathName]; !exists {
            // Add $ref redirect so full-path refs resolve to the short-name definition
            s.swagger.Definitions[fullPathName] = spec.Schema{
                SchemaProps: spec.SchemaProps{
                    Ref: spec.MustCreateRef("#/definitions/" + actualName),
                },
            }
        }
    }
}
```

This ensures:
- **NotUnique types** (e.g., `enum.Source`): $ref uses full-path name `github_com_..._enum.Source`, definition exists with that name → match
- **Unique types** (e.g., `enum.ApiVersion`): $ref uses full-path name `github_com_..._enum.ApiVersion`, redirect definition points to `enum.ApiVersion` → resolves correctly

### Part 3: Handle `fullTypeStr` propagation through arrays/maps

When `buildSchemaForType` recurses for array elements (`[]enum.Source`) or map values, the `fullTypeStr` needs to also be trimmed of `[]` or `map[...]` prefix. Handle this in the array and map branches.

## Files to Modify

1. **`internal/model/struct_field.go`** — Add `makeFullPathDefinitionName()`, modify `buildSchemaForType()` to use full-path refs
2. **`internal/orchestrator/service.go`** — Add redirect definitions after syncing schemas
3. **`internal/orchestrator/ref_reconciler.go`** (new, small file) — Utility function for building full-path definition names from orchestrator context

## Verification

1. Run `make test-project-1` and check `testing/real_actual_output.json`:
   - `event.Event.source` `$ref` should point to `github_com_chargebee_chargebee-go_v3_enum.Source`
   - That definition should exist
   - `event.Event.api_version` `$ref` should point to a full-path name that resolves (either directly or via redirect)
2. Run existing unit tests: `go test ./internal/model/... -v`
3. Run integration test: `go test ./testing/ -run TestCoreModelsIntegration -v`
4. Verify no broken $refs exist: search for all `$ref` values in output and confirm each referenced definition exists
