# Fix Remaining 6 Missing $ref Definitions

## Context

After the first round of fixes (138 → 0 on integration test), 6 definitions are still missing in the full `make test-project-1` output (1797 refs, 1789 defs). Three distinct root causes.

| # | Missing Definition | Category |
|---|---|---|
| 1 | `account_event.AccountEventPublic` | PUBLIC |
| 2 | `alias.AliasPublic` | PUBLIC |
| 3 | `serp_result.SerpResultJoinedPublic` | PUBLIC |
| 4 | `tos_signature.TosSignaturePublic` | PUBLIC |
| 5 | `github_com_chargebee_chargebee-go_v3_enum.Source` | CANONICAL |
| 6 | `lawsuit.map%5Bstring%5Dany` | MAP |

---

## Bug A: MAP — `lawsuit.map[string]any` (1 item)

**Root cause**: `processStructField()` in `struct_field_lookup.go`. Field is `StructField[[]map[string]any]`:
1. `GenericTypeArg()` extracts `[]map[string]any`
2. Line 285: checks `strings.HasPrefix(subTypeName, "map[")` → false (starts with `[]`)
3. Lines 293-300: strips `[]` prefix → `subTypeName = "map[string]any"`
4. Lines 304-305: checks `IsPrimitive()` / `IsAny()` → false (it's a map)
5. Falls through to else-branch → package-qualifies as `lawsuit.map[string]any`

**Fix**: Add `strings.HasPrefix(subTypeName, "map[")` to the existing stripped probe check at line 305:
```go
if strippedProbe.IsPrimitive() || strippedProbe.IsAny() || strings.HasPrefix(subTypeName, "map[") {
```

**File**: `internal/model/struct_field_lookup.go` — `processStructField()`, line 305

---

## Bug B: PUBLIC — Missing Public variant definitions (4 items)

**Root cause**: `buildSchemasRecursive()` uses the unqualified `schemaName` (e.g., `"AccountEvent"`) as the `processed` map key (lines 737-740). When two different packages each have a type with the same short name, the second type's variants get skipped because the processed key already exists from the first type.

Example: If `pkg_a.Foo` and `pkg_b.Foo` are both nested types, the first processes `"Foo"` and `"FooPublic"`, marking both in the processed map. The second never builds its definitions because `processed["Foo"]` is already true.

**Fix**: Use fully qualified key in the `processed` map:
```go
processedKey := packageName + "." + schemaName
if processed[processedKey] {
    return nil
}
processed[processedKey] = true
```

**File**: `internal/model/struct_field_lookup.go` — `buildSchemasRecursive()`, lines 737-740

---

## Bug C: CANONICAL — `github_com_chargebee_chargebee-go_v3_enum.Source` (1 item)

**Root cause**: The canonical name registration code from the previous round only exists in Case 2 (enum). It's missing from Case 1 (nil builder → opaque object) and Case 3 (empty struct → opaque object). For external types that resolve to Case 1 or 3, the definition is stored under short key `"enum.Source"` but the $ref uses the NotUnique full-path key `"github_com_chargebee_chargebee-go_v3_enum.Source"`.

**Fix**: Add canonical name registration to Case 1 and Case 3:
```go
if globalNameResolver != nil && strings.Contains(nestedPkgPath, "/") {
    canonicalName := globalNameResolver.ResolveDefinitionName(nestedPkgPath + "." + cleanNestedType)
    if canonicalName != opaqueKey {
        allSchemas[canonicalName] = opaqueSchema
    }
}
```

**File**: `internal/model/struct_field_lookup.go` — `buildSchemasRecursive()`, Cases 1 and 3

---

## Implementation Order

| Step | Bug | Impact | File |
|------|-----|--------|------|
| 1 | A (MAP) | 1 item | `struct_field_lookup.go` — `processStructField` line 305 |
| 2 | B (PUBLIC) | 4 items | `struct_field_lookup.go` — `buildSchemasRecursive` lines 737-740 |
| 3 | C (CANONICAL) | 1 item | `struct_field_lookup.go` — Cases 1 and 3 |

---

## Verification

After all fixes:
1. `go build ./internal/model/` — compile
2. `go test ./internal/model/ -v` — unit tests
3. `make test-project-1` — full end-to-end
4. Run missing definition analysis on swagger output — target 0 missing
5. Update change log
