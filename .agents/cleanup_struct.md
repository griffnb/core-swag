# Unified Struct Parser - Cleanup & Consolidation Plan

## Executive Summary

**Problem:** Three separate struct parsing implementations with ~9,000 lines of duplicated logic causing inconsistent behavior and maintenance burden.

**Solution:** Consolidate into ONE canonical unified parser with pluggable backends, clear separation of concerns, and incremental migration path.

**Timeline:** 3-4 weeks with 6 phases
**Expected Outcome:** ~40% code reduction (9,000 → 5,400 lines), single source of truth, consistent features

---

## Current State: Three Separate Implementations

### 1. CoreStructParser (`internal/model/`)

**Files:**
- `struct_field_lookup.go` (775 lines)
- `struct_field.go` (538 lines)
- `struct_builder.go` (66 lines)

**Strengths:**
- ✅ Full type information via go/packages
- ✅ Handles fields.StructField[T] wrapper extraction
- ✅ Enum detection and inline enum values
- ✅ Public mode support (public:"view|edit" tags)
- ✅ Comprehensive extended primitives (time.Time, UUID, decimal)
- ✅ Creates proper $ref for nested structs
- ✅ Caching for performance

**Weaknesses:**
- ❌ Requires go/packages (heavy dependency)
- ❌ Tightly coupled to StructBuilder/StructField models
- ❌ Complex recursive logic
- ❌ No AST-only fallback

**Used by:** SchemaBuilder via SetStructParser

---

### 2. Struct Parser Service (`internal/parser/struct/`)

**Files:**
- `service.go` (530 lines)
- `field_processor.go` (464 lines) - **JUST FIXED**
- `type_resolver.go` (174 lines)
- `tag_parser.go` (245 lines)

**Strengths:**
- ✅ Works with AST only (no types.Type required)
- ✅ Clean service architecture
- ✅ Proper tag parsing
- ✅ Type resolution utilities
- ✅ Handles embedded fields
- ✅ Public schema generation
- ✅ **NOW: Extended primitives, proper refs, enums** (after recent fixes)

**Weaknesses:**
- ⚠️ Recently added, still incomplete edge cases
- ⚠️ Some duplicate logic with CoreStructParser

**Used by:** Orchestrator.Parse() - adds schemas directly to SchemaBuilder

---

### 3. Schema Builder Fallback (`internal/schema/builder.go`)

**Files:**
- `builder.go` lines 114-203 (BuildSchema fallback)
- `builder.go` lines 268-432 (getFieldType, buildFieldSchema)

**Strengths:**
- ✅ Comprehensive extended primitive handling
- ✅ Proper enum detection via enumLookup
- ✅ Creates $ref for nested types
- ✅ Works from AST
- ✅ Handles arrays, maps, interfaces

**Weaknesses:**
- ❌ Embedded in SchemaBuilder (poor separation)
- ❌ Duplicate logic with other parsers
- ❌ Used as "fallback" rather than primary

**Used by:** SchemaBuilder.BuildSchema() when CoreStructParser unavailable

---

## Problem Summary

### 1. Triple Code Duplication
- Primitive type handling duplicated 3x
- Enum detection logic duplicated
- Array/slice handling duplicated
- $ref creation logic duplicated

### 2. Inconsistent Features

| Feature | CoreStructParser | Struct Service | Schema Fallback |
|---------|-----------------|----------------|-----------------|
| Extended primitives | ✅ Full | ✅ Full (NOW) | ✅ Full |
| Enum detection | ✅ Yes | ❌ No | ✅ Yes |
| fields.StructField[T] | ✅ Yes | ⚠️ Partial | ❌ No |
| Public mode | ✅ Yes | ✅ Yes | ❌ No |
| AST-only | ❌ No | ✅ Yes | ✅ Yes |
| Embedded fields | ✅ Yes | ✅ Yes | ⚠️ Partial |

### 3. Race Conditions & Order Dependencies
- Struct parser adds schemas to SchemaBuilder first
- Then SchemaBuilder checks if exists and returns early
- Order-dependent behavior is fragile

