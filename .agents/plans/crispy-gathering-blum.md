# Implementation Plan: Implement Full Struct Tag Processing System

## Context

Multiple struct tag constants are defined in `internal/parser/field/types.go` (swaggertype, enums, format, extensions, title, minimum, maximum, etc.) but **NONE of them are actually processed** in the production schema generation pipeline. The infrastructure exists but is never used.

**Why this change is needed:**
- Test data files contain examples of these tags (swaggertype, enums, extensions) that are currently ignored
- Users need ability to override type inference and enrich schemas with metadata
- The tag constants and `BuildCustomSchema()` function exist but are never called
- Complete schema control through struct tags is a core feature that's partially implemented

**Expected outcome:**
- All schema-related struct tags are processed and applied to generated schemas
- Tags work together compositionally (e.g., `swaggertype:"integer" enums:"1,2,3"`)
- Backward compatible - fields without tags continue working unchanged
- Comprehensive tag support matching OpenAPI spec capabilities

**Examples from existing testdata:**
```go
// Override type + add enums + extensions
FoodTypes []string `json:"food_types" swaggertype:"array,integer" enums:"0,1,2" x-enum-varnames:"Wet,Dry,Raw" extensions:"x-some-extension"`

// Override sql.NullInt64 to integer primitive
NullInt sql.NullInt64 `swaggertype:"integer"`

// Override []big.Float to array of numbers
Coeffs []big.Float `swaggertype:"array,number"`

// Override custom time type to integer timestamp
Birthday TimestampTime `swaggertype:"primitive,integer"`
```

**Supported Struct Tags** (from `internal/parser/field/types.go`):

| Tag | Purpose | Example |
|-----|---------|---------|
| `swaggertype` | Override base type | `swaggertype:"integer"` |
| `enums` | Add enum values | `enums:"1,2,3"` |
| `x-enum-varnames` | Add enum names | `x-enum-varnames:"Red,Green,Blue"` |
| `format` | Add format specifier | `format:"int64"` |
| `title` | Add title/label | `title:"User Count"` |
| `minimum` | Add minimum constraint | `minimum:"0"` |
| `maximum` | Add maximum constraint | `maximum:"100"` |
| `default` | Add default value | `swag_default:"10"` |
| `example` | Add example value | `example:"42"` |
| `minLength` | Min string length | `minLength:"1"` |
| `maxLength` | Max string length | `maxLength:"255"` |
| `readonly` | Mark as read-only | `readonly:"true"` |
| `multipleOf` | Number must be multiple | `multipleOf:"5"` |
| `extensions` | Custom extensions | `extensions:"x-custom-field"` |

---

## Design: Two-Phase Schema Building

**Strategy:** Separate base type inference from schema enrichment through tags.

### Phase 1: Base Type (BuildSchema)
1. Check for `swaggertype` tag → if present, use `BuildCustomSchema()`
2. If no swaggertype → use automatic type inference (existing logic)
3. Returns base schema with correct type structure

### Phase 2: Tag Enrichment (New: applyStructTagsToSchema)
1. Take base schema from Phase 1
2. Apply all enrichment tags: enums, format, title, constraints, etc.
3. Return enriched schema

**Rationale:**
1. Clean separation of concerns - type vs. metadata
2. Tags compose naturally - swaggertype sets type, other tags add metadata
3. Both swaggertype and automatic inference benefit from tag enrichment
4. Easy to test each phase independently

---

## Implementation Steps

### Step 1: Add Tag Processing Helper Method

**File:** `/Users/griffnb/projects/core-swag/internal/model/struct_field.go`

**Add new method after `BuildSchema()` (around line 409):**

