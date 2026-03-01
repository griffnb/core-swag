package field

import (
	"github.com/go-openapi/spec"
)

// TransToValidCollectionFormat validates and returns a collection format string.
// Returns empty string if the format is not valid.
func TransToValidCollectionFormat(format string) string {
	switch format {
	case "csv", "multi", "pipes", "tsv", "ssv":
		return format
	}

	return ""
}

// SchemaHelper provides schema-related utility functions needed by field parser.
// This interface allows the field parser to work without depending on the swag package.
type SchemaHelper interface {
	// BuildCustomSchema builds a custom schema from swaggertype tag values
	BuildCustomSchema(types []string) (*spec.Schema, error)

	// IsRefSchema determines if a schema is a reference schema
	IsRefSchema(schema *spec.Schema) bool

	// DefineType converts a string value to the appropriate Go type based on schema type
	DefineType(schemaType string, value string) (interface{}, error)

	// DefineTypeOfExample converts an example value string to the appropriate type
	DefineTypeOfExample(schemaType, arrayType, exampleValue string) (interface{}, error)

	// PrimitiveSchema creates a schema for a primitive type
	PrimitiveSchema(refType string) *spec.Schema

	// SetExtensionParam parses extension attributes and returns spec.Extensions
	SetExtensionParam(attr string) spec.Extensions

	// GetSchemaTypePath extracts type path from schema for validation
	GetSchemaTypePath(schema *spec.Schema, depth int) []string

	// IsNumericType determines if a type is numeric (integer or number)
	IsNumericType(typeName string) bool
}

// ParserConfig provides configuration needed by field parser.
type ParserConfig interface {
	// GetNamingStrategy returns the property naming strategy (camelcase, snakecase, pascalcase)
	GetNamingStrategy() string

	// IsRequiredByDefault returns whether fields are required by default
	IsRequiredByDefault() bool
}
