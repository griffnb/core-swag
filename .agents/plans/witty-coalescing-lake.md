# Fix Naming/Reference Resolution - Use Fully Qualified Names

## Context

The core-swag OpenAPI generator uses short (unqualified) names for struct lookups when it should use fully qualified `package.Type` names. This causes ambiguous redirect warnings when multiple packages define types with the same name (e.g., `phone.Properties` vs `account_identity_insurance.Properties`), and incorrect `$ref` targets.

The root cause: when `StructField[T]` has a same-package type parameter, `processStructField` sets `TypeString` to just the bare type name (`Properties`) instead of `phone.Properties`. This creates unqualified `$ref` values that then need the broken redirect system to resolve.

## Changes

### Change 1: Qualify same-package types in processStructField

**File:** `internal/model/struct_field_lookup.go` lines 305-308

When `StructField[T]` has a same-package type (no `.` or `/` in the extracted type name), `TypeString` is set to just the bare name:

```go
// BEFORE (line 305-308)
} else {
    f.TypeString = arrayPrefix + subTypeName  // "Properties"
    builder.Fields = append(builder.Fields, f)
    return
}
```

Fix: Add the base package name prefix using `c.basePackage` which is already available and set during `LookupStructFields`:

```go
// AFTER
} else {
    // Same-package type - qualify with current package name
    if c.basePackage != nil {
        f.TypeString = arrayPrefix + c.basePackage.Name + "." + subTypeName
    } else {
        f.TypeString = arrayPrefix + subTypeName
    }
    builder.Fields = append(builder.Fields, f)
    return
}
```

This follows the same pattern as `getQualifiedTypeName()` (line 592-603) which already exists for non-StructField types.

### Change 2: Remove unqualified redirect system

**File:** `internal/orchestrator/schema_builder.go`

With Change 1, all `$ref` values will use `package.Type` format. The `addUnqualifiedRedirects()` function (lines 185-217) and the unqualified lookup functions become unnecessary:

- Remove `addUnqualifiedRedirects()` entirely (or convert to a debug-only check that warns if any unqualified refs are found)
- Remove the call to `s.addUnqualifiedRedirects()` at line 45
- Remove `findUnqualifiedType()` (lines 138-153) - scans all definitions by bare type name
- Remove `findTypeByShortName()` (lines 160-179) - scans definitions by Go package + type name

In `buildSchemaForRef()`, simplify the lookup to just use the registry's `FindTypeSpecByName`:
```go
// BEFORE: 3-step fallback cascade
typeDef := s.registry.FindTypeSpecByName(baseName)
if typeDef == nil {
    typeDef = s.findUnqualifiedType(baseName)
}
if typeDef == nil {
    typeDef = s.findTypeByShortName(baseName)
}

// AFTER: Direct qualified lookup only
typeDef := s.registry.FindTypeSpecByName(baseName)
```

## Files Modified

| File | Change |
|------|--------|
| `internal/model/struct_field_lookup.go` | Qualify same-package types in `processStructField` (line 305-308) |
| `internal/orchestrator/schema_builder.go` | Remove `addUnqualifiedRedirects`, `findUnqualifiedType`, `findTypeByShortName`; simplify `buildSchemaForRef` |

## Verification

1. Run `go test ./...` - all existing unit tests must pass
2. Run `make test-project-1` and `make test-project-2` - verify output
3. Verify debug output no longer shows "Ambiguous redirect" warnings
4. Check that `$ref` values in generated swagger use `package.Type` format
