package testing

import (
	"testing"

	"github.com/griffnb/core-swag/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestSwaggerType_BasicIntegration(t *testing.T) {
	tests := []struct {
		name     string
		field    *model.StructField
		wantType string
	}{
		{
			name: "swaggertype integer for NullInt64",
			field: &model.StructField{
				Name:       "NullInt",
				TypeString: "sql.NullInt64",
				Tag:        `json:"null_int" swaggertype:"integer"`,
			},
			wantType: "integer",
		},
		{
			name: "swaggertype array,number for []big.Float",
			field: &model.StructField{
				Name:       "Coeffs",
				TypeString: "[]big.Float",
				Tag:        `json:"coeffs" swaggertype:"array,number"`,
			},
			wantType: "array",
		},
		{
			name: "swaggertype with enums",
			field: &model.StructField{
				Name:       "FoodTypes",
				TypeString: "[]string",
				Tag:        `json:"food_types" swaggertype:"array,integer" enums:"0,1,2"`,
			},
			wantType: "array",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			propName, schema, required, _, err := tt.field.ToSpecSchema(false, false, nil)

			assert.NoError(t, err)
			assert.NotEmpty(t, propName)
			assert.NotNil(t, schema)
			assert.Equal(t, 1, len(schema.Type))
			assert.Equal(t, tt.wantType, schema.Type[0])
			assert.True(t, required, "should be required by default")
		})
	}
}
