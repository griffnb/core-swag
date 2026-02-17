# Struct Parser Bug Fixes - Design Document

**Date:** 2026-02-16
**Status:** Approved
**Author:** Claude (Brainstorming Skill)

---

## Executive Summary

Fix 5 critical bugs in struct parser schema generation that cause incorrect OpenAPI schemas in `make test-project-1` output. All fixes target `internal/model/struct_field.go` and `internal/model/struct_field_lookup.go`. **Zero failing tests at completion.**

---

## Problem Statement

The current swagger output from `make test-project-1` contains the following bugs:

1. **Private fields leak into schemas**: BaseModel fields without json/column tags (`ChangeLogs`, `Client`, `ManualCache`, `SavingUser`) appear in generated schemas
2. **Array element types wrong**: `division_ids []string` generates as `array` of `object` instead of `array` of `string`
3. **Any type generates empty schema**: `external_user_info any` generates `{}` instead of `{"type": "object"}`
4. **Enum constants show as objects**: `constants.NJDLClassification` generates `{"type": "object"}` instead of inline enum values
5. **Public schemas reference non-Public nested types**: `AccountJoinedSessionPublic` references `classification.JoinedClassification` instead of `classification.JoinedClassificationPublic`

---

## Architecture Overview

### Approach

Fix bugs at their source in the canonical struct parsing implementation:
- **Primary location**: `internal/model/struct_field.go` - `ToSpecSchema()` method
- **Secondary location**: `internal/model/struct_field_lookup.go` - `ExtractFieldsRecursive()` method

### Why These Locations

The consolidation work proved that `struct_field.go` is the canonical implementation:
- Used by CoreStructParser (schema builder)
- Used by StructParserService (orchestrator)
- ~538 lines, well-tested, high confidence
- Fixing here ensures both code paths benefit immediately

### Risk Mitigation

- **TDD approach**: Write failing tests first for each bug
- **Incremental fixes**: One bug at a time with test verification
- **Regression prevention**: All existing tests must pass
- **Real-world validation**: Verify `make test-project-1` output

---

## Detailed Bug Fixes

### Bug 1: Embedded Field Filtering

**Problem:** BaseModel fields without json/column tags leak into schemas.

**Example:**
```go
type BaseModel struct {
    ChangeLogs  bool   // No json/column tag - should be excluded
    Client      *Client // No json/column tag - should be excluded
}
```

**Root Cause:** Fields without serialization tags are being processed.

**Fix Location:** `internal/model/struct_field_lookup.go` in `ExtractFieldsRecursive()` (line ~358-361)

**Solution Logic:**
```go
// Field must have EITHER json tag OR column tag to be included
jsonTag := tagMap["json"]
columnTag := tagMap["column"]

// Skip if json tag is explicitly "-"
if jsonTag == "-" {
    continue
}

// Skip if column tag is explicitly "-"
if columnTag == "-" {
    continue
}

// Skip if NEITHER json nor column tag exists (or both are empty)
if (jsonTag == "" || jsonTag == "-") && (columnTag == "" || columnTag == "-") {
    continue
}
```

**Test Cases:**
- Field with no json/column tags → excluded
- Field with `json:"name"` → included
- Field with `column:"db_name"` → included
- Field with `json:"-"` → excluded
- Field with `column:"-"` → excluded

---

### Bug 2: Array Element Type Resolution

**Problem:** `[]string` becomes `array` of `object` instead of `array` of `string`.

**Example:**
```go
DivisionIDs []string `json:"division_ids"`
// Current: {"type": "array", "items": {"type": "object"}}
// Expected: {"type": "array", "items": {"type": "string"}}
```

**Root Cause:** Array element type resolution doesn't properly extract the element type - treats `string` as unknown type.

**Fix Location:** `internal/model/struct_field.go` in `ToSpecSchema()` array handling (line ~258-294)

