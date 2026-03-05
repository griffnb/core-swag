package model

import (
	"fmt"
	"go/ast"
	"testing"

	"github.com/go-openapi/spec"
	"github.com/stretchr/testify/assert"
)

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

func TestIsGeneric(t *testing.T) {
	tests := []struct {
		name  string
		field *StructField
		want  bool
	}{
		{name: "StructField generic", field: &StructField{TypeString: "fields.StructField[*User]"}, want: true},
		{name: "IntConstantField generic", field: &StructField{TypeString: "fields.IntConstantField[constants.Role]"}, want: true},
		{name: "StringField generic", field: &StructField{TypeString: "fields.StringField[constants.Key]"}, want: true},
		{name: "Field generic", field: &StructField{TypeString: "fields.Field[SomeType]"}, want: true},
		{name: "plain string type", field: &StructField{TypeString: "string"}, want: false},
		{name: "struct type", field: &StructField{TypeString: "account.Properties"}, want: false},
		{name: "empty", field: &StructField{}, want: false},
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
		{name: "simple type", field: &StructField{TypeString: "fields.StructField[User]"}, want: "User"},
		{name: "pointer type", field: &StructField{TypeString: "fields.StructField[*User]"}, want: "User"},
		{name: "package qualified", field: &StructField{TypeString: "fields.StructField[*billing_plan.FeatureSet]"}, want: "billing_plan.FeatureSet"},
		{name: "map type", field: &StructField{TypeString: "fields.StructField[map[string]User]"}, want: "map[string]User"},
		{name: "not generic", field: &StructField{TypeString: "string"}, wantErr: true},
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

func TestToSpecSchema_PrimitiveTypes(t *testing.T) {
	tests := []struct {
		name         string
		field        *StructField
		public       bool
		wantPropName string
		wantType     []string
		wantFormat   string
		wantRequired bool
		wantNested   int
	}{
		{
			name: "string field with json tag",
			field: &StructField{
				Name:       "FirstName",
				TypeString: "string",
				Tag:        `json:"first_name"`,
			},
			public:       false,
			wantPropName: "first_name",
			wantType:     []string{"string"},
			wantRequired: true,
			wantNested:   0,
		},
		{
			name: "int field with omitempty",
			field: &StructField{
				Name:       "Age",
				TypeString: "int",
				Tag:        `json:"age,omitempty"`,
			},
			public:       false,
			wantPropName: "age",
			wantType:     []string{"integer"},
			wantRequired: false,
			wantNested:   0,
		},
		{
			name: "int64 field",
			field: &StructField{
				Name:       "ID",
				TypeString: "int64",
				Tag:        `json:"id"`,
			},
			public:       false,
			wantPropName: "id",
			wantType:     []string{"integer"},
			wantFormat:   "int64",
			wantRequired: true,
			wantNested:   0,
		},
		{
			name: "bool field",
			field: &StructField{
				Name:       "Active",
				TypeString: "bool",
				Tag:        `json:"active"`,
			},
			public:       false,
			wantPropName: "active",
			wantType:     []string{"boolean"},
			wantRequired: true,
			wantNested:   0,
		},
		{
			name: "float64 field",
			field: &StructField{
				Name:       "Price",
				TypeString: "float64",
				Tag:        `json:"price"`,
			},
			public:       false,
			wantPropName: "price",
			wantType:     []string{"number"},
			wantFormat:   "double",
			wantRequired: true,
			wantNested:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			propName, schema, required, nestedTypes, err := tt.field.ToSpecSchema(tt.public, false, nil)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantPropName, propName)
			assert.Equal(t, tt.wantRequired, required)
			assert.Equal(t, tt.wantNested, len(nestedTypes))
			if schema != nil {
				assert.Equal(t, len(tt.wantType), len(schema.Type))
				if len(tt.wantType) > 0 {
					assert.Equal(t, tt.wantType[0], schema.Type[0])
				}
				if tt.wantFormat != "" {
					assert.Equal(t, tt.wantFormat, schema.Format)
				}
			}
		})
	}
}