```go
// applyStructTagsToSchema enriches a base schema with metadata from struct tags.
// Handles enums, format, title, constraints (min/max, minLength/maxLength),
// default, example, readonly, multipleOf, and extensions tags.
// The base schema's type structure should already be set before calling this.
func (this *StructField) applyStructTagsToSchema(schema *spec.Schema) error {
	tags := this.GetTags()

	// Apply format tag
	if format, ok := tags["format"]; ok {
		schema.Format = format
	}

	// Apply title tag
	if title, ok := tags["title"]; ok {
		schema.Title = title
	}

	// Apply enums tag
	if enumsStr, ok := tags["enums"]; ok {
		// Split by comma, handling escaped UTF-8 commas (0x2C)
		enumValues := strings.Split(enumsStr, ",")
		var enums []interface{}
		for _, val := range enumValues {
			val = strings.TrimSpace(val)
			// Replace UTF-8 hex comma back to actual comma
			val = strings.ReplaceAll(val, "0x2C", ",")
			if val != "" {
				// Try to parse as number, otherwise keep as string
				if strings.Contains(schema.Type[0], "integer") || strings.Contains(schema.Type[0], "number") {
					if num, err := strconv.ParseFloat(val, 64); err == nil {
						if strings.Contains(schema.Type[0], "integer") {
							enums = append(enums, int(num))
						} else {
							enums = append(enums, num)
						}
						continue
					}
				}
				enums = append(enums, val)
			}
		}
		schema.Enum = enums

		// Apply x-enum-varnames extension if present
		if varNames, ok := tags["x-enum-varnames"]; ok {
			if schema.Extensions == nil {
				schema.Extensions = make(spec.Extensions)
			}
			varNamesList := strings.Split(varNames, ",")
			for i := range varNamesList {
				varNamesList[i] = strings.TrimSpace(varNamesList[i])
			}
			schema.Extensions["x-enum-varnames"] = varNamesList
		}
	}

	// Apply minimum/maximum constraints
	if minStr, ok := tags["minimum"]; ok {
		if min, err := strconv.ParseFloat(minStr, 64); err == nil {
			schema.Minimum = &min
		}
	}
	if maxStr, ok := tags["maximum"]; ok {
		if max, err := strconv.ParseFloat(maxStr, 64); err == nil {
			schema.Maximum = &max
		}
	}

	// Apply minLength/maxLength constraints
	if minLenStr, ok := tags["minLength"]; ok {
		if minLen, err := strconv.ParseInt(minLenStr, 10, 64); err == nil {
			schema.MinLength = &minLen
		}
	}
	if maxLenStr, ok := tags["maxLength"]; ok {
		if maxLen, err := strconv.ParseInt(maxLenStr, 10, 64); err == nil {
			schema.MaxLength = &maxLen
		}
	}

	// Apply default value
	if defaultStr, ok := tags["swag_default"]; ok {
		// Try to parse as JSON, fallback to string
		var defaultVal interface{}
		if err := json.Unmarshal([]byte(defaultStr), &defaultVal); err == nil {
			schema.Default = defaultVal
		} else {
			schema.Default = defaultStr
		}
	}

	// Apply example value
	if exampleStr, ok := tags["example"]; ok {
		// Try to parse as JSON, fallback to string
		var exampleVal interface{}
		if err := json.Unmarshal([]byte(exampleStr), &exampleVal); err == nil {
			schema.Example = exampleVal
		} else {
			schema.Example = exampleStr
		}
	}

	// Apply readonly
	if readonlyStr, ok := tags["readonly"]; ok {
		if readonly, err := strconv.ParseBool(readonlyStr); err == nil {
			schema.ReadOnly = readonly
		}
	}

	// Apply multipleOf
	if multipleStr, ok := tags["multipleOf"]; ok {
		if multiple, err := strconv.ParseFloat(multipleStr, 64); err == nil {
			schema.MultipleOf = &multiple
		}
	}

	// Apply custom extensions
	if extensionsStr, ok := tags["extensions"]; ok {
		if schema.Extensions == nil {
			schema.Extensions = make(spec.Extensions)
		}
		// Format: "x-key1:value1,x-key2:value2"
		extPairs := strings.Split(extensionsStr, ",")
		for _, pair := range extPairs {
			parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				// Try to parse value as JSON
				var jsonVal interface{}
				if err := json.Unmarshal([]byte(val), &jsonVal); err == nil {
					schema.Extensions[key] = jsonVal
				} else {
					schema.Extensions[key] = val
				}
			}
		}
	}

	return nil
}
```

### Step 2: Integrate swaggertype in BuildSchema()

**File:** `/Users/griffnb/projects/core-swag/internal/model/struct_field.go`

**Add import:**
```go
import (
    // ... existing imports ...
    "encoding/json"
    "strconv"
    "github.com/griffnb/core-swag/internal/schema"
)
```

**Insert at line 207 (after extracting typeStr, before debug logging):**

