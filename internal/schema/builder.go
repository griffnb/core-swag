// Package schema provides schema building and management functionality for OpenAPI schemas.
package schema

import (
	"go/ast"
	"strings"
	"unicode"

	"github.com/go-openapi/spec"
	"github.com/griffnb/core-swag/internal/domain"
	"github.com/griffnb/core-swag/internal/model"
)

// TypeResolver provides type lookup functionality.
type TypeResolver interface {
	FindTypeSpec(typeName string, file *ast.File) *domain.TypeSpecDef
}

// BuilderService handles schema construction and definition management.
type BuilderService struct {
	definitions        map[string]spec.Schema
	parsedSchemas      map[*domain.TypeSpecDef]string
	propNamingStrategy string // camelcase, snakecase, pascalcase, or empty (default camelcase)
	structParser       *model.CoreStructParser
	enumLookup         model.TypeEnumLookup
	requiredByDefault  bool
	typeResolver       TypeResolver // for resolving type aliases
}

// NewBuilder creates a new BuilderService instance.
func NewBuilder() *BuilderService {
	return &BuilderService{
		definitions:        make(map[string]spec.Schema),
		parsedSchemas:      make(map[*domain.TypeSpecDef]string),
		propNamingStrategy: "camelcase", // default
		structParser:       nil,         // Will be set if needed
		requiredByDefault:  false,
	}
}

// SetPropNamingStrategy sets the property naming strategy
func (b *BuilderService) SetPropNamingStrategy(strategy string) {
	if strategy == "" {
		strategy = "camelcase"
	}
	b.propNamingStrategy = strategy
}

// SetTypeResolver sets the type resolver for resolving type aliases
func (b *BuilderService) SetTypeResolver(resolver TypeResolver) {
	b.typeResolver = resolver
}

// SetStructParser sets the struct parser for proper field resolution
func (b *BuilderService) SetStructParser(parser *model.CoreStructParser) {
	b.structParser = parser
}

// SetEnumLookup sets the enum lookup for enum type resolution
func (b *BuilderService) SetEnumLookup(enumLookup model.TypeEnumLookup) {
	b.enumLookup = enumLookup
}

