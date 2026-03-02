# Fix: Enum Definitions Not Created for Non-Sibling Packages

## Context

In the real project, 59 `$ref`s point to `constants.*` types (e.g., `#/definitions/constants.Status`) but **none of those definitions are created** in the swagger output. The core_models integration test works because `constants` and `account` are siblings under `testdata/core_models/`. In the real project, `constants` is at `internal/constants` while `account` is at `internal/models/account` — they are NOT siblings.

The root cause is a "sibling-replace" algorithm used in two places that constructs package paths by replacing the last path segment:
```
parent: github.com/.../internal/models/account
guess:  github.com/.../internal/models/constants  ← WRONG
real:   github.com/.../internal/constants
```

## Files to Modify

| File | Action |
|------|--------|
| `internal/model/package_resolve.go` | NEW — `resolvePackagePath` helper |
| `internal/model/struct_field_lookup.go` | Replace lines 795-806 with `resolvePackagePath` call |
| `internal/model/enum_lookup.go` | Replace lines 111-113 with `resolvePackagePath` call |
| `internal/model/package_resolve_test.go` | NEW — unit tests for resolver |

## Implementation

### Step 1: Create `internal/model/package_resolve.go`

Add `resolvePackagePath(parentPkgPath, shortName string) string`:
1. Try sibling-replace first (preserves existing behavior for sibling packages)
2. Check if result exists in `globalPackageCache`
3. If not found, search `globalPackageCache` for packages whose `.Name` matches `shortName`
4. If multiple matches, prefer the one with the longest common path prefix with `parentPkgPath`
5. If still not found, check `enumPackageCache` same way
6. If nothing found, return sibling path (graceful degradation)

Also add `commonPrefixLength(a, b string) int` helper for disambiguation.

### Step 2: Fix `buildSchemasRecursive` in `struct_field_lookup.go`

Replace lines 795-806:
```go
// BEFORE (12 lines):
nestedPkgPath := pkgPath
if nestedPackageName != packageName {
    if idx := strings.LastIndex(pkgPath, "/"); idx >= 0 {
        nestedPkgPath = pkgPath[:idx+1] + nestedPackageName
    } else {
        nestedPkgPath = nestedPackageName
    }
}

// AFTER (3 lines):
nestedPkgPath := pkgPath
if nestedPackageName != packageName {
    nestedPkgPath = resolvePackagePath(pkgPath, nestedPackageName)
}
```

### Step 3: Fix `GetEnumsForType` in `enum_lookup.go`

Replace lines 111-113:
```go
// BEFORE:
lastSlash := strings.LastIndex(p.PkgPath, "/")
targetPkgPath = p.PkgPath[:lastSlash+1] + pkgPart

// AFTER:
targetPkgPath = resolvePackagePath(p.PkgPath, pkgPart)
```

### Step 4: Unit tests in `internal/model/package_resolve_test.go`

1. Sibling path found in cache → returns sibling (existing behavior)
2. Sibling NOT in cache, real path IS → returns real path (the fix)
3. Multiple matches → prefers longest common prefix
4. No match → returns sibling path (graceful degradation)
5. `commonPrefixLength` edge cases

## Verification

1. `go test ./internal/model/... -run TestResolvePackagePath` — new unit tests
2. `go test ./testing/... -run TestCoreModelsIntegration` — no regression
3. `make test-project-1` — real project should now have `constants.*` definitions
4. Check output: `grep -c '"constants\.' swagger.json` should show definitions exist