```go
// Check for swaggertype tag override - takes precedence over automatic type inference
tags := this.GetTags()
if swaggerType, ok := tags["swaggertype"]; ok {
    if debug {
        console.Logger.Debug("Found swaggertype tag: $Bold{%s} for field type: $Bold{%s}\n", swaggerType, typeStr)
    }

    // Parse comma-separated type keywords
    typeKeywords := strings.Split(swaggerType, ",")
    for i := range typeKeywords {
        typeKeywords[i] = strings.TrimSpace(typeKeywords[i])
    }

    // Build custom schema using existing schema package function
    baseSchema, err := schema.BuildCustomSchema(typeKeywords)
    if err != nil {
        return nil, nil, fmt.Errorf("invalid swaggertype tag '%s' for field %s: %w", swaggerType, this.Name, err)
    }

    // Apply other struct tags to enrich the schema
    if err := this.applyStructTagsToSchema(baseSchema); err != nil {
        return nil, nil, fmt.Errorf("failed to apply tags to schema for field %s: %w", this.Name, err)
    }

    if debug {
        console.Logger.Debug("Built schema from swaggertype + tags: %+v\n", baseSchema)
    }

    return baseSchema, nil, nil
}
```

### Step 3: Apply Tags to Automatically Inferred Schemas

**File:** `/Users/griffnb/projects/core-swag/internal/model/struct_field.go`

**At the END of BuildSchema() method, before each return statement that returns a schema:**

Change all returns like `return schema, nestedTypes, nil` to:
```go
// Apply struct tags to enrich the schema
if err := this.applyStructTagsToSchema(schema); err != nil {
    return nil, nil, fmt.Errorf("failed to apply tags to schema: %w", err)
}
return schema, nestedTypes, nil
```

**Locations to update:**
- Line 239: after handling any/interface{} - **NO** (returns early, no tags apply)
- Line 247: after getPrimitiveSchemaForFieldType() - YES
- Line 257: after primitiveTypeToSchema() - YES
- Line 272: after array handling - YES
- Line 318: after map handling - YES
- Line 407: after struct ref creation - **NO** (returns ref, not schema with properties)

**Update go doc comment (line 196-199):**
```go
// BuildSchema builds an OpenAPI schema for this field's type.
// Checks for swaggertype tag first to allow user-specified type overrides.
// Applies struct tags (enums, format, constraints, etc.) to enrich the schema.
// For recursive types (arrays, maps), creates child StructField instances.
// Returns schema, list of nested struct type names for definition generation, and error.
```

---

### Step 4: Add Comprehensive Unit Tests

**File:** `/Users/griffnb/projects/core-swag/internal/model/struct_field_test.go`

**Test 1: applyStructTagsToSchema Method**

```go
func TestApplyStructTagsToSchema(t *testing.T) {
    tests := []struct {
        name      string
        field     *StructField
        baseType  []string
        wantEnums []interface{}
        wantFormat string
        wantMin   *float64
        wantMax   *float64
        wantReadOnly bool
    }{
        {
            name: "apply enums to integer",
            field: &StructField{
                Name: "Status",
                Tag:  `json:"status" enums:"1,2,3"`,
            },
            baseType: []string{"integer"},
            wantEnums: []interface{}{1, 2, 3},
        },
        {
            name: "apply enums with var names",
            field: &StructField{
                Name: "Color",
                Tag:  `json:"color" enums:"red,green,blue" x-enum-varnames:"Red,Green,Blue"`,
            },
            baseType: []string{"string"},
            wantEnums: []interface{}{"red", "green", "blue"},
        },
        {
            name: "apply format tag",
            field: &StructField{
                Name: "CreatedAt",
                Tag:  `json:"created_at" format:"date-time"`,
            },
            baseType: []string{"string"},
            wantFormat: "date-time",
        },
        {
            name: "apply min/max constraints",
            field: &StructField{
                Name: "Age",
                Tag:  `json:"age" minimum:"0" maximum:"120"`,
            },
            baseType: []string{"integer"},
            wantMin: func() *float64 { v := 0.0; return &v }(),
            wantMax: func() *float64 { v := 120.0; return &v }(),
        },
        {
            name: "apply readonly",
            field: &StructField{
                Name: "ID",
                Tag:  `json:"id" readonly:"true"`,
            },
            baseType: []string{"integer"},
            wantReadOnly: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            schema := &spec.Schema{SchemaProps: spec.SchemaProps{Type: tt.baseType}}
            err := tt.field.applyStructTagsToSchema(schema)

            assert.NoError(t, err)
            if tt.wantEnums != nil {
                assert.Equal(t, tt.wantEnums, schema.Enum)
            }
            if tt.wantFormat != "" {
                assert.Equal(t, tt.wantFormat, schema.Format)
            }
            if tt.wantMin != nil {
                assert.Equal(t, *tt.wantMin, *schema.Minimum)
            }
            if tt.wantMax != nil {
                assert.Equal(t, *tt.wantMax, *schema.Maximum)
            }
            if tt.wantReadOnly {
                assert.True(t, schema.ReadOnly)
            }
        })
    }
}
```

