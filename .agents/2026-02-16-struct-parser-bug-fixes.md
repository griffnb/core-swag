# Struct Parser Bug Fixes Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix 5 critical bugs in struct parser schema generation to produce correct OpenAPI schemas with zero test failures.

**Architecture:** Fix bugs at their source in `internal/model/struct_field.go` (ToSpecSchema method) and `internal/model/struct_field_lookup.go` (ExtractFieldsRecursive method). Use TDD approach: write failing test, implement minimal fix, verify all tests pass, commit.

**Tech Stack:** Go 1.21+, go/types, go-openapi/spec, testify

---

## Phase 1: Fix Bug 1 - Embedded Field Filtering

### Task 1.1: Write test for fields without json/column tags

**Files:**
- Modify: `internal/model/struct_field_test.go`

**Step 1: Write failing test using @superpowers:testing skill**

Use `Skill(testing)` to create a test that verifies fields without json/column tags are excluded from schemas.

Test requirements:
- Create synthetic struct with embedded BaseModel-like type
- BaseModel fields have NO json or column tags (ChangeLogs, Client, etc.)
- Some fields have json tags (should be included)
- Some fields have column tags (should be included)
- Verify excluded fields NOT in schema.Properties
- Verify included fields ARE in schema.Properties

Expected test structure:
```go
// Test struct setup
type TestBaseModel struct {
    ChangeLogs  bool   // No tags - should be excluded
    Client      string // No tags - should be excluded
    ValidJSON   string `json:"valid_json"`    // Has json tag - included
    ValidColumn string `column:"valid_column"` // Has column tag - included
    ExcludeJSON string `json:"-"` // Explicitly excluded
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/model -run "TestEmbedded.*Tags" -v`

Expected: FAIL - fields without tags currently appear in schema

**Step 3: Implement fix in ExtractFieldsRecursive**

Modify: `internal/model/struct_field_lookup.go` at line ~358-361

Add filtering logic after tag parsing:

```go
// After line ~350 where tags are parsed:
tagPairs := strings.Split(tag, ":")
tagMap := make(map[string]string)
for j := 0; j < len(tagPairs)-1; j += 2 {
    key := tagPairs[j]
    value := strings.Trim(tagPairs[j+1], "\"")
    tagMap[key] = value
}

// ADD THIS FILTERING LOGIC:
jsonTag := tagMap["json"]
columnTag := tagMap["column"]

// Skip if json tag is explicitly "-"
if jsonTag == "-" {
    console.Logger.Debug("Skipping field %s because json tag is '-'\n", fieldName)
    continue
}

// Skip if column tag is explicitly "-"
if columnTag == "-" {
    console.Logger.Debug("Skipping field %s because column tag is '-'\n", fieldName)
    continue
}

// Skip if NEITHER json nor column tag exists or both are empty
if (jsonTag == "" || jsonTag == "-") && (columnTag == "" || columnTag == "-") {
    console.Logger.Debug("Skipping field %s because it has no json or column tag\n", fieldName)
    continue
}

// EXISTING CODE: Check if json tag is "-"
if tagMap["json"] == "-" {
    console.Logger.Debug("Skipping field %s because json tag is '-'\n", fieldName)
    continue
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/model -run "TestEmbedded.*Tags" -v`

Expected: PASS - fields without tags excluded, fields with tags included

**Step 5: Run all model tests to check for regressions**

Run: `go test ./internal/model -v`

Expected: All tests pass (no new failures)


## Phase 2: Fix Bug 2 - Array Element Type Resolution

### Task 2.1: Write test for array element type resolution

**Files:**
- Modify: `internal/model/struct_field_test.go`

**Step 1: Write failing test using @superpowers:testing skill**

Use `Skill(testing)` to create a test that verifies array element types are correctly resolved.

