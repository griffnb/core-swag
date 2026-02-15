# Core-Swag Systematic Restoration Plan
**Created:** 2026-02-14
**Status:** Project in broken state - needs systematic rebuild
**Goal:** Restore full swagger generation capability matching legacy project output

---

## 🎯 Executive Summary

**Current State (Updated 2026-02-15):**
- ✅ 6/6 services integrated and working (Loader, Registry, Schema, StructParser, RouteParser, Orchestrator)
- ✅ Critical test PASSING: TestCoreModelsIntegration (87 definitions, 5 paths, all 6 subtests passing)
- ✅ Custom models fully supported (fields.StringField, fields.IntField, fields.StructField[T])
- ✅ Public/private filtering working (22 public vs 28 base properties)
- ✅ @Public annotation working correctly
- ✅ AllOf composition working for response envelopes
- ✅ Embedded struct support with recursive field merging
- ⚠️ Still relies on some legacy code in internal/legacy_files/
- ⚠️ Need to test make test-project-1 and make test-project-2

**Target Output:**
- Generate swagger.json/yaml files with full OpenAPI 2.0 spec
- Support custom models (fields.StructField[T])
- Support public/private field filtering
- Support @Public annotation
- Support AllOf composition for response envelopes
- Handle generic types correctly

**Success Criteria:**
- ✅ TestCoreModelsIntegration passes (41 files → 87 definitions → 5 paths) ✅ DONE
- ⏳ make test-project-1 generates valid swagger matching project-1-example-swagger.json
- ⏳ make test-project-2 generates valid swagger matching project-2-example-swagger.json
- ⏳ Zero legacy code dependencies (remove internal/legacy_files/)
- ✅ All files < 500 lines per project standards

---

## 📊 Testing Strategy: Bottom-Up Pyramid

```
                    ┌─────────────────────┐
                    │  E2E Integration    │  ← make test-project-1/2
                    │  (Real Projects)    │
                    └─────────────────────┘
                            ▲
                    ┌───────┴───────┐
                    │  Integration  │  ← TestCoreModelsIntegration
                    │  Test Suite   │     TestRealProjectIntegration
                    └───────────────┘
                            ▲
                    ┌───────┴────────┐
                    │  Service Unit  │  ← Individual service tests
                    │  Tests         │     (loader, schema, parser)
                    └────────────────┘
                            ▲
                    ┌───────┴────────┐
                    │  Component     │  ← Converters, helpers
                    │  Tests         │     Type resolution, etc.
                    └────────────────┘
```

**Testing Philosophy:**
- **Build from bottom up** - Fix components → services → integration → E2E
- **Test at each layer** - Don't move up until current layer passes
- **No running E2E until integration tests pass** - Saves time, reduces confusion
- **Use legacy project as oracle** - Compare outputs against /Users/griffnb/projects/swag

---

## 🏗️ Restoration Phases

### Phase 0: Assessment & Baseline ✅ COMPLETE
**Goal:** Understand current state, identify gaps, create plan

**Tasks:**
- ✅ Document what's working vs broken
- ✅ Analyze integration test expectations
- ✅ Review legacy code dependencies
- ✅ Create systematic testing plan

**Deliverables:**
- ✅ This document
- ✅ Clear understanding of 25 vs 40 definition gap

---

### Phase 1: Component Testing Foundation
**Goal:** Ensure lowest-level building blocks work correctly
**Duration:** 1-2 days
**Status:** ✅ COMPLETE (Implemented during Phase 3.1)

#### 1.1 Type Resolution Testing
**File:** `internal/parser/struct/type_resolver_test.go` (create)

**What to test:**
```go
// Test 1: Basic type extraction
fields.StructField[string] → string
fields.StructField[int64] → int64

// Test 2: Nested generics
fields.StructField[*model.Account] → model.Account
fields.StructField[[]string] → []string

// Test 3: Multi-level nesting
Wrapper[Inner[int]] → int

// Test 4: Custom model detection
IsCustomModel("fields.StructField") → true
IsCustomModel("string") → false
```

**Success Criteria:**
- ✅ All type resolution tests pass
- ✅ Generic extraction working
- ✅ Custom model detection working

**Reference:** `/Users/griffnb/projects/swag/generics.go` (522 lines - port this logic)

---

#### 1.2 Field Tag Parsing Testing
**File:** `internal/parser/struct/tag_parser_test.go` (create)

