package schema

import (
	"testing"

	"github.com/go-openapi/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Test parseFieldOverrides - Parsing "field1=Type1,field2=Type2" syntax
// ============================================================================

func TestParseFieldOverrides_SingleField(t *testing.T) {
	// Parse simple single field override
	result, err := parseFieldOverrides("data=Account")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"data": "Account"}, result)
}

func TestParseFieldOverrides_MultipleFields(t *testing.T) {
	// Parse multiple field overrides separated by commas
	result, err := parseFieldOverrides("data=Account,meta=Metadata")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"data": "Account",
		"meta": "Metadata",
	}, result)
}

func TestParseFieldOverrides_ThreeFields(t *testing.T) {
	// Parse three field overrides
	result, err := parseFieldOverrides("data=Account,meta=Meta,status=Status")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"data":   "Account",
		"meta":   "Meta",
		"status": "Status",
	}, result)
}

func TestParseFieldOverrides_ArrayType(t *testing.T) {
	// Parse array type in field override
	result, err := parseFieldOverrides("data=[]Account")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"data": "[]Account"}, result)
}

func TestParseFieldOverrides_MapType(t *testing.T) {
	// Parse map type in field override
	result, err := parseFieldOverrides("data=map[string]int")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"data": "map[string]int"}, result)
}

func TestParseFieldOverrides_PointerType(t *testing.T) {
	// Parse pointer type in field override
	result, err := parseFieldOverrides("data=*Account")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"data": "*Account"}, result)
}

func TestParseFieldOverrides_NestedBraces(t *testing.T) {
	// Parse nested braces - should respect bracket depth
	result, err := parseFieldOverrides("data=Inner{field=Account}")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"data": "Inner{field=Account}"}, result)
}

func TestParseFieldOverrides_NestedBracesMultipleFields(t *testing.T) {
	// Parse multiple fields with nested braces
	result, err := parseFieldOverrides("data=Inner{field=Account},meta=Metadata")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"data": "Inner{field=Account}",
		"meta": "Metadata",
	}, result)
}

func TestParseFieldOverrides_DeepNesting(t *testing.T) {
	// Parse deeply nested braces
	result, err := parseFieldOverrides("data=A{b=B{c=C}}")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"data": "A{b=B{c=C}}"}, result)
}

func TestParseFieldOverrides_EmptyString(t *testing.T) {
	// Empty string should return empty map
	result, err := parseFieldOverrides("")
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestParseFieldOverrides_TrailingComma(t *testing.T) {
	// Trailing comma should be handled gracefully
	result, err := parseFieldOverrides("data=Account,")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"data": "Account"}, result)
}

func TestParseFieldOverrides_SpacesAroundEquals(t *testing.T) {
	// Spaces around equals should be trimmed
	result, err := parseFieldOverrides("data = Account")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"data": "Account"}, result)
}

func TestParseFieldOverrides_InvalidNoEquals(t *testing.T) {
	// Field without equals should return error
	_, err := parseFieldOverrides("data")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid field override")
}

func TestParseFieldOverrides_InvalidEmptyFieldName(t *testing.T) {
	// Empty field name should return error
	_, err := parseFieldOverrides("=Account")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty field name")
}

func TestParseFieldOverrides_InvalidEmptyTypeName(t *testing.T) {
	// Empty type name should return error
	_, err := parseFieldOverrides("data=")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty type name")
}

// ============================================================================
// Test parseCombinedType - Extracting base type and overrides
// ============================================================================

func TestParseCombinedType_BasicSingleField(t *testing.T) {
	// Parse basic combined type with single field
	baseType, overrides, err := parseCombinedType("Response{data=Account}")
	require.NoError(t, err)
	assert.Equal(t, "Response", baseType)
	assert.Equal(t, map[string]string{"data": "Account"}, overrides)
}

func TestParseCombinedType_PackageQualifiedBase(t *testing.T) {
	// Parse with package-qualified base type
	baseType, overrides, err := parseCombinedType("response.SuccessResponse{data=Account}")
	require.NoError(t, err)
	assert.Equal(t, "response.SuccessResponse", baseType)
	assert.Equal(t, map[string]string{"data": "Account"}, overrides)
}