Test requirements:
- Field with `[]string` type
- Field with `[]int` type
- Field with `[]*StructType` type
- Verify schema has correct array type with correct items type
- `[]string` → `{"type": "array", "items": {"type": "string"}}`
- `[]int` → `{"type": "array", "items": {"type": "integer"}}`
- `[]*User` → `{"type": "array", "items": {"$ref": "#/definitions/User"}}`

**Step 2: Run test to verify it fails**

Run: `go test ./internal/model -run "TestArray.*Type" -v`

Expected: FAIL - array items show as {"type": "object"} instead of correct type

**Step 3: Add helper function for primitive type checking**

Modify: `internal/model/struct_field.go` after line ~230 (before ToSpecSchema)

Add helper function:

```go
// isPrimitiveBasicType checks if a types.Type is a basic primitive (string, int, bool, float, etc.)
func isPrimitiveBasicType(t types.Type) bool {
    if t == nil {
        return false
    }

    // Handle named types by checking underlying
    if named, ok := t.(*types.Named); ok {
        t = named.Underlying()
    }

    basic, ok := t.(*types.Basic)
    if !ok {
        return false
    }

    // Check if it's a basic type (not unsafe.Pointer or untyped)
    info := basic.Info()
    return (info & (types.IsBoolean | types.IsInteger | types.IsFloat | types.IsString)) != 0
}

// getPrimitiveTypeString converts types.BasicKind to OpenAPI type string
func getPrimitiveTypeString(t types.Type) string {
    if named, ok := t.(*types.Named); ok {
        t = named.Underlying()
    }

    basic, ok := t.(*types.Basic)
    if !ok {
        return ""
    }

    switch basic.Kind() {
    case types.Bool:
        return "boolean"
    case types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
        types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64:
        return "integer"
    case types.Float32, types.Float64:
        return "number"
    case types.String:
        return "string"
    default:
        return ""
    }
}
```

**Step 4: Fix array element type resolution in ToSpecSchema**

Modify: `internal/model/struct_field.go` at line ~258-294 (array handling section)

Find the section that handles array types and update it:

```go
// Around line 258-294, find the array handling code
// Look for: if strings.HasPrefix(typeStr, "[]")

// Replace the array item type resolution with:
if strings.HasPrefix(typeStr, "[]") {
    schema.Type = []string{"array"}
    items := spec.Schema{}

    // Get the element type (after removing [])
    elemTypeStr := strings.TrimPrefix(typeStr, "[]")

    // Try to get actual type information for the element
    var elemType types.Type
    if sf.Type != nil {
        if sliceType, ok := sf.Type.(*types.Slice); ok {
            elemType = sliceType.Elem()
        }
    }

    // If we have type information, use it
    if elemType != nil {
        // Check if it's a basic primitive first
        if isPrimitiveBasicType(elemType) {
            items.Type = []string{getPrimitiveTypeString(elemType)}
        } else if isExtendedPrimitive(elemType.String()) {
            // Handle extended primitives (time.Time, UUID, etc.)
            extSchema := getExtendedPrimitiveSchema(elemType.String())
            items.Type = extSchema.Type
            if extSchema.Format != "" {
                items.Format = extSchema.Format
            }
        } else {
            // It's a struct or named type - create a reference
            // Extract package and type name for $ref
            typeName := elemType.String()
            if strings.Contains(typeName, ".") {
                // Has package path
                items.Ref = spec.MustCreateRef("#/definitions/" + typeName)
                nestedTypes = append(nestedTypes, typeName)
            } else {
                // Local type
                items.Type = []string{"object"}
            }
        }
    } else {
        // Fallback to string-based resolution (existing code)
        // ... keep existing fallback logic ...
    }

    schema.Items = &spec.SchemaOrArray{Schema: &items}
    return schema, nestedTypes, nil
}
```

**Step 5: Run test to verify it passes**

Run: `go test ./internal/model -run "TestArray.*Type" -v`

Expected: PASS - array items have correct types (string, integer, $ref)

**Step 6: Run all model tests to check for regressions**