**What to test:**
```go
// Test 1: JSON tag parsing
`json:"first_name"` → name: "first_name"
`json:"count,omitempty"` → name: "count", omitempty: true

// Test 2: Public tag parsing
`public:"view"` → visibility: "view"
`public:"edit"` → visibility: "edit"
(no tag) → visibility: "private"

// Test 3: Validation tags
`validate:"required,min=1,max=100"` → constraints

// Test 4: Combined tags
`json:"name" public:"view" validate:"required"`
```

**Success Criteria:**
- ✅ All tag parsing tests pass
- ✅ Public/private detection working
- ✅ Tag priority rules correct

**Reference:** `/Users/griffnb/projects/swag/field_parser.go` (700 lines - port this logic)

---

#### 1.3 AllOf Composition Testing
**File:** `internal/schema/allof_test.go` (may exist)

**What to test:**
```go
// Test 1: Basic AllOf structure
response.SuccessResponse{data=Account}
→
{
  "allOf": [
    {"$ref": "#/definitions/response.SuccessResponse"},
    {"properties": {"data": {"$ref": "#/definitions/Account"}}}
  ]
}

// Test 2: Multiple field overrides
Parent{field1=Type1,field2=Type2}

// Test 3: Array types
Response{data=[]Account}

// Test 4: Nested AllOf
Response{data=Inner{field=Type}}
```

**Success Criteria:**
- ✅ AllOf generation correct
- ✅ Property override working
- ✅ Nested structures handled

---

### Phase 2: Service Unit Testing
**Goal:** Test individual services in isolation
**Duration:** 2-3 days
**Status:** ✅ COMPLETE (Implemented during Phase 3.1)

#### 2.1 StructParserService Implementation & Testing
**File:** `internal/parser/struct/service.go` (CURRENTLY STUB)
**Test:** `internal/parser/struct/service_test.go` (create)

**What to implement:**
1. Parse struct definitions from AST
2. Extract field information
3. Apply type resolution (from 1.1)
4. Apply tag parsing (from 1.2)
5. Generate base schema
6. Generate Public variant schema if needed
7. Register both in registry

**What to test:**
```go
// Test 1: Simple struct
type Account struct {
    ID   int64  `json:"id"`
    Name string `json:"name"`
}
→ Generates 1 definition: "Account"

// Test 2: Public/Private struct
type Account struct {
    ID       int64  `json:"id" public:"view"`
    Password string `json:"password"` // no public tag
}
→ Generates 2 definitions: "Account" and "AccountPublic"

// Test 3: Custom model struct
type Account struct {
    FirstName fields.StructField[string] `json:"first_name" public:"edit"`
}
→ Extract string, apply tags

// Test 4: Embedded struct
type AccountJoined struct {
    Account
    JoinData
}
→ Merge fields correctly

// Test 5: Generic struct
type Wrapper[T any] struct {
    Data T `json:"data"`
}
type ConcreteWrapper Wrapper[string]
→ Resolve T to string
```

**Success Criteria:**
- ✅ All service tests pass
- ✅ Generates correct number of definitions
- ✅ Public variants created when needed
- ✅ Custom models handled
- ✅ Generic types resolved

**Implementation Steps:**
1. Port logic from `internal/legacy_files/field_parser.go`
2. Port logic from `internal/legacy_files/generics.go`
3. Break into smaller functions (<100 lines each)
4. Write tests for each function
5. Wire into service
6. Test service as a whole

**Reference Files:**
- `/Users/griffnb/projects/swag/field_parser.go` (700 lines)
- `/Users/griffnb/projects/swag/generics.go` (522 lines)
- `/Users/griffnb/projects/core-swag/internal/legacy_files/field_parser.go`
- `/Users/griffnb/projects/core-swag/internal/legacy_files/generics.go`

---

#### 2.2 RouteParserService Integration & Testing
**File:** `internal/parser/route/service.go` (EXISTS but not integrated)
**Test:** `internal/parser/route/service_test.go` (verify exists)

**What to implement:**
1. Wire service into orchestrator
2. Parse function comment annotations
3. Extract route information (@router, @param, @success, @failure)
4. Handle @Public annotation
5. Apply AllOf composition for responses
6. Generate spec.Operation objects

**What to test:**
```go
// Test 1: Basic route
// @Router /account [get]
// @Success 200 {object} Account
→ Generates path with Account reference

// Test 2: @Public annotation
// @Public
// @Router /auth/me [get]
// @Success 200 {object} Account
→ Should reference AccountPublic, not Account

// Test 3: AllOf composition
// @Success 200 {object} response.SuccessResponse{data=Account}
→ Generates AllOf structure

// Test 4: Multiple parameters
// @Param q query string false "search query"
// @Param limit query int false "limit" default(100)
→ Generates parameter array

// Test 5: Multiple response codes
// @Success 200 {object} Account
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
→ Multiple response entries
```