func TestParseCombinedType_PackageQualifiedOverride(t *testing.T) {
	// Parse with package-qualified override type
	baseType, overrides, err := parseCombinedType("response.SuccessResponse{data=account.Account}")
	require.NoError(t, err)
	assert.Equal(t, "response.SuccessResponse", baseType)
	assert.Equal(t, map[string]string{"data": "account.Account"}, overrides)
}

func TestParseCombinedType_MultipleFields(t *testing.T) {
	// Parse with multiple field overrides
	baseType, overrides, err := parseCombinedType("Response{data=Account,meta=Metadata}")
	require.NoError(t, err)
	assert.Equal(t, "Response", baseType)
	assert.Equal(t, map[string]string{
		"data": "Account",
		"meta": "Metadata",
	}, overrides)
}

func TestParseCombinedType_ArrayOverride(t *testing.T) {
	// Parse with array type in override
	baseType, overrides, err := parseCombinedType("Response{data=[]Account}")
	require.NoError(t, err)
	assert.Equal(t, "Response", baseType)
	assert.Equal(t, map[string]string{"data": "[]Account"}, overrides)
}

func TestParseCombinedType_NoOverrides(t *testing.T) {
	// Type with no overrides (just type name)
	baseType, overrides, err := parseCombinedType("Response")
	require.NoError(t, err)
	assert.Equal(t, "Response", baseType)
	assert.Empty(t, overrides)
}

func TestParseCombinedType_EmptyBraces(t *testing.T) {
	// Type with empty braces should return no overrides
	baseType, overrides, err := parseCombinedType("Response{}")
	require.NoError(t, err)
	assert.Equal(t, "Response", baseType)
	assert.Empty(t, overrides)
}

func TestParseCombinedType_NestedCombinedType(t *testing.T) {
	// Nested combined type in override
	baseType, overrides, err := parseCombinedType("Response{data=Inner{field=Account}}")
	require.NoError(t, err)
	assert.Equal(t, "Response", baseType)
	assert.Equal(t, map[string]string{"data": "Inner{field=Account}"}, overrides)
}

func TestParseCombinedType_InvalidNoClosingBrace(t *testing.T) {
	// Missing closing brace should return error
	_, _, err := parseCombinedType("Response{data=Account")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing closing brace")
}

func TestParseCombinedType_InvalidExtraClosingBrace(t *testing.T) {
	// Extra closing brace should return error
	_, _, err := parseCombinedType("Response{data=Account}}")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid format")
}

func TestParseCombinedType_EmptyString(t *testing.T) {
	// Empty string should return error
	_, _, err := parseCombinedType("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty type")
}

// ============================================================================
// Test shouldUseAllOf - Determining if AllOf composition is needed
// ============================================================================

func TestShouldUseAllOf_WithRefAndOverrides(t *testing.T) {
	// Schema with ref and overrides should use AllOf
	baseSchema := &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Ref: spec.MustCreateRef("#/definitions/Response"),
		},
	}
	overrides := map[string]spec.Schema{
		"data": {SchemaProps: spec.SchemaProps{Type: []string{"string"}}},
	}
	assert.True(t, shouldUseAllOf(baseSchema, overrides))
}

func TestShouldUseAllOf_NoOverrides(t *testing.T) {
	// No overrides should not use AllOf
	baseSchema := &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Ref: spec.MustCreateRef("#/definitions/Response"),
		},
	}
	assert.False(t, shouldUseAllOf(baseSchema, nil))
}

func TestShouldUseAllOf_EmptyOverrides(t *testing.T) {
	// Empty overrides map should not use AllOf
	baseSchema := &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Ref: spec.MustCreateRef("#/definitions/Response"),
		},
	}
	assert.False(t, shouldUseAllOf(baseSchema, map[string]spec.Schema{}))
}

func TestShouldUseAllOf_EmptyObjectWithOverrides(t *testing.T) {
	// Empty object (no ref, no properties) with overrides should not use AllOf
	// Instead, properties should be merged directly
	baseSchema := &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type: []string{"object"},
		},
	}
	overrides := map[string]spec.Schema{
		"data": {SchemaProps: spec.SchemaProps{Type: []string{"string"}}},
	}
	assert.False(t, shouldUseAllOf(baseSchema, overrides))
}

