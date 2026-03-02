# Fix: Nested type package path resolution breaks for non-sibling packages

## Context

`TestCoreModelsIntegration` produces correct output (all enums, definitions present) but `TestRealProjectIntegration` and real projects are missing definitions — particularly all `constants.*` enum types (0 of them appear).

**Root cause:** In `buildSchemasRecursive()`, when a nested type is from a different package, the code constructs the package path by replacing the last segment of the parent's path:

```go
// struct_field_lookup.go:767
nestedPkgPath = pkgPath[:idx+1] + nestedPackageName
```

This assumes packages are **siblings** (share the same parent directory). This works in testdata where everything is flat under `core_models/`, but breaks in real projects:

- Account pkg: `github.com/CrowdShield/atlas-go/internal/models/account`
- Constants pkg: `github.com/CrowdShield/atlas-go/internal/constants` (NOT under `models/`)
- Bug constructs: `github.com/CrowdShield/atlas-go/internal/models/constants` (WRONG)

The full import path IS available earlier in `BuildSchema()` via `fullTypeStr` (e.g., `github.com/.../internal/constants.Role`) but gets discarded — only the short ref name `constants.Role` is passed as a nested type.

## Fix: Two files, two changes

### 1. `internal/model/struct_field.go` — `BuildSchema()` (lines ~328-331, ~359-365)

**What:** Return full import paths in `nestedTypes` instead of short ref names (when the full path is available).

**Enum refs (line 330):**
```go
// BEFORE:
nestedTypes = append(nestedTypes, refName)

// AFTER: propagate full import path for correct cross-package resolution
if strings.Contains(fullTypeStr, "/") {
    nestedTypes = append(nestedTypes, fullTypeStr)
} else {
    nestedTypes = append(nestedTypes, refName)
}
```

**Struct refs (line 365):**
```go
// BEFORE:
nestedTypes = append(nestedTypes, refName)

// AFTER: propagate full import path, including Public suffix if needed
if strings.Contains(fullTypeStr, "/") {
    nestedRef := fullTypeStr
    if public {
        nestedRef += "Public"
    }
    nestedTypes = append(nestedTypes, nestedRef)
} else {
    nestedTypes = append(nestedTypes, refName)
}
```

### 2. `internal/model/struct_field_lookup.go` — `buildSchemasRecursive()` (lines ~743-772)

**What:** Parse full import paths when present; fall back to sibling-path guessing for short names.

Replace the nested type parsing block (lines 743-772) with:

```go
for _, nestedTypeName := range nestedTypes {
    var nestedPackageName, baseNestedType, nestedPkgPath string

    if strings.Contains(nestedTypeName, "/") {
        // Full import path: extract pkg path, package name, and type name
        // e.g. "github.com/.../constants.RolePublic"
        lastDot := strings.LastIndex(nestedTypeName, ".")
        if lastDot >= 0 {
            nestedPkgPath = nestedTypeName[:lastDot]
            baseNestedType = nestedTypeName[lastDot+1:]
            if lastSlash := strings.LastIndex(nestedPkgPath, "/"); lastSlash >= 0 {
                nestedPackageName = nestedPkgPath[lastSlash+1:]
            } else {
                nestedPackageName = nestedPkgPath
            }
        }
    } else if strings.Contains(nestedTypeName, ".") {
        // Short name (e.g., "account.Properties") — existing sibling logic
        parts := strings.Split(nestedTypeName, ".")
        nestedPackageName = parts[0]
        baseNestedType = parts[len(parts)-1]
        nestedPkgPath = pkgPath
        if nestedPackageName != packageName {
            if idx := strings.LastIndex(pkgPath, "/"); idx >= 0 {
                nestedPkgPath = pkgPath[:idx+1] + nestedPackageName
            } else {
                nestedPkgPath = nestedPackageName
            }
        }
    } else {
        nestedPackageName = packageName
        baseNestedType = nestedTypeName
        nestedPkgPath = pkgPath
    }

    // ... rest of function unchanged (cleanNestedType, LookupStructFields, etc.)
```

## Files to modify

1. `internal/model/struct_field.go` — `BuildSchema()` method (~lines 328-331 and 359-365)
2. `internal/model/struct_field_lookup.go` — `buildSchemasRecursive()` (~lines 743-772)

## Verification

1. Run `TestCoreModelsIntegration` — should still pass (full paths resolve correctly for sibling packages too)
2. Run `TestRealProjectIntegration` (`make test-project-2`) — should now produce `constants.*` enum definitions
3. Compare real output definitions list — should contain `constants.Role`, `constants.Status`, `constants.OrganizationType`, etc.
4. Run full test suite: `go test ./...`