// BuildSchema builds an OpenAPI schema from a TypeSpecDef.
// Returns the schema name and any error encountered.
func (b *BuilderService) BuildSchema(typeSpec *domain.TypeSpecDef) (string, error) {
	// Get schema name first
	schemaName := typeSpec.SchemaName
	if schemaName == "" {
		schemaName = typeSpec.TypeName()
	}

	// Check if already parsed
	if existingName, ok := b.parsedSchemas[typeSpec]; ok {
		return existingName, nil
	}

	// Check if schema already exists in definitions (may have been built by StructParser)
	if _, exists := b.definitions[schemaName]; exists {
		// Schema already exists - mark as parsed and return
		b.parsedSchemas[typeSpec] = schemaName
		return schemaName, nil
	}

	// Build schema from TypeSpec by parsing the AST
	schema := spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type: []string{"object"},
		},
	}

	// Handle different type kinds
	switch t := typeSpec.TypeSpec.Type.(type) {
	case *ast.StructType:
		// Direct struct type - use CoreStructParser if available for proper field resolution
		if b.structParser != nil {
			// Extract package path and type name
			packagePath := typeSpec.PkgPath
			typeName := typeSpec.TypeSpec.Name.Name

			// Use CoreStructParser to get properly parsed fields
			builder := b.structParser.LookupStructFields("", packagePath, typeName)
			if builder != nil {
				// Use StructBuilder.BuildSpecSchema() to generate schema with proper type references
				builtSchema, _, err := builder.BuildSpecSchema(typeName, false, false, b.enumLookup)
				if err == nil && builtSchema != nil {
					schema = *builtSchema
					break
				}
				// If CoreStructParser failed, fall back to simple AST parsing below
			}
		}

		// Fallback: Simple AST parsing (used when CoreStructParser not available or fails)
		structType := t
		schema.Properties = make(map[string]spec.Schema)
		if structType.Fields != nil {
			for _, field := range structType.Fields.List {
				// Get field name
				if len(field.Names) == 0 {
					continue // Embedded field, skip for now
				}
				fieldName := field.Names[0].Name

				// Get JSON tag if present, or apply naming strategy
				jsonName := b.applyNamingStrategy(fieldName)
				var example interface{}

				if field.Tag != nil {
					// Remove backticks from tag value
					tag := field.Tag.Value
					if len(tag) >= 2 && tag[0] == '`' && tag[len(tag)-1] == '`' {
						tag = tag[1 : len(tag)-1]
					}

					// Parse json tag
					if jsonIdx := indexOf(tag, "json:\""); jsonIdx >= 0 {
						jsonStart := jsonIdx + 6
						jsonEnd := indexOf(tag[jsonStart:], "\"")
						if jsonEnd > 0 {
							jsonTag := tag[jsonStart : jsonStart+jsonEnd]
							// Take first part before comma
							for i, ch := range jsonTag {
								if ch == ',' {
									jsonTag = jsonTag[:i]
									break
								}
							}
							if jsonTag != "" && jsonTag != "-" {
								jsonName = jsonTag
							}
						}
					}

					// Parse example tag
					if exampleIdx := indexOf(tag, "example:\""); exampleIdx >= 0 {
						exampleStart := exampleIdx + 9
						exampleEnd := indexOf(tag[exampleStart:], "\"")
						if exampleEnd > 0 {
							example = tag[exampleStart : exampleStart+exampleEnd]
						}
					}
				}

				// Create property schema
				propSchema := b.buildFieldSchema(field.Type, typeSpec.File, example)

				schema.Properties[jsonName] = propSchema
			}
		}
	case *ast.Ident:
		// Type alias to another named type (e.g., type CrossErrors errors.Errors)
		// Try to resolve the alias to the actual type
		if b.typeResolver != nil {
			resolvedType := b.typeResolver.FindTypeSpec(t.Name, typeSpec.File)
			if resolvedType != nil && resolvedType != typeSpec {
				// Recursively build schema for resolved type
				_, err := b.BuildSchema(resolvedType)
				if err != nil {
					return "", err
				}
				// Copy the resolved schema
				if resolvedSchema, ok := b.definitions[resolvedType.TypeName()]; ok {
					schema = resolvedSchema
				}
			}
		}
		// If no resolver or resolution failed, create empty object
		if schema.Properties == nil {
			schema.Properties = make(map[string]spec.Schema)
		}
	case *ast.SelectorExpr:
		// Type alias to external package type (e.g., type CrossErrors pkg.Errors)
		// Try to resolve the alias
		if b.typeResolver != nil && typeSpec.File != nil {
			// Build qualified name: package.Type
			var pkgName string
			if ident, ok := t.X.(*ast.Ident); ok {
				pkgName = ident.Name
			}
			typeName := pkgName + "." + t.Sel.Name

			resolvedType := b.typeResolver.FindTypeSpec(typeName, typeSpec.File)
			if resolvedType != nil && resolvedType != typeSpec {
				// Recursively build schema for resolved type
				_, err := b.BuildSchema(resolvedType)
				if err != nil {
					return "", err
				}
				// Copy the resolved schema
				if resolvedSchema, ok := b.definitions[resolvedType.TypeName()]; ok {
					schema = resolvedSchema
				}
			}
		}
		// If no resolver or resolution failed, create empty object
		if schema.Properties == nil {
			schema.Properties = make(map[string]spec.Schema)
		}
	}

	// Store in definitions
	b.definitions[schemaName] = schema
	b.parsedSchemas[typeSpec] = schemaName

	return schemaName, nil
}

