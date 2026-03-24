package typeregistry

import (
	"testing"

	"github.com/go-openapi/spec"
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
	assert.False(t, IsExtendedPrimitive("string"))
}

func TestToSchema(t *testing.T) {
	schema := ToSchema("types.UUID")
	assert.Equal(t, spec.StringOrArray{"string"}, schema.Type)
	assert.Equal(t, "uuid", schema.Format)

	schema = ToSchema("*decimal.Decimal")
	assert.Equal(t, spec.StringOrArray{"number"}, schema.Type)
	assert.Equal(t, "", schema.Format)

	schema = ToSchema("[]byte")
	assert.Equal(t, spec.StringOrArray{"string"}, schema.Type)
	assert.Equal(t, "byte", schema.Format)

	// Unknown type returns nil
	assert.Nil(t, ToSchema("unknown.Type"))
}
