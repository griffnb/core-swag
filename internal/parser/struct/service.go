package structparser

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
	"strings"

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
		// Handle embedded fields (no field name)
		if len(field.Names) == 0 {
			// Check if it has a json tag - if so, treat as named field
			hasJSONTag := false
			if field.Tag != nil {
				tagValue := field.Tag.Value
				if len(tagValue) >= 2 && tagValue[0] == '`' && tagValue[len(tagValue)-1] == '`' {
					tagValue = tagValue[1 : len(tagValue)-1]
				}
				tag := reflect.StructTag(tagValue)
				if jsonName := tag.Get("json"); jsonName != "" && jsonName != "-" {
					hasJSONTag = true
				}
			}

			if hasJSONTag {
				// Has json tag - process as named field with the type as an object reference
				// For this case, we'll create a simple object schema
				properties, required, err := s.processEmbeddedWithTag(file, field)
				if err != nil {
					continue
				}
				for propName, propSchema := range properties {
					schema.Properties[propName] = propSchema
				}
				schema.Required = append(schema.Required, required...)
				continue
			}

			// True embedded field - merge properties
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

// handleEmbeddedField processes an embedded struct field and returns its schema.
// It resolves the embedded type, recursively parses its fields, and returns a schema
// with all embedded properties merged.
func (s *Service) handleEmbeddedField(file *ast.File, field *ast.Field) (*spec.Schema, error) {
	// Check for json tag - if present with a name, NOT truly embedded
	if field.Tag != nil {
		tagValue := field.Tag.Value
		if len(tagValue) >= 2 && tagValue[0] == '`' && tagValue[len(tagValue)-1] == '`' {
			tagValue = tagValue[1 : len(tagValue)-1]
		}
		tag := reflect.StructTag(tagValue)
		if jsonName := tag.Get("json"); jsonName != "" && jsonName != "-" {
			// Has json tag with a name, so it's a named field, not embedded
			// Let processField handle it
			return nil, nil
		}
	}

	// Registry is required for embedded field resolution
	if s.registry == nil {
		return nil, nil
	}

	// Resolve embedded type name
	typeName, err := s.resolveEmbeddedTypeName(file, field.Type)
	if err != nil {
		// Unable to resolve - skip this embedded field
		return nil, nil
	}

	// Look up type in registry
	typeDef := s.registry.FindTypeSpec(typeName, file)
	if typeDef == nil {
		// Unknown type - might be external package not in registry
		return nil, nil
	}

	// Check if it's a struct type
	structType, ok := typeDef.TypeSpec.Type.(*ast.StructType)
	if !ok {
		// Not a struct - can't embed non-struct types
		return nil, nil
	}

	// Check if empty struct
	if structType.Fields == nil || len(structType.Fields.List) == 0 {
		// Empty struct - nothing to merge
		return nil, nil
	}

	// Recursively parse the embedded struct's fields
	embeddedSchema, err := s.ParseStruct(typeDef.File, structType.Fields)
	if err != nil {
		return nil, err
	}

	// Return the embedded schema to be merged
	return embeddedSchema, nil
}

// resolveEmbeddedTypeName resolves the type name from an embedded field's AST expression.
// It handles simple types (Ident), package-qualified types (SelectorExpr), and pointer types (StarExpr).
func (s *Service) resolveEmbeddedTypeName(file *ast.File, expr ast.Expr) (string, error) {
	switch t := expr.(type) {
	case *ast.Ident:
		// Simple type in same package: "BaseModel"
		return t.Name, nil

	case *ast.SelectorExpr:
		// Package-qualified type: "model.BaseModel"
		if pkg, ok := t.X.(*ast.Ident); ok {
			return pkg.Name + "." + t.Sel.Name, nil
		}
		return "", fmt.Errorf("unsupported selector expression in embedded field")

	case *ast.StarExpr:
		// Pointer type: "*BaseModel" → strip pointer
		return s.resolveEmbeddedTypeName(file, t.X)

	default:
		return "", fmt.Errorf("unsupported embedded type expression: %T", expr)
	}
}

// processEmbeddedWithTag handles embedded fields that have a json tag.
// These are treated as named fields rather than true embeddings.
// Example: type Outer struct { Inner `json:"inner"` }
func (s *Service) processEmbeddedWithTag(file *ast.File, field *ast.Field) (map[string]spec.Schema, []string, error) {
	if field.Tag == nil {
		return nil, nil, nil
	}

	// Parse the json tag to get the field name
	tagValue := field.Tag.Value
	if len(tagValue) >= 2 && tagValue[0] == '`' && tagValue[len(tagValue)-1] == '`' {
		tagValue = tagValue[1 : len(tagValue)-1]
	}
	tag := reflect.StructTag(tagValue)
	tagInfo := parseCombinedTags(tag)

	// If ignored, skip
	if tagInfo.Ignore {
		return nil, nil, nil
	}

	// Use json name or fall back to type name
	fieldName := tagInfo.JSONName
	if fieldName == "" {
		// Get type name from the field type
		typeName, err := s.resolveEmbeddedTypeName(file, field.Type)
		if err != nil {
			return nil, nil, err
		}
		// Use simple type name (after last dot)
		parts := strings.Split(typeName, ".")
		fieldName = parts[len(parts)-1]
	}

	// Create a simple object schema for the embedded type
	// This creates a reference to the type as a nested object
	properties := make(map[string]spec.Schema)
	properties[fieldName] = spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type: []string{"object"},
		},
	}

	// Check if required
	var required []string
	if tagInfo.Required {
		required = append(required, fieldName)
	}

	return properties, required, nil
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