func TestToSpecSchema_StructField_Simple(t *testing.T) {
	field := &StructField{
		Name:       "Properties",
		TypeString: "fields.StructField[*Properties]",
		Tag:        `json:"properties"`,
	}

	propName, schema, required, nestedTypes, err := field.ToSpecSchema(false, false, nil)
	assert.NoError(t, err)
	assert.Equal(t, "properties", propName)
	assert.True(t, required)
	assert.Equal(t, 1, len(nestedTypes))
	assert.Equal(t, "Properties", nestedTypes[0])
	assert.NotNil(t, schema)
	assert.Equal(t, "#/definitions/Properties", schema.Ref.String())
}

func TestToSpecSchema_StructField_Public(t *testing.T) {
	field := &StructField{
		Name:       "User",
		TypeString: "fields.StructField[*User]",
		Tag:        `public:"view" json:"user"`,
	}

	propName, schema, required, nestedTypes, err := field.ToSpecSchema(true, false, nil)
	assert.NoError(t, err)
	assert.Equal(t, "user", propName)
	assert.True(t, required)
	assert.Equal(t, 1, len(nestedTypes))
	assert.Equal(t, "UserPublic", nestedTypes[0]) // Should include Public suffix when public=true
	assert.NotNil(t, schema)
	assert.Equal(t, "#/definitions/UserPublic", schema.Ref.String())
}

func TestToSpecSchema_StructField_NotPublic(t *testing.T) {
	field := &StructField{
		Name:       "InternalData",
		TypeString: "fields.StructField[*InternalData]",
		Tag:        `json:"internal_data"`,
	}

	// When public=true but field has no public tag, should return nil
	propName, schema, required, nestedTypes, err := field.ToSpecSchema(true, false, nil)
	assert.NoError(t, err)
	assert.Equal(t, "", propName)
	assert.Nil(t, schema)
	assert.False(t, required)
	assert.Nil(t, nestedTypes)
}

