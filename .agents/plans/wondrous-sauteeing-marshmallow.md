# Fix: Enum Types Incorrectly Treated as Structs with Public Variants

## Context

Enum/constant types (e.g., `constants.Status`) are appearing in swagger output as:
```json
"constants.StatusPublic": { "type": "object", "title": "ConstantsStatusPublic" }
"constants.Status": { "type": "object", "title": "ConstantsStatus" }
```

They should appear as proper enum definitions like:
```json
"constants.Status": { "type": "integer", "enum": [100, 200, 300], "x-enum-varnames": ["..."] }
```

**Core principle**: Only struct types get "Public" variants. Enums/constants NEVER get a "Public" suffix.

**Root causes**:
1. `buildSchemasRecursive` in `struct_field_lookup.go` — `LookupStructFields()` returns non-nil empty `StructBuilder` for non-struct types. Code checks `if nestedBuilder == nil` so enum detection is never reached.
2. Route parser (`response.go`) and allof builder (`allof.go`) blindly append "Public" to ALL non-primitive types without checking if the type is actually a struct.

## Phase 1: Write Failing Tests (TDD)

### Test 1: Integration test — No Public enum definitions should exist
**File**: `testing/core_models_integration_test.go`

Add to `TestCoreModelsIntegration`:
```go
t.Run("Enum definitions should NEVER have Public variants", func(t *testing.T) {
    // Enums are identical regardless of public context
    for name := range swagger.Definitions {
        if strings.HasSuffix(name, "Public") {
            def := swagger.Definitions[name]
            // If it has enum values, it should NOT have a Public variant name
            assert.Empty(t, def.Enum,
                "Definition %s has Public suffix but contains enum values - enums should never have Public variants", name)
        }
    }
    // Specifically check known enum types don't have Public variants
    assert.NotContains(t, swagger.Definitions, "constants.StatusPublic")
    assert.NotContains(t, swagger.Definitions, "constants.RolePublic")
    assert.NotContains(t, swagger.Definitions, "constants.OrganizationTypePublic")
    assert.NotContains(t, swagger.Definitions, "constants.GlobalConfigKeyPublic")
    assert.NotContains(t, swagger.Definitions, "constants.NJDLClassificationPublic")
})
```

### Test 2: Integration test — Enum definitions should be proper enums not objects
**File**: `testing/core_models_integration_test.go`

Add to `TestCoreModelsIntegration`:
```go
t.Run("Enum definitions should have proper enum type not object", func(t *testing.T) {
    // constants.Status should be integer type with enum values, NOT type:"object"
    statusDef := swagger.Definitions["constants.Status"]
    assert.NotContains(t, statusDef.Type, "object",
        "constants.Status should NOT be type:object")
    assert.Contains(t, statusDef.Type, "integer",
        "constants.Status should be type:integer")
    assert.NotEmpty(t, statusDef.Enum,
        "constants.Status should have enum values")
})
```

### Test 3: Integration test — Public struct variants should reference base enum names
**File**: `testing/core_models_integration_test.go`

Add to `TestCoreModelsIntegration`:
```go
t.Run("Public struct variants should reference base enum names without Public suffix", func(t *testing.T) {
    // AccountPublic has role field (public:"view") which is IntConstantField[constants.Role]
    // The $ref should be constants.Role, NOT constants.RolePublic
    accountPublic := swagger.Definitions["account.AccountPublic"]
    require.NotNil(t, accountPublic)

    roleSchema, hasRole := accountPublic.Properties["role"]
    require.True(t, hasRole, "AccountPublic should have role field")
    assert.Equal(t, "#/definitions/constants.Role", roleSchema.Ref.String(),
        "Public struct should reference base enum name, not constants.RolePublic")

    // Same for status field from base.Structure
    statusSchema, hasStatus := accountPublic.Properties["status"]
    require.True(t, hasStatus, "AccountPublic should have status field")
    assert.Equal(t, "#/definitions/constants.Status", statusSchema.Ref.String(),
        "Public struct should reference base enum name, not constants.StatusPublic")

    // Same for config_key
    configSchema, hasConfig := accountPublic.Properties["config_key"]
    require.True(t, hasConfig, "AccountPublic should have config_key field")
    assert.Equal(t, "#/definitions/constants.GlobalConfigKey", configSchema.Ref.String(),
        "Public struct should reference base enum name, not constants.GlobalConfigKeyPublic")
})
```

### Test 4: Unit test — buildSchemaForType should not append Public to enum refs
**File**: `internal/model/struct_builder_test.go`