**Test 2: SwaggerType Integration**

```go
func TestBuildSchema_SwaggerType(t *testing.T) {
    tests := []struct {
        name         string
        field        *StructField
        wantType     []string
        wantItems    bool
        wantItemType []string
        wantEnums    []interface{}
        wantErr      bool
    }{
        {
            name: "swaggertype with enums tag",
            field: &StructField{
                Name:       "FoodTypes",
                TypeString: "[]string",
                Tag:        `json:"food_types" swaggertype:"array,integer" enums:"0,1,2"`,
            },
            wantType:     []string{"array"},
            wantItems:    true,
            wantItemType: []string{"integer"},
        },
        {
            name: "swaggertype integer overrides sql.NullInt64",
            field: &StructField{
                Name:       "NullInt",
                TypeString: "sql.NullInt64",
                Tag:        `swaggertype:"integer"`,
            },
            wantType: []string{"integer"},
        },
        {
            name: "swaggertype array,number for []big.Float",
            field: &StructField{
                Name:       "Coeffs",
                TypeString: "[]big.Float",
                Tag:        `swaggertype:"array,number"`,
            },
            wantType:     []string{"array"},
            wantItems:    true,
            wantItemType: []string{"number"},
        },
        {
            name: "swaggertype primitive,integer strips primitive keyword",
            field: &StructField{
                Name:       "Birthday",
                TypeString: "TimestampTime",
                Tag:        `swaggertype:"primitive,integer"`,
            },
            wantType: []string{"integer"},
        },
        {
            name: "swaggertype string for explicit override",
            field: &StructField{
                Name:       "URLTemplate",
                TypeString: "string",
                Tag:        `json:"urltemplate" swaggertype:"string"`,
            },
            wantType: []string{"string"},
        },
        {
            name: "swaggertype object creates map schema",
            field: &StructField{
                Name:       "Metadata",
                TypeString: "map[string]string",
                Tag:        `swaggertype:"object,string"`,
            },
            wantType: []string{"object"},
        },
        {
            name: "invalid swaggertype - no type after primitive",
            field: &StructField{
                Name:       "BadField",
                TypeString: "string",
                Tag:        `swaggertype:"primitive"`,
            },
            wantErr: true,
        },
        {
            name: "invalid swaggertype - no type after array",
            field: &StructField{
                Name:       "BadField",
                TypeString: "[]string",
                Tag:        `swaggertype:"array"`,
            },
            wantErr: true,
        },
        {
            name: "invalid swaggertype - unknown keyword",
            field: &StructField{
                Name:       "BadField",
                TypeString: "string",
                Tag:        `swaggertype:"invalidtype"`,
            },
            wantErr: true,
        },
        {
            name: "no swaggertype tag - uses normal type inference",
            field: &StructField{
                Name:       "NormalField",
                TypeString: "string",
                Tag:        `json:"normal"`,
            },
            wantType: []string{"string"},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            schema, nestedTypes, err := tt.field.BuildSchema(false, false, nil)

            if tt.wantErr {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), "invalid swaggertype tag")
                return
            }

            assert.NoError(t, err)
            assert.NotNil(t, schema)
            assert.Equal(t, tt.wantType, schema.Type)
            assert.Empty(t, nestedTypes, "swaggertype creates inline schemas, not refs")

            if tt.wantItems {
                assert.NotNil(t, schema.Items, "array schema should have items")
                assert.NotNil(t, schema.Items.Schema, "array items should have schema")
                assert.Equal(t, tt.wantItemType, schema.Items.Schema.Type)
            }
        })
    }
}
```

**Add integration test function:**