Run: `go test ./internal/model -v`

Expected: All tests pass (no new failures)


---

## Phase 3: Fix Bug 3 - Any Type Handling

### Task 3.1: Write test for any/interface{} type handling

**Files:**
- Modify: `internal/model/struct_field_test.go`

**Step 1: Write failing test using @superpowers:testing skill**

Use `Skill(testing)` to create a test that verifies any/interface{} types generate object schema.

Test requirements:
- Field with `any` type
- Field with `interface{}` type
- Verify schema is `{"type": "object"}` not `{}`

**Step 2: Run test to verify it fails**

Run: `go test ./internal/model -run "TestAny.*Type" -v`

Expected: FAIL - any/interface{} currently generates empty schema {}

**Step 3: Add helper function to detect any/interface{} types**

Modify: `internal/model/struct_field.go` after primitive type helpers

Add helper function:

```go
// isAnyType checks if a type is any or interface{}
func isAnyType(typeStr string) bool {
    if typeStr == "" {
        return false
    }

    // Check for "any" keyword (Go 1.18+)
    if typeStr == "any" {
        return true
    }

    // Check for "interface{}" or "interface {}"
    normalized := strings.ReplaceAll(typeStr, " ", "")
    if normalized == "interface{}" {
        return true
    }

    return false
}
```

**Step 4: Add any type handling in ToSpecSchema**

Modify: `internal/model/struct_field.go` at line ~234-376 (early in type resolution)

Add check near the beginning of type resolution logic:

```go
// Around line 240, after getting typeStr
typeStr := sf.TypeString
if typeStr == "" && sf.Type != nil {
    typeStr = sf.Type.String()
}

// ADD THIS CHECK:
// Handle any/interface{} types as object
if isAnyType(typeStr) {
    schema.Type = []string{"object"}
    return schema, nestedTypes, nil
}

// Continue with existing type resolution...
```

**Step 5: Run test to verify it passes**

Run: `go test ./internal/model -run "TestAny.*Type" -v`

Expected: PASS - any/interface{} generates {"type": "object"}

**Step 6: Run all model tests to check for regressions**

Run: `go test ./internal/model -v`

Expected: All tests pass (no new failures)

---

## Phase 4: Fix Bug 4 - Enum Detection with Underlying Type

### Task 4.1: Write test for enum detection

**Files:**
- Modify: `internal/model/struct_field_test.go`

**Step 1: Write failing test using @superpowers:testing skill**

Use `Skill(testing)` to create a test that verifies enum constants with primitive underlying types are detected correctly.

Test requirements:
- Enum constant type with int underlying (like NJDLClassification)
- Type alias like `type MyString string`
- Regular struct type
- Verify enum generates {"type": "integer", "enum": [values]}
- Verify type alias generates {"type": "string"}
- Verify struct generates {"$ref": "#/definitions/Type"}

**Step 2: Run test to verify it fails**

Run: `go test ./internal/model -run "TestEnum.*Underlying" -v`

Expected: FAIL - enums currently show as {"type": "object"}

**Step 3: Add helper function to check if underlying type is primitive**

Modify: `internal/model/struct_field.go` after other helper functions

Add helper:

```go
// isPrimitiveUnderlying checks if the underlying type of a named type is a primitive
func isPrimitiveUnderlying(underlying types.Type) bool {
    if underlying == nil {
        return false
    }

    basic, ok := underlying.(*types.Basic)
    if !ok {
        return false
    }

    // Check if it's a basic type
    info := basic.Info()
    return (info & (types.IsBoolean | types.IsInteger | types.IsFloat | types.IsString)) != 0
}

// primitiveSchemaFromUnderlying creates a schema from an underlying primitive type
func primitiveSchemaFromUnderlying(underlying types.Type) *spec.Schema {
    schema := &spec.Schema{}

    basic, ok := underlying.(*types.Basic)
    if !ok {
        schema.Type = []string{"object"}
        return schema
    }

    switch basic.Kind() {
    case types.Bool:
        schema.Type = []string{"boolean"}
    case types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
        types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64:
        schema.Type = []string{"integer"}
    case types.Float32, types.Float64:
        schema.Type = []string{"number"}
    case types.String:
        schema.Type = []string{"string"}
    default:
        schema.Type = []string{"object"}
    }

    return schema
}
```

