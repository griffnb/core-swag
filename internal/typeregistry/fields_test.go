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
	assert.Equal(t, "string", result.Schema.Type[0])
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