func TestShouldUseAllOf_ObjectWithPropertiesAndOverrides(t *testing.T) {
	// Object with existing properties and overrides should use AllOf
	baseSchema := &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type: []string{"object"},
			Properties: map[string]spec.Schema{
				"existing": {SchemaProps: spec.SchemaProps{Type: []string{"string"}}},
			},
		},
	}
	overrides := map[string]spec.Schema{
		"data": {SchemaProps: spec.SchemaProps{Type: []string{"string"}}},
	}
	assert.True(t, shouldUseAllOf(baseSchema, overrides))
}

func TestShouldUseAllOf_NilBaseSchema(t *testing.T) {
	// Nil base schema should not use AllOf
	overrides := map[string]spec.Schema{
		"data": {SchemaProps: spec.SchemaProps{Type: []string{"string"}}},
	}
	assert.False(t, shouldUseAllOf(nil, overrides))
}

// ============================================================================
// Test buildAllOfSchema - Building AllOf schema structures
// ============================================================================

func TestBuildAllOfSchema_BasicRefWithSingleProperty(t *testing.T) {
	// Build AllOf with ref base and single property override
	baseSchema := &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Ref: spec.MustCreateRef("#/definitions/Response"),
		},
	}
	overrides := map[string]spec.Schema{
		"data": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/Account"),
			},
		},
	}

	result := buildAllOfSchema(baseSchema, overrides)
	require.NotNil(t, result)
	require.Len(t, result.AllOf, 2)

	// First item should be the base ref
	assert.Equal(t, "#/definitions/Response", result.AllOf[0].Ref.String())

	// Second item should be object with property override
	assert.Equal(t, spec.StringOrArray{"object"}, result.AllOf[1].Type)
	require.Contains(t, result.AllOf[1].Properties, "data")
	dataProperty := result.AllOf[1].Properties["data"]
	assert.Equal(t, "#/definitions/Account", dataProperty.Ref.String())
}

func TestBuildAllOfSchema_MultiplePropertyOverrides(t *testing.T) {
	// Build AllOf with multiple property overrides
	baseSchema := &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Ref: spec.MustCreateRef("#/definitions/Response"),
		},
	}
	overrides := map[string]spec.Schema{
		"data": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/Account"),
			},
		},
		"meta": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/Metadata"),
			},
		},
	}

	result := buildAllOfSchema(baseSchema, overrides)
	require.NotNil(t, result)
	require.Len(t, result.AllOf, 2)

	// Check both properties exist in override
	assert.Contains(t, result.AllOf[1].Properties, "data")
	assert.Contains(t, result.AllOf[1].Properties, "meta")
}

func TestBuildAllOfSchema_ArrayItemOverride(t *testing.T) {
	// Build AllOf with array type in property override
	baseSchema := &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Ref: spec.MustCreateRef("#/definitions/Response"),
		},
	}
	overrides := map[string]spec.Schema{
		"data": {
			SchemaProps: spec.SchemaProps{
				Type: []string{"array"},
				Items: &spec.SchemaOrArray{
					Schema: &spec.Schema{
						SchemaProps: spec.SchemaProps{
							Ref: spec.MustCreateRef("#/definitions/Account"),
						},
					},
				},
			},
		},
	}

	result := buildAllOfSchema(baseSchema, overrides)
	require.NotNil(t, result)
	require.Len(t, result.AllOf, 2)

	// Check array property
	dataSchema := result.AllOf[1].Properties["data"]
	assert.Equal(t, spec.StringOrArray{"array"}, dataSchema.Type)
	require.NotNil(t, dataSchema.Items)
	assert.Equal(t, "#/definitions/Account", dataSchema.Items.Schema.Ref.String())
}

func TestBuildAllOfSchema_EmptyObjectMergesDirectly(t *testing.T) {
	// Empty object base should merge properties directly (no AllOf)
	baseSchema := &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type: []string{"object"},
		},
	}
	overrides := map[string]spec.Schema{
		"data": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/Account"),
			},
		},
	}

	result := buildAllOfSchema(baseSchema, overrides)
	require.NotNil(t, result)

	// Should NOT have AllOf (merged directly)
	assert.Empty(t, result.AllOf)

	// Should have the property merged into base
	require.Contains(t, result.Properties, "data")
	dataProperty := result.Properties["data"]
	assert.Equal(t, "#/definitions/Account", dataProperty.Ref.String())
}

