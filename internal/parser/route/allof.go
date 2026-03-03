package route

import (
	"go/ast"
	"strings"

	"github.com/go-openapi/spec"
	"github.com/griffnb/core-swag/internal/parser/route/domain"
	"github.com/griffnb/core-swag/internal/schema"
)

// buildAllOfResponseSchema handles combined type syntax like Response{data=Account}
// It uses Phase 1.3 AllOf composition functions.
// file is used for import resolution to produce fully qualified TypePath values.
func (s *Service) buildAllOfResponseSchema(dataType, packageName string, isPublic bool, file *ast.File) *domain.Schema {
	// Parse combined type syntax
	baseType, overrides, err := schema.ParseCombinedType(dataType)
	if err != nil {
		// If parsing fails, fall back to regular schema building
		return s.buildSchemaForTypeWithPublic(dataType, packageName, isPublic, file)
	}

	// If no overrides, just return regular schema
	if len(overrides) == 0 {
		return s.buildSchemaForTypeWithPublic(baseType, packageName, isPublic, file)
	}

	// Build base schema (qualified with package if needed)
	// NOTE: Do NOT apply @Public suffix to the base response wrapper type.
	// The @Public suffix should only apply to the data models in field overrides.
	baseQualifiedType := baseType
	if packageName != "" && !strings.Contains(baseType, ".") {
		baseQualifiedType = packageName + "." + baseType
	}

	baseSchema := &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Ref: spec.MustCreateRef("#/definitions/" + baseQualifiedType),
		},
	}

	// Build override schemas from field overrides
	overrideSchemas := make(map[string]spec.Schema)
	for fieldName, fieldType := range overrides {
		overrideSchemas[fieldName] = s.resolveOverrideTypeSchema(fieldType, packageName, isPublic, file)
	}

	// Build AllOf composition using Phase 1.3 function
	allOfSchema := schema.BuildAllOfSchema(baseSchema, overrideSchemas)

	// Convert spec.Schema to domain.Schema
	result := convertSpecSchemaToDomain(allOfSchema)

	// Resolve TypePath on all nested $ref schemas for unambiguous registry lookups
	s.resolveTypePathsInSchema(result, file)

	return result
}

// convertSpecSchemaToDomain converts a spec.Schema to domain.Schema
func convertSpecSchemaToDomain(s *spec.Schema) *domain.Schema {
	if s == nil {
		return nil
	}

	domainSchema := &domain.Schema{}

	// Handle AllOf - preserve the composition structure
	if len(s.AllOf) > 0 {
		domainSchema.AllOf = make([]*domain.Schema, 0, len(s.AllOf))
		for _, allOfSchema := range s.AllOf {
			converted := convertSpecSchemaToDomain(&allOfSchema)
			domainSchema.AllOf = append(domainSchema.AllOf, converted)
			// Extract properties from AllOf elements to the top-level schema
			// so callers can access override fields directly (e.g., data, meta)
			for name, prop := range converted.Properties {
				if domainSchema.Properties == nil {
					domainSchema.Properties = make(map[string]*domain.Schema)
				}
				domainSchema.Properties[name] = prop
			}
		}
		// Use type "object" for AllOf compositions
		domainSchema.Type = "object"
		return domainSchema
	}

	// Handle type
	if len(s.Type) > 0 {
		domainSchema.Type = s.Type[0]
	}

	// Handle reference
	if s.Ref.String() != "" {
		domainSchema.Ref = s.Ref.String()
	}

	// Handle items (for arrays)
	if s.Items != nil && s.Items.Schema != nil {
		domainSchema.Items = convertSpecSchemaToDomain(s.Items.Schema)
	}

	// Handle properties (for objects)
	if len(s.Properties) > 0 {
		domainSchema.Properties = make(map[string]*domain.Schema)
		for name, prop := range s.Properties {
			domainSchema.Properties[name] = convertSpecSchemaToDomain(&prop)
		}
	}

	// Handle additionalProperties (for map types)
	if s.AdditionalProperties != nil && s.AdditionalProperties.Schema != nil {
		domainSchema.AdditionalProperties = convertSpecSchemaToDomain(s.AdditionalProperties.Schema)
	}

	// Handle required fields
	if len(s.Required) > 0 {
		domainSchema.Required = s.Required
	}

	// Handle description
	if s.Description != "" {
		domainSchema.Description = s.Description
	}

	return domainSchema
}

// resolveOverrideTypeSchema recursively resolves a type expression into a spec.Schema.
// Handles: primitives, any/interface{}, []T, map[K]V, and model references.
func (s *Service) resolveOverrideTypeSchema(fieldType, packageName string, isPublic bool, file *ast.File) spec.Schema {
	// Handle array prefix: []T → {type: array, items: resolve(T)}
	if strings.HasPrefix(fieldType, "[]") {
		innerSchema := s.resolveOverrideTypeSchema(fieldType[2:], packageName, isPublic, file)
		return spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type:  []string{"array"},
				Items: &spec.SchemaOrArray{Schema: &innerSchema},
			},
		}
	}

	// Handle map prefix: map[K]V
	if strings.HasPrefix(fieldType, "map[") {
		idx := strings.Index(fieldType, "]")
		if idx >= 0 && idx+1 < len(fieldType) {
			valueType := fieldType[idx+1:]
			// map[string]any / map[string]interface{} → plain {type: "object"}
			if isWildcardMapValue(valueType) {
				return spec.Schema{
					SchemaProps: spec.SchemaProps{
						Type: []string{"object"},
					},
				}
			}
			valueSchema := s.resolveOverrideTypeSchema(valueType, packageName, isPublic, file)
			return *spec.MapProperty(&valueSchema)
		}
	}

	// Wildcard types → plain object
	if fieldType == "any" || fieldType == "interface{}" {
		return spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type: []string{"object"},
			},
		}
	}

	// Primitive types
	if isPrimitiveType(fieldType) {
		return spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type: []string{convertTypeToSchemaType(fieldType)},
			},
		}
	}

	// Model reference
	qualifiedFieldType := fieldType
	if packageName != "" && !strings.Contains(fieldType, ".") {
		qualifiedFieldType = packageName + "." + fieldType
	}
	if isPublic && s.isStructType(qualifiedFieldType) {
		qualifiedFieldType = qualifiedFieldType + "Public"
	}
	return spec.Schema{
		SchemaProps: spec.SchemaProps{
			Ref: spec.MustCreateRef("#/definitions/" + qualifiedFieldType),
		},
	}
}

// isWildcardMapValue returns true when a map value type means "any JSON value",
// so the map should be rendered as {type: "object"} without additionalProperties.
func isWildcardMapValue(valueType string) bool {
	return valueType == "any" || valueType == "interface{}"
}

// isPrimitiveType checks if a type is a Go/OpenAPI primitive or a wildcard type
// that should not be treated as a model reference.
func isPrimitiveType(typeName string) bool {
	primitives := map[string]bool{
		"int":           true,
		"int8":          true,
		"int16":         true,
		"int32":         true,
		"int64":         true,
		"uint":          true,
		"uint8":         true,
		"uint16":        true,
		"uint32":        true,
		"uint64":        true,
		"float32":       true,
		"float64":       true,
		"bool":          true,
		"string":        true,
		"byte":          true,
		"rune":          true,
		"any":           true,
		"interface{}":   true,
		"object":        true,
	}
	return primitives[typeName]
}
