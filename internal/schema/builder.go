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
		structParser:       nil,          // Will be set if needed
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

				// Check if this field is a custom interface type
				isCustomInterface := false
				if ident, ok := field.Type.(*ast.Ident); ok && b.typeResolver != nil {
					// Try to resolve the type
					resolvedType := b.typeResolver.FindTypeSpec(ident.Name, typeSpec.File)
					if resolvedType != nil {
						// Check if it's an interface type
						if _, ok := resolvedType.TypeSpec.Type.(*ast.InterfaceType); ok {
							isCustomInterface = true
						}
					}
				}

				// Create property schema
				fieldType := getFieldType(field.Type)

				// Special handling for interface types (error, interface{}, any, custom interfaces)
				// These should be represented as empty schemas (no type specified)
				var propSchema spec.Schema
				if fieldType == "interface" || isCustomInterface {
					propSchema = spec.Schema{
						SchemaProps: spec.SchemaProps{
							// Empty - allows any JSON value
						},
					}
				} else {
					propSchema = spec.Schema{
						SchemaProps: spec.SchemaProps{
							Type: []string{fieldType},
						},
					}
				}

				if example != nil {
					propSchema.Example = example
				}
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

func getFieldType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		// Basic type like string, int
		switch t.Name {
		case "string":
			return "string"
		case "int", "int32", "int64", "uint", "uint32", "uint64":
			return "integer"
		case "float32", "float64":
			return "number"
		case "bool":
			return "boolean"
		case "error", "any":
			// error and any are interface types - allow any JSON value
			return "interface"
		default:
			return "object"
		}
	case *ast.SelectorExpr:
		// Package-qualified type like time.Time, uuid.UUID
		if ident, ok := t.X.(*ast.Ident); ok {
			packageName := ident.Name
			typeName := t.Sel.Name
			// Handle special types
			if packageName == "time" && typeName == "Time" {
				return "string" // time.Time is represented as string in OpenAPI
			}
			if packageName == "uuid" && typeName == "UUID" {
				return "string" // UUID is represented as string
			}
			if packageName == "decimal" && typeName == "Decimal" {
				return "number" // decimal.Decimal is number
			}
		}
		return "object"
	case *ast.ArrayType:
		return "array"
	case *ast.StarExpr:
		// Pointer type - recurse
		return getFieldType(t.X)
	case *ast.InterfaceType:
		// interface{} or any - allow any JSON value
		return "interface"
	default:
		return "object"
	}
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