**Success Criteria:**
- ✅ Service wired into orchestrator
- ✅ All route parsing tests pass
- ✅ @Public annotation working
- ✅ AllOf composition working
- ✅ All annotation types supported

**Reference Files:**
- `/Users/griffnb/projects/swag/operation.go` (1,317 lines)
- `/Users/griffnb/projects/core-swag/internal/legacy_files/operation.go`
- `/Users/griffnb/projects/core-swag/internal/parser/route/converter.go` (review)

---

### Phase 3: Integration Testing
**Goal:** Test services working together
**Duration:** 2-3 days
**Status:** ✅ COMPLETE

#### 3.1 TestCoreModelsIntegration - The Critical Test
**File:** `testing/core_models_integration_test.go`
**Current State:** ✅ ALL 6 TESTS PASSING
**Test Data:** `testing/testdata/core_models/`

**What this tests:**
- Parse 41 Go files with custom models
- Generate 87 definitions (base + Public variants)
- Generate 5 API paths
- Apply public/private filtering
- Handle @Public annotation
- Apply AllOf composition

**Final Output:**
```
Files parsed: 41
Definitions generated: 87 (base schemas + Public variants)
Paths generated: 5
All tests: PASSING ✅
```

**Test Results:**
- ✅ Base_schemas_should_exist
- ✅ Public_variant_schemas_should_exist
- ✅ Base_Account_schema_should_have_correct_fields (28 properties with correct types)
- ✅ Public_Account_schema_should_filter_private_fields (22 properties correctly filtered)
- ✅ Operations_should_reference_correct_schemas (AllOf + @Public working)
- ✅ Generate_actual_output (87 definitions, 5 paths)

**What was fixed:**
1. **Embedded struct support** - Recursive field merging for embedded types
2. **Field type resolution** - Added support for `fields.StringField`, `fields.IntField`, etc.
3. **Column tag fallback** - Falls back to `column:` tag when `json:` tag missing
4. **AllOf composition** - Preserved AllOf structure in response wrappers
5. **@Public suffix** - Applied correctly to data models only, not response wrappers

**Success Criteria:**
- ✅ Test passes: 41 files → 87 definitions → 5 paths
- ✅ All Public variants generated
- ✅ Custom models parsed correctly
- ✅ Field types resolved correctly (string, integer, boolean, etc.)
- ✅ Embedded fields merged properly
- ✅ AllOf composition working
- ✅ @Public annotation applied

**Test Command:**
```bash
cd /Users/griffnb/projects/core-swag
go test -v ./testing -run TestCoreModelsIntegration
```

---

#### 3.2 TestRealProjectIntegration
**File:** `testing/core_models_integration_test.go`
**Purpose:** Test against real project structure

**Test Command:**
```bash
cd /Users/griffnb/projects/core-swag
go test -v ./testing -run TestRealProjectIntegration
```

**Success Criteria:**
- ✅ Test passes
- ✅ Generates valid swagger.json
- ✅ No panics or errors

---

### Phase 4: End-to-End Testing
**Goal:** Test against real projects
**Duration:** 1-2 days
**Status:** 🔴 NOT STARTED

#### 4.1 Project 1 - atlas-go
**Location:** `/Users/griffnb/projects/Crowdshield/atlas-go`
**Expected Output:** `testing/project-1-example-swagger.json`

**Test Command:**
```bash
make test-project-1
```

**Full Command:**
```bash
go install ./cmd/core-swag && \
cd /Users/griffnb/projects/Crowdshield/atlas-go && \
core-swag init \
  -g "main.go" \
  -d "./cmd/server,./internal/controllers,./internal/models" \
  --parseInternal \
  -pd \
  -o "./swag_docs"
```

**Success Criteria:**
- ✅ No errors during generation
- ✅ swagger.json generated
- ✅ swagger.yaml generated
- ✅ docs.go generated
- ✅ Output matches project-1-example-swagger.json structure
- ✅ All paths present
- ✅ All definitions present
- ✅ AllOf composition correct

**Validation:**
```bash
# Compare generated to expected
diff \
  /Users/griffnb/projects/Crowdshield/atlas-go/swag_docs/swagger.json \
  /Users/griffnb/projects/core-swag/testing/project-1-example-swagger.json
```

---

#### 4.2 Project 2 - go-the-schwartz
**Location:** `/Users/griffnb/projects/botbuilders/go-the-schwartz`
**Expected Output:** `testing/project-2-example-swagger.json`

