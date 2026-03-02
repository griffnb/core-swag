# StructField Method Consolidation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Move all standalone type analysis functions onto `*StructField` as methods so every field is self-describing and there's one code path for everything about a field.

**Architecture:** This is a pure refactor — no behavior changes. New methods are added alongside old functions first, callers are switched over, then old functions are removed. Existing tests validate no regressions throughout.

**Tech Stack:** Go, `go/types`, `github.com/go-openapi/spec`, `testify`

**Key files:**
- `internal/model/struct_field.go` (722 lines) — StructField type + all helpers
- `internal/model/struct_field_lookup.go` (905 lines) — CoreStructParser + extraction
- `internal/model/struct_field_test.go` (577 lines) — unit tests
- `internal/model/struct_builder_test.go` (1045 lines) — builder tests
- `internal/model/struct_builder.go` (75 lines) — StructBuilder (calls ToSpecSchema)

**Critical verification commands:**
- Unit tests: `go test ./internal/model/ -v -run TestToSpecSchema`
- All model tests: `go test ./internal/model/ -v`
- Integration: `go test ./testing/ -v -run TestRealProjectIntegration -timeout 120s`
- Full project 1: `make test-project-1`
- Full project 2: `make test-project-2`

---

### Task 1: Add EffectiveTypeString() method

**Files:**
- Modify: `internal/model/struct_field.go` (add method after GetTags at line ~38)
- Test: `internal/model/struct_field_test.go`

**Step 1: Write the failing test**

Add to `internal/model/struct_field_test.go`:

```go
func TestEffectiveTypeString(t *testing.T) {
	tests := []struct {
		name  string
		field *StructField
		want  string
	}{
		{
			name:  "uses TypeString when set",
			field: &StructField{TypeString: "account.Properties"},
			want:  "account.Properties",
		},
		{
			name:  "empty TypeString with nil Type returns empty",
			field: &StructField{},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.field.EffectiveTypeString()
			assert.Equal(t, tt.want, got)
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/model/ -v -run TestEffectiveTypeString`
Expected: FAIL — `EffectiveTypeString` not defined

**Step 3: Write minimal implementation**

Add to `internal/model/struct_field.go` after `GetTags()`:

```go
// EffectiveTypeString returns the resolved type string for this field.
// Uses TypeString if set, falls back to Type.String() if Type is available.
func (this *StructField) EffectiveTypeString() string {
	if this.TypeString != "" {
		return this.TypeString
	}
	if this.Type != nil {
		return this.Type.String()
	}
	return ""
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/model/ -v -run TestEffectiveTypeString`
Expected: PASS

**Step 5: Run all existing tests to verify no regressions**

Run: `go test ./internal/model/ -v`
Expected: All PASS (new method added alongside, nothing changed)

**Step 6: Commit**

```bash
git add internal/model/struct_field.go internal/model/struct_field_test.go
git commit -m "refactor: add EffectiveTypeString method to StructField"
```

---

### Task 2: Add IsGeneric() and GenericTypeArg() methods

These replace 3 different generic detection patterns and the `extractGenericTypeParameter` / `extractTypeParameter` standalone functions.

**Files:**
- Modify: `internal/model/struct_field.go`
- Test: `internal/model/struct_field_test.go`

**Step 1: Write the failing tests**

```go
func TestIsGeneric(t *testing.T) {
	tests := []struct {
		name  string
		field *StructField
		want  bool
	}{
		{
			name:  "StructField generic",
			field: &StructField{TypeString: "fields.StructField[*User]"},
			want:  true,
		},
		{
			name:  "IntConstantField generic",
			field: &StructField{TypeString: "fields.IntConstantField[constants.Role]"},
			want:  true,
		},
		{
			name:  "StringField generic",
			field: &StructField{TypeString: "fields.StringField[constants.Key]"},
			want:  true,
		},
		{
			name:  "Field generic",
			field: &StructField{TypeString: "fields.Field[SomeType]"},
			want:  true,
		},
		{
			name:  "plain string type",
			field: &StructField{TypeString: "string"},
			want:  false,
		},
		{
			name:  "struct type",
			field: &StructField{TypeString: "account.Properties"},
			want:  false,
		},
		{
			name:  "empty TypeString with no Type",
			field: &StructField{},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.field.IsGeneric())
		})
	}
}

func TestGenericTypeArg(t *testing.T) {
	tests := []struct {
		name    string
		field   *StructField
		want    string
		wantErr bool
	}{
		{
			name:  "simple type",
			field: &StructField{TypeString: "fields.StructField[User]"},
			want:  "User",
		},
		{
			name:  "pointer type",
			field: &StructField{TypeString: "fields.StructField[*User]"},
			want:  "User",
		},
		{
			name:  "package qualified type",
			field: &StructField{TypeString: "fields.StructField[*billing_plan.FeatureSet]"},
			want:  "billing_plan.FeatureSet",
		},
		{
			name:  "map type",
			field: &StructField{TypeString: "fields.StructField[map[string]User]"},
			want:  "map[string]User",
		},
		{
			name:    "not generic - no bracket",
			field:   &StructField{TypeString: "string"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.field.GenericTypeArg()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/model/ -v -run "TestIsGeneric|TestGenericTypeArg"`
