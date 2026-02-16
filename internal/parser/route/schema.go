// Package route provides schema resolution for route parameter and response types.
package route

import (
	"go/ast"
	"strings"

	"github.com/go-openapi/spec"
	"github.com/griffnb/core-swag/internal/parser/route/domain"
)

// isModelType checks if a type name represents a model type (not a primitive).
// Returns true for custom types and qualified types (package.Type), false for Go primitives.
// Also returns false for extended primitives like time.Time, UUID, and decimal.
func isModelType(typeName string) bool {
	// Strip pointer prefix for checking
	cleanType := strings.TrimPrefix(typeName, "*")

	// Basic Go primitive types
	primitives := map[string]bool{
		"int":     true,
		"int8":    true,
		"int16":   true,
		"int32":   true,
		"int64":   true,
		"uint":    true,
		"uint8":   true,
		"uint16":  true,
		"uint32":  true,
		"uint64":  true,
		"float32": true,
		"float64": true,
		"bool":    true,
		"string":  true,
		"byte":    true,
		"rune":    true,
		"object":  true, // OpenAPI keyword
		"array":   true, // OpenAPI keyword
		"file":    true, // Special type for file uploads
	}

	if primitives[cleanType] {
		return false
	}

	// Extended primitives (commonly treated as primitives in OpenAPI)
	extendedPrimitives := map[string]bool{
		"time.Time":                                 true,
		"decimal.Decimal":                           true,
		"github.com/shopspring/decimal.Decimal":     true,
		"types.UUID":                                true,
		"uuid.UUID":                                 true,
		"github.com/griffnb/core/lib/types.UUID":    true,
		"github.com/google/uuid.UUID":               true,
	}

	if extendedPrimitives[cleanType] {
		return false
	}

	// If it contains a dot, it's likely a qualified type (package.Type)
	// We already checked extended primitives above, so this is a real model
	if strings.Contains(cleanType, ".") {
		return true
	}

	// Anything else that's not a primitive is likely a custom type
	return true
}

// resolveTypeSchema resolves a type name to a schema using the type resolver.
// If useRef is true, it returns a schema with a $ref to the type definition.
// Returns a basic schema if no type resolver is available or type is not found.
func (s *Service) resolveTypeSchema(typeName string, file *ast.File, useRef bool) (*domain.Schema, error) {
	// TODO: Implement full type resolution when typeResolver interface is defined
	// For now, always return basic schema
	return &domain.Schema{
		Type: convertTypeToSchemaType(typeName),
	}, nil
}

// convertSpecSchema converts an OpenAPI spec.Schema to route domain.Schema.
// Handles references, types, items for arrays, properties for objects, and required fields.
func convertSpecSchema(specSchema *spec.Schema) *domain.Schema {
	if specSchema == nil {
		return nil
	}

	schema := &domain.Schema{}

	// Handle reference
	if specSchema.Ref.String() != "" {
		schema.Ref = specSchema.Ref.String()
		return schema
	}

	// Handle type
	if len(specSchema.Type) > 0 {
		schema.Type = specSchema.Type[0]
	}

	// Handle items (for arrays)
	if specSchema.Items != nil && specSchema.Items.Schema != nil {
		schema.Items = convertSpecSchema(specSchema.Items.Schema)
	}

	// Handle properties (for objects)
	if len(specSchema.Properties) > 0 {
		schema.Properties = make(map[string]*domain.Schema)
		for name, prop := range specSchema.Properties {
			schema.Properties[name] = convertSpecSchema(&prop)
		}
	}

	// Handle required fields
	if len(specSchema.Required) > 0 {
		schema.Required = specSchema.Required
	}

	// Handle description
	if specSchema.Description != "" {
		schema.Description = specSchema.Description
	}

	return schema
}
