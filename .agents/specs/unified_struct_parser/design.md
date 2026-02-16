# Unified Struct Parser - Design Document

## Overview

This design consolidates three separate struct parsing implementations into ONE canonical parser that handles all struct parsing in core-swag. The design prioritizes simplicity and practicality over abstraction.

**Core Principle:** ONE parser, ONE entry point, handles ALL cases.

**Location:** Enhance/consolidate into `internal/model/struct_builder.go` and related files, OR create new unified location if cleaner.

---

## Architecture Decision

### Keep It Simple

After analyzing the three existing parsers, the design adopts a **single concrete implementation** approach:

1. **No abstraction layers** - No TypeBackend interface, no pluggable backends
2. **Works from AST + registry** - Uses what we already have
3. **Graceful degradation** - Handles complex cases, degrades to simple ones naturally
4. **Single entry point** - All callers use the same API

### Why Not Pluggable Backends?

Analysis shows that:
- AST + registry lookup is sufficient for all known use cases
- go/packages adds heavy dependencies without clear benefit
- Abstraction adds complexity without solving a real problem
- YAGNI (You Aren't Gonna Need It) - build what we need, not what we might need

---

## High-Level Design

```
┌─────────────────────────────────────────────────────────────┐
│                    Single Entry Point                        │
│                  StructParser / StructBuilder                │
│                                                               │
│  ParseStruct(structType, options) → Schema + NestedTypes    │
│  ParseField(field, options) → PropertySchema                │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                   Core Parsing Logic                         │
│                                                               │
│  • Process each field in struct                              │
│  • Parse struct tags (json, public, validate, swaggerignore) │
│  • Resolve field type (primitive, enum, struct, array, map)  │
│  • Handle embedded fields (merge into parent)                │
│  • Handle generics (extract T from StructField[T])           │
│  • Apply validation constraints                              │
│  • Build OpenAPI schema                                      │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                    Supporting Utilities                      │
│                                                               │
│  • TypeResolver: Resolve Go type → OpenAPI type             │
│  • TagParser: Parse struct tags                              │
│  • EnumLookup: Detect and inline enum values                │
│  • PrimitiveMap: Map extended primitives (time.Time, UUID)   │
│  • Cache: Store parsed results by qualified type name        │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                  External Dependencies                       │
│                                                               │
│  • Registry: Look up types by qualified name                 │
│  • AST: Parse source files                                   │
│  • go/ast, go/types (NOT go/packages)                        │
└─────────────────────────────────────────────────────────────┘
```

---

## Components

### 1. Main Parser (StructBuilder / StructParser)

**Responsibilities:**
- Primary entry point for all struct parsing
- Orchestrate field processing
- Handle embedded field merging
- Manage caching
- Return complete schemas + list of nested types

**Key Methods:**
```go
type StructBuilder struct {
    registry     *registry.Service
    enumLookup   EnumLookup
    tagParser    *TagParser
    typeResolver *TypeResolver
    cache        *ParserCache
}

// Main entry points
func (sb *StructBuilder) ParseStruct(
    structType *ast.StructType,
    file *ast.File,
    options ParseOptions,
) (*spec.Schema, []string, error)

func (sb *StructBuilder) ParseField(
    field *ast.Field,
    file *ast.File,
    options ParseOptions,
) (*spec.Schema, bool, error) // schema, required, error

// Options for parsing behavior
type ParseOptions struct {
    Public        bool   // Filter for public:"view|edit" fields
    ForceRequired bool   // Make all fields required
    NamingStrategy string // camelCase, PascalCase, snake_case
}
```

**Logic Flow:**
1. Check cache for already-parsed type
2. Iterate through struct fields
3. For each field:
   - Parse tags
   - Check if field should be skipped (swaggerignore, public mode filtering)
   - If embedded field, recurse and merge properties
   - Otherwise, resolve field type and build property schema
4. Collect nested types that need schemas
5. Cache result
6. Return schema + nested types

### 2. TypeResolver

**Responsibilities:**
- Determine what kind of type a field is (primitive, enum, struct, array, map)
- Handle extended primitives (time.Time, UUID, decimal, json.RawMessage)
- Extract type parameters from generics (StructField[T] → T)
- Use registry to look up struct types
- Create appropriate OpenAPI schema for each type

**Key Logic:**
```go
type TypeResolver struct {
    registry     *registry.Service
    enumLookup   EnumLookup
    primitiveMap map[string]PrimitiveInfo // Extended primitive mappings
}

func (tr *TypeResolver) ResolveType(
    typeExpr ast.Expr,
    file *ast.File,
) (*ResolvedType, error)

type ResolvedType struct {
    Kind        TypeKind              // Primitive, Enum, Struct, Array, Map, etc.
    Schema      *spec.Schema          // For primitives/enums (inline)
    Reference   string                // For structs (#/definitions/TypeName)
    NestedTypes []string              // Struct types that need definitions
    ElementType *ResolvedType         // For arrays
    ValueType   *ResolvedType         // For maps
}

type TypeKind int
const (
    KindPrimitive TypeKind = iota
    KindExtendedPrimitive  // time.Time, UUID, etc.
    KindEnum
    KindStruct
    KindArray
    KindMap
    KindPointer
    KindInterface
    KindGeneric  // StructField[T]
)
```

**Resolution Strategy:**
1. **Unwrap pointers**: `*Type` → `Type`
2. **Check for generics**: `fields.StructField[T]` → extract `T`, resolve recursively
3. **Check primitives**: `string`, `int`, `bool`, etc. → OpenAPI primitive schema
4. **Check extended primitives**: `time.Time` → `{type: string, format: date-time}`
5. **Check enums**: Use enumLookup → inline enum values in schema
6. **Check arrays**: `[]T` → resolve T, return array schema with items
7. **Check maps**: `map[K]V` → resolve V, return object schema with additionalProperties
8. **Check structs**: Look up in registry → return $ref + add to nested types list
9. **Fallback**: Unknown type → `{type: object}` with warning

### 3. Field Processing

**Responsibilities:**
- Extract field name from json tag or field name
- Determine if field is required (validate:"required" tag, or pointer check)
- Apply validation constraints (min, max, minLength, maxLength, pattern)
- Handle swaggerignore
- Handle public mode filtering
- Handle embedded fields (merge into parent)

**Embedded Field Handling:**
```go
func (sb *StructBuilder) processField(field *ast.Field, file *ast.File, options ParseOptions) {
    // Parse tags
    tags := sb.tagParser.ParseTags(field.Tag)

    // Check swaggerignore
    if tags["swaggerignore"] == "true" {
        return // Skip field
    }

    // Check public mode filtering
    if options.Public && !hasPublicTag(tags) {
        return // Skip field
    }

    // Check for embedded field (no field name)
    if field.Names == nil || len(field.Names) == 0 {
        // This is an embedded field
        embeddedSchema := sb.ParseStruct(field.Type, file, options)
        // Merge embeddedSchema.Properties into parent schema
        return
    }

    // Normal field processing
    fieldName := getFieldName(field, tags) // json tag or field name
    required := isRequired(field, tags)

    // Resolve type
    resolvedType := sb.typeResolver.ResolveType(field.Type, file)

    // Build property schema
    propSchema := buildPropertySchema(resolvedType, tags)

    // Apply validation constraints
    applyConstraints(propSchema, tags)
}
```

### 4. Tag Parser

**Reuse existing:** `internal/parser/struct/tag_parser.go`

Parses struct tags into map:
```go
type FieldTags map[string]string

// Example: `json:"name,omitempty" validate:"required" public:"view"`
// → {"json": "name,omitempty", "validate": "required", "public": "view"}
```

### 5. Enum Lookup

**Reuse existing:** `internal/model/enum_lookup.go`

Detects if a type is an enum and retrieves its values:
```go
type EnumLookup interface {
    IsEnum(typeName string) bool
    GetEnumValues(typeName string) []interface{}
}
```

### 6. Cache

**Purpose:** Avoid re-parsing the same struct types

```go
type ParserCache struct {
    mu     sync.RWMutex
    cache  map[string]*CachedResult
}

type CachedResult struct {
    Schema      *spec.Schema
    NestedTypes []string
    ParsedAt    time.Time
}

func (c *ParserCache) Get(qualifiedTypeName string) (*CachedResult, bool)
func (c *ParserCache) Set(qualifiedTypeName string, result *CachedResult)
```

**Cache Key:** Fully qualified type name (e.g., `github.com/user/repo/pkg.TypeName`)

### 7. Extended Primitive Mappings

**Purpose:** Map Go types to OpenAPI types with format

```go
var ExtendedPrimitives = map[string]PrimitiveInfo{
    "time.Time":        {Type: "string", Format: "date-time"},
    "uuid.UUID":        {Type: "string", Format: "uuid"},
    "decimal.Decimal":  {Type: "number"},
    "json.RawMessage":  {Type: "object"},
    // Add more as needed
}

type PrimitiveInfo struct {
    Type   string
    Format string
}
```

---

## Data Models

### ParseOptions

```go
type ParseOptions struct {
    Public        bool   // Filter for public:"view|edit" fields only
    ForceRequired bool   // Make all fields required regardless of tags
    NamingStrategy string // "camelCase", "PascalCase", "snake_case" (default: original)
}
```

### ResolvedType

```go
type ResolvedType struct {
    Kind        TypeKind      // What kind of type this is
    Schema      *spec.Schema  // Inline schema (for primitives, enums)
    Reference   string        // $ref path (for structs)
    NestedTypes []string      // Types that need definitions
    ElementType *ResolvedType // For arrays
    ValueType   *ResolvedType // For maps
}
```

### FieldSchema

```go
type FieldSchema struct {
    Name     string        // JSON field name
    Schema   *spec.Schema  // OpenAPI schema for this field
    Required bool          // Is this field required?
    Nested   []string      // Nested types discovered
}
```

---

## Error Handling

### Error Types

```go
type ParseError struct {
    TypeName   string
    FieldName  string
    FilePath   string
    Line       int
    Reason     string
}

func (e *ParseError) Error() string {
    return fmt.Sprintf("parse error in %s.%s (%s:%d): %s",
        e.TypeName, e.FieldName, e.FilePath, e.Line, e.Reason)
}
```

### Error Scenarios

1. **Type not found in registry**
   - Try basic AST inference
   - Fall back to `{type: object}` with warning
   - Continue processing (don't fail entire parse)

2. **Tag parsing fails**
   - Log warning with field name and tag value
   - Use default values
   - Continue processing

3. **Circular references**
   - Detect via cache check before recursing
   - Use $ref to break cycle
   - Don't infinitely recurse

4. **Generic extraction fails**
   - Log warning
   - Treat as regular type
   - Continue processing

### Warnings vs Errors

- **Warnings:** Log but continue (type not found, tag parse failure)
- **Errors:** Stop processing (nil AST, invalid field structure)

---

## Testing Strategy

### Unit Tests (Per Component)

#### TypeResolver Tests
```go
func TestTypeResolver_PrimitiveTypes(t *testing.T)
func TestTypeResolver_ExtendedPrimitives(t *testing.T)
func TestTypeResolver_EnumDetection(t *testing.T)
func TestTypeResolver_GenericExtraction(t *testing.T)
func TestTypeResolver_Arrays(t *testing.T)
func TestTypeResolver_Maps(t *testing.T)
func TestTypeResolver_NestedStructs(t *testing.T)
func TestTypeResolver_UnknownTypesFallback(t *testing.T)
```

#### StructBuilder Tests
```go
func TestStructBuilder_SimplePrimitives(t *testing.T)
func TestStructBuilder_EmbeddedFields(t *testing.T)
func TestStructBuilder_PublicModeFiltering(t *testing.T)
func TestStructBuilder_SwaggerIgnore(t *testing.T)
func TestStructBuilder_ValidationConstraints(t *testing.T)
func TestStructBuilder_FieldsWrapper(t *testing.T)
func TestStructBuilder_RequiredDetermination(t *testing.T)
func TestStructBuilder_Caching(t *testing.T)
```

#### Cache Tests
```go
func TestCache_GetSet(t *testing.T)
func TestCache_ConcurrentAccess(t *testing.T)
func TestCache_CircularReferenceDetection(t *testing.T)
```

### Integration Tests

#### Real Project Tests
```go
func TestRealProjectIntegration(t *testing.T) {
    // Use actual test projects
    // Verify valid swagger.json generated
    // Check for no missing fields (more fields OK)
}

func TestMakeTestProject1(t *testing.T) {
    // make test-project-1
    // Compare output (allow more complete schemas)
}

func TestMakeTestProject2(t *testing.T) {
    // make test-project-2
    // Compare output (allow more complete schemas)
}
```

#### Comparison Tests
```go
func TestOutputComparison_vs_CoreStructParser(t *testing.T) {
    // Parse same struct with old and new parser
    // Verify new parser has >= fields
}

func TestOutputComparison_vs_StructService(t *testing.T) {
    // Parse same struct with old and new parser
    // Verify new parser has >= fields
}

func TestOutputComparison_vs_SchemaBuilderFallback(t *testing.T) {
    // Parse same struct with old and new parser
    // Verify new parser has >= fields
}
```

### Test Data

**Test Structs (Comprehensive Coverage):**
```go
// 1. ALL Primitive Types
type AllPrimitives struct {
    String    string  `json:"string"`
    Bool      bool    `json:"bool"`
    Int       int     `json:"int"`
    Int8      int8    `json:"int8"`
    Int16     int16   `json:"int16"`
    Int32     int32   `json:"int32"`
    Int64     int64   `json:"int64"`
    Uint      uint    `json:"uint"`
    Uint8     uint8   `json:"uint8"`
    Uint16    uint16  `json:"uint16"`
    Uint32    uint32  `json:"uint32"`
    Uint64    uint64  `json:"uint64"`
    Float32   float32 `json:"float32"`
    Float64   float64 `json:"float64"`
    Byte      byte    `json:"byte"`
    Rune      rune    `json:"rune"`
}

// 2. Pointers to ALL Primitive Types
type AllPointerPrimitives struct {
    String    *string  `json:"string"`
    Bool      *bool    `json:"bool"`
    Int       *int     `json:"int"`
    Int64     *int64   `json:"int64"`
    Float64   *float64 `json:"float64"`
}

// 3. Extended Primitives
type ExtendedPrimitives struct {
    Time       time.Time       `json:"time"`
    UUID       uuid.UUID       `json:"uuid"`
    Decimal    decimal.Decimal `json:"decimal"`
    RawMessage json.RawMessage `json:"raw_message"`
    // Add others as discovered
}

// 4. Pointers to Extended Primitives
type PointerExtendedPrimitives struct {
    Time       *time.Time       `json:"time"`
    UUID       *uuid.UUID       `json:"uuid"`
    Decimal    *decimal.Decimal `json:"decimal"`
    RawMessage *json.RawMessage `json:"raw_message"`
}

// 5. Simple Arrays (primitives)
type SimpleArrays struct {
    Strings  []string  `json:"strings"`
    Ints     []int     `json:"ints"`
    Bools    []bool    `json:"bools"`
    Float64s []float64 `json:"float64s"`
}

// 6. Arrays of Pointers
type ArraysOfPointers struct {
    Strings  []*string  `json:"strings"`
    Ints     []*int     `json:"ints"`
    Bools    []*bool    `json:"bools"`
}

// 7. Nested Arrays
type NestedArrays struct {
    Matrix2D     [][]int           `json:"matrix_2d"`
    Matrix3D     [][][]int         `json:"matrix_3d"`
    StringMatrix [][]string        `json:"string_matrix"`
}

// 8. Arrays of Structs
type ArraysOfStructs struct {
    Users     []User            `json:"users"`
    Addresses []Address         `json:"addresses"`
    Mixed     []*User           `json:"mixed"` // pointer elements
}

// 9. Simple Maps (string keys)
type SimpleMaps struct {
    StringMap map[string]string  `json:"string_map"`
    IntMap    map[string]int     `json:"int_map"`
    BoolMap   map[string]bool    `json:"bool_map"`
}

// 10. Maps with Complex Values
type ComplexMaps struct {
    StructMap   map[string]User      `json:"struct_map"`
    PointerMap  map[string]*User     `json:"pointer_map"`
    ArrayMap    map[string][]string  `json:"array_map"`
    NestedMap   map[string]map[string]int `json:"nested_map"`
}

// 11. Embedded Fields (single level)
type BaseModel struct {
    ID        int       `json:"id"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

type User struct {
    BaseModel           // Embedded
    Name      string    `json:"name"`
    Email     string    `json:"email"`
}

// 12. Multiple Embedded Fields
type Timestamped struct {
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

type Versioned struct {
    Version int `json:"version"`
}

type Document struct {
    Timestamped         // Embedded 1
    Versioned           // Embedded 2
    Title   string      `json:"title"`
    Content string      `json:"content"`
}

// 13. Nested Embedded Fields
type BaseEntity struct {
    ID int `json:"id"`
}

type AuditableEntity struct {
    BaseEntity          // Nested embed level 1
    CreatedAt time.Time `json:"created_at"`
}

type Product struct {
    AuditableEntity     // Nested embed level 2
    Name  string        `json:"name"`
    Price float64       `json:"price"`
}

// 14. Generics - ALL StructField[T] Variations
type GenericFields struct {
    // Primitives
    StringField fields.StructField[string]  `json:"string_field"`
    IntField    fields.StructField[int]     `json:"int_field"`
    BoolField   fields.StructField[bool]    `json:"bool_field"`

    // Pointers to primitives
    PtrString   fields.StructField[*string] `json:"ptr_string"`
    PtrInt      fields.StructField[*int]    `json:"ptr_int"`

    // Extended primitives
    TimeField   fields.StructField[time.Time]       `json:"time_field"`
    UUIDField   fields.StructField[uuid.UUID]       `json:"uuid_field"`
    PtrTime     fields.StructField[*time.Time]      `json:"ptr_time"`

    // Structs (same package)
    UserField   fields.StructField[User]            `json:"user_field"`
    PtrUser     fields.StructField[*User]           `json:"ptr_user"`

    // Structs (other package)
    CrossPkg    fields.StructField[*other_package.Account]       `json:"cross_pkg"`

    // Arrays of primitives
    StringArray fields.StructField[[]string]        `json:"string_array"`
    IntArray    fields.StructField[[]int]           `json:"int_array"`

    // Arrays of structs
    UserArray   fields.StructField[[]User]          `json:"user_array"`
    PtrUserArray fields.StructField[[]*User]        `json:"ptr_user_array"`

    // Arrays of structs (other package)
    CrossArray  fields.StructField[[]*other_package.Account]     `json:"cross_array"`

    // Maps with primitive values
    StringMap   fields.StructField[map[string]string]            `json:"string_map"`
    IntMap      fields.StructField[map[string]int]               `json:"int_map"`

    // Maps with struct values (same package)
    UserMap     fields.StructField[map[string]User]              `json:"user_map"`
    PtrUserMap  fields.StructField[map[string]*User]             `json:"ptr_user_map"`

    // Maps with struct values (other package)
    CrossMap    fields.StructField[map[string]*other_package.Account] `json:"cross_map"`

    // Nested collections
    NestedArray fields.StructField[[][]int]         `json:"nested_array"`
    NestedMap   fields.StructField[map[string]map[string]int] `json:"nested_map"`

    // Complex combinations
    MapOfArrays fields.StructField[map[string][]*User]          `json:"map_of_arrays"`
    ArrayOfMaps fields.StructField[[]map[string]string]         `json:"array_of_maps"`
}

// 15. Enums (string-based)
type Status string
const (
    StatusActive   Status = "active"
    StatusInactive Status = "inactive"
    StatusPending  Status = "pending"
)

type Priority int
const (
    PriorityLow Priority = iota
    PriorityMedium
    PriorityHigh
)

type OrderWithEnums struct {
    Status   Status   `json:"status"`
    Priority Priority `json:"priority"`
}

// 16. Enum Pointers and Arrays
type EnumVariations struct {
    Status       Status    `json:"status"`
    StatusPtr    *Status   `json:"status_ptr"`
    StatusArray  []Status  `json:"status_array"`
    StatusPtrArr []*Status `json:"status_ptr_arr"`
}

// 17. Validation Constraints (all types)
type ValidatedFields struct {
    // String constraints
    Email    string `json:"email" validate:"required,email"`
    Pattern  string `json:"pattern" pattern:"^[A-Z][a-z]+$"`
    MinMax   string `json:"min_max" minLength:"3" maxLength:"50"`

    // Number constraints
    Age      int     `json:"age" validate:"min=0,max=120"`
    Score    float64 `json:"score" validate:"min=0,max=100"`
    Positive int     `json:"positive" validate:"min=1"`

    // Required fields
    Required string  `json:"required" validate:"required"`
    Optional *string `json:"optional"` // Not required (pointer)
}

// 18. Public/Private Mode
type PublicPrivateFields struct {
    PublicView    string `json:"public_view" public:"view"`
    PublicEdit    string `json:"public_edit" public:"edit"`
    PublicBoth    string `json:"public_both" public:"view,edit"`
    PrivateField  string `json:"private_field"`
    InternalField string `json:"-"` // Not in JSON at all
}

// 19. SwaggerIgnore
type MixedVisibility struct {
    Visible1    string `json:"visible1"`
    Hidden      string `json:"hidden" swaggerignore:"true"`
    Visible2    string `json:"visible2"`
    AlsoHidden  string `swaggerignore:"true"` // No json tag
}

// 20. Nested Structs (deep nesting)
type Address struct {
    Street  string `json:"street"`
    City    string `json:"city"`
    Country string `json:"country"`
}

type Contact struct {
    Email   string  `json:"email"`
    Phone   string  `json:"phone"`
    Address Address `json:"address"` // Nested level 1
}

type Person struct {
    Name    string  `json:"name"`
    Contact Contact `json:"contact"` // Nested level 2
}

type Company struct {
    Name       string   `json:"name"`
    CEO        Person   `json:"ceo"`        // Nested level 3
    Employees  []Person `json:"employees"`  // Array of nested
}

// 21. Pointer to Nested Structs
type PointerNested struct {
    User    *User    `json:"user"`
    Address *Address `json:"address"`
    Contact *Contact `json:"contact"`
}

// 22. Circular References (self-referential)
type TreeNode struct {
    Value    string     `json:"value"`
    Children []TreeNode `json:"children"` // Recursive
}

type LinkedListNode struct {
    Value int             `json:"value"`
    Next  *LinkedListNode `json:"next"` // Recursive via pointer
}

// 23. Interface Types
type InterfaceFields struct {
    Any        interface{}            `json:"any"`
    AnyMap     map[string]interface{} `json:"any_map"`
    AnyArray   []interface{}          `json:"any_array"`
}

// 24. Mixed Everything (stress test)
type KitchenSink struct {
    // Embedded
    BaseModel

    // Primitives
    Name    string `json:"name" validate:"required"`
    Age     int    `json:"age" validate:"min=0,max=120"`

    // Extended primitives
    CreatedAt time.Time `json:"created_at"`
    UUID      uuid.UUID `json:"uuid"`

    // Enums
    Status Status `json:"status"`

    // Nested structs
    Address Address `json:"address"`

    // Pointers
    OptionalField *string `json:"optional_field"`

    // Arrays
    Tags        []string  `json:"tags"`
    Scores      []int     `json:"scores"`
    Contacts    []Contact `json:"contacts"`

    // Maps
    Metadata    map[string]string   `json:"metadata"`
    Preferences map[string]int      `json:"preferences"`
    Relations   map[string]*Person  `json:"relations"`

    // Generics
    GenericString fields.StructField[string]          `json:"generic_string"`
    GenericUser   fields.StructField[*User]           `json:"generic_user"`
    GenericArray  fields.StructField[[]string]        `json:"generic_array"`
    GenericMap    fields.StructField[map[string]int]  `json:"generic_map"`

    // Validation
    Email string `json:"email" validate:"required,email"`

    // Public/Private
    PublicInfo  string `json:"public_info" public:"view"`
    PrivateData string `json:"private_data"`

    // Ignored
    Internal string `json:"-"`
    Hidden   string `swaggerignore:"true"`
}
```

---

## Migration Plan

### Phase 1: Enhance Struct Builder

**Goal:** Enhance `internal/model/struct_builder.go` to be the ONE parser with all features

**Approach:** Enhance the existing struct_builder and pull in missing features from the other two parsers.

**What to Pull From Other Parsers:**

From `internal/parser/struct/service.go`:
- Clean tag parsing logic (`internal/parser/struct/tag_parser.go` - already reusable)
- Type resolution utilities (`internal/parser/struct/type_resolver.go`)
- Field filtering logic (public mode, swaggerignore)

From `internal/schema/builder.go` (fallback methods):
- Extended primitive mappings (time.Time, UUID, decimal, json.RawMessage)
- Validation constraint application logic
- Array/map schema building

From `internal/model/struct_field_lookup.go` (CoreStructParser):
- Enum detection integration (already uses EnumLookup interface)
- Caching strategy
- Generic type parameter extraction (StructField[T] → T)

**Tasks:**
1. Add enum detection (use existing EnumLookup interface)
2. Add full StructField[T] generic extraction (pull from CoreStructParser logic)
3. Ensure embedded field merging works correctly (verify and fix if needed)
4. Add caching layer (pull strategy from CoreStructParser)
5. Add all extended primitive mappings (pull from schema builder fallback)
6. Add validation constraint logic (pull from struct service)
7. Comprehensive unit tests for each feature

**Deliverables:**
- Enhanced struct_builder with ALL features
- 75%+ test coverage
- All unit tests pass

**Success:** Can parse any struct type with all features

### Phase 2: Integrate with SchemaBuilder

**Goal:** SchemaBuilder uses the ONE parser, remove fallback

**Tasks:**
1. Update `internal/schema/builder.go` to use enhanced struct_builder
2. Remove fallback methods (`buildFieldSchema`, `getFieldType`)
3. Update tests
4. Compare output (allow more complete schemas)

**Files Modified:**
- `internal/schema/builder.go` - Remove lines 268-432 (fallback logic)
- `internal/schema/builder.go` - Use struct_builder directly

**Deliverables:**
- SchemaBuilder uses ONE parser
- Fallback code removed (~329 lines deleted)
- All schema builder tests pass

**Success:** SchemaBuilder has no duplicate parsing logic

### Phase 3: Integrate with Orchestrator

**Goal:** Orchestrator uses the ONE parser

**Tasks:**
1. Update `internal/orchestrator/service.go` to use enhanced struct_builder
2. Remove dependency on `internal/parser/struct/service.go`
3. Update tests
4. Run real project tests (test-project-1, test-project-2)
5. Verify valid swagger.json output

**Files Modified:**
- `internal/orchestrator/service.go` - Use struct_builder instead of struct service
- Remove `internal/parser/struct/` import

**Deliverables:**
- Orchestrator uses ONE parser
- Real project tests pass
- Valid swagger.json generated

**Success:** Real projects generate correct swagger with ONE parser

### Phase 4: Delete Old Implementations

**Goal:** Remove all duplicate code

**Tasks:**
1. Delete `internal/parser/struct/service.go` (530 lines)
2. Delete `internal/parser/struct/field_processor.go` (464 lines)
3. Verify no remaining references
4. Update CLAUDE.md
5. Clean up imports

**Files Deleted:**
- `internal/parser/struct/service.go`
- `internal/parser/struct/field_processor.go`
- `internal/parser/struct/` directory (keep tag_parser.go if reusable elsewhere)

**Potentially Archive (not delete immediately):**
- `internal/model/struct_field_lookup.go` (if replaced by enhanced struct_builder)
- `internal/model/struct_field.go` (if replaced by enhanced struct_builder)

**Deliverables:**
- ~1,000+ lines of duplicate code removed
- All tests still pass
- Documentation updated

**Success:** Only ONE struct parser exists in codebase

---

## Design Decisions & Rationale

### Decision 1: No Abstraction Layer

**Rationale:**
- YAGNI - Don't build what we don't need
- AST + registry is sufficient
- Abstraction adds complexity without benefit
- Simpler code is easier to maintain

### Decision 2: Enhance Existing vs New Implementation

**Rationale:**
- `internal/model/struct_builder.go` was designed for this purpose
- Already has some infrastructure
- Less disruption than creating new package
- Can build incrementally

**Alternative:** If struct_builder is too coupled to other code, create clean new implementation.

### Decision 3: Allow More Complete Output

**Rationale:**
- Old parsers had bugs (missing fields, incomplete schemas)
- MORE complete schemas = better documentation
- Backward compatibility = functionally equivalent, not byte-identical
- Users benefit from more accurate schemas

### Decision 4: Use Registry + AST, Not go/packages

**Rationale:**
- Registry already has type lookup
- AST provides necessary structure
- go/packages is heavy dependency
- No clear benefit observed from go/packages in current code

### Decision 5: Cache by Qualified Type Name

**Rationale:**
- Prevents re-parsing same types
- Key insight: same qualified name = same type
- Performance boost for large projects with type reuse
- Simple map-based cache sufficient

---

## Performance Considerations

### Caching Strategy

**What to cache:**
- Parsed struct schemas (by qualified type name)
- Enum values (already in EnumLookup)
- Resolved types

**Cache invalidation:**
- Not needed within single parse run
- Fresh cache per orchestrator invocation

**Concurrency:**
- Use sync.RWMutex for cache access
- Most operations are reads (cache hits)

### Optimization Opportunities

1. **Early returns:** Skip field immediately if swaggerignore
2. **Lazy evaluation:** Only resolve nested types when needed
3. **Reuse parsed tags:** Don't re-parse same tags
4. **Registry lookups:** Minimize registry queries

### Expected Performance

**Target:** Within 2x of fastest current parser

**Measurement:**
- Benchmark simple struct (10-20 fields): < 100ms
- Benchmark complex struct (100+ fields): < 500ms
- Benchmark real projects: comparable to current

---

## Design Decisions

### Decision: Use struct_builder as the ONE parser

**Choice:** Enhance `internal/model/struct_builder.go` to be the canonical parser.

**Rationale:**
- Was designed for this purpose
- Already has some infrastructure
- Central location makes sense
- Less disruption than creating new package

### Decision: Handle registry lookup failures gracefully

**Approach:**
- Try basic AST inference (check if it's a struct type in current file)
- Fall back to `{type: object}` with warning
- Don't fail entire parse
- Log warning for debugging

### Decision: Deprecate but keep old parsers temporarily

**Approach:**
- Keep old implementations for 1-2 phases during migration
- Mark as deprecated
- Delete only after all integration complete and tested
- Provides rollback safety net

---

## Summary

**Core Design:**
- ONE parser: `internal/model/struct_builder.go`
- Works from AST + registry
- Handles ALL features (primitives, enums, generics, embedded, validation)
- Graceful degradation for unknown types
- Simple, no unnecessary abstraction

**Migration:**
1. Enhance struct_builder with all features from the other two parsers
2. Update SchemaBuilder to use enhanced struct_builder
3. Update Orchestrator to use enhanced struct_builder
4. Delete old implementations (struct service and schema builder fallback)

**Expected Outcome:**
- ~35% code reduction
- ONE source of truth
- More complete/correct schemas
- Easier maintenance

**Next Steps:**
- Approve design
- Create detailed task list
- Begin Phase 1 implementation