Expected: FAIL

**Step 3: Write implementation**

Add to `internal/model/struct_field.go`:

```go
// IsGeneric returns true if this field is a generic wrapper type like
// StructField[T], IntConstantField[T], StringField[T], or any Field[T].
func (this *StructField) IsGeneric() bool {
	typeStr := this.EffectiveTypeString()
	return strings.Contains(typeStr, "StructField[") ||
		strings.Contains(typeStr, "IntConstantField[") ||
		strings.Contains(typeStr, "StringField[") ||
		strings.Contains(typeStr, "Field[")
}

// GenericTypeArg extracts the type parameter T from a generic wrapper like Field[T].
// Handles nested brackets like Field[map[string][]User].
// Returns error if the type string has no brackets or mismatched brackets.
func (this *StructField) GenericTypeArg() (string, error) {
	typeStr := this.EffectiveTypeString()
	return extractGenericTypeParameter(typeStr)
}
```

Note: `GenericTypeArg` delegates to the existing `extractGenericTypeParameter` for now. That function will be inlined later when we remove standalone functions.

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/model/ -v -run "TestIsGeneric|TestGenericTypeArg"`
Expected: PASS

**Step 5: Run all model tests**

Run: `go test ./internal/model/ -v`
Expected: All PASS

**Step 6: Commit**

```bash
git add internal/model/struct_field.go internal/model/struct_field_test.go
git commit -m "refactor: add IsGeneric and GenericTypeArg methods to StructField"
```

---

### Task 3: Add type classification methods

Add `IsPrimitive()`, `IsAny()`, `IsFieldsWrapper()`, `IsSwaggerPrimitive()`, `IsGenericTypeArgStruct()`, `IsUnderlyingStruct()`.

**Files:**
- Modify: `internal/model/struct_field.go`
- Test: `internal/model/struct_field_test.go`

**Step 1: Write the failing tests**

```go
func TestIsPrimitive(t *testing.T) {
	tests := []struct {
		name  string
		field *StructField
		want  bool
	}{
		{name: "string", field: &StructField{TypeString: "string"}, want: true},
		{name: "int64", field: &StructField{TypeString: "int64"}, want: true},
		{name: "time.Time", field: &StructField{TypeString: "time.Time"}, want: true},
		{name: "*time.Time", field: &StructField{TypeString: "*time.Time"}, want: true},
		{name: "uuid.UUID", field: &StructField{TypeString: "uuid.UUID"}, want: true},
		{name: "decimal.Decimal", field: &StructField{TypeString: "decimal.Decimal"}, want: true},
		{name: "struct type", field: &StructField{TypeString: "account.Properties"}, want: false},
		{name: "empty", field: &StructField{}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.field.IsPrimitive())
		})
	}
}

func TestIsAny(t *testing.T) {
	tests := []struct {
		name  string
		field *StructField
		want  bool
	}{
		{name: "any keyword", field: &StructField{TypeString: "any"}, want: true},
		{name: "interface{}", field: &StructField{TypeString: "interface{}"}, want: true},
		{name: "interface with space", field: &StructField{TypeString: "interface {}"}, want: true},
		{name: "string", field: &StructField{TypeString: "string"}, want: false},
		{name: "empty", field: &StructField{}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.field.IsAny())
		})
	}
}

