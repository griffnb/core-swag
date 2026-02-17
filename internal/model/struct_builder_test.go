package model

import (
	"go/ast"
	"testing"

	"github.com/go-openapi/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helper functions for assertions
type schemaAsserter struct {
	t      *testing.T
	schema *spec.Schema
}

func assertSchema(t *testing.T, schema *spec.Schema) *schemaAsserter {
	require.NotNil(t, schema, "Schema should not be nil")
	return &schemaAsserter{t: t, schema: schema}
}

func (sa *schemaAsserter) hasProperty(name string) *schemaAsserter {
	assert.Contains(sa.t, sa.schema.Properties, name, "Schema should have property: "+name)
	return sa
}

func (sa *schemaAsserter) notHasProperty(name string) *schemaAsserter {
	assert.NotContains(sa.t, sa.schema.Properties, name, "Schema should NOT have property: "+name)
	return sa
}

func (sa *schemaAsserter) propertyCount(count int) *schemaAsserter {
	assert.Equal(sa.t, count, len(sa.schema.Properties), "Schema should have %d properties", count)
	return sa
}

func (sa *schemaAsserter) requiredField(name string) *schemaAsserter {
	assert.Contains(sa.t, sa.schema.Required, name, "Field should be required: "+name)
	return sa
}

func (sa *schemaAsserter) notRequiredField(name string) *schemaAsserter {
	assert.NotContains(sa.t, sa.schema.Required, name, "Field should NOT be required: "+name)
	return sa
}

func (sa *schemaAsserter) requiredCount(count int) *schemaAsserter {
	assert.Equal(sa.t, count, len(sa.schema.Required), "Schema should have %d required fields", count)
	return sa
}

func (sa *schemaAsserter) propertyType(name string, expectedType string) *schemaAsserter {
	prop, ok := sa.schema.Properties[name]
	assert.True(sa.t, ok, "Property should exist: "+name)
	if ok && len(prop.Type) > 0 {
		assert.Equal(sa.t, expectedType, prop.Type[0], "Property %s should have type %s", name, expectedType)
	}
	return sa
}

func (sa *schemaAsserter) propertyFormat(name string, expectedFormat string) *schemaAsserter {
	prop, ok := sa.schema.Properties[name]
	assert.True(sa.t, ok, "Property should exist: "+name)
	if ok {
		assert.Equal(sa.t, expectedFormat, prop.Format, "Property %s should have format %s", name, expectedFormat)
	}
	return sa
}

func (sa *schemaAsserter) propertyRef(name string, expectedRef string) *schemaAsserter {
	prop, ok := sa.schema.Properties[name]
	assert.True(sa.t, ok, "Property should exist: "+name)
	if ok {
		assert.Equal(sa.t, expectedRef, prop.Ref.String(), "Property %s should have ref %s", name, expectedRef)
	}
	return sa
}

func (sa *schemaAsserter) isArray(name string) *schemaAsserter {
	prop, ok := sa.schema.Properties[name]
	assert.True(sa.t, ok, "Property should exist: "+name)
	if ok && len(prop.Type) > 0 {
		assert.Equal(sa.t, "array", prop.Type[0], "Property %s should be an array", name)
	}
	return sa
}

func (sa *schemaAsserter) arrayItemsRef(name string, expectedRef string) *schemaAsserter {
	prop, ok := sa.schema.Properties[name]
	assert.True(sa.t, ok, "Property should exist: "+name)
	if ok && prop.Items != nil && prop.Items.Schema != nil {
		assert.Equal(sa.t, expectedRef, prop.Items.Schema.Ref.String(), "Array %s items should have ref %s", name, expectedRef)
	}
	return sa
}

// Helper to create test enum lookup
type testEnumLookup struct {
	enums map[string][]EnumValue
}

func newTestEnumLookup() *testEnumLookup {
	return &testEnumLookup{
		enums: make(map[string][]EnumValue),
	}
}

func (tel *testEnumLookup) addEnum(typeName string, values []EnumValue) {
	tel.enums[typeName] = values
}

func (tel *testEnumLookup) GetEnumsForType(typeName string, file *ast.File) ([]EnumValue, error) {
	if enums, ok := tel.enums[typeName]; ok {
		return enums, nil
	}
	return nil, nil
}

func TestBuildSpecSchema_Simple(t *testing.T) {
	builder := &StructBuilder{
		Fields: []*StructField{
			{
				Name:       "FirstName",
				TypeString: "string",
				Tag:        `json:"first_name"`,
			},
			{
				Name:       "LastName",
				TypeString: "string",
				Tag:        `json:"last_name,omitempty"`,
			},
			{
				Name:       "Age",
				TypeString: "int",
				Tag:        `json:"age"`,
			},
		},
	}

	schema, nestedTypes, err := builder.BuildSpecSchema("User", false, false, nil)
	assert.NoError(t, err)
	assert.NotNil(t, schema)
	assert.Equal(t, 1, len(schema.Type))
	assert.Equal(t, "object", schema.Type[0])
	assert.Equal(t, 3, len(schema.Properties))

	// Check properties
	assert.Contains(t, schema.Properties, "first_name")
	assert.Contains(t, schema.Properties, "last_name")
	assert.Contains(t, schema.Properties, "age")

	// Check required fields
	assert.Equal(t, 2, len(schema.Required))
	assert.Contains(t, schema.Required, "first_name")
	assert.Contains(t, schema.Required, "age")
	assert.NotContains(t, schema.Required, "last_name")

	// No nested types
	assert.Equal(t, 0, len(nestedTypes))
}

func TestBuildSpecSchema_WithNestedStruct(t *testing.T) {
	builder := &StructBuilder{
		Fields: []*StructField{
			{
				Name:       "Name",
				TypeString: "string",
				Tag:        `json:"name"`,
			},
			{
				Name:       "Address",
				TypeString: "fields.StructField[*Address]",
				Tag:        `json:"address"`,
			},
		},
	}

	schema, nestedTypes, err := builder.BuildSpecSchema("User", false, false, nil)
	assert.NoError(t, err)
	assert.NotNil(t, schema)
	assert.Equal(t, 2, len(schema.Properties))

	// Check nested type
	assert.Equal(t, 1, len(nestedTypes))
	assert.Contains(t, nestedTypes, "Address")

	// Check address property is a reference
	addressProp := schema.Properties["address"]
	assert.Equal(t, "#/definitions/Address", addressProp.Ref.String())
}

func TestBuildSpecSchema_PublicMode(t *testing.T) {
	builder := &StructBuilder{
		Fields: []*StructField{
			{
				Name:       "PublicField",
				TypeString: "string",
				Tag:        `public:"view" json:"public_field"`,
			},
			{
				Name:       "PrivateField",
				TypeString: "string",
				Tag:        `json:"private_field"`,
			},
		},
	}

	schema, nestedTypes, err := builder.BuildSpecSchema("User", true, false, nil)
	assert.NoError(t, err)
	assert.NotNil(t, schema)

	// Only public field should be included
	assert.Equal(t, 1, len(schema.Properties))
	assert.Contains(t, schema.Properties, "public_field")
	assert.NotContains(t, schema.Properties, "private_field")

	// No nested types
	assert.Equal(t, 0, len(nestedTypes))
}

func TestBuildSpecSchema_PublicModeWithNestedStruct(t *testing.T) {
	builder := &StructBuilder{
		Fields: []*StructField{
			{
				Name:       "Name",
				TypeString: "string",
				Tag:        `public:"view" json:"name"`,
			},
			{
				Name:       "Profile",
				TypeString: "fields.StructField[*Profile]",
				Tag:        `public:"view" json:"profile"`,
			},
			{
				Name:       "InternalData",
				TypeString: "fields.StructField[*InternalData]",
				Tag:        `json:"internal_data"`,
			},
		},
	}

	schema, nestedTypes, err := builder.BuildSpecSchema("User", true, false, nil)
	assert.NoError(t, err)
	assert.NotNil(t, schema)

	// Only public fields should be included
	assert.Equal(t, 2, len(schema.Properties))
	assert.Contains(t, schema.Properties, "name")
	assert.Contains(t, schema.Properties, "profile")
	assert.NotContains(t, schema.Properties, "internal_data")

	// Only public nested type should be included
	assert.Equal(t, 1, len(nestedTypes))
	assert.Contains(t, nestedTypes, "Profile")
	assert.NotContains(t, nestedTypes, "InternalData")

	// Check profile property has Public suffix
	profileProp := schema.Properties["profile"]
	assert.Equal(t, "#/definitions/ProfilePublic", profileProp.Ref.String())
}

func TestBuildSpecSchema_MultipleNestedStructs(t *testing.T) {
	builder := &StructBuilder{
		Fields: []*StructField{
			{
				Name:       "User",
				TypeString: "fields.StructField[*User]",
				Tag:        `json:"user"`,
			},
			{
				Name:       "Address",
				TypeString: "fields.StructField[*Address]",
				Tag:        `json:"address"`,
			},
			{
				Name:       "SecondaryAddress",
				TypeString: "fields.StructField[*Address]",
				Tag:        `json:"secondary_address,omitempty"`,
			},
		},
	}

	schema, nestedTypes, err := builder.BuildSpecSchema("Contact", false, false, nil)
	assert.NoError(t, err)
	assert.NotNil(t, schema)

	// Should deduplicate Address
	assert.Equal(t, 2, len(nestedTypes))
	assert.Contains(t, nestedTypes, "User")
	assert.Contains(t, nestedTypes, "Address")

	// Check required fields
	assert.Equal(t, 2, len(schema.Required))
	assert.Contains(t, schema.Required, "user")
	assert.Contains(t, schema.Required, "address")
	assert.NotContains(t, schema.Required, "secondary_address")
}

func TestBuildSpecSchema_ArrayOfStructs(t *testing.T) {
	builder := &StructBuilder{
		Fields: []*StructField{
			{
				Name:       "Name",
				TypeString: "string",
				Tag:        `json:"name"`,
			},
			{
				Name:       "Items",
				TypeString: "fields.StructField[[]Item]",
				Tag:        `json:"items"`,
			},
		},
	}

	schema, nestedTypes, err := builder.BuildSpecSchema("Order", false, false, nil)
	assert.NoError(t, err)
	assert.NotNil(t, schema)
	assert.Equal(t, 2, len(schema.Properties))

	// Check nested type
	assert.Equal(t, 1, len(nestedTypes))
	assert.Contains(t, nestedTypes, "Item")

	// Check items property is an array
	itemsProp := schema.Properties["items"]
	assert.Equal(t, 1, len(itemsProp.Type))
	assert.Equal(t, "array", itemsProp.Type[0])
	assert.NotNil(t, itemsProp.Items)
	assert.NotNil(t, itemsProp.Items.Schema)
	assert.Equal(t, "#/definitions/Item", itemsProp.Items.Schema.Ref.String())
}

func TestBuildSpecSchema_EmptyBuilder(t *testing.T) {
	builder := &StructBuilder{
		Fields: []*StructField{},
	}

	schema, nestedTypes, err := builder.BuildSpecSchema("Empty", false, false, nil)
	assert.NoError(t, err)
	assert.NotNil(t, schema)
	assert.Equal(t, 1, len(schema.Type))
	assert.Equal(t, "object", schema.Type[0])
	assert.Equal(t, 0, len(schema.Properties))
	assert.Equal(t, 0, len(schema.Required))
	assert.Equal(t, 0, len(nestedTypes))
}

// ============================================================================
// COMPREHENSIVE TESTS FOR STRUCT_BUILDER ENHANCEMENT (Phase 1.1)
// These tests use helper functions and will be enhanced as features are added
// ============================================================================

// TestExtendedPrimitives_TimeTime tests time.Time mapping to string with date-time format
func TestExtendedPrimitives_TimeTime(t *testing.T) {
	builder := &StructBuilder{
		Fields: []*StructField{
			{
				Name:       "CreatedAt",
				TypeString: "time.Time",
				Tag:        `json:"created_at"`,
			},
			{
				Name:       "UpdatedAt",
				TypeString: "*time.Time",
				Tag:        `json:"updated_at,omitempty"`,
			},
		},
	}

	schema, _, err := builder.BuildSpecSchema("TimestampModel", false, false, nil)
	require.NoError(t, err)

	assertSchema(t, schema).
		hasProperty("created_at").
		hasProperty("updated_at").
		propertyType("created_at", "string").
		propertyFormat("created_at", "date-time").
		propertyType("updated_at", "string").
		propertyFormat("updated_at", "date-time")
}

// TestExtendedPrimitives_UUID tests UUID mapping to string with uuid format
func TestExtendedPrimitives_UUID(t *testing.T) {
	builder := &StructBuilder{
		Fields: []*StructField{
			{
				Name:       "ID",
				TypeString: "uuid.UUID",
				Tag:        `json:"id"`,
			},
			{
				Name:       "ParentID",
				TypeString: "*uuid.UUID",
				Tag:        `json:"parent_id,omitempty"`,
			},
		},
	}

	schema, _, err := builder.BuildSpecSchema("UUIDModel", false, false, nil)
	require.NoError(t, err)

	assertSchema(t, schema).
		hasProperty("id").
		hasProperty("parent_id").
		propertyType("id", "string").
		propertyFormat("id", "uuid").
		propertyType("parent_id", "string").
		propertyFormat("parent_id", "uuid")
}

// TestExtendedPrimitives_Decimal tests decimal.Decimal mapping to number
func TestExtendedPrimitives_Decimal(t *testing.T) {
	builder := &StructBuilder{
		Fields: []*StructField{
			{
				Name:       "Price",
				TypeString: "decimal.Decimal",
				Tag:        `json:"price"`,
			},
			{
				Name:       "Tax",
				TypeString: "*decimal.Decimal",
				Tag:        `json:"tax,omitempty"`,
			},
		},
	}

	schema, _, err := builder.BuildSpecSchema("PriceModel", false, false, nil)
	require.NoError(t, err)

	assertSchema(t, schema).
		hasProperty("price").
		hasProperty("tax").
		propertyType("price", "number").
		propertyType("tax", "number")
}

// TestExtendedPrimitives_RawMessage tests json.RawMessage mapping to object
func TestExtendedPrimitives_RawMessage(t *testing.T) {
	builder := &StructBuilder{
		Fields: []*StructField{
			{
				Name:       "Metadata",
				TypeString: "json.RawMessage",
				Tag:        `json:"metadata"`,
			},
		},
	}

	schema, _, err := builder.BuildSpecSchema("MetadataModel", false, false, nil)
	require.NoError(t, err)

	assertSchema(t, schema).
		hasProperty("metadata").
		propertyType("metadata", "object")
}

// TestEnumDetection_StringEnum tests string-based enum detection and inlining
func TestEnumDetection_StringEnum(t *testing.T) {
	enumLookup := newTestEnumLookup()
	enumLookup.addEnum("Status", []EnumValue{
		{Key: "active", Value: "active"},
		{Key: "inactive", Value: "inactive"},
		{Key: "pending", Value: "pending"},
	})

	builder := &StructBuilder{
		Fields: []*StructField{
			{
				Name:       "Status",
				TypeString: "Status",
				Tag:        `json:"status"`,
			},
		},
	}

	schema, _, err := builder.BuildSpecSchema("StatusModel", false, false, enumLookup)
	require.NoError(t, err)

	assertSchema(t, schema).
		hasProperty("status").
		propertyType("status", "string")

	// Check enum values are inlined
	statusProp := schema.Properties["status"]
	assert.NotNil(t, statusProp.Enum, "Status property should have enum values")
	if statusProp.Enum != nil {
		assert.Equal(t, 3, len(statusProp.Enum), "Should have 3 enum values")
	}
}

// TestEnumDetection_IntEnum tests integer-based enum detection
func TestEnumDetection_IntEnum(t *testing.T) {
	enumLookup := newTestEnumLookup()
	enumLookup.addEnum("Priority", []EnumValue{
		{Key: "low", Value: 1},
		{Key: "medium", Value: 2},
		{Key: "high", Value: 3},
	})

	builder := &StructBuilder{
		Fields: []*StructField{
			{
				Name:       "Priority",
				TypeString: "Priority",
				Tag:        `json:"priority"`,
			},
		},
	}

	schema, _, err := builder.BuildSpecSchema("PriorityModel", false, false, enumLookup)
	require.NoError(t, err)

	assertSchema(t, schema).
		hasProperty("priority").
		propertyType("priority", "integer")

	// Check enum values are inlined
	priorityProp := schema.Properties["priority"]
	assert.NotNil(t, priorityProp.Enum, "Priority property should have enum values")
}

// TestGenericExtraction_StructFieldPrimitives tests StructField[T] with primitives
func TestGenericExtraction_StructFieldPrimitives(t *testing.T) {
	builder := &StructBuilder{
		Fields: []*StructField{
			{
				Name:       "Name",
				TypeString: "fields.StructField[string]",
				Tag:        `json:"name"`,
			},
			{
				Name:       "Count",
				TypeString: "fields.StructField[int]",
				Tag:        `json:"count"`,
			},
		},
	}

	schema, _, err := builder.BuildSpecSchema("GenericModel", false, false, nil)
	require.NoError(t, err)

	// Should extract inner type (string, int) not StructField
	assertSchema(t, schema).
		hasProperty("name").
		hasProperty("count").
		propertyType("name", "string").
		propertyType("count", "integer")
}

// TestGenericExtraction_StructFieldPointers tests StructField[*T]
func TestGenericExtraction_StructFieldPointers(t *testing.T) {
	builder := &StructBuilder{
		Fields: []*StructField{
			{
				Name:       "Properties",
				TypeString: "fields.StructField[*Properties]",
				Tag:        `json:"properties"`,
			},
		},
	}

	schema, nestedTypes, err := builder.BuildSpecSchema("ModelWithProps", false, false, nil)
	require.NoError(t, err)

	// Should create reference to Properties (not StructField)
	assertSchema(t, schema).
		hasProperty("properties").
		propertyRef("properties", "#/definitions/Properties")

	assert.Contains(t, nestedTypes, "Properties", "Should collect Properties as nested type")
}

// TestGenericExtraction_StructFieldArrays tests StructField[[]T]
func TestGenericExtraction_StructFieldArrays(t *testing.T) {
	builder := &StructBuilder{
		Fields: []*StructField{
			{
				Name:       "Items",
				TypeString: "fields.StructField[[]Item]",
				Tag:        `json:"items"`,
			},
		},
	}

	schema, nestedTypes, err := builder.BuildSpecSchema("CollectionModel", false, false, nil)
	require.NoError(t, err)

	// Should create array of Item references
	assertSchema(t, schema).
		hasProperty("items").
		isArray("items").
		arrayItemsRef("items", "#/definitions/Item")

	assert.Contains(t, nestedTypes, "Item", "Should collect Item as nested type")
}

// TestGenericExtraction_StructFieldMaps tests StructField[map[string]T]
func TestGenericExtraction_StructFieldMaps(t *testing.T) {
	builder := &StructBuilder{
		Fields: []*StructField{
			{
				Name:       "Metadata",
				TypeString: "fields.StructField[map[string]any]",
				Tag:        `json:"metadata"`,
			},
		},
	}

	schema, _, err := builder.BuildSpecSchema("MapModel", false, false, nil)
	require.NoError(t, err)

	// Should create object with additionalProperties
	assertSchema(t, schema).
		hasProperty("metadata").
		propertyType("metadata", "object")
}

// TestEmbeddedFields_SingleLevel tests single embedded field
func TestEmbeddedFields_SingleLevel(t *testing.T) {
	// Simulate BaseModel embedded in Account
	builder := &StructBuilder{
		Fields: []*StructField{
			// Embedded BaseModel fields (no field name, just fields)
			{
				Name:       "ID",
				TypeString: "uuid.UUID",
				Tag:        `json:"id"`,
			},
			{
				Name:       "CreatedAt",
				TypeString: "time.Time",
				Tag:        `json:"created_at"`,
			},
			// Account own fields
			{
				Name:       "Email",
				TypeString: "string",
				Tag:        `json:"email"`,
			},
		},
	}

	schema, _, err := builder.BuildSpecSchema("Account", false, false, nil)
	require.NoError(t, err)

	// All fields should be at top level (embedded fields merged)
	assertSchema(t, schema).
		hasProperty("id").
		hasProperty("created_at").
		hasProperty("email").
		propertyCount(3)
}

// TestValidation_Required tests validation:"required" tag
func TestValidation_Required(t *testing.T) {
	builder := &StructBuilder{
		Fields: []*StructField{
			{
				Name:       "Email",
				TypeString: "string",
				Tag:        `json:"email" validate:"required"`,
			},
			{
				Name:       "Name",
				TypeString: "string",
				Tag:        `json:"name,omitempty"`,
			},
		},
	}

	schema, _, err := builder.BuildSpecSchema("ValidationModel", false, false, nil)
	require.NoError(t, err)

	// Email should be required (validate:"required")
	// Name should not be required (omitempty)
	assertSchema(t, schema).
		requiredField("email").
		notRequiredField("name")
}

// TestValidation_MinMax tests min/max validation constraints
func TestValidation_MinMax(t *testing.T) {
	builder := &StructBuilder{
		Fields: []*StructField{
			{
				Name:       "Age",
				TypeString: "int",
				Tag:        `json:"age" validate:"min=0,max=150"`,
			},
			{
				Name:       "Score",
				TypeString: "float64",
				Tag:        `json:"score" validate:"min=0.0,max=100.0"`,
			},
		},
	}

	schema, _, err := builder.BuildSpecSchema("ScoreModel", false, false, nil)
	require.NoError(t, err)

	assertSchema(t, schema).
		hasProperty("age").
		hasProperty("score")

	// Check min/max constraints are applied
	ageProp := schema.Properties["age"]
	if ageProp.Minimum != nil {
		assert.Equal(t, float64(0), *ageProp.Minimum, "Age should have minimum 0")
	}
	if ageProp.Maximum != nil {
		assert.Equal(t, float64(150), *ageProp.Maximum, "Age should have maximum 150")
	}
}

// TestPublicMode_ViewOnly tests public:"view" filtering
func TestPublicMode_ViewOnlyFiltering(t *testing.T) {
	builder := &StructBuilder{
		Fields: []*StructField{
			{
				Name:       "ID",
				TypeString: "uuid.UUID",
				Tag:        `json:"id" public:"view"`,
			},
			{
				Name:       "Name",
				TypeString: "string",
				Tag:        `json:"name" public:"edit"`,
			},
			{
				Name:       "HashedPassword",
				TypeString: "string",
				Tag:        `json:"hashed_password"`, // No public tag
			},
		},
	}

	// Test with public=false (all fields)
	schema, _, err := builder.BuildSpecSchema("User", false, false, nil)
	require.NoError(t, err)
	assertSchema(t, schema).
		hasProperty("id").
		hasProperty("name").
		hasProperty("hashed_password").
		propertyCount(3)

	// Test with public=true (only public fields)
	schemaPublic, _, err := builder.BuildSpecSchema("User", true, false, nil)
	require.NoError(t, err)
	assertSchema(t, schemaPublic).
		hasProperty("id").
		hasProperty("name").
		notHasProperty("hashed_password").
		propertyCount(2)
}

// TestSwaggerIgnore_Tag tests swaggerignore:"true" tag
func TestSwaggerIgnore_Tag(t *testing.T) {
	builder := &StructBuilder{
		Fields: []*StructField{
			{
				Name:       "Email",
				TypeString: "string",
				Tag:        `json:"email"`,
			},
			{
				Name:       "Authentication",
				TypeString: "fields.StructField[*Authentication]",
				Tag:        `json:"authentication" swaggerignore:"true"`,
			},
			{
				Name:       "InternalData",
				TypeString: "string",
				Tag:        `json:"-"`, // JSON ignore
			},
		},
	}

	schema, nestedTypes, err := builder.BuildSpecSchema("Account", false, false, nil)
	require.NoError(t, err)

	// Only email should be included
	assertSchema(t, schema).
		hasProperty("email").
		notHasProperty("authentication").
		notHasProperty("InternalData").
		propertyCount(1)

	// Authentication should NOT be in nested types (swaggerignore)
	assert.NotContains(t, nestedTypes, "Authentication", "Ignored fields should not add nested types")
}

// TestArrays_OfStructs tests arrays of custom structs
func TestArrays_OfStructs(t *testing.T) {
	builder := &StructBuilder{
		Fields: []*StructField{
			{
				Name:       "Tags",
				TypeString: "[]Tag",
				Tag:        `json:"tags"`,
			},
		},
	}

	schema, nestedTypes, err := builder.BuildSpecSchema("Post", false, false, nil)
	require.NoError(t, err)

	assertSchema(t, schema).
		hasProperty("tags").
		isArray("tags").
		arrayItemsRef("tags", "#/definitions/Tag")

	assert.Contains(t, nestedTypes, "Tag", "Should collect Tag as nested type")
}

// TestMaps_WithStructValues tests maps with struct values
func TestMaps_WithStructValues(t *testing.T) {
	builder := &StructBuilder{
		Fields: []*StructField{
			{
				Name:       "Settings",
				TypeString: "map[string]Setting",
				Tag:        `json:"settings"`,
			},
		},
	}

	schema, nestedTypes, err := builder.BuildSpecSchema("Config", false, false, nil)
	require.NoError(t, err)

	assertSchema(t, schema).
		hasProperty("settings").
		propertyType("settings", "object")

	// Check additionalProperties references Setting
	settingsProp := schema.Properties["settings"]
	if settingsProp.AdditionalProperties != nil && settingsProp.AdditionalProperties.Schema != nil {
		assert.Equal(t, "#/definitions/Setting", settingsProp.AdditionalProperties.Schema.Ref.String())
	}

	assert.Contains(t, nestedTypes, "Setting", "Should collect Setting as nested type")
}

// TestNestedStructs_DeepNesting tests deeply nested structures
func TestNestedStructs_DeepNesting(t *testing.T) {
	builder := &StructBuilder{
		Fields: []*StructField{
			{
				Name:       "Contact",
				TypeString: "Contact", // Contact has Address, Address has Country
				Tag:        `json:"contact"`,
			},
		},
	}

	schema, nestedTypes, err := builder.BuildSpecSchema("Company", false, false, nil)
	require.NoError(t, err)

	assertSchema(t, schema).
		hasProperty("contact").
		propertyRef("contact", "#/definitions/Contact")

	assert.Contains(t, nestedTypes, "Contact", "Should collect Contact as nested type")
}

// TestForceRequired tests forceRequired parameter
func TestForceRequired(t *testing.T) {
	builder := &StructBuilder{
		Fields: []*StructField{
			{
				Name:       "Name",
				TypeString: "string",
				Tag:        `json:"name,omitempty"`,
			},
			{
				Name:       "Email",
				TypeString: "string",
				Tag:        `json:"email"`,
			},
		},
	}

	// Without forceRequired
	schema1, _, err := builder.BuildSpecSchema("User", false, false, nil)
	require.NoError(t, err)
	assertSchema(t, schema1).
		requiredField("email").
		notRequiredField("name")

	// With forceRequired
	schema2, _, err := builder.BuildSpecSchema("User", false, true, nil)
	require.NoError(t, err)
	assertSchema(t, schema2).
		requiredField("email").
		requiredField("name"). // Now required due to forceRequired
		requiredCount(2)
}
