# Centralize Type Registry Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Consolidate duplicated custom type mappings from 4 files into a single `internal/typeregistry/` package.

**Architecture:** New package `internal/typeregistry/` with two files: `registry.go` (flat map for simple custom types) and `fields.go` (fields wrapper helper with generic type extraction). Four consumer files delegate to this package instead of maintaining their own hardcoded maps/switches.

**Tech Stack:** Go, `github.com/go-openapi/spec`

**Spec:** `docs/superpowers/specs/2026-03-20-centralize-type-registry-design.md`

---

## Chunk 0: Capture Baselines

### Task 0: Save pre-refactor swagger output for diffing

- [ ] **Step 1: Generate and save baseline outputs**

```bash
make test-project-1
cp testing/test-project-1/swagger.json testing/test-project-1/swagger.json.bak
make test-project-2
cp testing/test-project-2/swagger.json testing/test-project-2/swagger.json.bak
```

These `.bak` files will be diffed against post-refactor output in Task 7.

---

## Chunk 1: Build the Type Registry Package

### Task 1: Create `internal/typeregistry/registry.go`

**Files:**
- Create: `internal/typeregistry/registry.go`
- Test: `internal/typeregistry/registry_test.go`

- [ ] **Step 1: Write failing tests for Lookup, IsExtendedPrimitive, and ToSchema**

Create `internal/typeregistry/registry_test.go`:

```go
package typeregistry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLookup_KnownTypes(t *testing.T) {
	tests := []struct {
		typeName   string
		wantType   string
		wantFormat string
	}{
		{"time.Time", "string", "date-time"},
		{"*time.Time", "string", "date-time"},
		{"types.UUID", "string", "uuid"},
		{"*types.UUID", "string", "uuid"},
		{"uuid.UUID", "string", "uuid"},
		{"github.com/google/uuid.UUID", "string", "uuid"},
		{"github.com/griffnb/core/lib/types.UUID", "string", "uuid"},
		{"types.URN", "string", "uri"},
		{"github.com/griffnb/core/lib/types.URN", "string", "uri"},
		{"decimal.Decimal", "number", ""},
		{"*decimal.Decimal", "number", ""},
		{"github.com/shopspring/decimal.Decimal", "number", ""},
		{"json.RawMessage", "object", ""},
		{"encoding/json.RawMessage", "object", ""},
		{"[]byte", "string", "byte"},
		{"[]uint8", "string", "byte"},
	}
	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			entry, ok := Lookup(tt.typeName)
			assert.True(t, ok, "expected %s to be found", tt.typeName)
			assert.Equal(t, tt.wantType, entry.SchemaType)
			assert.Equal(t, tt.wantFormat, entry.Format)
		})
	}
}

func TestLookup_UnknownTypes(t *testing.T) {
	_, ok := Lookup("MyCustomStruct")
	assert.False(t, ok)

	_, ok = Lookup("account.Account")
	assert.False(t, ok)
}

func TestIsExtendedPrimitive(t *testing.T) {
	assert.True(t, IsExtendedPrimitive("types.UUID"))
	assert.True(t, IsExtendedPrimitive("*time.Time"))
	assert.True(t, IsExtendedPrimitive("[]byte"))
	assert.False(t, IsExtendedPrimitive("account.Account"))
	assert.False(t, IsExtendedPrimitive("string")) // Go primitives are NOT extended primitives
}

func TestToSchema(t *testing.T) {
	schema := ToSchema("types.UUID")
	assert.Equal(t, []string{"string"}, schema.Type)
	assert.Equal(t, "uuid", schema.Format)

	schema = ToSchema("*decimal.Decimal")
	assert.Equal(t, []string{"number"}, schema.Type)
	assert.Equal(t, "", schema.Format)

	schema = ToSchema("[]byte")
	assert.Equal(t, []string{"string"}, schema.Type)
	assert.Equal(t, "byte", schema.Format)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/typeregistry/ -v -run TestLookup`
Expected: FAIL — package does not exist