func TestExtractTypeParameter(t *testing.T) {
	tests := []struct {
		name    string
		typeStr string
		want    string
		wantErr bool
	}{
		{
			name:    "simple type",
			typeStr: "fields.StructField[User]",
			want:    "User",
		},
		{
			name:    "pointer type",
			typeStr: "fields.StructField[*User]",
			want:    "User",
		},
		{
			name:    "package qualified type",
			typeStr: "fields.StructField[*billing_plan.FeatureSet]",
			want:    "billing_plan.FeatureSet",
		},
		{
			name:    "array type",
			typeStr: "fields.StructField[[]User]",
			want:    "[]User",
		},
		{
			name:    "map type",
			typeStr: "fields.StructField[map[string]User]",
			want:    "map[string]User",
		},
		{
			name:    "complex nested type",
			typeStr: "fields.StructField[map[string][]User]",
			want:    "map[string][]User",
		},
		{
			name:    "invalid - no bracket",
			typeStr: "fields.StructField",
			wantErr: true,
		},
		{
			name:    "invalid - mismatched brackets",
			typeStr: "fields.StructField[map[string]User",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := (&StructField{TypeString: tt.typeStr}).GenericTypeArg()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestBuildSchemaForType(t *testing.T) {
	tests := []struct {
		name       string
		typeStr    string
		public     bool
		wantType   []string
		wantRef    string
		wantNested []string
	}{
		{
			name:     "string",
			typeStr:  "string",
			public:   false,
			wantType: []string{"string"},
		},
		{
			name:     "int",
			typeStr:  "int",
			public:   false,
			wantType: []string{"integer"},
		},
		{
			name:       "struct without public",
			typeStr:    "User",
			public:     false,
			wantRef:    "#/definitions/User",
			wantNested: []string{"User"},
		},
		{
			name:       "struct with public",
			typeStr:    "User",
			public:     true,
			wantRef:    "#/definitions/UserPublic",
			wantNested: []string{"UserPublic"}, // Should include Public suffix when public=true
		},
		{
			name:       "package qualified struct",
			typeStr:    "billing_plan.FeatureSet",
			public:     true,
			wantRef:    "#/definitions/billing_plan.FeatureSetPublic",
			wantNested: []string{"billing_plan.FeatureSetPublic"}, // Should include Public suffix when public=true
		},
		{
			name:     "array of strings",
			typeStr:  "[]string",
			public:   false,
			wantType: []string{"array"},
		},
		{
			name:       "array of structs",
			typeStr:    "[]User",
			public:     false,
			wantType:   []string{"array"},
			wantNested: []string{"User"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, nestedTypes, err := (&StructField{TypeString: tt.typeStr}).BuildSchema(tt.public, false, nil)
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

func TestToSpecSchema_ArrayElementTypes(t *testing.T) {
	tests := []struct {
		name              string
		field             *StructField
		wantType          []string
		wantItemsType     []string
		wantItemsRef      string
		wantItemsFormat   string
	}{
		{
			name: "array of strings",
			field: &StructField{
				Name:       "Tags",
				TypeString: "[]string",
				Tag:        `json:"tags"`,
			},
			wantType:      []string{"array"},
			wantItemsType: []string{"string"},
		},
		{
			name: "array of integers",
			field: &StructField{
				Name:       "IDs",
				TypeString: "[]int",
				Tag:        `json:"ids"`,
			},
			wantType:      []string{"array"},
			wantItemsType: []string{"integer"},
		},
		{
			name: "array of int64",
			field: &StructField{
				Name:       "Timestamps",
				TypeString: "[]int64",
				Tag:        `json:"timestamps"`,
			},
			wantType:        []string{"array"},
			wantItemsType:   []string{"integer"},
			wantItemsFormat: "int64",
		},
		{
			name: "array of booleans",
			field: &StructField{
				Name:       "Flags",
				TypeString: "[]bool",
				Tag:        `json:"flags"`,
			},
			wantType:      []string{"array"},
			wantItemsType: []string{"boolean"},
		},
		{
			name: "array of floats",
			field: &StructField{
				Name:       "Prices",
				TypeString: "[]float64",
				Tag:        `json:"prices"`,
			},
			wantType:        []string{"array"},
			wantItemsType:   []string{"number"},
			wantItemsFormat: "double",
		},
		{
			name: "array of struct pointers",
			field: &StructField{
				Name:       "Users",
				TypeString: "[]*User",
				Tag:        `json:"users"`,
			},
			wantType:     []string{"array"},
			wantItemsRef: "#/definitions/User",
		},
		{
			name: "array of package qualified structs",
			field: &StructField{
				Name:       "Accounts",
				TypeString: "[]account.Account",
				Tag:        `json:"accounts"`,
			},
			wantType:     []string{"array"},
			wantItemsRef: "#/definitions/account.Account",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			propName, schema, required, nestedTypes, err := tt.field.ToSpecSchema(false, false, nil)
			assert.NoError(t, err)
			assert.NotNil(t, schema)
			assert.True(t, required, "Array fields should be required by default")

			// Check array type
			assert.Equal(t, len(tt.wantType), len(schema.Type), "Schema type length should match")
			if len(tt.wantType) > 0 && len(schema.Type) > 0 {
				assert.Equal(t, tt.wantType[0], schema.Type[0], "Schema should have type 'array'")
			}

			// Check items schema exists
			assert.NotNil(t, schema.Items, "Array schema should have items")
			assert.NotNil(t, schema.Items.Schema, "Array schema items should have schema")

			items := schema.Items.Schema

			// Check item type or reference
			if tt.wantItemsRef != "" {
				assert.Equal(t, tt.wantItemsRef, items.Ref.String(), "Array items should reference correct type")
				assert.Greater(t, len(nestedTypes), 0, "Should have nested types for struct arrays")
			} else {
				assert.Equal(t, len(tt.wantItemsType), len(items.Type), "Items type length should match")
				if len(tt.wantItemsType) > 0 && len(items.Type) > 0 {
					assert.Equal(t, tt.wantItemsType[0], items.Type[0], "Array items should have correct type")
				}
				if tt.wantItemsFormat != "" {
					assert.Equal(t, tt.wantItemsFormat, items.Format, "Array items should have correct format")
				}
			}

			// Verify property name
			assert.NotEmpty(t, propName)
		})
	}
}

func TestToSpecSchema_AnyInterfaceTypes(t *testing.T) {
	tests := []struct {
		name     string
		field    *StructField
		wantType []string
	}{
		{
			name: "any type",
			field: &StructField{
				Name:       "Data",
				TypeString: "any",
				Tag:        `json:"data"`,
			},
			wantType: nil,
		},
		{
			name: "interface{} type",
			field: &StructField{
				Name:       "Metadata",
				TypeString: "interface{}",
				Tag:        `json:"metadata"`,
			},
			wantType: nil,
		},
		{
			name: "interface{} with spaces",
			field: &StructField{
				Name:       "Options",
				TypeString: "interface {}",
				Tag:        `json:"options"`,
			},
			wantType: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			propName, schema, required, nestedTypes, err := tt.field.ToSpecSchema(false, false, nil)
			assert.NoError(t, err)
			assert.NotNil(t, schema)
			assert.True(t, required, "Any/interface fields should be required by default")

			// any/interface{} should produce empty schema (no type)
			assert.Equal(t, len(tt.wantType), len(schema.Type), "Schema type length should match")
			if len(tt.wantType) > 0 && len(schema.Type) > 0 {
				assert.Equal(t, tt.wantType[0], schema.Type[0])
			}

			// Should not generate nested types
			assert.Equal(t, 0, len(nestedTypes), "Any/interface should not generate nested types")

			// Verify property name
			assert.NotEmpty(t, propName)
		})
	}
}

func TestToSpecSchema_EnumWithUnderlyingType(t *testing.T) {
	// Mock enum lookup that returns enum values
	mockEnumLookup := &mockEnumLookup{
		enums: map[string][]EnumValue{
			"constants.Role": {
				{Key: "RoleAdmin", Value: 1, Comment: "Administrator"},
				{Key: "RoleUser", Value: 2, Comment: "Regular user"},
				{Key: "RoleGuest", Value: 3, Comment: "Guest user"},
			},
			"constants.Status": {
				{Key: "StatusActive", Value: int64(1), Comment: "Active"},
				{Key: "StatusInactive", Value: int64(0), Comment: "Inactive"},
			},
		},
	}

	tests := []struct {
		name          string
		field         *StructField
		enumLookup    TypeEnumLookup
		wantRef       string
		wantNested    string
		wantHasEnum   bool
		wantPrimitive bool
		wantType      []string
	}{
		{
			name: "enum with int underlying type creates $ref",
			field: &StructField{
				Name:       "Role",
				TypeString: "constants.Role",
				Tag:        `json:"role"`,
			},
			enumLookup:  mockEnumLookup,
			wantRef:     "#/definitions/constants.Role",
			wantNested:  "constants.Role",
			wantHasEnum: true,
		},
		{
			name: "enum with int64 underlying type creates $ref",
			field: &StructField{
				Name:       "Status",
				TypeString: "constants.Status",
				Tag:        `json:"status"`,
			},
			enumLookup:  mockEnumLookup,
			wantRef:     "#/definitions/constants.Status",
			wantNested:  "constants.Status",
			wantHasEnum: true,
		},
		{
			name: "non-enum type should not have enum values",
			field: &StructField{
				Name:       "Count",
				TypeString: "int",
				Tag:        `json:"count"`,
			},
			enumLookup:    mockEnumLookup,
			wantHasEnum:   false,
			wantPrimitive: true,
			wantType:      []string{"integer"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			propName, schema, required, nestedTypes, err := tt.field.ToSpecSchema(false, false, tt.enumLookup)
			assert.NoError(t, err)
			assert.NotNil(t, schema)
			assert.True(t, required)

			if tt.wantHasEnum {
				// Enum types should produce a $ref to the definition
				assert.Equal(t, tt.wantRef, schema.Ref.String(), "Should create $ref to enum definition")
				assert.Contains(t, nestedTypes, tt.wantNested, "Should include enum type in nestedTypes")
			} else if tt.wantPrimitive {
				// Non-enum primitives should have inline type
				assert.Contains(t, schema.Type, tt.wantType[0])
				assert.Nil(t, schema.Enum, "Non-enum should not have enum values")
			}

			assert.NotEmpty(t, propName)
		})
	}
}

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

// mockEnumLookup implements TypeEnumLookup for testing
type mockEnumLookup struct {
	enums map[string][]EnumValue
}

func (m *mockEnumLookup) GetEnumsForType(typeName string, file *ast.File) ([]EnumValue, error) {
	if enums, ok := m.enums[typeName]; ok {
		return enums, nil
	}
	return nil, fmt.Errorf("no enums for type: %s", typeName)
}

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
			wantType: []string{"string"},
		},
		{
			name:     "int",
			field:    &StructField{TypeString: "int"},
			wantType: []string{"integer"},
		},
		{
			name:       "struct without public",
			field:      &StructField{TypeString: "User"},
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
			wantType: []string{"array"},
		},
		{
			name:       "array of structs",
			field:      &StructField{TypeString: "[]User"},
			wantType:   []string{"array"},
			wantNested: []string{"User"},
		},
		{
			name:     "any type",
			field:    &StructField{TypeString: "any"},
			wantType: nil,
		},
		{
			name:     "interface{} type",
			field:    &StructField{TypeString: "interface{}"},
			wantType: nil,
		},
		{
			name:     "fields.StringField wrapper",
			field:    &StructField{TypeString: "fields.StringField"},
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

func TestApplyStructTagsToSchema(t *testing.T) {
	tests := []struct {
		name         string
		field        *StructField
		baseType     []string
		wantEnums    []interface{}
		wantVarNames []string
		wantFormat   string
		wantTitle    string
		wantMin      *float64
		wantMax      *float64
		wantReadOnly bool
	}{
		{
			name: "apply enums to integer",
			field: &StructField{
				Name: "Status",
				Tag:  `json:"status" enums:"1,2,3"`,
			},
			baseType:  []string{"integer"},
			wantEnums: []interface{}{1, 2, 3},
		},
		{
			name: "apply enums with var names",
			field: &StructField{
				Name: "Color",
				Tag:  `json:"color" enums:"red,green,blue" x-enum-varnames:"Red,Green,Blue"`,
			},
			baseType:     []string{"string"},
			wantEnums:    []interface{}{"red", "green", "blue"},
			wantVarNames: []string{"Red", "Green", "Blue"},
		},
		{
			name: "apply format tag",
			field: &StructField{
				Name: "CreatedAt",
				Tag:  `json:"created_at" format:"date-time"`,
			},
			baseType:   []string{"string"},
			wantFormat: "date-time",
		},
		{
			name: "apply title tag",
			field: &StructField{
				Name: "UserCount",
				Tag:  `json:"user_count" title:"TotalUsers"`,
			},
			baseType:  []string{"integer"},
			wantTitle: "TotalUsers",
		},
		{
			name: "apply min/max constraints",
			field: &StructField{
				Name: "Age",
				Tag:  `json:"age" minimum:"0" maximum:"120"`,
			},
			baseType: []string{"integer"},
			wantMin:  func() *float64 { v := 0.0; return &v }(),
			wantMax:  func() *float64 { v := 120.0; return &v }(),
		},
		{
			name: "apply readonly",
			field: &StructField{
				Name: "ID",
				Tag:  `json:"id" readonly:"true"`,
			},
			baseType:     []string{"integer"},
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
			if tt.wantVarNames != nil {
				assert.NotNil(t, schema.Extensions)
				assert.Equal(t, tt.wantVarNames, schema.Extensions["x-enum-varnames"])
			}
			if tt.wantFormat != "" {
				assert.Equal(t, tt.wantFormat, schema.Format)
			}
			if tt.wantTitle != "" {
				assert.Equal(t, tt.wantTitle, schema.Title)
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
			wantEnums:    []interface{}{0, 1, 2},
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
			name: "swaggertype object,string creates map schema",
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
			assert.Equal(t, len(tt.wantType), len(schema.Type))
			if len(tt.wantType) > 0 {
				assert.Equal(t, tt.wantType[0], schema.Type[0])
			}

			// swaggertype creates inline schemas, not refs
			if tt.field.Tag != `json:"normal"` {
				assert.Empty(t, nestedTypes, "swaggertype creates inline schemas, not refs")
			}

			if tt.wantItems {
				assert.NotNil(t, schema.Items, "array schema should have items")
				assert.NotNil(t, schema.Items.Schema, "array items should have schema")
				assert.Equal(t, len(tt.wantItemType), len(schema.Items.Schema.Type))
				if len(tt.wantItemType) > 0 {
					assert.Equal(t, tt.wantItemType[0], schema.Items.Schema.Type[0])
				}
			}

			if tt.wantEnums != nil {
				// For array types, enums should be on the array schema itself
				assert.Equal(t, tt.wantEnums, schema.Enum)
			}
		})
	}
}

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
			assert.Equal(t, len(tt.wantType), len(schema.Type))
			if len(tt.wantType) > 0 {
				assert.Equal(t, tt.wantType[0], schema.Type[0])
			}
			assert.Empty(t, nestedTypes)
		})
	}
}