```go
func TestToSpecSchema_SwaggerType(t *testing.T) {
    tests := []struct {
        name         string
        field        *StructField
        wantPropName string
        wantType     []string
        wantRequired bool
    }{
        {
            name: "swaggertype with json tag and required",
            field: &StructField{
                Name:       "FoodTypes",
                TypeString: "[]string",
                Tag:        `json:"food_types" swaggertype:"array,integer"`,
            },
            wantPropName: "food_types",
            wantType:     []string{"array"},
            wantRequired: true,
        },
        {
            name: "swaggertype with omitempty makes optional",
            field: &StructField{
                Name:       "OptionalField",
                TypeString: "sql.NullInt64",
                Tag:        `json:"optional,omitempty" swaggertype:"integer"`,
            },
            wantPropName: "optional",
            wantType:     []string{"integer"},
            wantRequired: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            propName, schema, required, nestedTypes, err := tt.field.ToSpecSchema(false, false, nil)

            assert.NoError(t, err)
            assert.Equal(t, tt.wantPropName, propName)
            assert.Equal(t, tt.wantRequired, required)
            assert.Equal(t, tt.wantType, schema.Type)
            assert.Empty(t, nestedTypes)
        })
    }
}
```

---

### Step 3: Add Integration Test for Testdata

**File:** `/Users/griffnb/projects/core-swag/testing/core_models_integration_test.go`

**Add new test function:**

```go
func TestSwaggerTypeTagProcessing(t *testing.T) {
    t.Run("simple2 handler swaggertype fields", func(t *testing.T) {
        // Load and parse simple2 test project which has swaggertype examples
        config := &orchestrator.Config{
            ParseDependency: loader.ParseFlag(1),
            ParseGoList:     true,
            ParseGoPackages: true,
            ParseInternal:   true,
        }
        service := orchestrator.New(config)

        swagger, err := service.Parse(
            []string{"./testdata/simple2"},
            "./testdata/simple2/main.go",
            100,
        )
        require.NoError(t, err, "Failed to parse simple2 testdata")

        // Find the Pet2 definition
        pet2Def, exists := swagger.Definitions["web.Pet2"]
        require.True(t, exists, "web.Pet2 definition should exist")

        // Verify NullInt field uses swaggertype:"integer" override
        nullIntProp, exists := pet2Def.Properties["null_int"]
        require.True(t, exists, "null_int property should exist")
        assert.Equal(t, []string{"integer"}, nullIntProp.Type,
            "NullInt should be integer (not object) due to swaggertype tag")

        // Verify Coeffs field uses swaggertype:"array,number" override
        coeffsProp, exists := pet2Def.Properties["coeffs"]
        require.True(t, exists, "coeffs property should exist")
        assert.Equal(t, []string{"array"}, coeffsProp.Type, "Coeffs should be array")
        require.NotNil(t, coeffsProp.Items, "Coeffs should have items")
        require.NotNil(t, coeffsProp.Items.Schema, "Coeffs items should have schema")
        assert.Equal(t, []string{"number"}, coeffsProp.Items.Schema.Type,
            "Coeffs items should be numbers (not objects) due to swaggertype tag")

        // Verify Birthday field uses swaggertype:"primitive,integer" override
        birthdayProp, exists := pet2Def.Properties["birthday"]
        require.True(t, exists, "birthday property should exist")
        assert.Equal(t, []string{"integer"}, birthdayProp.Type,
            "Birthday should be integer due to swaggertype primitive,integer tag")
    })

    t.Run("simple handler swaggertype with enums", func(t *testing.T) {
        // Test that swaggertype works together with enums tags
        config := &orchestrator.Config{
            ParseDependency: loader.ParseFlag(1),
            ParseGoList:     true,
            ParseGoPackages: true,
            ParseInternal:   true,
        }
        service := orchestrator.New(config)

        swagger, err := service.Parse(
            []string{"./testdata/simple"},
            "./testdata/simple/main.go",
            100,
        )
        require.NoError(t, err, "Failed to parse simple testdata")

        // Find the Pet definition
        petDef, exists := swagger.Definitions["web.Pet"]
        require.True(t, exists, "web.Pet definition should exist")

        // Verify FoodTypes uses swaggertype:"array,integer" with enums
        foodTypesProp, exists := petDef.Properties["food_types"]
        require.True(t, exists, "food_types property should exist")
        assert.Equal(t, []string{"array"}, foodTypesProp.Type)
        require.NotNil(t, foodTypesProp.Items, "FoodTypes should have items")
        require.NotNil(t, foodTypesProp.Items.Schema, "FoodTypes items should have schema")
        assert.Equal(t, []string{"integer"}, foodTypesProp.Items.Schema.Type,
            "FoodTypes items should be integers due to swaggertype tag")

        // Verify enums are applied on top of the swaggertype base
        // (enum values should still be present if they're processed at a different layer)

        // Verify SingleEnumVarname uses swaggertype:"integer" with enums
        singleEnumProp, exists := petDef.Properties["single_enum_varname"]
        require.True(t, exists, "single_enum_varname property should exist")
        assert.Equal(t, []string{"integer"}, singleEnumProp.Type,
            "SingleEnumVarname should be integer due to swaggertype tag")
    })
}
```