- [ ] **Step 3: Implement `registry.go`**

Create `internal/typeregistry/registry.go`:

```go
// Package typeregistry centralizes custom type-to-OpenAPI mappings.
// All custom external types (UUID, Decimal, Time, URN, etc.) are registered
// here so consumers don't need to maintain their own hardcoded maps.
package typeregistry

import (
	"strings"

	"github.com/go-openapi/spec"
)

// TypeEntry maps a custom Go type to its OpenAPI schema type and format.
type TypeEntry struct {
	SchemaType string // "string", "number", "integer", "boolean", "object"
	Format     string // "uuid", "date-time", "uri", "byte", ""
}

// registry is the central map of custom types to their OpenAPI representations.
// Go primitives (int, string, bool, etc.) are NOT included here — only extended
// types that need special OpenAPI mapping.
var registry = map[string]TypeEntry{
	// Time
	"time.Time": {SchemaType: "string", Format: "date-time"},

	// UUID variants
	"types.UUID":                             {SchemaType: "string", Format: "uuid"},
	"uuid.UUID":                              {SchemaType: "string", Format: "uuid"},
	"github.com/google/uuid.UUID":            {SchemaType: "string", Format: "uuid"},
	"github.com/griffnb/core/lib/types.UUID": {SchemaType: "string", Format: "uuid"},

	// URN variants
	"types.URN":                             {SchemaType: "string", Format: "uri"},
	"github.com/griffnb/core/lib/types.URN": {SchemaType: "string", Format: "uri"},

	// Decimal variants
	"decimal.Decimal":                        {SchemaType: "number", Format: ""},
	"github.com/shopspring/decimal.Decimal":  {SchemaType: "number", Format: ""},

	// JSON
	"json.RawMessage":          {SchemaType: "object", Format: ""},
	"encoding/json.RawMessage": {SchemaType: "object", Format: ""},

	// Byte arrays
	"[]byte":  {SchemaType: "string", Format: "byte"},
	"[]uint8": {SchemaType: "string", Format: "byte"},
}

// Lookup returns the TypeEntry for a custom type. Strips leading `*` before matching.
// Returns false if the type is not a registered custom type.
func Lookup(typeName string) (TypeEntry, bool) {
	clean := strings.TrimPrefix(typeName, "*")
	entry, ok := registry[clean]
	return entry, ok
}

// IsExtendedPrimitive returns true if the type is a registered custom type
// that should be treated as a primitive in OpenAPI (not a model/$ref).
// Does NOT include basic Go primitives — only extended types like UUID, Time, Decimal.
func IsExtendedPrimitive(typeName string) bool {
	_, ok := Lookup(typeName)
	return ok
}

// ToSchema builds an OpenAPI spec.Schema for a registered custom type.
// Returns nil if the type is not registered.
func ToSchema(typeName string) *spec.Schema {
	entry, ok := Lookup(typeName)
	if !ok {
		return nil
	}
	schema := &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type: []string{entry.SchemaType},
		},
	}
	if entry.Format != "" {
		schema.Format = entry.Format
	}
	return schema
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/typeregistry/ -v -run TestLookup && go test ./internal/typeregistry/ -v -run TestIsExtendedPrimitive && go test ./internal/typeregistry/ -v -run TestToSchema`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add internal/typeregistry/registry.go internal/typeregistry/registry_test.go
git commit -m "feat: add typeregistry package with central custom type map"
```

---

### Task 2: Create `internal/typeregistry/fields.go`

**Files:**
- Create: `internal/typeregistry/fields.go`
- Test: `internal/typeregistry/fields_test.go`

- [ ] **Step 1: Write failing tests for ResolveFieldsWrapper and IsFieldsWrapper**

Create `internal/typeregistry/fields_test.go`:

```go
package typeregistry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsFieldsWrapper(t *testing.T) {
	assert.True(t, IsFieldsWrapper("fields.StringField"))
	assert.True(t, IsFieldsWrapper("fields.IntConstantField[constants.Role]"))
	assert.True(t, IsFieldsWrapper("*fields.UUIDField"))
	assert.False(t, IsFieldsWrapper("string"))
	assert.False(t, IsFieldsWrapper("types.UUID"))
}

