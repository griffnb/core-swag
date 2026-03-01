## This Is a Port of a legacy project

## ⚠️ CRITICAL: When you dont know what code is supposed to do, reference the previous project until this is completed
Previous Project is located here: `/Users/griffnb/projects/swag`


## ⚠️ CRITICAL: The most important test
`testing/core_models_integration_test.go` :  `TestRealProjectIntegration`
To run on ACTUAL projects, theres 2 make commands to give full true outputs on real projects:
`make test-project-1`
Example output: testing/project-1-example-swagger.json
`make test-project-2`
Example output: testing/project-2-example-swagger.json

## ⚠️ CRITICAL: ALWAYS FOLLOW DOCUMENTATION AND PRD
**MANDATORY REQUIREMENT**: Before making ANY changes to this codebase, you MUST:

1. **Maintain consistency**: Any new features, APIs, or changes must align with existing patterns
2. If go docs are missing from a function or package, and you learn something important about it, ADD TO YOUR TODO LIST THAT YOU NEED TO UPDATE THAT GO DOC WITH WHAT YOU LEARNED
3. **VERY IMPORTANT** Do not make large files with lots of functionality.  Group functions together into files that relate them together.  This makes it easier to find grouped functions and their associated tests.  **LARGE FILES ARE BAD**
4. **CHANGE LOG** YOU MUST WITHOUT FAIL DO -> When you try something and it doesnt work, add to the change log ./.agents/change_log.md What you tried, why it didnt work, what you are trying next.


## 🔄 CHECKLIST UPDATE POLICY

**NEVER FORGET**: When you complete any phase, feature, or major milestone:

1. **IMMEDIATELY** update the todo list to mark items as completed
2. **ADD NEW PHASES** to the checklist as they are planned and implemented  
3. **KEEP DOCUMENTATION CURRENT** - the checklist should always reflect the actual project state
4. **UPDATE STATUS** for any infrastructure, integrations, or features that are now working

This ensures the checklist remains an accurate reflection of project progress and helps future development sessions.

**When implementing new features**:
1. Follow established patterns and conventions
2. Update documentation if adding new patterns

**IMPORTANT Before you begin, always launch the context-fetcher sub agent to gather the information required for the task.**

## 📐 STRUCT PARSING ARCHITECTURE

**Updated: 2026-02-16 (Ralph Loop Consolidation Complete)**

This project uses **TWO struct parsers** that work together, each serving a specific purpose:

### 1. StructParserService (Deprecated)
**Location:** `internal/parser/struct/service.go`
**Used by:** No production callers (removed from orchestrator in performance optimization)
**Status:** Deprecated — kept for test coverage only, scheduled for full removal
**Purpose:** Was file-level struct parsing during code generation, now replaced by demand-driven CoreStructParser pipeline

### 2. CoreStructParser (Schema-level)
**Location:** `internal/model/struct_field_lookup.go` (~775 lines)
**Used by:** SchemaBuilder (`internal/schema/builder.go`)
**Purpose:** Type-level struct field extraction with full type information
**Features:**
- Uses `go/packages` for complete type information
- Recursive field extraction with cross-package support
- Handles `fields.StructField[T]` generic expansion
- Global caching for performance
- Detects extended primitives (time.Time, UUID, decimal.Decimal)
- Supports Public variant generation

**When to use:** When building schemas via SchemaBuilder

### Core Implementation (Shared)
**Location:** `internal/model/struct_field.go` (~538 lines)
**Type:** `StructField` with `ToSpecSchema()` method
**Features:** ALL struct parsing features are implemented here:
- Extended primitives (time.Time → string+date-time, UUID → string+uuid, decimal → number)
- Enum detection and inlining (via TypeEnumLookup interface)
- Generic extraction (StructField[T], IntConstantField[T], StringConstantField[T])
- Embedded field merging
- Validation constraints (min/max, required, minLength/maxLength, pattern)
- Public mode filtering (public:"view|edit" tags)
- SwaggerIgnore handling (swaggerignore:"true", json:"-")
- Arrays and maps (recursive type resolution)
- Nested struct references ($ref with #/definitions/)
- Force required parameter

### Consolidation History
**Phase 1 (Iteration 1):** Verified all features implemented in struct_field.go
**Phase 2 (Iteration 2):** Removed SchemaBuilder fallback code (228 lines deleted)
**Result:** SchemaBuilder now exclusively uses CoreStructParser (no duplicate parsing)

### Key Design Decision
The orchestrator now uses a demand-driven pipeline where CoreStructParser builds schemas
only for route-referenced types. StructParserService is deprecated and no longer called.
- **CoreStructParser** handles all schema building (types, go/packages, Public variants)
- **StructField.ToSpecSchema()** is the shared implementation for all schema generation
