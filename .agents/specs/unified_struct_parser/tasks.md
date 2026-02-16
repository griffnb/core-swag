# Unified Struct Parser - Implementation Tasks

## Overview

This task list consolidates three separate struct parsing implementations into ONE canonical parser at `internal/model/struct_builder.go`. Tasks are organized for autonomous execution with clear verification at each step.

**Total Phases:** 4
**Approach:** Enhance struct_builder → Integrate SchemaBuilder → Integrate Orchestrator → Delete old code

---

## Phase 1: Enhance Struct Builder (Core Features)

### Task 1.1: Set up comprehensive test file for struct_builder using existing test data

**Requirements:** Requirement 6 (Testing & Validation)

**Implementation:**
1. Review existing test data in `testing/testdata/core_models/` directory
2. Review existing integration test: `testing/testdata/core_models/parser_integration_test.go`
3. Create or enhance `internal/model/struct_builder_test.go`
4. Import and use test structs from `testing/testdata/core_models` (these are real objects, best set to test against)
5. Create helper functions for test assertions that can be reused across all struct_builder tests
6. Set up test cases using the biggest/most complex structs from core_models

**Verification:**
- Run: `go test ./internal/model/struct_builder_test.go`
- Expected: Test file compiles
- Run: `go test -v ./internal/model -run TestStructBuilder`
- Expected: Tests run (may fail if features not implemented yet, that's OK)
- Check: Test structs imported from `testing/testdata/core_models`
- Expected: Real, complex objects used for testing

**Self-Correction:**
- If imports fail: Check import path for core_models test data
- If test file doesn't compile: Fix import errors, check package declaration
- If helper functions unclear: Look at existing test patterns in parser_integration_test.go

**Completion Criteria:**
- [ ] Test file exists and compiles
- [ ] Uses real test data from testing/testdata/core_models
- [ ] Helper functions created
- [ ] Tests can be run (passing not required yet)
- [ ] Integration test structure understood

**Escape Condition:** If stuck after 3 attempts, document blockers and move to next task.

---

### Task 1.2: Add extended primitive mappings to struct_builder

**Requirements:** Requirement 2.1 (Extended Primitives)

**Implementation:**
1. Read `internal/schema/builder.go` lines 268-432 to find extended primitive mappings
2. Extract the mapping logic for:
   - time.Time → {type: "string", format: "date-time"}
   - uuid.UUID → {type: "string", format: "uuid"}
   - decimal.Decimal → {type: "number"}
   - json.RawMessage → {type: "object"}
3. Create a map in struct_builder.go or new file `internal/model/primitives.go`:
   ```go
   var ExtendedPrimitives = map[string]PrimitiveInfo{...}
   ```
4. Update type resolution logic in struct_builder to check ExtendedPrimitives map
5. Write unit tests in struct_builder_test.go:
   - TestExtendedPrimitives_TimeTime
   - TestExtendedPrimitives_UUID
   - TestExtendedPrimitives_Decimal
   - TestExtendedPrimitives_RawMessage
   - TestExtendedPrimitives_Pointers (pointers to extended primitives)

**Verification:**
- Run tests: `go test ./internal/model -run TestExtendedPrimitives`
- Expected: All extended primitive tests pass
- Run: `go test ./internal/model -v`
- Expected: No regressions in existing tests

**Self-Correction:**
- If tests fail: Check primitive type name matching (e.g., "time.Time" vs "Time")
- If missing types: Review schema builder fallback for additional extended primitives
- If pointer handling fails: Ensure pointer unwrapping happens before primitive check

**Completion Criteria:**
- [ ] Extended primitive map created
- [ ] All 4 core extended primitives supported
- [ ] Pointer variants work
- [ ] Tests passing
- [ ] No existing test regressions

---

### Task 1.3: Add enum detection and inlining

**Requirements:** Requirement 2.3 (Enum Detection)

**Implementation:**
1. Verify EnumLookup interface exists (check `internal/model/enum_lookup.go`)
2. Ensure struct_builder has access to EnumLookup instance
3. In type resolution logic, add enum check:
   ```go
   if isEnum, enumValues := enumLookup.GetEnumValues(typeName); isEnum {
       // Create inline enum schema with values
   }
   ```
4. Handle both string and integer enums
5. Create test structs in struct_builder_test.go:
   - Status enum (string-based)
   - Priority enum (int-based)
   - Struct with enum fields
   - Pointers to enums
   - Arrays of enums
6. Write unit tests:
   - TestEnumDetection_StringEnum
   - TestEnumDetection_IntEnum
   - TestEnumDetection_InlineValues
   - TestEnumDetection_PointerEnum
   - TestEnumDetection_ArrayOfEnums

**Verification:**
- Run tests: `go test ./internal/model -run TestEnumDetection`
- Expected: All enum tests pass
- Check test output: Verify enum values are inlined in schema (not $ref)
- Run: `make test-project-1` (if available)
- Expected: Enum fields appear with enum values in swagger.json

**Self-Correction:**
- If enum not detected: Check EnumLookup is properly initialized
- If values not inlined: Ensure schema includes "enum" property with values array
- If int enums fail: Verify enum value types are handled correctly

**Completion Criteria:**
- [ ] Enum detection working via EnumLookup
- [ ] String enums supported
- [ ] Integer enums supported
- [ ] Enum values inlined in schema
- [ ] Tests passing
- [ ] No regressions

---

### Task 1.4: Add StructField[T] generic extraction

**Requirements:** Requirement 2.2 (Generic Extraction)

**Implementation:**
1. Read `internal/model/struct_field_lookup.go` to find generic extraction logic
2. Add function to extract type parameter from `fields.StructField[T]`:
   ```go
   func extractGenericType(expr ast.Expr) (ast.Expr, bool)
   ```
3. Handle all generic variations from design.md test data:
   - Primitives: `StructField[string]`, `StructField[int]`
   - Pointers: `StructField[*string]`, `StructField[*User]`
   - Arrays: `StructField[[]string]`, `StructField[[]*User]`
   - Maps: `StructField[map[string]int]`, `StructField[map[string]*User]`
   - Cross-package: `StructField[*other_package.Account]`
   - Nested collections: `StructField[[][]int]`, `StructField[map[string][]User]`
4. Integrate into type resolution: check if field type is StructField[T], extract T, resolve T recursively
5. Create comprehensive test struct in struct_builder_test.go (use GenericFields from design.md)
6. Write unit tests:
   - TestGenericExtraction_Primitives
   - TestGenericExtraction_Pointers
   - TestGenericExtraction_Structs
   - TestGenericExtraction_Arrays
   - TestGenericExtraction_Maps
   - TestGenericExtraction_NestedCollections
   - TestGenericExtraction_CrossPackage

**Verification:**
- Run tests: `go test ./internal/model -run TestGenericExtraction`
- Expected: All generic extraction tests pass
- Verify: `StructField[string]` resolves to string schema, not StructField schema
- Verify: `StructField[*User]` resolves to User $ref, not StructField $ref
- Run: `make test-project-1` (if available)
- Expected: Fields using StructField[T] wrapper show correct underlying type

**Self-Correction:**
- If extraction fails: Check AST traversal for IndexExpr or IndexListExpr nodes
- If cross-package fails: Ensure package path resolution works
- If nested collections fail: Ensure recursive type resolution works

**Completion Criteria:**
- [ ] Generic type extraction function created
- [ ] All StructField[T] variations supported (primitives, pointers, arrays, maps, nested)
- [ ] Cross-package generic types work
- [ ] Tests passing
- [ ] No regressions

---

### Task 1.5: Verify and enhance embedded field merging

**Requirements:** Requirement 2.5 (Embedded Fields)

**Implementation:**
1. Read current embedded field handling in struct_builder
2. Test with embedded field test cases from design.md:
   - Single embedded field (BaseModel in User)
   - Multiple embedded fields (Timestamped + Versioned in Document)
   - Nested embedding (BaseEntity → AuditableEntity → Product)
3. Ensure properties from embedded structs are merged into parent schema
4. Ensure JSON tag overrides work
5. Handle name conflicts (later field wins)
6. Write unit tests:
   - TestEmbeddedFields_SingleLevel
   - TestEmbeddedFields_Multiple
   - TestEmbeddedFields_Nested
   - TestEmbeddedFields_NameConflicts
   - TestEmbeddedFields_JSONTagOverrides

**Verification:**
- Run tests: `go test ./internal/model -run TestEmbeddedFields`
- Expected: All embedded field tests pass
- Verify: Embedded struct properties appear as top-level properties in parent schema
- Verify: Embedded struct name doesn't appear as a property
- Run existing integration tests
- Expected: No regressions

**Self-Correction:**
- If properties not merged: Check recursive struct parsing for embedded fields
- If conflicts not handled: Implement "last wins" or document the strategy
- If nested embedding fails: Ensure recursion depth is sufficient

**Completion Criteria:**
- [ ] Single embedded fields work
- [ ] Multiple embedded fields work
- [ ] Nested embedded fields work (3+ levels)
- [ ] Name conflicts handled correctly
- [ ] Tests passing
- [ ] No regressions

---

### Task 1.6: Add caching layer to struct_builder

**Requirements:** Requirement 5 (Performance & Caching), Requirement 7 (Error Handling - circular references)

**Implementation:**
1. Read caching strategy from `internal/model/struct_field_lookup.go` (CoreStructParser)
2. Create cache structure:
   ```go
   type ParserCache struct {
       mu    sync.RWMutex
       cache map[string]*CachedResult
   }

   type CachedResult struct {
       Schema      *spec.Schema
       NestedTypes []string
   }
   ```
3. Add cache field to struct_builder
4. Before parsing a struct:
   - Check cache by qualified type name
   - Return cached result if exists
5. After parsing a struct:
   - Store result in cache
6. Use cache to detect circular references (if type is currently being parsed, create $ref)
7. Write unit tests:
   - TestCache_BasicCaching
   - TestCache_CacheHit
   - TestCache_CircularReference
   - TestCache_ConcurrentAccess (if applicable)

**Verification:**
- Run tests: `go test ./internal/model -run TestCache`
- Expected: All cache tests pass
- Verify: Parsing same type twice uses cache (faster second time)
- Verify: Circular references don't cause infinite loops
- Run: `go test ./internal/model -race` (check for race conditions)
- Expected: No race conditions detected

**Self-Correction:**
- If circular references hang: Ensure cache check happens before recursing
- If race conditions: Add proper mutex locking
- If cache not hit: Check qualified type name generation consistency

**Completion Criteria:**
- [ ] Cache structure implemented
- [ ] Cache stores parsed results
- [ ] Cache returns cached results on hit
- [ ] Circular references handled (no infinite loops)
- [ ] Tests passing
- [ ] No race conditions

---

### Task 1.7: Add validation constraint application

**Requirements:** Requirement 2.7 (Validation Constraints)

**Implementation:**
1. Read validation logic from `internal/parser/struct/field_processor.go`
2. Parse validation tags:
   - `validate:"required"` → add to required fields list
   - `validate:"min=X"` → set minimum on number fields
   - `validate:"max=X"` → set maximum on number fields
   - `validate:"minLength=X"` → set minLength on string fields
   - `validate:"maxLength=X"` → set maxLength on string fields
   - `pattern:"regex"` → set pattern on string fields
3. Add constraint application to field processing
4. Create test struct from design.md (ValidatedFields)
5. Write unit tests:
   - TestValidation_Required
   - TestValidation_MinMax (numbers)
   - TestValidation_MinMaxLength (strings)
   - TestValidation_Pattern
   - TestValidation_MultipleConstraints

**Verification:**
- Run tests: `go test ./internal/model -run TestValidation`
- Expected: All validation tests pass
- Check schema output: Verify min/max/minLength/maxLength/pattern properties exist
- Verify: Required fields are in schema's required array
- Run existing tests
- Expected: No regressions

**Self-Correction:**
- If tags not parsed: Check tag parser integration
- If constraints not applied: Verify schema properties are set correctly
- If required fields missing: Check required array population

**Completion Criteria:**
- [ ] All validation tags parsed
- [ ] Constraints applied to schema
- [ ] Required fields identified correctly
- [ ] Tests passing
- [ ] No regressions

---

### Task 1.8: Add public mode filtering

**Requirements:** Requirement 2.4 (Public Mode)

**Implementation:**
1. Read public mode logic from `internal/parser/struct/field_processor.go`
2. Add ParseOptions parameter to struct_builder methods:
   ```go
   type ParseOptions struct {
       Public        bool
       ForceRequired bool
       NamingStrategy string
   }
   ```
3. When Public=true, filter fields:
   - Include only fields with `public:"view"` or `public:"edit"` tags
   - Skip fields without public tag
4. Create test struct from design.md (PublicPrivateFields)
5. Write unit tests:
   - TestPublicMode_ViewOnly
   - TestPublicMode_EditOnly
   - TestPublicMode_Both
   - TestPublicMode_PrivateFiltered
   - TestPublicMode_Disabled (all fields included)

**Verification:**
- Run tests: `go test ./internal/model -run TestPublicMode`
- Expected: All public mode tests pass
- Verify: With Public=true, only fields with public tags included
- Verify: With Public=false, all fields included
- Run existing tests
- Expected: No regressions

**Self-Correction:**
- If filtering not working: Check tag parsing for "public" tag
- If wrong fields included: Verify filter logic (view vs edit vs both)
- If tests fail: Ensure ParseOptions is passed through call chain

**Completion Criteria:**
- [ ] ParseOptions added to API
- [ ] Public mode filtering works
- [ ] View and edit modes supported
- [ ] Tests passing
- [ ] No regressions

---

### Task 1.9: Add swaggerignore handling

**Requirements:** Requirement 2.8 (SwaggerIgnore)

**Implementation:**
1. Read swaggerignore logic from existing parsers
2. In field processing, check for `swaggerignore:"true"` tag
3. If present, skip field entirely (don't add to schema)
4. Also handle JSON tag "-" (standard Go ignore)
5. Create test struct from design.md (MixedVisibility)
6. Write unit tests:
   - TestSwaggerIgnore_TagPresent
   - TestSwaggerIgnore_JSONDash
   - TestSwaggerIgnore_MixedFields

**Verification:**
- Run tests: `go test ./internal/model -run TestSwaggerIgnore`
- Expected: All tests pass
- Verify: Fields with swaggerignore tag don't appear in schema
- Verify: Fields with json:"-" don't appear in schema
- Run existing tests
- Expected: No regressions

**Self-Correction:**
- If fields still appear: Check early return in field processing
- If wrong fields ignored: Verify tag parsing

**Completion Criteria:**
- [ ] swaggerignore:"true" handling added
- [ ] json:"-" handling works
- [ ] Tests passing
- [ ] No regressions

---

### Task 1.10: Add comprehensive array and map handling

**Requirements:** Requirement 2.6 (Arrays, Maps, Pointers)

**Implementation:**
1. Verify pointer unwrapping: `*Type` → `Type`
2. Verify array handling: `[]Type` → array schema with items
3. Verify map handling: `map[K]V` → object schema with additionalProperties
4. Add support for nested collections:
   - `[][]int` → array of arrays
   - `map[string][]User` → map with array values
   - `[]*User` → array with pointer elements
5. Create test structs from design.md:
   - AllPointerPrimitives
   - SimpleArrays, ArraysOfPointers, NestedArrays, ArraysOfStructs
   - SimpleMaps, ComplexMaps
6. Write unit tests:
   - TestPointers_Unwrapping
   - TestArrays_Primitives
   - TestArrays_Structs
   - TestArrays_Pointers
   - TestArrays_Nested
   - TestMaps_PrimitiveValues
   - TestMaps_StructValues
   - TestMaps_NestedValues

**Verification:**
- Run tests: `go test ./internal/model -run TestPointers -run TestArrays -run TestMaps`
- Expected: All tests pass
- Verify: `*string` generates same schema as `string` (nullable if needed)
- Verify: `[]User` generates array schema with $ref to User
- Verify: `map[string]User` generates object with additionalProperties $ref to User
- Run existing tests
- Expected: No regressions

**Self-Correction:**
- If pointers not unwrapped: Add pointer traversal in type resolution
- If nested arrays fail: Ensure recursive array handling
- If maps fail: Check additionalProperties is set correctly

**Completion Criteria:**
- [ ] Pointers unwrapped correctly
- [ ] Arrays of all types work (primitives, structs, pointers)
- [ ] Nested arrays work
- [ ] Maps with all value types work (primitives, structs, pointers, arrays)
- [ ] Nested maps work
- [ ] Tests passing
- [ ] No regressions

---

### Task 1.11: Add nested struct reference handling

**Requirements:** Requirement 2.9 (Nested Struct References)

**Implementation:**
1. When resolving a struct type, create `$ref` to that type
2. Add the referenced type to list of nested types that need definitions
3. Handle deep nesting (struct → struct → struct)
4. Create test structs from design.md:
   - Address, Contact, Person, Company (deep nesting)
   - PointerNested (pointers to nested structs)
5. Write unit tests:
   - TestNestedStructs_SingleLevel
   - TestNestedStructs_DeepNesting
   - TestNestedStructs_Pointers
   - TestNestedStructs_Arrays ([]User)
   - TestNestedStructs_Maps (map[string]User)

**Verification:**
- Run tests: `go test ./internal/model -run TestNestedStructs`
- Expected: All tests pass
- Verify: Nested struct fields create $ref (not inline schema)
- Verify: Nested types are returned in list for definition creation
- Run: `make test-project-1`
- Expected: Nested structs have definitions in swagger.json

**Self-Correction:**
- If $ref not created: Check struct type detection
- If nested types not collected: Ensure list is populated correctly
- If deep nesting fails: Check recursion works properly

**Completion Criteria:**
- [ ] Struct fields create $ref
- [ ] Nested types collected for definitions
- [ ] Deep nesting works (3+ levels)
- [ ] Pointers to structs work
- [ ] Arrays/maps of structs work
- [ ] Tests passing
- [ ] No regressions

---

### Task 1.12: Phase 1 integration testing

**Requirements:** Requirement 6 (Testing & Validation)

**Implementation:**
1. Review existing integration test: `testing/testdata/core_models/parser_integration_test.go`
2. Run struct_builder against ALL test structs in `testing/testdata/core_models/`
3. Verify all features work together using real test objects:
   - Embedded fields
   - Extended primitives
   - Enums
   - Generics (StructField[T])
   - Nested structs
   - Arrays and maps
   - Validation constraints
   - Public mode
   - SwaggerIgnore
4. Create integration test that uses the biggest/most complex structs from core_models
5. Run ALL struct_builder tests
6. Verify 75%+ test coverage:
   ```bash
   go test ./internal/model -coverprofile=coverage.out
   go tool cover -func=coverage.out
   ```

**Verification:**
- Run: `go test ./internal/model -v`
- Expected: All tests pass
- Run: `go test ./testing/testdata/core_models/parser_integration_test.go -v`
- Expected: Integration test passes with struct_builder
- Run: `go test ./internal/model -coverprofile=coverage.out && go tool cover -func=coverage.out | grep total`
- Expected: Coverage > 75%
- Run: `make test` (all project tests)
- Expected: No regressions in other packages

**Self-Correction:**
- If integration test fails: Identify which real struct is broken, fix it
- If coverage low: Add tests for untested branches
- If regressions: Revert breaking changes, fix properly

**Completion Criteria:**
- [ ] Integration test with core_models passes
- [ ] All struct_builder unit tests pass
- [ ] Test coverage > 75%
- [ ] No regressions in other packages
- [ ] Tested against real, complex objects
- [ ] Phase 1 complete

---

## Phase 2: Integrate with SchemaBuilder

### Task 2.1: Update SchemaBuilder to use enhanced struct_builder

**Requirements:** Requirement 4 (Incremental Migration)

**Implementation:**
1. Read `internal/schema/builder.go` to understand current struct parsing
2. Find where SchemaBuilder uses fallback parsing (lines ~268-432)
3. Replace fallback logic with call to enhanced struct_builder:
   ```go
   // OLD: Complex fallback parsing
   // NEW:
   schema, nestedTypes, err := b.structBuilder.ParseStruct(structType, file, options)
   ```
4. Ensure struct_builder instance is available in SchemaBuilder
5. Pass appropriate ParseOptions (public mode, forceRequired, etc.)
6. Handle nested types returned by struct_builder

**Verification:**
- Run: `go test ./internal/schema -v`
- Expected: All schema builder tests pass
- Run: `make test-project-1 && make test-project-2`
- Expected: Valid swagger.json generated (same or more fields than before)
- Compare output: Check that schemas are functionally equivalent or more complete

**Self-Correction:**
- If tests fail: Check struct_builder integration, ensure options passed correctly
- If output differs: Verify differences are MORE complete schemas (acceptable)
- If nested types missing: Ensure nested type definitions are created

**Completion Criteria:**
- [ ] SchemaBuilder uses enhanced struct_builder
- [ ] All schema builder tests pass
- [ ] Real project tests generate valid swagger
- [ ] No missing fields (more fields OK)
- [ ] Nested types handled correctly

---

### Task 2.2: Remove SchemaBuilder fallback code

**Requirements:** Requirement 4 (Incremental Migration)

**Implementation:**
1. Identify fallback methods in `internal/schema/builder.go`:
   - `buildFieldSchema` (lines ~268-432)
   - `getFieldType` (if separate)
   - Any other duplicate parsing logic
2. Delete these methods
3. Remove any calls to fallback methods
4. Clean up imports (remove unused)
5. Update any comments referencing fallback behavior

**Verification:**
- Run: `go test ./internal/schema -v`
- Expected: All tests still pass (no regressions)
- Run: `make test-project-1 && make test-project-2`
- Expected: Same output as Task 2.1
- Check: `git diff internal/schema/builder.go | grep "^-" | wc -l`
- Expected: ~329 lines deleted (fallback code)

**Self-Correction:**
- If tests fail: Revert deletion, identify dependencies, fix them first
- If compilation errors: Find remaining calls to deleted methods, update them

**Completion Criteria:**
- [ ] Fallback methods deleted (~329 lines)
- [ ] No calls to deleted methods remain
- [ ] All schema builder tests pass
- [ ] Real project tests pass
- [ ] Imports cleaned up

---

### Task 2.3: Add side-by-side comparison test

**Requirements:** Requirement 6.4 (Comparison Tests)

**Implementation:**
1. Create test in `internal/schema/builder_test.go`:
   ```go
   func TestSchemaBuilder_OutputComparison(t *testing.T)
   ```
2. Use test structs from Phase 1
3. Compare OLD output (if saved) vs NEW output
4. Verify NEW output has >= fields as OLD output
5. Log any differences for manual review

**Verification:**
- Run: `go test ./internal/schema -run TestSchemaBuilder_OutputComparison -v`
- Expected: Test passes
- Review: Manually check logged differences
- Expected: Differences are MORE complete schemas (extra fields that were missing before)

**Self-Correction:**
- If new output has FEWER fields: Bug in new parser, fix it
- If new output is invalid: Fix schema generation

**Completion Criteria:**
- [ ] Comparison test created
- [ ] Test passes
- [ ] Differences reviewed and acceptable
- [ ] Phase 2 complete

---

## Phase 3: Integrate with Orchestrator

### Task 3.1: Update Orchestrator to use enhanced struct_builder

**Requirements:** Requirement 4 (Incremental Migration)

**Implementation:**
1. Read `internal/orchestrator/service.go` to understand current struct parsing flow
2. Find where Orchestrator uses struct parser service (lines ~288-305)
3. Replace struct service calls with enhanced struct_builder calls:
   ```go
   // OLD: s.structParser.ParseFile(astFile, fileInfo.Path)
   // NEW:
   schemas, err := s.structBuilder.ParseFileStructs(astFile, fileInfo.Path, options)
   for name, schema := range schemas {
       s.schemaBuilder.AddDefinition(name, schema)
   }
   ```
4. May need to add `ParseFileStructs` method to struct_builder if doesn't exist
5. Ensure struct_builder instance is available in Orchestrator
6. Remove dependency on `internal/parser/struct/service.go`

**Verification:**
- Run: `go test ./internal/orchestrator -v`
- Expected: All orchestrator tests pass
- Run: `make test-project-1`
- Expected: Valid swagger.json generated, compare to baseline
- Run: `make test-project-2`
- Expected: Valid swagger.json generated, compare to baseline
- Check: No import of `internal/parser/struct` in orchestrator

**Self-Correction:**
- If ParseFileStructs doesn't exist: Implement it in struct_builder
- If tests fail: Check integration, ensure file-level parsing works
- If swagger invalid: Fix schema generation

**Completion Criteria:**
- [ ] Orchestrator uses enhanced struct_builder
- [ ] All orchestrator tests pass
- [ ] make test-project-1 passes (valid swagger)
- [ ] make test-project-2 passes (valid swagger)
- [ ] No dependency on old struct parser service

---

### Task 3.2: Run TestRealProjectIntegration

**Requirements:** Requirement 6.2 (TestRealProjectIntegration)

**Implementation:**
1. Run integration test: `go test ./testing -run TestRealProjectIntegration -v`
2. If test exists, verify it passes
3. If test doesn't exist, check for similar integration tests
4. Review output for any errors or warnings
5. Compare output to baseline (if available)

**Verification:**
- Run: `go test ./testing -run TestRealProjectIntegration -v`
- Expected: Test passes
- Run: `make test-project-1 && cat testing/project-1-example-swagger.json`
- Expected: Valid JSON, all expected schemas present
- Run: `make test-project-2 && cat testing/project-2-example-swagger.json`
- Expected: Valid JSON, all expected schemas present

**Self-Correction:**
- If test fails: Review error messages, fix integration issues
- If output missing fields: Check struct parsing for missed types
- If output has extra fields: Review if they're correct additions (likely OK)

**Completion Criteria:**
- [ ] TestRealProjectIntegration passes
- [ ] test-project-1 generates valid swagger
- [ ] test-project-2 generates valid swagger
- [ ] Output reviewed and acceptable

---

### Task 3.3: Add orchestrator comparison test

**Requirements:** Requirement 6.4 (Comparison Tests)

**Implementation:**
1. Create test in `internal/orchestrator/service_test.go`:
   ```go
   func TestOrchestrator_OutputComparison(t *testing.T)
   ```
2. Parse a test project with new parser
3. Compare to saved baseline (if available)
4. Verify no MISSING fields (more fields OK)

**Verification:**
- Run: `go test ./internal/orchestrator -run TestOrchestrator_OutputComparison -v`
- Expected: Test passes
- Review differences
- Expected: Differences are MORE complete output (acceptable)

**Self-Correction:**
- If missing fields: Fix struct parsing
- If invalid output: Fix integration

**Completion Criteria:**
- [ ] Comparison test created
- [ ] Test passes
- [ ] Differences reviewed and acceptable
- [ ] Phase 3 complete

---

## Phase 4: Cleanup and Delete Old Code

### Task 4.1: Delete struct parser service files

**Requirements:** Requirement 4 (Incremental Migration)

**Implementation:**
1. Verify no remaining imports of `internal/parser/struct` in codebase:
   ```bash
   grep -r "internal/parser/struct" --include="*.go" .
   ```
2. If no imports found, delete files:
   - `internal/parser/struct/service.go` (530 lines)
   - `internal/parser/struct/field_processor.go` (464 lines)
3. Keep `internal/parser/struct/tag_parser.go` if reused by struct_builder
4. Update any documentation referencing struct service

**Verification:**
- Run: `grep -r "internal/parser/struct" --include="*.go" .`
- Expected: No matches (except tag_parser if kept)
- Run: `go test ./... -v`
- Expected: All tests pass
- Run: `make test-project-1 && make test-project-2`
- Expected: Output unchanged from Task 3.2

**Self-Correction:**
- If imports found: Update those files to use struct_builder instead
- If tests fail: Identify missed dependencies, fix them

**Completion Criteria:**
- [ ] service.go deleted (530 lines)
- [ ] field_processor.go deleted (464 lines)
- [ ] No remaining imports
- [ ] All tests pass

---

### Task 4.2: Evaluate and archive CoreStructParser files

**Requirements:** Requirement 4 (Incremental Migration)

**Implementation:**
1. Check if CoreStructParser is still used:
   ```bash
   grep -r "CoreStructParser\|struct_field_lookup" --include="*.go" .
   ```
2. If not used, consider moving to `/archive` directory:
   - `internal/model/struct_field_lookup.go` (775 lines)
   - `internal/model/struct_field.go` (538 lines)
   - (Keep struct_builder.go - that's the enhanced ONE parser)
3. Alternatively, add deprecation notices if keeping temporarily
4. Update documentation

**Verification:**
- Run: `grep -r "CoreStructParser\|struct_field_lookup" --include="*.go" .`
- Expected: No matches (or only in archived files)
- Run: `go test ./... -v`
- Expected: All tests pass
- Check: Total lines of struct parsing code reduced by ~35%

**Self-Correction:**
- If still used: Identify callers, migrate them to struct_builder
- If tests fail: Keep files for now, mark deprecated

**Completion Criteria:**
- [ ] CoreStructParser usage checked
- [ ] Files deleted or archived if not used
- [ ] Deprecation notices added if kept
- [ ] All tests pass
- [ ] Code reduction achieved (~35%)

---

### Task 4.3: Update documentation

**Requirements:** Requirement 4 (Incremental Migration)

**Implementation:**
1. Update `.claude/CLAUDE.md` to reference the ONE parser:
   - Remove references to multiple parsers
   - Document location of struct parsing: `internal/model/struct_builder.go`
2. Update any README files in `internal/` directories
3. Add Go doc comments to struct_builder if missing
4. Document the ParseOptions configuration
5. Update architecture diagrams (if any)

**Verification:**
- Read: `.claude/CLAUDE.md`
- Expected: Clear documentation of ONE parser location
- Read: `internal/model/struct_builder.go`
- Expected: Package and function Go doc comments present
- Run: `go doc internal/model StructBuilder`
- Expected: Documentation displayed

**Self-Correction:**
- If documentation unclear: Rewrite for clarity
- If examples missing: Add usage examples

**Completion Criteria:**
- [ ] CLAUDE.md updated
- [ ] README files updated
- [ ] Go doc comments added
- [ ] ParseOptions documented
- [ ] Architecture diagrams updated (if applicable)

---

### Task 4.4: Clean up imports and run final tests

**Requirements:** Requirement 4 (Incremental Migration)

**Implementation:**
1. Run `go mod tidy` to clean up dependencies
2. Run goimports to organize imports:
   ```bash
   goimports -w ./internal
   ```
3. Run linter:
   ```bash
   golangci-lint run ./...
   ```
4. Run all tests:
   ```bash
   go test ./... -v
   ```
5. Run real project tests:
   ```bash
   make test-project-1
   make test-project-2
   ```
6. Check test coverage:
   ```bash
   go test ./internal/model -coverprofile=coverage.out
   go tool cover -func=coverage.out | grep total
   ```

**Verification:**
- Run: `go mod tidy && git diff go.mod go.sum`
- Expected: No unnecessary dependencies removed
- Run: `golangci-lint run ./...`
- Expected: No errors (warnings OK)
- Run: `go test ./... -v`
- Expected: All tests pass
- Run: `make test-project-1 && make test-project-2`
- Expected: Valid swagger.json files generated
- Check coverage: `> 75%`
- Expected: Coverage target met

**Self-Correction:**
- If linter errors: Fix them
- If tests fail: Fix failures
- If coverage low: Add missing tests

**Completion Criteria:**
- [ ] Dependencies cleaned up
- [ ] Imports organized
- [ ] Linter passing
- [ ] All tests passing
- [ ] Real project tests passing
- [ ] Coverage > 75%
- [ ] Phase 4 complete

---

## Success Criteria Summary

The consolidation is complete when:

1. ✅ **ONE parser exists:** All struct parsing happens in `internal/model/struct_builder.go`
2. ✅ **All features supported:** Extended primitives, enums, generics, embedded fields, validation, public mode, swaggerignore, arrays, maps, nested structs
3. ✅ **Code reduction achieved:** ~35% reduction (~3,100 → ~2,000 lines or less)
4. ✅ **All tests pass:** Unit tests, integration tests, real project tests
5. ✅ **Old implementations deleted:** Struct service and schema builder fallback removed
6. ✅ **Documentation updated:** CLAUDE.md and code docs reference ONE parser
7. ✅ **Performance maintained:** Within 2x of current performance
8. ✅ **No regressions:** All existing functionality preserved or improved
9. ✅ **Valid output:** Real projects generate valid swagger.json (same or more fields)
10. ✅ **Test coverage:** > 75% coverage on struct_builder

---

## Escape Conditions

For any task, if stuck after 3 iterations:
1. Document the blocker clearly
2. Note what was tried
3. Move to next task
4. Return to blocked task after completing others (fresh perspective)

For any phase, if critical blocker prevents progress:
1. Document the issue
2. Review design for alternatives
3. Seek clarification if needed
4. Consider rolling back to previous phase

---

## Notes for Ralph Wiggum Execution

- Each task is designed to be self-contained
- Verification steps can be automated
- Self-correction guidance helps handle common issues
- Escape conditions prevent infinite loops
- Tasks build incrementally (each depends on previous)
- Tests provide immediate feedback on correctness
- Real project tests validate end-to-end functionality