func TestResolveFieldsWrapper_SimpleTypes(t *testing.T) {
	tests := []struct {
		typeName   string
		wantType   string
		wantFormat string
	}{
		{"fields.StringField", "string", ""},
		{"fields.IntField", "integer", ""},
		{"fields.DecimalField", "integer", ""},
		{"fields.UUIDField", "string", "uuid"},
		{"fields.BoolField", "boolean", ""},
		{"fields.FloatField", "number", ""},
		{"fields.TimeField", "string", "date-time"},
	}
	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			result, ok := ResolveFieldsWrapper(tt.typeName)
			assert.True(t, ok)
			assert.NotNil(t, result.Schema)
			assert.Equal(t, tt.wantType, result.Schema.Type[0])
			assert.Equal(t, tt.wantFormat, result.Schema.Format)
			assert.Empty(t, result.InnerType)
			assert.False(t, result.IsEnum)
		})
	}
}

func TestResolveFieldsWrapper_ConstantFields(t *testing.T) {
	result, ok := ResolveFieldsWrapper("fields.IntConstantField[constants.Role]")
	assert.True(t, ok)
	assert.Nil(t, result.Schema)
	assert.Equal(t, "constants.Role", result.InnerType)
	assert.True(t, result.IsEnum)
	assert.Equal(t, "integer", result.FallbackSchemaType)

	result, ok = ResolveFieldsWrapper("fields.StringConstantField[constants.Status]")
	assert.True(t, ok)
	assert.Nil(t, result.Schema)
	assert.Equal(t, "constants.Status", result.InnerType)
	assert.True(t, result.IsEnum)
	assert.Equal(t, "string", result.FallbackSchemaType)
}

func TestResolveFieldsWrapper_StructField(t *testing.T) {
	result, ok := ResolveFieldsWrapper("fields.StructField[account.Address]")
	assert.True(t, ok)
	assert.Nil(t, result.Schema)
	assert.Equal(t, "account.Address", result.InnerType)
	assert.False(t, result.IsEnum)
}

func TestResolveFieldsWrapper_UnknownFieldType(t *testing.T) {
	result, ok := ResolveFieldsWrapper("fields.SomeFutureField")
	assert.True(t, ok)
	assert.NotNil(t, result.Schema)
	assert.Equal(t, "string", result.Schema.Type[0]) // default fallback
}

func TestResolveFieldsWrapper_NotFieldsType(t *testing.T) {
	_, ok := ResolveFieldsWrapper("types.UUID")
	assert.False(t, ok)
}

