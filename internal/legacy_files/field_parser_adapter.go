package swag

import (
	"github.com/go-openapi/spec"
	"github.com/griffnb/core-swag/internal/parser/field"
)

// parserAdapter adapts the Parser to implement field.SchemaHelper and field.ParserConfig.
type parserAdapter struct {
	p *Parser
}

// Ensure parserAdapter implements both interfaces
var (
	_ field.SchemaHelper = (*parserAdapter)(nil)
	_ field.ParserConfig = (*parserAdapter)(nil)
)

// newParserAdapter creates a new parser adapter.
func newParserAdapter(p *Parser) *parserAdapter {
	return &parserAdapter{p: p}
}

// BuildCustomSchema implements field.SchemaHelper.
func (pa *parserAdapter) BuildCustomSchema(types []string) (*spec.Schema, error) {
	return BuildCustomSchema(types)
}

// IsRefSchema implements field.SchemaHelper.
func (pa *parserAdapter) IsRefSchema(schema *spec.Schema) bool {
	return IsRefSchema(schema)
}

// DefineType implements field.SchemaHelper.
func (pa *parserAdapter) DefineType(schemaType string, value string) (interface{}, error) {
	return DefineType(schemaType, value)
}

// DefineTypeOfExample implements field.SchemaHelper.
func (pa *parserAdapter) DefineTypeOfExample(schemaType, arrayType, exampleValue string) (interface{}, error) {
	return DefineTypeOfExample(schemaType, arrayType, exampleValue)
}

// PrimitiveSchema implements field.SchemaHelper.
func (pa *parserAdapter) PrimitiveSchema(refType string) *spec.Schema {
	return PrimitiveSchema(refType)
}

// SetExtensionParam implements field.SchemaHelper.
func (pa *parserAdapter) SetExtensionParam(attr string) spec.Extensions {
	return SetExtensionParam(attr)
}

// GetSchemaTypePath implements field.SchemaHelper.
func (pa *parserAdapter) GetSchemaTypePath(schema *spec.Schema, depth int) []string {
	return pa.p.GetSchemaTypePath(schema, depth)
}

// IsNumericType implements field.SchemaHelper.
func (pa *parserAdapter) IsNumericType(typeName string) bool {
	return IsNumericType(typeName)
}

// GetNamingStrategy implements field.ParserConfig.
func (pa *parserAdapter) GetNamingStrategy() string {
	return pa.p.PropNamingStrategy
}

// IsRequiredByDefault implements field.ParserConfig.
func (pa *parserAdapter) IsRequiredByDefault() bool {
	return pa.p.RequiredByDefault
}