### 4. Maintenance Nightmare
- Bug fixes require updating 3 locations
- Features must be implemented 3x
- Testing requires 3x effort
- No single source of truth

---

## Proposed Unified Architecture

### Design Principles

1. **Single Responsibility** - One canonical parser with clear phases
2. **Pluggable Backends** - Support both AST-only and go/packages modes
3. **Layered Design** - Separate concerns into discrete layers
4. **Incremental Migration** - Can adopt progressively without breaking changes
5. **Testability** - Each layer independently testable

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────┐
│                  Public API Layer                        │
│  UnifiedStructParser - Main entry point                 │
│  - ParseField(field, context) → FieldSchema             │
│  - ParseStruct(structType, context) → StructSchema      │
│  - ParseType(typeExpr, context) → Schema               │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│              Field Processing Layer                      │
│  FieldProcessor - Converts fields to schemas            │
│  - ProcessField(field) → PropertySchema                 │
│  - DetermineRequired(tags) → bool                       │
│  - ApplyConstraints(schema, tags) → Schema             │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│               Type Resolution Layer                      │
│  TypeResolver - Resolves Go types to OpenAPI types      │
│  - ResolveType(expr) → TypeInfo                        │
│  - IsPrimitive(type) → bool                            │
│  - IsEnum(type) → (bool, []EnumValue)                  │
│  - IsExtendedPrimitive(type) → bool                    │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│                Backend Abstraction Layer                 │
│  TypeBackend interface - Pluggable type info source     │
│  - GetTypeInfo(expr) → TypeDetails                      │
│  - ResolveTypeAlias(name) → UnderlyingType             │
│                                                          │
│  Implementations:                                        │
│  - ASTBackend: Works from AST only                      │
│  - PackagesBackend: Uses go/packages for full info      │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│                 Utility Services                         │
│  - TagParser: Parse struct tags                         │
│  - EnumLookup: Find enum values                         │
│  - SchemaBuilder: Construct spec.Schema                 │
│  - NamingStrategy: Apply naming conventions             │
└─────────────────────────────────────────────────────────┘
```

---

## Component Specifications

### 1. UnifiedStructParser (Core Entry Point)

**Location:** `internal/parser/unified/parser.go` (~400 lines)

**Interface:**
```go
type UnifiedStructParser struct {
    backend       TypeBackend
    tagParser     *TagParser
    enumLookup    EnumLookup
    typeCache     map[string]*ParsedType
    options       ParserOptions
}

type ParserOptions struct {
    Public            bool          // Filter for public fields
    ForceRequired     bool          // All fields required
    NamingStrategy    string        // camelCase, PascalCase, snake_case
    ParseExtensions   bool          // Parse x-* extensions
}

// Main entry points
func (p *UnifiedStructParser) ParseField(field *ast.Field, file *ast.File) (*FieldSchema, error)
func (p *UnifiedStructParser) ParseStruct(structType *ast.StructType, file *ast.File) (*StructSchema, error)
func (p *UnifiedStructParser) ParseType(typeExpr ast.Expr, file *ast.File) (*TypeSchema, error)

// Legacy compatibility - keep existing signatures
func (p *UnifiedStructParser) ToSpecSchema(field *StructField, public bool, forceRequired bool) (string, *spec.Schema, bool, []string, error)
func (p *UnifiedStructParser) BuildSpecSchema(typeName string, fields []*StructField, public bool) (*spec.Schema, []string, error)
```

**Dependencies:**
- TypeBackend (pluggable)
- TagParser (reuse existing)
- EnumLookup (reuse existing)
- SchemaConstructor (new utility)

---

### 2. TypeBackend (Abstraction Layer)

**Location:** `internal/parser/unified/backend.go` (~200 lines)

**Purpose:** Abstract over AST vs go/packages type information

**Interface:**
```go
type TypeBackend interface {
    // Get detailed type information
    GetTypeInfo(expr ast.Expr, file *ast.File) (*TypeInfo, error)

    // Resolve type aliases
    ResolveAlias(typeName string, file *ast.File) (*TypeInfo, error)

    // Check if type is defined
    IsDefined(typeName string, file *ast.File) bool

    // Get package information
    GetPackage(pkgPath string) (*PackageInfo, error)
}

