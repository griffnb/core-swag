package structparser

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
	"strings"

	"github.com/go-openapi/spec"
	"github.com/griffnb/core-swag/internal/model"
	"github.com/griffnb/core-swag/internal/registry"
	"github.com/griffnb/core-swag/internal/schema"
)

// Service handles struct parsing for OpenAPI schema generation.
// It supports both standard Go structs and custom model structs with fields.StructField[T].
type Service struct {
	registry      *registry.Service
	schemaBuilder *schema.BuilderService
	enumLookup    model.TypeEnumLookup
}

// resolveFullTypeName converts a short type name like "constants.Role" to full package path
// like "github.com/user/project/internal/constants.Role" using the registry
func (s *Service) resolveFullTypeName(typeName string, file *ast.File) string {
	// If type already has full path (contains /), return as-is
	if strings.Contains(typeName, "/") {
		return typeName
	}

	// Get package info for this file from registry
	files := s.registry.Files()
	fileInfo, ok := files[file]
	if !ok || fileInfo == nil {
		// Can't resolve - return as-is
		return typeName
	}

	// Extract package name from type (everything before last dot)
	lastDot := strings.LastIndex(typeName, ".")
	if lastDot == -1 {
		// No package qualifier - return as-is
		return typeName
	}

	pkgName := typeName[:lastDot]
	baseTypeName := typeName[lastDot+1:]

	// Get the package path for this file
	filePkgPath := fileInfo.PackagePath
	if filePkgPath == "" {
		return typeName
	}

	// Resolve the full package path
	// If it's the same package, use the file's package path
	if file.Name != nil && file.Name.Name == pkgName {
		return filePkgPath + "." + baseTypeName
	}

	// Otherwise, look for import of this package
	for _, imp := range file.Imports {
		if imp.Path == nil {
			continue
		}
		// Remove quotes from import path
		importPath := strings.Trim(imp.Path.Value, `"`)

		// Check if this import matches the package name
		// Get the last segment of the import path
		lastSlash := strings.LastIndex(importPath, "/")
		importPkgName := importPath
		if lastSlash >= 0 {
			importPkgName = importPath[lastSlash+1:]
		}

		// Handle import aliases
		if imp.Name != nil && imp.Name.Name == pkgName {
			return importPath + "." + baseTypeName
		} else if imp.Name == nil && importPkgName == pkgName {
			return importPath + "." + baseTypeName
		}
	}

	// Couldn't resolve - try relative to current package
	if strings.Contains(filePkgPath, "/") {
		lastSlash := strings.LastIndex(filePkgPath, "/")
		parentPath := filePkgPath[:lastSlash]
		return parentPath + "/" + pkgName + "." + baseTypeName
	}

	// Final fallback - return as-is
	return typeName
}

// NewService creates a new struct parser service
func NewService(registry *registry.Service, schemaBuilder *schema.BuilderService, enumLookup model.TypeEnumLookup) *Service {
	return &Service{
		registry:      registry,
		schemaBuilder: schemaBuilder,
		enumLookup:    enumLookup,
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
		properties, required, err := s.processField(file, field, false)
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
	return s.processField(file, field, false)
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

			// Always generate a Public variant for every struct type.
			// For types with public tags, fields are filtered to only include
			// fields with public:"view" or public:"edit" tags.
			// For types without any public tags, the Public variant is an empty
			// object — public filtering is always strict.
			// This ensures nested $ref chains work correctly when a Public schema
			// references another type's Public variant.
			publicSchemaName := schemaName + "Public"
			if s.schemaBuilder != nil {
				publicSchema, err := s.BuildPublicSchema(astFile, structType.Fields)
				if err != nil {
					continue
				}
				if publicSchema != nil {
					s.schemaBuilder.AddDefinition(publicSchemaName, *publicSchema)
				} else {
					// No public fields — register empty object so $ref chains resolve
					s.schemaBuilder.AddDefinition(publicSchemaName, emptyObjectSchema())
				}
			}
		}
	}

	return nil
}