**Step 4: Reorder type resolution to check underlying type for enums**

Modify: `internal/model/struct_field.go` at line ~303-340 (type resolution section)

Move the logic to check underlying type BEFORE checking for struct types:

```go
// Around line 303, find where named types are checked
// This is approximately where the enum detection currently happens

// For named types (custom types), check underlying type first
if sf.Type != nil {
    if named, ok := sf.Type.(*types.Named); ok {
        underlying := named.Underlying()

        // If underlying is struct → look it up as struct (existing behavior)
        if _, isStruct := underlying.(*types.Struct); isStruct {
            // Continue with struct handling (existing code)
            // Keep the existing struct handling logic here
        } else if isPrimitiveUnderlying(underlying) {
            // If underlying is primitive → check if it's an enum first
            if enumLookup != nil {
                typeName := sf.Type.String()
                if values, err := enumLookup.GetEnumsForType(typeName, nil); err == nil && len(values) > 0 {
                    // It's an enum! Create inline enum schema
                    schema := primitiveSchemaFromUnderlying(underlying)

                    // Add enum values
                    enumValues := make([]interface{}, len(values))
                    varNames := make([]string, len(values))
                    comments := make([]string, len(values))

                    for i, val := range values {
                        enumValues[i] = val.Value
                        varNames[i] = val.Key
                        if val.Comment != "" {
                            comments[i] = val.Comment
                        }
                    }

                    schema.Enum = enumValues
                    schema.VendorExtensible.AddExtension("x-enum-varnames", varNames)
                    if len(comments) > 0 {
                        schema.VendorExtensible.AddExtension("x-enum-comments", comments)
                    }

                    return schema, nestedTypes, nil
                }
            }

            // Not an enum, just a type alias - use underlying primitive type
            return primitiveSchemaFromUnderlying(underlying), nestedTypes, nil
        }
    }
}

// Continue with existing type resolution for other cases...
```

**Step 5: Run test to verify it passes**

Run: `go test ./internal/model -run "TestEnum.*Underlying" -v`

Expected: PASS - enums detected and inlined, type aliases resolved to primitives

**Step 6: Run all model tests to check for regressions**

Run: `go test ./internal/model -v`

Expected: All tests pass (no new failures)

---

## Phase 5: Fix Bug 5 - Public Reference Propagation

### Task 5.1: Write test for public reference propagation

**Files:**
- Modify: `internal/model/struct_field_test.go`

**Step 1: Write failing test using @superpowers:testing skill**

Use `Skill(testing)` to create a test that verifies public schemas reference Public variants of nested types.

Test requirements:
- Public schema with nested struct field
- Public schema with array of nested structs
- Non-public schema with nested struct (control case)
- Verify public schema refs end with "Public"
- Verify non-public schema refs don't have "Public" suffix

**Step 2: Run test to verify it fails**

Run: `go test ./internal/model -run "TestPublic.*Reference" -v`

Expected: FAIL - public schemas currently reference non-Public types

**Step 3: Fix reference generation to append Public suffix**

Modify: `internal/model/struct_field.go` at line ~296-376 (reference generation)

Find where `$ref` is created for nested structs and update:

```go
// Find the section where refs are created (approximately line 350-370)
// Look for: schema.Ref = spec.MustCreateRef(...)

// When creating $ref for nested struct types:
// OLD CODE might look like:
// refName := packagePath + "." + typeName
// schema.Ref = spec.MustCreateRef("#/definitions/" + refName)

// REPLACE WITH:
refName := packagePath + "." + typeName
if public {
    refName += "Public"
}
schema.Ref = spec.MustCreateRef("#/definitions/" + refName)
nestedTypes = append(nestedTypes, refName)

// Also need to handle arrays of structs - in array items section
// Find where array items refs are created and apply same logic
if strings.HasPrefix(typeStr, "[]") {
    // ... array handling code ...
    // When setting items.Ref:
    itemRefName := packagePath + "." + elemTypeName
    if public {
        itemRefName += "Public"
    }
    items.Ref = spec.MustCreateRef("#/definitions/" + itemRefName)
    nestedTypes = append(nestedTypes, itemRefName)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/model -run "TestPublic.*Reference" -v`

Expected: PASS - public schemas reference TypePublic, non-public reference Type

**Step 5: Run all model tests to check for regressions**

Run: `go test ./internal/model -v`

Expected: All tests pass (no new failures)

---

## Phase 6: Fix Pre-existing Test Failures

### Task 6.1: Investigate pre-existing test failures

**Files:**
- Check: `internal/model/struct_field_lookup_test.go`

**Step 1: Run the failing tests to see current error messages**

Run: `go test ./internal/model -run "TestBuildAllSchemas" -v`

Expected: Shows exactly what's failing in TestBuildAllSchemas_BillingPlan and TestBuildAllSchemas_Account

**Step 2: Analyze the failures**

Review the test output and determine:
- Are failures caused by bugs we just fixed?
- Are there additional issues not covered by our fixes?
- What specific assertions are failing?

Document findings in `.agents/change_log.md` under today's date.

**Step 3: If failures are now fixed, verify tests pass**

Run: `go test ./internal/model -run "TestBuildAllSchemas" -v`

Expected: If our bug fixes resolved the root causes, these tests should now pass

**Step 4: If failures persist, debug and fix**

For each remaining failure:
1. Identify the root cause
2. Write a focused test to isolate the issue
3. Implement the fix
4. Verify the fix works
5. Commit

Follow TDD pattern for any additional fixes needed.

**Step 5: Verify all model tests pass**

Run: `go test ./internal/model -v`

Expected: ALL tests pass, including TestBuildAllSchemas_BillingPlan and TestBuildAllSchemas_Account

**Step 6: Document resolution**

Update `.agents/change_log.md` with:
- What the failures were
- What fixed them (likely our 5 bug fixes)
- Confirmation that all tests now pass

---

## Phase 7: Integration Validation

### Task 7.1: Run make test-project-1 and verify output

**Files:**
- Check: `/Users/griffnb/projects/Crowdshield/atlas-go/swag_docs/swagger.json`

**Step 1: Run the project test**

Run: `make test-project-1`

Expected: Completes successfully, generates swagger.json

**Step 2: Verify Bug 1 fix (no BaseModel private fields)**

Run:
```bash
jq '.definitions["account.AccountJoinedFull"].properties | keys | .[]' \
  /Users/griffnb/projects/Crowdshield/atlas-go/swag_docs/swagger.json | \
  grep -E "changeLogs|client|manualCache|savingUser"
```

Expected: NO output (these fields should not exist)

**Step 3: Verify Bug 2 fix (division_ids array type)**

Run:
```bash
jq '.definitions["account.Properties"].properties.division_ids' \
  /Users/griffnb/projects/Crowdshield/atlas-go/swag_docs/swagger.json
```

Expected output:
```json
{
  "type": "array",
  "items": {
    "type": "string"
  }
}
```

**Step 4: Verify Bug 3 fix (external_user_info type)**

Run:
```bash
jq '.definitions["account.Properties"].properties.external_user_info' \
  /Users/griffnb/projects/Crowdshield/atlas-go/swag_docs/swagger.json
```

Expected output:
```json
{
  "type": "object"
}
```

**Step 5: Verify Bug 4 fix (NJDLClassification enum)**