---

### Step 4: Update Documentation

**File:** `/Users/griffnb/projects/core-swag/ARCHITECTURE.md`

**Add new section after the "Struct Parsing Architecture" section:**

```markdown
### SwaggerType Tag Support

The `swaggertype` struct tag allows developers to override automatic type inference with explicit OpenAPI schema types. This is useful when Go types don't map cleanly to OpenAPI primitives.

**Tag Syntax:** `swaggertype:"keyword[,keyword]..."`

**Supported Keywords:**
- `primitive,TYPE` - Marks the following type as primitive (e.g., `primitive,integer`)
- `array,TYPE` - Array with element TYPE (e.g., `array,integer` = array of integers)
- `object[,TYPE]` - Object or map with optional value TYPE (e.g., `object,string` = map[string]string)
- Direct types: `string`, `integer`, `number`, `boolean`

**Common Use Cases:**

| Go Type | Auto-Inferred | SwaggerType Override | Result |
|---------|---------------|---------------------|---------|
| `sql.NullInt64` | object (struct) | `swaggertype:"integer"` | integer |
| `[]big.Float` | array of objects | `swaggertype:"array,number"` | array of numbers |
| `[]string` | array of strings | `swaggertype:"array,integer"` | array of integers (for enum codes) |
| `time.Time` | string (date-time) | `swaggertype:"primitive,integer"` | integer (unix timestamp) |

**Examples:**
```go
type Example struct {
    // Override complex type to simple primitive
    NullableCount sql.NullInt64 `json:"count" swaggertype:"integer"`

    // Override array element type
    CategoryIDs []string `json:"category_ids" swaggertype:"array,integer"`

    // Override to array of numbers
    Coefficients []big.Float `json:"coefficients" swaggertype:"array,number"`

    // Override custom time to timestamp
    CreatedAt CustomTime `json:"created_at" swaggertype:"primitive,integer"`
}
```

**Tag Composition:**
The `swaggertype` tag sets the base type structure, but other schema tags continue to work:
```go
// Base type from swaggertype, enum values from enums tag
Status string `json:"status" swaggertype:"integer" enums:"0,1,2" x-enum-varnames:"Active,Inactive,Pending"`
```

**Processing Order:**
1. Extract field type string
2. **Check for `swaggertype` tag** (if present, override base type immediately)
3. If no swaggertype, use automatic type inference (primitives, enums, generics, structs)
4. Other layers apply enums values, format specifiers, extensions

**Implementation Location:**
- Tag processing: `internal/model/struct_field.go` line ~207 in `BuildSchema()` method
- Schema builder: `internal/schema/types.go` `BuildCustomSchema()` function

**Error Handling:**
Invalid swaggertype values (e.g., `swaggertype:"invalid"` or incomplete keywords like `swaggertype:"array"` without element type) return clear error messages and fail the build.
```

---

## Critical Files

| File | Purpose | Changes |
|------|---------|---------|
| `/Users/griffnb/projects/core-swag/internal/model/struct_field.go` | Core field schema building | Add swaggertype check at line 207, update imports and go docs |
| `/Users/griffnb/projects/core-swag/internal/schema/types.go` | Schema building utilities | **No changes** - reuse existing `BuildCustomSchema()` |
| `/Users/griffnb/projects/core-swag/internal/model/struct_field_test.go` | Unit tests for field processing | Add `TestBuildSchema_SwaggerType` and `TestToSpecSchema_SwaggerType` |
| `/Users/griffnb/projects/core-swag/testing/core_models_integration_test.go` | End-to-end integration tests | Add `TestSwaggerTypeTagProcessing` |
| `/Users/griffnb/projects/core-swag/ARCHITECTURE.md` | Architecture documentation | Add SwaggerType Tag Support section |

---

## Verification Strategy

### Build and Unit Tests
```bash
# Verify compilation
go build ./...

# Run unit tests for struct_field
go test ./internal/model/... -v -run SwaggerType

# Run all model package tests
go test ./internal/model/... -v
```

