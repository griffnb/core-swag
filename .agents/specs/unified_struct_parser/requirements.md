# Unified Struct Parser - Requirements

## Introduction

The core-swag codebase currently has three separate struct parsing implementations totaling ~3,100 lines of code with significant duplication:
1. CoreStructParser (`internal/model/`) - 1,379 lines
2. Struct Parser Service (`internal/parser/struct/`) - 1,413 lines
3. Schema Builder Fallback (`internal/schema/builder.go`) - 329 lines

This duplication leads to:
- Inconsistent feature support across parsers
- Bug fixes requiring updates in multiple locations
- Order-dependent behavior and potential race conditions
- Confusion about which parser to use when
- High maintenance burden

The goal is to consolidate these into **ONE canonical struct parser** that:
- Handles ALL struct types (simple and complex)
- Works from AST + registry (no heavy dependencies)
- Handles embedded fields properly
- Can degrade gracefully to handle regular struct types
- Is the single entry point for all struct parsing
- Reduces code by eliminating duplication (~40% reduction target)

---

## Requirements

### 1. Single Canonical Parser

**User Story:** As a core-swag developer, I want ONE struct parser that handles all cases, so that there's no confusion about which parser to use and no code duplication.

**Acceptance Criteria:**

1. **WHEN** any component needs to parse a struct, **THEN** there SHALL be exactly ONE parser to call
2. **WHEN** the parser is invoked, **THEN** it SHALL handle both simple and complex struct types without requiring different code paths
3. **WHEN** parsing is complete, **THEN** all three existing implementations SHALL be removed/replaced
4. **WHEN** the consolidation is done, **THEN** total struct parsing code SHALL be reduced by at least 35%
5. **WHEN** looking at the codebase, **THEN** it SHALL be obvious where all struct parsing logic lives

### 2. Complete Feature Support

**User Story:** As a core-swag user, I want the single parser to support all features from all three existing parsers, so that no functionality is lost during consolidation.

**Acceptance Criteria:**

1. **WHEN** parsing primitive types, **THEN** the parser SHALL support all extended primitives:
   - time.Time → string with format: date-time
   - uuid.UUID → string with format: uuid
   - decimal.Decimal → number
   - json.RawMessage → object
   - All standard Go primitives (string, int, bool, float, etc.)
2. **WHEN** encountering `fields.StructField[T]`, **THEN** the parser SHALL:
   - Extract the type parameter T from the generic wrapper
   - Use T as the actual field type in the schema
   - Properly handle nested generic types
3. **WHEN** a field type is an enum, **THEN** the parser SHALL:
   - Detect enum types automatically via enum lookup
   - Inline enum values in the schema
   - Support both string and integer enums
4. **WHEN** public mode is enabled, **THEN** the parser SHALL:
   - Filter fields based on `public:"view|edit"` tags
   - Create separate public schemas
   - Keep private fields in non-public schemas
5. **WHEN** encountering embedded fields, **THEN** the parser SHALL:
   - Merge embedded field properties into the parent schema
   - Respect JSON tag overrides from the embedding struct
   - Handle multiple levels of embedding recursively
   - Properly handle name conflicts in embedded fields
6. **WHEN** parsing arrays, maps, or pointers, **THEN** the parser SHALL:
   - Correctly unwrap pointer types (*Type → Type)
   - Generate proper array schemas with item type references
   - Generate proper map schemas with additionalProperties
   - Handle nested collections ([][]Type, map[string][]Type, etc.)
7. **WHEN** validation tags are present, **THEN** the parser SHALL apply constraints:
   - min/max for numbers
   - minLength/maxLength for strings
   - pattern for regex validation
   - required for required fields
8. **WHEN** `swaggerignore:"true"` tag is present, **THEN** the parser SHALL skip that field entirely
9. **WHEN** nested struct types are encountered, **THEN** the parser SHALL:
   - Create proper $ref references to the nested type
   - Register the nested type for schema generation
   - Handle circular references without infinite loops

### 3. Graceful Degradation

**User Story:** As a core-swag developer, I want the parser to handle both complex and simple cases, so that it works seamlessly regardless of struct complexity.

**Acceptance Criteria:**

1. **WHEN** parsing a simple struct with only primitives, **THEN** the parser SHALL handle it without unnecessary complexity
2. **WHEN** parsing a struct with embedded fields, **THEN** the parser SHALL properly merge them into the parent
3. **WHEN** parsing a struct with nested struct references, **THEN** the parser SHALL create proper $ref links
4. **WHEN** a type cannot be fully resolved, **THEN** the parser SHALL fall back to a safe default (object type) and log a warning
5. **WHEN** registry lookup fails for a type, **THEN** the parser SHALL attempt basic AST-based inference before failing

### 4. Incremental Migration

**User Story:** As a core-swag developer, I want incremental migration without breaking existing functionality, so that consolidation happens safely.

**Acceptance Criteria:**

1. **WHEN** building the new parser, **THEN** all existing tests SHALL continue to pass (or produce MORE complete/correct output)
2. **WHEN** integrating with SchemaBuilder, **THEN** the output SHALL be functionally equivalent or MORE complete than before
3. **WHEN** integrating with Orchestrator, **THEN** the Parse() flow SHALL produce valid swagger.json with same or MORE fields
4. **WHEN** output differs from old parsers, **THEN** the difference SHALL be MORE complete schemas (missing fields now included)
5. **WHEN** replacing old parsers, **THEN** all callers SHALL be updated to use the ONE new parser
6. **WHEN** migration is complete, **THEN** the three old implementations SHALL be deleted

### 5. Performance & Caching

**User Story:** As a core-swag user, I want efficient parsing with caching, so that large projects don't slow down.