func TestBuildAllOfSchema_NoOverridesReturnsBase(t *testing.T) {
	// No overrides should return base schema unchanged
	baseSchema := &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Ref: spec.MustCreateRef("#/definitions/Response"),
		},
	}

	result := buildAllOfSchema(baseSchema, nil)
	require.NotNil(t, result)

	// Should be identical to base (no AllOf)
	assert.Empty(t, result.AllOf)
	assert.Equal(t, "#/definitions/Response", result.Ref.String())
}

func TestBuildAllOfSchema_EmptyOverridesReturnsBase(t *testing.T) {
	// Empty overrides map should return base schema unchanged
	baseSchema := &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Ref: spec.MustCreateRef("#/definitions/Response"),
		},
	}

	result := buildAllOfSchema(baseSchema, map[string]spec.Schema{})
	require.NotNil(t, result)

	// Should be identical to base (no AllOf)
	assert.Empty(t, result.AllOf)
	assert.Equal(t, "#/definitions/Response", result.Ref.String())
}

func TestBuildAllOfSchema_NilBaseReturnsEmpty(t *testing.T) {
	// Nil base schema should return empty schema
	overrides := map[string]spec.Schema{
		"data": {SchemaProps: spec.SchemaProps{Type: []string{"string"}}},
	}

	result := buildAllOfSchema(nil, overrides)
	require.NotNil(t, result)
	assert.Empty(t, result.AllOf)
	assert.Empty(t, result.Ref.String())
}

// ============================================================================
// Integration Tests - Full parsing and building
// ============================================================================

func TestAllOfIntegration_SimpleResponseWithData(t *testing.T) {
	// Integration: Parse and build "response.SuccessResponse{data=account.Account}"
	baseType, overrides, err := parseCombinedType("response.SuccessResponse{data=account.Account}")
	require.NoError(t, err)
	assert.Equal(t, "response.SuccessResponse", baseType)
	assert.Equal(t, map[string]string{"data": "account.Account"}, overrides)

	// Simulate building the schema
	baseSchema := &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Ref: spec.MustCreateRef("#/definitions/response.SuccessResponse"),
		},
	}
	overrideSchemas := map[string]spec.Schema{
		"data": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/account.Account"),
			},
		},
	}

	result := buildAllOfSchema(baseSchema, overrideSchemas)
	require.NotNil(t, result)
	require.Len(t, result.AllOf, 2)
	assert.Equal(t, "#/definitions/response.SuccessResponse", result.AllOf[0].Ref.String())
	dataProperty := result.AllOf[1].Properties["data"]
	assert.Equal(t, "#/definitions/account.Account", dataProperty.Ref.String())
}

func TestAllOfIntegration_ArrayDataType(t *testing.T) {
	// Integration: Parse and build "response.SuccessResponse{data=[]account.Account}"
	baseType, overrides, err := parseCombinedType("response.SuccessResponse{data=[]account.Account}")
	require.NoError(t, err)
	assert.Equal(t, "response.SuccessResponse", baseType)
	assert.Equal(t, map[string]string{"data": "[]account.Account"}, overrides)

	// Would need type resolver to build full schema with array
	// Just verify parsing works correctly
}

func TestAllOfIntegration_NestedCombinedType(t *testing.T) {
	// Integration: Parse nested "Response{data=Inner{field=Type}}"
	baseType, overrides, err := parseCombinedType("Response{data=Inner{field=Type}}")
	require.NoError(t, err)
	assert.Equal(t, "Response", baseType)
	assert.Equal(t, map[string]string{"data": "Inner{field=Type}"}, overrides)

	// The nested "Inner{field=Type}" would be parsed recursively
}

func TestAllOfIntegration_NoOverridesSimpleRef(t *testing.T) {
	// Integration: Simple type with no overrides
	baseType, overrides, err := parseCombinedType("response.SuccessResponse")
	require.NoError(t, err)
	assert.Equal(t, "response.SuccessResponse", baseType)
	assert.Empty(t, overrides)

	baseSchema := &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Ref: spec.MustCreateRef("#/definitions/response.SuccessResponse"),
		},
	}

	result := buildAllOfSchema(baseSchema, nil)
	require.NotNil(t, result)
	assert.Empty(t, result.AllOf)
	assert.Equal(t, "#/definitions/response.SuccessResponse", result.Ref.String())
}
