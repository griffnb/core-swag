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

	switch t := typeSpec.TypeSpec.Type.(type) {
	case *ast.StructType:
		// Use CoreStructParser for proper field resolution
		// This handles all field types, generics, embedded fields, validation, etc.
		if b.structParser == nil {
			// CoreStructParser not available - should not happen in normal operation
			// Return empty object schema as fallback
			schema.Properties = make(map[string]spec.Schema)
			break
		}

		// Extract package path and type name
		packagePath := typeSpec.PkgPath
		typeName := typeSpec.TypeSpec.Name.Name

		// Use CoreStructParser to get properly parsed fields
		builder := b.structParser.LookupStructFields("", packagePath, typeName)
		if builder == nil {
			// Type not found in CoreStructParser cache - return empty object
			schema.Properties = make(map[string]spec.Schema)
			break
		}

		// Use StructBuilder.BuildSpecSchema() to generate schema with proper type references
		builtSchema, _, err := builder.BuildSpecSchema(typeName, false, b.requiredByDefault, b.enumLookup)
		if err != nil || builtSchema == nil {
			// Schema building failed - return empty object
			schema.Properties = make(map[string]spec.Schema)
			break
		}

		schema = *builtSchema

	case *ast.Ident:
		// Type alias to basic type (e.g., type Role int, type Status string)
		// First check if this is an enum type (int/string with const values)
		if b.enumLookup != nil {
			// Build fully qualified type name: packagePath.TypeName
			fullTypeName := typeSpec.PkgPath + "." + typeSpec.TypeSpec.Name.Name

			// Try to get enum values
			enumValues, err := b.enumLookup.GetEnumsForType(fullTypeName, typeSpec.File)

			if err == nil && len(enumValues) > 0 {

				// This is an enum type - create enum schema
				// Determine underlying type from the alias
				underlyingType := "integer" // Default for int-based enums
				if t.Name == "string" {
					underlyingType = "string"
				}

				// Build enum value list
				enumList := make([]any, len(enumValues))
				varNames := make([]string, len(enumValues))
				for i, ev := range enumValues {
					enumList[i] = ev.Value
					varNames[i] = ev.Key
				}

				schema = spec.Schema{
					SchemaProps: spec.SchemaProps{
						Type: []string{underlyingType},
						Enum: enumList,
					},
					VendorExtensible: spec.VendorExtensible{
						Extensions: spec.Extensions{
							"x-enum-varnames": varNames,
						},
					},
				}
				// Enum schema created, skip alias resolution
			}
		}

		// If not an enum, try to resolve the alias to the actual type
		if schema.Properties == nil && schema.Type == nil {
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
			if schema.Properties == nil && schema.Type == nil {
				schema.Properties = make(map[string]spec.Schema)
			}
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
