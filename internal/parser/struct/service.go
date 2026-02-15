package structparser

import (
	"go/ast"
	"go/token"
	"reflect"

	"github.com/go-openapi/spec"
	"github.com/griffnb/core-swag/internal/registry"
	"github.com/griffnb/core-swag/internal/schema"
)

// Service handles struct parsing for OpenAPI schema generation.
// It supports both standard Go structs and custom model structs with fields.StructField[T].
type Service struct {
	registry      *registry.Service
	schemaBuilder *schema.BuilderService
}

// NewService creates a new struct parser service
func NewService(registry *registry.Service, schemaBuilder *schema.BuilderService) *Service {
	return &Service{
		registry:      registry,
		schemaBuilder: schemaBuilder,
	}
}

// ParseStruct parses a struct's fields and returns its OpenAPI schema.
// Processes all fields, merging embedded struct fields, and applying tags.
func (s *Service) ParseStruct(file *ast.File, fields *ast.FieldList) (*spec.Schema, error) {
	if fields == nil {
		return &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type:       []string{"object"},
				Properties: make(map[string]spec.Schema),
			},
		}, nil
	}

	schema := &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type:       []string{"object"},
			Properties: make(map[string]spec.Schema),
			Required:   []string{},
		},
	}

	// Process each field
	for _, field := range fields.List {
		// Handle embedded fields
		if len(field.Names) == 0 {
			embeddedSchema, err := s.handleEmbeddedField(file, field)
			if err != nil {
				continue // Skip on error
			}
			if embeddedSchema != nil {
				// Merge properties from embedded struct
				for propName, propSchema := range embeddedSchema.Properties {
					schema.Properties[propName] = propSchema
				}
				// Merge required fields
				schema.Required = append(schema.Required, embeddedSchema.Required...)
			}
			continue
		}

		// Process regular field
		properties, required, err := processField(file, field)
		if err != nil {
			continue // Skip on error
		}

		// Add properties to schema
		for propName, propSchema := range properties {
			schema.Properties[propName] = propSchema
		}

		// Add to required list
		schema.Required = append(schema.Required, required...)
	}

	return schema, nil
}

// ParseField parses an individual struct field and returns its properties and required status.
// This is the public entry point that delegates to processField.
func (s *Service) ParseField(file *ast.File, field *ast.Field) (map[string]spec.Schema, []string, error) {
	return processField(file, field)
}

// ParseFile parses all struct types in a file and registers them with the schema builder.
// This is the main entry point for orchestrator integration.
func (s *Service) ParseFile(astFile *ast.File, filePath string) error {
	if astFile == nil {
		return nil
	}

	// Iterate through all declarations in the file
	for _, decl := range astFile.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		// Process each type specification
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			// Only process struct types
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			// Parse the base struct schema
			baseSchema, err := s.ParseStruct(astFile, structType.Fields)
			if err != nil {
				// Log error but continue with other types
				continue
			}

			// Generate schema name (package.TypeName format)
			schemaName := astFile.Name.Name + "." + typeSpec.Name.Name

			// Register base schema with schema builder
			if s.schemaBuilder != nil {
				s.schemaBuilder.AddDefinition(schemaName, *baseSchema)
			}

			// Check if we should generate a Public variant
			if s.ShouldGeneratePublic(structType.Fields) {
				publicSchema, err := s.BuildPublicSchema(astFile, structType.Fields)
				if err != nil {
					// Log error but continue
					continue
				}

				if publicSchema != nil {
					// Generate Public variant schema name
					publicSchemaName := schemaName + "Public"

					// Register Public variant with schema builder
					if s.schemaBuilder != nil {
						s.schemaBuilder.AddDefinition(publicSchemaName, *publicSchema)
					}
				}
			}
		}
	}

	return nil
}

// ParseDefinition parses a type definition and generates schema(s).
// This will be implemented in future phases for full integration.
func (s *Service) ParseDefinition(typeSpec ast.Expr) (*spec.Schema, error) {
	// TODO: Will implement in future phases
	// This will handle both standard structs and custom models
	return nil, nil
}

// ShouldGeneratePublic checks if a Public variant schema should be generated.
// Returns true if any fields have public:"view" or public:"edit" tags.
func (s *Service) ShouldGeneratePublic(fields *ast.FieldList) bool {
	if fields == nil {
		return false
	}

	// Check if any field has public tag
	for _, field := range fields.List {
		if hasPublicTag(field) {
			return true
		}
	}

	return false
}

// BuildPublicSchema creates a Public variant schema with only public fields.
// Filters fields based on public:"view" and public:"edit" tags.
func (s *Service) BuildPublicSchema(file *ast.File, fields *ast.FieldList) (*spec.Schema, error) {
	if fields == nil {
		return nil, nil
	}

	schema := &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type:       []string{"object"},
			Properties: make(map[string]spec.Schema),
			Required:   []string{},
		},
	}

	hasPublicFields := false

	// Process each field
	for _, field := range fields.List {
		// Skip embedded fields for Public variant
		if len(field.Names) == 0 {
			continue
		}

		// Check if field has public tag
		if !hasPublicTag(field) {
			continue
		}

		hasPublicFields = true

		// Process field
		properties, required, err := processField(file, field)
		if err != nil {
			continue
		}

		// Add properties to schema
		for propName, propSchema := range properties {
			schema.Properties[propName] = propSchema
		}

		// Add to required list
		schema.Required = append(schema.Required, required...)
	}

	// Return nil if no public fields found
	if !hasPublicFields {
		return nil, nil
	}

	return schema, nil
}

// handleEmbeddedField processes an embedded struct field and returns its schema
func (s *Service) handleEmbeddedField(file *ast.File, field *ast.Field) (*spec.Schema, error) {
	// For now, we skip embedded fields as they require type resolution
	// This will be implemented in future phases with registry integration
	// TODO: Implement full embedded struct resolution with registry
	return nil, nil
}

// hasPublicTag checks if a field has public:"view" or public:"edit" tag
func hasPublicTag(field *ast.Field) bool {
	if field.Tag == nil {
		return false
	}

	// Remove backticks from tag value
	tagValue := field.Tag.Value
	if len(tagValue) >= 2 && tagValue[0] == '`' && tagValue[len(tagValue)-1] == '`' {
		tagValue = tagValue[1 : len(tagValue)-1]
	}

	tag := reflect.StructTag(tagValue)

	// Use Phase 1.2 tag parser
	tagInfo := parseCombinedTags(tag)

	return tagInfo.Visibility == "view" || tagInfo.Visibility == "edit"
}
