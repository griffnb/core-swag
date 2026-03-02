package model

import (
	"fmt"
	"go/ast"
	"testing"

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
			wantType: []string{"object"},
		},
		{
			name: "interface{} type",
			field: &StructField{
				Name:       "Metadata",
				TypeString: "interface{}",
				Tag:        `json:"metadata"`,
			},
			wantType: []string{"object"},
		},
		{
			name: "interface{} with spaces",
			field: &StructField{
				Name:       "Options",
				TypeString: "interface {}",
				Tag:        `json:"options"`,
			},
			wantType: []string{"object"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			propName, schema, required, nestedTypes, err := tt.field.ToSpecSchema(false, false, nil)
			assert.NoError(t, err)
			assert.NotNil(t, schema)
			assert.True(t, required, "Any/interface fields should be required by default")

			// Check that schema has object type
			assert.Equal(t, len(tt.wantType), len(schema.Type), "Schema type length should match")
			if len(tt.wantType) > 0 && len(schema.Type) > 0 {
				assert.Equal(t, tt.wantType[0], schema.Type[0], "Any/interface should generate object type")
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
			wantType: []string{"object"},
		},
		{
			name:     "interface{} type",
			field:    &StructField{TypeString: "interface{}"},
			wantType: []string{"object"},
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