func TestExtractConstantFieldEnumType(t *testing.T) {
	assert.Equal(t, "constants.Role", ExtractConstantFieldEnumType("*fields.IntConstantField[constants.Role]"))
	assert.Equal(t, "constants.Status", ExtractConstantFieldEnumType("fields.StringConstantField[constants.Status]"))
	assert.Equal(t, "github.com/foo/constants.Role", ExtractConstantFieldEnumType("fields.IntConstantField[github.com/foo/constants.Role]"))
	assert.Equal(t, "", ExtractConstantFieldEnumType("fields.StringField"))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/typeregistry/ -v -run TestIsFieldsWrapper`
Expected: FAIL — function does not exist

- [ ] **Step 3: Implement `fields.go`**

Create `internal/typeregistry/fields.go`:

```go
package typeregistry

import (
	"strings"

	"github.com/go-openapi/spec"
)

// FieldsResult holds the resolution result for a fields package wrapper type.
type FieldsResult struct {
	// Schema is the resolved OpenAPI schema. Nil when InnerType is set
	// (caller must resolve the inner type themselves).
	Schema *spec.Schema

	// InnerType is the extracted generic type parameter (e.g., "constants.Role"
	// from "fields.IntConstantField[constants.Role]"). Empty for non-generic wrappers.
	InnerType string

	// IsEnum is true for IntConstantField[T] and StringConstantField[T].
	IsEnum bool

	// FallbackSchemaType is the schema type to use if enum lookup fails
	// (e.g., "integer" for IntConstantField, "string" for StringConstantField).
	// Only set when IsEnum is true.
	FallbackSchemaType string
}

// IsFieldsWrapper returns true if the type is a fields package wrapper type.
func IsFieldsWrapper(typeName string) bool {
	clean := strings.TrimPrefix(typeName, "*")
	return strings.HasPrefix(clean, "fields.") || strings.Contains(clean, "/fields.")
}

// ResolveFieldsWrapper resolves a fields package wrapper type to its OpenAPI schema.
// Returns false if the type is not a fields wrapper.
//
// For simple wrappers (StringField, IntField, etc.), returns a concrete Schema.
// For generic wrappers (IntConstantField[T], StructField[T]), returns InnerType
// with a nil Schema — the caller must resolve the inner type.
// For unknown fields wrappers, returns a default string schema.
func ResolveFieldsWrapper(typeName string) (*FieldsResult, bool) {
	if !IsFieldsWrapper(typeName) {
		return nil, false
	}

	// Check for constant field types with enum parameters first
	if strings.Contains(typeName, "IntConstantField[") {
		innerType := ExtractConstantFieldEnumType(typeName)
		return &FieldsResult{
			InnerType:          innerType,
			IsEnum:             true,
			FallbackSchemaType: "integer",
		}, true
	}
	if strings.Contains(typeName, "StringConstantField[") {
		innerType := ExtractConstantFieldEnumType(typeName)
		return &FieldsResult{
			InnerType:          innerType,
			IsEnum:             true,
			FallbackSchemaType: "string",
		}, true
	}

	// Check for StructField[T] — caller resolves inner type recursively
	if strings.Contains(typeName, "StructField[") {
		innerType := ExtractConstantFieldEnumType(typeName) // same bracket extraction
		return &FieldsResult{
			InnerType: innerType,
		}, true
	}

	// Simple wrapper types — return concrete schema
	if strings.Contains(typeName, "UUIDField") {
		return &FieldsResult{
			Schema: &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"string"}, Format: "uuid"}},
		}, true
	}
	if strings.Contains(typeName, "StringField") {
		return &FieldsResult{
			Schema: &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"string"}}},
		}, true
	}
	if strings.Contains(typeName, "IntField") || strings.Contains(typeName, "DecimalField") {
		return &FieldsResult{
			Schema: &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"integer"}}},
		}, true
	}
	if strings.Contains(typeName, "BoolField") {
		return &FieldsResult{
			Schema: &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"boolean"}}},
		}, true
	}
	if strings.Contains(typeName, "FloatField") {
		return &FieldsResult{
			Schema: &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"number"}}},
		}, true
	}
	if strings.Contains(typeName, "TimeField") {
		return &FieldsResult{
			Schema: &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"string"}, Format: "date-time"}},
		}, true
	}

	// Unknown fields wrapper — default to string
	return &FieldsResult{
		Schema: &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"string"}}},
	}, true
}