**Test Command:**
```bash
make test-project-2
```

**Full Command:**
```bash
go install ./cmd/core-swag && \
cd /Users/griffnb/projects/botbuilders/go-the-schwartz && \
core-swag init \
  -g "main.go" \
  -d "./cmd/server,./internal/controllers,./internal/models,./applications" \
  --parseInternal \
  -pd \
  -o "./swag_docs"
```

**Success Criteria:**
- ✅ No errors during generation
- ✅ swagger.json generated
- ✅ Output matches project-2-example-swagger.json structure
- ✅ All paths present
- ✅ All definitions present

---

### Phase 5: Legacy Code Removal
**Goal:** Remove dependency on legacy code
**Duration:** 1 day
**Status:** 🔴 NOT STARTED

**Files to Remove:**
1. `internal/legacy_files/parser.go` (2,336 lines)
2. `internal/legacy_files/operation.go` (1,317 lines)
3. `internal/legacy_files/packages.go` (788 lines)
4. `internal/legacy_files/field_parser.go` (700 lines)
5. `internal/legacy_files/generics.go` (522 lines)
6. Other legacy files (~10,000+ more lines)

**Prerequisites:**
- ✅ Phase 3 integration tests passing
- ✅ Phase 4 E2E tests passing
- ✅ All functionality ported to new services

**Process:**
1. Comment out legacy file imports in orchestrator
2. Run all tests
3. If tests fail, identify missing functionality
4. Port missing functionality
5. Re-run tests
6. Repeat until all tests pass
7. Delete legacy files
8. Remove `internal/legacy_files/` directory

**Success Criteria:**
- ✅ All tests still passing
- ✅ Zero legacy code references
- ✅ Directory removed

---

### Phase 6: Code Quality & Compliance
**Goal:** Meet project standards
**Duration:** 1 day
**Status:** 🔴 NOT STARTED

**Tasks:**

#### 6.1 File Size Compliance
**Rule:** No files > 500 lines

**Current Violations:**
- Check all files in new implementation
- Split any files > 500 lines

**Process:**
1. Run: `find internal -name "*.go" -exec wc -l {} \; | sort -rn | head -20`
2. Identify files > 500 lines
3. Refactor into smaller, cohesive modules
4. Update tests

#### 6.2 Documentation
**Rule:** All functions need godoc comments

**Tasks:**
1. Add package-level documentation
2. Add function-level documentation for all exported functions
3. Document complex internal functions
4. Add examples for key functions

**Files to Document:**
- `internal/parser/struct/service.go`
- `internal/parser/route/service.go`
- Any new helper files created

#### 6.3 Test Coverage
**Goal:** >80% coverage for new code

**Command:**
```bash
make test
# Review coverage.out
```

**Tasks:**
1. Add missing unit tests
2. Add edge case tests
3. Add error case tests

---

## 🧪 Testing Checklist

### Component Tests (Phase 1)
- [ ] Type resolution tests passing
- [ ] Field tag parsing tests passing
- [ ] AllOf composition tests passing
- [ ] Generic type extraction tests passing
- [ ] Custom model detection tests passing

### Service Tests (Phase 2)
- [ ] StructParserService tests passing
- [ ] RouteParserService tests passing
- [ ] Service integration with orchestrator verified
- [ ] Public variant generation working
- [ ] @Public annotation handling working

### Integration Tests (Phase 3)
- [ ] TestCoreModelsIntegration passing (41→40→5)
- [ ] TestRealProjectIntegration passing
- [ ] Custom model parsing verified
- [ ] Public/private filtering verified
- [ ] AllOf composition verified

### E2E Tests (Phase 4)
- [ ] make test-project-1 succeeds
- [ ] Project 1 output matches expected
- [ ] make test-project-2 succeeds
- [ ] Project 2 output matches expected
- [ ] No errors or panics in real projects

### Cleanup (Phase 5)
- [ ] Legacy code removed
- [ ] All tests still passing
- [ ] No import references to legacy_files

### Quality (Phase 6)
- [ ] All files < 500 lines
- [ ] All functions documented
- [ ] Test coverage > 80%
- [ ] CHANGE_LOG.md updated

---

## 🚀 Execution Order

**DO NOT SKIP AHEAD** - Each phase depends on previous phases passing.