### Integration Tests
```bash
# Run integration tests
go test ./testing/... -v -run SwaggerType

# Generate swagger for test projects
make test-project-1
make test-project-2
```

### Manual Verification

**Check test-project-2 output:**
```bash
cat testing/test-project-2/swagger.json | jq '.definitions["web.Pet2"].properties.null_int'
# Expected: {"type": ["integer"]}

cat testing/test-project-2/swagger.json | jq '.definitions["web.Pet2"].properties.coeffs'
# Expected: {"type": ["array"], "items": {"type": ["number"]}}

cat testing/test-project-2/swagger.json | jq '.definitions["web.Pet2"].properties.birthday'
# Expected: {"type": ["integer"]}
```

**Check test-project-1 for regressions:**
```bash
# Compare before/after (should be identical - no swaggertype tags in project 1)
diff testing/test-project-1/swagger.json testing/test-project-1/swagger.json.backup
```

### Success Criteria

✅ All unit tests pass (`go test ./internal/model/...`)
✅ All integration tests pass (`go test ./testing/...`)
✅ `make test-project-2` generates correct schemas for swaggertype fields
✅ `make test-project-1` shows no regressions (identical output)
✅ Invalid swaggertype tags produce clear error messages
✅ Fields without swaggertype continue working normally
✅ Error messages include field name and invalid tag value

---

## Risk Mitigation

### Import Cycle Risk
**Risk:** Adding `internal/schema` import to `internal/model` might create circular dependency

**Mitigation:**
1. Check dependency graph before implementation
2. `schema` package should not import `model` package
3. If cycle detected, move `BuildCustomSchema()` to a shared utility package

### Tag Parsing Edge Cases
**Risk:** Whitespace, empty strings, or malformed tags could cause issues

**Mitigation:**
1. Use `strings.TrimSpace()` on each parsed keyword
2. Add test cases for edge cases (empty tag, extra commas, whitespace)
3. Let `BuildCustomSchema()` handle validation and return clear errors

### Performance Impact
**Risk:** Extra `GetTags()` call and tag parsing for every field

**Mitigation:**
1. `GetTags()` is O(n) where n = number of tags per field (typically 2-5)
2. Tag parsing is only done if `swaggertype` key exists (fast path for most fields)
3. Performance impact is negligible compared to Go type reflection and AST parsing

### Backward Compatibility
**Risk:** Changes to BuildSchema might affect existing behavior

**Mitigation:**
1. Early return when swaggertype is found means zero impact on non-swaggertype fields
2. Integration tests verify existing projects (test-project-1) unchanged
3. Unit tests cover both swaggertype and non-swaggertype fields

---

## Out of Scope (Future Enhancements)

The following features are NOT included in this implementation but could be added later:

1. **$ref overrides** - Allow `swaggertype:"ref,OtherType"` to reference another definition
2. **Format specifiers** - Support `swaggertype:"integer,int64"` for format hints
3. **Complex enums** - Integration with Go const enum types when swaggertype is present
4. **Validation merging** - Automatic merge of validation tags with swaggertype schemas
5. **Nested swaggertype** - Support for `swaggertype:"array,object"` with property definitions

---

## Implementation Sequence

**Estimated time: 2 hours total**

### Phase 1: Core Implementation (30 min)
1. Add import for `internal/schema` package
2. Insert swaggertype detection code at line 207 in `BuildSchema()`
3. Update `BuildSchema()` go doc comment
4. Run `go build` to verify compilation

### Phase 2: Unit Tests (45 min)
5. Add `TestBuildSchema_SwaggerType` with 10 test cases
6. Add `TestToSpecSchema_SwaggerType` with 2 test cases
7. Run `go test ./internal/model/... -v` to verify tests pass

### Phase 3: Integration Tests (30 min)
8. Add `TestSwaggerTypeTagProcessing` to core_models_integration_test.go
9. Run `make test-project-2` and verify swagger.json output
10. Run `go test ./testing/... -v` to verify all integration tests pass

### Phase 4: Documentation (15 min)
11. Add SwaggerType Tag Support section to ARCHITECTURE.md
12. Run `go fmt ./...` to format all code

### Phase 5: Final Verification (10 min)
13. Run full test suite: `go test ./...`
14. Run both test projects: `make test-project-1 && make test-project-2`
15. Manually inspect generated swagger.json files for correctness