// emptyObjectSchema returns a minimal empty object schema.
// Used for Public variants of structs that have no public-tagged fields.
func emptyObjectSchema() spec.Schema {
	return spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type:       []string{"object"},
			Properties: make(map[string]spec.Schema),
		},
	}
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
	// For backward compatibility with tests, call with nil file, genDecl and typeSpec
	return s.shouldGeneratePublicInternal(nil, nil, nil, fields)
}

func (s *Service) shouldGeneratePublicInternal(file *ast.File, genDecl *ast.GenDecl, typeSpec *ast.TypeSpec, fields *ast.FieldList) bool {
	if fields == nil {
		return false
	}

	// Check for @NoPublic annotation in struct comments (attached to GenDecl)
	if genDecl != nil && genDecl.Doc != nil {
		for _, comment := range genDecl.Doc.List {
			if strings.Contains(comment.Text, "@NoPublic") {
				return false
			}
		}
	}

	// Check if any field has public tag
	for _, field := range fields.List {
		// Check direct fields
		if len(field.Names) > 0 && hasPublicTag(field) {
			return true
		}

		// Check embedded fields recursively
		if len(field.Names) == 0 && s.registry != nil {
			// Check for json tag - if present, not truly embedded
			if field.Tag != nil {
				tagValue := field.Tag.Value
				if len(tagValue) >= 2 && tagValue[0] == '`' && tagValue[len(tagValue)-1] == '`' {
					tagValue = tagValue[1 : len(tagValue)-1]
				}
				tag := reflect.StructTag(tagValue)
				if jsonName := tag.Get("json"); jsonName != "" && jsonName != "-" {
					// Has json tag - check if it has public tag
					if hasPublicTag(field) {
						return true
					}
					continue
				}
			}

			// True embedded field - check if embedded type has public fields
			typeName, err := s.resolveEmbeddedTypeName(file, field.Type)
			if err != nil {
				continue
			}

			// Look up type in registry (use file context if available)
			typeDef := s.registry.FindTypeSpec(typeName, file)
			if typeDef == nil {
				continue
			}

			// Check if it's a struct type
			structType, ok := typeDef.TypeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			// Recursively check if embedded struct has public fields
			// Note: We don't have genDecl for embedded types, so pass nil
			if s.shouldGeneratePublicInternal(typeDef.File, nil, typeDef.TypeSpec, structType.Fields) {
				return true
			}
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
		// Handle embedded fields - recursively check for public fields
		if len(field.Names) == 0 {
			// Check for json tag - if present with a name, NOT truly embedded
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
				// Has json tag - skip for now (handled as named field)
				continue
			}

			// True embedded field - need to get its TypeSpec to recursively build Public schema
			if s.registry == nil {
				continue
			}

			// Resolve embedded type name
			typeName, err := s.resolveEmbeddedTypeName(file, field.Type)
			if err != nil {
				continue
			}

			// Look up type in registry
			typeDef := s.registry.FindTypeSpec(typeName, file)
			if typeDef == nil {
				continue
			}

			// Check if it's a struct type
			structType, ok := typeDef.TypeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			// Recursively build Public schema from embedded struct's fields
			embeddedPublicSchema, err := s.BuildPublicSchema(typeDef.File, structType.Fields)
			if err == nil && embeddedPublicSchema != nil {
				// Merge public properties from embedded struct
				for propName, propSchema := range embeddedPublicSchema.Properties {
					schema.Properties[propName] = propSchema
					hasPublicFields = true
				}
				schema.Required = append(schema.Required, embeddedPublicSchema.Required...)
			}
			continue
		}

		// Check if field has public tag
		if !hasPublicTag(field) {
			continue
		}

		hasPublicFields = true

		// Process field with public=true to add Public suffix to nested struct refs
		properties, required, err := s.processField(file, field, true)
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