```
Phase 1: Components (1-2 days)
   ├── 1.1 Type Resolution Tests
   ├── 1.2 Field Tag Tests
   └── 1.3 AllOf Tests
   ↓
Phase 2: Services (2-3 days)
   ├── 2.1 StructParserService
   └── 2.2 RouteParserService
   ↓
Phase 3: Integration (2-3 days)
   ├── 3.1 TestCoreModelsIntegration ← CRITICAL
   └── 3.2 TestRealProjectIntegration
   ↓
Phase 4: E2E (1-2 days)
   ├── 4.1 Project 1 (atlas-go)
   └── 4.2 Project 2 (go-the-schwartz)
   ↓
Phase 5: Cleanup (1 day)
   └── Remove legacy code
   ↓
Phase 6: Quality (1 day)
   ├── File size compliance
   ├── Documentation
   └── Test coverage
```

**Total Estimated Duration:** 8-12 days

---

## 📝 Key Reference Files

### Current Project
- **Integration Test:** `testing/core_models_integration_test.go`
- **Test Data:** `testing/testdata/core_models/`
- **Expected Outputs:** `testing/project-1-example-swagger.json`, `testing/project-2-example-swagger.json`
- **Orchestrator:** `internal/orchestrator/service.go`
- **Services to Complete:**
  - `internal/parser/struct/service.go` (STUB)
  - `internal/parser/route/service.go` (EXISTS, needs integration)

### Legacy Project (Reference Only)
- **Parser:** `/Users/griffnb/projects/swag/parser.go`
- **Operation Parser:** `/Users/griffnb/projects/swag/operation.go`
- **Field Parser:** `/Users/griffnb/projects/swag/field_parser.go`
- **Generics Handler:** `/Users/griffnb/projects/swag/generics.go`
- **Package Registry:** `/Users/griffnb/projects/swag/packages.go`

### Test Projects
- **Project 1:** `/Users/griffnb/projects/Crowdshield/atlas-go`
- **Project 2:** `/Users/griffnb/projects/botbuilders/go-the-schwartz`

---

## 🎯 Success Metrics

### Must Pass
- ✅ TestCoreModelsIntegration: 41 files → 40 definitions → 5 paths
- ✅ make test-project-1: generates valid swagger.json
- ✅ make test-project-2: generates valid swagger.json
- ✅ All unit tests passing
- ✅ All integration tests passing

### Must Remove
- ✅ internal/legacy_files/ directory deleted
- ✅ Zero references to legacy code
- ✅ Zero files > 500 lines

### Must Have
- ✅ Custom model support (fields.StructField[T])
- ✅ Public/Private filtering
- ✅ @Public annotation
- ✅ AllOf composition
- ✅ Generic type resolution
- ✅ Complete documentation

---

## 🔍 Current Gap Analysis

**25 vs 40 Definitions Gap:**

The critical test expects 40 definitions but only gets 25. This is a **15 definition shortfall**.

**Root Cause:**
- StructParserService is a stub → not generating Public variants
- Not detecting `public:"view|edit"` tags
- Not creating *Public schemas

**Expected Definitions Pattern:**
```
Account → generates Account + AccountPublic (2 definitions)
Session → generates Session + SessionPublic (2 definitions)
Profile → generates Profile + ProfilePublic (2 definitions)
... etc
```

**If 15 models need Public variants:**
- 15 base models = 15 definitions
- 15 Public variants = 15 definitions
- Total = 30 definitions
- Plus 10 other models without public tags = 10 definitions
- **Grand Total = 40 definitions ✓**

**Fix:**
- Implement StructParserService
- Detect public tags
- Generate Public variants
- Register both variants

---

## 📋 Next Steps

1. **Start with Phase 1.1** - Type resolution tests
2. **Update CHANGE_LOG.md** with each attempt
3. **Use legacy project as reference** when stuck
4. **Don't skip ahead** - each phase builds on previous
5. **Test continuously** - catch issues early

---

## 🚨 Red Flags to Watch For

- **Skipping tests** - Every phase must have passing tests before moving on
- **Large files** - Split any file approaching 500 lines immediately
- **Copying legacy code wholesale** - Refactor into smaller, cleaner functions
- **Missing documentation** - Document as you go, not at the end
- **Dual-write patterns** - Temporary only, remove ASAP
- **Placeholder code** - No TODOs or "implement later" comments

---

## 🎉 When Complete

The project will:
- Generate full OpenAPI 2.0 swagger documentation
- Support all custom model patterns
- Support public/private field filtering
- Support all annotation types
- Have zero legacy code dependencies
- Meet all project quality standards
- Pass all tests consistently
- Work on real production projects

**This is a complete, production-ready port of the legacy swag tool with improved architecture.**