// ExtractConstantFieldEnumType extracts the type parameter from a generic fields type.
// e.g., "*fields.IntConstantField[constants.Role]" -> "constants.Role"
// Returns empty string if the type has no brackets.
func ExtractConstantFieldEnumType(typeStr string) string {
	if !strings.Contains(typeStr, "[") {
		return ""
	}
	start := strings.Index(typeStr, "[")
	end := strings.LastIndex(typeStr, "]")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return typeStr[start+1 : end]
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/typeregistry/ -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add internal/typeregistry/fields.go internal/typeregistry/fields_test.go
git commit -m "feat: add fields wrapper resolution to typeregistry"
```

---

## Chunk 2: Refactor Consumer Files

### Task 3: Refactor `internal/domain/utils.go`

**Files:**
- Modify: `internal/domain/utils.go:74-99` (`IsExtendedPrimitiveType`)
- Modify: `internal/domain/utils.go:128-160` (`TransToValidPrimitiveSchema`)

- [ ] **Step 1: Run existing tests to establish baseline**

Run: `go test ./internal/domain/ -v`
Expected: ALL PASS (baseline)

- [ ] **Step 2: Refactor `IsExtendedPrimitiveType` to delegate to typeregistry**

In `internal/domain/utils.go`, replace the `IsExtendedPrimitiveType` function body:

```go
func IsExtendedPrimitiveType(typeName string) bool {
	// Strip pointer prefix for checking
	cleanType := strings.TrimPrefix(typeName, "*")

	// Check basic Go primitives first
	if IsGolangPrimitiveType(cleanType) {
		return true
	}

	// Check extended primitives via centralized registry
	return typeregistry.IsExtendedPrimitive(typeName)
}
```

Add import: `"github.com/griffnb/core-swag/internal/typeregistry"`

- [ ] **Step 3: Refactor `TransToValidPrimitiveSchema` to delegate to typeregistry**

Replace the extended primitives section (lines 147-158) with a typeregistry delegation. Keep Go primitive handling inline:

```go
func TransToValidPrimitiveSchema(typeName string) *spec.Schema {
	// Strip pointer prefix for processing
	cleanType := strings.TrimPrefix(typeName, "*")

	switch cleanType {
	case "int", "uint":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{INTEGER}}}
	case "uint8", "int8", "uint16", "int16", "byte", "int32", "uint32", "rune":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{INTEGER}, Format: "int32"}}
	case "uint64", "int64":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{INTEGER}, Format: "int64"}}
	case "float32":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{NUMBER}, Format: "float"}}
	case "float64":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{NUMBER}, Format: "double"}}
	case "bool":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{BOOLEAN}}}
	case "string":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{STRING}}}
	}

	// Check extended primitives via centralized registry
	if schema := typeregistry.ToSchema(typeName); schema != nil {
		return schema
	}

	return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{typeName}}}
}
```

- [ ] **Step 4: Run domain tests**

Run: `go test ./internal/domain/ -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add internal/domain/utils.go
git commit -m "refactor: domain/utils delegates to typeregistry for custom types"
```

---

### Task 4: Refactor `internal/model/struct_field.go`

**Files:**
- Modify: `internal/model/struct_field.go:87-111` (`IsPrimitive`)
- Modify: `internal/model/struct_field.go:1064-1096` (`primitiveTypeToSchema`)
- Modify: `internal/model/struct_field.go:947-1009` (`getPrimitiveSchemaForFieldType`)
- Modify: `internal/model/struct_field.go:125-129` (`IsFieldsWrapper`)
- Remove: `internal/model/struct_field.go:1012-1022` (`extractConstantFieldEnumType` — moved to typeregistry)

- [ ] **Step 1: Run existing model tests to establish baseline**

Run: `go test ./internal/model/ -v`
Expected: ALL PASS (baseline)

- [ ] **Step 2: Refactor `IsPrimitive` to use typeregistry.Lookup**

Replace the `IsPrimitive` method. Reuse `domain.IsGolangPrimitiveType()` to avoid duplicating the Go primitives map:

```go
func (this *StructField) IsPrimitive() bool {
	typeStr := this.EffectiveTypeString()
	// Strip pointer for Go primitive check
	clean := strings.TrimPrefix(typeStr, "*")

	// Check basic Go primitives via domain package
	if domain.IsGolangPrimitiveType(clean) {
		return true
	}

	// Check extended primitives via centralized registry
	_, ok := typeregistry.Lookup(typeStr)
	return ok
}
```

Add imports: `"github.com/griffnb/core-swag/internal/typeregistry"` and `"github.com/griffnb/core-swag/internal/domain"`

- [ ] **Step 3: Refactor `primitiveTypeToSchema` to use typeregistry.ToSchema**

Replace the extended primitives section (cases after float64):

```go
func primitiveTypeToSchema(typeStr string) *spec.Schema {
	// Strip pointer for Go primitive check
	clean := strings.TrimPrefix(typeStr, "*")

	switch clean {
	case "string":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"string"}}}
	case "bool":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"boolean"}}}
	case "int", "uint":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"integer"}}}
	case "int8", "uint8", "int16", "uint16", "int32", "uint32", "byte", "rune":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"integer"}, Format: "int32"}}
	case "int64", "uint64":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"integer"}, Format: "int64"}}
	case "float32":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"number"}, Format: "float"}}
	case "float64":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"number"}, Format: "double"}}
	}

	// Check extended primitives via centralized registry
	if schema := typeregistry.ToSchema(typeStr); schema != nil {
		return schema
	}

	return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{typeStr}}}
}
```

- [ ] **Step 4: Refactor `IsFieldsWrapper` to use typeregistry**

```go
func (this *StructField) IsFieldsWrapper() bool {
	return typeregistry.IsFieldsWrapper(this.EffectiveTypeString())
}
```

- [ ] **Step 5: Refactor `getPrimitiveSchemaForFieldType` to use typeregistry**

Replace the function body. The enum resolution logic (calling `enumLookup.GetEnumsForType`) stays here since it depends on the caller's context. Only the type identification delegates to typeregistry:

```go
func getPrimitiveSchemaForFieldType(typeStr string, originalTypeStr string, enumLookup TypeEnumLookup) (*spec.Schema, []string, error) {
	result, ok := typeregistry.ResolveFieldsWrapper(typeStr)
	if !ok {
		// Not a fields wrapper — fallback to string (matches previous default)
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"string"}}}, nil, nil
	}

	// Simple wrapper types — return concrete schema directly
	if result.Schema != nil {
		return result.Schema, nil, nil
	}

	// Enum constant fields — try to resolve enum via lookup
	if result.IsEnum && result.InnerType != "" {
		normalizedEnum := normalizeTypeName(result.InnerType)
		// Try full path from originalTypeStr for accurate extraction
		fullEnumType := typeregistry.ExtractConstantFieldEnumType(originalTypeStr)
		if fullEnumType == "" {
			fullEnumType = result.InnerType
		}
		if enumLookup != nil {
			enums, err := enumLookup.GetEnumsForType(normalizedEnum, nil)
			if err == nil && len(enums) > 0 {
				refName := resolveRefName(normalizedEnum, fullEnumType)
				schema := spec.RefSchema("#/definitions/" + refName)
				if strings.Contains(fullEnumType, "/") {
					return schema, []string{fullEnumType}, nil
				}
				return schema, []string{refName}, nil
			}
		}
		// Fallback to base type if enum lookup fails
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{result.FallbackSchemaType}}}, nil, nil
	}

	// StructField[T] — return nil schema, caller resolves via GenericTypeArg path
	// This case is actually handled by IsGeneric() in BuildSchema before we get here,
	// but included for completeness.
	if result.InnerType != "" {
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"string"}}}, nil, nil
	}

	return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"string"}}}, nil, nil
}
```

- [ ] **Step 6: Remove the now-unused `extractConstantFieldEnumType` function**

Delete the `extractConstantFieldEnumType` function (lines 1012-1022) from `struct_field.go`. Update `ConstantFieldEnumType` method to use `typeregistry.ExtractConstantFieldEnumType`:

```go
func (this *StructField) ConstantFieldEnumType() string {
	return typeregistry.ExtractConstantFieldEnumType(this.EffectiveTypeString())
}
```

- [ ] **Step 7: Run model tests**

Run: `go test ./internal/model/ -v`
Expected: ALL PASS

- [ ] **Step 8: Commit**

```bash
git add internal/model/struct_field.go
git commit -m "refactor: model/struct_field delegates to typeregistry"
```

---

### Task 5: Refactor `internal/parser/route/response.go`

**Files:**
- Modify: `internal/parser/route/response.go:200-226` (`convertTypeToSchemaType`)

- [ ] **Step 1: Run existing route parser tests to establish baseline**

Run: `go test ./internal/parser/route/ -v`
Expected: ALL PASS (baseline)

- [ ] **Step 2: Refactor `convertTypeToSchemaType` to use typeregistry**

Replace the function:

```go
func convertTypeToSchemaType(dataType string) string {
	// Strip pointer prefix for processing
	cleanType := strings.TrimPrefix(dataType, "*")

	switch cleanType {
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "byte", "rune":
		return "integer"
	case "float32", "float64":
		return "number"
	case "bool":
		return "boolean"
	case "string", "[]byte", "[]uint8":
		return "string"
	}

	// Check extended primitives via centralized registry
	if entry, ok := typeregistry.Lookup(dataType); ok {
		return entry.SchemaType
	}

	// For custom types, treat as object
	return "object"
}
```

Add import: `"github.com/griffnb/core-swag/internal/typeregistry"`

- [ ] **Step 3: Run route parser tests**

Run: `go test ./internal/parser/route/ -v`
Expected: ALL PASS

- [ ] **Step 4: Commit**

```bash
git add internal/parser/route/response.go
git commit -m "refactor: route/response delegates to typeregistry for type conversion"
```

---

### Task 6: Refactor `internal/parser/route/schema.go`

**Files:**
- Modify: `internal/parser/route/schema.go:15-75` (`isModelType`)

- [ ] **Step 1: Refactor `isModelType` to use typeregistry**

Replace the function. Keep the special keywords (`any`, `interface{}`, `object`, `array`, `file`) locally since they are NOT extended primitives:

```go
func isModelType(typeName string) bool {
	// Strip pointer prefix for checking
	cleanType := strings.TrimPrefix(typeName, "*")

	// Basic Go primitive types
	primitives := map[string]bool{
		"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
		"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
		"float32": true, "float64": true, "bool": true, "string": true,
		"byte": true, "rune": true,
		"any": true, "interface{}": true, // Go wildcard types
		"object": true, "array": true, "file": true, // OpenAPI keywords
	}

	if primitives[cleanType] {
		return false
	}

	// Check extended primitives via centralized registry
	if typeregistry.IsExtendedPrimitive(typeName) {
		return false
	}

	// If it contains a dot, it's likely a qualified type (package.Type)
	// We already checked extended primitives above, so this is a real model
	if strings.Contains(cleanType, ".") {
		return true
	}

	// Anything else that's not a primitive is likely a custom type
	return true
}
```

Add import: `"github.com/griffnb/core-swag/internal/typeregistry"` (the `route` package import section)

- [ ] **Step 2: Run route parser tests**

Run: `go test ./internal/parser/route/ -v`
Expected: ALL PASS

- [ ] **Step 3: Commit**

```bash
git add internal/parser/route/schema.go
git commit -m "refactor: route/schema delegates to typeregistry for model type detection"
```

---

## Chunk 3: Integration Validation

### Task 7: Run full integration tests

**Files:** None (validation only)

- [ ] **Step 1: Run all unit tests**

Run: `go test ./... -v`
Expected: ALL PASS

- [ ] **Step 2: Run integration test project 1**

Run: `make test-project-1`
Expected: Generates `/Users/griffnb/projects/core-swag/testing/test-project-1/swagger.json` with identical output to pre-refactor

- [ ] **Step 3: Run integration test project 2**

Run: `make test-project-2`
Expected: Generates `/Users/griffnb/projects/core-swag/testing/test-project-2/swagger.json` with identical output to pre-refactor

- [ ] **Step 4: Verify no behavioral changes**

Before starting this task, save copies of both swagger.json outputs. After refactoring, diff them:

```bash
diff testing/test-project-1/swagger.json testing/test-project-1/swagger.json.bak
diff testing/test-project-2/swagger.json testing/test-project-2/swagger.json.bak
```

Expected: No differences (or only whitespace/ordering changes)

- [ ] **Step 5: Final commit if any cleanup needed**

```bash
git commit -m "refactor: complete type registry centralization"
```