type TypeInfo struct {
    Name          string        // Short name: "Account"
    QualifiedName string        // Full name: "account.Account"
    PackagePath   string        // github.com/.../account
    Kind          TypeKind      // Struct, Primitive, Enum, Array, etc.
    Underlying    ast.Expr      // Underlying AST expression
    Fields        []*ast.Field  // For struct types
    // For go/packages backend
    TypesType     types.Type    // Full type info (optional)
}

type TypeKind int
const (
    KindStruct TypeKind = iota
    KindPrimitive
    KindEnum
    KindArray
    KindMap
    KindPointer
    KindInterface
)
```

**Implementations:**

#### ASTBackend (`backend_ast.go` ~200 lines)
- Lightweight, AST-only parsing
- Uses registry for type lookup
- No external dependencies

```go
type ASTBackend struct {
    registry *registry.Service  // For type lookup
    files    map[string]*ast.File
}
```

#### PackagesBackend (`backend_packages.go` ~200 lines)
- Full type information via go/packages
- Slower but more accurate
- Replaces CoreStructParser's package loading

```go
type PackagesBackend struct {
    packages map[string]*packages.Package
    cache    map[string]*TypeInfo
}
```

---

### 3. TypeResolver (Type Analysis Layer)

**Location:** `internal/parser/unified/type_resolver.go` (~600 lines)

**Responsibilities:**
- Determine if type is primitive, enum, struct, etc.
- Handle extended primitives (time.Time, UUID, decimal)
- Extract type parameters from generics (fields.StructField[T])
- Resolve package-qualified types

**Key Functions:**
```go
type TypeResolver struct {
    backend    TypeBackend
    enumLookup EnumLookup
}

func (r *TypeResolver) ResolveType(expr ast.Expr, file *ast.File) (*ResolvedType, error)

type ResolvedType struct {
    Kind          TypeKind
    Schema        *spec.Schema      // For primitives/enums
    Reference     string            // For nested structs
    NestedTypes   []string          // Types that need definitions
    ElementType   *ResolvedType     // For arrays
    ValueType     *ResolvedType     // For maps
}

// Key methods
func (r *TypeResolver) IsPrimitive(typeInfo *TypeInfo) bool
func (r *TypeResolver) IsExtendedPrimitive(typeInfo *TypeInfo) bool
func (r *TypeResolver) IsEnum(typeInfo *TypeInfo) (bool, []EnumValue)
func (r *TypeResolver) ExtractGenericParameter(typeExpr ast.Expr) (ast.Expr, error)
func (r *TypeResolver) HandleFieldsWrapper(typeInfo *TypeInfo) (*ResolvedType, error)
```

**This is the HEART of the unified parser - handles 80% of complexity**

---

### 4. FieldProcessor (Field Schema Generator)

**Location:** `internal/parser/unified/field_processor.go` (~500 lines)

**Responsibilities:**
- Process individual struct fields
- Apply tags (json, public, validate)
- Determine required status
- Apply validation constraints
- Handle embedded fields

**Key Logic:**
```go
type FieldProcessor struct {
    typeResolver *TypeResolver
    tagParser    *TagParser
    options      ParserOptions
}

func (p *FieldProcessor) ProcessField(field *ast.Field, file *ast.File) (*FieldSchema, error) {
    // 1. Parse tags
    tags := p.tagParser.ParseTags(field.Tag)

    // 2. Check filters (swaggerignore, public mode)
    if p.shouldSkipField(field, tags) {
        return nil, nil
    }

    // 3. Resolve field type
    resolvedType := p.typeResolver.ResolveType(field.Type, file)

    // 4. Build schema
    schema := p.buildSchema(resolvedType, tags)

    // 5. Determine required
    required := p.determineRequired(field, tags, resolvedType)

    // 6. Apply constraints
    p.applyConstraints(schema, tags)

    return &FieldSchema{...}, nil
}
```

---

### 5. Supporting Services

**SchemaConstructor** (NEW: ~200 lines)
```go
type SchemaConstructor struct {
    namingStrategy string
}