```go
func TestBuildSpecSchema_EnumFieldInPublicMode_NoPublicSuffix(t *testing.T) {
    enumLookup := newTestEnumLookup()
    enumLookup.addEnum("constants.Status", []EnumValue{
        {Key: "STATUS_ACTIVE", Value: 100},
        {Key: "STATUS_DISABLED", Value: 200},
        {Key: "STATUS_DELETED", Value: 300},
    })
    enumLookup.addEnum("constants.Role", []EnumValue{
        {Key: "ROLE_USER", Value: 1},
        {Key: "ROLE_ADMIN", Value: 100},
    })

    builder := &StructBuilder{
        Fields: []*StructField{
            {
                Name:       "Status",
                TypeString: "constants.Status",
                Tag:        `public:"view" json:"status"`,
            },
            {
                Name:       "Role",
                TypeString: "constants.Role",
                Tag:        `public:"view" json:"role"`,
            },
            {
                Name:       "Profile",
                TypeString: "account.Profile",
                Tag:        `public:"view" json:"profile"`,
            },
        },
    }

    // Build Public variant
    schema, nestedTypes, err := builder.BuildSpecSchema("Account", true, false, enumLookup)
    require.NoError(t, err)

    // Enum fields should reference base name (no Public suffix)
    statusProp := schema.Properties["status"]
    assert.Equal(t, "#/definitions/constants.Status", statusProp.Ref.String(),
        "Enum ref should NOT have Public suffix in public mode")

    roleProp := schema.Properties["role"]
    assert.Equal(t, "#/definitions/constants.Role", roleProp.Ref.String(),
        "Enum ref should NOT have Public suffix in public mode")

    // Struct fields SHOULD have Public suffix
    profileProp := schema.Properties["profile"]
    assert.Equal(t, "#/definitions/account.ProfilePublic", profileProp.Ref.String(),
        "Struct ref SHOULD have Public suffix in public mode")

    // Enum types should be in nestedTypes WITHOUT Public suffix
    assert.Contains(t, nestedTypes, "constants.Status")
    assert.Contains(t, nestedTypes, "constants.Role")
    assert.NotContains(t, nestedTypes, "constants.StatusPublic")
    assert.NotContains(t, nestedTypes, "constants.RolePublic")
    // Struct types SHOULD have Public suffix
    assert.Contains(t, nestedTypes, "account.ProfilePublic")
}
```

### Test 5: Unit test — route parser shouldn't append Public to non-struct types
**File**: `internal/parser/route/response_test.go`

Add test using mock registry to verify `buildSchemaForTypeWithPublic` doesn't append Public to enum types.

## Phase 2: Implementation

### Change 1: Fix `buildSchemasRecursive` enum detection (CORE FIX)
**File**: `internal/model/struct_field_lookup.go` (lines 804-896)

Change nil check to also check empty builder fields:
```go
isEmptyBuilder := nestedBuilder == nil || len(nestedBuilder.Fields) == 0
if isEmptyBuilder {
    // Strip "Public" suffix if present — enums never have Public variants
    cleanNestedType := baseNestedType
    if strings.HasSuffix(cleanNestedType, "Public") {
        cleanNestedType = strings.TrimSuffix(cleanNestedType, "Public")
    }

    // Try enum lookup
    enumLookup := &ParserEnumLookup{...}
    enums, err := enumLookup.GetEnumsForType(nestedPackageName+"."+cleanNestedType, nil)
    if err == nil && len(enums) > 0 {
        // Create ONLY base enum schema — NO Public variant
        baseEnumSchema := buildProperEnumSchema(enums, nestedPackageName, cleanNestedType)
        allSchemas[nestedPackageName+"."+cleanNestedType] = baseEnumSchema
        processed[cleanNestedType] = true
        processed[cleanNestedType+"Public"] = true  // block Public variant
        continue
    }
    if nestedBuilder == nil { continue }
}
```

### Change 2: Fix `buildSchemaForTypeWithPublic` — struct check before Public
**File**: `internal/parser/route/response.go` (lines 175-178)

Add `isStructType()` helper using existing `s.registry`, only append "Public" for structs:
```go
if isPublic && !s.hasNoPublicAnnotation(qualifiedType) && s.isStructType(qualifiedType) {
    qualifiedType = qualifiedType + "Public"
}
```

### Change 3: Fix allof.go — same struct check for override fields
**File**: `internal/parser/route/allof.go` (lines 69-71)

```go
if isPublic && !isPrimitiveType(fieldType) && s.isStructType(qualifiedFieldType) {
    qualifiedFieldType = qualifiedFieldType + "Public"
}
```

### Change 4: Orchestrator — handle non-struct Public refs
**File**: `internal/orchestrator/schema_builder.go`

When `buildSchemaForRef` encounters a non-struct type with "Public" suffix, build the base schema and skip the Public variant.

## Files to Modify
1. `testing/core_models_integration_test.go` — Add failing tests (Phase 1)
2. `internal/model/struct_builder_test.go` — Add unit test for enum in public mode (Phase 1)
3. `internal/model/struct_field_lookup.go` — Fix `buildSchemasRecursive` enum detection (Phase 2)
4. `internal/parser/route/response.go` — Add `isStructType`, guard Public suffix (Phase 2)
5. `internal/parser/route/allof.go` — Same guard for allof overrides (Phase 2)
6. `internal/orchestrator/schema_builder.go` — Handle non-struct Public refs (Phase 2)

## Verification
1. Run new tests — should fail before implementation, pass after
2. `go test ./testing/... -run TestCoreModelsIntegration` — integration tests
3. `go test ./internal/model/... -run TestBuildSpecSchema_EnumField` — unit tests
4. `make test-project-1` / `make test-project-2` — real project output check
5. Grep output JSON: no `StatusPublic` or `RolePublic` definitions for enums
