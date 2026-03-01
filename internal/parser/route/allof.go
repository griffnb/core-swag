package route

import (
	"strings"

	"github.com/go-openapi/spec"
	"github.com/griffnb/core-swag/internal/parser/route/domain"
	"github.com/griffnb/core-swag/internal/schema"
)

// buildAllOfResponseSchema handles combined type syntax like Response{data=Account}
// It uses Phase 1.3 AllOf composition functions
func (s *Service) buildAllOfResponseSchema(dataType, packageName string, isPublic bool) *domain.Schema {
	// Parse combined type syntax
	baseType, overrides, err := schema.ParseCombinedType(dataType)
	if err != nil {
		// If parsing fails, fall back to regular schema building
		return s.buildSchemaForTypeWithPublic(dataType, packageName, isPublic)
	}

	// If no overrides, just return regular schema
	if len(overrides) == 0 {
		return s.buildSchemaForTypeWithPublic(baseType, packageName, isPublic)
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
		// Parse field type to handle maps: map[string]Account
		isMap := strings.HasPrefix(fieldType, "map[")
		var mapValueType string
		if isMap {
			idx := strings.Index(fieldType, "]")
			if idx >= 0 && idx+1 < len(fieldType) {
				mapValueType = fieldType[idx+1:]
				fieldType = mapValueType
			} else {
				isMap = false
			}
		}

		// Parse field type to handle arrays: []Account
		isArray := strings.HasPrefix(fieldType, "[]")
		if isArray {
			fieldType = strings.TrimPrefix(fieldType, "[]")
		}

		// Qualify type if needed
		qualifiedFieldType := fieldType
		if packageName != "" && !strings.Contains(fieldType, ".") && !isPrimitiveType(fieldType) {
			qualifiedFieldType = packageName + "." + fieldType
		}

		// Apply @Public suffix if needed and not a primitive
		if isPublic && !isPrimitiveType(fieldType) {
			qualifiedFieldType = qualifiedFieldType + "Public"
		}

		// Build the inner value schema (used by all branches)
		var valueSchema spec.Schema
		if isPrimitiveType(fieldType) {
			valueSchema = spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type: []string{convertTypeToSchemaType(fieldType)},
				},
			}
		} else {
			valueSchema = spec.Schema{
				SchemaProps: spec.SchemaProps{
					Ref: spec.MustCreateRef("#/definitions/" + qualifiedFieldType),
				},
			}
		}

		// Build field schema
		var fieldSchema spec.Schema
		if isMap {
			// Map type: map[string]ValueType → object with additionalProperties
			fieldSchema = *spec.MapProperty(&valueSchema)
		} else if isArray {
			// Array type: []ValueType
			fieldSchema = spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type: []string{"array"},
					Items: &spec.SchemaOrArray{
						Schema: &valueSchema,
					},
				},
			}
		} else {
			fieldSchema = valueSchema
		}

		overrideSchemas[fieldName] = fieldSchema
	}

	// Build AllOf composition using Phase 1.3 function
	allOfSchema := schema.BuildAllOfSchema(baseSchema, overrideSchemas)

	// Convert spec.Schema to domain.Schema
	return convertSpecSchemaToDomain(allOfSchema)
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

// isPrimitiveType checks if a type is a Go/OpenAPI primitive
func isPrimitiveType(typeName string) bool {
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
	}
	return primitives[typeName]
}