func (c *SchemaConstructor) BuildStructSchema(fields []*FieldSchema) *spec.Schema
func (c *SchemaConstructor) BuildPrimitiveSchema(typeInfo *TypeInfo) *spec.Schema
func (c *SchemaConstructor) BuildEnumSchema(enumValues []EnumValue) *spec.Schema
func (c *SchemaConstructor) BuildArraySchema(elementSchema *spec.Schema) *spec.Schema
func (c *SchemaConstructor) BuildRefSchema(typeName string, public bool) *spec.Schema
```

**TagParser** - REUSE existing `internal/parser/struct/tag_parser.go`
**EnumLookup** - REUSE existing `internal/model/enum_lookup.go`

---

## File Structure

```
internal/parser/unified/
├── parser.go              # UnifiedStructParser - main entry (~400 lines)
├── backend.go             # TypeBackend interface (~50 lines)
├── backend_ast.go         # ASTBackend implementation (~200 lines)
├── backend_packages.go    # PackagesBackend implementation (~200 lines)
├── type_resolver.go       # TypeResolver (~600 lines)
├── field_processor.go     # FieldProcessor (~500 lines)
├── schema_constructor.go  # SchemaConstructor (~200 lines)
├── options.go             # ParserOptions config (~100 lines)
├── cache.go               # Caching utilities (~150 lines)
├── primitives.go          # Primitive type maps (~150 lines)
├── generics.go            # Generic type handling (~150 lines)
│
├── parser_test.go         # Integration tests (~400 lines)
├── backend_test.go        # Backend tests (~200 lines)
├── type_resolver_test.go  # TypeResolver tests (~300 lines)
├── field_processor_test.go # FieldProcessor tests (~300 lines)
└── README.md              # Package documentation
```

**Estimated totals:**
- New code: ~2,800 lines (implementation + tests)
- Removed: ~3,800 lines (duplicated code from 3 parsers)
- **Net reduction: ~1,000 lines**

---

## Migration Strategy - 6 Phases

### Phase 1: Create Unified Core (Week 1, Days 1-3)

**Goal:** Build foundation without breaking existing code

**Tasks:**
1. Create `internal/parser/unified/` package structure
2. Implement TypeBackend interface
3. Implement ASTBackend (minimal)
4. Implement TypeResolver core
5. Implement FieldProcessor core
6. Write unit tests for each component (TDD)

**Deliverables:**
- [ ] `backend.go` + `backend_ast.go` with tests
- [ ] `type_resolver.go` with primitive detection + tests
- [ ] `field_processor.go` with basic field handling + tests
- [ ] `parser.go` skeleton + tests
- [ ] 80%+ test coverage

**Success Criteria:**
- All existing tests still pass
- Can parse simple struct with primitives
- No changes to existing code

**Risk:** None - purely additive

---

### Phase 2: Feature Parity (Week 1-2, Days 4-7)

**Goal:** Unified parser handles ALL cases from existing parsers

**Tasks:**
1. Add extended primitive support (time.Time, UUID, decimal)
2. Add fields.StructField[T] extraction (generics.go)
3. Add enum detection and inlining
4. Add public mode filtering
5. Add embedded field merging
6. Add array/map/pointer handling
7. Add validation constraints (min, max, pattern)
8. Add caching layer
9. Integration tests with real structs

**Deliverables:**
- [ ] `primitives.go` - Extended primitive maps
- [ ] `generics.go` - StructField[T] extraction
- [ ] Enum support in TypeResolver
- [ ] Public mode in FieldProcessor
- [ ] `cache.go` - Memoization
- [ ] Integration tests pass

**Success Criteria:**
- Feature matrix 100% green
- Can parse complex nested structs
- Can parse fields.StructField[T]
- Can detect and inline enums
- Performance < 2x CoreStructParser

**Risk:** Medium - complex features

---

### Phase 3: Integrate with SchemaBuilder (Week 2, Days 8-10)

**Goal:** Replace Schema Builder's fallback logic

**Tasks:**
1. Add UnifiedStructParser to SchemaBuilder
2. Update BuildSchema() to use unified parser
3. Remove buildFieldSchema() fallback logic
4. Remove getFieldType() duplicate code
5. Update tests
6. Performance profiling
7. Side-by-side comparison tests

**Files Modified:**
- `internal/schema/builder.go` - Remove lines 268-432 (fallback)
- `internal/schema/builder.go` - Update BuildSchema() to use unified parser

**Deliverables:**
- [ ] SchemaBuilder uses UnifiedStructParser
- [ ] Fallback code removed
- [ ] All schema builder tests pass
- [ ] Performance acceptable

**Success Criteria:**
- Zero schema regressions
- Fallback code deleted
- Performance maintained

**Risk:** Low - schema builder is secondary path

---

### Phase 4: Integrate with Orchestrator (Week 2-3, Days 11-14)

**Goal:** Replace Struct Parser Service with unified parser

**Tasks:**
1. Update Orchestrator to use UnifiedStructParser
2. Create adapter for ParseFile() flow
3. Migrate public schema generation
4. Remove structparser.Service dependency
5. Update all Orchestrator tests
6. Integration test with real projects

**Files Modified:**
- `internal/orchestrator/service.go` - Use UnifiedStructParser
- Remove dependency on `internal/parser/struct/service.go`

**Deliverables:**
- [ ] Orchestrator uses unified parser
- [ ] Step 3.5 (struct parsing) simplified
- [ ] All orchestrator tests pass
- [ ] make test-project-1 generates correct output
- [ ] make test-project-2 generates correct output

**Success Criteria:**
- Real project tests pass
- Output identical to current
- No performance regression

**Risk:** HIGH - orchestrator is main code path

---

### Phase 5: Migrate CoreStructParser (Week 3, Days 15-18)

**Goal:** Move CoreStructParser logic into PackagesBackend

**Tasks:**
1. Implement PackagesBackend using go/packages
2. Port caching logic from CoreStructParser
3. Port recursive field extraction
4. Create adapter for BuildAllSchemas() API
5. Update all CoreStructParser callers
6. Mark CoreStructParser as deprecated

**Files Modified:**
- `internal/parser/unified/backend_packages.go` - Port CoreStructParser logic
- `internal/model/struct_field_lookup.go` - Add deprecation notice

**Deliverables:**
- [ ] PackagesBackend feature-complete
- [ ] All CoreStructParser callers migrated
- [ ] Deprecation notices added
- [ ] Documentation updated

**Success Criteria:**
- PackagesBackend produces identical output
- All tests pass
- Zero references to CoreStructParser in main paths

**Risk:** Medium - complex migration

---

### Phase 6: Cleanup & Delete (Week 3-4, Days 19-21)

**Goal:** Remove all deprecated code

**Tasks:**
1. Delete `internal/parser/struct/service.go`
2. Delete `internal/parser/struct/field_processor.go`
3. Delete Schema Builder fallback methods (already done in Phase 3)
4. Archive CoreStructParser (move to `/archive` or delete)
5. Update all documentation
6. Clean up imports
7. Remove unused dependencies
8. Final integration tests
9. Update README.md and CLAUDE.md

**Files Deleted:**
- `internal/parser/struct/service.go` (530 lines)
- `internal/parser/struct/field_processor.go` (464 lines)
- Optionally: `internal/model/struct_field_lookup.go` (775 lines)
- Optionally: `internal/model/struct_field.go` (538 lines)
- Optionally: `internal/model/struct_builder.go` (66 lines)

**Deliverables:**
- [ ] All deprecated code deleted
- [ ] Documentation updated
- [ ] Zero dead code
- [ ] Clean imports
- [ ] Updated architecture diagrams

**Success Criteria:**
- Single struct parsing implementation
- All tests pass
- Documentation accurate
- Code reduction achieved

**Risk:** Low - just cleanup

---

## Entry Points & Usage Examples

### For Schema Builder (Replacing Fallback)

**BEFORE:**
```go
// In schema/builder.go lines 114-203
switch t := typeSpec.TypeSpec.Type.(type) {
case *ast.StructType:
    // Complex fallback logic...
    for _, field := range structType.Fields.List {
        propSchema := b.buildFieldSchema(field.Type, typeSpec.File, example)
        schema.Properties[jsonName] = propSchema
    }
}
```

**AFTER:**
```go
// In schema/builder.go
switch t := typeSpec.TypeSpec.Type.(type) {
case *ast.StructType:
    if b.unifiedParser != nil {
        schema, nestedTypes, err := b.unifiedParser.ParseStruct(t, typeSpec.File)
        if err == nil {
            return schema, nestedTypes, nil
        }
    }
    // Minimal fallback if parser unavailable
}
```

---

### For Orchestrator (Replacing Struct Service)

**BEFORE:**
```go
// orchestrator/service.go lines 288-305
if s.structParser != nil {
    for astFile, fileInfo := range loadResult.Files {
        err = s.structParser.ParseFile(astFile, fileInfo.Path)
        if err != nil {
            return nil, err
        }
    }
}
```

**AFTER:**
```go
// orchestrator/service.go
if s.unifiedParser != nil {
    for astFile, fileInfo := range loadResult.Files {
        schemas, err := s.unifiedParser.ParseFileStructs(astFile, fileInfo.Path)
        if err != nil {
            return nil, err
        }
        for name, schema := range schemas {
            s.schemaBuilder.AddDefinition(name, schema)
        }
    }
}
```

---

### For CoreStructParser Callers (Via Adapter)

**BEFORE:**
```go
parser := &model.CoreStructParser{}
parser.LoadPackages(baseModule, packages)
builder := parser.LookupStructFields(baseModule, pkgPath, typeName)
schema, nestedTypes, err := builder.BuildSpecSchema(typeName, public, forceRequired, enumLookup)
```

**AFTER:**
```go
// Use PackagesBackend for full type info
backend := unified.NewPackagesBackend(baseModule, packages)
parser := unified.NewParser(backend, unified.ParserOptions{
    Public:        public,
    ForceRequired: forceRequired,
})
parser.SetEnumLookup(enumLookup)
schema, nestedTypes, err := parser.ParseStructByName(pkgPath, typeName)
```

---

## Testing Strategy

### Unit Tests (Per Component)

#### TypeBackend Tests (~200 lines)
```go
func TestASTBackend_ResolveLocalType(t *testing.T)
func TestASTBackend_ResolveImportedType(t *testing.T)
func TestASTBackend_ResolveAlias(t *testing.T)
func TestPackagesBackend_GetFullTypeInfo(t *testing.T)
func TestPackagesBackend_CachingWorks(t *testing.T)
```

#### TypeResolver Tests (~300 lines)
```go
func TestTypeResolver_PrimitiveDetection(t *testing.T)
func TestTypeResolver_ExtendedPrimitives(t *testing.T)
func TestTypeResolver_EnumDetection(t *testing.T)
func TestTypeResolver_GenericExtraction(t *testing.T)
func TestTypeResolver_ArrayHandling(t *testing.T)
func TestTypeResolver_MapHandling(t *testing.T)
func TestTypeResolver_PointerStripping(t *testing.T)
```

#### FieldProcessor Tests (~300 lines)
```go
func TestFieldProcessor_TagParsing(t *testing.T)
func TestFieldProcessor_RequiredDetermination(t *testing.T)
func TestFieldProcessor_PublicFiltering(t *testing.T)
func TestFieldProcessor_EmbeddedFields(t *testing.T)
func TestFieldProcessor_ValidationConstraints(t *testing.T)
func TestFieldProcessor_SwaggerIgnore(t *testing.T)
```

#### UnifiedStructParser Tests (~400 lines)
```go
func TestUnifiedParser_SimplePrimitives(t *testing.T)
func TestUnifiedParser_NestedStructs(t *testing.T)
func TestUnifiedParser_ArraysOfStructs(t *testing.T)
func TestUnifiedParser_FieldsWrappers(t *testing.T)
func TestUnifiedParser_EnumInlining(t *testing.T)
func TestUnifiedParser_PublicSchemas(t *testing.T)
func TestUnifiedParser_RecursiveTypes(t *testing.T)
```

### Integration Tests

**Real Project Tests:**
- Use existing `testing/core_models_integration_test.go`
- `TestRealProjectIntegration` - Must pass with identical output
- `make test-project-1` - Verify swagger.json matches expected
- `make test-project-2` - Verify swagger.json matches expected

**Migration Tests:**
```go
func TestSchemaBuilder_OutputIdentical(t *testing.T) {
    // Compare old vs new output
}