**Solution Logic:**
```go
// When building array schema, resolve element type properly
// Check: is it a basic type (string, int, bool)?
// Use type information, not just type name string matching
if isPrimitiveType(elemType) {
    items.Type = getPrimitiveSchemaType(elemType)
} else if isExtendedPrimitive(elemType) {
    // Handle time.Time, UUID, etc.
    items = getExtendedPrimitiveSchema(elemType)
}
```

**Test Cases:**
- `[]string` → `{"type": "array", "items": {"type": "string"}}`
- `[]int` → `{"type": "array", "items": {"type": "integer"}}`
- `[]*User` → `{"type": "array", "items": {"$ref": "#/definitions/User"}}`

---

### Bug 3: Any Type Handling

**Problem:** Fields with type `any` or `interface{}` generate empty schema `{}`.

**Example:**
```go
ExternalUserInfo any `json:"external_user_info"`
// Current: {}
// Expected: {"type": "object"}
```

**Root Cause:** Unknown types return empty schema instead of defaulting to object type.

**Fix Location:** `internal/model/struct_field.go` in `ToSpecSchema()` type resolution (line ~234-376)

**Solution Logic:**
```go
// Before returning empty schema, check if type is any/interface{}
if isAnyType(typeStr) {
    schema.Type = []string{"object"}
    return schema, nestedTypes, nil
}
```

**Test Cases:**
- Field with `any` type → `{"type": "object"}`
- Field with `interface{}` type → `{"type": "object"}`

---

### Bug 4: Enum Detection with Underlying Type Check

**Problem:** Enum constants like `NJDLClassification` (underlying type `int`) show as `{"type": "object"}` instead of inline enum values or primitive.

**Example:**
```go
type NJDLClassification int
const (
    NJ_DL_LAW_ENFORCEMENT NJDLClassification = 1
    NJ_DL_PROSECUTOR      NJDLClassification = 2
)
// Current: {"type": "object"}
// Expected: {"type": "integer", "enum": [1, 2], ...}
```

**Root Cause:** Not checking the underlying type to determine if it's an enum vs struct vs type alias.

**Fix Location:** `internal/model/struct_field.go` in `ToSpecSchema()` type resolution (line ~303-340)

**Solution Logic:**
```go
// For named types, check the underlying type first
if named, ok := fieldType.(*types.Named); ok {
    underlying := named.Underlying()

    // If underlying is struct → look it up as struct
    if _, isStruct := underlying.(*types.Struct); isStruct {
        // Continue with struct handling
    }

    // If underlying is primitive → check if it's an enum
    if isPrimitive(underlying) {
        // Try enum lookup first
        if enumLookup != nil {
            if values, err := enumLookup.GetEnumsForType(typeName, nil); err == nil && len(values) > 0 {
                // Inline enum values
                return enumSchema, nestedTypes, nil
            }
        }
        // Not an enum, use the underlying primitive type (handles type aliases)
        return primitiveSchemaFromUnderlying(underlying), nestedTypes, nil
    }
}
```

**Test Cases:**
- Enum constant with `int` underlying → `{"type": "integer", "enum": [...]}`
- Type alias `type MyString string` → `{"type": "string"}`
- Regular struct type → `{"$ref": "#/definitions/TypeName"}`

---

### Bug 5: Public Reference Propagation

**Problem:** Public schemas reference non-Public nested struct types.

**Example:**
```go
// In AccountJoinedSessionPublic schema:
"classifications": {
    "type": "array",
    "items": {
        "$ref": "#/definitions/classification.JoinedClassification" // Wrong
        // Should be: "#/definitions/classification.JoinedClassificationPublic"
    }
}
```

**Root Cause:** When generating `$ref` for nested structs, the "Public" suffix isn't added even when parent is in public mode.

**Fix Location:** `internal/model/struct_field.go` in `ToSpecSchema()` reference generation (line ~296-376)

**Solution Logic:**
```go
// When creating $ref for nested struct in public mode:
refName := packagePath + "." + typeName
if public {
    refName += "Public"
}
schema.Ref = spec.MustCreateRef("#/definitions/" + refName)
```

