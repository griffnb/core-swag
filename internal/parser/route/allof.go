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
	baseQualifiedType := baseType
	if packageName != "" && !strings.Contains(baseType, ".") {
		baseQualifiedType = packageName + "." + baseType
	}

	// Apply @Public suffix if needed
	if isPublic {
		baseQualifiedType = baseQualifiedType + "Public"
	}

	baseSchema := &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Ref: spec.MustCreateRef("#/definitions/" + baseQualifiedType),
		},
	}

	// Build override schemas from field overrides
	overrideSchemas := make(map[string]spec.Schema)
	for fieldName, fieldType := range overrides {
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

		// Build field schema
		var fieldSchema spec.Schema
		if isArray {
			// Array of types
			if isPrimitiveType(fieldType) {
				fieldSchema = spec.Schema{
					SchemaProps: spec.SchemaProps{
						Type: []string{"array"},
						Items: &spec.SchemaOrArray{
							Schema: &spec.Schema{
								SchemaProps: spec.SchemaProps{
									Type: []string{convertTypeToSchemaType(fieldType)},
								},
							},
						},
					},
				}
			} else {
				fieldSchema = spec.Schema{
					SchemaProps: spec.SchemaProps{
						Type: []string{"array"},
						Items: &spec.SchemaOrArray{
							Schema: &spec.Schema{
								SchemaProps: spec.SchemaProps{
									Ref: spec.MustCreateRef("#/definitions/" + qualifiedFieldType),
								},
							},
						},
					},
				}
			}
		} else {
			// Single type
			if isPrimitiveType(fieldType) {
				fieldSchema = spec.Schema{
					SchemaProps: spec.SchemaProps{
						Type: []string{convertTypeToSchemaType(fieldType)},
					},
				}
			} else {
				fieldSchema = spec.Schema{
					SchemaProps: spec.SchemaProps{
						Ref: spec.MustCreateRef("#/definitions/" + qualifiedFieldType),
					},
				}
			}
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
			domainSchema.AllOf = append(domainSchema.AllOf, convertSpecSchemaToDomain(&allOfSchema))
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