func TestIsFieldsWrapper(t *testing.T) {
	tests := []struct {
		name  string
		field *StructField
		want  bool
	}{
		{name: "StringField", field: &StructField{TypeString: "fields.StringField"}, want: true},
		{name: "IntField", field: &StructField{TypeString: "fields.IntField"}, want: true},
		{name: "StructField generic", field: &StructField{TypeString: "fields.StructField[User]"}, want: true},
		{name: "plain string", field: &StructField{TypeString: "string"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.field.IsFieldsWrapper())
		})
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/model/ -v -run "TestIsPrimitive$|TestIsAny$|TestIsFieldsWrapper$"`
Expected: FAIL

**Step 3: Write implementation**

Add to `internal/model/struct_field.go`:

```go
// IsPrimitive returns true if this field's type is a Go primitive or an extended
// primitive (time.Time, UUID, decimal.Decimal).
func (this *StructField) IsPrimitive() bool {
	return isPrimitiveType(this.EffectiveTypeString())
}

// IsAny returns true if this field's type is any or interface{}.
func (this *StructField) IsAny() bool {
	return isAnyType(this.EffectiveTypeString())
}

// IsFieldsWrapper returns true if this field is a fields package wrapper type
// like fields.StringField, fields.IntField, etc.
func (this *StructField) IsFieldsWrapper() bool {
	return isFieldsWrapperType(this.EffectiveTypeString())
}

// IsSwaggerPrimitive returns true if the field's Go type is a struct that should
// be treated as a primitive in Swagger (e.g., time.Time, decimal.Decimal, UUID).
func (this *StructField) IsSwaggerPrimitive() bool {
	if this.Type == nil {
		return false
	}
	// Unwrap pointer
	t := this.Type
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	return shouldTreatAsSwaggerPrimitive(named)
}

// IsGenericTypeArgStruct checks whether the first type argument of this field's
// generic type is a struct. Returns true if the type has no type arguments
// (not generic), or if the argument's underlying Go type is a struct.
func (this *StructField) IsGenericTypeArgStruct() bool {
	return isGenericTypeArgStruct(this.Type)
}

// IsUnderlyingStruct checks whether this field's underlying Go type is a struct.
// Unwraps pointers and named types. Returns true for unknown types (safe default).
func (this *StructField) IsUnderlyingStruct() bool {
	return isUnderlyingStruct(this.Type)
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/model/ -v -run "TestIsPrimitive$|TestIsAny$|TestIsFieldsWrapper$"`
Expected: PASS

**Step 5: Run all model tests**

Run: `go test ./internal/model/ -v`
Expected: All PASS

**Step 6: Commit**

```bash
git add internal/model/struct_field.go internal/model/struct_field_test.go
git commit -m "refactor: add type classification methods to StructField"
```

---

### Task 4: Add NormalizedType() and ConstantFieldEnumType() methods

**Files:**
- Modify: `internal/model/struct_field.go`
- Test: `internal/model/struct_field_test.go`

**Step 1: Write the failing tests**

```go
func TestNormalizedType(t *testing.T) {
	tests := []struct {
		name  string
		field *StructField
		want  string
	}{
		{
			name:  "full module path",
			field: &StructField{TypeString: "github.com/griffnb/core-swag/testing/testdata/core_models/constants.UnionStatus"},
			want:  "constants.UnionStatus",
		},
		{
			name:  "already short form",
			field: &StructField{TypeString: "constants.UnionStatus"},
			want:  "constants.UnionStatus",
		},
		{
			name:  "pointer with full path",
			field: &StructField{TypeString: "*github.com/griffnb/core-swag/testing/testdata/core_models/constants.UnionStatus"},
			want:  "*constants.UnionStatus",
		},
		{
			name:  "simple type",
			field: &StructField{TypeString: "string"},
			want:  "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.field.NormalizedType())
		})
	}
}

func TestConstantFieldEnumType(t *testing.T) {
	tests := []struct {
		name  string
		field *StructField
		want  string
	}{
		{
			name:  "IntConstantField",
			field: &StructField{TypeString: "*fields.IntConstantField[constants.Role]"},
			want:  "constants.Role",
		},
		{
			name:  "StringConstantField",
			field: &StructField{TypeString: "*fields.StringConstantField[constants.GlobalConfigKey]"},
			want:  "constants.GlobalConfigKey",
		},
		{
			name:  "not a constant field",
			field: &StructField{TypeString: "string"},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.field.ConstantFieldEnumType())
		})
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/model/ -v -run "TestNormalizedType|TestConstantFieldEnumType"`
Expected: FAIL

**Step 3: Write implementation**

```go
// NormalizedType returns the type string in short form (package.Type instead of
// full/module/path/package.Type). Handles pointers.
func (this *StructField) NormalizedType() string {
	return normalizeTypeName(this.EffectiveTypeString())
}

// ConstantFieldEnumType extracts the enum type parameter from IntConstantField[T]
// or StringConstantField[T]. Returns empty string if not a constant field.
func (this *StructField) ConstantFieldEnumType() string {
	return extractConstantFieldEnumType(this.EffectiveTypeString())
}
```

**Step 4: Run tests, verify pass, run all model tests**

Run: `go test ./internal/model/ -v -run "TestNormalizedType|TestConstantFieldEnumType"` then `go test ./internal/model/ -v`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/model/struct_field.go internal/model/struct_field_test.go
git commit -m "refactor: add NormalizedType and ConstantFieldEnumType methods to StructField"
```

---

### Task 5: Add PrimitiveSchema() and FieldsWrapperSchema() methods

**Files:**
- Modify: `internal/model/struct_field.go`
- Test: `internal/model/struct_field_test.go`

**Step 1: Write the failing tests**

```go
func TestPrimitiveSchema(t *testing.T) {
	tests := []struct {
		name       string
		field      *StructField
		wantType   string
		wantFormat string
	}{
		{name: "string", field: &StructField{TypeString: "string"}, wantType: "string"},
		{name: "bool", field: &StructField{TypeString: "bool"}, wantType: "boolean"},
		{name: "int", field: &StructField{TypeString: "int"}, wantType: "integer"},
		{name: "int64", field: &StructField{TypeString: "int64"}, wantType: "integer", wantFormat: "int64"},
		{name: "float64", field: &StructField{TypeString: "float64"}, wantType: "number", wantFormat: "double"},
		{name: "time.Time", field: &StructField{TypeString: "time.Time"}, wantType: "string", wantFormat: "date-time"},
		{name: "uuid.UUID", field: &StructField{TypeString: "uuid.UUID"}, wantType: "string", wantFormat: "uuid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := tt.field.PrimitiveSchema()
			assert.Equal(t, tt.wantType, schema.Type[0])
			if tt.wantFormat != "" {
				assert.Equal(t, tt.wantFormat, schema.Format)
			}
		})
	}
}

func TestFieldsWrapperSchema(t *testing.T) {
	tests := []struct {
		name     string
		field    *StructField
		wantType string
	}{
		{name: "StringField", field: &StructField{TypeString: "fields.StringField"}, wantType: "string"},
		{name: "IntField", field: &StructField{TypeString: "fields.IntField"}, wantType: "integer"},
		{name: "BoolField", field: &StructField{TypeString: "fields.BoolField"}, wantType: "boolean"},
		{name: "FloatField", field: &StructField{TypeString: "fields.FloatField"}, wantType: "number"},
		{name: "UUIDField", field: &StructField{TypeString: "fields.UUIDField"}, wantType: "string"},
		{name: "TimeField", field: &StructField{TypeString: "fields.TimeField"}, wantType: "string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, nested, err := tt.field.FieldsWrapperSchema(nil)
			assert.NoError(t, err)
			assert.Nil(t, nested)
			assert.Equal(t, tt.wantType, schema.Type[0])
		})
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/model/ -v -run "TestPrimitiveSchema$|TestFieldsWrapperSchema$"`
Expected: FAIL

**Step 3: Write implementation**

```go
// PrimitiveSchema returns the OpenAPI schema for this field's primitive type.
func (this *StructField) PrimitiveSchema() *spec.Schema {
	return primitiveTypeToSchema(this.EffectiveTypeString())
}

// FieldsWrapperSchema returns the OpenAPI schema for a fields package wrapper type
// (StringField, IntField, IntConstantField[T], etc.).
func (this *StructField) FieldsWrapperSchema(enumLookup TypeEnumLookup) (*spec.Schema, []string, error) {
	return getPrimitiveSchemaForFieldType(this.EffectiveTypeString(), this.TypeString, enumLookup)
}
```

**Step 4: Run tests, verify pass, run all model tests**

Run: `go test ./internal/model/ -v`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/model/struct_field.go internal/model/struct_field_test.go
git commit -m "refactor: add PrimitiveSchema and FieldsWrapperSchema methods to StructField"
```

---

### Task 6: Add BuildSchema() method

This is the big one — replaces `buildSchemaForType` standalone function. It delegates to all the StructField methods added in previous tasks, and creates child StructField instances for recursive array/map types.

**Files:**
- Modify: `internal/model/struct_field.go`
- Test: `internal/model/struct_field_test.go`

**Step 1: Write the failing tests**

These mirror the existing `TestBuildSchemaForType` tests but call `BuildSchema` on a StructField instead:

```go
func TestBuildSchema(t *testing.T) {
	tests := []struct {
		name       string
		field      *StructField
		public     bool
		wantType   []string
		wantRef    string
		wantNested []string
	}{
		{
			name:     "string",
			field:    &StructField{TypeString: "string"},
			public:   false,
			wantType: []string{"string"},
		},
		{
			name:     "int",
			field:    &StructField{TypeString: "int"},
			public:   false,
			wantType: []string{"integer"},
		},
		{
			name:       "struct without public",
			field:      &StructField{TypeString: "User"},
			public:     false,
			wantRef:    "#/definitions/User",
			wantNested: []string{"User"},
		},
		{
			name:       "struct with public",
			field:      &StructField{TypeString: "User"},
			public:     true,
			wantRef:    "#/definitions/UserPublic",
			wantNested: []string{"UserPublic"},
		},
		{
			name:     "array of strings",
			field:    &StructField{TypeString: "[]string"},
			public:   false,
			wantType: []string{"array"},
		},
		{
			name:       "array of structs",
			field:      &StructField{TypeString: "[]User"},
			public:     false,
			wantType:   []string{"array"},
			wantNested: []string{"User"},
		},
		{
			name:     "any type",
			field:    &StructField{TypeString: "any"},
			public:   false,
			wantType: []string{"object"},
		},
		{
			name:     "interface{} type",
			field:    &StructField{TypeString: "interface{}"},
			public:   false,
			wantType: []string{"object"},
		},
		{
			name:     "fields.StringField wrapper",
			field:    &StructField{TypeString: "fields.StringField"},
			public:   false,
			wantType: []string{"string"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, nestedTypes, err := tt.field.BuildSchema(tt.public, false, nil)
			assert.NoError(t, err)
			assert.NotNil(t, schema)

			if tt.wantRef != "" {
				assert.Equal(t, tt.wantRef, schema.Ref.String())
			} else if len(tt.wantType) > 0 {
				assert.Equal(t, len(tt.wantType), len(schema.Type))
				if len(tt.wantType) > 0 {
					assert.Equal(t, tt.wantType[0], schema.Type[0])
				}
			}

			if tt.wantNested != nil {
				assert.Equal(t, tt.wantNested, nestedTypes)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/model/ -v -run TestBuildSchema$`
Expected: FAIL — `BuildSchema` not defined

**Step 3: Write implementation**

Add `BuildSchema` method to `internal/model/struct_field.go`. This is a method version of `buildSchemaForType` that uses `this` for type info and delegates to StructField methods:

```go
// BuildSchema builds an OpenAPI schema for this field's type.
// For recursive types (arrays, maps), creates child StructField instances.
// Returns schema, list of nested struct type names for definition generation, and error.
func (this *StructField) BuildSchema(
	public bool,
	forceRequired bool,
	enumLookup TypeEnumLookup,
) (*spec.Schema, []string, error) {
	var nestedTypes []string
	typeStr := this.EffectiveTypeString()

	var debug bool
	if strings.Contains(typeStr, "constants.") {
		debug = true
	}
	if debug {
		console.Logger.Debug("Building schema for type: $Bold{%s} (TypeString: $Bold{%s})\n", typeStr, this.TypeString)
	}

	// Save full type string before normalization for accurate $ref creation.
	fullTypeStr := typeStr

	// Normalize type name to short form
	typeStr = normalizeTypeName(typeStr)
	if debug && typeStr != this.TypeString {
		console.Logger.Debug("Normalized type name to: $Bold{%s}\n", typeStr)
	}

	// Remove pointer prefix
	isPointer := strings.HasPrefix(typeStr, "*")
	if isPointer {
		typeStr = strings.TrimPrefix(typeStr, "*")
	}
	fullTypeStr = strings.TrimPrefix(fullTypeStr, "*")

	// Handle any/interface{} types as object
	if isAnyType(typeStr) {
		if debug {
			console.Logger.Debug("Detected any/interface{} type: $Bold{%s}\n", typeStr)
		}
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"object"}}}, nil, nil
	}

	// Check if this is a fields wrapper type
	if isFieldsWrapperType(typeStr) {
		if debug {
			console.Logger.Debug("Detected fields wrapper type: $Bold{%s}\n", typeStr)
		}
		return getPrimitiveSchemaForFieldType(typeStr, this.TypeString, enumLookup)
	}

	// Handle primitive types
	if isPrimitiveType(typeStr) {
		schema := primitiveTypeToSchema(typeStr)
		if debug {
			console.Logger.Debug("Detected Is Primitive type: $Bold{%s} Schema %+v\n", typeStr, schema)
		}
		return schema, nil, nil
	}

	// Handle arrays — create child StructField for element type
	if strings.HasPrefix(typeStr, "[]") {
		elemType := strings.TrimPrefix(typeStr, "[]")
		fullElemType := elemType
		if strings.HasPrefix(fullTypeStr, "[]") {
			fullElemType = strings.TrimPrefix(fullTypeStr, "[]")
		}
		elemField := &StructField{TypeString: fullElemType}
		elemSchema, elemNestedTypes, err := elemField.BuildSchema(public, forceRequired, enumLookup)
		if err != nil {
			return nil, nil, err
		}
		schema := spec.ArrayProperty(elemSchema)
		return schema, elemNestedTypes, nil
	}

	// Handle maps — create child StructField for value type
	if strings.HasPrefix(typeStr, "map[") {
		bracketCount := 0
		valueStart := -1
		for i, ch := range typeStr {
			if ch == '[' {
				bracketCount++
			} else if ch == ']' {
				bracketCount--
				if bracketCount == 0 {
					valueStart = i + 1
					break
				}
			}
		}
		if valueStart == -1 {
			return nil, nil, fmt.Errorf("invalid map type: %s", typeStr)
		}
		fullValueType := typeStr[valueStart:]
		if strings.HasPrefix(fullTypeStr, "map[") {
			fullBracketCount := 0
			fullValueStart := -1
			for i, ch := range fullTypeStr {
				if ch == '[' {
					fullBracketCount++
				} else if ch == ']' {
					fullBracketCount--
					if fullBracketCount == 0 {
						fullValueStart = i + 1
						break
					}
				}
			}
			if fullValueStart != -1 {
				fullValueType = fullTypeStr[fullValueStart:]
			}
		}
		valueField := &StructField{TypeString: fullValueType}
		valueSchema, valueNestedTypes, err := valueField.BuildSchema(public, forceRequired, enumLookup)
		if err != nil {
			return nil, nil, err
		}
		schema := spec.MapProperty(valueSchema)
		return schema, valueNestedTypes, nil
	}

	// Handle struct types — filter out "any" and "interface{}"
	if typeStr == "any" || typeStr == "interface{}" {
		return &spec.Schema{}, nil, nil
	}

	// Check if this is an enum type
	if enumLookup != nil {
		if debug {
			console.Logger.Debug("Checking enum for type: $Bold{%s}\n", typeStr)
		}
		enums, err := enumLookup.GetEnumsForType(typeStr, nil)
		if err == nil && len(enums) > 0 {
			if debug {
				console.Logger.Debug("Detected Enum type: $Bold{%s} with %d values, creating $ref\n", typeStr, len(enums))
			}
			refName := resolveRefName(typeStr, fullTypeStr)
			schema := spec.RefSchema("#/definitions/" + refName)
			nestedTypes = append(nestedTypes, refName)
			return schema, nestedTypes, nil
		}
		if debug {
			if err != nil {
				console.Logger.Debug("Error looking up enums for type: $Bold{%s}: $Red{%s}\n", typeStr, err.Error())
			}
		}
	} else {
		if debug {
			console.Logger.Debug("No enumLookup provided, skipping enum check for type: $Bold{%s}\n", typeStr)
		}
	}

	// Struct ref — validate brackets
	typeName := typeStr
	bracketDepth := 0
	for _, ch := range typeName {
		if ch == '[' {
			bracketDepth++
		} else if ch == ']' {
			bracketDepth--
		}
	}
	if bracketDepth != 0 {
		console.Logger.Debug("Skipping reference creation for malformed type name with unbalanced brackets: %s\n", typeName)
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"object"}}}, nil, nil
	}

	refName := resolveRefName(typeName, fullTypeStr)
	if public {
		refName = refName + "Public"
	}

	schema := spec.RefSchema("#/definitions/" + refName)
	nestedTypes = append(nestedTypes, refName)
	if debug {
		console.Logger.Debug("Created Ref Schema for type: $Bold{$Red{%s}} Ref: $Bold{#/definitions/%s}\n", typeStr, refName)
	}
	return schema, nestedTypes, nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/model/ -v -run TestBuildSchema$`
Expected: PASS

**Step 5: Run all model tests**

Run: `go test ./internal/model/ -v`
Expected: All PASS

**Step 6: Commit**

```bash
git add internal/model/struct_field.go internal/model/struct_field_test.go
git commit -m "refactor: add BuildSchema method to StructField"
```

---

### Task 7: Switch ToSpecSchema to use new methods

Now rewrite `ToSpecSchema` to delegate to `BuildSchema` and use StructField methods for generic detection and effective-public computation.

**Files:**
- Modify: `internal/model/struct_field.go` (lines 90-187)

**Step 1: Run existing tests as baseline**

Run: `go test ./internal/model/ -v`
Expected: All PASS — capture this as baseline

**Step 2: Rewrite ToSpecSchema**

Replace `ToSpecSchema` (lines 90-187) with:

```go
// ToSpecSchema converts a StructField to OpenAPI spec.Schema
// propName: extracted from json tag (first part before comma)
// schema: the OpenAPI schema for this field
// required: true if omitempty is absent from json tag (or forceRequired is true)
// nestedTypes: list of struct type names encountered for recursive definition generation
// forceRequired: if true, field is always required regardless of omitempty tag
func (this *StructField) ToSpecSchema(
	public bool,
	forceRequired bool,
	enumLookup TypeEnumLookup,
) (propName string, schema *spec.Schema, required bool, nestedTypes []string, err error) {
	// Filter field if public mode and field is not public
	if public && !this.IsPublic() {
		return "", nil, false, nil, nil
	}

	// Check for swaggerignore tag
	tags := this.GetTags()
	if swaggerIgnore, ok := tags["swaggerignore"]; ok && strings.EqualFold(swaggerIgnore, "true") {
		console.Logger.Debug("$Red{$Bold{Ignoring field %s due to swaggerignore tag}}\n", this.Name)
		return "", nil, false, nil, nil
	}

	// Extract property name from json tag
	jsonTag := tags["json"]
	if jsonTag == "" {
		jsonTag = tags["column"]
	}
	if jsonTag == "" {
		return "", nil, false, nil, nil
	}

	parts := strings.Split(jsonTag, ",")
	propName = parts[0]

	// Check for omitempty to determine required
	if forceRequired {
		required = true
	} else {
		required = true
		for _, part := range parts[1:] {
			if strings.TrimSpace(part) == "omitempty" {
				required = false
				break
			}
		}
	}

	// Skip if json tag is "-"
	if propName == "-" {
		return "", nil, false, nil, nil
	}

	// Resolve the effective type string for schema building
	// For generic wrappers, extract the type parameter
	var schemaField *StructField
	if this.IsGeneric() {
		extractedType, extractErr := this.GenericTypeArg()
		if extractErr != nil {
			return "", nil, false, nil, fmt.Errorf("failed to extract type parameter from %s: %w", this.EffectiveTypeString(), extractErr)
		}

		// Determine effective public: only struct type args get Public suffix
		effectivePublic := public
		if public && this.Type != nil {
			if !this.IsGenericTypeArgStruct() {
				effectivePublic = false
			}
		}

		schemaField = &StructField{TypeString: extractedType, Type: this.Type}
		schema, nestedTypes, err = schemaField.BuildSchema(effectivePublic, forceRequired, enumLookup)
	} else {
		// Determine effective public: only struct types get Public suffix
		effectivePublic := public
		if public && this.Type != nil {
			if !this.IsUnderlyingStruct() {
				effectivePublic = false
			}
		}

		schema, nestedTypes, err = this.BuildSchema(effectivePublic, forceRequired, enumLookup)
	}

	if err != nil {
		return "", nil, false, nil, fmt.Errorf("failed to build schema for type %s: %w", this.EffectiveTypeString(), err)
	}

	return propName, schema, required, nestedTypes, nil
}
```

**Step 3: Run ALL existing tests**

Run: `go test ./internal/model/ -v`
Expected: All PASS — identical behavior, different code path

**Step 4: Run integration tests**

Run: `go test ./testing/ -v -run TestRealProjectIntegration -timeout 120s`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/model/struct_field.go
git commit -m "refactor: switch ToSpecSchema to use BuildSchema and StructField methods"
```

---

### Task 8: Update struct_field_lookup.go callers to use StructField methods

Switch `LookupStructFields`, `processStructField`, and `checkNamed` to use the new StructField methods instead of duplicating type analysis.

**Files:**
- Modify: `internal/model/struct_field_lookup.go` (lines 230, 247, 584)

**Step 1: Run existing tests as baseline**

Run: `go test ./internal/model/ -v`
Expected: All PASS

**Step 2: Update LookupStructFields line 230**

Change:
```go
if f.Type != nil && strings.Contains(f.Type.String(), "fields.StructField") {
```
To:
```go
if f.IsGeneric() && strings.Contains(f.EffectiveTypeString(), "fields.StructField") {
```

Note: We keep the `fields.StructField` check because `processStructField` specifically handles StructField[T] expansion (extracting sub-fields), not all generic types.

**Step 3: Update processStructField line 247**

Change:
```go
parts := strings.Split(f.Type.String(), ".StructField[")
if len(parts) != 2 {
    builder.Fields = append(builder.Fields, f)
    return
}
subTypeName := strings.TrimSuffix(parts[1], "]")
```
To:
```go
if !f.IsGeneric() {
    builder.Fields = append(builder.Fields, f)
    return
}
subTypeName, err := f.GenericTypeArg()
if err != nil {
    builder.Fields = append(builder.Fields, f)
    return
}
```

**Step 4: Update checkNamed line 584**

Create a temporary StructField to call IsSwaggerPrimitive:
```go
// Instead of: if shouldTreatAsSwaggerPrimitive(named) {
tempField := &StructField{Type: fieldType}
if tempField.IsSwaggerPrimitive() {
```

**Step 5: Run ALL tests**

Run: `go test ./internal/model/ -v`
Expected: All PASS

**Step 6: Run integration tests**

Run: `go test ./testing/ -v -run TestRealProjectIntegration -timeout 120s`
Expected: PASS

**Step 7: Commit**

```bash
git add internal/model/struct_field_lookup.go
git commit -m "refactor: update struct_field_lookup callers to use StructField methods"
```

---

### Task 9: Remove old standalone functions

Now that all callers use StructField methods, remove the standalone functions that are no longer called directly.

**Files:**
- Modify: `internal/model/struct_field.go`

**Step 1: Run all tests as baseline**

Run: `go test ./internal/model/ -v`
Expected: All PASS

**Step 2: Remove standalone functions one at a time**

Remove these functions from `struct_field.go`:
- `extractTypeParameter` (line 217-219) — was just a wrapper for `extractGenericTypeParameter`, now replaced by `GenericTypeArg()`

Check if `extractGenericTypeParameter`, `isPrimitiveType`, `isAnyType`, `isFieldsWrapperType`, `normalizeTypeName`, `shouldTreatAsSwaggerPrimitive`, `extractConstantFieldEnumType`, `primitiveTypeToSchema`, `getPrimitiveSchemaForFieldType`, `isGenericTypeArgStruct`, `isUnderlyingStruct`, `buildSchemaForType` are still called anywhere (by the new methods that delegate to them).

**Important**: The new methods currently *delegate* to the standalone functions. Do NOT remove the standalone functions that are still called by the new methods. Only remove:
- `extractTypeParameter` — duplicate, only called by tests
- `buildSchemaForType` — replaced by `BuildSchema` method (verify no callers)

For the other standalone functions (`isPrimitiveType`, `isAnyType`, etc.), they are still called by the method implementations. These can be inlined into the methods in a follow-up task, or left as private implementation details.

**Step 3: Check for compilation errors**

Run: `go build ./internal/model/`
Expected: BUILD SUCCESS

**Step 4: Run all tests**

Run: `go test ./internal/model/ -v`
Expected: All PASS (some test functions may need updating if they called `extractTypeParameter` directly)

**Step 5: Commit**

```bash
git add internal/model/struct_field.go internal/model/struct_field_test.go
git commit -m "refactor: remove replaced standalone functions"
```

---

### Task 10: Update tests to use new method signatures

Update `TestExtractTypeParameter` and `TestBuildSchemaForType` in `struct_field_test.go` to use the new StructField methods instead of calling standalone functions.

**Files:**
- Modify: `internal/model/struct_field_test.go`

**Step 1: Update TestExtractTypeParameter**

Rename to `TestGenericTypeArg_Legacy` and change calls from `extractTypeParameter(tt.typeStr)` to `(&StructField{TypeString: tt.typeStr}).GenericTypeArg()`.

**Step 2: Update TestBuildSchemaForType**

Rename to `TestBuildSchema_Legacy` and change calls from `buildSchemaForType(tt.typeStr, tt.public, false, "", nil)` to `(&StructField{TypeString: tt.typeStr}).BuildSchema(tt.public, false, nil)`.

**Step 3: Run all tests**

Run: `go test ./internal/model/ -v`
Expected: All PASS

**Step 4: Commit**

```bash
git add internal/model/struct_field_test.go
git commit -m "refactor: update tests to use StructField methods"
```

---

### Task 11: Full integration verification

**Step 1: Run all model unit tests**

Run: `go test ./internal/model/ -v`
Expected: All PASS

**Step 2: Run full test suite**

Run: `go test ./... -timeout 120s`
Expected: All PASS

**Step 3: Run integration tests on real projects**

Run: `make test-project-1`
Expected: Generates swagger without errors

Run: `make test-project-2`
Expected: Generates swagger without errors

**Step 4: Final commit**

If any fixups were needed, commit them. Otherwise, tag this as complete.

**Step 5: Update change log**

Add entry to `.agents/change_log.md` documenting the consolidation.