func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func indexOf(s, substr string) int {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// getFieldType returns the OpenAPI type for an AST expression.
// Returns the type string, format string, and qualified type name (for references).
// For extended primitives, returns proper type with format.
// For custom types, returns qualified name for reference creation.
func getFieldType(expr ast.Expr) (string, string, string) {
	return getFieldTypeImpl(expr, "")
}

func getFieldTypeImpl(expr ast.Expr, prefix string) (schemaType string, format string, qualifiedName string) {
	switch t := expr.(type) {
	case *ast.Ident:
		// Basic type like string, int, or custom type
		switch t.Name {
		case "string":
			return "string", "", ""
		case "int", "int32", "int64", "uint", "uint32", "uint64", "int8", "int16", "uint8", "uint16", "byte", "rune":
			return "integer", "", ""
		case "float32", "float64":
			return "number", "", ""
		case "bool":
			return "boolean", "", ""
		case "error", "any":
			// error and any are interface types - allow any JSON value
			return "interface", "", ""
		default:
			// Custom type (enum or struct) - return as qualified name for reference
			return "object", "", t.Name
		}
	case *ast.SelectorExpr:
		// Package-qualified type like time.Time, uuid.UUID, constants.Role
		if ident, ok := t.X.(*ast.Ident); ok {
			packageName := ident.Name
			typeName := t.Sel.Name
			fullType := packageName + "." + typeName

			// Check for extended primitives using domain package
			if domain.IsExtendedPrimitiveType(fullType) {
				// Handle specific extended primitives with formats
				switch {
				case packageName == "time" && typeName == "Time":
					return "string", "date-time", ""
				case (packageName == "uuid" || packageName == "types") && typeName == "UUID":
					return "string", "uuid", ""
				case packageName == "decimal" && typeName == "Decimal":
					return "number", "", ""
				default:
					// Other extended primitives - use TransToValidPrimitiveSchema logic
					schema := domain.TransToValidPrimitiveSchema(fullType)
					if schema != nil && len(schema.Type) > 0 {
						return schema.Type[0], schema.Format, ""
					}
					return "string", "", ""
				}
			}

			// Not an extended primitive - must be a custom type (enum or struct)
			// Return qualified name for reference creation
			return "object", "", fullType
		}
		return "object", "", ""
	case *ast.ArrayType:
		return "array", "", ""
	case *ast.StarExpr:
		// Pointer type - recurse but keep qualified name
		return getFieldTypeImpl(t.X, prefix)
	case *ast.InterfaceType:
		// interface{} or any - allow any JSON value
		return "interface", "", ""
	case *ast.MapType:
		return "object", "", "" // Maps are represented as objects in OpenAPI
	default:
		return "object", "", ""
	}
}

// buildFieldSchema builds a schema for a struct field, handling primitives, refs, enums, and arrays.
func (b *BuilderService) buildFieldSchema(fieldType ast.Expr, file *ast.File, example interface{}) spec.Schema {
	// Get field type information
	schemaType, format, qualifiedName := getFieldType(fieldType)

	// Handle array types
	if schemaType == "array" {
		if arrayType, ok := fieldType.(*ast.ArrayType); ok {
			elemSchema := b.buildFieldSchema(arrayType.Elt, file, nil)
			schema := spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type:  []string{"array"},
					Items: &spec.SchemaOrArray{Schema: &elemSchema},
				},
			}
			if example != nil {
				schema.Example = example
			}
			return schema
		}
	}

	// Handle interface types
	if schemaType == "interface" {
		return spec.Schema{
			SchemaProps: spec.SchemaProps{
				// Empty - allows any JSON value
			},
		}
	}

	// Handle custom types (enums or structs) - qualifiedName is set
	if qualifiedName != "" {
		// Check if it's an enum type first
		if b.enumLookup != nil {
			enums, err := b.enumLookup.GetEnumsForType(qualifiedName, file)
			if err == nil && len(enums) > 0 {
				// It's an enum - create inline enum schema
				schema := spec.Schema{
					SchemaProps: spec.SchemaProps{},
				}
				// Determine enum base type from first value
				if len(enums) > 0 {
					switch enums[0].Value.(type) {
					case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
						schema.Type = []string{"integer"}
					case string:
						schema.Type = []string{"string"}
					case float32, float64:
						schema.Type = []string{"number"}
					default:
						schema.Type = []string{"integer"}
					}
				}
				// Add enum values
				var enumValues []interface{}
				for _, e := range enums {
					enumValues = append(enumValues, e.Value)
				}
				schema.Enum = enumValues
				if example != nil {
					schema.Example = example
				}
				return schema
			}
		}

		// Not an enum - create reference to nested type
		schema := spec.Schema{
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/" + qualifiedName),
			},
		}
		return schema
	}

	// Primitive type - create schema with type and format
	schema := spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type: []string{schemaType},
		},
	}
	if format != "" {
		schema.Format = format
	}
	if example != nil {
		schema.Example = example
	}
	return schema
}

// AddDefinition adds a schema definition with the given name.
func (b *BuilderService) AddDefinition(name string, schema spec.Schema) error {
	b.definitions[name] = schema
	// Note: We don't have the TypeSpecDef here to add to parsedSchemas
	// This is OK - BuildSchema will check definitions first
	return nil
}

// GetDefinition retrieves a schema definition by name.
// Returns the schema and true if found, zero schema and false otherwise.
func (b *BuilderService) GetDefinition(name string) (spec.Schema, bool) {
	schema, ok := b.definitions[name]
	return schema, ok
}

// Definitions returns all schema definitions.
func (b *BuilderService) Definitions() map[string]spec.Schema {
	return b.definitions
}

// applyNamingStrategy applies the configured naming strategy to a field name
func (b *BuilderService) applyNamingStrategy(fieldName string) string {
	switch strings.ToLower(b.propNamingStrategy) {
	case "snakecase", "snake_case":
		return toSnakeCase(fieldName)
	case "pascalcase", "pascal_case":
		return fieldName // Keep as-is (PascalCase)
	case "camelcase", "camel_case", "":
		return toCamelCase(fieldName)
	default:
		return toCamelCase(fieldName) // Default to camelCase
	}
}

// toCamelCase converts PascalCase to camelCase (lowercase first letter)
func toCamelCase(s string) string {
	if s == "" {
		return ""
	}
	// Convert first character to lowercase
	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

// toSnakeCase converts PascalCase to snake_case
func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			result = append(result, '_')
		}
		result = append(result, unicode.ToLower(r))
	}
	return string(result)
}