Run:
```bash
jq '.definitions["constants.NJDLClassification"]' \
  /Users/griffnb/projects/Crowdshield/atlas-go/swag_docs/swagger.json
```

Expected output should have:
```json
{
  "type": "integer",
  "enum": [1, 2, 3, 4, 5],
  "x-enum-varnames": [...],
  "x-enum-comments": [...]
}
```

**Step 6: Verify Bug 5 fix (Public reference propagation)**

Run:
```bash
jq '.definitions["account.AccountJoinedSessionPublic"].properties.classifications' \
  /Users/griffnb/projects/Crowdshield/atlas-go/swag_docs/swagger.json
```

Expected output:
```json
{
  "type": "array",
  "items": {
    "$ref": "#/definitions/classification.JoinedClassificationPublic"
  }
}
```

**Step 7: Run full test suite**

Run: `go test ./... -v`

Expected: ALL tests pass across entire codebase

**Step 8: Document success**

Update `.agents/change_log.md` with:
```markdown
## 2026-02-16: Struct Parser Bug Fixes - COMPLETE ✅

**All 5 bugs fixed and verified:**
1. ✅ Embedded fields without tags excluded
2. ✅ Array element types correct (division_ids is string[])
3. ✅ Any type generates {"type": "object"}
4. ✅ Enum constants inline with values
5. ✅ Public schemas reference TypePublic variants

**Test Results:**
- ✅ All unit tests pass (internal/model)
- ✅ TestBuildAllSchemas_BillingPlan passes
- ✅ TestBuildAllSchemas_Account passes
- ✅ Full test suite passes (zero failures)
- ✅ make test-project-1 generates correct swagger.json

**Files Modified:**
- internal/model/struct_field.go (added helpers, fixed type resolution)
- internal/model/struct_field_lookup.go (added tag filtering)
- internal/model/struct_field_test.go (added tests for each bug)

**Integration Validation:**
- account.AccountJoinedFull: no private fields ✅
- account.Properties.division_ids: string[] ✅
- account.Properties.external_user_info: object ✅
- constants.NJDLClassification: enum values ✅
- Public schemas: reference TypePublic ✅
```

---

## Success Criteria Checklist

Before considering this complete, verify:

- [ ] All 5 new bug tests pass
- [ ] All existing struct_field_test.go tests pass
- [ ] TestBuildAllSchemas_BillingPlan passes
- [ ] TestBuildAllSchemas_Account passes
- [ ] All struct_builder tests pass (26 tests)
- [ ] Full test suite passes (go test ./...)
- [ ] make test-project-1 generates valid swagger.json
- [ ] account.AccountJoinedFull has no BaseModel private fields
- [ ] account.Properties.division_ids is string[] not object[]
- [ ] account.Properties.external_user_info is {"type": "object"}
- [ ] constants.NJDLClassification has enum values
- [ ] Public schemas reference TypePublic variants
- [ ] **Zero test failures across entire codebase**

---

## Troubleshooting

### If tests fail after a fix:

1. Read the test output carefully
2. Check if the fix introduced a regression
3. Add debug logging to understand what's happening
4. Run individual tests to isolate the issue
5. Revert the problematic commit if needed
6. Fix the issue properly and re-test

### If make test-project-1 output is incorrect:

1. Verify the fix is actually in the code
2. Rebuild: `go install ./cmd/core-swag`
3. Run make test-project-1 again
4. Check the actual swagger.json with jq commands
5. If still wrong, add debug logging to understand flow

### If integration tests fail:

1. Run `go test ./internal/model -v` to check unit tests
2. Unit tests passing but integration fails = different code path
3. Add logging to see which parser is being used
4. Ensure CoreStructParser and StructParserService both use struct_field.go

---

## References

- Design Document: `.agents/plans/2026-02-16-struct-parser-bug-fixes-design.md`
- Testing Skill: `@superpowers:testing`
- CLAUDE.md Architecture: `.claude/CLAUDE.md` (Struct Parsing Architecture section)