**Acceptance Criteria:**

1. **WHEN** a struct type is parsed, **THEN** the result SHALL be cached by fully qualified type name
2. **WHEN** the same type is encountered again, **THEN** the cached result SHALL be returned immediately
3. **WHEN** parsing completes, **THEN** performance SHALL be comparable to current fastest parser (within 2x)
4. **WHEN** parsing repeated types, **THEN** cache hit rate SHALL be > 70%
5. **WHEN** cache is used, **THEN** circular reference detection SHALL prevent infinite loops

### 6. Testing & Validation

**User Story:** As a core-swag developer, I want comprehensive testing for the single parser, so that consolidation doesn't introduce regressions.

**Acceptance Criteria:**

1. **WHEN** the parser is implemented, **THEN** unit test coverage SHALL be > 75%
2. **WHEN** integration tests run, **THEN** `TestRealProjectIntegration` SHALL pass (output may be MORE complete)
3. **WHEN** real project tests run, **THEN**:
   - `make test-project-1` SHALL generate valid swagger.json (same or more fields)
   - `make test-project-2` SHALL generate valid swagger.json (same or more fields)
4. **WHEN** replacing old parsers, **THEN** side-by-side comparison SHALL verify no MISSING fields (more fields is OK)
5. **WHEN** each feature is added, **THEN** it SHALL have dedicated unit tests
6. **WHEN** output differs from old parsers, **THEN** manual review SHALL confirm extra fields are correct additions

### 7. Error Handling

**User Story:** As a core-swag developer, I want clear error messages, so that parsing issues are easy to debug.

**Acceptance Criteria:**

1. **WHEN** type resolution fails, **THEN** errors SHALL include:
   - Full type name and package path
   - Source file and line number if available
   - Reason for failure
2. **WHEN** tag parsing fails, **THEN** errors SHALL include field name and the problematic tag value
3. **WHEN** circular type references occur, **THEN** the parser SHALL detect them and prevent infinite loops
4. **WHEN** parsing fails, **THEN** error messages SHALL be actionable and clear
5. **WHEN** parsing succeeds with warnings, **THEN** warnings SHALL be logged but not halt execution

### 8. Code Organization

**User Story:** As a core-swag developer, I want well-organized code, so that the single parser is easy to maintain and understand.

**Acceptance Criteria:**

1. **WHEN** the parser is created/enhanced, **THEN** it SHALL have a clear home (likely `internal/model/struct_builder.go` or similar)
2. **WHEN** code is written, **THEN** related functionality SHALL be grouped into logical files:
   - Main parsing logic in one place
   - Helper utilities in separate files if needed (primitives, generics, etc.)
   - No single file exceeding 800 lines
3. **WHEN** the parser is complete, **THEN** all public functions SHALL have Go doc comments
4. **WHEN** consolidation is done, **THEN** CLAUDE.md SHALL be updated to point to the ONE parser
5. **WHEN** looking at the code, **THEN** it SHALL be obvious where ALL struct parsing happens

---

## Feature Matrix (Current vs Target)

| Feature | CoreStructParser | Struct Service | Schema Fallback | ONE Parser Target |
|---------|------------------|----------------|-----------------|-------------------|
| Extended primitives | ✅ Full | ✅ Full | ✅ Full | ✅ MUST HAVE |
| Enum detection | ✅ Yes | ❌ No | ✅ Yes | ✅ MUST HAVE |
| fields.StructField[T] | ✅ Yes | ⚠️ Partial | ❌ No | ✅ MUST HAVE |
| Public mode | ✅ Yes | ✅ Yes | ❌ No | ✅ MUST HAVE |
| Embedded fields | ✅ Yes | ✅ Yes | ⚠️ Partial | ✅ MUST HAVE |
| Nested struct $refs | ✅ Yes | ✅ Yes | ✅ Yes | ✅ MUST HAVE |
| Caching | ✅ Yes | ❌ No | ❌ No | ✅ MUST HAVE |
| Validation constraints | ✅ Yes | ✅ Yes | ✅ Yes | ✅ MUST HAVE |
| Works from AST + registry | ❌ No | ✅ Yes | ✅ Yes | ✅ MUST HAVE |

---

## Success Criteria Summary

The struct parser consolidation is considered complete when:

1. ✅ Exactly ONE parser exists that handles all struct parsing
2. ✅ All 8 requirement sections have 100% of acceptance criteria met
3. ✅ Code reduction of at least 35% achieved (~3,100 → ~2,000 lines or less)
4. ✅ All existing tests pass (with same or MORE complete output)
5. ✅ Both real project tests generate valid swagger (`test-project-1`, `test-project-2`)
6. ✅ Performance is maintained (within 2x of current fastest parser)
7. ✅ All three old implementations removed from codebase
8. ✅ Documentation updated to point to the ONE parser
9. ✅ No MISSING fields (more fields than before is acceptable and likely correct)
10. ✅ All callers updated to use the ONE parser

---

## Out of Scope

The following are explicitly **NOT** part of this consolidation effort:

- ❌ Changing the OpenAPI schema output format or structure
- ❌ Adding new parsing features beyond what the three existing parsers already support
- ❌ Modifying how the orchestrator coordinates parsing (only updating which parser it calls)
- ❌ Changing the registry, schema builder, or other component public APIs
- ❌ Refactoring unrelated code outside of struct parsing
- ❌ Major performance rewrites (just maintain current performance)
- ❌ Changes to struct tag syntax or naming conventions
- ❌ Creating complex abstraction layers or interfaces (KEEP IT SIMPLE)
- ❌ Supporting go/packages if AST + registry is sufficient
