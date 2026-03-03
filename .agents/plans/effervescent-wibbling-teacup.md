# Fix 4 Categories of Missing $ref Definitions

## Context

After fixing 74 missing definitions (global_struct, address, constants, etc.), 137 still remain. We're targeting the 4 most impactful categories (18 items), which all stem from edge cases in the struct parsing pipeline.

**Current state**: Missing definitions threshold = 138. Target after fixes: ~120.

---

## Bug 1: WRONG TYPE (9 items) — `package.string` $refs

**Examples**: `account_classification.string`, `account.string`, `lead.string`

**Root cause**: `processStructField()` in `struct_field_lookup.go` line 291-355. For `StructField[[]string]`:
1. `GenericTypeArg()` extracts `[]string`
2. `IsPrimitive()` on `"[]string"` → false (only bare `"string"` is in the map)
3. Strips `[]` prefix → `subTypeName = "string"`
4. Falls to else-branch (line 345) → qualifies as `basePackage.Name + ".string"` → `"account_classification.string"`
5. This gets used as a struct ref, creating a dangling `$ref`

**Fix**: Insert primitive check after line 300 (after stripping `[]`/`*` prefixes):
```go
// After stripping array/pointer prefixes, check if bare type is primitive.
// Catches StructField[[]string] where "string" should not be package-qualified.
strippedProbe := &StructField{TypeString: subTypeName}
if strippedProbe.IsPrimitive() || strippedProbe.IsAny() {
    f.TypeString = arrayPrefix + subTypeName
    builder.Fields = append(builder.Fields, f)
    return
}
```

**File**: `internal/model/struct_field_lookup.go` — `processStructField()`, after line 300

---

## Bug 2: PROJECT (4 items) — Empty structs and external types

**Affected**: `atlas_phone.MetaData` (empty struct), `data_broker_domain.MetaData` (empty struct), `ironclad.Contract` (external), `plivo.PhoneNumber` (external)

**Root cause**: `buildSchemasRecursive()` line 828:
```go
isNonStruct := nestedBuilder == nil || len(nestedBuilder.Fields) == 0
```
Conflates two cases: nil builder (external/unloadable) vs empty struct (0 fields). Both skip definition creation.

**Fix**: Split into three cases in `buildSchemasRecursive()` lines 828-880:

1. **`nestedBuilder == nil`** (external type): Create opaque `{type: "object"}` definition
2. **`len(nestedBuilder.Fields) == 0` + is enum**: Existing enum handling (unchanged)
3. **`len(nestedBuilder.Fields) == 0` + not enum**: Create empty `{type: "object"}` definition

**File**: `internal/model/struct_field_lookup.go` — `buildSchemasRecursive()`, replace lines 828-880

---

## Bug 3: MALFORMED (3 items) — Struct tags leaking into type names

**Examples**: `data_broker_domain.FraudAlert%20json%3A%22alert_details%22...`

**Root cause**: Anonymous struct field processing in `ExtractFieldsRecursive` leaks struct tag content into type names. `spec.Ref.String()` URL-encodes the spaces/colons.

**Fix**: Defensive guard in `BuildSchema()` after the bracket validation (line 362), before `resolveRefName`:
```go
if strings.ContainsAny(typeName, " \t\"'`=:;") {
    console.Logger.Debug("Skipping ref for malformed type name: %s\n", typeName)
    return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"object"}}}, nil, nil
}
```

**File**: `internal/model/struct_field.go` — `BuildSchema()`, after line 362

---

## Bug 4: NOTUNIQUE (2 items) — $ref / definition key mismatch

**Root cause**: `resolveRefName` returns full-path name for NotUnique types (e.g., `github_com_chargebee_chargebee-go_v3_enum.Source`), but `buildSchemasRecursive` stores definitions under short key (`enum.Source`) at line 749.

**Fix**: After storing schema at line 777, also register under the canonical name if different:
```go
if globalNameResolver != nil && strings.Contains(pkgPath, "/") {
    lookupType := strings.TrimSuffix(schemaName, "Public")
    isPublicSchema := strings.HasSuffix(schemaName, "Public")
    canonicalName := globalNameResolver.ResolveDefinitionName(pkgPath + "." + lookupType)
    if isPublicSchema {
        canonicalName += "Public"
    }
    if canonicalName != fullSchemaName {
        allSchemas[canonicalName] = schema
    }
}
```

Same pattern for enum schemas (after line 864).

**File**: `internal/model/struct_field_lookup.go` — `buildSchemasRecursive()`, after line 777 and after line 864

---

## Implementation Order

| Step | Bug | Impact | File |
|------|-----|--------|------|
| 1 | WRONG TYPE | 9 items | `struct_field_lookup.go` — `processStructField` |
| 2 | MALFORMED | 3 items | `struct_field.go` — `BuildSchema` |
| 3 | PROJECT | 4 items | `struct_field_lookup.go` — `buildSchemasRecursive` |
| 4 | NOTUNIQUE | 2 items | `struct_field_lookup.go` — `buildSchemasRecursive` |

---

## Verification

After each fix:
1. Run `go test ./internal/model/ -v` (unit tests)
2. Run `go test ./testing/ -run TestRealProjectIntegration -v -count=1` (integration)
3. Verify missing count decreases, no regressions

After all fixes:
1. `make test-project-1` — full end-to-end
2. Update `knownMissingThreshold` in integration test to new count (~120)
3. Update change log