func TestOrchestrator_OutputIdentical(t *testing.T) {
    // Compare old vs new output
}

func TestCoreStructParser_AdapterWorks(t *testing.T) {
    // Verify adapter produces same results
}
```

### Performance Tests

**Benchmarks:**
```go
func BenchmarkUnifiedParser_SimpleStruct(b *testing.B)
func BenchmarkUnifiedParser_NestedStruct(b *testing.B)
func BenchmarkUnifiedParser_LargeStruct(b *testing.B)
func BenchmarkUnifiedParser_vs_CoreStructParser(b *testing.B)
func BenchmarkASTBackend_vs_PackagesBackend(b *testing.B)
```

**Acceptance Criteria:**
- UnifiedParser < 2x slower than CoreStructParser
- < 100ms for typical struct (10-20 fields)
- < 500ms for complex struct (100+ fields with nesting)
- ASTBackend < 50% overhead vs PackagesBackend

---

## Risk Mitigation

### Risk 1: Breaking Existing Functionality

**Likelihood:** Medium
**Impact:** High

**Mitigation:**
- Keep all 3 implementations during migration
- Feature flags to toggle parsers
- Comprehensive integration tests
- Side-by-side comparison tests
- Rollback plan for each phase

### Risk 2: Performance Regression

**Likelihood:** Medium
**Impact:** Medium

**Mitigation:**
- Profile early and often
- Caching at multiple layers
- Lazy evaluation where possible
- Benchmark against real projects
- Performance gates in CI

### Risk 3: Incomplete Feature Coverage

**Likelihood:** Low
**Impact:** High

**Mitigation:**
- Create feature matrix checklist
- Test against all known edge cases
- Review legacy swag test cases
- Test with both real projects
- Community testing before final cutover

### Risk 4: Complex Migration

**Likelihood:** Medium
**Impact:** Medium

**Mitigation:**
- Incremental phases (6 phases over 3-4 weeks)
- Backward-compatible adapters
- Deprecation warnings
- Detailed documentation
- Rollback plan for each phase

---

## Success Metrics

### Code Quality Metrics
- **Lines of code reduced:** Target 40% reduction (9,000 → 5,400 lines)
- **Test coverage:** Maintain > 80% coverage
- **Cyclomatic complexity:** < 15 per function
- **Duplication:** 0% duplicated logic across parsers

### Functional Metrics
- **Feature parity:** 100% of existing features supported
- **Bug count:** Zero regressions in integration tests
- **API compatibility:** All existing public APIs work
- **Real project tests:** Both test-project-1 and test-project-2 pass

### Performance Metrics
- **Parse time:** < 2x current performance
- **Memory usage:** < 1.5x current usage
- **Cache hit rate:** > 80% for repeated types
- **Benchmark suite:** All benchmarks within acceptable range

---

## Phase Checklist

### Phase 1: Create Unified Core ✓
- [ ] Create package structure
- [ ] Implement TypeBackend interface
- [ ] Implement ASTBackend
- [ ] Implement TypeResolver
- [ ] Implement FieldProcessor
- [ ] Write unit tests (80%+ coverage)
- [ ] All existing tests pass

### Phase 2: Feature Parity ✓
- [ ] Extended primitives (time.Time, UUID, decimal)
- [ ] fields.StructField[T] extraction
- [ ] Enum detection & inlining
- [ ] Public mode filtering
- [ ] Embedded field merging
- [ ] Array/map/pointer handling
- [ ] Validation constraints
- [ ] Caching layer
- [ ] Integration tests pass

### Phase 3: SchemaBuilder Integration ✓
- [ ] Add UnifiedStructParser to SchemaBuilder
- [ ] Update BuildSchema()
- [ ] Remove fallback code
- [ ] Update tests
- [ ] Performance profiling
- [ ] Side-by-side comparison tests pass

### Phase 4: Orchestrator Integration ✓
- [ ] Update Orchestrator
- [ ] Create ParseFile adapter
- [ ] Migrate public schema generation
- [ ] Remove structparser.Service
- [ ] All Orchestrator tests pass
- [ ] make test-project-1 passes
- [ ] make test-project-2 passes

### Phase 5: CoreStructParser Migration ✓
- [ ] Implement PackagesBackend
- [ ] Port caching logic
- [ ] Port recursive extraction
- [ ] Create BuildAllSchemas adapter
- [ ] Update all callers
- [ ] Add deprecation notices
- [ ] All tests pass

### Phase 6: Cleanup ✓
- [ ] Delete deprecated struct service
- [ ] Delete deprecated field processor
- [ ] Archive/delete CoreStructParser
- [ ] Update documentation
- [ ] Clean up imports
- [ ] Final integration tests
- [ ] Update CLAUDE.md

---

## Critical Files for Implementation

### Top 5 Files (Highest Priority)

1. **`internal/parser/unified/parser.go`** (~400 lines)
   - Core entry point, main API surface
   - Orchestrates all layers
   - Public interface for all callers

2. **`internal/parser/unified/type_resolver.go`** (~600 lines)
   - Heart of the system - handles 80% of complexity
   - Type analysis logic
   - Primitive/enum/struct detection
   - Generic parameter extraction

3. **`internal/parser/unified/backend_ast.go`** (~200 lines)
   - Lightweight AST-only backend
   - Makes system work without go/packages
   - Critical for performance

4. **`internal/parser/unified/field_processor.go`** (~500 lines)
   - Field-level schema generation
   - Tag handling
   - Constraints application
   - Replaces triple duplication

5. **`internal/orchestrator/service.go`** (existing, ~50 line modification)
   - Integration point
   - Wires unified parser into main flow
   - Demonstrates usage pattern

---

## Next Actions

### Immediate (Start Phase 1)
1. Create `internal/parser/unified/` package directory
2. Create `backend.go` with TypeBackend interface
3. Create `type_resolver.go` skeleton with primitive detection
4. Write first tests

### Week 1 Goal
- Complete Phases 1 & 2
- Have unified parser with feature parity
- All tests passing

### Week 2 Goal
- Complete Phases 3 & 4
- SchemaBuilder and Orchestrator using unified parser
- Real projects working

### Week 3 Goal
- Complete Phases 5 & 6
- All deprecated code removed
- Documentation updated
- Ship it! 🚀

---

## Questions to Answer

1. **Should we keep CoreStructParser for a deprecation period?**
   - Recommendation: Yes, keep for 1-2 releases with deprecation warnings

2. **Can we do this incrementally without breaking changes?**
   - Yes - each phase is backward compatible

3. **What if performance is worse?**
   - Caching and lazy evaluation should keep it within 2x
   - ASTBackend will be faster for most cases

4. **How do we ensure feature parity?**
   - Feature matrix checklist
   - Comprehensive test suite
   - Real project validation

5. **What about community impact?**
   - No public API changes during migration
   - Deprecation warnings before removal
   - Clear migration guide

---

## References

- Current implementations analysis: `.agents/final-fix-summary.md`
- Recent fixes: `.agents/change_log.md`
- Schema builder fix: `.agents/schema-builder-fix-summary.md`
- Extended primitives: `.agents/primitive-types-update-summary.md`