**Test Cases:**
- Public schema with nested struct → `$ref` points to `TypePublic`
- Non-public schema with nested struct → `$ref` points to `Type`
- Public schema with array of nested structs → array items `$ref` points to `TypePublic`

---

## Testing Strategy

### Test-Driven Development

**Use `Skill(testing)` to create proper Go unit tests** for each bug fix.

### Test Location

Add tests to existing file: `internal/model/struct_field_test.go`

### Test Coverage

Create tests for:
1. Embedded fields without json/column tags
2. Array element type resolution ([]string, []int, []*Struct)
3. Any type handling (interface{})
4. Enum detection with underlying type check
5. Public reference propagation

### Pre-existing Test Failures - MUST FIX

**Currently failing:**
- `TestBuildAllSchemas_BillingPlan`
- `TestBuildAllSchemas_Account`

**Approach:** Investigate and fix as part of this effort. Likely caused by the same root bugs.

### Success Criteria

**ALL tests must pass:**
- ✅ All new bug tests pass
- ✅ All existing struct_field_test.go tests pass
- ✅ `TestBuildAllSchemas_BillingPlan` passes
- ✅ `TestBuildAllSchemas_Account` passes
- ✅ All struct_builder tests pass (26 tests)
- ✅ `make test-project-1` generates valid swagger.json
- ✅ **Zero test failures in entire test suite**

### Integration Validation

Verify `make test-project-1` output:
- `account.AccountJoinedFull` has no BaseModel fields without tags
- `account.Properties.division_ids` is `string[]` not `object[]`
- `account.Properties.external_user_info` is `{"type": "object"}`
- `constants.NJDLClassification` has enum values
- Public schemas reference `TypePublic` variants

---

## Implementation Approach

### Phase 1: Write Failing Tests
Use `Skill(testing)` to create unit tests for each bug. Tests should fail initially.

### Phase 2: Fix Bugs One-by-One
Fix each bug in order, verifying tests pass after each fix:
1. Bug 1: Embedded field filtering
2. Bug 2: Array element type resolution
3. Bug 3: Any type handling
4. Bug 4: Enum detection
5. Bug 5: Public reference propagation

### Phase 3: Fix Pre-existing Failures
Investigate and fix `TestBuildAllSchemas_BillingPlan` and `TestBuildAllSchemas_Account`.

### Phase 4: Integration Verification
Run `make test-project-1` and verify all issues resolved in actual output.

### Phase 5: Final Test Suite Verification
Run full test suite and confirm **zero failures**.

---

## Files to Modify

### Primary Files
- `internal/model/struct_field.go` - Main schema generation logic
- `internal/model/struct_field_lookup.go` - Field extraction logic
- `internal/model/struct_field_test.go` - Add new test cases

### Helper Functions to Add/Modify
- `isAnyType(typeStr string) bool` - Detect any/interface{} types
- `isPrimitive(underlying types.Type) bool` - Check if underlying type is primitive
- `primitiveSchemaFromUnderlying(underlying types.Type) *spec.Schema` - Get schema from underlying type

---

## Risk Assessment

### Low Risk
- Changes are surgical and focused
- TDD approach catches regressions early
- Existing test suite provides safety net

### Medium Risk
- Array element type resolution may have edge cases (nested arrays, maps)
- Enum detection timing may affect other type checks

### Mitigation
- Test each edge case thoroughly
- Run full test suite after each fix
- Verify with real project output

---

## Success Metrics

1. **Zero test failures** across entire codebase
2. **Valid swagger.json** generated by `make test-project-1`
3. **Correct schemas** for Account.AccountJoinedFull and related types
4. **No regressions** in existing functionality
5. **Improved test coverage** with 5+ new test cases

---

## Next Steps

After design approval:
1. Invoke `Skill(superpowers:writing-plans)` to create detailed implementation plan
2. Execute implementation following TDD approach
3. Verify all success criteria met
